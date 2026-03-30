# Multi Provider

Configure and switch between multiple LLM providers at runtime

> Run the same prompt across Claude, OpenAI, and Ollama simultaneously using one unified interface that normalizes responses and token usage.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## What this demonstrates

**Difficulty: Starter** 🟢 · LLM

- **Summary:** Using multiple providers in one application
- **Scenario:** Configure and switch between multiple LLM providers at runtime
- **`tech_tags` in manifest:** `LLM` — example id **`multi-provider`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, python, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).
- **Models:** Set cloud provider API keys and/or run **Ollama** locally when you execute the **Run** steps (interactive flags like `--help` / `--verbose` are documented below).

## Real-World Application

Multi-vendor AI gateway, provider comparison tool

## Technologies

LLM

## Language Implementations

| Language | Path | Status |
|----------|------|--------|
| Rust | `rust/` | Available |
| Go | `go/` | Available |
| Python | `python/` | Available |

## Build

Attach an **installed SDK** (`NXUSKIT_SDK_DIR`). See the repository [README.md](../../../README.md) and `scripts/test-examples.sh`.

```bash
# From `/examples/patterns/multi-provider`:
cd rust && cargo build
cd go && make build
cd python && python3 main.py --help
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
make build && bin/multi-provider
```

### Python
```bash
cd python
python main.py
```
