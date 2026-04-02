#!/usr/bin/env python3
"""Audit direct non-SDK dependencies across all examples.

Scans Cargo.toml, go.mod, and requirements.txt files under examples/,
identifies direct dependencies that are not part of the nxusKit SDK,
and writes the results to conformance/dependency-audit.json.

Usage:
    python3 scripts/audit-example-deps.py           # Generate inventory
    python3 scripts/audit-example-deps.py --check   # Check freshness (exit 2 if stale)
"""

import json
import os
import re
import sys
import tempfile
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
EXAMPLES_DIR = REPO_ROOT / "examples"
MANIFEST_PATH = REPO_ROOT / "conformance" / "examples_manifest.json"
OUTPUT_PATH = REPO_ROOT / "conformance" / "dependency-audit.json"

# SDK crate/module/package names to exclude
SDK_CRATES = {
    "nxuskit", "nxuskit-engine", "nxuskit-examples-interactive",
    "nxuskit-examples-support", "llm-patterns", "clips-wire",
    "nxuskit-examples-clips-wire", "auth-helper-core",
}
SDK_GO_MODULES = {
    "github.com/nxus-SYSTEMS/nxusKit/packages/nxuskit-go",
    "github.com/nxus-SYSTEMS/nxusKit/examples/shared/go/interactive",
}
# Also match the internal repo variant of the shared module path
# (uses a prefix match to avoid embedding the internal repo name)
_SDK_GO_PREFIX = "github.com/nxus-SYSTEMS/"
SDK_PYTHON_PACKAGES = {"nxuskit-py", "nxuskit", "nxuskit_py"}

# Universal Rust boilerplate — still reported but classified separately
RUST_BOILERPLATE = {
    "serde", "serde_json", "tokio", "clap", "futures",
}


def parse_cargo_toml_deps(path: Path) -> list[str]:
    """Extract [dependencies] crate names from a Cargo.toml."""
    deps = []
    in_deps = False
    in_workspace_deps = False
    content = path.read_text()

    for line in content.splitlines():
        stripped = line.strip()

        # Track section headers
        if stripped.startswith("["):
            in_deps = stripped in ("[dependencies]", "[target.'cfg(unix)'.dependencies]",
                                   "[target.\"cfg(unix)\".dependencies]")
            in_workspace_deps = stripped == "[workspace.dependencies]"
            continue

        if not (in_deps or in_workspace_deps):
            continue

        # Skip empty lines and comments
        if not stripped or stripped.startswith("#"):
            continue

        # Parse dep name (before = sign or as table key)
        match = re.match(r'^([a-zA-Z0-9_-]+)\s*[={]', stripped)
        if match:
            crate_name = match.group(1)
            # Normalize underscore/hyphen
            normalized = crate_name.replace("_", "-")
            if normalized not in SDK_CRATES:
                deps.append(crate_name)

    return deps


def parse_go_mod_deps(path: Path) -> list[str]:
    """Extract direct (non-indirect) require entries from go.mod."""
    deps = []
    in_require = False
    content = path.read_text()

    for line in content.splitlines():
        stripped = line.strip()

        if stripped.startswith("require ("):
            in_require = True
            continue
        if stripped == ")":
            in_require = False
            continue

        if in_require:
            # Skip indirect deps
            if "// indirect" in stripped:
                continue
            # Parse module path
            parts = stripped.split()
            if parts:
                mod_path = parts[0]
                if mod_path not in SDK_GO_MODULES and not mod_path.startswith("//") and not mod_path.startswith(_SDK_GO_PREFIX):
                    deps.append(mod_path)

        # Single-line require
        if stripped.startswith("require ") and "(" not in stripped:
            parts = stripped.split()
            if len(parts) >= 3:
                mod_path = parts[1]
                if mod_path not in SDK_GO_MODULES and not mod_path.startswith(_SDK_GO_PREFIX):
                    deps.append(mod_path)

    return deps


def parse_requirements_txt_deps(path: Path) -> list[str]:
    """Extract package names from requirements.txt."""
    deps = []
    content = path.read_text()

    for line in content.splitlines():
        stripped = line.strip()
        if not stripped or stripped.startswith("#") or stripped.startswith("-"):
            continue
        # Package name is before any version specifier
        match = re.match(r'^([a-zA-Z0-9_-]+)', stripped)
        if match:
            pkg = match.group(1).lower().replace("-", "_")
            normalized = pkg.replace("_", "-")
            if normalized not in SDK_PYTHON_PACKAGES and pkg not in SDK_PYTHON_PACKAGES:
                deps.append(match.group(1))

    return deps


def discover_example_name(impl_path: Path) -> tuple[str, str]:
    """Given a path under examples/, return (example_name, category)."""
    # Path pattern: examples/<category>/<name>/...
    rel = impl_path.relative_to(EXAMPLES_DIR)
    parts = rel.parts
    if len(parts) >= 2:
        return parts[1], parts[0]
    return parts[0], "unknown"


def build_inventory() -> list[dict]:
    """Build the complete dependency inventory."""
    # Load manifest for example metadata
    with open(MANIFEST_PATH) as f:
        manifest = json.load(f)

    # Build a lookup from manifest
    manifest_examples = {ex["name"]: ex for ex in manifest.get("examples", [])}

    # Collect deps per example
    example_deps: dict[str, dict] = {}

    # Scan Cargo.toml files
    for cargo_path in sorted(EXAMPLES_DIR.rglob("Cargo.toml")):
        # Skip target/ and shared/ workspace Cargo.toml files
        if "target" in cargo_path.parts:
            continue
        rel = cargo_path.relative_to(EXAMPLES_DIR)
        # Identify example name
        parts = rel.parts
        if len(parts) < 3:
            continue  # Not deep enough to be an example impl
        category = parts[0]
        name = parts[1]

        # Accept Cargo.toml in rust/ subdir or in sub-crate dirs (e.g., auth-helper/core/)
        # Skip if it's a workspace-level Cargo.toml (no [dependencies] section typically)
        if parts[2] not in ("rust", "core", "cli", "egui"):
            continue

        if name not in example_deps:
            example_deps[name] = {
                "name": name,
                "category": category,
                "languages": set(),
                "direct_deps": {"rust": [], "go": [], "python": []},
            }
        example_deps[name]["languages"].add("rust")
        rust_deps = parse_cargo_toml_deps(cargo_path)
        # Merge, avoiding duplicates
        existing = set(example_deps[name]["direct_deps"]["rust"])
        for d in rust_deps:
            if d not in existing:
                example_deps[name]["direct_deps"]["rust"].append(d)
                existing.add(d)

    # Scan go.mod files
    for gomod_path in sorted(EXAMPLES_DIR.rglob("go.mod")):
        if "shared" in gomod_path.parts:
            continue
        rel = gomod_path.relative_to(EXAMPLES_DIR)
        parts = rel.parts
        if len(parts) < 3:
            continue
        category = parts[0]
        name = parts[1]

        if name not in example_deps:
            example_deps[name] = {
                "name": name,
                "category": category,
                "languages": set(),
                "direct_deps": {"rust": [], "go": [], "python": []},
            }
        example_deps[name]["languages"].add("go")
        go_deps = parse_go_mod_deps(gomod_path)
        existing = set(example_deps[name]["direct_deps"]["go"])
        for d in go_deps:
            if d not in existing:
                example_deps[name]["direct_deps"]["go"].append(d)
                existing.add(d)

    # Scan requirements.txt files
    for req_path in sorted(EXAMPLES_DIR.rglob("requirements.txt")):
        rel = req_path.relative_to(EXAMPLES_DIR)
        parts = rel.parts
        if len(parts) < 3:
            continue
        category = parts[0]
        name = parts[1]

        if name not in example_deps:
            example_deps[name] = {
                "name": name,
                "category": category,
                "languages": set(),
                "direct_deps": {"rust": [], "go": [], "python": []},
            }
        example_deps[name]["languages"].add("python")
        py_deps = parse_requirements_txt_deps(req_path)
        existing = set(example_deps[name]["direct_deps"]["python"])
        for d in py_deps:
            if d not in existing:
                example_deps[name]["direct_deps"]["python"].append(d)
                existing.add(d)

    # Classify and format output
    inventory = []
    for name in sorted(example_deps.keys()):
        entry = example_deps[name]
        # Filter out boilerplate from classification (but keep in listing)
        non_boilerplate_rust = [
            d for d in entry["direct_deps"]["rust"]
            if d not in RUST_BOILERPLATE
        ]
        has_direct = (
            bool(non_boilerplate_rust)
            or bool(entry["direct_deps"]["go"])
            or bool(entry["direct_deps"]["python"])
        )

        # Use manifest data for languages if available
        manifest_entry = manifest_examples.get(name, {})
        languages = sorted(entry["languages"])
        if manifest_entry:
            languages = sorted(set(languages) | set(manifest_entry.get("languages", [])))

        inventory.append({
            "name": name,
            "category": entry.get("category", manifest_entry.get("category", "unknown")),
            "classification": "has-direct-deps" if has_direct else "sdk-only",
            "languages": languages,
            "direct_deps": {
                "rust": sorted(entry["direct_deps"]["rust"]),
                "go": sorted(entry["direct_deps"]["go"]),
                "python": sorted(entry["direct_deps"]["python"]),
            },
        })

    return inventory


def main():
    check_mode = "--check" in sys.argv

    inventory = build_inventory()

    if check_mode:
        # Compare against committed file
        if not OUTPUT_PATH.exists():
            print(f"ERROR: {OUTPUT_PATH} does not exist. Run without --check first.", file=sys.stderr)
            sys.exit(2)

        new_content = json.dumps({"examples": inventory}, indent=2) + "\n"
        old_content = OUTPUT_PATH.read_text()

        if new_content != old_content:
            print(f"ERROR: {OUTPUT_PATH} is stale. Regenerate with:", file=sys.stderr)
            print(f"  python3 scripts/audit-example-deps.py", file=sys.stderr)
            sys.exit(2)

        print(f"OK: {OUTPUT_PATH} is up to date.")
        sys.exit(0)

    # Write inventory
    output = {"examples": inventory}
    OUTPUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT_PATH.write_text(json.dumps(output, indent=2) + "\n")

    # Summary
    total = len(inventory)
    has_deps = sum(1 for e in inventory if e["classification"] == "has-direct-deps")
    sdk_only = total - has_deps
    print(f"Dependency audit complete: {total} examples ({has_deps} has-direct-deps, {sdk_only} sdk-only)")
    print(f"Written to: {OUTPUT_PATH}")

    if has_deps > 0:
        print("\nExamples with direct non-SDK dependencies:")
        for e in inventory:
            if e["classification"] == "has-direct-deps":
                deps = []
                for lang in ("rust", "go", "python"):
                    lang_deps = [d for d in e["direct_deps"][lang] if d not in RUST_BOILERPLATE]
                    if lang_deps:
                        deps.append(f"{lang}: {', '.join(lang_deps)}")
                print(f"  {e['name']}: {'; '.join(deps)}")


if __name__ == "__main__":
    main()
