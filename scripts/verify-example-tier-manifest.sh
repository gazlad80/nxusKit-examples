#!/usr/bin/env bash
# Ensure conformance/examples_manifest.json tier values match
# conformance/example-tiers.json (vendored from SDK source; see
# scripts/sync-example-tiers-from-sdk.sh and sdk-packaging/docs/tier-comparison.md).
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MANIFEST="$ROOT/conformance/examples_manifest.json"
TIERS="$ROOT/conformance/example-tiers.json"
if [[ ! -f "$TIERS" ]]; then
  echo "error: missing $TIERS" >&2
  exit 1
fi
FAILED=0
while IFS= read -r name; do
  want="$(jq -r --arg n "$name" '.examples[$n] // empty' "$TIERS")"
  got="$(jq -r --arg n "$name" '.examples[] | select(.name == $n) | .tier' "$MANIFEST")"
  if [[ -z "$want" ]]; then
    echo "error: example '$name' missing from example-tiers.json examples" >&2
    FAILED=1
    continue
  fi
  if [[ "$want" != "$got" ]]; then
    echo "error: tier mismatch for '$name': examples_manifest=$got example-tiers=$want" >&2
    FAILED=1
  fi
done < <(jq -r '.examples[].name' "$MANIFEST")
if [[ "$FAILED" -ne 0 ]]; then
  exit 1
fi
echo "OK: all example tiers match example-tiers.json"
