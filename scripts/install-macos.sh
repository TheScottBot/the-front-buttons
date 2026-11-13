#!/usr/bin/env bash
# Installs The Front Buttons as a per user login agent on macOS (TE10), with no
# administrator rights: the agent lives in ~/Library/LaunchAgents and runs in the
# person's own session. The property list comes from the binary itself, so the
# label, the paths and the keys are written once, in Go.
#
# The binary is copied to a folder of its own and runs from there, so the agent
# never depends on where it was built, which may be a temporary folder macOS
# clears on restart. It is signed with the code signing certificate named
# below if the keychain has one, so macOS keeps the Accessibility permission
# across reinstalls; otherwise ad hoc, as before.
#
# Usage: scripts/install-macos.sh path/to/the-front-buttons
#
# FRONT_BUTTONS_SIGNING_IDENTITY names the certificate; it defaults to
# "The Front Buttons Signing".
set -euo pipefail

signing_identity="${FRONT_BUTTONS_SIGNING_IDENTITY:-The Front Buttons Signing}"
install_directory="$HOME/Library/Application Support/the-front-buttons/bin"

if [[ $# -ne 1 ]]; then
	echo "usage: $0 path/to/the-front-buttons" >&2
	exit 2
fi

# Refuse anything but an executable file before touching the system.
if [[ ! -f "$1" || ! -x "$1" ]]; then
	echo "error: $1 is not an executable file" >&2
	exit 2
fi

source_binary="$(cd "$(dirname "$1")" && pwd)/$(basename "$1")"
label="$("$source_binary" -print-launch-agent | plutil -extract Label raw -o - -)"
binary="$install_directory/the-front-buttons"
staged="$binary.new"

mkdir -p "$install_directory"
cp "$source_binary" "$staged"

# Go signs its builds ad hoc with the identifier a.out, which is how the binary
# would appear in System Settings, Privacy and Security, Accessibility, where
# the key presses and window keys need it. Signing it again with the agent's own
# label gives that entry a name a person can find. Ad hoc, macOS ties the
# permission to this exact build, so every rebuild needs granting again; signed
# with a certificate, the permission follows the label and the certificate, and
# a rebuild keeps it. The first signing with a new certificate may ask, once,
# to use its key.
if security find-identity -p codesigning | grep -qF "\"$signing_identity\""; then
	codesign --force --sign "$signing_identity" --identifier "$label" "$staged"
	echo "Signed with the certificate \"$signing_identity\"."
else
	codesign --force --sign - --identifier "$label" "$staged"
	echo "Signed ad hoc: Accessibility must be granted again after each reinstall. To keep it, create a" \
		"self signed code signing certificate named \"$signing_identity\" in Keychain Access, then install again."
fi

# Moved into place rather than copied over, so a running copy keeps its own
# file until it is stopped below.
mv -f "$staged" "$binary"

plist_text="$("$binary" -print-launch-agent)"
printf '%s\n' "$plist_text" | plutil -lint - >/dev/null
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
