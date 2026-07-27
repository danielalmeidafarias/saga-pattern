package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gpubsub "cloud.google.com/go/pubsub/v2"

	"github.com/danielalmeidafarias/saga-pattern/internal/saga"
	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/db"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
	msgpubsub "github.com/danielalmeidafarias/saga-pattern/pkg/msg/pubsub"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	database, err := db.OpenSQLite(env("DATABASE_DSN", "/data/saga.db"))
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	store, err := saga.NewSQLiteStore(database)
	if err != nil {
		log.Fatal(err)
	}
	client, err := gpubsub.NewClient(ctx, env("PUBSUB_PROJECT_ID", "saga-local"))
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	broker := msgpubsub.NewPublisher(client)
	defer broker.Close()
	orchestrator := saga.NewSagaOrchestrator(store, store, msg.NewOutboxPublisher(store))
	handler := saga.NewHandler(saga.NewCreateSagaUseCase(store, orderSagaResolver()), orchestrator)
	dispatcher := msg.NewOutboxDispatcher(store, broker)
	recovery := saga.NewProcessSagaUseCase(store, orchestrator)
	go runRecovery(ctx, dispatcher, recovery)
	go serveHealth()
	consumer := msgpubsub.NewConsumer(client.Subscriber(contracts.SubscriptionSaga))
	if consumeErr := consumer.Subscribe(ctx, func(ctx context.Context, message msg.Message) *pkg.Error {
		if message.Type == contracts.OrderCreated {
			return handler.HandleTrigger(ctx, message)
		}
		return handler.HandleResult(ctx, message)
	}); consumeErr != nil {
		log.Fatal(consumeErr.Message)
	}
}

func runRecovery(ctx context.Context, dispatcher *msg.OutboxDispatcher, recovery *saga.ProcessSagaUseCase) {
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := dispatcher.Dispatch(ctx, 100); err != nil {
				log.Printf("outbox: %s", err.Message)
			}
			for _, status := range []saga.Status{saga.StatusRunning, saga.StatusCompensating} {
				if err := recovery.Run(ctx, saga.ProcessSagaUseCaseInput{Status: status, BatchSize: 100}); err != nil {
					log.Printf("recovery: %s", err.Message)
				}
			}
		}
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func serveHealth() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	if err := http.ListenAndServe(env("HTTP_ADDR", ":8084"), mux); err != nil {
		log.Fatal(err)
	}
}
