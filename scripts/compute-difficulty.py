#!/usr/bin/env python3
"""Compute difficulty classification for all examples in the manifest.

Reads conformance/examples_manifest.json and conformance/difficulty-thresholds.json,
computes a weighted composite score from 5 dimensions, and writes the difficulty
field back into the manifest.

Dimensions (per PoR examples-difficulty-classification-20260328.md):
  1. Concept count (tech_tags count)
  2. Code volume (SLOC via tokei, averaged across implementations)
  3. Cognitive complexity (radon for Python, gocognit for Go, heuristic for Rust)
  4. Prerequisite depth (external system count from tech_tags)
  5. File count (source files per implementation, averaged)

Usage:
  python3 scripts/compute-difficulty.py [--dry-run] [--verbose]
"""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
MANIFEST_PATH = REPO_ROOT / "conformance" / "examples_manifest.json"
THRESHOLDS_PATH = REPO_ROOT / "conformance" / "difficulty-thresholds.json"

# Tags that indicate external system dependencies beyond the base SDK
EXTERNAL_TAGS = {"CLIPS", "Solver", "BN", "ZEN", "MCP", "Vision"}
SDK_ONLY_TAGS = {"LLM", "Streaming", "Auth", "OAuth"}


def load_json(path: Path) -> dict:
    with open(path) as f:
        return json.load(f)


def save_json(path: Path, data: dict) -> None:
    with open(path, "w") as f:
        json.dump(data, f, indent=2, ensure_ascii=False)
        f.write("\n")


def sloc_via_tokei(impl_path: str) -> int:
    """Count SLOC for an implementation directory using tokei."""
    full_path = REPO_ROOT / impl_path
    if not full_path.is_dir():
        return 0
    try:
        result = subprocess.run(
            ["tokei", str(full_path), "--output", "json"],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode != 0:
            return 0
        data = json.loads(result.stdout)
        total = data.get("Total", {})
        return total.get("code", 0)
    except (subprocess.TimeoutExpired, json.JSONDecodeError, FileNotFoundError):
        return 0


def cc_python(impl_path: str) -> float:
    """Compute average cognitive complexity for Python files using radon."""
    full_path = REPO_ROOT / impl_path
    if not full_path.is_dir():
        return 0.0
    try:
        from radon.complexity import cc_visit

        complexities = []
        for py_file in full_path.rglob("*.py"):
            code = py_file.read_text(errors="replace")
            results = cc_visit(code)
            complexities.extend(r.complexity for r in results)
        if not complexities:
            return 0.0
        return sum(complexities) / len(complexities)
    except ImportError:
        return 0.0


def cc_go(impl_path: str) -> float:
    """Compute average cognitive complexity for Go files using gocognit."""
    full_path = REPO_ROOT / impl_path
    if not full_path.is_dir():
        return 0.0
    try:
        result = subprocess.run(
            ["gocognit", str(full_path)],
            capture_output=True,
            text=True,
            timeout=30,
        )
        if result.returncode != 0:
            return 0.0
        # gocognit output: "N package FuncName file:line:col"
        values = []
        for line in result.stdout.strip().splitlines():
            parts = line.split()
            if parts and parts[0].isdigit():
                values.append(int(parts[0]))
        if not values:
            return 0.0
        return sum(values) / len(values)
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return 0.0


def cc_rust_heuristic(impl_path: str) -> float:
    """Estimate cognitive complexity for Rust files using a simple heuristic.

    Counts control flow keywords and nesting depth as a proxy for CC.
    This is a fallback since rust-code-analysis-cli failed to compile.
    """
    full_path = REPO_ROOT / impl_path
    if not full_path.is_dir():
        return 0.0
    cc_keywords = re.compile(
        r"\b(if|else|match|for|while|loop|&&|\|\||\.map\(|\.filter\()\b"
    )
    total_cc = 0
    fn_count = 0
    current_fn_cc = 0
    in_fn = False
    brace_depth = 0

    for rs_file in full_path.rglob("*.rs"):
        for line in rs_file.read_text(errors="replace").splitlines():
            stripped = line.strip()
            if stripped.startswith("fn ") or stripped.startswith("pub fn "):
                if in_fn and fn_count > 0:
                    total_cc += current_fn_cc
                in_fn = True
                fn_count += 1
                current_fn_cc = 0
                brace_depth = 0
            if in_fn:
                brace_depth += line.count("{") - line.count("}")
                hits = len(cc_keywords.findall(line))
                # Weight by nesting depth
                current_fn_cc += hits * max(1, brace_depth)
    if in_fn:
        total_cc += current_fn_cc
    if fn_count == 0:
        return 0.0
    return total_cc / fn_count


def count_source_files(impl_path: str, lang: str) -> int:
    """Count source files for a given language implementation."""
    full_path = REPO_ROOT / impl_path
    if not full_path.is_dir():
        return 0
    ext_map = {"rust": ".rs", "go": ".go", "python": ".py"}
    ext = ext_map.get(lang, "")
    if not ext:
        return 0
    return sum(1 for _ in full_path.rglob(f"*{ext}"))


def score_dimension(value: float, thresholds: list[float]) -> int:
    """Map a value to 0, 1, or 2 based on thresholds."""
    if value <= thresholds[0]:
        return 0
    elif value <= thresholds[1]:
        return 1
    return 2


def compute_difficulty(example: dict, weights: dict, thresholds: dict) -> dict:
    """Compute difficulty score for a single example."""
    implementations = example.get("implementations", {})
    tech_tags = example.get("tech_tags", [])

    # 1. Concept count: number of tech_tags
    concept_count = len(tech_tags)
    concept_score = score_dimension(concept_count, [1.5, 2.5])

    # 2. Code volume: average SLOC across implementations
    sloc_values = []
    for impl_path in implementations.values():
        sloc = sloc_via_tokei(impl_path)
        if sloc > 0:
            sloc_values.append(sloc)
    avg_sloc = sum(sloc_values) / max(len(sloc_values), 1)
    volume_score = score_dimension(avg_sloc, [100, 500])

    # 3. Cognitive complexity: average across implementations
    cc_values = []
    for lang, impl_path in implementations.items():
        if lang == "python":
            cc = cc_python(impl_path)
        elif lang == "go":
            cc = cc_go(impl_path)
        elif lang == "rust":
            cc = cc_rust_heuristic(impl_path)
        else:
            continue
        if cc > 0:
            cc_values.append(cc)
    avg_cc = sum(cc_values) / max(len(cc_values), 1)
    cc_score = score_dimension(avg_cc, [5, 10])

    # 4. Prerequisite depth: external system count
    external_count = len(set(tech_tags) & EXTERNAL_TAGS)
    prereq_score = score_dimension(external_count, [0.5, 1.5])

    # 5. File count: average source files across implementations
    file_counts = []
    for lang, impl_path in implementations.items():
        fc = count_source_files(impl_path, lang)
        if fc > 0:
            file_counts.append(fc)
    avg_files = sum(file_counts) / max(len(file_counts), 1)
    file_score = score_dimension(avg_files, [2.5, 5.5])

    # Weighted composite
    composite = (
        weights["concept_count"] * concept_score
        + weights["code_volume"] * volume_score
        + weights["cognitive_complexity"] * cc_score
        + weights["prerequisite_depth"] * prereq_score
        + weights["file_count"] * file_score
    )

    # Map to difficulty level
    if composite <= thresholds["starter_max"]:
        level = "starter"
    elif composite <= thresholds["intermediate_max"]:
        level = "intermediate"
    else:
        level = "advanced"

    return {
        "level": level,
        "composite": round(composite, 3),
        "scores": {
            "concept_count": concept_score,
            "code_volume": volume_score,
            "cognitive_complexity": cc_score,
            "prerequisite_depth": prereq_score,
            "file_count": file_score,
        },
        "raw": {
            "tech_tags": concept_count,
            "avg_sloc": round(avg_sloc),
            "avg_cc": round(avg_cc, 1),
            "external_deps": external_count,
            "avg_files": round(avg_files, 1),
        },
    }


def main() -> int:
    parser = argparse.ArgumentParser(description="Compute example difficulty")
    parser.add_argument(
        "--dry-run", action="store_true", help="Print only, don't write"
    )
    parser.add_argument("--verbose", "-v", action="store_true")
    args = parser.parse_args()

    manifest = load_json(MANIFEST_PATH)
    config = load_json(THRESHOLDS_PATH)
    weights = config["weights"]
    thresh = config["thresholds"]

    print(f"Computing difficulty for {len(manifest['examples'])} examples...\n")
    print(
        f"{'Example':<30} {'Level':<14} {'Score':>6}  {'Tags':>4} {'SLOC':>5} {'CC':>5} {'Ext':>3} {'Files':>5}"
    )
    print("-" * 90)

    for example in manifest["examples"]:
        name = example["name"]
        result = compute_difficulty(example, weights, thresh)

        # Skip if override exists
        if example.get("difficulty_override"):
            print(f"{name:<30} {'[override]':<14} {'':>6}  (keeping manual override)")
            continue

        level = result["level"]
        raw = result["raw"]
        print(
            f"{name:<30} {level:<14} {result['composite']:>6.3f}  "
            f"{raw['tech_tags']:>4} {raw['avg_sloc']:>5} {raw['avg_cc']:>5.1f} "
            f"{raw['external_deps']:>3} {raw['avg_files']:>5.1f}"
        )

        if args.verbose:
            print(f"  scores: {result['scores']}")

        if not args.dry_run:
            example["difficulty"] = level

    if not args.dry_run:
        save_json(MANIFEST_PATH, manifest)
        print(f"\n✓ Wrote difficulty to {MANIFEST_PATH}")
    else:
        print("\n(dry-run: no changes written)")

    return 0


if __name__ == "__main__":
    sys.exit(main())
