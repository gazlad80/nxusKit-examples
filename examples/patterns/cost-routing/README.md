# Model Router (Cost Tiers) Pattern

Demonstrates intelligent routing of requests to different models based on task complexity to optimize cost and quality.

> Route every LLM request to the right model for the job, so you stop paying premium prices for economy-grade tasks.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## What this demonstrates

**Difficulty: Starter** 🟢 · LLM

- **Summary:** Cost-aware provider routing and selection
- **Scenario:** Route requests to the cheapest capable provider
- **`tech_tags` in manifest:** `LLM` — example id **`cost-routing`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).
- **Models:** Set cloud provider API keys and/or run **Ollama** locally when you execute the **Run** steps (interactive flags like `--help` / `--verbose` are documented below).

## Key nxusKit Features Demonstrated

| Feature | Description |
|---------|-------------|
| **Provider Abstraction** | Same interface for all models enables seamless tier switching |
| **Request Portability** | `ChatRequest` works identically across model tiers |
| **Token Normalization** | Consistent token usage tracking for cost calculations |

**Provider Compatibility**: Works with any provider supporting multiple models (OpenAI, Ollama, etc.)

## Pattern Overview

Not all prompts need the most expensive model. This pattern analyzes prompt characteristics and routes requests to an appropriate cost tier, balancing quality requirements with cost efficiency.

## Cost Tiers

| Tier | Model | Use Case | Cost |
|------|-------|----------|------|
| Economy | gpt-4o-mini | Simple queries, lookups | $ |
| Standard | gpt-4o | General tasks, explanations | $$ |
| Premium | gpt-4-turbo | Complex analysis, reasoning | $$$ |

## Classification Heuristics

The example uses simple heuristics to classify tasks:

1. **Premium tier triggers**:
   - Keywords: "analyze", "compare", "evaluate", "synthesize", "critique"
   - Prompt length > 1000 characters

2. **Standard tier**:
   - Prompt length > 200 characters (without premium keywords)

3. **Economy tier**:
   - Short, simple prompts

## Real-World Application

Cost-optimized AI platform, budget-aware inference.

## Technologies

LLM

## Language Implementations

| Language | Path | Status |
|----------|------|--------|
| Rust | `rust/` | Available |
| Go | `go/` | Available |

## Build

Attach an **installed SDK** (`NXUSKIT_SDK_DIR`). See the repository [README.md](../../../README.md) and `scripts/test-examples.sh`.

```bash
# From `/examples/patterns/cost-routing`:
cd rust && cargo build
cd go && make build
```
## Library usage

### Rust

```rust
use cost_routing::{classify_task, routed_chat, CostTier};

// Classify without making API call
let tier = classify_task(prompt);
println!("Would use: {} tier", tier.as_str());

// Or route and execute
let result = routed_chat(&provider, prompt).await?;
println!("Used {} tier", result.tier.as_str());
```

### Go

```go
// Classify without making API call
tier := ClassifyTask(prompt)
fmt.Println("Would use:", tier.Name(), "tier")

// Or route and execute
result, err := RoutedChat(ctx, provider, prompt)
fmt.Println("Used", result.Tier.Name(), "tier")
```

## Run

### Rust
```bash
cd rust
cargo run
```

### Go
```bash
cd go
go run .
```

## Interactive Modes

All examples support debugging flags:

```bash
# Verbose mode - show raw HTTP request/response data
cargo run -- --verbose      # Rust
go run . --verbose          # Go

# Step mode - pause at each step with explanations
cargo run -- --step         # Rust
go run . --step             # Go

# Combined mode
cargo run -- --verbose --step
```

Or use environment variables:
```bash
export NXUSKIT_VERBOSE=1
export NXUSKIT_STEP=1
```

## Testing

```bash
# Rust
cd rust && cargo test

# Go
cd go && go test -v
```

## Advanced Classification

For production, consider more sophisticated approaches:

1. **Intent classification**: Use a small model to classify intent first
2. **Historical data**: Learn from past prompt/tier success rates
3. **User preferences**: Allow users to specify quality requirements
4. **Dynamic pricing**: Adjust tiers based on current API pricing

## Production Considerations

1. **Calibration**: Tune thresholds based on your specific use cases
2. **Monitoring**: Track accuracy and cost savings by tier
3. **Override capability**: Allow manual tier selection for edge cases
4. **A/B testing**: Validate routing decisions improve outcomes
