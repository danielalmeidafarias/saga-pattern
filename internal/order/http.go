package order

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/danielalmeidafarias/saga-pattern/pkg/contracts"
	"github.com/danielalmeidafarias/saga-pattern/pkg/msg"
)

func NewHTTPHandler(service *Service, publisher msg.Publisher) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			UUID  string `json:"uuid"`
			Items []Item `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil || len(request.Items) == 0 {
			http.Error(w, "invalid order", http.StatusBadRequest)
			return
		}
		if request.UUID == "" {
			request.UUID = uuid.NewString()
		}
		amount := 0.0
		contractItems := make([]contracts.OrderItem, 0, len(request.Items))
		for _, item := range request.Items {
			if item.ProductUUID == "" || item.Quantity <= 0 || item.Price < 0 {
				http.Error(w, "invalid order item", http.StatusBadRequest)
				return
			}
			amount += item.Price * float64(item.Quantity)
			contractItems = append(contractItems, contracts.OrderItem{ProductUUID: item.ProductUUID, Name: item.Name, Price: item.Price, Quantity: item.Quantity})
		}
		order := Order{ID: uuid.NewString(), UUID: request.UUID, Items: request.Items, Amount: amount, Status: Pending}
		if err := service.Create(order); err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		payload, _ := json.Marshal(contracts.OrderCreatedPayload{Amount: amount, Items: contractItems})
		event := msg.NewMessage(contracts.TopicSagaEvents, contracts.OrderCreated, payload)
		event.ID, event.OrderID = uuid.NewString(), order.UUID
		if err := publisher.Publish(r.Context(), event); err != nil {
			http.Error(w, err.Message, http.StatusBadGateway)
			return
		}
		writeJSON(w, http.StatusAccepted, order)
	})
	mux.HandleFunc("GET /orders/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		order, err := service.Get(r.PathValue("uuid"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, order)
	})
	mux.HandleFunc("POST /__test/failures", func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Operation string `json:"operation"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid failure", http.StatusBadRequest)
			return
		}
		service.SetFailure(request.Operation)
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("DELETE /__test/failures", func(w http.ResponseWriter, _ *http.Request) {
		service.SetFailure("")
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	return mux
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
