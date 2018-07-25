package integration

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	computedb "nexus/data/compute"
	"nexus/fs"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"github.com/robertkrimen/otto"
	uuid "github.com/satori/go.uuid"
	compute "google.golang.org/api/compute/v1"
)

type computeInitialiser struct{}

func (c *computeInitialiser) Apply(r *Run) error {
	obj, errMake := makeObject(r.VM)
	if errMake != nil {
		return errMake
	}

	if err := obj.Set("get", func(call otto.FunctionCall) otto.Value {
		gcpProject := call.Argument(1).String()
		gcpZone := call.Argument(2).String()
		instance, err := computedb.Get(r.Ctx, call.Argument(0).String(), r.Base.OwnerID, db)
		if err != nil {
			return r.VM.MakeCustomError("not-found-error", err.Error())
		}

		conf, err := google.JWTConfigFromJSON([]byte(instance.Auth), compute.CloudPlatformScope)
		if err != nil {
			return r.VM.MakeCustomError("oauth-error", err.Error())
		}
		computeService, err := compute.New(conf.Client(oauth2.NoContext))
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		output, errMake := makeObject(r.VM)
		if errMake != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}

		output.Set("success", true)
		output.Set("compute_id", instance.UID)
		output.Set("instance_name", instance.Name)
		output.Set("instance_expiry", instance.ExpiresAt)
		output.Set("instance_expiry_nano", instance.ExpiresAt.UnixNano())
		if err := c.makeRunClosure(r, output, computeService.Instances, *instance, instance.SSH, gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := c.makeWriteFileClosure(r, output, computeService.Instances, *instance, instance.SSH, gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := c.makeReadFileClosure(r, output, computeService.Instances, *instance, instance.SSH, gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := c.makeBootedClosure(r, output, computeService.Instances, *instance, gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := c.makeGetIPClosure(r, output, computeService.Instances, *instance, gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		return output.Value()
	}); err != nil {
		return err
	}

	if err := obj.Set("new_instance", func(call otto.FunctionCall) otto.Value {
		gcpCredsPath := call.Argument(0).String()
		gcpProject := call.Argument(1).String()
		gcpZone := call.Argument(2).String()
		instanceDuration, err := call.Argument(3).ToInteger()
		if err != nil {
			return r.VM.MakeCustomError("type-error", err.Error())
		}
		expiry := time.Now().Add(time.Second * time.Duration(instanceDuration))
		gcpMachineType := "zones/" + gcpZone + "/machineTypes/" + call.Argument(4).String()
		gcpImageURL := call.Argument(5).String()

		var b bytes.Buffer
		err = fs.Contents(r.Ctx, gcpCredsPath, r.Base.OwnerID, &b)
		if err != nil {
			return r.VM.MakeCustomError("fs-error", err.Error())
		}

		conf, err := google.JWTConfigFromJSON(b.Bytes(), compute.CloudPlatformScope)
		if err != nil {
			return r.VM.MakeCustomError("oauth-error", err.Error())
		}

		computeService, err := compute.New(conf.Client(oauth2.NoContext))
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		var privateKeyBytes bytes.Buffer
		privateKeyPEM := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}
		if err = pem.Encode(&privateKeyBytes, privateKeyPEM); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		pub, err := ssh.NewPublicKey(&privateKey.PublicKey)
		if err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		authSection := "xxx: " + string(ssh.MarshalAuthorizedKey(pub))

		name := "tempvm-" + uuid.Must(uuid.NewV4()).String()
		instance := &compute.Instance{
			Name:        name,
			MachineType: gcpMachineType,
			Tags: &compute.Tags{
				Items: []string{"nexusauto-timebound", "duration-" + fmt.Sprintf("%d", instanceDuration), "builder-vm"},
			},
			Disks: []*compute.AttachedDisk{{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				Mode:       "READ_WRITE",
				DeviceName: name,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: gcpImageURL,
					DiskSizeGb:  40,
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

		op, err := computeService.Instances.Insert(gcpProject, gcpZone, instance).Do()
		if err != nil {
			return r.VM.MakeCustomError("gcp-error", err.Error())
		}
		if op.Error != nil {
			return r.VM.MakeCustomError("gcp-error", fmt.Sprint(op.Error.Errors))
		}

		instanceMeta, _ := instance.MarshalJSON()
		inst := computedb.Instance{
			ID:        fmt.Sprint(op.Id),
			Name:      name,
			ExpiresAt: expiry,
			Auth:      b.String(),
			OwnerID:   r.Base.OwnerID,
			Kind:      "GCP",
			Metadata:  string(instanceMeta),
			SSH:       privateKeyBytes.String(),
		}
		cid, err := computedb.New(r.Ctx, inst, db)
		if err != nil {
			// TODO: Delete instance. on error
			return r.VM.MakeCustomError("db-error", err.Error())
		}
		inst.UID = int(cid)

		output, errMake := makeObject(r.VM)
		if errMake != nil {
			// TODO: Delete instance. on error
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}

		output.Set("success", true)
		output.Set("compute_id", inst.UID)
		output.Set("instance_name", name)
		output.Set("instance_expiry", expiry)
		output.Set("instance_expiry_nano", expiry.UnixNano())
		if err := c.makeRunClosure(r, output, computeService.Instances, inst, privateKeyBytes.String(), gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := c.makeWriteFileClosure(r, output, computeService.Instances, inst, privateKeyBytes.String(), gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := c.makeReadFileClosure(r, output, computeService.Instances, inst, privateKeyBytes.String(), gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := c.makeBootedClosure(r, output, computeService.Instances, inst, gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}
		if err := c.makeGetIPClosure(r, output, computeService.Instances, inst, gcpProject, gcpZone); err != nil {
			return r.VM.MakeCustomError("internal-error", err.Error())
		}

		return output.Value()
	}); err != nil {
		return err
	}

	return r.VM.Set("compute", obj)
}

func (c *computeInitialiser) makeBootedClosure(r *Run, obj *otto.Object, instService *compute.InstancesService,
	instance computedb.Instance, project, zone string) error {
	if err := obj.Set("run_status", func(call otto.FunctionCall) otto.Value {
		serial, err := instService.GetSerialPortOutput(project, zone, instance.Name).Do()
		if err != nil && !strings.Contains(err.Error(), "resourceNotReady") {
			return r.VM.MakeCustomError("gcp-error", err.Error())
		}

		output, errMake := makeObject(r.VM)
		if errMake != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}

		output.Set("success", true)
		if err != nil && strings.Contains(err.Error(), "resourceNotReady") {
			output.Set("booted", false)
			output.Set("pending", true)
		} else {
			output.Set("serial_data", serial.Contents)
			output.Set("serial_offset", serial.Start)
			output.Set("booted", strings.Contains(serial.Contents, instance.Name+" login:"))
			output.Set("pending", false)
		}

		return output.Value()
	}); err != nil {
		return err
	}
	return nil
}

func (c *computeInitialiser) makeRunClosure(r *Run, obj *otto.Object, instService *compute.InstancesService,
	instance computedb.Instance, privateKey, project, zone string) error {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		User: "xxx",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if err := obj.Set("run", func(call otto.FunctionCall) otto.Value {
		i, err := instService.Get(project, zone, instance.Name).Do()
		if err != nil {
			return r.VM.MakeCustomError("gcp-error", err.Error())
		}

		client, err := ssh.Dial("tcp", getExternalIP(i)+":22", config)
		if err != nil {
			return r.VM.MakeCustomError("dial-error", err.Error())
		}
		session, err := client.NewSession()
		if err != nil {
			return r.VM.MakeCustomError("ssh-error", err.Error())
		}
		defer session.Close()
		var b bytes.Buffer
		session.Stdout = &b
		err = session.Run(call.Argument(0).String())

		output, errMake := makeObject(r.VM)
		if errMake != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}

		output.Set("success", err == nil)
		output.Set("error", err)
		output.Set("output", b.String())
		output.Set("output_raw", b.Bytes())

		return output.Value()
	}); err != nil {
		return err
	}
	return nil
}

func (c *computeInitialiser) makeWriteFileClosure(r *Run, obj *otto.Object, instService *compute.InstancesService,
	instance computedb.Instance, privateKey, project, zone string) error {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		User: "xxx",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if err := obj.Set("write_file", func(call otto.FunctionCall) otto.Value {
		i, err := instService.Get(project, zone, instance.Name).Do()
		if err != nil {
			return r.VM.MakeCustomError("gcp-error", err.Error())
		}

		client, err := ssh.Dial("tcp", getExternalIP(i)+":22", config)
		if err != nil {
			return r.VM.MakeCustomError("dial-error", err.Error())
		}
		session, err := client.NewSession()
		if err != nil {
			return r.VM.MakeCustomError("ssh-error", err.Error())
		}
		defer session.Close()
		var b bytes.Buffer
		session.Stdout = &b
		session.Stdin = bytes.NewBufferString(call.Argument(1).String())
		err = session.Run("dd of=" + call.Argument(0).String())

		output, errMake := makeObject(r.VM)
		if errMake != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}

		output.Set("success", err == nil)
		output.Set("error", err)

		return output.Value()
	}); err != nil {
		return err
	}
	return nil
}

func (c *computeInitialiser) makeReadFileClosure(r *Run, obj *otto.Object, instService *compute.InstancesService,
	instance computedb.Instance, privateKey, project, zone string) error {
	key, err := ssh.ParsePrivateKey([]byte(privateKey))
	if err != nil {
		return err
	}
	config := &ssh.ClientConfig{
		User: "xxx",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if err := obj.Set("read_file", func(call otto.FunctionCall) otto.Value {
		i, err := instService.Get(project, zone, instance.Name).Do()
		if err != nil {
			return r.VM.MakeCustomError("gcp-error", err.Error())
		}

		client, err := ssh.Dial("tcp", getExternalIP(i)+":22", config)
		if err != nil {
			return r.VM.MakeCustomError("dial-error", err.Error())
		}
		session, err := client.NewSession()
		if err != nil {
			return r.VM.MakeCustomError("ssh-error", err.Error())
		}
		defer session.Close()
		var b bytes.Buffer
		session.Stdout = &b
		err = session.Run("cat " + call.Argument(0).String())

		output, errMake := makeObject(r.VM)
		if errMake != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}

		output.Set("success", err == nil)
		output.Set("error", err)
		output.Set("contents", b.String())
		output.Set("contents_raw", b.Bytes())

		return output.Value()
	}); err != nil {
		return err
	}
	return nil
}

func (c *computeInitialiser) makeGetIPClosure(r *Run, obj *otto.Object, instService *compute.InstancesService,
	instance computedb.Instance, project, zone string) error {
	if err := obj.Set("getIP", func(call otto.FunctionCall) otto.Value {
		i, err := instService.Get(project, zone, instance.Name).Do()
		if err != nil {
			return r.VM.MakeCustomError("gcp-error", err.Error())
		}
		output, errMake := makeObject(r.VM)
		if errMake != nil {
			return r.VM.MakeCustomError("internal-error", errMake.Error())
		}

		output.Set("success", true)
		output.Set("ip", getExternalIP(i))
		return output.Value()
	}); err != nil {
		return err
	}
	return nil
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
