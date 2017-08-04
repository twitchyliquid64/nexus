package data

import (
	"database/sql"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	gosqlite3 "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/goamz/aws"
	"github.com/rlmcpherson/s3gof3r"
)

var (
	dbDumpInProgress     = false
	dbDumpPagesRemaining = 0
	dbDumpPagesTotal     = 0
	dbUploadInProgress   = false

	dbConfiguredBackupInterval time.Duration
	dbLastBackup               time.Time
)

// GetBackupStatistics returns information to be displayed in the stats page.
func GetBackupStatistics() map[string]interface{} {
	return map[string]interface{}{
		"Dump in progress":   dbDumpInProgress,
		"Upload in progress": dbUploadInProgress,
		"Backup interval":    dbConfiguredBackupInterval,
		"Last backup":        dbLastBackup,
	}
}

// StartBackups is called to initialise periodic backups
func StartBackups(backupInterval time.Duration) {
	dbConfiguredBackupInterval = backupInterval
	go backupRoutine(backupInterval)
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

func backupUpload(fPath string) error {
	dbUploadInProgress = true
	defer func() { dbUploadInProgress = false }()

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
	w, err := s3Access.Bucket(os.Getenv("AWS_BACKUP_BUCKET_NAME")).PutWriter(os.Getenv("AWS_BACKUP_PATH"), nil, uploadConfig)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, d)
	return err
}

func backupRoutine(backupInterval time.Duration) {
	lastRun := time.Now()
	for {
		time.Sleep(time.Second * 10)
		if lastRun.Add(backupInterval).Before(time.Now()) {
			lastRun = time.Now()
			log.Println("[backup] Starting backup.")

			if len(sqlite3conn) == 0 {
				log.Println("[backup] No sqlite3conn found, is the db initialized?")
				continue
			}
			backupFile, err := doBackup(sqlite3conn[0])
			if err != nil {
				log.Printf("[backup] Failed with error: %s", err)
				if backupFile != "" {
					os.Remove(backupFile)
				}
				continue
			}
			log.Printf("[backup] Backup dump to %q finished", backupFile)

			err = backupUpload(backupFile)
			if err != nil {
				log.Printf("[backup] Backup update to %q failed: %v", backupFile, err)
			}
			log.Printf("[backup] Backup upload finished")
			dbLastBackup = time.Now()

			if backupFile != "" {
				err = os.Remove(backupFile)
				if err != nil {
					log.Printf("[backup] Failed to delete backup file: %s", err)
				}
			}
		}
	}
}

func doBackup(srcDBConn *gosqlite3.SQLiteConn) (string, error) {
	f, err := ioutil.TempFile("", "db-backup-")
	if err != nil {
		return "", err
	}
	f.Close()
	fName := f.Name()

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

	return fName, stepErr
}
