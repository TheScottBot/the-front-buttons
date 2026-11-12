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

### Known limitations

- Every protocol value is read from source and unconfirmed on a deck until
  experiment 1; the key order and the input report header in particular.
- No Roll20 preset until its launch URL is known (TD13).
- Saving from the editor drops hand written comments.
- Windows packaged applications launch through the Apps folder rather than COM
  (DV3).
