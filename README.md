# OpenTelemetry AI ValueOps Collector

Vendor-neutral AI unit economics for **OpenTelemetry, Azure AI Foundry, NVIDIA NIM, vLLM,
Kubernetes, LangChain, LangGraph, CrewAI and MCP**.

[Architecture](docs/architecture.md) · [Span contract](docs/span-contract.md) ·
[Security](SECURITY.md) · [A2Z SOC](https://a2zsoc.com)

This repository is an executable OSS foundation for connecting an AI trace to its model,
GPU and tool cost and then to an accepted business outcome. It answers a harder question than
cost per token: **what did each accepted outcome cost, and did the tenant remain profitable?**

## Working capabilities

- Go HTTP service with authenticated trace, outcome, summary and Prometheus endpoints
- Versioned rate cards selected by event timestamp
- Token, GPU-second and tool-call cost calculation
- Trace-to-outcome correlation without prompt or completion collection
- Cost per accepted outcome, gross margin, attribution coverage and retry-waste KPIs
- Duplicate event and trace collision rejection
- Unit and API tests, race detection, container build, Helm chart and Azure Terraform foundation
- Native OpenTelemetry traces-to-metrics `aivalue` connector
- Collector Builder manifest with OTLP, batching, memory limiting and Prometheus export

The native connector is alpha. Durable business-outcome correlation remains in the authenticated
service while the connector converts outcome attributes already present on spans. Implemented and
planned boundaries are documented in [the architecture](docs/architecture.md).

## Build the Collector distribution

```bash
go install go.opentelemetry.io/collector/cmd/builder@v0.159.0
builder --config distribution/builder-config.yaml
./distribution/generated/otelcol-ai-valueops --config distribution/collector-config.yaml
```

The example pipeline receives OTLP on ports `4317` and `4318` and exports bounded economic
metrics for Prometheus on port `9464`.

## Run locally

```bash
export VALUEOPS_API_KEY='replace-with-a-random-secret'
go run ./cmd/valueops
```

In another shell:

```bash
curl -sS -X POST http://localhost:8080/v1/traces \
  -H 'Content-Type: application/json' -H "X-API-Key: $VALUEOPS_API_KEY" \
  --data-binary @examples/trace.json

curl -sS -X POST http://localhost:8080/v1/outcomes \
  -H 'Content-Type: application/json' -H "X-API-Key: $VALUEOPS_API_KEY" \
  --data-binary @examples/outcome.json

curl -sS http://localhost:8080/v1/summary -H "X-API-Key: $VALUEOPS_API_KEY"
curl -sS http://localhost:8080/metrics -H "X-API-Key: $VALUEOPS_API_KEY"
```

## API contract

| Endpoint | Purpose |
|---|---|
| `POST /v1/traces` | Ingest normalized model/GPU/tool execution metadata |
| `POST /v1/outcomes` | Correlate a business outcome using `trace_id` |
| `GET /v1/summary` | Return the in-memory economic ledger |
| `GET /metrics` | Export bounded Prometheus KPIs |
| `GET /healthz` | Kubernetes liveness and readiness |

The MVP intentionally keeps no durable state. Production adapters must add OIDC or mTLS,
tenant authorization, a durable ledger, retention controls and native OTLP before handling real
customer telemetry.

## Roadmap

1. Add PostgreSQL and ClickHouse append-only ledgers with migrations.
2. Add OpenCost, DCGM, NIM, Azure Cost Management and Foundry evaluation adapters.
3. Add bounded state for asynchronous outcome correlation inside the Collector pipeline.
4. Generate review-gated GitOps routing proposals and shadow-test evidence.
5. Reconcile post-change billing before recording independently verified savings.

## Commercial implementation boundary

Indicative ranges: USD 7,500–20,000 for discovery, USD 20,000–60,000 for instrumentation,
USD 75,000–250,000 for production implementation and USD 8,000–35,000 monthly for managed AI
ValueOps. These are positioning ranges, not quotes or guaranteed savings.

Apache-2.0 licensed. Contributions that improve interoperability, evidence quality and safe
default behavior are welcome.
