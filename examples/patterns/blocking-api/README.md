# Blocking API

Make synchronous LLM calls without async runtime

> Call LLMs synchronously from any Rust code — no async runtime required, no await syntax, no tokio dependency to manage.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## What this demonstrates

**Difficulty: Starter** 🟢 · LLM

- **Summary:** Synchronous blocking API for simpler use cases
- **Scenario:** Make synchronous LLM calls without async runtime
- **`tech_tags` in manifest:** `LLM` — example id **`blocking-api`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).
- **Models:** Set cloud provider API keys and/or run **Ollama** locally when you execute the **Run** steps (interactive flags like `--help` / `--verbose` are documented below).

## Real-World Application

CLI tools, batch processing scripts

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
# From `/examples/patterns/blocking-api`:
cd rust && cargo build
cd go && make build
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
make build && bin/blocking-api
```
