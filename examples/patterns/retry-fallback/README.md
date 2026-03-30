# Multi-Provider Fallback Pattern

Demonstrates automatic failover between multiple LLM providers for improved reliability.

> Chain multiple LLM providers together so a failed request automatically reroutes to the next available provider without changing your application code.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## What this demonstrates

**Difficulty: Starter** 🟢 · LLM

- **Summary:** Retry and fallback strategies across providers
- **Scenario:** Automatically retry failed requests and fall back to alternate providers
- **`tech_tags` in manifest:** `LLM` — example id **`retry-fallback`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).
- **Models:** Set cloud provider API keys and/or run **Ollama** locally when you execute the **Run** steps (interactive flags like `--help` / `--verbose` are documented below).

## Key nxusKit Features Demonstrated

| Feature | Description |
|---------|-------------|
| **Provider Abstraction** | `LLMProvider` trait/interface enables heterogeneous provider collections |
| **Unified Error Handling** | Same error types across all providers enable consistent fallback logic |
| **Provider Interchangeability** | Swap providers without changing application code |

**Provider Compatibility**: Claude, OpenAI, Ollama, and any custom provider implementing `LLMProvider`

## Pattern Overview

When working with LLM APIs, network issues, rate limits, or provider outages can interrupt service. This pattern chains multiple providers together, automatically falling back to the next provider when one fails.

## Key Features

- Sequential provider chain with automatic failover
- First successful response wins
- Error aggregation for debugging
- Provider-agnostic implementation

## Real-World Application

High-availability AI service, resilient inference pipeline.

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
# From `/examples/patterns/retry-fallback`:
cd rust && cargo build
cd go && make build
```
## Library usage

### Rust

```rust
use retry_fallback::chat_with_fallback;
use nxuskit::prelude::*;

// Create multiple providers (could be different providers in production)
let providers: Vec<Box<dyn LLMProvider>> = vec![
    Box::new(OllamaProvider::builder().base_url("http://primary:11434").model("llama3").build()?),
    Box::new(OllamaProvider::builder().base_url("http://secondary:11434").model("llama3").build()?),
];

// Request automatically falls back on failure
let response = chat_with_fallback(&providers, &request).await?;
```

### Go

```go
import llmkit "github.com/llmkit/nxuskit-go"

// Create multiple providers
providers := []llmkit.LLMProvider{provider1, provider2, provider3}

// Request automatically falls back on failure
resp, err := ChatWithFallback(ctx, providers, req)
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

Both implementations include unit tests using mock providers:

```bash
# Rust
cd rust && cargo test

# Go
cd go && go test -v
```

## Production Considerations

1. **Provider diversity**: Use different providers (OpenAI, Claude, Ollama) for true redundancy
2. **Timeout handling**: Set appropriate timeouts to avoid long waits before failover
3. **Circuit breakers**: Consider adding circuit breaker pattern for repeated failures
4. **Metrics**: Track which providers are failing to identify systemic issues
