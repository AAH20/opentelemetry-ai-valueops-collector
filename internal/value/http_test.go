package value

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIRequiresAuthenticationAndCorrelatesOutcome(t *testing.T) {
	engine := testEngine(t)
	handler := API{Engine: engine, APIKey: "secret"}.Handler()
	unauthorized := httptest.NewRequest(http.MethodGet, "/v1/summary", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, unauthorized)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.Code)
	}

	event := `{"event_id":"event-1","trace_id":"trace-1","tenant_id":"tenant-1","workflow_id":"triage","provider":"azure-openai","model":"model","occurred_at":"2026-09-01T00:00:00Z","input_tokens":100000,"output_tokens":50000,"tool_calls":1,"attempt":1,"latency_ms":900}`
	request := httptest.NewRequest(http.MethodPost, "/v1/traces", strings.NewReader(event))
	request.Header.Set("X-API-Key", "secret")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("trace response: %d %s", response.Code, response.Body.String())
	}

	outcome := `{"trace_id":"trace-1","accepted":true,"evaluation_passed":true,"outcome_type":"resolved","business_value_usd":80,"revenue_usd":20}`
	request = httptest.NewRequest(http.MethodPost, "/v1/outcomes", strings.NewReader(outcome))
	request.Header.Set("X-API-Key", "secret")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusAccepted {
		t.Fatalf("outcome response: %d %s", response.Code, response.Body.String())
	}
}
