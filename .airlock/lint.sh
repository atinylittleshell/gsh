#!/usr/bin/env bash
set -euo pipefail

# Compute changed files between base and head
BASE_SHA="${AIRLOCK_BASE_SHA:-HEAD~1}"
HEAD_SHA="${AIRLOCK_HEAD_SHA:-HEAD}"

# Get changed files
CHANGED_FILES=$(git diff --name-only --diff-filter=ACMR "$BASE_SHA" "$HEAD_SHA" 2>/dev/null || git diff --name-only --diff-filter=ACMR HEAD 2>/dev/null || true)

if [ -z "$CHANGED_FILES" ]; then
  echo "No changed files detected."
  exit 0
fi

# Filter Go files
GO_FILES=$(echo "$CHANGED_FILES" | grep '\.go$' || true)

if [ -z "$GO_FILES" ]; then
  echo "No Go files changed. Nothing to lint."
  exit 0
fi

echo "Changed Go files:"
echo "$GO_FILES"
echo ""

# Collect unique directories containing changed Go files
GO_DIRS=$(echo "$GO_FILES" | xargs -I{} dirname {} | sort -u | sed 's|^|./|')

# Step 1: Format with auto-fix
echo "==> Running golangci-lint fmt..."
echo "$GO_DIRS" | xargs golangci-lint fmt || true

# Step 2: Lint with auto-fix
echo "==> Running golangci-lint run --fix..."
echo "$GO_DIRS" | xargs golangci-lint run --fix || true

# Step 3: Lint in check mode (verify)
echo "==> Running golangci-lint run (check mode)..."
echo "$GO_DIRS" | xargs golangci-lint run
echo ""
echo "All lint checks passed."
