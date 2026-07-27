package payment

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
	paymentUUID := uuid.NewSHA1(uuid.NameSpaceOID, []byte("payment:"+command.OrderID)).String()
	switch command.Type {
	case contracts.CreatePaymentRequested:
		var payload contracts.OrderCreatedPayload
		if err := json.Unmarshal(command.Payload, &payload); err != nil {
			return pkg.NewError("INVALID_MESSAGE", "decode payment command", err)
		}
		_, err := h.service.Get(paymentUUID)
		if errors.Is(err, sql.ErrNoRows) {
			err = h.service.Create(Payment{ID: uuid.NewString(), UUID: paymentUUID, Amount: payload.Amount, Method: Pix, Status: Succeeded})
		}
		return h.reply(ctx, command, contracts.PaymentCreated, contracts.PaymentCreationFailed, contracts.ResourceCreated{UUID: paymentUUID}, err)
	case contracts.CancelPaymentRequested:
		err := h.service.Cancel(paymentUUID)
		return h.reply(ctx, command, contracts.PaymentCanceled, contracts.PaymentCancelFailed, nil, err)
	default:
		return pkg.NewError("INVALID_MESSAGE", "unsupported payment command", nil)
	}
}

func (h *Handler) reply(ctx context.Context, command msg.Message, successType, failureType msg.MessageType, payload any, businessErr error) *pkg.Error {
	result := msg.NewMessage(contracts.TopicSagaEvents, successType, nil)
	result.ID = command.ID + ":result"
	result.SagaID, result.StepID, result.OrderID = command.SagaID, command.StepID, command.OrderID
	if businessErr != nil {
		result.Type = failureType
		result.Failure = &msg.Failure{Code: "PAYMENT_FAILED", Message: businessErr.Error()}
	} else if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return pkg.NewError("SERIALIZATION_ERROR", "serialize payment result", err)
		}
		result.Payload = encoded
	}
	return h.publisher.Publish(ctx, result)
}
