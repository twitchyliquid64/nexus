package data

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"nexus/data/dlock"
	"os"
	"strings"
	"time"

	gosqlite3 "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
	"github.com/rlmcpherson/s3gof3r"
)

type verificationInfo struct {
	Name   string
	Detail string
	Result error
}

var (
	lastRun = time.Time{}

	tablesToVerify        = []string{"fs_minifiles", "messaging_messages", "users", "integration_runnable"}
	dbVerificationState   error
	dbDumpLastSize        float64
	dbVerificationResults []verificationInfo

	dbDumpInProgress     = false
	dbDumpPagesRemaining = 0
	dbDumpPagesTotal     = 0
	dbDumpDuration       = time.Duration(0)
	dbUploadInProgress   = false
	dbUploadDuration     = time.Duration(0)

	dbConfiguredBackupInterval time.Duration
	dbLastBackup               time.Time

	startBackupTriggerChan chan bool
)

// GetBackupStatistics returns information to be displayed in the stats page.
func GetBackupStatistics() map[string]interface{} {
	return map[string]interface{}{
		"Dump in progress":     dbDumpInProgress,
		"Upload in progress":   dbUploadInProgress,
		"Backup interval":      dbConfiguredBackupInterval,
		"Last backup":          dbLastBackup,
		"Dump time":            dbDumpDuration,
		"Upload time":          dbUploadDuration,
		"Last backup size":     dbDumpLastSize,
		"Verification error":   dbVerificationState,
		"Verification details": dbVerificationResults,
	}
}

// StartBackups is called to initialise periodic backups
func StartBackups(backupInterval time.Duration) {
	startBackupTriggerChan = make(chan bool)
	dbConfiguredBackupInterval = backupInterval
	go func() {
		for {
			time.Sleep(backupInterval)
			startBackupTriggerChan <- true
		}
	}()
	go backupRoutine()
}

func getS3Handle() (*s3gof3r.S3, error) {
	keys, err := s3gof3r.EnvKeys()
	if err != nil {
		return nil, err
	}
	if _, ok := aws.Regions[os.Getenv("AWS_REGION")]; !ok {
		return nil, errors.New("no or unknown AWS region specified in env 'AWS_REGION': " + os.Getenv("AWS_REGION"))
	}
	if os.Getenv("AWS_BACKUP_BUCKET_NAME") == "" {
		return nil, errors.New("no bucket name specified in env 'AWS_BACKUP_BUCKET_NAME'")
	}
	if os.Getenv("AWS_BACKUP_PATH") == "" {
		return nil, errors.New("no backup path specified in env 'AWS_BACKUP_PATH'")
	}
	return s3gof3r.New(strings.Replace(aws.Regions[os.Getenv("AWS_REGION")].S3Endpoint, "https://", "", -1), keys), nil
}

// BackupNow triggers the backup routine to start immediately.
func BackupNow() {
	startBackupTriggerChan <- true
}

func backupUpload(fPath string) error {
	dbUploadInProgress = true
	defer func() {
		dbUploadInProgress = false
		dbUploadDuration = time.Now().Sub(lastRun) - dbDumpDuration
	}()

	uploadConfig := &s3gof3r.Config{
		Concurrency: 2,
		PartSize:    6 * 1024 * 1024,
		NTry:        10,
		Md5Check:    true,
		Scheme:      "https",
		Client:      s3gof3r.ClientWithTimeout(12 * time.Second),
	}
	s3Access, err := getS3Handle()
	if err != nil {
		return err
	}
	d, err := os.Open(fPath)
	if err != nil {
		return err
	}
	defer d.Close()

	w, err := s3Access.Bucket(os.Getenv("AWS_BACKUP_BUCKET_NAME")).PutWriter(computeBackupPath(), nil, uploadConfig)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, d)
	return err
}

func computeBackupPath() string {
	backupPath := os.Getenv("AWS_BACKUP_PATH")
	backupPath = strings.Replace(backupPath, "{{DAY}}", fmt.Sprintf("%02d", time.Now().Day()), -1)
	backupPath = strings.Replace(backupPath, "{{MONTH}}", fmt.Sprintf("%02d", time.Now().Month()), -1)
	backupPath = strings.Replace(backupPath, "{{YEAR}}", fmt.Sprintf("%02d", time.Now().Year()), -1)
	backupPath = strings.Replace(backupPath, "{{YEARDAY}}", fmt.Sprintf("%03d", time.Now().YearDay()), -1)
	return backupPath
}

func backupRoutine() {
	lastRun = time.Now()
	for {
		<-startBackupTriggerChan // wait till we have the signal to start.
		if dbDumpInProgress || dbUploadInProgress {
			continue
		}
		lastRun = time.Now()
		log.Println("[backup] Starting backup.")

		if len(sqlite3conn) == 0 {
			dbVerificationState = fmt.Errorf("precondition failed: check db initialization")
			log.Println("[backup] No sqlite3conn found, is the db initialized?")
			continue
		}
		backupFile, err := doBackup(sqlite3conn[0])
		if err != nil {
			dbVerificationState = fmt.Errorf("backup failed: %v", err)
			log.Printf("[backup] Failed with error: %s", err)
			if backupFile != "" {
				os.Remove(backupFile)
			}
			continue
		}
		log.Printf("[backup] Backup dump to %q finished", backupFile)

		err = backupUpload(backupFile)
		if err != nil {
			dbVerificationState = fmt.Errorf("upload failed: %v", err)
			log.Printf("[backup] Backup update to %q failed: %v", backupFile, err)
		}
		log.Printf("[backup] Backup upload finished")
		dbLastBackup = time.Now()

		if backupFile != "" {
			if s, err2 := os.Stat(backupFile); err2 == nil {
				dbDumpLastSize = float64(s.Size()/1024) / 1024
			}
			checkBackup(backupFile)

			err = os.Remove(backupFile)
			if err != nil {
				log.Printf("[backup] Failed to delete backup file: %s", err)
			}
		}
	}
}

func checkBackup(path string) {
	dbVerificationResults = nil
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		dbVerificationState = err
		return
	}
	defer db.Close()

	// first check the number of tables at least meets the default tables
	r, err := db.Query("SELECT COUNT(*) FROM sqlite_master WHERE type in ('table', 'view') AND name not like 'sqlite?_%' escape '?'")
	if err != nil {
		dbVerificationState = err
		return
	}
	r.Next()
	var tableCount int
	if err2 := r.Scan(&tableCount); err2 != nil {
		dbVerificationState = err2
		return
	}
	dbVerificationResults = append(dbVerificationResults, verificationInfo{Name: "tables_count", Detail: fmt.Sprintf("%d tables default, %d tables in backup", len(tables), tableCount)})
	if tableCount < len(tables) {
		dbVerificationState = fmt.Errorf("Table count mismatch! Expected at least %d, got %d", len(tables), tableCount)
		return
	}

	// check count of rows in some tables are within 5% of each other
	for _, table := range tablesToVerify {
		backupCount, liveCount, err2 := countRowsInTable(db, table)
		if err2 != nil {
			dbVerificationState = err2
			return
		}
		ratio := float64(backupCount) / float64(liveCount)
		dbVerificationResults = append(dbVerificationResults, verificationInfo{Name: table, Detail: fmt.Sprintf("%d rows live, %d rows in backup", liveCount, backupCount)})
		if ratio > 1.05 || ratio < 0.95 {
			dbVerificationState = fmt.Errorf("table %q has >5 percent row count mismatch (ratio = %2f, %d != %d)", table, ratio, backupCount, liveCount)
			dbVerificationResults[len(dbVerificationResults)-1].Result = dbVerificationState
		}
	}

	// check the size of the object in S3
	dbVerificationResults = append(dbVerificationResults, verificationInfo{Name: "remote read"})
	remoteSize, _, err := getRemoteBackupDetails()
	if err != nil {
		dbVerificationState = err
		dbVerificationResults[len(dbVerificationResults)-1].Result = err
		return
	}
	dbVerificationResults = append(dbVerificationResults, verificationInfo{Name: "size match", Detail: fmt.Sprintf("%.1fMB", float64(remoteSize/1024)/1024)})
	if (float64(remoteSize/1024) / 1024) != dbDumpLastSize {
		dbVerificationState = fmt.Errorf("remote backup size mismatch")
		dbVerificationResults[len(dbVerificationResults)-1].Result = err
		return
	}
}

func getRemoteBackupDetails() (int64, string, error) {
	auth, err := aws.EnvAuth()
	if err != nil {
		return 0, "", err
	}
	key, err := s3.New(auth, aws.Regions[os.Getenv("AWS_REGION")]).Bucket(os.Getenv("AWS_BACKUP_BUCKET_NAME")).GetKey(computeBackupPath())
	if err != nil {
		return 0, "", err
	}
	return key.Size, key.ETag, nil
}

func hashFileMD5(filePath string) (string, error) {
	// credit: cleaned up version of http://www.mrwaggel.be/post/generate-md5-hash-of-a-file-in-golang/
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)[:16]), nil
}

func countRowsInTable(backupDb *sql.DB, tableName string) (int, int, error) {
	r, err := backupDb.Query("SELECT COUNT(*) FROM " + tableName)
	if err != nil {
		return 0, 0, err
	}
	defer r.Close()
	r.Next()
	var backupCount int
	if err = r.Scan(&backupCount); err != nil {
		return 0, 0, err
	}

	r2, err := dbForBackups.Query("SELECT COUNT(*) FROM " + tableName)
	if err != nil {
		return 0, 0, err
	}
	defer r2.Close()
	r2.Next()
	var homeCount int
	if err = r2.Scan(&homeCount); err != nil {
		return 0, 0, err
	}

	return backupCount, homeCount, nil
}

func doBackup(srcDBConn *gosqlite3.SQLiteConn) (string, error) {
	defer func() {
		dbDumpDuration = time.Now().Sub(lastRun)
	}()
	f, err := ioutil.TempFile("", "db-backup-")
	if err != nil {
		return "", err
	}
	f.Close()
	fName := f.Name()

	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	destDB, err := sql.Open("sqlite3_conn_hook_backup", fName)
	if err != nil {
		return fName, err
	}
	defer destDB.Close()
	err = destDB.Ping()
	if err != nil {
		return fName, err
	}

	destDBConn := sqlite3backupconn[0]
	sqlite3backupconn = []*gosqlite3.SQLiteConn{}
	backup, err := destDBConn.Backup("main", srcDBConn, "main")
	if err != nil {
		return fName, err
	}
	defer backup.Finish()

	_, stepErr := backup.Step(0)
	for stepErr == nil || stepErr == gosqlite3.ErrLocked || stepErr == gosqlite3.ErrBusy {
		dbDumpPagesRemaining = backup.Remaining()
		dbDumpPagesTotal = backup.PageCount()
		dbDumpInProgress = true

		var isDone bool
		isDone, stepErr = backup.Step(4)
		if isDone {
			break
		}
	}

	dbDumpInProgress = false
	return fName, stepErr
}
