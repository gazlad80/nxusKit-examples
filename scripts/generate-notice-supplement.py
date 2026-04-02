#!/usr/bin/env python3
"""Generate NOTICE supplement for example-only dependencies.

Scans Cargo.toml and go.mod files under examples/, filters against
compliance/sdk-covered-deps.txt, extracts license info from local
crate/module caches, and outputs compliance/NOTICE-supplement.txt.

Python examples are SDK-only and need no supplement.

Usage:
    python3 scripts/generate-notice-supplement.py           # Generate supplement
    python3 scripts/generate-notice-supplement.py --check   # Check freshness (exit 2 if stale)
    python3 scripts/generate-notice-supplement.py --verbose  # Show discovery progress
"""

import json
import os
import re
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
EXAMPLES_DIR = REPO_ROOT / "examples"
SDK_COVERED_PATH = REPO_ROOT / "compliance" / "sdk-covered-deps.txt"
SUPPLEMENT_PATH = REPO_ROOT / "compliance" / "NOTICE-supplement.txt"
DEP_AUDIT_PATH = REPO_ROOT / "conformance" / "dependency-audit.json"

# Cargo registry cache location
CARGO_REGISTRY = Path.home() / ".cargo" / "registry" / "src"
# Go module cache location
# Go module cache: check GOMODCACHE env, then GOPATH/pkg/mod, then common locations
_gomodcache = os.environ.get("GOMODCACHE", "")
if _gomodcache:
    GO_MOD_CACHE = Path(_gomodcache)
else:
    _gopath = os.environ.get("GOPATH", "")
    candidates = [
        Path(_gopath) / "pkg" / "mod" if _gopath else None,
        Path.home() / ".go" / "pkg" / "mod",
        Path.home() / "go" / "pkg" / "mod",
    ]
    GO_MOD_CACHE = next((c for c in candidates if c and c.exists()), Path.home() / "go" / "pkg" / "mod")

# Copyleft licenses that must be flagged as errors
COPYLEFT_LICENSES = {
    "GPL-2.0", "GPL-3.0", "AGPL-3.0", "LGPL-2.0", "LGPL-2.1", "LGPL-3.0",
    "GPL-2.0-only", "GPL-3.0-only", "AGPL-3.0-only",
    "GPL-2.0-or-later", "GPL-3.0-or-later", "AGPL-3.0-or-later",
    "LGPL-2.0-only", "LGPL-2.1-only", "LGPL-3.0-only",
    "LGPL-2.0-or-later", "LGPL-2.1-or-later", "LGPL-3.0-or-later",
}

VERBOSE = "--verbose" in sys.argv


def info(msg: str) -> None:
    print(f"[supplement] {msg}")


def verbose(msg: str) -> None:
    if VERBOSE:
        print(f"[supplement]   {msg}")


def load_sdk_covered() -> set[str]:
    """Load dep names covered by SDK NOTICE."""
    if not SDK_COVERED_PATH.exists():
        info(f"WARNING: {SDK_COVERED_PATH} not found — treating all deps as uncovered")
        return set()
    names = set()
    for line in SDK_COVERED_PATH.read_text().splitlines():
        stripped = line.strip()
        if stripped:
            names.add(stripped)
            # Also add normalized versions
            names.add(stripped.replace("-", "_"))
            names.add(stripped.replace("_", "-"))
    return names


def load_example_deps() -> dict[str, list[str]]:
    """Load example-only deps from dependency audit."""
    if not DEP_AUDIT_PATH.exists():
        info(f"WARNING: {DEP_AUDIT_PATH} not found — run audit-example-deps.py first")
        return {"rust": [], "go": []}

    with open(DEP_AUDIT_PATH) as f:
        audit = json.load(f)

    rust_deps: set[str] = set()
    go_deps: set[str] = set()

    for ex in audit.get("examples", []):
        if ex.get("classification") == "has-direct-deps":
            for dep in ex.get("direct_deps", {}).get("rust", []):
                rust_deps.add(dep)
            for dep in ex.get("direct_deps", {}).get("go", []):
                go_deps.add(dep)

    return {"rust": sorted(rust_deps), "go": sorted(go_deps)}


def find_crate_license(crate_name: str) -> tuple[str, str]:
    """Find license info for a Rust crate from the cargo registry cache."""
    if not CARGO_REGISTRY.exists():
        return "Unknown", ""

    # Search for the crate directory in registry cache
    for registry_dir in CARGO_REGISTRY.iterdir():
        if not registry_dir.is_dir():
            continue
        # Find latest version of the crate
        crate_dirs = sorted(
            registry_dir.glob(f"{crate_name}-*"),
            key=lambda p: p.name,
            reverse=True,
        )
        for crate_dir in crate_dirs:
            if not crate_dir.is_dir():
                continue
            # Verify it's the right crate (not a prefix match)
            dir_name = crate_dir.name
            # Strip version suffix: crate_name-X.Y.Z
            if not re.match(rf"^{re.escape(crate_name)}-\d", dir_name):
                continue

            verbose(f"Found crate cache: {crate_dir}")

            # Read Cargo.toml for license field
            cargo_toml = crate_dir / "Cargo.toml"
            license_id = "Unknown"
            if cargo_toml.exists():
                content = cargo_toml.read_text()
                m = re.search(r'^license\s*=\s*"([^"]+)"', content, re.MULTILINE)
                if m:
                    license_id = m.group(1)

            # Read LICENSE file
            license_text = ""
            for license_file in ("LICENSE", "LICENSE-MIT", "LICENSE-APACHE",
                                  "LICENSE.md", "LICENSE.txt", "COPYING"):
                lf = crate_dir / license_file
                if lf.exists():
                    license_text = lf.read_text()
                    break

            return license_id, license_text

    return "Unknown", ""


def find_go_module_license(module_path: str) -> tuple[str, str]:
    """Find license info for a Go module from the module cache."""
    if not GO_MOD_CACHE.exists():
        return "Unknown", ""

    # Go module cache uses lowercase with '!' for uppercase
    escaped = ""
    for ch in module_path:
        if ch.isupper():
            escaped += "!" + ch.lower()
        else:
            escaped += ch

    # Find versioned directories
    mod_dir_base = GO_MOD_CACHE / escaped
    if mod_dir_base.exists() and mod_dir_base.is_dir():
        # Direct match (rare)
        pass
    else:
        # Look for versioned dirs: module@vX.Y.Z
        parent = mod_dir_base.parent
        if parent.exists():
            candidates = sorted(
                parent.glob(f"{mod_dir_base.name}@*"),
                key=lambda p: p.name,
                reverse=True,
            )
            if candidates:
                mod_dir_base = candidates[0]
            else:
                return "Unknown", ""
        else:
            return "Unknown", ""

    verbose(f"Found Go module cache: {mod_dir_base}")

    # Read LICENSE file
    license_text = ""
    license_id = "Unknown"
    for license_file in ("LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING", "NOTICE"):
        lf = mod_dir_base / license_file
        if lf.exists():
            license_text = lf.read_text()
            # Try to detect license type from content
            if "MIT License" in license_text or "Permission is hereby granted" in license_text:
                license_id = "MIT"
            elif "Apache License" in license_text:
                license_id = "Apache-2.0"
            elif "Redistribution and use in source and binary forms" in license_text:
                if "3. Neither" in license_text or "the names of" in license_text.lower():
                    license_id = "BSD-3-Clause"
                else:
                    license_id = "BSD-2-Clause"
            elif "BSD" in license_text and "Redistribution" in license_text:
                license_id = "BSD-3-Clause"
            break

    return license_id, license_text


def generate_supplement() -> tuple[str, bool]:
    """Generate the supplement text. Returns (text, has_copyleft_error)."""
    sdk_covered = load_sdk_covered()
    example_deps = load_example_deps()
    has_copyleft = False

    sections: list[str] = []
    sections.append("nxusKit Examples — Additional Dependencies")
    sections.append("=" * 44)
    sections.append("")
    sections.append("The following dependencies are used directly by examples")
    sections.append("and are NOT covered by the SDK's NOTICE file.")
    sections.append("")

    # Rust supplement
    uncovered_rust = [d for d in example_deps["rust"] if d not in sdk_covered]
    if uncovered_rust:
        sections.append("Rust Dependencies (example-only)")
        sections.append("-" * 33)
        sections.append("")
        for dep in uncovered_rust:
            license_id, license_text = find_crate_license(dep)
            verbose(f"Rust dep {dep}: {license_id}")

            # Check for copyleft
            if any(cl in license_id for cl in COPYLEFT_LICENSES):
                info(f"ERROR: Copyleft license detected: {dep} ({license_id})")
                has_copyleft = True

            sections.append(f"{dep}")
            sections.append(f"License: {license_id}")
            if license_text:
                sections.append("")
                sections.append("--- License Text ---")
                sections.append(license_text.rstrip())
                sections.append("--- End License Text ---")
            sections.append("")

    # Go supplement
    uncovered_go = [d for d in example_deps["go"] if d not in sdk_covered]
    if uncovered_go:
        sections.append("Go Dependencies (example-only)")
        sections.append("-" * 31)
        sections.append("")
        for dep in uncovered_go:
            license_id, license_text = find_go_module_license(dep)
            verbose(f"Go dep {dep}: {license_id}")

            if any(cl in license_id for cl in COPYLEFT_LICENSES):
                info(f"ERROR: Copyleft license detected: {dep} ({license_id})")
                has_copyleft = True

            sections.append(f"{dep}")
            sections.append(f"License: {license_id}")
            if license_text:
                sections.append("")
                sections.append("--- License Text ---")
                sections.append(license_text.rstrip())
                sections.append("--- End License Text ---")
            sections.append("")

    if not uncovered_rust and not uncovered_go:
        sections.append("(No additional dependencies — all example deps are covered by the SDK NOTICE)")
        sections.append("")

    return "\n".join(sections) + "\n", has_copyleft


def main() -> None:
    check_mode = "--check" in sys.argv

    info("Generating NOTICE supplement for example-only deps...")
    supplement_text, has_copyleft = generate_supplement()

    if has_copyleft:
        info("FATAL: Copyleft-licensed dependencies detected. Cannot proceed.")
        sys.exit(1)

    if check_mode:
        if not SUPPLEMENT_PATH.exists():
            info(f"ERROR: {SUPPLEMENT_PATH} does not exist. Run without --check first.")
            sys.exit(2)
        if supplement_text != SUPPLEMENT_PATH.read_text():
            info(f"ERROR: {SUPPLEMENT_PATH} is stale. Regenerate with:")
            info("  python3 scripts/generate-notice-supplement.py")
            sys.exit(2)
        info(f"OK: {SUPPLEMENT_PATH} is up to date.")
        sys.exit(0)

    SUPPLEMENT_PATH.write_text(supplement_text)
    info(f"Written to {SUPPLEMENT_PATH}")


if __name__ == "__main__":
    main()
