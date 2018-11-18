package compute

import (
	"context"
	"database/sql"
	"nexus/data/dlock"
	"nexus/data/util"
	"strconv"
	"time"
)

// UserInstanceTable (compute_personal) implements the databaseTable interface.
type UserInstanceTable struct{}

// Setup is called on initialization to create necessary structures in the database.
func (t *UserInstanceTable) Setup(ctx context.Context, db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
	CREATE TABLE IF NOT EXISTS compute_personal (
		rowid INTEGER PRIMARY KEY AUTOINCREMENT,
    owner_uid INT NOT NULL,
	  instance_uid INT NOT NULL,
    ip VARCHAR(64) NOT NULL,
		status VARCHAR(64) NOT NULL,
    user_sshkey VARCHAR(2048) NOT NULL,

		CONSTRAINT fk_compute_instances
			FOREIGN KEY (instance_uid)
			REFERENCES compute_instances(rowid)
			ON DELETE CASCADE
	);

  CREATE INDEX IF NOT EXISTS compute_personal_owner ON compute_personal(owner_uid);
	CREATE INDEX IF NOT EXISTS compute_personal_instance_uid ON compute_personal(instance_uid);
	`)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	return nil
}

// Forms is called by the form renderer to get any settings forms relevant to this table.
func (t *UserInstanceTable) Forms() []*util.FormDescriptor {
	return []*util.FormDescriptor{
		&util.FormDescriptor{
			SettingsTitle: "Personal Instances",
			ID:            "personalInstances",
			Desc:          "Personal Instances created & managed by Nexus.",
			Forms: []*util.ActionDescriptor{
				&util.ActionDescriptor{
					Name:   "Create instance",
					ID:     "personal_instance_add",
					IcoStr: "add",
					Fields: []*util.Field{
						&util.Field{
							Name: "Credential file path",
							ID:   "cred",
							Kind: "text",
							Val:  "/minifs/creds/default_cloud.json",
						},
						&util.Field{
							Name: "Name",
							ID:   "name",
							Kind: "text",
						},
						&util.Field{
							Name: "Project name",
							ID:   "project",
							Kind: "text",
						},
						&util.Field{
							Name: "Zone",
							ID:   "machine_zone",
							Kind: "select",
							SelectOptions: map[string]string{
								"us-west1-a":        "us-west1",
								"us-west2-a":        "us-west2",
								"us-east1-a":        "us-east1",
								"us-east4-a":        "us-east4",
								"us-central1-a":     "us-central1",
								"europe-west3-a":    "europe-west3",
								"asia-southeast1-a": "asia-southeast1",
							},
						},
						&util.Field{
							Name: "Image",
							ID:   "machine_image",
							Kind: "select",
							SelectOptions: map[string]string{
								"projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20181029": "Ubuntu 18.04",
							},
						},
						&util.Field{
							Name: "Machine type",
							ID:   "machine_type",
							Kind: "select",
							SelectOptions: map[string]string{
								"n1-standard-1":  "Standard (1 vCPU, 3.75Gb)",
								"n1-standard-2":  "Standard (2 vCPU, 7.50Gb)",
								"n1-standard-4":  "Standard (4 vCPU, 15Gb)",
								"n1-standard-16": "Standard (16 vCPU, 60Gb)",
								"n1-highcpu-2":   "High-CPU (2 vCPU, 1.80Gb)",
								"n1-highcpu-4":   "High-CPU (4 vCPU, 7.20Gb)",
								"n1-highcpu-16":  "High-CPU (16 vCPU, 14.8Gb)",
							},
						},
						&util.Field{
							Name: "Duration",
							ID:   "duration",
							Kind: "select",
							SelectOptions: map[string]string{
								"300":   "5 Minutes",
								"900":   "15 Minutes",
								"3600":  "1 Hour",
								"7200":  "2 Hours",
								"10800": "3 Hours",
								"18000": "5 Hours",
							},
						},
						&util.Field{
							Name: "SSH Username",
							ID:   "sshuser",
							Kind: "text",
						},
						&util.Field{
							Name: "SSH Public Key",
							ID:   "sshkey",
							Kind: "text",
						},
					},
					OnSubmit: t.makePersonalInstanceHandler,
				},
			},
			Tables: []*util.TableDescriptor{
				&util.TableDescriptor{
					Name:    "Personal Instances",
					ID:      "personal_instances",
					Cols:    []string{"#", "Name", "IP", "Expiry", "Status"},
					Actions: []*util.TableAction{},
					FetchContent: func(ctx context.Context, userID int, db *sql.DB) ([]interface{}, error) {
						instances, err := GetAllPersonal(ctx, userID, db)
						if err != nil {
							return nil, err
						}

						var out []interface{}
						for _, inst := range instances {
							out = append(out, []interface{}{inst.UID, inst.JoinedName, inst.IP, inst.JoinedExpiry, inst.Status})
						}
						return out, nil
					},
				},
			},
		},
	}
}

// PersonalInstance represents an instance created for a user to use (rather than an integration).
type PersonalInstance struct {
	UID         int
	OwnerID     int
	InstanceUID int
	IP          string
	Status      string
	UserSSH     string

	JoinedName   string
	JoinedExpiry time.Time
}

func UpdateIPPersonal(ctx context.Context, uid int64, ip string, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
				UPDATE compute_personal
					SET ip=?
					WHERE rowid=?;
			`, ip, uid)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func UpdateStatusPersonal(ctx context.Context, uid int64, status string, db *sql.DB) error {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
				UPDATE compute_personal
					SET status=?
					WHERE rowid=?;
			`, status, uid)
	if err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func NewPersonal(ctx context.Context, i PersonalInstance, db *sql.DB) (int64, error) {
	dlock.Lock().Lock()
	defer dlock.Lock().Unlock()

	tx, err := db.Begin()
	if err != nil {
		return 0, err
	}

	e, err := tx.Exec(`
				INSERT INTO
					compute_personal (owner_uid, instance_uid, ip, status, user_sshkey)
					VALUES (
						?, ?, ?, ?, ?
					);
			`, i.OwnerID, i.InstanceUID, i.IP, i.Status, i.UserSSH)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	lastID, err := e.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	return lastID, tx.Commit()
}

func GetAllPersonal(ctx context.Context, userID int, db *sql.DB) ([]PersonalInstance, error) {
	dlock.Lock().RLock()
	defer dlock.Lock().RUnlock()

	res, err := db.QueryContext(ctx, `
		SELECT p.rowid, p.owner_uid, p.instance_uid, p.ip, p.status, p.user_sshkey,
		 			 c.name, c.expires_at
		FROM compute_personal p
		INNER JOIN compute_instances c ON p.instance_uid = c.rowid
		WHERE p.owner_uid=?;`, userID)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	var output []PersonalInstance
	for res.Next() {
		var o PersonalInstance
		if err := res.Scan(&o.UID, &o.OwnerID, &o.InstanceUID, &o.IP, &o.Status, &o.UserSSH, &o.JoinedName, &o.JoinedExpiry); err != nil {
			return nil, err
		}
		output = append(output, o)
	}
	return output, nil
}

func (t *UserInstanceTable) makePersonalInstanceHandler(ctx context.Context, vals map[string]string, userID int, db *sql.DB) error {
	durationSeconds, err := strconv.Atoi(vals["duration"])
	if err != nil {
		return err
	}

	return createPersonalInstance(ctx, vals["cred"], vals["name"], vals["project"], vals["machine_zone"], vals["machine_type"], vals["machine_image"],
		vals["sshuser"]+": "+vals["sshkey"], userID, durationSeconds, db)
}
