package compute

import (
	"context"

	compute "cloud.google.com/go/compute/apiv1"
	"cloud.google.com/go/compute/apiv1/computepb"
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

	return op.Wait(ctx)
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
