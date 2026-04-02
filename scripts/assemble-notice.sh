#!/usr/bin/env bash
# assemble-notice.sh — Assemble NOTICE from SDK base + examples supplement
#
# Usage:
#   scripts/assemble-notice.sh [OPTIONS]
#
# Options:
#   --check    Regenerate to temp, diff against committed NOTICE, exit 2 if stale
#   --verbose  Print progress
#   --help     Print this usage message
#
# Exit codes:
#   0  Success (or --check: NOTICE up to date)
#   1  Error (missing inputs)
#   2  --check mode: NOTICE is stale

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SDK_BASE="$REPO_ROOT/compliance/sdk-notice-base.txt"
SUPPLEMENT="$REPO_ROOT/compliance/NOTICE-supplement.txt"

CHECK_MODE=false
VERBOSE=false

for arg in "$@"; do
    case "$arg" in
        --check)   CHECK_MODE=true ;;
        --verbose) VERBOSE=true ;;
        --help|-h)
            sed -n '2,/^$/{ s/^# \{0,1\}//; p; }' "$0"
            exit 0
            ;;
        *)
            echo "ERROR: Unknown option: $arg" >&2
            exit 1
            ;;
    esac
done

info()    { echo "[assemble] $*"; }
verbose() { $VERBOSE && echo "[assemble]   $*" || true; }

# Verify inputs exist
if [[ ! -f "$SDK_BASE" ]]; then
    echo "ERROR: SDK NOTICE base not found: $SDK_BASE" >&2
    echo "Copy it from the SDK repo NOTICE file into $SDK_BASE" >&2
    exit 1
fi

# Generate supplement if it doesn't exist
if [[ ! -f "$SUPPLEMENT" ]]; then
    info "Supplement not found, generating..."
    python3 "$REPO_ROOT/scripts/generate-notice-supplement.py" || {
        echo "ERROR: Failed to generate supplement" >&2
        exit 1
    }
fi

YEAR="$(date +%Y)"

assemble() {
    local out_dir="$1"
    local notice_file="$out_dir/NOTICE"
    local md_file="$out_dir/THIRD-PARTY-NOTICES.md"

    # ── Plain text NOTICE ──
    {
        cat "$SDK_BASE"
        echo ""
        echo "================================================================================"
        echo ""
        cat "$SUPPLEMENT"
    } > "$notice_file"

    verbose "NOTICE: $(wc -l < "$notice_file" | tr -d ' ') lines"

    # ── Markdown THIRD-PARTY-NOTICES.md ──
    {
        cat <<MDHEADER
# Third-Party Notices

**nxusKit Examples** — Copyright $YEAR nxus Systems

This document lists third-party software used in nxusKit and its examples,
along with their respective license information.

---

## SDK Dependencies

The following section is from the nxusKit SDK NOTICE file.

\`\`\`text
MDHEADER
        # SDK base content (skip the first header lines for cleaner markdown)
        tail -n +9 "$SDK_BASE"
        echo '```'
        echo ""
        echo "---"
        echo ""
        echo "## Example-Only Dependencies"
        echo ""
        echo '```text'
        cat "$SUPPLEMENT"
        echo '```'
    } > "$md_file"

    verbose "THIRD-PARTY-NOTICES.md: $(wc -l < "$md_file" | tr -d ' ') lines"
}

if $CHECK_MODE; then
    OUT_DIR="$(mktemp -d)"
    trap 'rm -rf "$OUT_DIR"' EXIT

    info "Check mode: regenerating from committed base + supplement..."

    # In check mode, assemble from committed inputs (no regeneration of supplement).
    # The supplement is regenerated locally by the developer; CI only verifies
    # the final NOTICE matches what the committed inputs would produce.
    if [[ ! -f "$SUPPLEMENT" ]]; then
        info "ERROR: $SUPPLEMENT does not exist. Run: python3 scripts/generate-notice-supplement.py"
        exit 2
    fi

    assemble "$OUT_DIR"

    stale=false
    for file in NOTICE THIRD-PARTY-NOTICES.md; do
        if [[ ! -f "$REPO_ROOT/$file" ]]; then
            info "ERROR: $file does not exist. Run without --check first."
            stale=true
            continue
        fi
        if ! diff -q "$OUT_DIR/$file" "$REPO_ROOT/$file" &>/dev/null; then
            info "ERROR: $file is stale. Regenerate with: scripts/assemble-notice.sh"
            stale=true
        else
            verbose "$file is up to date."
        fi
    done

    if $stale; then
        exit 2
    fi
    info "OK: NOTICE files are up to date."
else
    info "Assembling NOTICE files..."
    assemble "$REPO_ROOT"
    info "Done. Generated:"
    info "  $REPO_ROOT/NOTICE"
    info "  $REPO_ROOT/THIRD-PARTY-NOTICES.md"
fi
