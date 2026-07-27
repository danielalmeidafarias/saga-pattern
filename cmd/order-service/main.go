package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	gpubsub "cloud.google.com/go/pubsub/v2"

	"github.com/danielalmeidafarias/saga-pattern/internal/order"
	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
	msgpubsub "github.com/danielalmeidafarias/saga-pattern/pkg/msg/pubsub"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	database, err := db.OpenSQLite(env("DATABASE_DSN", "/data/order.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	service, err := order.NewService(database, "")
	if err != nil {
		log.Fatal(err)
	}
	client, err := gpubsub.NewClient(ctx, env("PUBSUB_PROJECT_ID", "saga-local"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	publisher := msgpubsub.NewPublisher(client)
	defer publisher.Close()
	go serve(env("HTTP_ADDR", ":8080"), order.NewHTTPHandler(service, publisher))
	consumer := msgpubsub.NewConsumer(client.Subscriber(contracts.SubscriptionOrder))
	if consumeErr := consumer.Subscribe(ctx, order.NewHandler(service).Handle); consumeErr != nil {
		log.Fatal(consumeErr.Message)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func serve(address string, handler http.Handler) {
	if err := http.ListenAndServe(address, handler); err != nil {
		log.Fatal(err)
	}
}
