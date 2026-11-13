# Setting up

How to run The Front Buttons, start it at login, and give it the permissions
it needs on macOS.

## Running

From the repository root, with Go 1.27.4 and a C compiler (cgo is used for USB HID):

```bash
go run ./cmd/the-front-buttons
```

`go run` is always a valid way to run it. To build a binary instead:

```bash
go build -o the-front-buttons ./cmd/the-front-buttons
```

It reads its configuration from the per machine file and prints the editor's
address:

| | macOS | Windows |
|---|---|---|
| Configuration | `~/Library/Application Support/the-front-buttons/config.yaml` | `%AppData%\the-front-buttons\config.yaml` |
| Logs | `~/Library/Application Support/the-front-buttons/state/` | `%AppData%\the-front-buttons\state\` |

Pass `-config` with a path to use another file. Close Elgato's own software
first: a deck it holds is reported as busy in the log.

## Starting at login

Neither needs administrator rights. Build the binary first, then on macOS:

```bash
scripts/install-macos.sh ./the-front-buttons
```

This copies the binary to `~/Library/Application Support/the-front-buttons/bin`,
installs a per user login agent that runs it from there, and starts it, so the
agent never depends on where the binary was built. `scripts/uninstall-macos.sh`
removes both. On Windows, in PowerShell:

```powershell
scripts\install-windows.ps1 -BinaryPath .\the-front-buttons.exe
```

This registers a task that runs at your logon, in your session, at least
privilege. `scripts\uninstall-windows.ps1` removes it. Both scripts take the
start up definition from the binary itself (`-print-launch-agent` and
`-print-scheduled-task`), so it is written in one place.

## Key presses on macOS

A `press_key` key sends F13 to F24 (F13 to F20 on macOS), and `press_shortcut`
and the JetBrains window keys need the same permission: macOS's Accessibility,
which recent macOS lists in System Settings, Privacy and Security, as Device
Control and Data Access. Signed ad hoc, macOS ties the permission to the exact
build, so after each install it needs granting again:

1. Remove any old the-front-buttons entry from that list; do not add one with
   the plus button, which macOS does not match to this program.
2. Press the key once, and let macOS add its own entry.
3. Switch that entry on. The daemon notices within a few seconds and restarts
   itself to use it (the deck blinks off and on); then press the key.

A daemon run from a terminal rather than the login agent stops instead of
restarting, and says so in its log.

The install script signs the binary as `io.github.thescottbot.the-front-buttons`.

To keep the permission across reinstalls, sign with a certificate of your own,
once:

1. In Keychain Access, choose Certificate Assistant, Create a Certificate.
2. Name it `The Front Buttons Signing`, with Identity Type Self Signed Root and
   Certificate Type Code Signing, and create it.
3. Install again. The script signs with the certificate when it finds it (macOS
   may ask once to let codesign use its key), and macOS then ties the permission
   to the certificate and the name rather than the build: grant it once more,
   and rebuilds keep it.

`FRONT_BUTTONS_SIGNING_IDENTITY` names a different certificate.
