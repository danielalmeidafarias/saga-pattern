package pubsub

import (
	"context"
	"fmt"

	gpubsub "cloud.google.com/go/pubsub/v2"
	"cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
)

var topics = []string{
	contracts.TopicSagaEvents,
	contracts.TopicInventoryCommands,
	contracts.TopicPaymentCommands,
	contracts.TopicShippingCommands,
	contracts.TopicOrderEvents,
}

var subscriptions = map[string]string{
	contracts.SubscriptionSaga:      contracts.TopicSagaEvents,
	contracts.SubscriptionInventory: contracts.TopicInventoryCommands,
	contracts.SubscriptionPayment:   contracts.TopicPaymentCommands,
	contracts.SubscriptionShipping:  contracts.TopicShippingCommands,
	contracts.SubscriptionOrder:     contracts.TopicOrderEvents,
}

func EnsureTopology(ctx context.Context, client *gpubsub.Client, projectID string) error {
	for _, topic := range topics {
		name := fmt.Sprintf("projects/%s/topics/%s", projectID, topic)
		if _, err := client.TopicAdminClient.CreateTopic(ctx, &pubsubpb.Topic{Name: name}); err != nil && status.Code(err) != codes.AlreadyExists {
			return fmt.Errorf("create topic %s: %w", topic, err)
		}
	}
	for subscription, topic := range subscriptions {
		name := fmt.Sprintf("projects/%s/subscriptions/%s", projectID, subscription)
		topicName := fmt.Sprintf("projects/%s/topics/%s", projectID, topic)
		if _, err := client.SubscriptionAdminClient.CreateSubscription(ctx, &pubsubpb.Subscription{Name: name, Topic: topicName}); err != nil && status.Code(err) != codes.AlreadyExists {
			return fmt.Errorf("create subscription %s: %w", subscription, err)
		}
	}
	return nil
}
