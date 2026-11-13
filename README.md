This is for the lions living in the wiry frames of my peripherals' bodies

# The Front Buttons

The Front Buttons is a Go daemon that drives an Elgato Stream Deck XL directly
over USB HID, with no Elgato software, on macOS (Apple silicon) and Windows
(amd64). Each key is an instance of a building block, called a brick, placed by
per machine configuration, written by hand or in a local web editor.

The bottom left key pages back and the bottom right key pages forward, on every
page, wrapping around. Live keys, such as a GitHub Actions status, keep updating
while their page is hidden, so switching to a page shows its current state at
once. Moving the deck between machines through a dock is normal operation: the
daemon keeps running and repaints the deck the moment it arrives.

Status: built and tested against the protocol as documented by open source
libraries, ahead of the hardware (`IMPLEMENTATION_DEVIATIONS.md`, DV1). The gates
that need the physical deck are listed in `TESTING.md`.

## Running

From this directory, with Go 1.27.3 and a C compiler (cgo is used for USB HID):

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

This installs a per user login agent and starts it. `scripts/uninstall-macos.sh`
removes it. On Windows, in PowerShell:

```powershell
scripts\install-windows.ps1 -BinaryPath .\the-front-buttons.exe
```

This registers a task that runs at your logon, in your session, at least
privilege. `scripts\uninstall-windows.ps1` removes it. Both scripts take the
start up definition from the binary itself (`-print-launch-agent` and
`-print-scheduled-task`), so it is written in one place.

## Configuration

The file is YAML. Every field is checked when it loads; a file with any problem
is refused whole, the last good configuration keeps running, and the problems
are logged and shown in the editor.

```yaml
schema_version: 1
pages:
  - name: Work
    keys:
      - row: 0
        column: 0
        kind: open_url
        parameters:
          url: https://github.com
          label: GitHub
          icon: brand-github
      - row: 0
        column: 1
        preset: discord_channel
        fields:
          server_id: "123456789"
          channel_id: "987654321"
          label: Party
      - row: 0
        column: 2
        kind: pipeline_status
        parameters:
          repository: octo-org/hello-world
          workflow: ci.yml
          branch: main
          credential: github-work
          label: Build
```

Rows count from 0 at the top and columns from 0 at the left, on the XL's 4 by 8
grid. Row 3, columns 0 and 7 are the navigation keys and cannot be assigned. A
key names exactly one of `kind` with `parameters`, or `preset` with `fields`.
Quote values YAML would read as numbers or booleans, such as IDs. Saving from
the editor rewrites the file and drops hand written comments.

### Icons

`icon` names one of the 5,000 or so bundled Tabler icons; the editor's **Icons**
page lists them all, drawn as they look on a key, with a search. For a picture
of your own, put a PNG in the `icons` folder beside the configuration file
(`~/Library/Application Support/the-front-buttons/icons/` on macOS,
`%AppData%\the-front-buttons\icons\` on Windows) and name it in `icon_file`,
which is used instead of `icon`:

```yaml
      - row: 0
        column: 3
        preset: dndbeyond_character
        fields:
          character_id: "143332240"
          label: Yeku
          icon_file: yeku.png
```

The name is a plain file name ending `.png`, with no folders. The file must be a
PNG of at most 512 pixels on each side and 1 MB. A key whose file is missing or
refused shows the missing icon marker, and the log says why.

### Key presses on macOS

A `press_key` key sends F13 to F24 (F13 to F20 on macOS). It needs macOS's
Accessibility permission, which recent macOS lists in System Settings, Privacy
and Security, as Device Control and Data Access. macOS ties the permission to
the exact build, so after installing or reinstalling:

1. Remove any old the-front-buttons entry from that list; do not add one with
   the plus button, which macOS does not match to this program.
2. Press the key once, and let macOS add its own entry.
3. Switch that entry on. The daemon notices within a few seconds and restarts
   itself to use it (the deck blinks off and on); then press the key.

A daemon run from a terminal rather than the login agent stops instead of
restarting, and says so in its log.

The install script signs the binary as `io.github.thescottbot.the-front-buttons`.

## Credentials

Secrets, such as a GitHub token, never go in the configuration file. A key names
a credential (`credential: github-work`), and the secret is stored in the
Keychain on macOS or Credential Manager on Windows through the editor's
credentials page, which never shows it again. For GitHub, use a fine grained
token with read access to Actions on the repositories the keys watch.

## The editor

The editor is at <http://127.0.0.1:48080/>, listening on this machine only. If
another program already holds port 48080, the daemon uses any free port instead,
logs a warning, and prints the address it chose; `-editor-port` picks another
number, and `-editor-port 0` always lets the system choose. An `open_editor` key
opens the current address from the deck. Every key, page and form in it works
from the keyboard.

## Checking

```bash
go test ./...
```

```bash
go tool -modfile=tools/go.mod golangci-lint run ./...
```

```bash
scripts/check-duplicate-code.sh
```

The first command includes the repository checks (prose and typography,
operating system use, this file's first line, and this file's brick reference).
The second runs the linters, including the import layering rules. The third looks
for duplicated code across every package. `TESTING.md` lists every suite and
every gate. Checking tools are pinned in their own module under `tools/`, so they
never enter the daemon's dependencies.

## Brick reference

Generated from each kind's and preset's declaration. Do not edit by hand: run
`UPDATE_README=1 go test ./internal/reference/` after changing a declaration.

### `brightness`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `level` | integer | required | 1 to 100 |  | How bright the deck goes, in percent, when the key is pressed; pressing it again goes back to the usual brightness. The key shows the percentage unless it has a label. |

### `deck_off`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `level` | integer | optional | 1 to 100 |  | How bright the deck is, in percent, while every other key is off; 20 if left out. Pressing this key, or any other, turns the keys back on. |

### `open_application`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `application` | text | required | 1 to 512 characters |  | The application to launch: a bundle identifier or name on macOS; an executable path or application user model ID on Windows. |
| `use_app_icon` | boolean | optional |  |  | Draw the application's own icon instead of Icon or Icon file, and colour the key to match the icon's edge; a Background you set still wins. Not available on Windows yet, where the key shows the missing icon marker. |

### `open_editor`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `open_url`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `url` | text | required | 1 to 2048 characters | link_url | The address the key opens: https, or a discord:// link, which opens the Discord app. |

### `page_marker`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |

### `pipeline_status`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `repository` | text | required | 3 to 140 characters |  | The GitHub repository, written owner/name. |
| `workflow` | text | required | 1 to 255 characters | token | The workflow's file name, such as ci.yml, or its numeric ID. |
| `branch` | text | required | 1 to 255 characters | branch | The branch whose runs the key shows. |
| `credential` | text, secret | required | 1 to 64 characters | identifier | The name of the stored GitHub token, a fine grained token with read access to Actions. |
| `interval` | duration | optional | 30s to 1h |  | How often to poll when no run is active; a minute if left out. Every 15 seconds while a run is queued or in progress. GitHub may ask for longer, and is obeyed. |
| `api_base_url` | text | optional | 9 to 2048 characters | https_url | The GitHub API address for GitHub Enterprise; api.github.com if left out. |

### `press_key`

A brick kind.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `key` | text | required |  | function_key | The key to press: F13 to F24. macOS has only F13 to F20, and asks for Accessibility permission the first time. |

### `discord_channel`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_server`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `server_id` | text | required | 1 to 32 characters | digits | The server ID, from Copy Server ID with Developer Mode on. |
| `channel_id` | text | required | 1 to 32 characters | digits | The channel ID, from Copy Channel ID with Developer Mode on. |

### `dndbeyond_character`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_character`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `character_id` | text | required | 1 to 32 characters | digits | The number at the end of the character sheet's address. |

### `teams_channel`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_channel`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `channel_id` | text | required | 4 to 256 characters |  | The channel ID beginning 19:, from the channel's URL. |
| `channel_name` | text | required | 1 to 256 characters |  | The channel's name. |
| `group_id` | text | required | 1 to 64 characters | identifier | The team's group ID, from the channel's URL. |
| `tenant_id` | text | required | 1 to 64 characters | identifier | The organisation's tenant ID, from the channel's URL. |

### `teams_chat`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_chat`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `participants` | text | required | 3 to 2048 characters |  | Comma separated user principal names, usually email addresses. |

### `teams_chat_by_id`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_chat`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `chat_id` | text | required | 4 to 256 characters |  | The chat ID beginning 19:, from a message's Copy link. |

### `teams_private_channel`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_channel`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `channel_id` | text | required | 4 to 256 characters |  | The channel ID beginning 19:, from the channel's URL. |
| `channel_name` | text | required | 1 to 256 characters |  | The channel's name. |
| `group_id` | text | required | 1 to 64 characters | identifier | The team's group ID, from the channel's URL. |
| `tenant_id` | text | required | 1 to 64 characters | identifier | The organisation's tenant ID, from the channel's URL. |

### `teams_shared_channel`

A preset that expands into `open_url`. Its action, which the editor shows, is `open_channel`.

| Field | Type | Required | Bounds | Format | Description |
|---|---|---|---|---|---|
| `label` | text | optional | 0 to 24 characters |  | Text shown on the key. |
| `icon` | text | optional | 1 to 64 characters | identifier | Name of a bundled icon. |
| `background` | text | optional |  | colour | Key background colour, written #RRGGBB. |
| `icon_file` | text | optional | 5 to 128 characters | png_file | A PNG of your own in the icons folder beside the configuration, used instead of Icon: up to 512 pixels square and 1 MB. |
| `channel_id` | text | required | 4 to 256 characters |  | The channel ID beginning 19:, from the channel's URL. |
| `channel_name` | text | required | 1 to 256 characters |  | The channel's name. |
| `group_id` | text | required | 1 to 64 characters | identifier | The team's group ID, from the channel's URL. |
| `tenant_id` | text | required | 1 to 64 characters | identifier | The organisation's tenant ID, from the channel's URL. |

## Where it came from

Written from the project specification and the author's engineering datum,
version 2. The Stream Deck protocol is read from the open source
python-elgato-streamdeck library; the commit read, and everything else the work
rests on, is recorded in `docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md`.
Vendored files are listed with their digests in `PROVENANCE.md`.
