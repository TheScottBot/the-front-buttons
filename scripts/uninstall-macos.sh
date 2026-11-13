#!/usr/bin/env bash
# Removes the login agent that install-macos.sh installed. The label comes from
# the binary, as at install.
#
# Usage: scripts/uninstall-macos.sh path/to/the-front-buttons
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 path/to/the-front-buttons" >&2
	exit 2
fi

if [[ ! -f "$1" || ! -x "$1" ]]; then
	echo "error: $1 is not an executable file" >&2
	exit 2
fi

label="$("$1" -print-launch-agent | plutil -extract Label raw -o - -)"
plist_path="$HOME/Library/LaunchAgents/$label.plist"
domain="gui/$(id -u)"

if launchctl print "$domain/$label" >/dev/null 2>&1; then
	launchctl bootout "$domain/$label"
fi

# Only the files this product installed are removed: the agent, and the copy
# of the binary the install script runs it from.
if [[ -f "$plist_path" ]]; then
	rm "$plist_path"
fi

installed_binary="$HOME/Library/Application Support/the-front-buttons/bin/the-front-buttons"
if [[ -f "$installed_binary" ]]; then
	rm "$installed_binary"
	rmdir "$(dirname "$installed_binary")" 2>/dev/null || true
fi

echo "Removed $label."
