#!/usr/bin/env bash
# Licensing / entitlement scenarios (launch readiness PoR §4).
#
# Default: prints the matrix and exits 0.
#
# Automated slices (no browser) when RUN_ENTITLEMENT_TESTS=1:
#   E1  Clear token + env → CE command must succeed (0); Pro command must fail (non-zero).
#   E4  Invalid token file → same expectations as E1 (restore after).
#
# Required environment (E1 / E4):
#   ENT_CE_CMD   shell command for a Community-tier binary (must exit 0 without license)
#   ENT_PRO_CMD  shell command for a Pro-tier binary (must exit non-zero without valid Pro license)
#
# Optional:
#   ENT_TOKEN_FILE  (default: $HOME/.nxuskit/license.token)
#   ENT_INVALID_JWT contents written to token file for E4 (default: "invalid.jwt.stub")
#   ENT_RUN_E4      set to 1 to run scenario E4 after E1
#   ENT_JSON        set to 1 to print a JSON summary to stdout at the end (human progress on stderr)
#
# E2 / E3 / E5 remain manual (interactive login, OSS vs Pro bundle pairing) until CLI stderr
# is stable and/or nxuskit-cli is available in CI.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOKEN_FILE="${ENT_TOKEN_FILE:-$HOME/.nxuskit/license.token}"
INVALID_JWT="${ENT_INVALID_JWT:-invalid.jwt.stub}"
JSON_OUT="${ENT_JSON:-0}"
SCENARIO_JSONL=""

emit_json_summary() {
  [[ "$JSON_OUT" == "1" ]] || return 0
  if [[ ! -f "${SCENARIO_JSONL:-}" ]]; then
    jq -nc '{ok: true, scenarios: []}'
    return
  fi
  jq -s '{ok: (all(.status == "ok")), scenarios: .}' "${SCENARIO_JSONL}"
}

cat <<'EOF' >&2
Entitlement scenario matrix (see DevOps PoR nxuskit-examples-launch-readiness-20260325.md):

  E1  No token          CE example OK | Pro example → license required
  E2  Trial Pro         interactive login
  E3  Valid Pro key     interactive / purchase id
  E4  Invalid token     CE OK | Pro → license required
  E5  OSS + Pro ex      CE OK | Pro-tier example → entitlement error

EOF

if [[ "${RUN_ENTITLEMENT_TESTS:-}" != "1" ]]; then
  echo "skip: set RUN_ENTITLEMENT_TESTS=1 and ENT_CE_CMD / ENT_PRO_CMD to execute E1/E4." >&2
  exit 0
fi

if [[ -z "${ENT_CE_CMD:-}" ]] || [[ -z "${ENT_PRO_CMD:-}" ]]; then
  echo "error: set ENT_CE_CMD and ENT_PRO_CMD (see script header)." >&2
  exit 1
fi

if [[ "$JSON_OUT" == "1" ]]; then
  if ! command -v jq >/dev/null 2>&1; then
    echo "error: ENT_JSON=1 requires jq" >&2
    exit 1
  fi
  SCENARIO_JSONL="$(mktemp)"
fi

record_scenario() {
  local id="$1" ce="$2" pro="$3" st="$4"
  [[ "$JSON_OUT" == "1" ]] || return 0
  jq -nc --arg id "$id" --argjson ce "$ce" --argjson pro "$pro" --arg st "$st" \
    '{id:$id, ce_exit:$ce, pro_exit:$pro, status:$st}' >>"${SCENARIO_JSONL}"
}

BACKUP_DIR="$(mktemp -d)"
cleanup() {
  if [[ -f "${BACKUP_DIR}/license.token.saved" ]]; then
    mkdir -p "$(dirname "$TOKEN_FILE")"
    mv "${BACKUP_DIR}/license.token.saved" "$TOKEN_FILE"
  else
    rm -f "$TOKEN_FILE"
  fi
  rmdir "$BACKUP_DIR" 2>/dev/null || true
  rm -f "${SCENARIO_JSONL:-}"
}
trap cleanup EXIT

backup_token() {
  if [[ -f "$TOKEN_FILE" ]]; then
    cp "$TOKEN_FILE" "${BACKUP_DIR}/license.token.saved"
  fi
}

scenario_no_token() {
  backup_token
  rm -f "$TOKEN_FILE"
  unset NXUSKIT_LICENSE_TOKEN || true
  export NXUSKIT_LICENSE_TOKEN=""

  echo "== CE (expect 0): $ENT_CE_CMD" >&2
  set +e
  bash -c "$ENT_CE_CMD"
  local ce_ok=$?

  echo "== Pro (expect non-zero): $ENT_PRO_CMD" >&2
  bash -c "$ENT_PRO_CMD"
  local pro_st=$?
  set -e

  if [[ "$ce_ok" -ne 0 ]]; then
    echo "error: CE command exited $ce_ok, expected 0" >&2
    record_scenario "E1" "$ce_ok" "$pro_st" "fail"
    emit_json_summary
    exit 1
  fi
  if [[ "$pro_st" -eq 0 ]]; then
    echo "error: Pro command exited 0, expected non-zero without license" >&2
    record_scenario "E1" "$ce_ok" "$pro_st" "fail"
    emit_json_summary
    exit 1
  fi
  record_scenario "E1" "$ce_ok" "$pro_st" "ok"
  echo "OK: no-token scenario (CE ok, Pro blocked)." >&2
}

scenario_invalid_token() {
  backup_token
  mkdir -p "$(dirname "$TOKEN_FILE")"
  printf '%s' "$INVALID_JWT" >"$TOKEN_FILE"
  unset NXUSKIT_LICENSE_TOKEN || true

  echo "== CE with bad token (expect 0): $ENT_CE_CMD" >&2
  set +e
  bash -c "$ENT_CE_CMD"
  local ce_ok=$?

  echo "== Pro with bad token (expect non-zero): $ENT_PRO_CMD" >&2
  bash -c "$ENT_PRO_CMD"
  local pro_st=$?
  set -e

  if [[ "$ce_ok" -ne 0 ]]; then
    echo "error: CE command exited $ce_ok with invalid token" >&2
    record_scenario "E4" "$ce_ok" "$pro_st" "fail"
    emit_json_summary
    exit 1
  fi
  if [[ "$pro_st" -eq 0 ]]; then
    echo "error: Pro command exited 0 with invalid token" >&2
    record_scenario "E4" "$ce_ok" "$pro_st" "fail"
    emit_json_summary
    exit 1
  fi
  record_scenario "E4" "$ce_ok" "$pro_st" "ok"
  echo "OK: invalid-token scenario." >&2
}

scenario_no_token

if [[ "${ENT_RUN_E4:-}" == "1" ]]; then
  scenario_invalid_token
fi

echo "Entitlement automation finished (repo: ${REPO_ROOT})." >&2

emit_json_summary
