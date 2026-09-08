package value

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

type API struct {
	Engine *Engine
	APIKey string
}

func (a API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.Handle("POST /v1/traces", a.authorize(http.HandlerFunc(a.ingest)))
	mux.Handle("POST /v1/outcomes", a.authorize(http.HandlerFunc(a.outcome)))
	mux.Handle("GET /v1/summary", a.authorize(http.HandlerFunc(a.summary)))
	mux.Handle("GET /metrics", a.authorize(http.HandlerFunc(a.metrics)))
	return mux
}

func (a API) authorize(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		provided := r.Header.Get("X-API-Key")
		if a.APIKey == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(a.APIKey)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decode(w http.ResponseWriter, r *http.Request, destination any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func statusFor(err error) int {
	if errors.Is(err, ErrDuplicateEvent) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrUnknownTrace) {
		return http.StatusNotFound
	}
	return http.StatusUnprocessableEntity
}

func (a API) ingest(w http.ResponseWriter, r *http.Request) {
	var event TraceEvent
	if !decode(w, r, &event) {
		return
	}
	if err := a.Engine.Ingest(event); err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}
func (a API) outcome(w http.ResponseWriter, r *http.Request) {
	var outcome Outcome
	if !decode(w, r, &outcome) {
		return
	}
	if err := a.Engine.RecordOutcome(outcome); err != nil {
		http.Error(w, err.Error(), statusFor(err))
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "correlated"})
}
func (a API) summary(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, a.Engine.Summary())
}
func (a API) metrics(w http.ResponseWriter, _ *http.Request) {
	summary := a.Engine.Summary()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprintf(w, "ai_value_traces_total %d\nai_value_outcome_attribution_ratio %g\nai_value_cost_usd %g\nai_value_retry_waste_usd %g\n", summary.TraceCount, summary.OutcomeAttributionPct/100, summary.TotalCostUSD, summary.RetryWasteUSD)
}
