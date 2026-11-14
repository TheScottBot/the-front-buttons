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

Rows count from 0 at the top and columns from 0 at the left, on the XL's 4 by 8
grid. Row 3, columns 0 and 7 are the navigation keys and cannot be assigned. A
key names exactly one of `kind` with `parameters`, or `preset` with `fields`.
Quote values YAML would read as numbers or booleans, such as IDs. Saving from
the editor rewrites the file and drops hand written comments.

## App screens

An app screen is a page of one application's keys. Tick App screen on an
`open_application` key: a tap still just opens the application, and holding the
key for half a second opens it and shows that page as well. The editor creates
the page the first time. The application's
icon is top left on its screen, and pressing it goes back to the regular page.
Paging never reaches an app screen, and back and forward leave one.

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
