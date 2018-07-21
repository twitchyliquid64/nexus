package compute

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"nexus/data/compute"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	computegcp "google.golang.org/api/compute/v1"
)

var db *sql.DB
var computeLock sync.Mutex

func deleteGCPInstance(ctx context.Context, instance compute.Instance) error {
	conf, err2 := google.JWTConfigFromJSON([]byte(instance.Auth), computegcp.CloudPlatformScope)
	if err2 != nil {
		return err2
	}

	var authInfo struct {
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal([]byte(instance.Auth), &authInfo); err != nil {
		return err
	}
	var instanceMeta struct {
		MachineType string `json:"machineType"`
	}
	if err := json.Unmarshal([]byte(instance.Metadata), &instanceMeta); err != nil {
		return err
	}
	zone := strings.Split(strings.TrimPrefix(instanceMeta.MachineType, "zones/"), "/")[0]

	computeService, err := computegcp.New(conf.Client(oauth2.NoContext))
	if err != nil {
		return err
	}
	op, err := computeService.Instances.Delete(authInfo.ProjectID, zone, instance.Name).Do()
	if err != nil {
		log.Printf("instance.Delete(%q, %q, %q) error: %v", authInfo.ProjectID, zone, instance.Name, err)
		return err
	}
	if op.Error != nil {
		log.Printf("instance.Delete(%q, %q, %q) operation error: %v", authInfo.ProjectID, zone, instance.Name, op.Error.Errors)
		return err
	}

	return compute.Delete(ctx, instance.UID, db)
}

func checkDeleteExpiredInstances() error {
	computeLock.Lock()
	defer computeLock.Unlock()
	ctx := context.Background()

	instances, err := compute.GetAll(ctx, db)
	if err != nil {
		log.Printf("compute.GetAll() error: %v", err)
		return err
	}

	for _, instance := range instances {
		if instance.ExpiresAt.Before(time.Now()) {
			switch instance.Kind {
			case "GCP":
				if err := deleteGCPInstance(ctx, instance); err != nil {
					log.Printf("Failed to remove GCP instance: %v", err)
				}
			}
		}
	}

	return nil
}

func expiredCheckerRoutine() {
	time.Sleep(time.Second * 10)

	for {
		if err := checkDeleteExpiredInstances(); err != nil {
			log.Printf("[checkDeleteExpiredInstances] Error: %v", err)
		}
		time.Sleep(time.Second * 50)
	}
}

// Initialise sets up the compute subsystem.
func Initialise(ctx context.Context, database *sql.DB) error {
	db = database
	go expiredCheckerRoutine()
	return nil
}
