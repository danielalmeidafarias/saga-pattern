package shipping

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

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
	shippingUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("shipping:"+command.OrderID)).String()
	switch command.Type {
	case contracts.CreateShippingRequested:
		_, err := h.service.Get(shippingUUID)
		if errors.Is(err, sql.ErrNoRows) {
			err = h.service.Create(Shipping{ID: uuid.NewString(), UUID: shippingUUID, OrderUUID: command.OrderID, Status: Created})
		}
		return h.reply(ctx, command, contracts.ShippingCreated, contracts.ShippingCreationFailed, contracts.ResourceCreated{UUID: shippingUUID}, err)
	case contracts.CancelShippingRequested:
		err := h.service.Cancel(shippingUUID)
		return h.reply(ctx, command, contracts.ShippingCanceled, contracts.ShippingCancelFailed, nil, err)
	default:
		return pkg.NewError("INVALID_MESSAGE", "unsupported shipping command", nil)
	}
}

func (h *Handler) reply(ctx context.Context, command msg.Message, successType, failureType msg.MessageType, payload any, businessErr error) *pkg.Error {
	result := msg.NewMessage(contracts.TopicSagaEvents, successType, nil)
	result.ID = command.ID + ":result"
	result.SagaID, result.StepID, result.OrderID = command.SagaID, command.StepID, command.OrderID
	if businessErr != nil {
		result.Type = failureType
		result.Failure = &msg.Failure{Code: "SHIPPING_FAILED", Message: businessErr.Error()}
	} else if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return pkg.NewError("SERIALIZATION_ERROR", "serialize shipping result", err)
		}
		result.Payload = encoded
	}
	return h.publisher.Publish(ctx, result)
}
