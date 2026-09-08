package aivalueconnector

import (
	"errors"
	"fmt"
	"time"
)

const maxDimensionLimit = 6

var allowedDimensions = map[string]struct{}{
	"tenant.id": {}, "gen_ai.operation.name": {}, "gen_ai.provider.name": {},
	"gen_ai.request.model": {}, "service.name": {}, "deployment.environment.name": {},
}

type RateCard struct {
	Provider            string  `mapstructure:"provider"`
	Model               string  `mapstructure:"model"`
	EffectiveFrom       string  `mapstructure:"effective_from"`
	InputPerMillionUSD  float64 `mapstructure:"input_per_million_usd"`
	OutputPerMillionUSD float64 `mapstructure:"output_per_million_usd"`
	GPUHourUSD          float64 `mapstructure:"gpu_hour_usd"`
	ToolCallUSD         float64 `mapstructure:"tool_call_usd"`
}

type Config struct {
	Dimensions    []string   `mapstructure:"dimensions"`
	MaxDimensions int        `mapstructure:"max_dimensions"`
	HashTenantID  bool       `mapstructure:"hash_tenant_id"`
	DropUnpriced  bool       `mapstructure:"drop_unpriced"`
	RateCards     []RateCard `mapstructure:"rate_cards"`
}

func (cfg *Config) Validate() error {
	if cfg.MaxDimensions < 1 || cfg.MaxDimensions > maxDimensionLimit {
		return fmt.Errorf("max_dimensions must be between 1 and %d", maxDimensionLimit)
	}
	if len(cfg.Dimensions) > cfg.MaxDimensions {
		return errors.New("dimensions exceed max_dimensions")
	}
	seen := map[string]struct{}{}
	for _, dimension := range cfg.Dimensions {
		if _, ok := allowedDimensions[dimension]; !ok {
			return fmt.Errorf("unsupported dimension %q", dimension)
		}
		if _, ok := seen[dimension]; ok {
			return fmt.Errorf("duplicate dimension %q", dimension)
		}
		seen[dimension] = struct{}{}
	}
	for _, rate := range cfg.RateCards {
		if rate.Provider == "" || rate.Model == "" {
			return errors.New("rate card provider and model are required")
		}
		if _, err := time.Parse(time.RFC3339, rate.EffectiveFrom); err != nil {
			return fmt.Errorf("invalid effective_from for %s/%s: %w", rate.Provider, rate.Model, err)
		}
		if min(rate.InputPerMillionUSD, rate.OutputPerMillionUSD, rate.GPUHourUSD, rate.ToolCallUSD) < 0 {
			return errors.New("rate card values cannot be negative")
		}
	}
	return nil
}
