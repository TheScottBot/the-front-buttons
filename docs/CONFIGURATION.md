# Configuration

The per machine file, `config.yaml`; [SETUP.md](SETUP.md) says where it is. The
editor writes it for you, but it is plain YAML you can edit by hand.

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

A key held down on the deck is drawn pressed, dimmed and a little smaller,
until it is let go, so every press is seen whether or not it changes anything
else.

Rows count from 0 at the top and columns from 0 at the left, on the XL's 4 by 8
grid. Row 3, columns 0 and 7 are the navigation keys and cannot be assigned. A
key names exactly one of `kind` with `parameters`, or `preset` with `fields`.
Quote values YAML would read as numbers or booleans, such as IDs. Saving from
the editor rewrites the file and drops hand written comments.

## App screens

An app screen is a page of one application's keys. Tick App screen on an
`open_application` key: a tap still just opens the application, and holding the
key for half a second opens it and shows that page as well; a pink bar fills
along the key's bottom as the half second passes. The editor creates
the page the first time. The application's
icon is top left on its screen, and pressing it goes back to the regular page.
Paging never reaches an app screen, and back and forward leave one.

An application may have more than one screen: **Add another screen** on any of
its screens adds one straight after its last, or give several pages the same
`application`. Holding the app key opens the first; back and forward then page
between the application's screens, wrapping round, and their tiles show which
screen it is, such as 1/2. With a single screen they leave it, as before.

Holding a `page_marker` key for half a second opens the page dashboard on the
deck, in place of the page, with the same bar filling towards it. Its top left key is the marker you held, and
pressing it goes back to the page. The next six keys of the top row show the
local time as HHMMSS, a figure to a key, in the look of Thyme, the author's
clock: its pink JetBrains Mono figures on its navy. Each pair is drawn towards
each other, with a small colon after the hours and the minutes, so the six keys
read as 15:31:42. The top right key is a
world icon, which shows the world clocks: unless the file names its own, see
[World clocks](#world-clocks), UTC, London (LDN), Berlin (BER) and Sao Paulo
(SAO), a row each, its label and then its HHMMSS, with the world icon
kept in the top right to go back to the dashboard, and the local date, YY, MM
and DD, down the right column beneath it, small in each key's top right corner
in Thyme's date colour, with its word, year, month or day, a little smaller in
the bottom right. Both tick each second. The
second row holds your regular pages, each by its number and name,
the page you came from marked; the third holds the app screens, each by its
application's icon and name. Pressing a page or screen shows it. With more app
screens than a row holds, the paging tiles in the bottom corners move that row
along, a row's worth at a time. The dashboard stays until a key is pressed; a
tap on the marker, rather than a hold, still does nothing. The editor's
Dashboard section shows it as the deck draws it.

```yaml
pages:
  - name: Work
    keys:
      - row: 0
        column: 1
        kind: open_application
        parameters:
          application: com.jetbrains.goland
          app_screen: true
  - name: GoLand
    application: com.jetbrains.goland
    keys:
      - row: 1
        column: 1
        kind: jetbrains_build
        parameters:
          label: Build
```

A page names an `application` to be its screen, one per application, and row 0,
column 0 is kept for the icon. Some applications have bricks of their own,
the `jetbrains_` keys every JetBrains IDE shares, GoLand, IntelliJ IDEA,
PyCharm, Rider and the rest, which only those applications' screens may hold;
each IDE has its own screen. GitKraken, Claude, ChatGPT and VS Code, with GitHub Copilot's keys, have
keys of their own: `gitkraken_`, `claude_`, `chatgpt_` and `vscode_`. A key that
sends key presses on an app screen, a shortcut or one of the application's own,
brings the application to the front first unless it is there already; if it is
not open, nothing is sent and the key shows "not open".

## Notification badges

Tick Show notifications on an `open_application` key, `show_notifications:
true` in the file, and the key shows the badge its application has on its Dock
icon, top right in the Dock's red: a count, or a dot for a badge without one.
It follows the Dock within a couple of seconds. Reading the Dock needs the
Accessibility permission key presses already need. Not available on Windows
yet.

## Triple press to close

Tick Triple press to close on an `open_application` key,
`triple_press_to_close: true` in the file, and pressing the key three times
quickly quits its application outright, as its own Quit command does, rather
than closing its window. The first press still opens the application, the
second does nothing, and each must come within half a second of the one
before. Three dots top left on the key show the count, one filled for each
press made, until the next press is no longer awaited. After the quit the key
does nothing until it has been left alone for
a second and a half, so a fourth press does not open the application again.
An application with unsaved work may still ask about it before it quits.

Windows has no one way to quit an application, so there the third press closes
the application's windows, each as its close button does. That quits most
applications, but one that keeps running in the notification area when its
window closes, as Signal and Discord can, keeps running. Not yet tried on a
Windows machine.

## World clocks

The world clocks are set on the editor's Dashboard section, under the preview
of the world clocks, or by hand in the file; the editor keeps them as written
when it saves anything else. Add `world_clocks` beside `pages`,
in `~/Library/Application Support/the-front-buttons/config.yaml` on macOS,
listing each clock's label and zone:

```yaml
schema_version: 1
world_clocks:
  - label: UTC
    zone: UTC
  - label: LDN
    zone: Europe/London
  - label: ARQ
    zone: America/Sao_Paulo
pages:
  # ...
```

There may be one to four clocks, a row each from the top, in the order
written; rows past the last are blank, and the world icon and the date stay
where they are. A label is one to four letters or digits, and no two clocks
share one. A zone is a name from the zone database, such as Europe/Berlin or
Asia/Tokyo, checked as the file loads; a city without a zone of its own takes
its region's, as Araraquara takes America/Sao_Paulo. Any clock that breaks
these refuses the file, as any other problem does. Leave `world_clocks` out
for the four above. A change shows the next time the world view opens.

## Mural

A `mural_share_link` key pastes a mural's link where you are typing, such as a
meeting's chat: a press puts its message, a space and the link on the
clipboard and pastes them, as Command V does on a Mac and Ctrl V on Windows.
Leave the message out to paste the link alone. The link must be one of Mural's
own, on `app.mural.co`, as the board's address bar or Mural's Share gives it.
The text is left on the clipboard, so you can paste it again. The paste is a
key press, so on macOS it needs the Accessibility permission key presses
already need.

```yaml
      - row: 1
        column: 0
        kind: mural_share_link
        parameters:
          link: https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d
          message: "Today's retro board:"
          label: Retro
```

The timer keys act on one board each, the one their link names, through
Mural's API, signed in once from the editor (below):
`mural_start_timer` starts the board's timer, for five minutes unless
`duration` says otherwise, up to 90 minutes, with its sound at the end unless
`silent` is ticked; `mural_add_minute` and `mural_add_half_minute` add a minute
and thirty seconds to it; `mural_stop_timer` stops it; and `mural_pause_timer`
pauses it while it runs and resumes it while it is paused. The link is the
board's own address, as its browser's address bar shows it; a key with no label
says what it does, such as Start 5:00, +0:30, Stop or Pause. The person signed
in must be able to run the board's timer, as its owner or a facilitator. A key
Mural refuses says why: not_set_up or not_connected for the sign in, not_found
for a board Mural does not know or the person cannot reach, refused for
anything else Mural turns down, or network; the pause key says no_timer when
the board has no timer running.

`mural_timer` is the timer on one key: it shows the board's timer as it runs,
in minutes and seconds, with an hourglass while it runs, a pause icon while it
is paused, and a red background in its last minute. A press starts the timer
when none runs, for `duration` (five minutes if left out) with its sound unless
`silent` is ticked, pauses it while it runs, and resumes it while it is paused.
It reads the board every ten seconds and counts down in between, so a timer
started or changed on the board itself shows within ten seconds, and a press
shows at once. Like the pause key, it needs the `murals:read` scope.

`mural_private_mode` runs the board's private mode, which hides everyone's
work from everyone else until it ends, for a silent brainstorm. The key shows
an open eye while private mode is off, and a closed eye on a purple key while
it is on; a press turns it on or off. Tick `hide_authors` to hide who wrote
what while it is on. It reads the board every ten seconds, so private mode
turned on or off on the board itself shows within ten seconds, and needs the
`murals:read` scope too.

```yaml
      - row: 2
        column: 0
        kind: mural_start_timer
        parameters:
          link: https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d
          duration: 10m
      - row: 2
        column: 1
        kind: mural_add_minute
        parameters:
          link: https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d
      - row: 2
        column: 2
        kind: mural_add_half_minute
        parameters:
          link: https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d
      - row: 2
        column: 3
        kind: mural_pause_timer
        parameters:
          link: https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d
      - row: 2
        column: 4
        kind: mural_stop_timer
        parameters:
          link: https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d
      - row: 2
        column: 5
        kind: mural_timer
        parameters:
          link: https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d
          duration: 10m
      - row: 2
        column: 6
        kind: mural_private_mode
        parameters:
          link: https://app.mural.co/t/team/m/team/1712345678901/0a1b2c3d
          hide_authors: true
```

### Signing in to Mural

The Mural keys use Mural's API, through an app of your own, signed in once
from the editor:

1. In Mural, choose your avatar, then **Create and manage apps**, and
   **New app**. Name it as you like. Give it the redirect URL the editor's
   credentials page shows, `http://127.0.0.1:48080/mural/callback`, and tick
   the scopes `murals:read` and `murals:write`: the keys read a board's state,
   such as whether its timer is paused, as well as change it. Mural shows the
   client secret once, so keep it to hand.
2. On the editor's credentials page, under **Mural**, enter the client ID and
   client secret and store them. They go to the Keychain, or Credential
   Manager on Windows, are sent only to app.mural.co, and are never shown again.
3. Choose **Connect to Mural**, and allow the app on Mural's page. Mural
   returns to the editor, which says it is connected.

The sign in lasts until Mural ends it; connect again if a key says it is not
connected. A sign in made before the keys read boards has only `murals:write`:
tick `murals:read` in the app as well, then connect again. If the editor had to use another port than 48080, the redirect URL
the credentials page shows changes with it, and the app's redirect URL must
match.

## Icons

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

## Credentials

Secrets, such as a GitHub token, never go in the configuration file. A key names
a credential (`credential: github-work`), and the secret is stored in the
Keychain on macOS or Credential Manager on Windows through the editor's
credentials page, which never shows it again. For GitHub, use a fine grained
token with read access to Actions on the repositories the keys watch.

## Platform bridge

How the daemon reaches the system's own libraries: through cgo, as it always
has, or through purego, which calls them from Go with no C compiler. Both are
built in while the two are compared; leave the setting out and nothing
changes. It is set by hand only, at the top level beside `pages`:

```yaml
schema_version: 1
platform_bridge: purego
pages:
  # ...
```

The value is `cgo` or `purego`, in lower case exactly; leaving the field out
means `cgo`. Anything else refuses the file, as any other problem does:
`unknown_bridge` for any other text, empty text included, and `wrong_type`
for a number, a list or anything that is not text. The editor does not show
the setting, but keeps it as written when it saves anything else, and writes
it only when the file has it. Its undo and redo refuse rather than overwrite a
hand edit, as for any change made by hand.

It is read once, when the daemon starts. A change made while it runs leaves
it on the bridge it started with until the next start; restart it to use the
new one. On macOS, restart the login agent with:

```bash
launchctl kickstart -k gui/$(id -u)/io.github.thescottbot.the-front-buttons
```

On Windows, stop it with `-stop` and start the task again, as
[SETUP.md](SETUP.md) describes.

The log says which way each run went, in three lines:

- `platform bridge`, once at start: `asked`, the bridge the file names (cgo
  when it names none), `platform`, the bridge the macOS adapters use, and
  `hid`, the deck's driver. It is a warning when either falls back:
  `platform_fell_back` when cgo is asked for in a build without cgo, which
  uses purego; `hid_fell_back` when the driver asked for is not in the build;
  `hid_unavailable` when no driver is. On Windows `platform` reads "unused on
  this system", as only the deck's driver takes the setting there.
- `platform bridge not read from the configuration file; using cgo`, a warning
  at start when the file could not be read or was refused. The daemon starts
  on cgo, and the file's problems are logged as usual.
- `platform bridge change takes effect at the next start`, with `running` and
  `file`, once when a saved file first names a different bridge from the one
  running. Changing it back says nothing.

What `purego` means today: every macOS adapter (the clipboard, quitting,
key presses and the Accessibility check, bringing an application forward,
Dock badges, window cycling and application icons) and the deck's driver
(IOKit on macOS, hid.dll on Windows) run through purego. Application icons
are drawn identically through both bridges, pixel for pixel. What acts on the
desktop through purego, such as pressing keys or raising windows, and the deck
through the purego driver, have not yet been tried, so `cgo` remains the
default and the safer choice.
