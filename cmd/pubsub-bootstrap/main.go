package main

import (
	"context"
	"log"
	"os"
	"time"

	gpubsub "cloud.google.com/go/pubsub/v2"

	msgpubsub "github.com/danielalmeidafarias/saga-pattern/pkg/msg/pubsub"
)

func main() {
	ctx := context.Background()
	projectID := env("PUBSUB_PROJECT_ID", "saga-local")
	for attempt := 1; attempt <= 30; attempt++ {
		client, err := gpubsub.NewClient(ctx, projectID)
		if err == nil {
			err = msgpubsub.EnsureTopology(ctx, client, projectID)
			_ = client.Close()
		}
		if err == nil {
			log.Print("Pub/Sub topology ready")
			return
		}
		if attempt == 30 {
			log.Fatal(err)
		}
		time.Sleep(time.Second)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
