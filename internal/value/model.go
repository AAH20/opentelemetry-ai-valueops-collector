package value

import "time"

type RateCard struct {
	Provider            string    `json:"provider"`
	Model               string    `json:"model"`
	EffectiveFrom       time.Time `json:"effective_from"`
	InputPerMillionUSD  float64   `json:"input_per_million_usd"`
	OutputPerMillionUSD float64   `json:"output_per_million_usd"`
	GPUHourUSD          float64   `json:"gpu_hour_usd"`
	ToolCallUSD         float64   `json:"tool_call_usd"`
}

type TraceEvent struct {
	EventID      string    `json:"event_id"`
	TraceID      string    `json:"trace_id"`
	TenantID     string    `json:"tenant_id"`
	WorkflowID   string    `json:"workflow_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	OccurredAt   time.Time `json:"occurred_at"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	GPUSeconds   float64   `json:"gpu_seconds"`
	ToolCalls    int64     `json:"tool_calls"`
	Attempt      int       `json:"attempt"`
	LatencyMS    float64   `json:"latency_ms"`
}

type Outcome struct {
	TraceID          string  `json:"trace_id"`
	Accepted         bool    `json:"accepted"`
	EvaluationPassed bool    `json:"evaluation_passed"`
	OutcomeType      string  `json:"outcome_type"`
	BusinessValueUSD float64 `json:"business_value_usd"`
	RevenueUSD       float64 `json:"revenue_usd"`
}

type LedgerRow struct {
	TraceID          string  `json:"trace_id"`
	TenantID         string  `json:"tenant_id"`
	WorkflowID       string  `json:"workflow_id"`
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	CostUSD          float64 `json:"cost_usd"`
	Accepted         bool    `json:"accepted"`
	EvaluationPassed bool    `json:"evaluation_passed"`
	BusinessValueUSD float64 `json:"business_value_usd"`
	RevenueUSD       float64 `json:"revenue_usd"`
	GrossMarginPct   float64 `json:"gross_margin_pct"`
	RetryWaste       bool    `json:"retry_waste"`
}

type Summary struct {
	TraceCount                int         `json:"trace_count"`
	AttributedOutcomeCount    int         `json:"attributed_outcome_count"`
	OutcomeAttributionPct     float64     `json:"outcome_attribution_pct"`
	TotalCostUSD              float64     `json:"total_cost_usd"`
	RetryWasteUSD             float64     `json:"retry_waste_usd"`
	RetryWastePct             float64     `json:"retry_waste_pct"`
	CostPerAcceptedOutcomeUSD *float64    `json:"cost_per_accepted_outcome_usd"`
	Rows                      []LedgerRow `json:"rows"`
}
