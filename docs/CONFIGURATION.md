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
tap on the marker, rather than a hold, still does nothing.

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
An application with unsaved work may still ask about it before it quits. Not
available on Windows yet.

## World clocks

The world clocks are set in the file only; the editor has no form for them,
but keeps them as written when it saves. Add `world_clocks` beside `pages`,
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
