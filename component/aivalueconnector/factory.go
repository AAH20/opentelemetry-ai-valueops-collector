package aivalueconnector

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

var componentType = component.MustNewType("aivalue")

func NewFactory() connector.Factory {
	return connector.NewFactory(
		componentType,
		createDefaultConfig,
		connector.WithTracesToMetrics(createTracesToMetrics, component.StabilityLevelAlpha),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		Dimensions:    []string{"tenant.id", "gen_ai.operation.name", "gen_ai.provider.name", "gen_ai.request.model"},
		MaxDimensions: 4,
		HashTenantID:  true,
		DropUnpriced:  true,
	}
}

func createTracesToMetrics(
	_ context.Context,
	_ connector.Settings,
	config component.Config,
	next consumer.Metrics,
) (connector.Traces, error) {
	cfg := config.(*Config)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	implementation := &valueConnector{config: *cfg, next: next}
	traceConsumer, err := consumer.NewTraces(
		implementation.consumeTraces,
		consumer.WithCapabilities(consumer.Capabilities{MutatesData: false}),
	)
	if err != nil {
		return nil, err
	}
	return &tracesConnector{Traces: traceConsumer}, nil
}

type tracesConnector struct {
	consumer.Traces
	component.StartFunc
	component.ShutdownFunc
}
