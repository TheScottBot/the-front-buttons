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

### Fixed

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
- Windows packaged applications launch through the Apps folder rather than COM
  (DV3).
