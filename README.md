This is for the lions living in the wiry frames of my peripherals' bodies

# The Front Buttons

Your Elgato Stream Deck XL, driven straight over USB by one small Go daemon,
with no Elgato software. Set it up from a web editor on your own machine.

- **Keys that do things:** open sites and apps, send shortcuts, show GitHub
  Actions runs, play Spotify, dim the deck, and more.
- **App screens:** hold an app's key and the deck switches to that app's own
  keys, such as Build and Debug for a JetBrains IDE, sessions for Claude or
  Copilot's suggestions for VS Code. The app's icon, top left, takes you back.
- **Badges:** an app's Dock badge, such as unread messages, shows on its key.
- **Pages:** the bottom corners page back and forward; live keys keep updating
  while hidden.
- **Moves between machines:** dock the deck to another computer and it
  repaints the moment it arrives.

Runs on macOS (Apple silicon), tested on the deck. Windows (amd64) is built and
tested in CI but not yet run on a Windows machine.

## Run it

You need Go 1.27.4 and a C compiler. Close Elgato's own software first.

```bash
go run ./cmd/the-front-buttons
```

To have it start at login on macOS:

```bash
go build -o the-front-buttons ./cmd/the-front-buttons
```

```bash
scripts/install-macos.sh ./the-front-buttons
```

Windows, and the macOS permission for key presses, are in
[docs/SETUP.md](docs/SETUP.md).

## Set it up

Press an `open_editor` key on the deck, or run:

```bash
"$HOME/Library/Application Support/the-front-buttons/bin/the-front-buttons" -open-editor
```

## More

| | |
|---|---|
| [docs/SETUP.md](docs/SETUP.md) | Running, starting at login, macOS permissions and signing |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | The configuration file, app screens, badges, icons, credentials |
| [docs/EDITOR.md](docs/EDITOR.md) | The editor and its sessions |
| [docs/BRICKS.md](docs/BRICKS.md) | Every brick and preset, and their fields |
| [SECURITY.md](SECURITY.md) | What it does and does not do, and what it trusts |
| [PRIVACY.md](PRIVACY.md) | What it stores and sends |
| [TESTING.md](TESTING.md) | Running the checks, every suite and gate |
| [CHANGELOG.md](CHANGELOG.md) | What has changed |

## Where it came from

Written from the project specification and the author's engineering datum,
version 2. The Stream Deck protocol is read from the open source
python-elgato-streamdeck library; the commit read, and everything else the work
rests on, is recorded in `docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md`.
Vendored files are listed with their digests in `PROVENANCE.md`.
