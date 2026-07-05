package platform

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/hairizuan-tw/gimme-context/internal/coordination"
)

const serviceName = "gimme-context-api"

// Handler returns the API's transport-independent HTTP contract.
func Handler(ready func() bool) http.Handler {
	return HandlerWithStore(ready, coordination.NewStore())
}

func HandlerWithStore(ready func() bool, store *coordination.Store) http.Handler {
	store.SetContextService(coordination.ContextService{
		Prometheus: coordination.HTTPMetricsClient{BaseURL: os.Getenv("PROMETHEUS_URL")},
		Loki:       coordination.HTTPLokiClient{BaseURL: os.Getenv("LOKI_URL")},
		Retries:    2,
	})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", health(http.StatusOK, "live"))
	mux.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		if !ready() {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/v1", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": serviceName, "version": "v1"})
	})
	coordination.Register(mux, store)
	return mux
}

func health(status int, state string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, status, map[string]string{"status": state})
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
