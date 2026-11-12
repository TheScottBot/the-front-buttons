#!/usr/bin/env bash
# Builds the editor's drag and drop script from its TypeScript with esbuild,
# pinned in tools/go.mod and run as a Go tool, so no Node or npm is needed.
# esbuild strips the types without checking them.
#
# go generate runs this to write internal/editor/static/drag.js, which the
# editor embeds and serves; a test runs it again into a scratch file and fails
# if the two differ, so the served script is never edited by hand.
#
# Usage: scripts/build-editor-script.sh [output file]
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="${1:-$root/internal/editor/static/drag.js}"

cd "$root"
go tool -modfile=tools/go.mod esbuild internal/editor/web/drag.ts \
	--bundle --format=iife --target=es2020 --charset=utf8 --legal-comments=none \
	--log-level=warning --outfile="$output"
