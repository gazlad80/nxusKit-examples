#!/usr/bin/env bash
# Check content staleness by recomputing content_hash for each example
# and comparing against the stored hash in the manifest.
#
# Exit 0: ≤3 stale examples (with warnings)
# Exit 1: >3 stale examples (hard failure)
#
# Requires: tokei, python3
# Does NOT require: ANTHROPIC_API_KEY (no API calls)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MANIFEST="$REPO_ROOT/conformance/examples_manifest.json"

if ! command -v tokei &>/dev/null; then
  echo "Error: tokei is not installed. Install with: brew install tokei"
  exit 1
fi

if ! command -v python3 &>/dev/null; then
  echo "Error: python3 is not installed."
  exit 1
fi

# Use the same hash logic as apply-content.py (per-language SLOC via tokei)
STALE_COUNT=$(python3 - "$MANIFEST" "$REPO_ROOT" <<'PYEOF'
import hashlib
import json
import subprocess
import sys
from pathlib import Path

manifest_path = Path(sys.argv[1])
repo_root = Path(sys.argv[2])

manifest = json.load(open(manifest_path))
stale = []


def sloc_via_tokei(impl_path):
    """Per-language SLOC — must match apply-content.py exactly."""
    full_path = repo_root / impl_path
    if not full_path.is_dir():
        return {}
    try:
        result = subprocess.run(
            ["tokei", str(full_path), "--output", "json"],
            capture_output=True, text=True, timeout=30
        )
        if result.returncode != 0:
            return {}
        data = json.loads(result.stdout)
        counts = {}
        for lang_key, lang_data in data.items():
            if lang_key == "Total":
                continue
            code = 0
            if isinstance(lang_data, dict):
                code = lang_data.get("code", 0)
            if code > 0:
                counts[lang_key] = code
        return counts
    except Exception:
        return {}


def compute_sloc_counts(example):
    merged = {}
    for impl_path in example.get("implementations", {}).values():
        lang_counts = sloc_via_tokei(impl_path)
        for lang, count in lang_counts.items():
            merged[lang] = merged.get(lang, 0) + count
    return merged


def compute_content_hash(example, sloc_counts):
    description = example.get("description", "")
    scenario = example.get("scenario", "")
    real_world = example.get("real_world_application", "")
    tech_tags = ",".join(sorted(example.get("tech_tags", [])))
    difficulty = example.get("difficulty", "")
    sloc_str = ",".join(
        f"{lang}:{sloc}" for lang, sloc in sorted(sloc_counts.items())
    )
    payload = (
        f"{description}\n{scenario}\n{real_world}\n"
        f"{tech_tags}\n{difficulty}\n{sloc_str}"
    )
    return hashlib.sha256(payload.encode()).hexdigest()


for ex in manifest["examples"]:
    stored_hash = ex.get("content_hash")
    if not stored_hash:
        continue

    sloc_counts = compute_sloc_counts(ex)
    computed = compute_content_hash(ex, sloc_counts)

    if computed != stored_hash:
        stale.append(ex["name"])
        print(
            f"::warning::Content may be stale for '{ex['name']}' "
            f"(hash mismatch)",
            file=sys.stderr,
        )

print(len(stale))
PYEOF
)

echo "Content staleness check: $STALE_COUNT stale example(s)"

if [ "$STALE_COUNT" -gt 3 ]; then
  echo "::error::More than 3 examples have stale content ($STALE_COUNT). Regenerate with: python3 scripts/generate-content.py"
  exit 1
fi

if [ "$STALE_COUNT" -gt 0 ]; then
  echo "Warning: $STALE_COUNT example(s) have stale content. Consider regenerating."
fi

echo "Content staleness check passed."
exit 0
