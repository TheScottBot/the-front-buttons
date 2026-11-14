# The Front Buttons Specification

Status: Draft
Applies: Engineering datum version 2, 20 September 2026
Optional modules applied: none. Module A is not applied because the project writes no C: hidapi arrives through cgo as a recorded dependency (datum 0.13), not as project code. Module B is not applied because there is one product. If the browser tab extension (excluded below) is built later, Module B is reconsidered then, because a daemon and an extension implementing one protocol is exactly its case.
Implementation log: `IMPLEMENTATION_LOG.md`
Deviation record: `IMPLEMENTATION_DEVIATIONS.md`
Identifiers: phases TE1 to TE10, decisions TD1 to TD9, findings TF1 onward

## What this specification is not

Read on 11 November 2026:

- the engineering datum, version 2
- Elgato SDK overview and plugin architecture pages, used only to rule out the plugin route
- the `sstallion/go-hid` and `matthewpi/streamdeck` package pages
- the python-elgato-streamdeck documentation index
- the Microsoft Q&A thread on the retirement of legacy third-party meeting and call control APIs (MC1266901)

Not read:

- any source implementing the Stream Deck XL protocol
- hidapi documentation
- the GitHub REST API documentation for Actions
- Microsoft Teams and Discord deep link documentation
- Windows and macOS documentation for opening URLs and launching applications
- macOS documentation on HID device access permissions
- the reference implementations named in the datum

Every protocol value, endpoint, URL format and permission in this document is a Plan stage question. The ones the design leans on are listed in Appendix A.

## What the subject requires beyond the datum

- The deck moves between two machines through a dock. Disconnection and reconnection are normal operation, not faults.
- Configuration is per machine, lives outside the binary, and is written both by hand and by the editor.
- The editor is a local web server that can cause applications and URLs to open. Requests reaching it from the browser are untrusted (datum 0.10), including requests forged by other web pages.
- Tokens, such as the GitHub token, are secrets and never touch the config file, the logs or the editor's responses.

## 1. Product

This is for the lions living in the wiry frames of my peripherals' bodies

The Front Buttons is a Go daemon that drives an Elgato Stream Deck XL directly over USB HID, with no Elgato software, on a MacBook (macOS, Apple silicon) and a Dell work laptop (Windows, amd64). Each key is an instance of a building block, called a brick, placed by per machine configuration. Configuration is composed by hand or in a local web editor.

In scope:

- the device protocol for the Stream Deck XL
- surviving connect and disconnect as the dock switches machines
- pages, with reserved navigation keys
- rendering key images
- the brick framework and its schema mechanism
- these bricks: `open_url`, `open_application` and `pipeline_status` (GitHub Actions)
- presets over `open_url` for Discord channels, Teams chats and channels, D&D Beyond and Roll20
- live state
- the local web editor
- credential storage
- start at login
- documentation

Out of scope:

- focusing an existing browser tab (a later browser extension; until then `open_url` and installed site app windows cover it)
- Teams mute and meeting controls (TF2)
- other Stream Deck models (the model table admits them later)
- CI providers other than GitHub Actions (the provider port admits them later)
- reaching the editor from another machine
- syncing configuration between machines

### Settled by the author

| Item | Settlement |
|---|---|
| Device access | Direct USB HID. Elgato software is not used |
| Language | Go. cgo is acceptable on both machines |
| Browser tabs | `open_url` first; tab focusing is a later phase |
| CI | GitHub Actions |
| Configuration | Per machine, outside the binary |
| Editor | In scope, running locally |
| Navigation | The bottom left key pages back and the bottom right key pages forward, on every page. Paging wraps around |
| Live state | Live bricks keep updating while their page is not shown. Switching pages shows current state immediately |
| Name | The Front Buttons |
| Description | The first line of the product description and of `README.md` is exactly: This is for the lions living in the wiry frames of my peripherals' bodies |

## 2. Design

Normative. The rule throughout is: **declare once, derive everywhere.** A brick's parameters, a deck model's geometry and a preset's fields are each declared in one place. Validation, decoding, editor forms, documentation and rendering all derive from that one declaration.

### Package layout and dependency direction

```
cmd/the-front-buttons/      composition root only: wires adapters into the core
internal/deck/              model table, pure protocol encode and decode, DeckDevice port
internal/deck/hidtransport/ hidapi adapter (the only cgo package)
internal/layout/            pages, positions, reserved navigation, wrap around
internal/render/            appearance to image; the only drawing code
internal/brick/             kind registry, schema derivation, brick interfaces
internal/bricks/            one package per brick kind
internal/presets/           preset table over brick kinds
internal/live/              live state hub, shared pollers, appearance cache
internal/config/            load, validate, migrate, watch, atomic write
internal/platform/          OS ports and their build tagged adapters
internal/githubactions/     GitHub Actions client, behind the pipeline provider port
internal/editor/            local HTTP server, handlers, templates
internal/diagnostics/       counters and structured logging setup
```

- **MUST:** `cmd/` is the only place adapters are constructed and injected. Brick packages depend on port interfaces, never on adapters.
- **MUST:** `runtime.GOOS` checks and OS specific imports appear only in `internal/platform/` and `internal/deck/hidtransport/`, as build tagged files (`_darwin.go`, `_windows.go`). Elsewhere a CI check rejects them (TE1).
- **MUST:** `internal/render/` is the only drawing code. The device and the editor preview both show the same `image.Image` it produces, each encoded for its target.
- **MUST:** layering is enforced by an import rule check in CI, proven against a planted violation (TE1).

### Brick framework

This is the heart of the design, and the place the anti duplication rules matter most.

- **MUST:** each brick kind declares its parameters exactly once, as a Go struct with field tags carrying name, requiredness, bounds, description and a secret flag. The schema is derived from that struct by reflection, and the same derived schema drives:
  - config validation, with unknown fields rejected (datum 0.11)
  - decoding into the typed struct
  - editor forms
  - the generated brick reference in `README.md`

  No brick writes its own validator, form or documentation.
- **MUST:** appearance parameters common to every brick (label, icon, background) are one embedded struct, declared once.
- **MUST:** interfaces stay small (interface segregation). A brick implements only what it uses:
  - `Pressable`: does something when pressed
  - `LiveSource`: publishes appearances over time until its context is cancelled
- **MUST:** kinds register through one generic registration function that takes the parameter type and a constructor receiving decoded parameters and injected ports. Adding a kind means adding one package and one registration line, and changing nothing else.
- **MUST:** presets are data, not code. A preset row names a target brick kind, a set of friendly fields (declared with the same tag mechanism) and a template expanding them into the target's parameters. Configuration stores the preset name and its fields, so the editor can re-edit them. Expansion happens at load.
- **MUST:** a brick in error shows that on its key, with an error glyph and a short reason category, never failing silently (datum 0.15).

### Bricks in scope

| Kind | Pressable | Live | Parameters (illustrative, settled in TE7 and TE8) |
|---|---|---|---|
| `open_url` | Yes | No | URL, plus appearance |
| `open_application` | Yes | No | Application identifier, interpreted by the OS adapter (bundle identifier or name on macOS; executable path or application user model ID on Windows), plus appearance |
| `pipeline_status` | Yes: opens the latest failing run if the latest run failed, otherwise the latest run | Yes: success, failure, in progress, queued, cancelled, unknown | Repository, workflow, branch, credential name, poll interval, API base URL (TD7), plus appearance |

| Preset | Over | Friendly fields |
|---|---|---|
| Discord channel | `open_url` | Server ID, channel ID |
| Teams chat | `open_url` | Participants or chat ID, per Plan |
| Teams channel | `open_url` | Per Plan |
| D&D Beyond character | `open_url` | Character ID |
| Roll20 game | `open_url` | Game ID |

### Ports

| Port | Responsibility | macOS adapter | Windows adapter |
|---|---|---|---|
| `DeckDevice` | Key events in; key images, brightness and reset out | hidtransport | hidtransport |
| `DeckWatcher` | Reports connect and disconnect of supported decks | hidtransport enumeration | hidtransport enumeration |
| `URLOpener` | Opens a URL or deep link with the system handler | Per Plan | Per Plan |
| `ApplicationLauncher` | Launches or raises an application | Per Plan | Per Plan |
| `CredentialStore` | Reads and writes named secrets | Keychain | Credential Manager |
| `ConfigLocation` | Gives the per machine config and state paths | Per TD1 | Per TD1 |
| `PipelineProvider` | Returns the latest run state and its URL for a workflow | `githubactions`, the same on both OSes | Same |

Each port has one contract test suite. Every adapter and every test double runs that same suite, so doubles cannot drift from real behaviour. Fixtures and doubles live once, in a shared test support package.

### Deck model and protocol

- **MUST:** model facts are rows in one table: vendor and product IDs, grid, key image size, orientation, image format, report sizes and command bytes. The protocol package encodes and decodes using that table and has no per model branches. XL variants with different product IDs are separate rows.
- **MUST:** encode and decode are pure functions over bytes, testable without a device. The transport only moves bytes.
- **MUST:** the input report parser is bounded and fuzzed (datum 0.10).

### Layout

- **MUST:** positions are row and column, matching the XL's 4 by 8 grid from the model table. The bottom left and bottom right positions are reserved on every page; configuration assigning them is rejected with a clear reason.
- **MUST:** navigation keys show the page name or number so the current page is never ambiguous. Paging wraps from last to first and back.
- **MUST:** an empty position renders blank, not an error.

### Live state

- **MUST:** the live hub runs every live brick in configuration, on every page, for as long as the deck is configured. It holds the latest appearance per key. A page switch pushes cached images without waiting on any brick.
- **MUST:** live bricks watching the same source share one poller. Two keys watching one workflow make one request per interval.
- **MUST:** on config reload, only bricks whose definition changed are restarted. Unchanged keys keep their state.
- **MUST:** with no deck connected, live bricks keep running, so the deck shows current state the instant it reconnects. GUIDANCE: if battery or rate limits make this costly, pausing polling while disconnected can be raised as a deviation, not decided silently.

### Configuration

- **MUST:** one file per machine at the `ConfigLocation` path, carrying a schema version (datum 0.11).
- **MUST:** unknown fields and unknown kinds are rejected.
- **MUST:** an invalid file never replaces the running configuration. The last good configuration keeps running, and the rejection is logged and shown in the editor.
- **MUST:** editor writes are atomic (write a temporary file, then rename), and the file watcher treats the editor's own write like any other change, so there is one reload path.
- **MUST NOT:** secrets appear in the file. Bricks reference credentials by name.

### Editor

- **MUST:** it listens on the loopback interface only. It rejects requests whose Host is not the loopback address and port it bound to (DNS rebinding) and whose Origin is not its own. It requires a per launch token for every state changing request, because any web page open in the same browser can otherwise send requests to it.
- **MUST:** palette, forms and preset forms come from the brick registry and the preset table. The editor holds no list of kinds or fields of its own.
- **MUST:** previews come from `internal/render`.
- **MUST:** secrets are write only. The editor can set a named credential, but never reads one back or shows it.
- **MUST:** it meets datum 0.16 in full. The key grid must be operable from the keyboard, with a keyboard alternative to any drag and drop.

## 3. Decisions

Asked in the datum's five part form during Plan. Recommendations are labelled as such.

| ID | Decision | Options | Recommendation | Blocks |
|---|---|---|---|---|
| TD1 | Identifier spellings derived from the name | Module path; binary; config folder; credential service name; editor title | Display name The Front Buttons everywhere a human reads it; `the-front-buttons` for the binary and credential service; module path chosen by the author | TE1 |
| TD2 | Licence | Any compatible with hidapi and the Go HID wrapper | Read both licences in Plan first, then choose | TE10 |
| TD3 | HID library | `sstallion/go-hid`; `karalabe/hid` (bundles hidapi source); another found in Plan | Choose after Plan records maintenance, whether hidapi is bundled, hotplug support and behaviour on both OSes | TE6 |
| TD4 | Config file format | YAML; TOML; JSON | YAML: it reads well for hand editing pages of keys before the editor exists, at the cost of one dependency recorded under datum 0.13 | TE2 |
| TD5 | Icons | User supplied image files only; plus a bundled permissively licensed set | User files plus one bundled set whose licence is read and recorded in Plan, so the editor has a usable palette from day one | TE4 |
| TD6 | Editor front end | Go templates with htmx; a TypeScript single page app | Go templates with htmx, vendored with a recorded digest. Forms are generated server side from the one schema, so no second copy of brick types exists in TypeScript, and almost no JavaScript is written | TE9 |
| TD7 | GitHub host | github.com only; also a configurable API base URL for GitHub Enterprise | A configurable base URL defaulting to github.com, since the work repositories may sit on an Enterprise host | TE8 |
| TD8 | Polling | Default and minimum interval for `pipeline_status` | Settle after Plan reads GitHub's rate limits and conditional request rules. Values go in Appendix A until then | TE8 |
| TD9 | Opening the editor | Printed URL only; an `open_editor` brick; a tray icon | Printed URL plus an `open_editor` brick, so the deck opens its own editor. No tray icon, which would add a dependency for little gain | TE9 |

## 4. Plan

Produces `docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md`, as the datum requires.

**Read:**

- python-elgato-streamdeck source for the XL device class and transport, as the protocol reference; record the commit read
- the candidate Go HID libraries and their bundled or required hidapi version
- the GitHub REST endpoints for workflow runs, authentication with fine grained tokens, rate limits and conditional requests
- Teams and Discord deep link formats
- macOS and Windows mechanisms for opening URLs and launching applications, including how new Teams is launched on Windows
- macOS requirements, if any, for opening a non keyboard HID device
- the chosen credential store mechanism on each OS
- the reference implementations

**Experiments, on both machines:**

1. Open the XL through each candidate HID library, read key reports, write one key image, then unplug and replug and record enumeration behaviour.
2. Switch the dock between machines with the deck attached, and record what each OS reports and how long it takes.
3. Open each preset's deep link from a shell on each OS and record what opens.
4. Build with cgo on each machine, and on GitHub Actions runners for macOS arm64 and Windows amd64.

**Exit:** the datum's Plan exit, with TD1 to TD9 answered or explicitly open.

## 5. Execute

Each phase carries the datum's five headings.

### TE1: repository and checks

- **Work:**
  - Go module and layout
  - golangci-lint at its strictest, including the duplicate code linter, with warnings as errors
  - the import layering check
  - the OS check (no `runtime.GOOS` or OS imports outside the permitted packages)
  - the prose and typography scan
  - a check that the first line of `README.md` is the settled description line
  - GitHub Actions running all of this on macOS arm64 and Windows amd64
- **Excluded:** behaviour.
- **Assumptions to verify:** runner images provide a C toolchain for cgo (experiment 4).
- **Tests first:** every check fails on a planted violation before passing on the real tree. The planted violations are: a duplicated block across two brick packages; a brick importing an adapter; `runtime.GOOS` in `internal/live`; an en dash; and a changed README first line.
- **Done when:** every planted failure is recorded, CI is green on both runners, and the author reproduces this on both machines.

### TE2: config and the brick schema mechanism

- **Work:** schema derivation from tagged structs; validation; decoding; schema version and migration hook; unknown field rejection; preset table and expansion; atomic write; file watching with last good retention.
- **Excluded:** concrete brick behaviour.
- **Assumptions to verify:** TD4.
- **Tests first:**
  - a test brick kind whose struct round trips through schema, validation and decoding
  - every tag rule enforced
  - unknown field and unknown kind rejection
  - reserved position rejection
  - preset expansion
  - an invalid file leaving the previous configuration running
  - atomic write surviving an interrupted write
  - a fuzz harness on the config decoder
- **Done when:** green, with the fuzz run recorded.

### TE3: layout

- **Work:** pages, positions, reserved navigation, wrap around, page indication.
- **Excluded:** drawing.
- **Assumptions to verify:** XL grid orientation and key index order (Plan reading).
- **Tests first:** position to key index mapping for the whole grid; wrap in both directions; single page behaviour; navigation keys present on every page.
- **Done when:** green.

### TE4: rendering

- **Work:** appearance to image, covering label, icon, background, status glyphs, error state and navigation labels; font choice recorded under datum 0.13; icon loading per TD5.
- **Excluded:** encoding for the device.
- **Assumptions to verify:** key image size from the model table.
- **Tests first:**
  - golden image tests for each status, an error, a long label and a missing icon
  - a test proving every status is distinguishable without colour, by glyph or text
- **Done when:** green, plus an author gate: legibility on the physical deck at arm's length in the dock, under normal desk lighting, on both machines.

### TE5: deck protocol

- **Work:** the model table rows for the XL variants; pure encode for images (including chunking and orientation), brightness and reset; pure decode for key state reports.
- **Excluded:** transport.
- **Assumptions to verify:** every value in Appendix A marked protocol.
- **Tests first:**
  - byte level fixtures derived from the recorded reference source for each command
  - decode of every key in pressed and released states
  - malformed, short and oversized reports rejected
  - a fuzz harness on decode
- **Done when:** green, with the fuzz run recorded.

### TE6: transport, watcher and daemon loop

- **Work:** the hidtransport adapter (TD3); deck watching; the daemon main loop connecting events, layout, live cache and rendering; full repaint on connect; a clear, logged state when the deck is busy (for example, held by Elgato software).
- **Excluded:** bricks beyond a test brick.
- **Assumptions to verify:** experiments 1 and 2.
- **Tests first:**
  - the adapter and a double both pass the `DeckDevice` and `DeckWatcher` contract suites
  - the loop, driven by doubles, repaints on connect, survives disconnect mid write, and pages correctly
- **Done when:** green, plus an author gate: on both machines, pressing keys, paging both ways, and switching the dock back and forth ten times without restarting the daemon.

### TE7: platform ports and simple bricks

- **Work:** `URLOpener`, `ApplicationLauncher` and `ConfigLocation` adapters for both OSes; the `open_url` and `open_application` bricks; the presets.
- **Excluded:** live bricks.
- **Assumptions to verify:** experiment 3; the URL opening and launching mechanisms read in Plan.
- **Tests first:**
  - each adapter passes its port contract suite, using a test double for process launching so tests run nothing real
  - bricks are tested against port doubles
  - preset expansion tested per preset
- **Done when:** green, plus an author gate: every preset and both bricks working on each machine.

### TE8: live state and `pipeline_status`

- **Work:** the live hub with shared pollers and the appearance cache; the `CredentialStore` adapters; the GitHub Actions client using conditional requests; `pipeline_status`, with press opening the right run.
- **Excluded:** other providers.
- **Assumptions to verify:** GitHub endpoints and rate limit behaviour (Plan reading); TD7; TD8.
- **Tests first:**
  - the client tested against a local fake server for success, failure, in progress, 304, rate limited and unauthorised
  - two keys on one workflow produce one request per interval
  - state updates while off page and is shown instantly on page switch
  - reload restarts only changed bricks
  - no token appears in any log line, checked by log capture
- **Done when:** green, plus an author gate: a real workflow turning red then green on the deck at work, with a press opening the failing run.

### TE9: editor

- **Work:**
  - loopback server with the protections in section 2
  - page and key grid
  - palette and forms from the registry and presets
  - previews from the renderer
  - write only credential entry
  - save through the atomic write
  - validation errors shown at the field
  - diagnostics view
  - the `open_editor` brick (TD9)
- **Excluded:** remote access.
- **Assumptions to verify:** TD6.
- **Tests first:**
  - rejections for wrong Host, wrong Origin and missing token
  - generated forms match the schema for every registered kind, so adding a test kind adds its form with no editor change
  - a saved configuration round trips
  - accessibility checks for keyboard operation, focus, labels and announcements
- **Done when:** green, plus an author gate: building a full page from scratch in the editor on each machine, including once using the keyboard only.

### TE10: running, installing and documents

- **Work:**
  - optional install scripts: a LaunchAgent on macOS and a scheduled task at logon on Windows, neither needing administrator rights
  - plain `go run` documented as always valid
  - datum Part 2 documents: README (first line settled, brick reference generated from the registry), TESTING, CHANGELOG, LICENSE (TD2), SECURITY, and PRIVACY (Teams deep links can hold colleagues' addresses in local config)
- **Excluded:** packaging for distribution to others.
- **Assumptions to verify:** login mechanisms (Plan reading).
- **Tests first:** the generated brick reference is checked in CI against the registry, failing on drift; install scripts are seen to fail on a planted invalid path.
- **Done when:** green, plus an author gate: the daemon starts at login on both machines and the deck is live without opening a terminal.

## 6. Verify

- A reproduction report per phase, as datum Stage three requires, from both machines where hardware is involved
- The duplicate code linter reports nothing at release
- Every port's contract suite passes for every adapter and double
- Every phase shows Complete in `IMPLEMENTATION_LOG.md`
- TD1 to TD9 resolved or deferred with the author's agreement

## 7. Observability

`log/slog` structured output to stderr and a rotating file in the state folder. The diagnostics view in the editor reads the same counters:

- key presses, by brick kind
- brick errors, by kind and reason category
- pipeline polls, by result (including 304 and rate limited), with remaining rate limit
- deck connect, disconnect and busy events
- render failures
- config reloads, accepted or rejected with reason
- editor requests rejected, by reason

Startup states the version, config path, deck model if connected, registered brick kinds, and any configured credential that is missing.

Logs carry brick kind, page and position, never URLs (they can contain chat participants), tokens or request bodies.

## 8. Security and privacy

- **Trust boundaries:**
  - USB input reports
  - the config file (it is the user's own, but it is parsed as untrusted)
  - editor requests
  - GitHub API responses
- **The editor** is covered in section 2. The key risk is a hostile web page in the same browser trying to rewrite configuration so a key opens something harmful. Host, Origin and the per launch token checks exist for that reason.
- **Secrets** are in the OS credential store only. The GitHub token is a fine grained token with read access to Actions on the chosen repositories.
- **The work machine** runs a background process that listens on a loopback port. SECURITY.md states this plainly, for the author to point IT at.
- **Personal data:** none is collected. Local config may hold colleagues' identifiers inside Teams deep links; PRIVACY.md says so.

## Findings

| ID | Finding | Owner |
|---|---|---|
| TF1 | The Elgato SDK documentation currently describes only Node.js plugin backends | Informs the route; no action |
| TF2 | Microsoft has announced the retirement of legacy third party meeting and call control APIs (MC1266901), which cover mute, camera and similar controls. No date or replacement was found | Teams mute stays out of scope until a supported API is identified |
| TF3 | The Go Stream Deck library reviewed (`matthewpi/streamdeck`) is Linux only, so a cross platform HID layer is needed | TD3 |
| TF4 | Regular Firefox requires signed extensions, and Edge needs developer mode or a store listing | Recorded for the later tab focusing extension |

## Appendix A: unverified values

| Value | Area | Source needed | Verified |
|---|---|---|---|
| XL vendor and product IDs, including variants | Protocol | Reference source | No |
| Key image size, format and orientation | Protocol | Reference source | No |
| Report sizes, command bytes and image chunk layout | Protocol | Reference source | No |
| Key index order across the 4 by 8 grid | Protocol | Reference source, experiment 1 | No |
| macOS permission requirements for HID access | Platform | Plan reading, experiment 1 | No |
| Windows and macOS URL opening and application launch mechanisms | Platform | Plan reading | No |
| New Teams launch identifier on Windows | Platform | Experiment 3 | No |
| Teams and Discord deep link formats | Presets | Plan reading, experiment 3 | No |
| GitHub workflow run endpoint, token scopes, rate limits, conditional request accounting | Pipeline | GitHub REST documentation | No |
| Default and minimum poll interval | Pipeline | TD8 | No |
| Credential store mechanism per OS | Platform | Plan reading | No |
| Config and state folder paths per OS | Platform | TD1, Plan reading | No |
| Duplicate code linter threshold | Checks | Author, TE1 | No |
| Toolchain, golangci-lint and dependency versions | Tooling | Plan, datum 0.13 | No |
| Reference implementations exist at the exact paths in the datum | Plan | Plan | No |
