# Changelog

All notable changes to this project are documented here. The format follows
Keep a Changelog. Each tag will name the contract digest it was built against
(none yet: the configuration contract is provisional), every change to shared
code since the previous tag, and which author gates were cleared at which commit.

## Unreleased

Built ahead of the hardware (deviation DV1). No author gate has been cleared.

### Added

- Repository checks: prose and typography, operating system use, the README's
  first line and its brick reference; golangci-lint with every linter, import
  layering and formatting; duplicate code across packages; CI on macOS arm64 and
  Windows amd64 (TE1).
- The brick schema derived from tagged parameter structs, presets as data, and
  the configuration file: decoded strictly, versioned, written atomically,
  watched, with the last good configuration kept (TE2).
- Key numbering, wrap around paging, page indication and page layout (TE3).
- The key renderer, status glyphs told apart without colour, user PNG icons and
  the bundled Tabler icon font (TE4).
- The Stream Deck XL model table and protocol codec, fuzzed (TE5).
- The HID transport over go-hid, deck watching across dock switches, and the
  daemon loop (TE6).
- The `open_url` and `open_application` bricks, presets for Discord, Teams and
  D&D Beyond, the operating system adapters and the composition root (TE7).
- The live hub, the GitHub Actions client, the `pipeline_status` brick and the
  credential store (TE8).
- The local editor and the `open_editor` brick (TE9).
- Start at login for both systems without administrator rights, the rotating log
  file, the generated brick reference and the remaining documents (TE10).

### Changed

- The editor is restyled after the author's sites, punkdev.fyi and
  cryptid.wine: a hot pink page, monospace text, misprinted lowercase
  headings, black tags, heavy rules and slab buttons. Every colour pair still
  meets WCAG AA (the lowest is 6.1 to 1) and focus is a thick ink and yellow
  ring. A key's kind now shows on its own line under its label in the grid.

- The editor now stays at <http://127.0.0.1:48080/> across restarts, falling
  back to any free port, with a warning, only if another program holds 48080;
  `-editor-port` picks another port (the author's decision, reversing TE9's new
  port at every launch).

- The editor has an Icons page: every bundled icon, drawn as on a key, with a
  search, linked from each key's Icon field.
- A key can show a PNG of the person's own: `icon_file` names a file in the
  `icons` folder beside the configuration (TD5's user icons, never before
  connected to a key).

- A configured key can be moved or copied to a free tile on any page from its
  editor page, keeping its kind, parameters and preset. The destination is one
  list of free tiles only, so paging and occupied tiles are never offered.

- Keys can be dragged onto a free tile in the editor's grid to move them, or
  copied by holding Option (Alt) while dropping. The script is TypeScript,
  built by esbuild pinned as a Go tool; a check fails if the served script
  drifts from it.

- Clicking a key in the editor's grid edits it in a popup over the grid
  rather than on a page of its own; the key's page still works without the
  script.

- Preset keys are named by their action in the editor (`open_server`,
  `open_character`, `open_chat`, `open_channel`) rather than by `open_url`.

- An app picker on the Application field lists the installed applications by
  name on macOS and Windows and fills in the identifier (Windows not yet run
  on a machine).
- The Icons page pages through all the icons, 120 at a time, keeping the
  search.

- A `press_key` brick sends one momentary press of an extended function key,
  F13 to F24 (macOS has F13 to F20 only, and asks for Accessibility
  permission).

- `open_application` keys can draw the application's own icon on a key
  coloured to match its edge (macOS; Windows not yet), and ticking it sets the
  Icon fields aside.
- The press_key key is chosen from a list of F13 to F24.
- On macOS the daemon restarts itself once Accessibility is granted, so a new
  permission takes effect without a manual restart; the install script signs
  the binary as io.github.thescottbot.the-front-buttons.

- Dragging a key shows a bin, out of the way of the tiles; dropping a key on
  it removes it, and the page offers Undo.

- Every key is framed in black, so the XL's slight offset and bowed edges no
  longer show as a strip on light keys. Each key sits in it as a rounded tile
  with smooth, blended edges, bulged slightly so they look straight through
  the key caps.

- The editor follows the system into a dark mode in the colours of the author's
  Thyme clock: its near black panel, pink text and pink outlines, with the tilts
  straightened. Every rule names a colour role, and each role pair meets WCAG AA
  contrast in both palettes.

- A switch in the editor's header chooses light or dark whatever the system's
  setting. The choice is kept in a cookie the editor reads, so each page is
  drawn in it from the start without a flash of the other palette.

- Two keys that change the deck itself. `brightness` toggles the deck between
  its usual brightness and a set percentage, which it shows unless labelled.
  `deck_off` turns every other key off and the deck down low (20 percent unless
  set); pressing it, or any key, turns them back on without doing anything else.
  The chosen brightness survives the deck being unplugged and returned.
- A bounded number in the editor, such as a brightness level, can be set with a
  slider as well as typed; the two stay in step. A new key's form starts at the
  number's default: 50 percent for brightness, 20 for off.
- A page's keys can be copied to a new page, named as given, or onto another
  page beside its keys. A copy that would land on a key copies nothing and
  names the clashing tiles.
- A `page_marker` key marks a page and does nothing when pressed; without a
  label it shows its page's number. Copying a page leaves it out, so the page
  copied onto keeps its own.
- A brick drawer in the pages screen's left gutter opens on a click and offers
  every brick and preset to drag. Dropped on an empty tile, a brick opens that
  tile's key editor with it chosen; while one is carried, keys and paging tiles
  are dimmed and crossed, as only an empty tile takes one.
- The page copy forms sit in a drawer, closed until opened, with a chevron.
- The brick drawer holds a drawer for each size: 1x1 with every brick and
  preset, and 1x2 and 1x3 with keys placed down a column at once, a D&D Beyond
  sheet and a Roll20 campaign, and a Discord channel for the 1x3. Carried over
  the grid, a group numbers the free tiles it would fill; dropped, it opens one
  form for every part's details, whose parts can be reordered.
- A `roll20_campaign` preset opens a Roll20 campaign by its ID.
- Four Spotify keys, and a 2x2 in the drawer that places them at once:
  `spotify_album` shows the current track's art and brings Spotify forward,
  `spotify_play_pause` plays or pauses and shows which a press will do, and
  `spotify_previous` and `spotify_next` change track. On macOS they drive
  Spotify through AppleScript, which asks once for the Automation permission;
  on Windows they say media control is unavailable.
- Groups in the drawer may be any width and height, filled left to right, then
  top to bottom. A 4x1 places the four Spotify keys in a row.
- The editor grid's back and forward tiles page as the deck's do, wrapping
  round at either end, each named for the page it goes to.
- `jetbrains_next_window` and `jetbrains_previous_window` bring forward the
  next or previous window across every running JetBrains IDE, GoLand, PyCharm
  and Rider among them, in a stable order, wrapping round; a 2x1 in the drawer
  places both. They use the Accessibility permission F13 already needs, and a
  refused press asks for it and restarts the daemon once it is granted, as
  F13's does; on Windows they say moving between windows is unavailable.
- A `press_shortcut` key sends a key with any combination of modifiers held,
  such as ⌘C or Ctrl+C. In the editor the key is chosen from a list and each
  modifier is a chip, named as the Mac or Windows names it, with a switch
  between the two; each machine names them its own way first. A key without a
  label shows its shortcut.
- App screens: an `open_application` key with App screen ticked also shows
  that application's own page, with the application's icon top left to go
  back. A page is an app screen when it names an `application`; there is one
  per application, paging never reaches one, and back and forward leave it. The
  editor lists app screens under the regular pages, numbers the regular pages
  alone, and creates an application's screen the first time a key asks for it.
- An application's own bricks, offered only on its app screen, starting with
  GoLand: about fifty actions, among them run, debug, build, the debugger's
  steps, navigation, refactoring, Git and the tool windows, each pressing the
  shortcut of the default keymap for the platform it runs on. The drawer has a
  GoLand section on GoLand's screen, with a 2x2 of run, debug, stop and rerun
  and a 4x1 of the debugger's steps. GoLand EAP shares them.
- A key that sends key presses on an app screen brings the application to the
  front first, unless it is there already, so the presses reach it; if it is
  not open, nothing is sent and the key says so. Windows does not bring it
  forward yet.
- A press taking a quarter of a second or more is logged as a slow key press,
  with how long it took.
- Every JetBrains IDE has the app screen keys GoLand had: IntelliJ IDEA,
  PyCharm, GoLand, WebStorm, PhpStorm, Rider, CLion, RubyMine, DataGrip,
  DataSpell, RustRover, Aqua and Writerside, and Android Studio, built on the
  same platform. They share one set of keys, renamed from `goland_` to
  `jetbrains_`, such as `jetbrains_build`; each IDE has its own screen, named
  for itself. A configuration using the `goland_` names needs them changed.
- An `open_application` key with Show notifications ticked shows the badge its
  application has on its Dock icon, top right: a red pill with a count, or a red
  dot for a badge without one, such as Discord's. The Dock is read every two
  seconds while a key shows a badge, and not at all otherwise; the application
  is never asked. Windows does not show badges yet.

### Fixed

- Security: Go 1.27.4, fixing GO-2026-6859, a crash in `net/http` the editor's
  request size limit could reach. A GitHub token is sent only to the host it
  was stored for, `api.github.com` unless an Enterprise host is given on the
  credentials page, whatever a configuration says. Every editor page needs a
  session, given by a one time login link from the deck's editor key or
  `-open-editor`, so other accounts on the machine cannot use the editor.
  SECURITY.md now states what the daemon does and does not do, citing the code,
  and a test fails if the code runs a program it does not list.

- The macOS install script copies the binary to
  `~/Library/Application Support/the-front-buttons/bin` and runs the login agent
  from there, so the agent no longer depends on where the binary was built,
  which may be a temporary folder cleared on restart. It signs with the code
  signing certificate "The Front Buttons Signing" when the keychain has one, so
  the Accessibility permission survives reinstalls; otherwise ad hoc, as before.
  The uninstall script removes the copy too.

- CI on Windows: the golden image comparison allows a few shades of rounding,
  as the regenerated icon goldens, made on an arm64 Mac, differed by four
  pixels on the amd64 Windows runner. A failure now says by how much.

- Icons and labels are centred on the key: icons sat up to a pixel and a half
  left of centre on the XL, and labels up to a pixel, which showed on the off
  key's power icon.
- The Spotify album and play keys showed "failed": the script reading Spotify
  set a variable named after one of Spotify's own properties. Each read is now
  also bounded in time.

- Two watchers initialising hidapi at once aborted the process; hidapi is now
  initialised once per process, behind one lock (first run on a real deck).
- Paging took seconds and keys drew one at a time: every write waited out the
  key read in progress, and macOS's occasional general IOKit error on a write was
  taken for an unplugging. Reads no longer wait holding the device's lock, and a
  write is tried three times before it counts as failed. A full page draws in
  about 33 milliseconds.
- Every plain save from the editor was refused as Forbidden: its no-referrer
  policy made browsers send Origin: null. The policy is now same-origin.
- Reinstalling the macOS login agent could fail with launchd error 5; the
  script now waits for the old copy to stop.
- The Discord preset opened the browser; it now emits Discord's discord:// app
  link, and `open_url` accepts that one app scheme besides https (the author's
  decision amending TD13).
- The editor's grid caption read "Page 0 plus one of 1"; it now counts from
  one.
- A refused save in the editor now starts its title with "Error:" and focuses
  the first field in error.
- A press error on a live key, such as pressing before any run was known, hid
  the newer run state; a newer publish now replaces it.
- The pipeline key now opens the newest run, the one it shows, shows a failure
  as soon as any job fails, and polls every 15 seconds while a run is active
  (the author's decisions, amending TD10).
- CI runs every step under Bash, so Windows no longer splits arguments at dots,
  and the Windows only code passes lint.
- Icons sat near the top of their keys; an icon and label without a status word
  are now centred together, and an icon alone is centred.

### Known limitations

- The protocol is confirmed on the MacBook only; the Dell is still owed.
- No Roll20 preset until its launch URL is known (TD13).
- Saving from the editor drops hand written comments.
- Three GoLand shortcuts are not yet confirmed in GoLand: rerun on macOS
  (⌘R), run tests on macOS (⌃⇧R) and optimise imports (⌃⌥O).
- Windows packaged applications launch through the Apps folder rather than COM
  (DV3).
