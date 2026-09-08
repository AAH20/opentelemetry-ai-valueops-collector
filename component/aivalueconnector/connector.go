package aivalueconnector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

type valueConnector struct {
	config Config
	next   consumer.Metrics
}

type measurement struct {
	cost, businessValue, revenue, accepted, retryWaste, attributed float64
	timestamp                                                      pcommon.Timestamp
	attributes                                                     map[string]string
}

func (c *valueConnector) consumeTraces(ctx context.Context, traces ptrace.Traces) error {
	measurements := make([]measurement, 0, traces.SpanCount())
	resourceSpans := traces.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		resource := resourceSpans.At(i)
		scopeSpans := resource.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			spans := scopeSpans.At(j).Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				item, priced := c.measure(span, resource.Resource().Attributes())
				if priced || !c.config.DropUnpriced {
					measurements = append(measurements, item)
				}
			}
		}
	}
	if len(measurements) == 0 {
		return nil
	}
	return c.next.ConsumeMetrics(ctx, buildMetrics(measurements))
}

func (c *valueConnector) measure(span ptrace.Span, resource pcommon.Map) (measurement, bool) {
	attributes := span.Attributes()
	provider := firstString(attributes, "gen_ai.provider.name", "gen_ai.system")
	model := firstString(attributes, "gen_ai.request.model", "gen_ai.response.model")
	input := number(attributes, "gen_ai.usage.input_tokens")
	output := number(attributes, "gen_ai.usage.output_tokens")
	gpuSeconds := number(attributes, "gen_ai.usage.gpu_seconds")
	toolCalls := number(attributes, "gen_ai.usage.tool_calls")
	businessValue := number(attributes, "ai.value.business_value_usd")
	revenue := number(attributes, "ai.value.revenue_usd")
	attempt := number(attributes, "ai.value.attempt")
	if min(input, output, gpuSeconds, toolCalls, businessValue, revenue, attempt) < 0 {
		return measurement{}, false
	}
	rate, priced := c.rate(provider, model, span.StartTimestamp().AsTime())
	cost := input/1_000_000*rate.InputPerMillionUSD + output/1_000_000*rate.OutputPerMillionUSD + gpuSeconds/3600*rate.GPUHourUSD + toolCalls*rate.ToolCallUSD
	accepted, attributed := boolean(attributes, "ai.value.outcome.accepted")
	attributedValue := 0.0
	if attributed {
		attributedValue = 1
	}
	retry := attempt > 1 || (attributed && !accepted)
	retryWaste := 0.0
	if retry {
		retryWaste = cost
	}
	return measurement{
		cost: cost, businessValue: businessValue,
		revenue: revenue, accepted: boolNumber(accepted),
		retryWaste: retryWaste, attributed: attributedValue, timestamp: span.EndTimestamp(),
		attributes: c.dimensions(attributes, resource),
	}, priced
}

func (c *valueConnector) rate(provider, model string, occurred time.Time) (RateCard, bool) {
	candidates := make([]RateCard, 0)
	for _, rate := range c.config.RateCards {
		effective, _ := time.Parse(time.RFC3339, rate.EffectiveFrom)
		if rate.Provider == provider && rate.Model == model && !effective.After(occurred) {
			candidates = append(candidates, rate)
		}
	}
	if len(candidates) == 0 {
		return RateCard{}, false
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].EffectiveFrom > candidates[j].EffectiveFrom })
	return candidates[0], true
}

func (c *valueConnector) dimensions(span, resource pcommon.Map) map[string]string {
	result := map[string]string{}
	for _, key := range c.config.Dimensions {
		value := firstString(span, key)
		if value == "" {
			value = firstString(resource, key)
		}
		if value == "" {
			continue
		}
		if key == "tenant.id" && c.config.HashTenantID {
			digest := sha256.Sum256([]byte(value))
			value = hex.EncodeToString(digest[:8])
		}
		result[key] = value
	}
	return result
}

func buildMetrics(items []measurement) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	scope := metrics.ResourceMetrics().AppendEmpty().ScopeMetrics().AppendEmpty()
	scope.Scope().SetName("github.com/AAH20/opentelemetry-ai-valueops-collector/aivalueconnector")
	definitions := []struct {
		name, unit string
		read       func(measurement) float64
	}{
		{"gen_ai.value.cost", "USD", func(m measurement) float64 { return m.cost }},
		{"gen_ai.value.business_value", "USD", func(m measurement) float64 { return m.businessValue }},
		{"gen_ai.value.revenue", "USD", func(m measurement) float64 { return m.revenue }},
		{"gen_ai.value.accepted_outcomes", "{outcome}", func(m measurement) float64 { return m.accepted }},
		{"gen_ai.value.retry_waste", "USD", func(m measurement) float64 { return m.retryWaste }},
		{"gen_ai.value.attributed_outcomes", "{outcome}", func(m measurement) float64 { return m.attributed }},
	}
	for _, definition := range definitions {
		metric := scope.Metrics().AppendEmpty()
		metric.SetName(definition.name)
		metric.SetUnit(definition.unit)
		sum := metric.SetEmptySum()
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
		sum.SetIsMonotonic(true)
		for _, item := range items {
			point := sum.DataPoints().AppendEmpty()
			point.SetDoubleValue(definition.read(item))
			point.SetTimestamp(item.timestamp)
			for key, value := range item.attributes {
				point.Attributes().PutStr(key, value)
			}
		}
	}
	return metrics
}

func firstString(attributes pcommon.Map, keys ...string) string {
	for _, key := range keys {
		if value, ok := attributes.Get(key); ok && value.Type() == pcommon.ValueTypeStr {
			return value.Str()
		}
	}
	return ""
}
func number(attributes pcommon.Map, key string) float64 {
	value, ok := attributes.Get(key)
	if !ok {
		return 0
	}
	if value.Type() == pcommon.ValueTypeInt {
		return float64(value.Int())
	}
	if value.Type() == pcommon.ValueTypeDouble {
		return value.Double()
	}
	return 0
}
func boolean(attributes pcommon.Map, key string) (bool, bool) {
	value, ok := attributes.Get(key)
	return ok && value.Type() == pcommon.ValueTypeBool && value.Bool(), ok && value.Type() == pcommon.ValueTypeBool
}
func boolNumber(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
