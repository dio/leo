package compute

import (
	"context"
	"errors"
	"net"
	"strings"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	backoff "github.com/cenkalti/backoff/v4"
	"google.golang.org/protobuf/proto"
)

type Instance struct {
	ProjectID string
	Zone      string
	Name      string
}

func (i *Instance) Region() string {
	return i.Zone[0:strings.LastIndex(i.Zone, "-")]
}

func (i *Instance) Delete(ctx context.Context) error {
	instance, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return err
	}
	defer instance.Close()

	req := &computepb.DeleteInstanceRequest{
		Project:  i.ProjectID,
		Zone:     i.Zone,
		Instance: i.Name,
	}

	op, err := instance.Delete(ctx, req)
	if err != nil {
		return err
	}

	if err := op.Wait(ctx); err != nil {
		return err
	}

	return nil
}

func (i *Instance) Create(ctx context.Context, machineType, machineImage string, nonSpot bool) error {
	instance, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return err
	}
	defer instance.Close()

	sched := &computepb.Scheduling{
		OnHostMaintenance: proto.String("TERMINATE"),
		AutomaticRestart:  proto.Bool(false),
	}

	// NOT nonSpot means spot.
	if !nonSpot {
		sched.Preemptible = proto.Bool(true)
		sched.ProvisioningModel = proto.String("SPOT")
		sched.InstanceTerminationAction = proto.String("DELETE")
	}

	req := &computepb.InsertInstanceRequest{
		Project: i.ProjectID,
		Zone:    i.Zone,
		InstanceResource: &computepb.Instance{
			Zone: proto.String("projects/" + i.ProjectID + "/zones/" + i.Zone),
			Name: proto.String(i.Name),
			Labels: map[string]string{
				"purpose": "builder",
			},
			Disks: []*computepb.AttachedDisk{
				{
					InitializeParams: &computepb.AttachedDiskInitializeParams{
						DiskSizeGb:  proto.Int64(80),
						DiskType:    proto.String("projects/" + i.ProjectID + "/zones/" + i.Zone + "/diskTypes/pd-balanced"),
						SourceImage: proto.String("projects/" + i.ProjectID + "/global/images/" + machineImage),
					},
					AutoDelete: proto.Bool(true),
					Boot:       proto.Bool(true),
					Type:       proto.String(computepb.AttachedDisk_PERSISTENT.String()),
					Mode:       proto.String(computepb.AttachedDisk_READ_WRITE.String()),
				},
			},
			NetworkInterfaces: []*computepb.NetworkInterface{
				{
					AccessConfigs: []*computepb.AccessConfig{
						{
							Name:        proto.String("External NAT"),
							NetworkTier: proto.String("PREMIUM"),
						},
					},
					StackType:  proto.String("IPV4_ONLY"),
					Subnetwork: proto.String("projects/" + i.ProjectID + "/regions/" + i.Region() + "/subnetworks/default"),
				},
			},
			// For example: n2-standard-8.
			MachineType: proto.String("projects/" + i.ProjectID + "/zones/" + i.Zone + "/machineTypes/" + machineType),
			ServiceAccounts: []*computepb.ServiceAccount{
				{
					Email: proto.String("tetrateio@" + i.ProjectID + ".iam.gserviceaccount.com"),
					Scopes: []string{
						"https://www.googleapis.com/auth/cloud-platform",
					},
				},
			},
			MinCpuPlatform: proto.String("Automatic"),
			Scheduling:     sched,
		},
	}

	op, err := instance.Insert(ctx, req)
	if err != nil {
		return err
	}

	if err := op.Wait(ctx); err != nil {
		return err
	}

	return i.check(ctx, instance)
}

func (i *Instance) Start(ctx context.Context) error {
	instance, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return err
	}
	defer instance.Close()

	req := &computepb.StartInstanceRequest{
		Project:  i.ProjectID,
		Zone:     i.Zone,
		Instance: i.Name,
	}

	op, err := instance.Start(ctx, req)
	if err != nil {
		return err
	}

	if err := op.Wait(ctx); err != nil {
		return err
	}

	// Sleep for 10 secs.
	time.Sleep(10 * time.Second)

	if err := i.check(ctx, instance); err != nil {
		return err
	}

	// Check twice.
	return i.check(ctx, instance)
}

func (i *Instance) Stop(ctx context.Context) error {
	instance, err := compute.NewInstancesRESTClient(ctx)
	if err != nil {
		return err
	}
	defer instance.Close()

	req := &computepb.StopInstanceRequest{
		Project:  i.ProjectID,
		Zone:     i.Zone,
		Instance: i.Name,
	}

	op, err := instance.Stop(ctx, req)
	if err != nil {
		return err
	}

	return op.Wait(ctx)
}

func (i *Instance) check(ctx context.Context, client *compute.InstancesClient) error {
	req := &computepb.GetInstanceRequest{
		Project:  i.ProjectID,
		Zone:     i.Zone,
		Instance: i.Name,
	}

	instance, err := client.Get(ctx, req)
	if err != nil {
		return err
	}

	var ip string
	for _, netif := range instance.NetworkInterfaces {
		for _, a := range netif.AccessConfigs {
			if *a.Name == "External NAT" {
				ip = *a.NatIP
				break
			}
		}
	}

	if len(ip) == 0 {
		return errors.New("invalid external NAT IP address")
	}

	return retry(func() error {
		addr := net.JoinHostPort(ip, "22")
		d := &net.Dialer{
			Timeout: 10 * time.Second,
		}
		_, err := d.DialContext(ctx, "tcp", addr)
		return err
	})
}

// retry does retries.
func retry(op backoff.Operation) error {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     5 * time.Second,
		RandomizationFactor: backoff.DefaultRandomizationFactor,
		Multiplier:          backoff.DefaultMultiplier,
		MaxInterval:         backoff.DefaultMaxInterval,
		MaxElapsedTime:      backoff.DefaultMaxElapsedTime,
		Stop:                backoff.Stop,
		Clock:               backoff.SystemClock,
	}
	b.Reset()
	return backoff.Retry(op, backoff.WithMaxRetries(b, 10))
}
