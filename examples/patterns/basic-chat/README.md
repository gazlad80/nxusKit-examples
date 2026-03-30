# Basic Chat

Send a single chat message and receive a response.

> Send your first LLM chat message in minutes, then swap providers without touching your application code.

## Edition

**Community** — runs on the OSS / Community SDK edition.

## What this demonstrates

**Difficulty: Starter** 🟢 · LLM

- **Summary:** Basic chat completion with a simple prompt
- **Scenario:** Send a single chat message and receive a response
- **`tech_tags` in manifest:** `LLM` — example id **`basic-chat`** in `conformance/examples_manifest.json`.

## Prerequisites

- **SDK:** Use an installed SDK tree (`NXUSKIT_SDK_DIR`, `NXUSKIT_LIB_PATH` as needed); `test-examples.sh` resolves Go/Rust/Python deps from that tree only — see [README.md](../../../README.md), `scripts/setup-sdk.sh`, and `scripts/test-examples.sh`.
- **Languages in this example:** go, python, rust (paths under this directory; Python may live under a sibling `python/` or shared reference per **Language Implementations**).
- **Models:** Set cloud provider API keys and/or run **Ollama** locally when you execute the **Run** steps (interactive flags like `--help` / `--verbose` are documented below).

## Real-World Application

Customer support chatbot, FAQ assistant

## Technologies

LLM

## Language Implementations

| Language | Path | Status |
|----------|------|--------|
| Rust | `rust/` | Available |
| Go | `go/` | Available |
| Python | `python/` | Available |

## Build

Use an installed SDK (`NXUSKIT_SDK_DIR`); from the repo root, `scripts/test-examples.sh` patches `nxuskit` path deps to that tree and builds this crate.

```bash
cd rust && cargo build
cd ../go && make build
cd ../python && python3 main.py --help
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
make build && bin/basic-chat
```

### Python
```bash
cd python
python main.py
```

See also: [`EXAMPLE_README_TEMPLATE.md`](../../EXAMPLE_README_TEMPLATE.md) for the standard README sections.
