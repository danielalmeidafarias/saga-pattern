package inventory

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

type Handler struct {
	service   *Service
	publisher msg.Publisher
}

func NewHandler(service *Service, publisher msg.Publisher) *Handler {
	return &Handler{service: service, publisher: publisher}
}

func (h *Handler) Handle(ctx context.Context, command msg.Message) *pkg.Error {
	switch command.Type {
	case contracts.ReserveInventoryRequested:
		reservationUUID := namedUUID("inventory", command.OrderID)
		err := h.service.Reserve(Reservation{UUID: reservationUUID, OrderUUID: command.OrderID, Status: Reserved})
		return h.reply(ctx, command, contracts.InventoryReserved, contracts.InventoryReservationFailed, contracts.ResourceCreated{UUID: reservationUUID}, err)
	case contracts.ReleaseInventoryRequested:
		err := h.service.Release(command.OrderID)
		return h.reply(ctx, command, contracts.InventoryReleased, contracts.InventoryReleaseFailed, nil, err)
	default:
		return pkg.NewError("INVALID_MESSAGE", "unsupported inventory command", nil)
	}
}

func (h *Handler) reply(ctx context.Context, command msg.Message, successType, failureType msg.MessageType, payload any, businessErr error) *pkg.Error {
	result := msg.NewMessage(contracts.TopicSagaEvents, successType, nil)
	result.ID = command.ID + ":result"
	result.SagaID, result.StepID, result.OrderID = command.SagaID, command.StepID, command.OrderID
	if businessErr != nil {
		result.Type = failureType
		result.Failure = &msg.Failure{Code: "INVENTORY_FAILED", Message: businessErr.Error()}
	} else if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return pkg.NewError("SERIALIZATION_ERROR", "serialize inventory result", err)
		}
		result.Payload = encoded
	}
	return h.publisher.Publish(ctx, result)
}

func namedUUID(service, orderID string) string {
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(service+":"+orderID)).String()
}
