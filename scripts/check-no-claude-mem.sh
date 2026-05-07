#!/usr/bin/env bash
# check-no-claude-mem.sh — ensure no AGPL strings from claude-mem were copied
# into this codebase. Run as a CI pre-test step.
#
# Exit 1 if any forbidden string is found; exit 0 otherwise.

set -euo pipefail

FORBIDDEN=(
  "buildObservationPrompt"
  "buildSummaryPrompt"
  "<observed_from_primary_session>"
  "SSEBroadcaster"
  "viewer-bundle.js"
  '"mem-search"'
)

FAIL=0
for pattern in "${FORBIDDEN[@]}"; do
  # Search tracked files only (-r recursive, ignore .git and this script itself)
  if grep -r --include="*.go" --include="*.json" --include="*.md" \
             --include="*.ts" --include="*.js" \
             --exclude-dir=".git" --exclude-dir="scripts" \
             -- "$pattern" . 2>/dev/null | grep -q .; then
    echo "ERROR: forbidden string found: $pattern" >&2
    grep -r --include="*.go" --include="*.json" --include="*.md" \
             --include="*.ts" --include="*.js" \
             --exclude-dir=".git" --exclude-dir="scripts" -n \
             -- "$pattern" . 2>/dev/null >&2 || true
    FAIL=1
  fi
done

if [ "$FAIL" -eq 1 ]; then
  echo "" >&2
  echo "FAIL: One or more claude-mem AGPL strings detected in the diff." >&2
  echo "      Do NOT copy code, prompts, or schemas from claude-mem (AGPL-3.0)." >&2
  exit 1
fi

echo "OK: no claude-mem strings found."
