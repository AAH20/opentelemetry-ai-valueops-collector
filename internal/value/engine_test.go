package value

import (
	"errors"
	"testing"
	"time"
)

func testEngine(t *testing.T) *Engine {
	t.Helper()
	engine, err := NewEngine([]RateCard{{
		Provider: "azure-openai", Model: "model", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		InputPerMillionUSD: 10, OutputPerMillionUSD: 20, ToolCallUSD: 1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func testEvent() TraceEvent {
	return TraceEvent{EventID: "event-1", TraceID: "trace-1", TenantID: "tenant-1", WorkflowID: "triage", Provider: "azure-openai", Model: "model", OccurredAt: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), InputTokens: 100000, OutputTokens: 50000, ToolCalls: 1, Attempt: 1, LatencyMS: 900}
}

func TestSummaryAttributesCostToAcceptedOutcome(t *testing.T) {
	engine := testEngine(t)
	if err := engine.Ingest(testEvent()); err != nil {
		t.Fatal(err)
	}
	if err := engine.RecordOutcome(Outcome{TraceID: "trace-1", Accepted: true, EvaluationPassed: true, OutcomeType: "incident-resolved", BusinessValueUSD: 80, RevenueUSD: 20}); err != nil {
		t.Fatal(err)
	}
	summary := engine.Summary()
	if summary.TotalCostUSD != 3 || summary.OutcomeAttributionPct != 100 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if summary.CostPerAcceptedOutcomeUSD == nil || *summary.CostPerAcceptedOutcomeUSD != 3 {
		t.Fatalf("unexpected unit cost: %+v", summary.CostPerAcceptedOutcomeUSD)
	}
	if summary.Rows[0].GrossMarginPct != 85 {
		t.Fatalf("unexpected margin: %f", summary.Rows[0].GrossMarginPct)
	}
}

func TestDuplicateAndUnknownTraceAreRejected(t *testing.T) {
	engine := testEngine(t)
	event := testEvent()
	if err := engine.Ingest(event); err != nil {
		t.Fatal(err)
	}
	if err := engine.Ingest(event); !errors.Is(err, ErrDuplicateEvent) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	if err := engine.RecordOutcome(Outcome{TraceID: "missing"}); !errors.Is(err, ErrUnknownTrace) {
		t.Fatalf("expected unknown trace error, got %v", err)
	}
}

func TestHistoricalRateCardIsSelected(t *testing.T) {
	rates := []RateCard{
		{Provider: "p", Model: "m", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), InputPerMillionUSD: 10},
		{Provider: "p", Model: "m", EffectiveFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), InputPerMillionUSD: 100},
	}
	engine, _ := NewEngine(rates)
	event := testEvent()
	event.Provider = "p"
	event.Model = "m"
	if err := engine.Ingest(event); err != nil {
		t.Fatal(err)
	}
	if engine.Summary().TotalCostUSD != 1 {
		t.Fatalf("future rate card changed historical cost")
	}
}
