# Alert Triage Integration

LLM-powered alert triage for observability systems, with batch processing and structured output.

> Turn alert noise into prioritized action by letting an LLM classify severity, surface root causes, and recommend remediation steps automatically.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## Overview

This example demonstrates using LLMs to triage monitoring alerts, providing priority assessment, likely causes, and suggested remediation actions. It's designed to work with Alertmanager-format alerts.

## What this demonstrates

**Difficulty: Starter** 🟢 · LLM

- **Summary:** Alert triage with LLM-powered analysis
- **Scenario:** Classify and prioritize alerts using LLM reasoning
- **`tech_tags` in manifest:** `LLM` — example id **`alert-triage`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).
- **Models:** Set cloud provider API keys and/or run **Ollama** locally when you execute the **Run** steps (interactive flags like `--help` / `--verbose` are documented below).

## Real-World Application

SOC alert triage, IT incident management.

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
# From `/examples/integrations/alert-triage`:
cd rust && cargo build
cd go && make build
```
## Features

- Batch alert processing for efficiency
- Structured JSON output with priority, cause, and actions
- Compatible with Alertmanager webhook format
- Prioritization based on severity and context

## Alert Format

Input alerts follow the Alertmanager format:

```json
{
  "alertname": "HighMemoryUsage",
  "severity": "warning",
  "instance": "web-server-01",
  "description": "Memory usage above 85% for 5 minutes"
}
```

## Triage Output

Each alert receives a triage result:

```json
{
  "alertname": "HighMemoryUsage",
  "priority": 3,
  "summary": "Memory pressure on web server",
  "likely_cause": "Memory leak or increased traffic",
  "suggested_actions": [
    "Check for memory leaks in application logs",
    "Review recent deployments",
    "Consider horizontal scaling"
  ]
}
```

## Priority Scale

| Priority | Meaning | Response |
|----------|---------|----------|
| 1 | Critical | Immediate action required |
| 2 | High | Respond within 1 hour |
| 3 | Medium | Respond within 4 hours |
| 4 | Low | Respond within 24 hours |
| 5 | Informational | No action required |

## Library usage

### Rust

```rust
use alert_triage::{triage_alerts, Alert};

let alerts = vec![
    Alert { alertname: "...", severity: "critical", ... },
];

let results = triage_alerts(&provider, "llama3", &alerts).await?;
for result in results {
    println!("{}: Priority {} - {}", result.alertname, result.priority, result.summary);
}
```

### Go

```go
alerts := []Alert{{AlertName: "...", Severity: "critical", ...}}

results, err := TriageAlerts(ctx, provider, "llama3", alerts)
for _, result := range results {
    fmt.Printf("%s: Priority %d - %s\n", result.AlertName, result.Priority, result.Summary)
}
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

## Sample Data

A `sample_alerts.json` file is provided with example alerts for testing.

## Testing

```bash
# Rust
cd rust && cargo test

# Go
cd go && go test -v
```

## Integration Ideas

1. **Alertmanager webhook**: Receive alerts via HTTP webhook
2. **PagerDuty integration**: Update incident priority based on triage
3. **Slack notifications**: Send enriched alerts to Slack channels
4. **Runbook linking**: Match alerts to relevant runbooks
5. **Historical learning**: Improve triage based on past incident resolutions
