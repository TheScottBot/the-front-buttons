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
  editor page, keeping its kind, parameters and preset.

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
