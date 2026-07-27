package shipping

import (
	"encoding/json"
	"net/http"
)

func NewHTTPHandler(service *Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /shippings/{uuid}", func(w http.ResponseWriter, r *http.Request) {
		shipping, err := service.Get(r.PathValue("uuid"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, shipping)
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

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
