#!/usr/bin/env bash
# Installs The Front Buttons as a per user login agent on macOS (TE10), with no
# administrator rights: the agent lives in ~/Library/LaunchAgents and runs in the
# person's own session. The property list comes from the binary itself, so the
# label, the paths and the keys are written once, in Go.
#
# Usage: scripts/install-macos.sh path/to/the-front-buttons
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 path/to/the-front-buttons" >&2
	exit 2
fi

# Refuse anything but an executable file before touching the system.
if [[ ! -f "$1" || ! -x "$1" ]]; then
	echo "error: $1 is not an executable file" >&2
	exit 2
fi

binary="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
plist_text="$("$binary" -print-launch-agent)"

printf '%s\n' "$plist_text" | plutil -lint - >/dev/null

label="$(printf '%s\n' "$plist_text" | plutil -extract Label raw -o - -)"
log_path="$(printf '%s\n' "$plist_text" | plutil -extract StandardErrorPath raw -o - -)"
agents_directory="$HOME/Library/LaunchAgents"
plist_path="$agents_directory/$label.plist"
domain="gui/$(id -u)"

mkdir -p "$agents_directory" "$(dirname "$log_path")"

# Replace a running copy rather than fail on it, so the script can be run again
# after rebuilding.
if launchctl print "$domain/$label" >/dev/null 2>&1; then
	launchctl bootout "$domain/$label"
fi

printf '%s\n' "$plist_text" >"$plist_path"
launchctl bootstrap "$domain" "$plist_path"

echo "Installed $label. It is running now and starts at every login."
