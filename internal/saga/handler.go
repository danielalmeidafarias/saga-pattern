package saga

import (
	"context"

	"github.com/danielalmeidafarias/saga-pattern/pkg"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

// Handler is the Pub/Sub entry point. Subscription wiring belongs to cmd/saga-orchestrator.
type Handler struct {
	createSaga   *CreateSagaUseCase
	orchestrator *SagaOrchestrator
}

func NewHandler(createSaga *CreateSagaUseCase, orchestrator *SagaOrchestrator) *Handler {
	return &Handler{createSaga: createSaga, orchestrator: orchestrator}
}

func (h *Handler) HandleTrigger(ctx context.Context, message msg.Message) *pkg.Error {
	if err := h.createSaga.Run(ctx, CreateSagaUseCaseInput{Message: message}); err != nil {
		return err
	}
	return h.orchestrator.Advance(ctx, message.SagaID)
}

func (h *Handler) HandleResult(ctx context.Context, message msg.Message) *pkg.Error {
	return h.orchestrator.HandleResult(ctx, message)
}
