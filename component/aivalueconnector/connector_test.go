package aivalueconnector

import (
	"context"
	"testing"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func testConfig() *Config {
	return &Config{
		Dimensions:    []string{"tenant.id", "gen_ai.provider.name", "gen_ai.request.model"},
		MaxDimensions: 3,
		HashTenantID:  true,
		DropUnpriced:  true,
		RateCards: []RateCard{{
			Provider:            "nvidia",
			Model:               "nim-model",
			EffectiveFrom:       "2026-01-01T00:00:00Z",
			InputPerMillionUSD:  10,
			OutputPerMillionUSD: 20,
			GPUHourUSD:          4,
			ToolCallUSD:         1,
		}},
	}
}

func testTraces() ptrace.Traces {
	traces := ptrace.NewTraces()
	span := traces.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetStartTimestamp(1788220800000000000)
	span.SetEndTimestamp(1788220801000000000)
	attributes := span.Attributes()
	attributes.PutStr("tenant.id", "customer-one")
	attributes.PutStr("gen_ai.provider.name", "nvidia")
	attributes.PutStr("gen_ai.request.model", "nim-model")
	attributes.PutInt("gen_ai.usage.input_tokens", 100000)
	attributes.PutInt("gen_ai.usage.output_tokens", 50000)
	attributes.PutDouble("gen_ai.usage.gpu_seconds", 900)
	attributes.PutInt("gen_ai.usage.tool_calls", 1)
	attributes.PutBool("ai.value.outcome.accepted", true)
	attributes.PutDouble("ai.value.business_value_usd", 100)
	attributes.PutDouble("ai.value.revenue_usd", 20)
	attributes.PutInt("ai.value.attempt", 1)
	return traces
}

func TestConnectorConvertsTracesToEconomicMetrics(t *testing.T) {
	var captured pmetric.Metrics
	next, err := consumer.NewMetrics(func(_ context.Context, metrics pmetric.Metrics) error {
		captured = pmetric.NewMetrics()
		metrics.CopyTo(captured)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	factory := NewFactory()
	instance, err := factory.CreateTracesToMetrics(
		context.Background(),
		connector.Settings{ID: component.NewID(componentType)},
		testConfig(),
		next,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := instance.ConsumeTraces(context.Background(), testTraces()); err != nil {
		t.Fatal(err)
	}
	metrics := captured.ResourceMetrics().At(0).ScopeMetrics().At(0).Metrics()
	if metrics.Len() != 6 {
		t.Fatalf("expected six metrics, got %d", metrics.Len())
	}
	if metrics.At(0).Name() != "gen_ai.value.cost" {
		t.Fatalf("unexpected first metric %q", metrics.At(0).Name())
	}
	cost := metrics.At(0).Sum().DataPoints().At(0)
	if cost.DoubleValue() != 4 {
		t.Fatalf("expected cost 4, got %f", cost.DoubleValue())
	}
	tenant, ok := cost.Attributes().Get("tenant.id")
	if !ok || tenant.Str() == "customer-one" || len(tenant.Str()) != 16 {
		t.Fatalf("tenant ID was not bounded and hashed: %q", tenant.Str())
	}
}

func TestConfigRejectsUnboundedDimensions(t *testing.T) {
	config := testConfig()
	config.Dimensions = append(config.Dimensions, "untrusted.attribute")
	config.MaxDimensions = 4
	if err := config.Validate(); err == nil {
		t.Fatal("expected unsupported dimension to be rejected")
	}
}

func TestUnpricedSpanIsDropped(t *testing.T) {
	consumed := false
	next, _ := consumer.NewMetrics(func(context.Context, pmetric.Metrics) error {
		consumed = true
		return nil
	})
	implementation := &valueConnector{config: *testConfig(), next: next}
	traces := testTraces()
	traces.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).Attributes().PutStr(
		"gen_ai.request.model", "unpriced",
	)
	if err := implementation.consumeTraces(context.Background(), traces); err != nil {
		t.Fatal(err)
	}
	if consumed {
		t.Fatal("unpriced span reached the metrics pipeline")
	}
}
