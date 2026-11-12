#!/usr/bin/env bash
# Builds the editor's script from its TypeScript with esbuild,
# pinned in tools/go.mod and run as a Go tool, so no Node or npm is needed.
# esbuild strips the types without checking them. An import of something a
# module does not export would leave a feature silently dead, so that warning is
# made an error.
#
# go generate runs this to write internal/editor/static/editor.js, which the
# editor embeds and serves; a test runs it again into a scratch file and fails
# if the two differ, so the served script is never edited by hand.
#
# Usage: scripts/build-editor-script.sh [output file]
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
output="${1:-$root/internal/editor/static/editor.js}"

cd "$root"
go tool -modfile=tools/go.mod esbuild internal/editor/web/editor.ts \
	--bundle --format=iife --target=es2020 --charset=utf8 --legal-comments=none \
	--log-level=warning --log-override:import-is-undefined=error --outfile="$output"
