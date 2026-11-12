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

# Go signs its builds ad hoc with the identifier a.out, which is how the binary
# then appears in System Settings, Privacy and Security, Accessibility, where a
# press_key key needs it. Signing it again, still ad hoc, with the agent's own
# label gives that entry a name a person can find. macOS still ties the
# permission to this exact build, so a rebuilt binary needs granting again.
codesign --force --sign - --identifier "$label" "$binary"
log_path="$(printf '%s\n' "$plist_text" | plutil -extract StandardErrorPath raw -o - -)"
agents_directory="$HOME/Library/LaunchAgents"
plist_path="$agents_directory/$label.plist"
domain="gui/$(id -u)"

mkdir -p "$agents_directory" "$(dirname "$log_path")"

# Replace a running copy rather than fail on it, so the script can be run again
# after rebuilding.
if launchctl print "$domain/$label" >/dev/null 2>&1; then
	launchctl bootout "$domain/$label"

	# bootout returns before launchd has finished removing the service, and a
	# bootstrap straight after it fails with error 5. Wait up to ten seconds for
	# the old copy to go.
	for _ in $(seq 1 100); do
		launchctl print "$domain/$label" >/dev/null 2>&1 || break
		sleep 0.1
	done

	if launchctl print "$domain/$label" >/dev/null 2>&1; then
		echo "error: the running copy of $label did not stop" >&2
		exit 1
	fi
fi

printf '%s\n' "$plist_text" >"$plist_path"
launchctl bootstrap "$domain" "$plist_path"

echo "Installed $label. It is running now and starts at every login."
