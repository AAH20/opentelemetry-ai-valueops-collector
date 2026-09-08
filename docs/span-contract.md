# AI value span contract

The alpha connector reads a small, allowlisted set of numeric and categorical attributes. It does
not read prompts, completions, message bodies, authorization headers or arbitrary attributes.

| Attribute | Type | Purpose |
|---|---|---|
| `gen_ai.provider.name` | string | Rate-card provider key |
| `gen_ai.request.model` | string | Rate-card model key |
| `gen_ai.operation.name` | string | Bounded workflow dimension |
| `gen_ai.usage.input_tokens` | integer/double | Input cost driver |
| `gen_ai.usage.output_tokens` | integer/double | Output cost driver |
| `gen_ai.usage.gpu_seconds` | integer/double | Self-hosted GPU cost driver |
| `gen_ai.usage.tool_calls` | integer/double | Tool cost driver |
| `tenant.id` | string | Hashed by default before metric export |
| `ai.value.outcome.accepted` | boolean | Accepted-outcome increment |
| `ai.value.business_value_usd` | integer/double | Customer-supplied value evidence |
| `ai.value.revenue_usd` | integer/double | Customer-supplied revenue evidence |
| `ai.value.attempt` | integer/double | Retry-waste classification |

Rate cards are effective-dated. A span with no applicable card is dropped by default; silently
assuming a zero price would corrupt unit economics. Generated metrics use delta temporality and
carry no more than the configured dimension ceiling. Only six known dimension names are accepted.

Business value and revenue are evidence inputs, not verified savings. Verification requires
reconciliation against downstream billing and business-system records.
