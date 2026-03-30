# Structured Output (JSON Mode) Pattern

Demonstrates extracting typed, validated structured data from LLM responses using JSON mode.

> Stop parsing free-form LLM text — get typed, validated JSON structs back every time, across every provider.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## What this demonstrates

**Difficulty: Starter** 🟢 · LLM

- **Summary:** JSON mode and structured output generation
- **Scenario:** Request and parse structured JSON responses from an LLM
- **`tech_tags` in manifest:** `LLM` — example id **`structured-output`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).
- **Models:** Set cloud provider API keys and/or run **Ollama** locally when you execute the **Run** steps (interactive flags like `--help` / `--verbose` are documented below).

## Key nxusKit Features Demonstrated

| Feature | Description |
|---------|-------------|
| **JSON Mode Abstraction** | nxusKit handles provider-specific JSON mode implementations |
| **Type-Safe Responses** | Strong typing with serde (Rust) / json (Go) for reliability |
| **Provider-Agnostic Schemas** | Same schema works across OpenAI, Claude, and Ollama |

**Provider Compatibility**: Ollama (format parameter), OpenAI (response_format), Claude (via tool use)

## Pattern Overview

LLMs naturally produce unstructured text, but many applications need structured data. This pattern uses JSON mode to ensure valid JSON output, then parses and validates it into typed structures.

## Key Features

- JSON mode ensures valid JSON response
- Type-safe parsing into structs
- Validation of enumerated fields
- Helper functions for testing

## Real-World Application

Data extraction, form auto-fill, API response generation.

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
# From `/examples/patterns/structured-output`:
cd rust && cargo build
cd go && make build
```
## Log Classification Example

The example classifies log entries into structured categories:

```json
{
  "severity": "error",
  "category": "auth",
  "summary": "Failed login attempt for admin user",
  "actionable": true
}
```

### Valid Values
- **severity**: `info`, `warning`, `error`, `critical`
- **category**: `auth`, `network`, `system`, `application`

## Library usage

### Rust

```rust
use structured_output::{classify_log, LogClassification};

let classification = classify_log(&provider, "llama3", log_entry).await?;

println!("Severity: {}", classification.severity);
println!("Actionable: {}", classification.actionable);
```

### Go

```go
classification, err := ClassifyLog(ctx, provider, "llama3", logEntry)

fmt.Println("Severity:", classification.Severity)
fmt.Println("Actionable:", classification.Actionable)
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

## Prompt Engineering Tips

1. **Be explicit**: Specify the exact JSON format in the system prompt
2. **List valid values**: Enumerate allowed values for each field
3. **Provide examples**: Include example JSON in complex cases
4. **Validate output**: Always validate parsed JSON against your schema

## Production Considerations

1. **Retry on validation failure**: LLMs may occasionally produce invalid values
2. **Fallback handling**: Have a default category for edge cases
3. **Schema evolution**: Version your schemas for backward compatibility
4. **Rate limiting**: JSON mode may use more tokens due to formatting
