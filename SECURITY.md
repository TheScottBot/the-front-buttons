# Security

What The Front Buttons does and does not do, what it trusts, and what it does
not protect against. Every statement here names the code it rests on, so it can
be checked rather than taken on trust; a test fails the build if the platform
code, where every call into the operating system is kept, names a program this
page does not list.

## What runs

One background process per signed in user: the daemon. It needs no
administrator rights and installs no driver. On macOS it is a per user login
agent in `~/Library/LaunchAgents`, running the binary from
`~/Library/Application Support/the-front-buttons/bin`
(`scripts/install-macos.sh`); on Windows a per user logon task at least
privilege (`scripts/install-windows.ps1`).

## Network

**Listening.** Only the editor, on the loopback interface, `127.0.0.1`, port
48080 or any free port if that is taken (`internal/editor/server.go`, `Listen`).
Nothing listens on any other interface, so no other machine can reach it.

**Outgoing.** Three kinds of request, and no others:

- **The GitHub API**, for pipeline status keys only: the workflow runs of the
  repository a key names, with the token stored for it
  (`internal/githubactions/client.go`). The token is sent only to the host it
  was stored for, `api.github.com` unless a GitHub Enterprise host was given on
  the editor's credentials page; that binding lives beside the token in the
  credential store, never in configuration, so a configuration from elsewhere
  cannot send the token anywhere else (`internal/platform/credentials.go`,
  `CredentialHost`). Redirects to another domain drop the token (Go's
  `net/http`).
- **Mural's API**, for the Mural timer keys only: its token endpoint,
  `https://app.mural.co/api/public/v1/authorization/oauth2/token`, sent the
  code Mural returns, or the stored refresh token, with the client ID and
  secret of the person's own Mural app (`internal/muralapi/client.go`); and a
  board's timer, `/api/public/v1/murals/<workspace>.<number>/timer`, started
  or added to with the access token, for the one board a key's link names
  (`internal/muralapi/timer.go`). Each
  of the three is bound to `app.mural.co` in the credential store, and the
  client refuses to send one bound anywhere else, so a configuration naming
  one as a GitHub key's credential cannot send it to GitHub, nor the GitHub
  client send it anywhere but where it is bound. Only an OAuth error code
  from an answer is ever repeated, never the rest, which may echo a token.
- **Spotify's image server**, for the album key only: the current track's art,
  from `https://i.scdn.co/image/` and nowhere else, once per track, at most
  2 MB within 5 seconds (`internal/platform/systemadapter/spotify.go`,
  `spotify_darwin.go`).

Keys that open a link, such as Discord, D&D Beyond, Roll20, Teams or any URL,
hand it to the operating system's opener: the browser or app makes that
connection, not the daemon. Links are `https`, or `discord://` on discord.com
(`internal/brick/field.go`, `isLinkURL`).

There is no telemetry, analytics, crash reporting or update check.

## Programs run

Each by its fixed path, with its arguments as separate values, never through a
shell or `cmd.exe`:

| Program | When | Code |
|---|---|---|
| `/usr/bin/open` | a key opens a link or an application (macOS) | `internal/platform/systemadapter/launch.go` |
| `/usr/bin/osascript` | the Spotify keys: five fixed scripts, nothing from configuration | `internal/platform/systemadapter/spotify.go` |
| `/usr/bin/plutil` | the editor's app picker reads installed applications' names | `internal/platform/systemadapter/system_darwin.go` |
| `/usr/bin/security` | the Keychain, through go-keyring; a secret goes on its standard input, never its command line | `internal/platform/systemadapter/credentials.go` |
| `powershell.exe` | the editor's app picker lists the Start menu (Windows), with one fixed command | `internal/platform/systemadapter/system_windows.go` |
| `powershell.exe` | Triple press to close reads a Start menu entry's shortcut (Windows), with one fixed command; the entry's app ID goes in an environment variable, never in the command | `internal/platform/systemadapter/quit_windows.go` |
| the daemon itself | `-accessibility-check`, asking macOS afresh whether it is trusted | `internal/platform/systemadapter/keys_darwin.go` |

On Windows, links and applications open through `ShellExecuteW` and key presses
go through `SendInput`, both called directly. Triple press to close lists the
top level windows, reads each one's process's program path and app ID, and posts
the application's own windows the message their close buttons send, all called
directly; it never ends a process. A Squirrel install's program is found by the
`Update.exe` its shortcut runs, which is only matched against, never run.

## Operating system permissions and access

- **Accessibility (macOS)**, granted by the person, for five things only:
  posting the key presses keys are configured to send (`press_key`,
  `press_shortcut`, the paste a `mural_share_link` key sends, and an
  application's own keys, such as GoLand's); reading
  and raising the windows of JetBrains IDEs when a window key is pressed, and
  no other application's (`windows_darwin.go`); bringing an app screen's
  application to the front before one of its keys sends key presses, and
  reading which application is in front to know it has come
  (`activation_darwin.go`); reading the badge on each Dock icon, such as a
  count of unread messages, every two seconds while a key shows one, from the
  Dock alone and never from the application (`badges_darwin.go`); and asking
  whether it is trusted. It never reads keystrokes or window contents.
- **Automation of Spotify (macOS)**, asked for on first use: play, pause, change
  track, and read whether it is playing and its art.
- **USB**: opens Elgato devices only, vendor ID 0x0FD9
  (`internal/deck/hidtransport/hidtransport.go`). On macOS both drivers open
  the deck seized, for this process alone: hidapi through go-hid, and the
  IOKit driver with `kIOHIDOptionsTypeSeizeDevice`, as hidapi does
  (`iokit_darwin.go`). On Windows the purego driver opens it shared for
  reading and writing, as hidapi does there (`winhid_windows.go`).
- **AppKit**: reads applications' icons and the running applications.
- **The clipboard**: a `mural_share_link` key writes its configured message
  and link to the clipboard, in place of what was there, and pastes it; the
  daemon never reads the clipboard (`clipboard_darwin.go`, `clipboard_windows.go`).

Installs signed with the person's own code signing certificate keep the
Accessibility grant across reinstalls; ad hoc signed ones need it granted again
(`docs/SETUP.md`, Key presses on macOS).

## System libraries without cgo

The `platform_bridge` setting (docs/CONFIGURATION.md) chooses between cgo and
purego, `github.com/ebitengine/purego`, the one module the purego work adds.
Purego calls the system's own libraries from Go, opened by their fixed system
paths and never from a path in configuration or the environment:

- **macOS**: CoreFoundation, Foundation, AppKit and ApplicationServices from
  `/System/Library/Frameworks`, and `/usr/lib/libobjc.A.dylib`, for the
  adapters (`internal/platform/objcbridge/frameworks_darwin.go`); IOKit and
  CoreFoundation for the deck's driver (`internal/deck/hidtransport/iokit_darwin.go`).
- **Windows**: `hid.dll` and `setupapi.dll` from the system folder, for the
  deck's driver, with kernel32 through `golang.org/x/sys`, already a
  dependency (`internal/deck/hidtransport/winhid_windows.go`).

They are the same libraries the cgo code links against, called for the same
things; nothing new is reached. Each adapter and driver looks up its functions
once, in one place that recovers from a missing function or library, and then
reports itself unavailable rather than crashing the daemon
(`internal/platform/objcbridge/binding_darwin.go`, `Bind`); an Objective C
class or selector is checked before it is first used, so a missing one is
reported the same way rather than raising an exception. The IOKit driver makes
two purego callbacks, for input reports and for the deck's removal, once per
process, and finds the device each call is for by a number it registered
(`iokit_input_darwin.go`). Purego opens no connection and runs no program, so
the Network and Programs run sections above are unchanged.

## Files

All under the person's own folders, created readable by the person only where
they hold anything private:

- **Read**: the configuration file, at most 1 MiB (`internal/config`); PNG icons
  in its icons folder, opened through an `os.Root` so neither a name nor a
  symbolic link reaches outside it (`internal/usericons`); installed application
  bundles, for the app picker.
- **Written**: the configuration file, atomically; the rotating log, mode 0600,
  in the state folder, mode 0700; the editor's session file, mode 0600, holding
  its address and token for `-open-editor` (`cmd/the-front-buttons/session.go`).
  Windows keeps no permission bits; there these files are private because the
  state folder is in the person's own profile, which other accounts cannot
  open.

## Secrets

Tokens live only in the macOS Keychain or Windows Credential Manager, under the
service name `the-front-buttons`, each with the host it may be sent to beside
it. Mural's sign in keeps three there: `mural-client-id`, `mural-client-secret`
and `mural-refresh-token`, bound to `app.mural.co`; its access token, which
lasts fifteen minutes, is held in memory only. Configuration names a credential and never holds one. The editor's
credentials page is write only: it shows each name, whether it is stored and its
host, never the value. A test runs the whole composition against a GitHub that
refuses the token and fails if the token reaches the log.

## The log

It records the configuration file's path, key kinds and positions, application
identifiers and the badges keys show, icon file names, credential names, reason categories, and the
editor's request method and path without query. It never records a URL, an ID,
a secret, a request body or what a key typed.

## Trust boundaries

| Boundary | Treated as | Defences |
|---|---|---|
| USB input reports from the deck | hostile bytes | exact length checked before decoding; the decoder cannot read past its buffer; fuzzed in CI |
| The configuration file | untrusted, though the person's own | bounded at 1 MiB; one YAML document, no anchors, aliases or duplicate keys; every field checked by type, bounds and format; unknown fields refused; fuzzed in CI |
| Editor requests | untrusted, from web pages and from other accounts on the machine | the editor's own Host only, defeating DNS rebinding; a session cookie, HttpOnly and SameSite Strict, given only by a one time login link, on every page; any Origin the editor's own, and every change carrying the 256 bit token; bodies bounded at 64 KiB; a strict content security policy; no framing; the deck's event stream limited to four at once and to pages drawn like the deck |
| GitHub API responses | untrusted | bounded at 1 MiB; only three fields read; a run's address opened only if https |
| Spotify's answers | untrusted | the art fetched only from `i.scdn.co`; scripts fixed, never built from configuration |
| URLs and application names from configuration | untrusted | refused if empty, over 4,096 bytes, led by a hyphen or holding a control character; passed as one argument, never through a shell |

## The deck in the editor

With Render like deck on, the editor shows the deck as it runs, through
`internal/deckmirror`, which the daemon writes and the editor reads, in one
process. It holds only pictures the daemon drew of the keys whose look depends
on live state (live keys, keys showing a Dock badge, keys showing a press's
error), and word of which tile was pressed, let go of or redrawn. It never holds the state
behind a picture: a track's name, a run's result, a badge's count or a token.
Album art reaches the browser only as pixels the daemon drew, never as
Spotify's own file.

Two requests read it, behind the same guard as every other: a key's picture
asked for live (`/preview.png` with `live=1`), and an event stream
(`/events`) naming tiles alone. Both give nothing with the switch off. The
stream is the one long request: it has no write deadline, ends when its page
closes, and at most four may be open at once
(`internal/editor/events.go`, `cmd/the-front-buttons/main.go`). The daemon
never waits on the editor: a page that stops reading loses events.

## Opening the editor

Every editor page needs a session. The deck's editor key opens a fresh one time
login link, valid once within two minutes; `the-front-buttons -open-editor`
reads the address and token from the state folder, asks the running daemon for
such a link with the token as a header, and opens it, so the token never appears
in a command line another account could see (`internal/editor/session.go`).

One page needs no session: `/mural/callback`, where Mural returns after the
person allows their app, as that return comes from Mural's site and so carries
no SameSite Strict cookie. It acts only on a state the editor minted when the
person, with a session, chose Connect to Mural: 32 random bytes, taken once,
within ten minutes. Only then is Mural's code traded. The page it shows holds
no editor token and no secret, and repeats only an OAuth error code. Connect to
Mural is a link rather than a form, as the editor's content security policy
lets forms go only to the editor itself, redirects included
(`internal/editor/muralsignin.go`).

## What it does not protect against

- **Software running as the same person.** It can read the configuration file,
  the log and the session file, and can read the Keychain entries, which are
  created by `/usr/bin/security` and so are readable through it. Protection
  between programs of one person is the operating system's, not this daemon's.
  The session and the token binding protect against web pages, other accounts
  on the machine, and configurations from elsewhere.
- **Someone who can edit the configuration.** A key opens what it is configured
  to open, so such a person can make a key open a harmful https link or press
  any shortcut. They cannot send a stored token to a host it was not stored for.
- **Colleagues' addresses in configuration.** A Teams link can hold them
  (PRIVACY.md).

## Limitations

- Windows code is built, linted and unit tested on every change, but has not
  yet run on a Windows machine.
- The editor has not been checked with a screen reader.
- Dependencies were checked with `govulncheck` on 13 November 2026: the only
  finding, GO-2026-6859 in Go's own `net/http`, is fixed by Go 1.27.4, which the
  build now requires, and the check on 1.27.4 found no vulnerabilities. Run
  `go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...` to check again.
  That check predates purego, which has not been checked since it was added.
- The purego bridge has not yet driven the physical deck, nor acted on the
  desktop (key presses, quitting, raising windows, the clipboard); cgo stays
  the default until it has.

## Reporting

Report a problem to the repository's owner privately rather than in a public
issue.
