package queue

import (
	"context"
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/pubsub"
)

type InputsBuild struct {
	Name         string `json:"name"`
	Target       string `json:"target"`
	IstioVersion string `json:"istioVersion"`
	Arguments    string `json:"arguments"`
}

type InputsBuildEnvoy struct {
	Name      string `json:"name"`
	Target    string `json:"target"`
	Envoy     string `json:"envoy"`
	Arguments string `json:"arguments"`
}

func Publish(ctx context.Context, topicID string, msg []byte) error {
	client, err := pubsub.NewClient(ctx, os.Getenv("GCLOUD_PROJECT"))
	if err != nil {
		return err
	}
	defer client.Close()

	t := client.Topic(topicID)
	result := t.Publish(ctx, &pubsub.Message{
		Data: []byte(msg),
	})
	id, err := result.Get(ctx)
	if err != nil {
		return err
	}
	fmt.Println("published", id)
	return nil
}

func Pull(ctx context.Context, subID string, f func(context.Context, *pubsub.Message)) error {
	client, err := pubsub.NewClient(ctx, os.Getenv("GCLOUD_PROJECT"))
	if err != nil {
		return err
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	sub := client.Subscription(subID)
	return sub.Receive(ctx, f)
}
