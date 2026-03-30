#!/usr/bin/env bash
# Print dynamic-library linkage for a Rust example binary and assert libnxuskit is referenced.
# Usage: scripts/check-rust-example-linkage.sh path/to/binary
#        (after `cargo build` or `cargo build --release`; binary is often target/debug/... or target/release/...)

set -euo pipefail

[[ $# -eq 1 ]] || { echo "usage: $0 <path-to-binary>" >&2; exit 1; }
bin="$1"
[[ -f "$bin" ]] || { echo "error: not a file: $bin" >&2; exit 1; }

case "$(uname -s)" in
  Darwin)
    otool -L "$bin"
    otool -L "$bin" | grep -E 'libnxuskit|nxuskit' >/dev/null \
      || { echo "error: no libnxuskit reference in $bin (otool -L)" >&2; exit 1; }
    ;;
  MINGW*|MSYS*|CYGWIN*)
    echo "error: Windows linkage check not automated; use dumpbin /dependents" >&2
    exit 2
    ;;
  *)
    if command -v ldd >/dev/null 2>&1; then
      ldd "$bin"
      ldd "$bin" | grep -E 'libnxuskit|nxuskit' >/dev/null \
        || { echo "error: no libnxuskit reference in $bin (ldd)" >&2; exit 1; }
    else
      echo "error: ldd not available" >&2
      exit 2
    fi
    ;;
esac

echo "OK: $bin links libnxuskit"
