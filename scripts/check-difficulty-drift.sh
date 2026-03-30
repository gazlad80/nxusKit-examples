#!/usr/bin/env bash
# Check difficulty drift: recompute difficulty for each example and compare
# against the manifest value. Fail if any example's computed difficulty
# diverges by more than 1 level from the manifest without an override.
#
# Levels: starter=0, intermediate=1, advanced=2
# Drift >1 level = error (e.g., starter→advanced or vice versa)
#
# Requires: tokei, python3, radon, gocognit (same as compute-difficulty.py)

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if ! command -v tokei &>/dev/null; then
  echo "::notice::tokei not installed — skipping difficulty drift check."
  exit 0
fi

DRIFT_COUNT=$(python3 - "$REPO_ROOT" <<'PYEOF'
import json
import subprocess
import re
import sys
from pathlib import Path

repo_root = Path(sys.argv[1])
manifest = json.load(open(repo_root / "conformance" / "examples_manifest.json"))
config = json.load(open(repo_root / "conformance" / "difficulty-thresholds.json"))
weights = config["weights"]
thresh = config["thresholds"]

EXTERNAL_TAGS = {"CLIPS", "Solver", "BN", "ZEN", "MCP", "Vision"}
LEVEL_MAP = {"starter": 0, "intermediate": 1, "advanced": 2}
LEVEL_NAMES = {0: "starter", 1: "intermediate", 2: "advanced"}


def score_dim(value, lo, hi):
    if value <= lo:
        return 0
    elif value <= hi:
        return 1
    return 2


def sloc_tokei(path):
    if not path.is_dir():
        return 0
    try:
        r = subprocess.run(
            ["tokei", str(path), "--output", "json"],
            capture_output=True, text=True, timeout=30,
        )
        return json.loads(r.stdout).get("Total", {}).get("code", 0)
    except Exception:
        return 0


def cc_python(path):
    if not path.is_dir():
        return 0.0
    try:
        from radon.complexity import cc_visit
        ccs = []
        for f in path.rglob("*.py"):
            ccs.extend(r.complexity for r in cc_visit(f.read_text(errors="replace")))
        return sum(ccs) / max(len(ccs), 1)
    except Exception:
        return 0.0


def cc_go(path):
    if not path.is_dir():
        return 0.0
    try:
        r = subprocess.run(
            ["gocognit", str(path)], capture_output=True, text=True, timeout=30,
        )
        vals = [int(l.split()[0]) for l in r.stdout.strip().splitlines() if l and l.split()[0].isdigit()]
        return sum(vals) / max(len(vals), 1)
    except Exception:
        return 0.0


def cc_rust(path):
    if not path.is_dir():
        return 0.0
    kw = re.compile(r"\b(if|else|match|for|while|loop|&&|\|\|)\b")
    total, fns, cur, in_fn, depth = 0, 0, 0, False, 0
    for f in path.rglob("*.rs"):
        for line in f.read_text(errors="replace").splitlines():
            s = line.strip()
            if s.startswith("fn ") or s.startswith("pub fn "):
                if in_fn:
                    total += cur
                in_fn, fns, cur, depth = True, fns + 1, 0, 0
            if in_fn:
                depth += line.count("{") - line.count("}")
                cur += len(kw.findall(line)) * max(1, depth)
    if in_fn:
        total += cur
    return total / max(fns, 1)


def count_files(path, ext):
    if not path.is_dir():
        return 0
    return sum(1 for _ in path.rglob(f"*{ext}"))


drift = []
for ex in manifest["examples"]:
    manifest_diff = ex.get("difficulty", "")
    if not manifest_diff:
        continue
    if ex.get("difficulty_override"):
        continue  # Skip overridden examples

    impls = ex.get("implementations", {})
    tags = ex.get("tech_tags", [])

    # Compute all dimensions
    slocs = [sloc_tokei(repo_root / p) for p in impls.values()]
    avg_sloc = sum(slocs) / max(len(slocs), 1)

    ccs = []
    for lang, p in impls.items():
        full = repo_root / p
        if lang == "python":
            c = cc_python(full)
        elif lang == "go":
            c = cc_go(full)
        elif lang == "rust":
            c = cc_rust(full)
        else:
            continue
        if c > 0:
            ccs.append(c)
    avg_cc = sum(ccs) / max(len(ccs), 1)

    ext_map = {"rust": ".rs", "go": ".go", "python": ".py"}
    fcs = [count_files(repo_root / p, ext_map.get(l, "")) for l, p in impls.items()]
    avg_files = sum(fcs) / max(len(fcs), 1)

    ext_count = len(set(tags) & EXTERNAL_TAGS)

    composite = (
        weights["concept_count"] * score_dim(len(tags), 1.5, 2.5)
        + weights["code_volume"] * score_dim(avg_sloc, 100, 500)
        + weights["cognitive_complexity"] * score_dim(avg_cc, 5, 10)
        + weights["prerequisite_depth"] * score_dim(ext_count, 0.5, 1.5)
        + weights["file_count"] * score_dim(avg_files, 2.5, 5.5)
    )

    if composite <= thresh["starter_max"]:
        computed = "starter"
    elif composite <= thresh["intermediate_max"]:
        computed = "intermediate"
    else:
        computed = "advanced"

    manifest_level = LEVEL_MAP.get(manifest_diff, 1)
    computed_level = LEVEL_MAP.get(computed, 1)

    if abs(manifest_level - computed_level) > 1:
        drift.append(ex["name"])
        print(
            f"::error::Difficulty drift for '{ex['name']}': "
            f"manifest={manifest_diff}, computed={computed} "
            f"(>1 level difference, add difficulty_override to suppress)",
            file=sys.stderr,
        )

print(len(drift))
PYEOF
)

echo "Difficulty drift check: $DRIFT_COUNT example(s) with >1 level drift"

if [ "$DRIFT_COUNT" -gt 0 ]; then
  echo "::error::$DRIFT_COUNT example(s) have difficulty drift >1 level. Run compute-difficulty.py to review."
  exit 1
fi

echo "Difficulty drift check passed."
exit 0
