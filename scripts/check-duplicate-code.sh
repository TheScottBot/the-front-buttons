#!/usr/bin/env bash
# Fails if any block of Go code is duplicated anywhere in the tree (engineering
# datum 0.7).
#
# golangci-lint's dupl compares only within one package (finding TF9), so the
# standalone mibk/dupl, pinned in tools/go.mod, scans every Go file across every
# package instead. It always exits zero, even when it finds duplicates, so this
# script turns any output into a failure.
#
# The threshold is dupl's default of 100 tokens, which caught the planted cross
# package duplicate in TE1. It is passed explicitly so a change to the tool's
# default cannot silently change the rule.
set -euo pipefail

duplicate_code_threshold_tokens=100

cd "$(dirname "$0")/.."

duplicate_report="$(go tool -modfile=tools/go.mod dupl -plumbing -t "$duplicate_code_threshold_tokens" .)"

if [[ -n "$duplicate_report" ]]; then
	echo "Duplicate code found (threshold ${duplicate_code_threshold_tokens} tokens):"
	echo "$duplicate_report"
	exit 1
fi

echo "No duplicate code at or above ${duplicate_code_threshold_tokens} tokens."
