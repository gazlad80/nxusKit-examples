# CLI Assistant Integration

Converts natural language queries into shell commands using LLM with streaming output.

> Turn plain English into shell commands instantly — build a streaming CLI assistant that speaks your language and runs in your terminal.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## Overview

This example demonstrates a practical CLI tool that translates human-readable requests into executable shell commands. It showcases streaming for real-time output as the command is generated.

## What this demonstrates

**Difficulty: Starter** 🟢 · LLM

- **Summary:** Interactive CLI assistant with LLM backend
- **Scenario:** Build an interactive terminal assistant powered by an LLM
- **`tech_tags` in manifest:** `LLM` — example id **`cli-assistant`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).
- **Models:** Set cloud provider API keys and/or run **Ollama** locally when you execute the **Run** steps (interactive flags like `--help` / `--verbose` are documented below).

## Real-World Application

Developer productivity tool, command-line copilot.

## Technologies

LLM

## Language Implementations

| Language | Path | Status |
|----------|------|--------|
| Rust | `rust/` | Available |
| Go | `go/` | Available |

## Features

- Natural language to shell command translation
- Streaming output shows command as it's generated
- Safety comments for dangerous operations
- Works with common Unix commands

## Example Queries

| Query | Generated Command |
|-------|------------------|
| "find all rust files modified in the last week" | `find . -name "*.rs" -mtime -7` |
| "show disk usage sorted by size" | `du -sh * \| sort -h` |
| "list all running docker containers" | `docker ps` |

## Build

```bash
cd rust && cargo build
cd ../go && make build
```

## Library usage

### Rust

```rust
use nxuskit::prelude::*;

let provider = OllamaProvider::builder().model("llama3").build()?;
let command = generate_command(&provider, query).await?;
println!("Generated: {}", command);
```

### Go

```go
command, err := GenerateCommand(ctx, provider, query)
fmt.Println("Generated:", command)
```

## Run

### Rust
```bash
cd rust
cargo run

# Interactive mode
cargo run -- "your query here"
```

### Go
```bash
cd go
go run .

# Interactive mode
go run . "your query here"
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

## Safety Considerations

The LLM is instructed to:
- Add warning comments for dangerous operations (rm -rf, etc.)
- Use common, well-known Unix commands
- Provide clarifying comments for ambiguous requests

**Important**: Always review generated commands before executing them.

## Extending the Example

Ideas for enhancement:
1. Add command execution with confirmation prompt
2. Implement command history
3. Add support for platform-specific commands (Windows, macOS, Linux)
4. Include command explanation mode
5. Add syntax highlighting for output
