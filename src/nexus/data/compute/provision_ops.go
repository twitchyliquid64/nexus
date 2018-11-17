package compute

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"log"
	"nexus/fs"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

func createCompute(ctx context.Context, path string, ownerID int) (*compute.Service, *bytes.Buffer, error) {
	var b bytes.Buffer
	err := fs.Contents(ctx, path, ownerID, &b)
	if err != nil {
		return nil, nil, err
	}
	conf, err := google.JWTConfigFromJSON(b.Bytes(), compute.CloudPlatformScope)
	if err != nil {
		return nil, nil, err
	}
	cs, err := compute.New(conf.Client(oauth2.NoContext))
	if err != nil {
		return nil, nil, err
	}
	return cs, &b, nil
}

func createKey(sshkey string) (string, *rsa.PrivateKey, *bytes.Buffer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", nil, nil, err
	}
	var privateKeyBytes bytes.Buffer
	privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
	if err = pem.Encode(&privateKeyBytes, privateKeyPEM); err != nil {
		return "", nil, nil, err
	}
	pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", nil, nil, err
	}
	authSection := "nexusprovision: " + string(ssh.MarshalAuthorizedKey(pub)) + "\n" + sshkey
	return authSection, privateKey, &privateKeyBytes, nil
}

func createPersonalInstance(ctx context.Context, path, name, project, zone, machineType, imageURL, sshkey string, ownerID, durationSeconds int, db *sql.DB) error {
	expiry := time.Now().Add(time.Second * time.Duration(durationSeconds))
	gcpMachineType := "zones/" + zone + "/machineTypes/" + machineType

	computeService, authInfo, err := createCompute(ctx, path, ownerID)
	if err != nil {
		return err
	}

	authSection, priv, privateKeyBytes, err := createKey(sshkey)
	if err != nil {
		return err
	}

	instance := &compute.Instance{
		Name:        name,
		MachineType: gcpMachineType,
		Tags: &compute.Tags{
			Items: []string{"nexusauto-timebound", "duration-" + fmt.Sprintf("%d", durationSeconds), "builder-vm"},
		},
		Disks: []*compute.AttachedDisk{{
			AutoDelete: true,
			Boot:       true,
			Type:       "PERSISTENT",
			Mode:       "READ_WRITE",
			DeviceName: name,
			InitializeParams: &compute.AttachedDiskInitializeParams{
				SourceImage: imageURL,
				DiskSizeGb:  100,
			},
		}},
		NetworkInterfaces: []*compute.NetworkInterface{{
			Network: "global/networks/default",
			AccessConfigs: []*compute.AccessConfig{{
				Name: "External NAT",
			}},
		}},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{{
				Key:   "ssh-keys",
				Value: &authSection,
			}},
		},
	}

	op, err := computeService.Instances.Insert(project, zone, instance).Do()
	if err != nil {
		return err
	}
	if op.Error != nil {
		return fmt.Errorf("Operation failed: %s", fmt.Sprint(op.Error.Errors))
	}

	instanceMeta, _ := instance.MarshalJSON()
	inst := Instance{
		ID:        fmt.Sprint(op.Id),
		Name:      name,
		ExpiresAt: expiry,
		Auth:      authInfo.String(),
		OwnerID:   ownerID,
		Kind:      "GCP",
		Metadata:  string(instanceMeta),
		SSH:       privateKeyBytes.String(),
	}
	cid, err := New(ctx, inst, db)
	if err != nil {
		// TODO: Delete instance. on error
		return err
	}
	inst.UID = int(cid)

	persUID, err := NewPersonal(ctx, PersonalInstance{
		OwnerID:     ownerID,
		InstanceUID: inst.UID,
		Status:      "provisioning",
		UserSSH:     sshkey,
	}, db)
	if err != nil {
		return err
	}

	go instanceProvisioner(project, zone, instance, inst, priv, computeService, persUID, db)
	return nil
}

func instanceProvisioner(project, zone string, instance *compute.Instance, inst Instance, priv *rsa.PrivateKey, serv *compute.Service, persUID int64, db *sql.DB) {
	for x := 0; x < 45; x++ {
		time.Sleep(time.Millisecond * 3500)
		serial, err := serv.Instances.GetSerialPortOutput(project, zone, instance.Name).Do()
		if err != nil {
			if strings.Contains(err.Error(), "resourceNotReady") {
				continue
			}
			log.Printf("[Instance Provisioner] GetSerialOutput(%q) err: %v", instance.Name, err)
			return
		}

		if strings.Contains(serial.Contents, "Finished running startup scripts.") {
			break
		}
	}
	UpdateStatusPersonal(context.Background(), persUID, "Booting", db)

	key, _ := ssh.NewSignerFromKey(priv)
	config := &ssh.ClientConfig{
		User: "nexusprovision",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// get IP
	i, err := serv.Instances.Get(project, zone, instance.Name).Do()
	if err != nil {
		log.Printf("[Instance Provisioner] instance.Get(%q) err: %v", instance.Name, err)
		return
	}
	UpdateIPPersonal(context.Background(), persUID, getExternalIP(i), db)

	// login & change password
	client, err := ssh.Dial("tcp", getExternalIP(i)+":22", config)
	if err != nil {
		log.Printf("[Instance Provisioner] ssh connect(%q) err: %v", instance.Name, err)
		return
	}
	defer client.Close()
	time.Sleep(time.Second * 2)

	session, err := client.NewSession()
	if err != nil {
		return
	}
	var b bytes.Buffer
	session.Stderr = os.Stderr
	session.Stdout = &b
	err = session.Run("sudo service ssh restart")
	log.Printf("[Instance Provisioner] ssh restart for %q err: %v", instance.Name, err)
	log.Printf("[Instance Provisioner] ssh restart output for %q: %v", instance.Name, b.String())
	session.Close()

	UpdateStatusPersonal(context.Background(), persUID, "Ready", db)
}

func getExternalIP(i *compute.Instance) string {
	for _, interf := range i.NetworkInterfaces {
		for _, c := range interf.AccessConfigs {
			if c.Name == "External NAT" {
				return c.NatIP
			}
		}
	}
	return ""
}
