package value

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

var (
	ErrDuplicateEvent = errors.New("duplicate event_id")
	ErrUnknownRate    = errors.New("no effective rate card")
	ErrUnknownTrace   = errors.New("unknown trace_id")
)

type Engine struct {
	mu       sync.RWMutex
	rates    []RateCard
	events   map[string]TraceEvent
	traceIDs map[string]string
	outcomes map[string]Outcome
}

func NewEngine(rates []RateCard) (*Engine, error) {
	for _, rate := range rates {
		if rate.Provider == "" || rate.Model == "" || rate.EffectiveFrom.IsZero() {
			return nil, errors.New("rate card requires provider, model and effective_from")
		}
		if min(rate.InputPerMillionUSD, rate.OutputPerMillionUSD, rate.GPUHourUSD, rate.ToolCallUSD) < 0 {
			return nil, errors.New("rate card values cannot be negative")
		}
	}
	return &Engine{rates: rates, events: map[string]TraceEvent{}, traceIDs: map[string]string{}, outcomes: map[string]Outcome{}}, nil
}

func (e *Engine) Ingest(event TraceEvent) error {
	if event.EventID == "" || event.TraceID == "" || event.TenantID == "" || event.WorkflowID == "" {
		return errors.New("event_id, trace_id, tenant_id and workflow_id are required")
	}
	if min(float64(event.InputTokens), float64(event.OutputTokens), event.GPUSeconds, float64(event.ToolCalls), event.LatencyMS) < 0 || event.Attempt < 1 {
		return errors.New("event quantities cannot be negative and attempt must be positive")
	}
	if _, err := e.rateFor(event); err != nil {
		return err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.events[event.EventID]; exists {
		return ErrDuplicateEvent
	}
	if existing, exists := e.traceIDs[event.TraceID]; exists && existing != event.EventID {
		return fmt.Errorf("trace_id already belongs to event %s", existing)
	}
	e.events[event.EventID] = event
	e.traceIDs[event.TraceID] = event.EventID
	return nil
}

func (e *Engine) RecordOutcome(outcome Outcome) error {
	if outcome.TraceID == "" || outcome.BusinessValueUSD < 0 || outcome.RevenueUSD < 0 {
		return errors.New("valid trace_id and non-negative outcome values are required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.traceIDs[outcome.TraceID]; !exists {
		return ErrUnknownTrace
	}
	e.outcomes[outcome.TraceID] = outcome
	return nil
}

func (e *Engine) rateFor(event TraceEvent) (RateCard, error) {
	candidates := make([]RateCard, 0)
	for _, rate := range e.rates {
		if rate.Provider == event.Provider && rate.Model == event.Model && !rate.EffectiveFrom.After(event.OccurredAt) {
			candidates = append(candidates, rate)
		}
	}
	if len(candidates) == 0 {
		return RateCard{}, fmt.Errorf("%w for %s/%s at %s", ErrUnknownRate, event.Provider, event.Model, event.OccurredAt.Format(time.RFC3339))
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].EffectiveFrom.After(candidates[j].EffectiveFrom) })
	return candidates[0], nil
}

func eventCost(event TraceEvent, rate RateCard) float64 {
	return float64(event.InputTokens)/1_000_000*rate.InputPerMillionUSD +
		float64(event.OutputTokens)/1_000_000*rate.OutputPerMillionUSD +
		event.GPUSeconds/3600*rate.GPUHourUSD + float64(event.ToolCalls)*rate.ToolCallUSD
}

func round(value float64) float64 { return math.Round(value*1_000_000) / 1_000_000 }

func (e *Engine) Summary() Summary {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := Summary{Rows: []LedgerRow{}}
	accepted := 0
	for _, event := range e.events {
		rate, _ := e.rateFor(event)
		cost := eventCost(event, rate)
		outcome, attributed := e.outcomes[event.TraceID]
		row := LedgerRow{TraceID: event.TraceID, TenantID: event.TenantID, WorkflowID: event.WorkflowID, Provider: event.Provider, Model: event.Model, CostUSD: round(cost)}
		if attributed {
			result.AttributedOutcomeCount++
			row.Accepted = outcome.Accepted
			row.EvaluationPassed = outcome.EvaluationPassed
			row.BusinessValueUSD = outcome.BusinessValueUSD
			row.RevenueUSD = outcome.RevenueUSD
			if outcome.RevenueUSD > 0 {
				row.GrossMarginPct = round((outcome.RevenueUSD - cost) / outcome.RevenueUSD * 100)
			}
			if outcome.Accepted {
				accepted++
			}
		}
		row.RetryWaste = event.Attempt > 1 || (attributed && !outcome.Accepted)
		if row.RetryWaste {
			result.RetryWasteUSD += cost
		}
		result.TotalCostUSD += cost
		result.Rows = append(result.Rows, row)
	}
	result.TraceCount = len(result.Rows)
	if result.TraceCount > 0 {
		result.OutcomeAttributionPct = round(float64(result.AttributedOutcomeCount) / float64(result.TraceCount) * 100)
	}
	if result.TotalCostUSD > 0 {
		result.RetryWastePct = round(result.RetryWasteUSD / result.TotalCostUSD * 100)
	}
	if accepted > 0 {
		amount := round(result.TotalCostUSD / float64(accepted))
		result.CostPerAcceptedOutcomeUSD = &amount
	}
	result.TotalCostUSD, result.RetryWasteUSD = round(result.TotalCostUSD), round(result.RetryWasteUSD)
	sort.Slice(result.Rows, func(i, j int) bool { return result.Rows[i].TraceID < result.Rows[j].TraceID })
	return result
}
