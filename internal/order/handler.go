package order

import (
	"context"
	"encoding/json"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Handle(_ context.Context, message msg.Message) *pkg.Error {
	var err error
	switch message.Type {
	case contracts.OrderCreationSucceeded:
		var payload contracts.OrderCreationSucceededPayload
		if decodeErr := json.Unmarshal(message.Payload, &payload); decodeErr != nil {
			return pkg.NewError("INVALID_MESSAGE", "decode order completion", decodeErr)
		}
		err = h.service.Complete(message.OrderID, payload.PaymentUUID, payload.ShippingUUID)
	case contracts.OrderCreationFailed:
		err = h.service.Fail(message.OrderID)
	default:
		return pkg.NewError("INVALID_MESSAGE", "unsupported order event", nil)
	}
	if err != nil {
		return pkg.NewError("ORDER_UPDATE_FAILED", "apply order event", err)
	}
	return nil
}
