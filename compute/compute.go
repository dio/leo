package compute

import (
	"context"
	"errors"
	"net"
	"time"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
	backoff "github.com/cenkalti/backoff/v4"
)

type Instance struct {
	ProjectID string
	Zone      string
	Name      string
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
