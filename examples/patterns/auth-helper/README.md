# Auth Helper

Demonstrates OAuth login flow and credential management using the nxuskit SDK.

> Wire up OAuth flows and API key credentials across every provider your app supports, with a single unified auth interface.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## What this demonstrates

**Difficulty: Starter** 🟢 · Auth · OAuth

- **Summary:** OAuth login flow and credential management helper *(from `conformance/examples_manifest.json`: **auth-helper**)*.
- **Scenario:** List providers, check auth status, set credentials, initiate OAuth flows.
- **`tech_tags`:** `Auth`, `OAuth`.
- Listing all supported providers and their authentication methods
- Checking authentication status for individual providers
- Setting and removing API key credentials
- Initiating an OAuth login flow (for OAuth-capable providers)
- Viewing a complete authentication status dashboard

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** Go, Rust (CLI and egui GUI).

## Tier

Community (free)

## Languages

| Language | Path | Description |
|----------|------|-------------|
| Go | [go/](go/) | CLI-based auth management |
| Rust (CLI) | [cli/](cli/) | Terminal-based OAuth + credential management |
| Rust (core) | [core/](core/) | Shared auth library used by CLI and egui |
| Rust (egui) | [egui/](egui/) | GUI-based OAuth flow with egui |

## Build

Attach an **installed SDK** (`NXUSKIT_SDK_DIR`). See the repository [README.md](../../../README.md) and `scripts/test-examples.sh`.

```bash
# Go
cd go && make build

# Rust CLI
cd cli && cargo build

# Rust GUI (requires egui dependencies)
cd egui && cargo build
```

## Run

```bash
# Go
cd go && ./bin/auth-helper

# Rust CLI
cd cli && cargo run

# Rust GUI
cd egui && cargo run
```

## Notes

- No current providers support OAuth yet (infrastructure is ready for future providers)
- API key authentication works with all providers that require credentials
- The OAuth flow launches a browser and starts a localhost callback server
