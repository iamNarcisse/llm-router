package server

import (
	"encoding/json"
	"net/http"

	router "llm-router/services/router/internal/router"
)

func NewHealthHandler(r *router.Router) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, req *http.Request) {
		embeddingOK, qdrantOK := r.HealthCheck(req.Context())

		status := http.StatusOK
		if !embeddingOK || !qdrantOK {
			status = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(map[string]any{
			"healthy":   embeddingOK && qdrantOK,
			"embedding": embeddingOK,
			"qdrant":    qdrantOK,
		})
	})

	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	return mux
}
