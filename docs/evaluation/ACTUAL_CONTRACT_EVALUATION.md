# Actual contract evaluation

The Plan stage record for The Front Buttons, as the engineering datum (version 2,
20 September 2026), Stage one, requires. Phase findings and reproduction reports
are appended below as phases complete.

This record is written under deviation DV1 (`IMPLEMENTATION_DEVIATIONS.md`): the
Stream Deck XL is not yet in hand, so hardware values are read from open source
implementations and marked **read from source, not confirmed on hardware** until
the experiments in specification section 4 are run.

Every claim below says how it was obtained:

- **Read**: the agent read the source file or official page itself.
- **Read by delegate**: a subagent fetched the official page on the date given
  and quoted it; the fetch tool summarises pages, so quotes are near verbatim and
  each load bearing one is re checked before the phase that relies on it.
- **Run**: the agent ran it for real, with the command recorded.
- **Not done**: named, with the reason.

## 1. Environment

Read on 11 November 2026, on the author's MacBook.

| Item | Value | How |
|---|---|---|
| Operating system | macOS 27.2, build 26B7124 | Run: `sw_vers` |
| Architecture | arm64 | Run: `uname -m` |
| Go | go1.27.3 darwin/arm64, Homebrew | Run: `go version` |
| cgo | `CGO_ENABLED=1`, `CC=cc` | Run: `go env` |
| C compiler | Apple clang 21.0.0 (clang-2100.4.1.1), Command Line Tools | Run: `clang --version` |
| golangci-lint | Not installed | Run: `which golangci-lint` |
| hidapi system library | Not installed (not needed if the Go wrapper bundles it, see 4) | Run: `brew list --versions hidapi` |
| GitHub CLI | 2.103.3 | Run: `gh --version` |

The Dell (Windows, amd64) has not been examined. Its toolchain is recorded when
the author runs the Windows side.

## 2. Reference implementations

Each repository was confirmed at its exact path and canonical casing with
`gh api repos/OWNER/NAME` before cloning, and shallow cloned into a scratch
directory outside the repository.

| Repository | Commit | Commit date | Licence |
|---|---|---|---|
| `FtrOnOff/FtrIO` | `7e214d5b3d62d31333031413b43f69d9c7e65a7d` | 26 June 2026 | MIT |
| `FtrOnOff/ftrio-python` | `b95ec4e1feaf1b3c74a30a43edd5e03c90f7ad68` | 8 July 2026 | None declared to GitHub |
| `FtrOnOff/ftrio-rust` | `e814d40758eb3524d64e9e40688bb8c59de65989` | 9 July 2026 | MIT |

Read: repository layout, CI workflows, test files, test doubles, the error type,
the changelogs and the porting notes.

Taken from them, and applied here in Go form:

- **Test names state subject, behaviour and condition.** The C# form is
  `ToggleParser_FallsBackToBase_WhenKeyMissingFromEnvFile`; the Rust and Python
  form is a descriptive snake case sentence. Go test names follow the Go form
  `TestSubject_Behaviour_WhenCondition`, which carries the same three parts.
- **Test doubles are named for what they stand in for, with a `TestDouble`
  suffix** (`CustomStrategyTestDouble`, `FtrIOContextAccessorTestDouble`), and
  carry a doc comment saying what they verify.
- **Shared test support lives in one place** (`tests/common/mod.rs`,
  `conftest.py`). Here that is one test support package (spec section 2, Ports).
- **One error type per failure family, with named fields** (`ToggleError` with
  `DoesNotExist { toggle_key }`). Here: typed errors with named fields and a
  reason category, wrapped with `%w`.
- **Comments explain why**, including at the top of CI workflow files and steps
  (the Rust `msrv` job explains why it exists).
- **A CI job proves the declared toolchain floor is real** by building on exactly
  that version. Here: the CI builds on the Go version named in `go.mod`.
- **Deviations are a table with a one line justification each**
  (`PORTING_NOTES.md`). Here: `IMPLEMENTATION_DEVIATIONS.md`.
- **Changelog** follows Keep a Changelog with per release sections.
- **Acronyms as one word in identifiers** (Rust denies
  `clippy::upper_case_acronyms`; `FtrIoContextAccessor`). Go idiom is the
  opposite (initialisms stay upper case: `URL`, `HID`, `ID`), and datum 0.5 says
  language idiom wins except on name length, so Go initialism rules apply here.

Not taken: the reference changelogs use United States spelling (behavior) and
quotation marks for emphasis in places. Datum 0.8 governs this repository instead.

## 3. Stream Deck XL protocol

Source: `abcminiuser/python-elgato-streamdeck`, commit
`064f6cf7a83146316b76cf84c946952302182091` (30 August 2026), package version
0.10.0, MIT licence. Read on 11 November 2026. Files:
`src/StreamDeck/ProductIDs.py`, `src/StreamDeck/DeviceManager.py`,
`src/StreamDeck/Devices/StreamDeck.py`, `src/StreamDeck/Devices/StreamDeckXL.py`,
`src/StreamDeck/ImageHelpers/PILHelper.py`,
`src/StreamDeck/Transport/LibUSBHIDAPI.py`.

Every value in this section is **read from source, not confirmed on hardware**.

### 3.1 Identity

| Value | Reading |
|---|---|
| Vendor ID | `0x0FD9` |
| XL product ID | `0x006C` (`USB_PID_STREAMDECK_XL`) |
| XL V2 product ID | `0x008F` (`USB_PID_STREAMDECK_XL_V2`) |
| XL V2 module product ID | `0x00BA` (`USB_PID_STREAMDECK_XL_V2_MODULE`) |
| Device class for all three | `StreamDeckXL` (DeviceManager table) |

`0x00C6` (`USB_PID_STREAMDECK_PLUS_XL`) maps to a different class,
`StreamDeckPlusXL`, and is not an XL variant. It is excluded from the model table.

The three XL rows share every protocol value below, so the model table has three
rows differing only in product ID.

### 3.2 Geometry and image format

| Value | Reading |
|---|---|
| Key count | 32 |
| Grid | 4 rows, 8 columns |
| Key image | 96 by 96 pixels |
| Image format | JPEG |
| Flip | horizontal and vertical (`KEY_FLIP = (True, True)`), applied after any rotation |
| Rotation | 0 |
| JPEG quality used by the reference | 100 |

Flipping on both axes is equivalent to a rotation of 180 degrees. The renderer
produces an upright image; the protocol encoder applies the flip, so the editor
preview and the device show the same `image.Image` (spec section 2).

### 3.3 Key index order

Not stated by the reference source. The reference example treats index
`key_count() - 1` as its exit key, and `key_layout()` returns `(rows, columns)`,
which is consistent with row major numbering from the top left (index
`row * 8 + column`) but does not prove it. **Inferred, unverified.** Experiment 1
settles it. The layout package takes the order from the model table, so a
correction is one table value and one fixture.

### 3.4 Input report (key state)

- The reference reads `4 + 32 = 36` bytes with `hid_read`, non blocking, at a
  default poll of 20 Hz.
- Bytes 4 to 35 are one byte per key; nonzero means pressed.
- Bytes 0 to 3 are not interpreted by the reference. Their content (report ID
  and any length field) is **unknown** and is recorded by experiment 1. Until
  then the decoder accepts exactly 36 bytes, requires a report ID byte matching
  the table, and treats bytes 1 to 3 as a header whose values are checked only
  once experiment 1 has recorded them.
- A read returning 0 bytes means no report is waiting; a negative result is a
  transport error, after which the reference stops reading.

### 3.5 Output reports

| Command | Report | Bytes |
|---|---|---|
| Reset key stream | output, 1024 bytes | byte 0 `0x02`, the rest zero; sent once on open |
| Key image page | output, 1024 bytes | 8 byte header then up to 1016 bytes of JPEG, zero padded |
| Reset | feature, 32 bytes | `0x03 0x02`, the rest zero |
| Brightness | feature | `0x03 0x08 percent`, the rest zero, percent clamped to 0 to 100 |
| Serial number | get feature, report ID `0x06`, 32 bytes | ASCII from offset 2, up to the first NUL |
| Firmware version | get feature, report ID `0x05`, 32 bytes | ASCII from offset 6, up to the first NUL |

Key image page header, in order: `0x02`, `0x07`, key index, `1` if this is the
last page else `0`, payload length low byte, payload length high byte, page
number low byte, page number high byte. Pages are numbered from 0, and each
carries at most 1016 bytes of the image.

Two observations on the reference, recorded as evidence, not copied:

- **Brightness report length.** The reference writes
  `payload[0:2] = [0x03, 0x08, percent]` into a 32 byte array. Python slice
  assignment inserts, so the report actually sent is 33 bytes. The device
  evidently tolerates this. This project sends 32 bytes with the same first three
  values; experiment 1 confirms the device accepts it.
- **Key index bound.** The reference's check `min(max(key, 0), 32) != key`
  admits key 32, which does not exist. This project's encoder rejects any index
  outside 0 to 31.

### 3.6 macOS feature report quirk

The reference oversizes `hid_get_feature_report` by one byte on macOS only,
citing a bug in hidapi 0.9.0. The candidate Go wrapper bundles hidapi 0.15.0
(section 4), so the workaround is probably unneeded. **Unverified;** experiment 1
reads the serial number on macOS to settle it.

## 4. Go HID libraries (TD3)

Read on 11 November 2026, from shallow clones and `gh api`.

| | `sstallion/go-hid` | `karalabe/hid` |
|---|---|---|
| Commit read | `3879eca0960538df22249770709b4ac27c8550d2` (tag `v0.15.0`, 23 May 2025) | `f5d04adeffebd8c81a6f69ec187baebc31856b42` (15 March 2026) |
| Repository status | Active, not archived; 4 open issues | **Archived** (read only) |
| Licence | BSD 2 clause (wrapper); bundled hidapi offers GPL v3, BSD style or the original HIDAPI licence | 3 clause BSD for hidapi, 2 clause BSD for the wrapper, LGPL 2.1 for bundled libusb (used on Linux only) |
| hidapi bundled | Yes, source in tree, version 0.15.0 (`HID_API_VERSION_MINOR 15`) | Yes, version 0.14.0 |
| Build | cgo; macOS links IOKit and CoreFoundation; Windows needs only a C compiler | cgo; same frameworks plus `-lobjc` |
| Go floor | `go 1.17` | `go 1.19` |
| Hotplug | No: `hidapi.h` has no hotplug API. Watching is by periodic enumeration | No |
| API | `Init`, `Enumerate(vid, pid, func)`, `OpenPath`, `Write`, `ReadWithTimeout`, `SendFeatureReport`, `GetFeatureReport`, `Close`, `GetVersionStr` | `Enumerate`, `Open`, `Write`, `ReadTimeout`, `SendFeatureReport`, `GetFeatureReport`, `Close` |

Upstream hidapi (`libusb/hidapi`): latest release `hidapi-0.15.0`, 18 May 2025,
so go-hid bundles the current release. Read with `gh api`.

Open go-hid issues relevant to this project, read with `gh api`:

- #18: on macOS `hid.Enumerate` panics on an invalid HID string. The watcher
  enumerates on a timer, so a panic there would take the daemon down. The
  adapter recovers from it at the enumeration boundary and counts it, or a fix
  lands upstream first.
- #15: transient `IOHIDDeviceSetReport failed (0xE00002BC)` on macOS under cgo.
  This is the same call the image writes make; experiment 1 watches for it.
- #17: an invalid `free()` pointer report. Read before TE6.

Consequence for DeckWatcher: neither library offers hotplug callbacks, so
connect and disconnect are detected by enumerating at an interval and by write
or read errors on the open device. The interval is a value for TE6.

## 5. GitHub Actions (TD7, TD8)

Read by delegate on 11 November 2026, docs.github.com only.

- **Endpoint:** `GET /repos/{owner}/{repo}/actions/workflows/{workflow_id}/runs`,
  where `workflow_id` is the numeric ID or the file name. Filters include
  `branch`, `status` and `per_page` (default 30, maximum 100).
  <https://docs.github.com/en/rest/actions/workflow-runs?apiVersion=2022-11-28>
- **Run fields used:** `status`, `conclusion` (both nullable strings) and
  `html_url`.
- **Value sets:** the workflow run page does not enumerate `status` and
  `conclusion` for the response. The `status` filter accepts completed,
  action_required, cancelled, failure, neutral, skipped, stale, success,
  timed_out, in_progress, queued, requested, waiting and pending. The check runs
  page lists status as queued, in_progress, completed, waiting, requested,
  pending. Any value outside the mapping becomes the unknown appearance, counted
  by reason. The mapping itself is TD10.
- **API version header:** `X-GitHub-Api-Version`. Supported: `2026-03-10`, and
  `2022-11-28` until 10 March 2028. Requests without the header default to
  `2022-11-28`. Header to send: TD8 batch.
  <https://docs.github.com/en/rest/about-the-rest-api/api-versions>
- **Fine grained token:** repository permission Actions, read.
  <https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens>
- **Primary rate limit:** 5,000 requests per hour per authenticated user; headers
  `x-ratelimit-limit`, `x-ratelimit-remaining`, `x-ratelimit-used`,
  `x-ratelimit-reset`, `x-ratelimit-resource`.
  <https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api>
- **Conditional requests:** send the saved `etag` as `If-None-Match`; a `304`
  made with a correct `Authorization` header does not count against the primary
  rate limit.
  <https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api>
  **Uncertainty:** the workflow run page lists a 304 response for the repository
  wide runs endpoint but not for the per workflow one. Whether the per workflow
  endpoint returns an `etag` and honours `If-None-Match` needs a real request
  with the author's token (experiment 5, below).
- **Rate limited responses:** `403` or `429`. Honour `retry-after` if present;
  else if `x-ratelimit-remaining` is 0, wait until `x-ratelimit-reset` (UTC epoch
  seconds); else wait at least one minute, then back off exponentially.
- **Secondary limits:** 100 concurrent requests; 900 points per minute.
- **Polling guidance:** poll on a fixed schedule only as often as needed, and obey
  `x-poll-interval` if present. No documented default interval for workflow runs.
- **Base URLs:** github.com `https://api.github.com`; Enterprise Server
  `https://HOSTNAME/api/v3`. GHE.com with data residency: documentation says to
  use the enterprise's dedicated URL and shows `https://api.SUBDOMAIN.ghe.com`
  only for SCIM; the REST base `https://api.SUBDOMAIN.ghe.com` is **inferred**.

Budget arithmetic for TD8: one workflow polled every 30 seconds is 120 requests
per hour, 2.4 percent of the primary limit before any 304 saving; ten distinct
workflows at that interval are 1,200 per hour, 24 percent.

## 6. Platform mechanisms

Read by delegate on 11 November 2026, from developer.apple.com, learn.microsoft.com,
the local macOS 27.2 man pages and SDK headers, and upstream repositories.

### 6.1 Opening URLs and launching applications

- **macOS, URL:** `NSWorkspace.open(_:)` opens a URL with its LaunchServices
  handler and is safe from any thread. `open(1)` with a URL does the same; `-u`
  forces scheme handling even if the URL also matches a file path.
  <https://developer.apple.com/documentation/appkit/nsworkspace/open(_:)>
- **macOS, application:** `NSWorkspace.urlForApplication(withBundleIdentifier:)`
  then `openApplication(at:configuration:completionHandler:)`;
  `OpenConfiguration.activates` defaults to true, which raises a running app.
  `open(1)` flags: `-b bundle_identifier`, `-a application`.
- **Windows, URL:** `ShellExecuteW` with verb `open`; success is a return above
  32; initialise COM first with
  `COINIT_APARTMENTTHREADED | COINIT_DISABLE_OLE1DDE`.
  <https://learn.microsoft.com/en-us/windows/win32/api/shellapi/nf-shellapi-shellexecutew>
- **Windows, packaged app by application user model ID:**
  `IApplicationActivationManager::ActivateApplication(appUserModelId, arguments, options, processId)`,
  with `AO_NOERRORUI` available.
  <https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-iapplicationactivationmanager-activateapplication>
  The `explorer.exe shell:AppsFolder\ID` form has no Microsoft documentation as
  a launch method and is not used.
- **Windows, desktop app by path:** `ShellExecuteW` with the executable path.
- **Windows, never `cmd /c start`.** cmd treats `&`, `|` and parentheses as
  special, and URL query strings routinely contain `&`; Go's argument quoting
  explicitly excludes cmd.exe. Passing a configured URL through cmd would let a
  configuration value run commands. `ShellExecuteW` takes the URL as one string.
  <https://learn.microsoft.com/en-us/windows-server/administration/windows-commands/cmd>

Consequence for the adapters: on macOS, shelling out to `/usr/bin/open` with the
URL as one argument (no shell) avoids an Objective C bridge; cgo with AppKit is
the alternative. On Windows, `ShellExecuteW` and `ActivateApplication` are
reachable through `golang.org/x/sys/windows` and COM. The choice is TD11.

### 6.2 macOS HID permission

Apple's IOHIDFamily source (`apple-oss-distributions/IOHIDFamily`, tag
`IOHIDFamily-2238.120.5`) requires Input Monitoring authorisation only for devices
conforming to Generic Desktop Keyboard, Generic Desktop Mouse or Digitizer
Touchpad. A vendor defined usage page does not trigger the prompt. hidapi issue
#371 agrees.

The Stream Deck XL's usage page and usage are **unknown**. Experiment 1 records
them from enumeration. If it exposes only a vendor or consumer collection, no
permission is needed; if it exposes a keyboard collection, the daemon needs Input
Monitoring and the busy state (TE6) must say so.

### 6.3 Credential stores

| | `zalando/go-keyring` | `99designs/keyring` |
|---|---|---|
| Latest tag | v0.2.8, 23 March 2026 | v1.2.2, 19 December 2022 |
| Last commit | 24 July 2026 | 19 December 2022 |
| Licence | MIT | MIT |
| macOS mechanism | runs `/usr/bin/security -i`, secret written to its standard input (not its arguments), stored base64 with a prefix; refuses over 4096 bytes of command | cgo, via `99designs/go-keychain` pinned to a 2019 pseudo version |
| Windows mechanism | `danieljoos/wincred`; refuses secrets over 2560 bytes | `danieljoos/wincred` v1.1.2 |

Windows `CREDENTIALW.CredentialBlobSize` is at most `CRED_MAX_CREDENTIAL_BLOB_SIZE`
(5 times 512, so 2560 bytes).
<https://learn.microsoft.com/en-us/windows/win32/api/wincred/ns-wincred-credentialw>
A GitHub fine grained token is well inside that. Library choice is part of TD12.

### 6.4 Start at login without administrator rights

- **macOS:** a per user agent in `~/Library/LaunchAgents`, loaded with
  `launchctl bootstrap gui/UID PATH`. `KeepAlive` implies `RunAtLoad`; crashing
  jobs are throttled. (local `man launchd.plist`, `man launchctl`)
- **Windows, scheduled task:** a task registered from a non administrator account
  under that account's own context with an interactive logon type needs no
  password and no elevation.
  <https://learn.microsoft.com/en-us/windows/win32/taskschd/security-contexts-for-running-tasks>
  Whether `schtasks /create /sc onlogon` (which means any user) works without
  elevation is **not stated**; a task defined in XML with a logon trigger scoped
  to the user's ID is the documented route. **Unverified** until run on the Dell.
- **Windows, alternative:** `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`,
  a command line of at most 260 characters, which the system may delay.

### 6.5 Licences for bundled material (TD2, TD5, TD6)

| Item | Version | Licence |
|---|---|---|
| hidapi (bundled by go-hid) | 0.15.0 | choice of GPL v3, BSD style, or original HIDAPI licence |
| go-hid | v0.15.0 | BSD 2 clause |
| htmx | v4.0.1 (also v2.0.11 on the 2.x line), 1 November 2026 | Zero Clause BSD |
| Go fonts (`golang.org/x/image/font/gofont`) | from `golang.org/x/image` | BSD 3 clause style, Bigelow and Holmes |
| Lucide icons | 1.72.0, 11 November 2026 | ISC, with Feather derived icons under MIT |
| Tabler Icons | v3.51.0, 8 November 2026 | MIT |
| Material Symbols | content updated 23 October 2026 | Apache 2.0 |

Every item above is compatible with an MIT or BSD licence for this project.

### 6.6 Deep link formats (presets)

Read by delegate on 11 November 2026. Status per format: **official** (vendor
documentation), **observed** (seen on the vendor's site, not documented),
**unofficial** (forum only), **not found**.

| Preset | Format | Status | Source |
|---|---|---|---|
| Teams chat by participants | `https://teams.microsoft.com/l/chat/0/0?users=UPN1,UPN2` with optional `topicName` (three or more users), `message` (draft only, never sent) and `tenantId` (undescribed) | Official | <https://learn.microsoft.com/en-us/microsoftteams/platform/concepts/build-and-test/deep-link-teams> |
| Teams chat by ID | `https://teams.microsoft.com/l/chat/CHAT_ID/conversations`, chat ID of the form `19:...`, copied from a message's Copy link | Official | same page |
| Teams channel | `https://teams.microsoft.com/l/channel/CHANNEL_ID/CHANNEL_NAME?groupId=GROUP_ID&tenantId=TENANT_ID`, both path parts URL encoded; private channels add `ngc=true`; shared channels add `ngc=true&allowXTenantAccess=true` | Official; no part is labelled optional, so all four are treated as required | same page |
| Teams desktop scheme | `msteams://` skips the browser's client picker; must not be prefixed to an `https://` link; the exact full form for chat and channel links is not given | Official handler, form not found | <https://learn.microsoft.com/en-us/microsoftteams/platform/concepts/build-and-test/deep-links> |
| Discord channel | `https://discord.com/channels/SERVER_ID/CHANNEL_ID`; IDs copied with Developer Mode (User Settings, Advanced) | Official example; channel position inferred | <https://docs.discord.com/developers/tutorials/hosting-on-cloudflare-workers>, support article 206346498 |
| Discord app scheme | `discord://` channel form | Not found | |
| D&D Beyond character | `https://www.dndbeyond.com/characters/CHARACTER_ID` | Observed | <https://www.dndbeyond.com/characters> |
| Roll20 game | No launch URL form documented. Help Centre says Launch Game goes straight to the tabletop; a forum champion (not staff) says the Launch link carries the campaign. `setcampaign` and `campaigns/details` forms appear in no fetched source | Not found | help article 29619836768791; forum permalink 11906221 |

New Teams on Windows: package family name `MSTeams_8wekyb3d8bbwe`, application
ID `MSTeams_8wekyb3d8bbwe!MSTeams`, documented by Microsoft for RemoteApp
publishing as `shell:appsFolder\MSTeams_8wekyb3d8bbwe!MSTeams`. The executable
path changes with every update, so a path based launch is unsuitable.
<https://learn.microsoft.com/en-us/microsoftteams/teams-client-vdi-requirements-deploy>

Privacy: the participants form puts colleagues' user principal names (usually
email addresses) in the configuration file, as spec section 8 anticipates. Logs
never carry URLs (spec section 7).

## 7. Experiments

| Number | Experiment | Status |
|---|---|---|
| 1 | Open the XL through each HID library, read reports, write an image, replug | Not done: no deck (DV1). karalabe/hid is archived, so experiment 1 runs against go-hid only unless the author wants both |
| 2 | Dock switch between machines | Not done: no deck (DV1) |
| 3 | Open each preset's deep link from a shell on each OS | Not done: author's machines and accounts (DV1) |
| 4 | Build with cgo on each machine and on GitHub Actions runners | Not done. A throwaway build of go-hid on this Mac, enumerating HID devices, was refused by the session's permission classifier; it awaits the author's decision. The runner builds need a push, which is the author's |
| 5 | One authorised request to the per workflow runs endpoint, then a repeat with `If-None-Match`, recording `etag`, `304` and rate limit headers | Added by this record. Not done: needs the author's token, run by the author |

## 8. Findings

| ID | Finding | Evidence | Owner and likely fix |
|---|---|---|---|
| TF5 | go-hid `Enumerate` can panic on macOS on an invalid HID string | go-hid issue #18 | Upstream go-hid. Until fixed, the hidtransport adapter recovers at the enumeration boundary, counts it and reports the watcher as faulted rather than crashing (TE6) |
| TF6 | The reference library sends a 33 byte brightness feature report and admits key index 32 | Section 3.5 | Upstream python-elgato-streamdeck. No action here beyond not copying either |
| TF7 | `go-yaml/yaml` (the long standing `gopkg.in/yaml.v3`) is archived; `yaml/go-yaml` is maintained, Apache 2.0, v4.0.0 released 4 November 2026; `goccy/go-yaml` is maintained, MIT, v1.19.2 | `gh api`, 11 November 2026 | Informs TD4 |
| TF9 | (Resolved in TE1 by the standalone dupl, proven on the plant.) golangci-lint's `dupl` compares code only within one package. At thresholds of 150, 125, 100 and 75 tokens it found nothing in a scratch tree holding the same 25 line function in `internal/bricks/plantedone` and `internal/bricks/plantedtwo`; at 50 it reported only repetition inside single test files. It cannot catch the cross package duplicate TE1 must plant | TE1 planted run, 11 November 2026 | This project, TE1: run the standalone `mibk/dupl` (MIT, v1.1.0, which scans files across the whole tree) pinned in `tools/go.mod`, subject to the author's permission (see TE1 below) |
| TF10 | golangci-lint keeps one issue per line by default (`uniq-by-line`), so a planted import layering violation was hidden behind a different linter's finding on the same line | TE1 planted run | This project, TE1: fixed by `uniq-by-line: false` and no caps; the plant then failed as required |
| TF11 | go-yaml v4 `WithUniqueKeys(true)` does not reject a duplicated mapping key when the target is a `yaml.Node`; `a: 1` then `a: 2` loads with no error. Multiple documents are rejected as documented | TE2 test `TestDecode_RejectsMalformedDuplicateAndMultipleDocuments`, then a probe of `yaml.Load` into a node, 11 November 2026 | Upstream go-yaml (worth an issue). This project: the decoder walks the tree and rejects duplicates itself |
| TF12 | golangci-lint v2.15.0 ignores an unknown key in `.golangci.yml`: a misspelt `exhaustruct_v5` setting (`exclude` instead of `ignore-patterns`) switched nothing on and produced no warning. `golangci-lint config verify` refuses it (exit 3) | TE4, the agent's own configuration error, 11 November 2026 | This project, TE4: CI now runs `config verify` before linting |
| TF8 | The spec's press behaviour for `pipeline_status` reads the same either way: if the latest run failed, opening the latest failing run is opening the latest run | Spec section 2, Bricks in scope | Author, TD10 part b |

## 9. Decisions for the author

Asked in the datum's five part form (0.3), batched by the phase that needs them.
Recommendations are labelled. TD1 to TD9 are the spec's; TD10 to TD13 arose in
this Plan stage.

### Needed by TE1

**TD1: identifier spellings.**

1. Blocked: the module path, binary name, config folder, credential service name,
   LaunchAgent label and editor title.
2. Why evidence does not settle it: these are naming choices; Go's
   `os.UserConfigDir` gives the parent folder (macOS
   `~/Library/Application Support`, Windows `%AppData%`) but not the child name;
   a LaunchAgent label is conventionally reverse DNS and needs a domain the author
   is happy to use.
3. Options: module path `github.com/TheScottBot/the-front-buttons` or another;
   folder `the-front-buttons` or `The Front Buttons`; label
   `io.github.thescottbot.the-front-buttons` or another.
4. Trade offs: a lower case folder with no spaces is easier to type in a shell
   and in scripts; a display name folder matches what macOS users see elsewhere
   in Application Support. The module path is permanent once imports use it.
5. **Recommendation:** display name The Front Buttons wherever a human reads it,
   including the editor title; `the-front-buttons` for the binary, the
   credential service and the folder under `os.UserConfigDir`, with config at
   `the-front-buttons/config.yaml` and state (logs) in `the-front-buttons/state/`;
   label `io.github.thescottbot.the-front-buttons`; module path
   `github.com/TheScottBot/the-front-buttons` if that is where it will live.

**Duplicate code linter threshold** (Appendix A row). Recommendation: golangci-lint's
`dupl` at its default threshold, lowered only if it misses the planted duplicate in
TE1. The value is recorded when TE1 proves it against the planted violation.

### Needed by TE2

**TD4: config file format and library.**

1. Blocked: the format, and the library that parses it.
2. Why: the spec recommends YAML, but the familiar YAML library is archived (TF7),
   so the recommendation now costs a choice of fork.
3. Options: YAML with `yaml/go-yaml` v4; YAML with `goccy/go-yaml`; TOML with
   `BurntSushi/toml` (MIT, v1.6.0); JSON with the standard library.
4. Trade offs: YAML reads best for nested pages of keys and allows comments, at
   the cost of a dependency and YAML's implicit typing (strings like `no` or
   `0123`), which the schema layer must guard. TOML is less ambiguous but deep
   nesting of pages and keys reads awkwardly. JSON needs no dependency and
   `DisallowUnknownFields` is built in, but has no comments and is unpleasant to
   write by hand before the editor exists.
5. **Recommendation:** YAML through `yaml/go-yaml` v4 (maintained by the YAML
   organisation, Apache 2.0), decoded in strict mode so unknown fields fail, with
   every value typed by the derived schema rather than by YAML's guessing. The
   library's strict mode and licence are read from source and recorded under
   datum 0.13 before TE2 starts.

### Needed by TE4

**TD5: icons.**

1. Blocked: whether a bundled icon set ships, and which.
2. Why: Lucide and Tabler are SVG sets; drawing SVG in Go needs a rasteriser
   dependency, which the spec did not anticipate.
3. Options: user files only; Lucide (ISC, 1.72.0) or Tabler (MIT, v3.51.0)
   rasterised at run time through an SVG dependency; the same set rasterised once
   to PNG at 96 pixels by a one time tool (datum 0.13: recorded, not a
   dependency) and vendored with licence and digests.
4. Trade offs: run time SVG gives any size but adds a parser of untrusted input
   to the daemon; pre rasterised PNG adds no runtime dependency but fixes the
   sizes and grows the repository.
5. **Recommendation:** user supplied PNG files plus a curated subset of Lucide
   rasterised once to PNG and vendored, with the tool, its version and its
   invocation recorded. User SVG support can come later as its own decision.

### Needed by TE6

**TD3: HID library.**

1. Blocked: which Go HID wrapper the adapter uses.
2. Why: needs a choice between candidates; experiment 1 is deferred (DV1).
3. Options: `sstallion/go-hid` v0.15.0; `karalabe/hid` (archived).
4. Trade offs: go-hid is maintained, bundles the current hidapi 0.15.0, is BSD 2
   clause, and has open macOS issues (TF5, #15). karalabe/hid is archived, so a
   fix there is nobody's job (datum 0.13).
5. **Recommendation:** `sstallion/go-hid` v0.15.0, with the TF5 guard; experiment
   1 runs against go-hid only.

### Needed by TE7

**TD11: how the platform adapters open URLs and launch applications.**

1. Blocked: the mechanism behind `URLOpener` and `ApplicationLauncher` on each OS.
2. Why: each OS offers several documented routes (section 6.1) with different
   dependency and safety costs.
3. Options: macOS, run `/usr/bin/open` directly (no shell) or call AppKit through
   cgo. Windows URLs and paths, `ShellExecuteW`. Windows packaged apps, COM
   `ActivateApplication` called directly, the `go-ole` dependency, or
   `ShellExecuteW` on `shell:appsFolder\ID` (the form Microsoft uses for Teams
   RemoteApp, not documented as a general launch method).
4. Trade offs: `/usr/bin/open` is documented, needs no Objective C, and is safe
   when the URL is one argument and validated not to start with a hyphen; AppKit
   through cgo adds a second cgo package against the layout rule. Direct COM is
   more code but no dependency; `go-ole` is less code and a dependency.
5. **Recommendation:** macOS `/usr/bin/open -u URL`, `open -b BUNDLE_ID` and
   `open -a NAME`, each as discrete arguments with no shell; Windows
   `ShellExecuteW` for URLs and executable paths, and direct COM
   `ActivateApplication` for application user model IDs. Never `cmd`.

**TD13: preset URL forms.**

1. Blocked: which form each preset emits, and what to do where no form is
   documented.
2. Why: section 6.6. Teams is documented; Discord's app scheme and Roll20's launch
   link are not.
3. Options: Teams `https://` only, or `msteams://` once experiment 3 shows a
   working form. Discord `https://` only, or an undocumented `discord://`. Roll20:
   the author copies the real Launch Game link target so the preset is built from
   it; or no Roll20 preset (plain `open_url`).
4. Trade offs: `https://` Teams links pass through the browser's client picker
   each time; `msteams://` skips it but its exact form is not documented. A
   Discord `https://` link opens the browser, not the app, unless the system
   routes it. Guessing a Roll20 form would break datum 0.2.
5. **Recommendation:** Teams `https://` presets for chat by participants, chat by
   ID, and channel (with a channel type field: standard, private, shared), plus an
   optional desktop flag added only if experiment 3 proves an `msteams://` form;
   Discord `https://` only; D&D Beyond as observed; Roll20 held until the author
   copies one Launch Game link, then built from it.

### Needed by TE8

**TD7: GitHub host.**

1. Blocked: github.com only, or a configurable API base URL.
2. Why: the work repositories' host is the author's knowledge.
3. Options: fixed `https://api.github.com`; configurable, defaulting to it
   (Enterprise Server is `https://HOSTNAME/api/v3`; GHE.com is inferred as
   `https://api.SUBDOMAIN.ghe.com`).
4. Trade offs: configurable costs one validated field and an allow list check
   that it is `https`; fixed is simpler but cannot reach an Enterprise host.
5. **Recommendation:** configurable, `https` only, defaulting to
   `https://api.github.com`. Tell me if the work repositories are on github.com,
   Enterprise Server or GHE.com.

**TD8: polling interval and API version.**

1. Blocked: default and minimum interval for `pipeline_status`; the
   `X-GitHub-Api-Version` value.
2. Why: GitHub documents no default interval; 304 responses are free only if the
   per workflow endpoint supports them (experiment 5).
3. Options: default 30, 60 or 120 seconds; minimum 15 or 30 seconds. API version
   `2022-11-28` (supported until 10 March 2028, the version the endpoint page
   was read at) or `2026-03-10`.
4. Trade offs: section 5 arithmetic: ten workflows at 30 seconds use 24 percent
   of the hourly limit with no 304 saving; at 60 seconds, 12 percent. Shorter
   feels livelier on the deck. Pinning `2026-03-10` needs its response schema read
   first.
5. **Recommendation:** default 60 seconds, minimum 30, always lengthened by
   `x-poll-interval`, `retry-after` or a reset wait when GitHub sends them; send
   `X-GitHub-Api-Version: 2022-11-28` and revisit before 2028.

**TD10: run state mapping and press target.**

1. Blocked: (a) how GitHub's `status` and `conclusion` values map to the six
   key states; (b) what a press opens.
2. Why: GitHub's values outnumber the spec's six states, and the spec's press rule
   reads the same either way (TF8).
3. Options for (a): see the recommendation; the open choices are where
   `timed_out`, `action_required`, `neutral`, `skipped` and `stale` go. Options
   for (b): always the latest run; or, while a run is in progress or queued, the
   most recent completed run if it failed, otherwise the latest run.
4. Trade offs: mapping more values to failure is noisier but safer; a press that
   finds the last failure while a fix is running is more useful at work but needs
   a second query or a larger page of runs.
5. **Recommendation:** (a) `completed` with `success` is success; with `failure`
   or `timed_out` is failure; with `cancelled` is cancelled; `in_progress` is in
   progress; `queued`, `waiting`, `requested`, `pending` and `action_required`
   are queued; `neutral`, `skipped`, `stale`, null and anything unrecognised are
   unknown, counted by value. (b) Press opens the most recent completed failed run
   while a newer run is in progress or queued, otherwise the latest run, fetched
   from one request with `per_page` large enough to find it.

**TD12: credential store library.**

1. Blocked: which library backs `CredentialStore`.
2. Why: two candidates, section 6.3.
3. Options: `zalando/go-keyring` v0.2.8; `99designs/keyring` v1.2.2; hand written
   adapters.
4. Trade offs: go-keyring is maintained, needs no cgo, and passes the secret on
   standard input to `/usr/bin/security`, never on the command line;
   99designs/keyring has had no commit since 2022 and pins a 2019 cgo Keychain
   binding. Hand written adapters avoid a dependency but reimplement Keychain
   access.
5. **Recommendation:** `zalando/go-keyring` v0.2.8.

### Needed by TE9

**TD6: editor front end.**

1. Blocked: server rendered templates with htmx, or a TypeScript single page app;
   and which htmx line.
2. Why: htmx has two current lines: v2.0.11 and v4.0.1 (major release 28
   August 2026), both Zero Clause BSD.
3. Options: templates with htmx 2.0.11; templates with htmx 4.0.1; a TypeScript
   app.
4. Trade offs: the spec's reasoning for templates stands (one schema, no second
   copy in TypeScript). htmx 4 is the newest line but two months old; 2.x is
   long proven and still maintained.
5. **Recommendation:** Go templates with htmx 2.0.11, vendored with its digest.

**TD9: opening the editor.** No new evidence. **Recommendation** unchanged from
the spec: the printed URL plus an `open_editor` brick, no tray icon.

### Needed by TE10

**TD2: licence.**

1. Blocked: this project's licence.
2. Why: an author choice; Plan has now read every bundled licence (section 6.5).
3. Options: MIT; BSD 2 clause; Apache 2.0.
4. Trade offs: all are compatible with go-hid (BSD 2), hidapi (taken under its BSD
   option), go-keyring (MIT), yaml/go-yaml (Apache 2.0), htmx (0BSD), Go fonts
   (BSD 3) and Lucide (ISC and MIT). Apache 2.0 adds a patent grant and a NOTICE
   obligation; MIT matches FtrIO and ftrio-rust.
5. **Recommendation:** MIT, holder as the author chooses.

### Not a decision: a permission

Experiment 4's local half, a throwaway Go program in the scratch directory that
fetches go-hid v0.15.0, builds it with cgo and lists HID devices on this Mac, was
refused by the session's permission classifier. It writes nothing in the
repository and opens no device. It proves cgo and the bundled hidapi build here
before TE6 relies on it. The author may allow it, run it themselves, or leave it
until TE6.

## 10. Plan stage exit status

Exited on 11 November 2026, under DV1. Every decision TD1 to TD13 is answered
(section 11). Experiments 1 to 5 remain deferred under DV1 or await the author,
and every Appendix A row is read from source rather than verified on hardware.
Each phase that depends on an experiment lists it under Assumptions to verify and
cannot clear its author gate until the experiment is recorded.

## 11. Author decisions, 11 November 2026

### 11.1 Description line corrected

The author asked for the spelling of the settled description to be checked.
`wirery` is not a word; the intended word is `wiry`. The author's specification
file was corrected in both places the line appears (section 1, and the Settled by
the author table). The line TE1's README check enforces is now exactly:

This is for the lions living in the wiry frames of my peripherals bodies

Open with the author: `peripherals bodies` reads as a plural possessive, which
would be written `peripherals' bodies`. Not changed without the author's word,
because the check enforces the line byte for byte.

(11 November 2026, later: the author answered "Yes, add it". The specification
was corrected in both places. The line TE1's README check enforces is now exactly:

This is for the lions living in the wiry frames of my peripherals' bodies

The apostrophe is U+0027, a plain ASCII apostrophe, not a typographic quotation
mark, so the TE1 prose scan must not treat it as quotation.)

The rest of the specification was scanned on the same date for en and em dashes,
converter mangled hyphen runs and common United States spellings. No hits outside
the file name `LICENSE`.

### 11.2 Decisions accepted

The author accepted every recommendation in section 9 as written: TD1, TD2, TD3,
TD6, TD7, TD8, TD9, TD10, TD11, TD12 and TD13, plus the duplicate code linter
approach. TD4 and TD5 were accepted subject to the further evaluation in 11.3 and
11.4. The work repositories' GitHub host (TD7) was not stated; the configurable
base URL covers every case.

### 11.3 TD4 revisited: is YAML the right format for a Go project

Read on 11 November 2026: `yaml/go-yaml` at tag `v4.0.0`, commit
`f2f9563503f11b7a60da0c532175dcc693ce8d5a`, module `go.yaml.in/yaml/v4`,
Apache 2.0, `go 1.18`. Its README states v4.0.0 is the stable release, with v2
and v3 frozen and receiving security fixes only.

What the source shows:

- `WithKnownFields()` rejects unknown fields (datum 0.11).
- `WithUniqueKeys()` rejects duplicate mapping keys.
- Tests assert an `excessive aliasing` error, so alias expansion is bounded.
- `Node` keeps `HeadComment` and `LineComment`, so comments can survive an editor
  save if the editor rewrites through the node tree.

Weighed for this project:

| Concern | YAML (go-yaml v4) | JSON (standard library) | TOML (BurntSushi) |
|---|---|---|---|
| Hand editing pages of keys before the editor exists | Best: nesting reads naturally, comments allowed | Poor: no comments, brace and comma noise | Fair: deep arrays of tables read awkwardly |
| Unknown field rejection | `WithKnownFields` | `DisallowUnknownFields` | Possible through undecoded key metadata, not read |
| Duplicate keys | Rejected with `WithUniqueKeys` | `encoding/json` accepts them silently, last wins | Not read |
| Comments surviving an editor save | Possible through `Node` | No comments to keep | Lost |
| Implicit typing traps | Present, neutralised by decoding into typed structs and validating through the derived schema | None | Few |
| Dependency | One, Apache 2.0 (NOTICE must ship) | None | One, MIT |
| Familiarity in Go tooling | The common choice (golangci-lint, GitHub Actions, Kubernetes) | Common for machine written files | Less common |

Conclusion: YAML remains the right choice. JSON's zero dependency is real, but
silently accepting duplicate keys breaks the strictness the datum requires and it
is the worst of the three to write by hand. Library: `go.yaml.in/yaml/v4`.
Its one risk is age: the v4 major is a week old. TE2's fuzz harness on the
decoder runs against it from the start, and v3 (frozen, security fixes) is the
fallback if v4 misbehaves.

Raised for TE9, not decided here: whether the editor preserves hand written
comments by saving through the node tree, or documents that saving from the
editor drops them.

### 11.4 TD5 revisited: the best route for icons and PNG files

The question was how to get icons onto keys well. Read on 11 November 2026:
the `lucide-static@1.72.0` and `@tabler/icons-webfont@3.51.0` packages, by file
listing and by their codepoint tables, fetched into the scratch directory.

Finding: both sets ship as a TrueType font as well as SVG. `lucide.ttf`
(997,420 bytes, 2,376 icons, ISC) has no Discord mark. `tabler-icons.ttf`
(1,849,916 bytes, 5,296 icons, MIT) includes `brand-discord`, `brand-teams`,
`brand-github`, `dice-5` and the status shapes.

That changes the route. The renderer already needs a TrueType pipeline for labels
(`golang.org/x/image/font/opentype` with the Go fonts). An icon font goes through
that same pipeline, so:

- no SVG rasteriser dependency, and no untrusted SVG parser in the daemon
- no one time conversion tool and no folder of pre rendered PNG files
- icons scale to any size and take any colour, so a status can recolour its icon
  while the glyph and text (datum 0.16) still carry the meaning
- the editor palette lists icons by name from the font's codepoint table

The whole image path then uses only the standard library and the Go project's own
`golang.org/x/image`:

| Step | Package |
|---|---|
| Decode a user's PNG icon | `image/png`, with `png.DecodeConfig` checked against a pixel bound and the file read under a byte bound first (datum 0.10) |
| Scale it to the key | `golang.org/x/image/draw` |
| Draw labels and font icons | `golang.org/x/image/font/opentype`, Go fonts, Tabler icon font |
| Encode for the device | `image/jpeg`, after the flip from the model table |
| Encode for the editor preview | `image/png` |

Revised recommendation, for the author to confirm: bundled icons come from the
vendored Tabler icon font (MIT, v3.51.0, with its licence and digest recorded under
datum 0.13), addressed by name; user icons are PNG files only, bounded as above.
SVG user icons stay out of scope. Brand marks depict trademarks of their owners;
for keys on the author's own desk this is ordinary use, and nothing is
distributed.

### 11.5 TD5 confirmed

The author confirmed the revised route on 11 November 2026: "Yes, use the font".
Bundled icons come from the vendored Tabler icon font, `@tabler/icons-webfont`
v3.51.0, MIT, addressed by name through its codepoint table, with licence and
digest recorded under datum 0.13 at TE4. User icons are PNG files only, read under
a byte bound and checked with `png.DecodeConfig` against a pixel bound before
decoding. SVG user icons are out of scope.

## TE1: repository and checks

Started 11 November 2026, after rereading datum Part 0 and 0.17.

### What was built

- Module `github.com/TheScottBot/the-front-buttons`, `go 1.27.3`, and the package
  layout of specification section 2, each package with a doc comment saying why it
  exists. Ports in `internal/platform`; their adapters in
  `internal/platform/systemadapter`, so a brick can depend on a port without
  importing operating system code.
- `internal/repositorychecks`: the prose and typography scan (dash characters
  anywhere; hyphen runs in prose, which in Go means comments and string literals
  only; the Markdown table separator and long flag token exemptions), the
  operating system check (imports of `syscall`, `C`, `os/exec`,
  `golang.org/x/sys`; `runtime.GOOS` under any import name; operating system file
  suffixes and build constraints, using Go's own lists copied from
  `internal/syslist`), and the README first line check. `TestRepository_HasNoViolations`
  runs them over the tree as part of `go test`.
- `.golangci.yml`: every linter enabled, three deprecated duplicates disabled with
  the reason, depguard enforcing that only `cmd/the-front-buttons` imports
  adapters, every finding reported, and the gofmt, gofumpt and goimports
  formatters checked.
- `tools/go.mod`: golangci-lint v2.15.0 as a Go tool, in its own module.
- `.github/workflows/ci.yml`: build and vet, `go test -race`, and lint, on
  `macos-26` (arm64) and `windows-2025` (amd64), runner labels read from
  `actions/runner-images` on 11 November 2026; actions pinned by commit hash
  (`actions/checkout` v7.0.1 `3d3c42e5aac5ba805825da76410c181273ba90b1`,
  `actions/setup-go` v7.0.0 `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e`).
- `.gitattributes`: LF line endings everywhere, so the Windows runner sees the
  same bytes as macOS.
- `README.md` with the settled first line, and `TESTING.md`.

Tool recorded under datum 0.13 (ships in nothing built): golangci-lint v2.15.0,
module sum `h1:w94zf8HAYI7hm3164kwKsvZRuqXvY+YxeFoTbu1OLg8=`, invoked as
`go tool -modfile=tools/go.mod golangci-lint run ./...`.

### Tests first

The repository check tests were written against stubs returning no violations
and run before any implementation: 37 failures, each for the expected reason (a
planted violation not reported). The implementation then made them pass.

### Planted violations

Each plant was made in its own scratch copy of the tree, outside the repository.
Output verbatim except for one marker: `go test` prefixes a failing test with
three hyphens, which this file's own prose scan forbids, so that prefix is written
here as `[FAIL]`.

```text
##### planted: duplicate
$ go test ./...
ok  	github.com/TheScottBot/the-front-buttons/internal/repositorychecks	0.256s
exit 0
$ go tool -modfile=tools/go.mod golangci-lint run ./...
internal/bricks/plantedone/plantedone.go:20:21: Magic number: 12, in <condition> detected (mnd)
		if len(trimmed) > 12 {
		                  ^
internal/bricks/plantedtwo/plantedtwo.go:20:21: Magic number: 12, in <condition> detected (mnd)
		if len(trimmed) > 12 {
		                  ^
internal/bricks/plantedone/plantedone.go:17:6: stringscutprefix: HasPrefix + TrimPrefix can be simplified to CutPrefix (modernize)
		if strings.HasPrefix(trimmed, "#") {
		   ^
internal/bricks/plantedtwo/plantedtwo.go:17:6: stringscutprefix: HasPrefix + TrimPrefix can be simplified to CutPrefix (modernize)
		if strings.HasPrefix(trimmed, "#") {
		   ^
internal/bricks/plantedone/plantedone.go:28:2: return with no blank line before (nlreturn)
	return strings.Join(selected, separator) + " (" + strings.Repeat("x", len(selected)) + ")"
	^
internal/bricks/plantedtwo/plantedtwo.go:28:2: return with no blank line before (nlreturn)
	return strings.Join(selected, separator) + " (" + strings.Repeat("x", len(selected)) + ")"
	^
internal/bricks/plantedone/plantedone.go:13:3: missing whitespace above this line (invalid statement above assign) (wsl_v5)
		trimmed := strings.TrimSpace(label)
		^
internal/bricks/plantedone/plantedone.go:23:3: missing whitespace above this line (invalid statement above assign) (wsl_v5)
		selected = append(selected, strings.ToUpper(trimmed))
		^
internal/bricks/plantedone/plantedone.go:25:2: missing whitespace above this line (invalid statement above if) (wsl_v5)
	if len(selected) == 0 {
	^
internal/bricks/plantedtwo/plantedtwo.go:13:3: missing whitespace above this line (invalid statement above assign) (wsl_v5)
		trimmed := strings.TrimSpace(label)
		^
internal/bricks/plantedtwo/plantedtwo.go:25:2: missing whitespace above this line (invalid statement above if) (wsl_v5)
	if len(selected) == 0 {
	^
11 issues:
* mnd: 2
* modernize: 2
* nlreturn: 2
* wsl_v5: 5
exit 1
##### planted: adapter
$ go test ./...
ok  	github.com/TheScottBot/the-front-buttons/internal/repositorychecks	0.324s
exit 0
$ go tool -modfile=tools/go.mod golangci-lint run ./...
internal/bricks/plantedbrick/plantedbrick.go:4:8: blank-imports: a blank import should be only in a main or test package, or have a comment justifying it (revive)
import _ "github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
       ^
1 issues:
* revive: 1
exit 1
##### planted: goos
$ go test ./...
[FAIL] TestRepository_HasNoViolations (0.01s)
    repository_test.go:151: internal/live/planted.go:7: operating-system-check: runtime.GOOS outside a permitted package; inject behaviour through a port instead
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/repositorychecks	0.278s
FAIL
exit 1
$ go tool -modfile=tools/go.mod golangci-lint run ./...
0 issues.
exit 0
##### planted: endash
$ go test ./...
[FAIL] TestRepository_HasNoViolations (0.01s)
    repository_test.go:151: IMPLEMENTATION_LOG.md:36: dash-character: forbidden dash character U+2013; use a comma, colon, semicolon, parentheses or a new sentence
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/repositorychecks	0.200s
FAIL
exit 1
$ go tool -modfile=tools/go.mod golangci-lint run ./...
0 issues.
exit 0
##### planted: readme
$ go test ./...
[FAIL] TestRepository_HasNoViolations (0.01s)
    repository_test.go:151: README.md:1: readme-first-line: first line must be exactly "This is for the lions living in the wiry frames of my peripherals' bodies", found "This is for the lions living in the wiry frames of my peripherals' bodies changed"
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/repositorychecks	0.190s
FAIL
exit 1
$ go tool -modfile=tools/go.mod golangci-lint run ./...
0 issues.
exit 0
##### planted: adapter (rerun after uniq-by-line: false)
$ go tool -modfile=tools/go.mod golangci-lint run ./...
internal/bricks/plantedbrick/plantedbrick.go:4:8: import 'github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter' is not allowed from list 'adapters-only-from-composition-root': an adapter; only cmd/the-front-buttons may import it, everything else uses the ports in internal/platform (depguard)
import _ "github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
       ^
internal/bricks/plantedbrick/plantedbrick.go:4:8: blank-imports: a blank import should be only in a main or test package, or have a comment justifying it (revive)
import _ "github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter"
       ^
2 issues:
* depguard: 1
* revive: 1
exit 1
##### planted: format
$ go tool -modfile=tools/go.mod golangci-lint run ./...
internal/live/planted.go:3:1: File is not properly formatted (gofmt)
func   Badly( )   int { return 1 }
^
internal/live/planted.go:3:1: File is not properly formatted (gofumpt)
func   Badly( )   int { return 1 }
^
internal/live/planted.go:3:1: File is not properly formatted (goimports)
func   Badly( )   int { return 1 }
^
internal/live/planted.go:3:1: exported: exported function Badly should have comment or be unexported (revive)
func   Badly( )   int { return 1 }
^
4 issues:
* gofmt: 1
* gofumpt: 1
* goimports: 1
* revive: 1
exit 1
##### planted: cmdallowed
$ go tool -modfile=tools/go.mod golangci-lint run ./...
0 issues.
exit 0
```

Reading of the results:

| Plant | Check | Result |
|---|---|---|
| en dash in a tracked file | prose scan | Failed as required |
| runtime.GOOS in internal/live | operating system check | Failed as required |
| changed README first line | README check | Failed as required |
| brick importing an adapter | depguard | Hidden on the first run (TF10); failed as required after the fix |
| composition root importing an adapter | depguard exemption | Passed, as required |
| unformatted Go file | gofmt, gofumpt, goimports | Failed as required |
| duplicated block across two brick packages | golangci-lint dupl | **Did not fail** (TF9) |

### Blocked, then resolved

Adding `mibk/dupl` to `tools/go.mod` and running it was refused by the session's
permission classifier. (11 November 2026, later: the author ran
`cd tools && go get -tool github.com/mibk/dupl@v1.1.0` themselves, output
`go: added github.com/mibk/dupl v1.1.0`.)

mibk/dupl v1.1.0 read from the module cache: MIT; default threshold 100 tokens;
walks every `.go` file under the given path, skipping `vendor` directories; always
exits zero, so `scripts/check-duplicate-code.sh` fails on any output. Recorded
under datum 0.13 as a tool that ships in nothing built. golangci-lint's own `dupl`
is disabled so the threshold lives in one place.

The planted duplicate, rerun against the script in the same scratch copy:

```text
##### planted: duplicate (rerun with the standalone mibk/dupl v1.1.0)
$ scripts/check-duplicate-code.sh
Duplicate code found (threshold 100 tokens):
internal/bricks/plantedone/plantedone.go:2-29: duplicate of internal/bricks/plantedtwo/plantedtwo.go:2-29
internal/bricks/plantedtwo/plantedtwo.go:2-29: duplicate of internal/bricks/plantedone/plantedone.go:2-29
exit 1
```

On the real tree the script reports no duplicate code at 100 tokens (and none at
75). Duplicate code linter threshold (Appendix A row): **100 tokens**, dupl's
default, as recommended in section 9 and accepted, now proven against the plant.

### Reproduction report, 11 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64 |
| Go | go1.27.3 darwin/arm64 |
| golangci-lint | v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| `go test -race -count=1 ./...` | `internal/repositorychecks` ok; 49 tests and subtests passed, 0 failed, 0 skipped; every other package has no tests yet |
| `go tool -modfile=tools/go.mod golangci-lint run ./...` | 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Warnings | 0 (golangci-lint prints no warnings with this configuration) |
| Memory and undefined behaviour analysis | race detector, clean |
| Fuzzing | not applicable: no parser of untrusted input yet |
| Tests skipped | none |
| Tests requiring the author | none automated |
| Not run | CI on either runner (needs the author's push); anything on the Dell |
| Outstanding | author gate: reproduction on both machines and CI green on both runners |

### Ignored files (added 11 November 2026, at the author's request)

The author asked for a `.gitignore` that excludes editor, operating system,
build, test and log output properly. Until then the repository checks walked
every file on disk, including an untracked `.idea/` editor directory, while datum
0.8 asks for the scan to cover tracked files.

- `.gitignore` written for editor state (`.idea/`, `.vscode/`, swap and backup
  files), operating system clutter, the root build binary and `bin/` and `dist/`,
  Delve debugger builds, test binaries, coverage and profiles, and logs. Checked
  with `git check-ignore -v` against `.idea/workspace.xml`, `the-front-buttons`,
  `coverage.html` and a `.log` file: each matched its intended line.
- The repository checks now read the root `.gitignore` and skip what it names, so
  the ignore list lives in one place (datum 0.7). They read a deliberate subset of
  git's grammar (single segment globs, a leading slash to anchor, a trailing slash
  for directories) and report any other line as `unsupported-ignore-pattern`
  rather than guess, because a wrong guess would scan ignored files or skip
  tracked ones.
- Tests first: six new tests in `gitignore_test.go`, run before the
  implementation; the four that plant something to skip or reject failed for the
  expected reason, the two that expect scanning to continue passed already, as
  they should.

Planted in scratch copies, output verbatim except the `[FAIL]` marker as above:

```text
##### planted: ignoredfile
$ go test ./internal/repositorychecks/
[FAIL] TestRepository_HasNoViolations (0.01s)
    repository_test.go:151: docs/planted.md:1: dash-character: forbidden dash character U+2013; use a comma, colon, semicolon, parentheses or a new sentence
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/repositorychecks	0.258s
FAIL
##### planted: unsupportedpattern
$ go test ./internal/repositorychecks/
[FAIL] TestRepository_HasNoViolations (0.01s)
    repository_test.go:151: .gitignore:35: unsupported-ignore-pattern: pattern "!keep.log" is outside the subset the repository checks read: negation is not supported
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/repositorychecks	0.183s
FAIL
```

The en dash planted inside `.idea/` was skipped while the one in `docs/` was
reported, and the unsupported negation pattern failed the build.

## TE2: config and the brick schema mechanism

Started 11 November 2026, after rereading datum Part 0 and 0.17.

### Dependency record: go.yaml.in/yaml/v4 v4.0.0 (datum 0.13)

Recorded before adding, from the source at commit
`f2f9563503f11b7a60da0c532175dcc693ce8d5a` read on 11 November 2026.

| Item | Record |
|---|---|
| Behaviour provided | Parsing and emitting YAML, with a node tree that keeps each value's tag, line and column, and comments |
| Why the standard library is insufficient | Go has no YAML package; TD4 chose YAML for hand editing |
| Maintenance | Maintained by the YAML organisation; v4.0.0 released 4 November 2026 and stated stable; v2 and v3 frozen with security fixes |
| Licence | Apache 2.0, with files ported from libyaml under MIT (listed in its NOTICE). Apache 2.0 requires the NOTICE to accompany distributions; nothing is distributed (spec TE10 excludes packaging), and TE10's LICENSE work records it |
| Size | about 20,300 lines of non test Go; no dependencies of its own (`go.mod` requires nothing) |
| Platform support | Pure Go, no cgo; builds on both targets |
| Security | Parses untrusted input. Mitigations: the file is read under a byte bound before parsing (datum 0.10); alias expansion is bounded upstream (tests assert an excessive aliasing error); duplicate keys are rejected by the decoder's own check (11 November 2026, later: corrected after a TE2 test showed v4's unique key option does not apply when decoding into a node tree, finding TF11); values are never decoded by YAML's own typing, only through the derived schema, which checks each node's tag; a fuzz harness runs on the decoder |
| Test strategy | Unit tests through the config package's public decoder, plus the fuzz harness |
| Replacement cost | Contained: only `internal/config` imports it, enforced by depguard, so a swap to v3 or another parser touches one package |

Added with `go get go.yaml.in/yaml/v4@v4.0.0`. depguard now confines it to
`internal/config`; proven against a planted import (below).

### What was built

- **`internal/layout`**: `Position`, `Grid` and the reserved navigation
  positions (bottom left back, bottom right forward), with a grid too small for
  two distinct navigation keys refused. Built here rather than in TE3 because
  TE2's tests reject reserved positions; TE3 builds pages, wrapping and page
  indication on top. The grid is injected, never assumed to be the XL's.
- **`internal/brick`**: schema derivation by reflection from tagged parameter
  structs; the shared `Appearance` struct, embedded and declared once; value
  validation by type, bounds and format; assignment and reading through the
  schema; the generic `Register` function and the registry.
- **`internal/presets`**: preset rows as data (friendly fields declared with the
  same tags, plus `text/template` templates keyed by target field), checked
  against the registry when the table is built; expansion escapes every value
  through `path_segment` or `query`, passes appearance through, and validates
  the result with the target kind's schema.
- **`internal/config`**: the decoder (bounded before parsing, one YAML document,
  no anchors or aliases, no duplicate keys, schema version required, migration
  hook, strict decode of every level through the schemas with values read by
  YAML tag only, positions checked against the grid, every problem collected
  with path and line, values never echoed), the schema driven encoder (preset
  keys written as their preset, zero optional values left out), atomic write
  through a filesystem port, and the store (bounded read, digest to skip
  unchanged content, last good configuration kept on any rejection or read
  failure, deep copies out, saves through the same file and reload path, a
  polling watcher logging each change of result once by reason, never value).
- **`internal/testsupport`**: the shared fake kinds and presets every package's
  tests use.

### The configuration contract, provisional (datum 0.11)

The file shape, as decoded and encoded:

| Level | Fields |
|---|---|
| Top | `schema_version` (required integer), `pages` (required list) |
| Page | `name` (required text, 1 to 24 characters), `keys` (required list) |
| Key | `row`, `column` (required integers); exactly one of `kind` with optional `parameters`, or `preset` with `fields` |
| Parameters and fields | the brick kind's or preset's derived schema |

Bounds, in `internal/config/contract.go`: file 1,048,576 bytes; 64 pages; page
name 24 characters; keys per page at most the grid's positions. The appearance
label bound of 24 characters is provisional pending TE4's legibility gate.

### Design choices made in TE2, for the author to see

These are within the specification's text, but each is a choice the author may
want to revisit:

1. **The secret flag means a credential name.** The specification asks for a
   secret flag in the tags and also forbids secrets in the file. Read together,
   a field marked secret holds the name of a credential in the credential store;
   the editor (TE9) offers write only entry for the secret it names. The schema
   enforces that secret fields are text.
2. **A `format` tag joins the specified tags** (name, requiredness, bounds,
   description, secret). Without it each brick would need its own validator for
   URLs and IDs, which the specification forbids. Formats: `https_url`,
   `identifier`, `digits`, `colour`, one check each in one table.
3. **The watcher polls** once per interval and compares a digest, rather than
   subscribing to filesystem events. That needs no dependency, behaves the same
   on both operating systems, and copes with editors that replace a file rather
   than write to it. The interval is a TE6 value.
4. **YAML anchors and aliases are refused**, which removes alias expansion from
   the attack surface; hand written configuration has no need of them.
5. **Saving from the editor drops hand written comments** in this phase. Keeping
   them is possible through the node tree; it is TE9's decision, as raised in
   11.3.

### Tests first

Each package's tests were written against stubs and run before the
implementation, and every one failed for the expected reason (a stub reporting
nothing, or for presets a stub schema panicking on a nil type, both meaning
not yet implemented): layout 4 tests; brick 11 tests and 17 tag rule subtests;
presets, the whole package; config decoding 19 tests with their subtests; store
and atomic write 14 tests. Runs observed in the terminal on 11 November 2026;
their output was not saved to files.

Two tests found real faults while going green:

- the duplicate key case found TF11, so the decoder checks duplicates itself;
- the save test found that `Store.Current` shared slices with the running
  configuration, so an editor's change would have altered what was running
  without a save or reload. `Current` now returns a deep copy, and a new test
  proves a parameter edit cannot leak in.

### Planted violation for the new check

```text
##### planted: yamloutsideconfig
$ go tool -modfile=tools/go.mod golangci-lint run --enable-only depguard ./...
internal/live/planted.go:4:8: import 'go.yaml.in/yaml/v4' is not allowed from list 'yaml-only-in-config': only internal/config may parse YAML (depguard)
import _ "go.yaml.in/yaml/v4"
       ^
1 issues:
* depguard: 1
```

### Fuzzing

`FuzzDecode` seeds the valid file, an empty file, minimal files, a hexadecimal
integer, an anchor and alias, a duplicate key and unbalanced brackets. Its
property: no panic, and any accepted file survives encode then decode unchanged.
Runs on this Mac: 60 seconds, 1,807,442 executions, no failure; then 30 seconds,
628,325 executions, no failure. The fuzzer's progress output showed pauses at 0
executions per second that coincided with new interesting inputs, which is
consistent with Go's fuzzer pausing to minimise each new input rather than a slow decoder. To rule
that out, inputs at the 1 MiB bound were timed: deep sequence nesting 2.8 ms and
deep mapping nesting 6.9 ms (both refused by the parser), 209,715 empty pages
53 ms, 209,715 duplicate keys 74 ms, one long scalar 6.9 ms.

### Reproduction report, 11 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| `go test -race -count=1 ./...` | brick, config, layout, presets, repositorychecks ok; 163 tests and subtests passed, 0 failed, 0 skipped |
| `go tool -modfile=tools/go.mod golangci-lint run ./...` | 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Fuzz | `go test -run '^$' -fuzz FuzzDecode -fuzztime 30s ./internal/config/`: 628,325 executions, pass |
| Memory analysis | race detector, clean |
| Warnings | 0 |
| Tests requiring the author | none |
| Not run | CI on either runner (needs the author's push); anything on the Dell |
| Outstanding | no author gate for TE2; CI on both runners |

(11 November 2026, later: the author accepted the five TE2 design choices above,
answering "Accept, start TE3".)

## TE3: layout

Started 11 November 2026, after rereading datum Part 0 and 0.17.

### What was built

- **Key numbering** (`keyindex.go`): a position's device key number, and back,
  counted row by row from the top left. The order is **inferred from the
  reference source and unverified** until experiment 1 (Appendix A, DV1); it is
  computed in one function so a correction is one change and one test.
- **Paging** (`paging.go`): a value type that moves forward and back with wrap
  around in both directions, stays put on a single page, has no current page with
  no pages, and on a reload keeps the current page if it still exists or moves to
  the last page if not.
- **Page indication**: the current page's number (counted from one), the page
  count, and the current, previous and next page names, absent rather than
  guessed when names and count disagree. The specification asks that the
  navigation keys show the page name or number; how they draw it is TE4's, which
  has the legibility gate.
- **Arrangement** (`arrange.go`): one page laid out by key number: the navigation
  keys at their reserved positions on every page, including empty pages, each
  brick at its key, every other key empty (drawn blank, not as an error). It
  refuses a brick on a navigation key, off the grid or sharing a key, as a guard
  behind the configuration decoder.

### Tests first

Written against stubs and run before the implementation; output (the go test
failure prefix written as `[FAIL]`, as above):

```text
[FAIL] TestArrange_RefusesBricksOnReservedOffGridOrSharedPositions (0.00s)
[FAIL] TestGrid_NavigationKeys_AreTheLastRowsFirstAndLastKeys (0.00s)
[FAIL] TestArrange_PutsNavigationOnEveryPage_IncludingEmptyOnes (0.00s)
[FAIL] TestArrange_SlotAt_FindsTheSlotForAPressedKey (0.00s)
[FAIL] TestPaging_Resize_KeepsThePageWhenItStillExists_AndClampsWhenItDoesNot (0.00s)
[FAIL] TestPaging_Indication_IsAbsent_WhenNamesDoNotMatchThePageCount (0.00s)
panic: runtime error: index out of range [24] with length 0 [recovered, repanicked]
panic({0x104c850c0?, 0x799d0f626018?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/layout	0.231s
FAIL
```

The stub returned an empty arrangement, so the run stopped at a panic indexing
it; every test reached before that failed on its assertion.

### Reproduction report, 11 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| `go test -race -count=1 ./...` | every package ok; 178 tests and subtests passed, 0 failed, 0 skipped |
| `go tool -modfile=tools/go.mod golangci-lint run ./...` | 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Memory analysis | race detector, clean |
| Fuzz | not applicable: TE3 parses no input |
| Warnings | 0 |
| Not run | CI on either runner (needs the author's push); anything on the Dell |
| Outstanding | no author gate for TE3; the key order assumption awaits experiment 1 |

## TE4: rendering

Started 11 November 2026, after rereading datum Part 0 and 0.17.

### Dependency record: golang.org/x/image v0.47.0 (datum 0.13)

Order of events, stated plainly: the agent ran `go get golang.org/x/image@v0.48.0`
before writing this record, which reverses the order datum 0.13 requires. It then
chose v0.47.0 instead, because v0.48.0 was published two days earlier (9 November
2026) and a release that new has had little exposure; `go mod tidy` removed the
unused requirement, so nothing depended on it. This record is written before any
code imports the package.

| Item | Record |
|---|---|
| Behaviour provided | TrueType and OpenType parsing and glyph rasterising (`font/opentype`, `font/sfnt`), the Go fonts (`font/gofont`), and image scaling (`draw`) |
| Why the standard library is insufficient | The standard library draws rectangles and images but has no font rasteriser and no quality image scaler |
| Maintenance | Maintained by the Go team under the Go project; tagged releases monthly; v0.47.0 published 9 October 2026 |
| Licence | BSD 3 clause (Go authors); the Go fonts under the same terms (Bigelow and Holmes) |
| Size | about 35,000 lines across the packages used, most of it the embedded Go font data |
| Platform support | Pure Go in the packages used; no cgo and no `unsafe` in `font/opentype`, `font/sfnt` or `draw` (searched) |
| Security | Parses font files. The only fonts parsed are vendored ones (the Go fonts, compiled in, and the icon font TD5 vendors with a digest), never fonts from configuration or the network. User icons are PNG, decoded by the standard library under byte and pixel bounds |
| Test strategy | Golden image tests through the render package, plus unit tests of text fitting |
| Replacement cost | Contained to `internal/render`, enforced by depguard |
| Requires | `golang.org/x/sys` v0.49.0 and `golang.org/x/text` v0.43.0, Go project modules |

### Font choice (datum 0.13)

Go Medium for labels and Go Bold for status words and navigation, from
`golang.org/x/image/font/gofont`. Reasons: designed for legibility at small
sizes with clearly distinguished letter forms (the Go fonts were drawn so that
1, l and I differ), compiled into the binary so nothing is read from disk, and
licensed with the package. Whether they are legible on the physical deck at arm's
length is the TE4 author gate.

### What was built

- **`internal/render`**: the renderer and the only drawing code. Key layout in
  hundredths of the key width, so it follows the model table's key size. A key
  shows an icon (bundled, user PNG, or a missing icon marker) and a label; a live
  key shows a status glyph, the label and the status word; a key in error shows
  an error glyph, the label and the reason category, and error outranks status.
  Navigation keys show a chevron, the current page as number of count, and the
  name of the page they lead to; blank keys are plain background. Labels and
  words that do not fit step down through smaller sizes before being shortened
  with an ellipsis. Text is black or white, whichever contrasts more with the
  background; a status colour is used only where it reaches 3 to 1 against the
  background and otherwise falls back to the text colour.
- **Status glyphs**, drawn as vector shapes: passed a tick, failed a cross,
  running three dots, queued an hourglass, cancelled a circle with a slash,
  unknown a question mark, error a triangle with an exclamation mark. Words:
  PASSED, FAILED, RUNNING, QUEUED, CANCELLED, UNKNOWN.
- **User icons**: PNG only, refused over 1 MiB before reading and over 512
  pixels a side from the header before decoding.
- **`internal/render/tablericons`**: the vendored Tabler icon font as an
  `IconSet`, with names read from the vendored stylesheet at load (5,000 or
  more). `PROVENANCE.md` records the three files' digests; the prose scan's
  vendored list names them.
- **Shared golden helper** in `internal/testsupport`, used by both render
  packages.
- **depguard** confines `golang.org/x/image` to `internal/render` (and the shared
  test support's golden helper); **CI** verifies the linter configuration first
  (TF12).

### Decisions taken by the agent under DV2

Each is recorded here because the author is away; each can be reversed.

1. **Downloading the icon font.** The author's TD5 answer ("Yes, use the font")
   was taken as approval to download it. Version 3.49.0 (5 October 2026) was
   chosen over the newest, 3.51.0 (8 November), so the release had more than two
   weeks of exposure. Files and digests are in `PROVENANCE.md`.
2. **x/image v0.47.0** rather than v0.48.0, for the same reason; `go mod tidy`
   twice reselected the newest and was pinned back each time, with `x/sys`
   v0.49.0 and `x/text` v0.43.0 to match.
3. **Status words, glyph shapes and colours** as listed above, and the error
   state outranking a status.
4. **Navigation key design**: both keys show the current page number, and each
   names the page it leads to.
5. **Two linter relaxations, each scoped and explained in the configuration**:
   `mnd` is off for `internal/render/glyphs.go`, whose numbers are outline
   coordinates; `exhaustruct_v5` ignores `render.Appearance`, whose zero values
   mean absent by design.

Every size and position is provisional until the TE4 author gate.

### Tests first

Written against stubs and run before the implementation (output with the go test
failure prefix written as `[FAIL]`):

```text
[FAIL] TestRender_ProducesAKeySizedImage (0.00s)
[FAIL] TestParseColour_AcceptsHashSixHexDigitsOnly (0.00s)
[FAIL] TestStatusWord_IsDistinctForEveryStatus (0.00s)
[FAIL] TestRenderNavigation_MatchesGoldenImages (0.00s)
[FAIL] TestDecodeUserIcon_RefusesAnythingButPNG (0.00s)
[FAIL] TestFitText_ShortensALongLabelToFit_WithAnEllipsis (0.00s)
[FAIL] TestNewRenderer_RefusesImplausibleKeySizes (0.00s)
[FAIL] TestDecodeUserIcon_RefusesBeforeDecoding_WhenDimensionsExceedTheBound (0.00s)
[FAIL] TestDecodeUserIcon_RefusesFilesOverTheByteBound (0.00s)
[FAIL] TestDecodeUserIcon_AcceptsASmallPNG (0.00s)
[FAIL] TestRender_MatchesGoldenImages (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/user_icon (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/light_background (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/error_state (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/long_label (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/label_with_bundled_icon (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/status_queued (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/status_in_progress (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/status_cancelled (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/status_unknown (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/missing_icon (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/status_success (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/label_only (0.00s)
    [FAIL] TestRender_MatchesGoldenImages/status_failure (0.00s)
[FAIL] TestForegroundFor_MeetsWCAGContrastOnEveryBackground (0.00s)
[FAIL] TestRender_EveryStatusIsDistinguishableWithoutColour (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/render	0.216s
FAIL
```

The icon set's tests were likewise run against a stub first: the load and draw
tests failed, and the two that expect refusals passed already, as they should.

The first golden images, inspected as a contact sheet before being trusted,
showed two faults that were fixed and regenerated: thin seams where a stroke met
its round end (opposite polygon windings cancelling in the rasteriser; every
solid shape is now wound one way), and CANCELLED shortened to CANCELL with an
ellipsis (status words now step down through smaller sizes like labels). The
golden images are therefore a record of what the renderer draws as reviewed by
the agent, not of what the author has accepted; the TE4 gate is the author's.

### Planted violations for the new checks

```text
##### planted: imageoutsiderender
$ go tool -modfile=tools/go.mod golangci-lint run --enable-only depguard ./...
internal/live/planted.go:4:8: import 'golang.org/x/image/draw' is not allowed from list 'image-library-only-in-render': only internal/render draws (specification section 2) (depguard)
import _ "golang.org/x/image/draw"
       ^
1 issues:
* depguard: 1

##### planted: misspelt linter setting (the agent's own error, caught before the fix)
$ go tool -modfile=tools/go.mod golangci-lint config verify
jsonschema: "linters.settings.exhaustruct_v5" does not validate with "/properties/linters/properties/settings/properties/exhaustruct_v5/$ref/additionalProperties": additional properties 'exclude' not allowed
The command is terminated due to an error: the configuration contains invalid elements
exit 3
```

### Reproduction report, 11 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| `go test -race -count=1 ./...` | every package ok; 209 tests and subtests passed, 0 failed, 0 skipped; 21 golden images |
| `go tool -modfile=tools/go.mod golangci-lint config verify` | valid |
| `go tool -modfile=tools/go.mod golangci-lint run ./...` | 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Memory analysis | race detector, clean |
| Fuzz | not run in TE4: the only new parser input is a user PNG, decoded by the standard library under byte and pixel bounds checked first; a fuzz target for `DecodeUserIcon` is a candidate if the author wants one |
| Warnings | 0 |
| **Author gate, outstanding** | legibility on the physical deck at arm's length in the dock, under normal desk lighting, on both machines. Not claimed |
| Not run | CI on either runner; anything on the Dell |

## TE5: deck protocol

Started 11 November 2026, after rereading datum Part 0 and 0.17. Every value is
read from source (section 3) and **not confirmed on hardware** (DV1).

### What was built

- **Model table** (`internal/deck/model.go`): three rows, one per XL product ID
  (`0x006C`, `0x008F`, `0x00BA`), sharing one protocol: 4 by 8 keys, 96 pixel
  JPEG images flipped on both axes at quality 100, 1024 byte output reports with
  an 8 byte image header, 32 byte feature reports, 36 byte input reports with a 4
  byte header, and every report ID, command and text offset. The protocol code
  reads only the table.
- **Pure encoders**: reset key stream, reset, brightness, key image pages, and
  `EncodeKeyImage`, which flips an upright image as the model requires and
  encodes it as JPEG. Only this function knows the deck draws images flipped; the
  renderer and the editor preview stay upright.
- **Pure decoders**: key states from an input report (exact length only, header
  not interpreted, any non zero key byte pressed as in the reference), key
  changes between two reports, and the serial number and firmware text from
  feature reports (printable ASCII to the first NUL, other bytes shown as ?).

### Decisions taken by the agent under DV2

1. **Refuse, do not clamp.** The reference clamps brightness into 0 to 100 and
   admits key 32 (TF6). Here an out of range brightness, a key the model lacks,
   an empty image or an oversized one is refused with `ErrOutOfRange` (datum
   0.10: refuse rather than truncate), so a caller's bug shows.
2. **A key image is bounded at 64 pages** (65,024 bytes). A worst case noisy key
   at quality 100 encodes well within it (tested).
3. **The input report header is not checked.** Its four bytes are unknown until
   experiment 1, and the reference never reads them; checking a guessed report ID
   would refuse every real report if the guess were wrong. Experiment 1 records
   the header and a later change can check it.
4. **Brightness is 32 bytes**, not the reference's accidental 33 (TF6).
   Experiment 1 confirms the device accepts it.
5. **The device ports are TE6's.** TE5 is bytes in and bytes out; the port
   interfaces that move those bytes are designed with the adapter in TE6.

### Tests first

Written against stubs and run first (go test failure prefix written as
`[FAIL]`):

```text
[FAIL] TestEncodeKeyImagePages_WritesExactReferenceHeaders (0.00s)
[FAIL] TestDecodeFeatureText_ReplacesUnprintableBytes_AndRefusesShortReports (0.00s)
[FAIL] TestDecodeKeyStates_ReadsOneByteperKeyAfterTheHeader (0.00s)
[FAIL] TestLookupModel_FindsEveryXLVariant_WithTheSameProtocolFacts (0.00s)
[FAIL] TestDecodeKeyStates_DecodesEveryKeyPressedAndReleased (0.00s)
[FAIL] TestDecodeKeyStates_TreatsAnyNonZeroByteAsPressed_AsTheReferenceDoes (0.00s)
[FAIL] TestEncodeKeyImage_FlipsBothAxes_SoTheDeckShowsItUpright (0.00s)
[FAIL] TestEncodeBrightness_RefusesRatherThanClampsOutOfRangeValues (0.00s)
[FAIL] TestEncodeReset_MatchesTheReference (0.00s)
[FAIL] TestEncodeKeyImagePages_RefusesKeysThatDoNotExist_AndEmptyOrOversizedImages (0.00s)
[FAIL] TestEncodeKeyImagePages_FitsAnExactPageInOnePage (0.00s)
[FAIL] TestEncodeResetKeyStream_MatchesTheReference (0.00s)
[FAIL] TestEncodeKeyImage_RefusesAnImageOfTheWrongSize (0.00s)
[FAIL] TestEncodeKeyImage_FitsWithinTheImageBound (0.00s)
[FAIL] TestDecodeKeyStates_RefusesShortAndOversizedReports (0.00s)
[FAIL] TestEncodeBrightness_MatchesTheReference_InThirtyTwoBytes (0.00s)
[FAIL] TestDecodeFeatureText_ReadsTheSerialAndFirmwareAsTheReferenceDoes (0.00s)
[FAIL] TestKeyChanges_ListsOnlyKeysWhoseStateChanged (0.00s)
[FAIL] TestLookupModel_CarriesTheXLFactsReadFromSource (0.00s)
[FAIL] FuzzDecodeKeyStates (0.00s)
    [FAIL] FuzzDecodeKeyStates/seed#0 (0.00s)
    [FAIL] FuzzDecodeKeyStates/seed#1 (0.00s)
    [FAIL] FuzzDecodeKeyStates/seed#2 (0.00s)
    [FAIL] FuzzDecodeKeyStates/seed#3 (0.00s)
    [FAIL] FuzzDecodeKeyStates/seed#4 (0.00s)
[FAIL] FuzzDecodeSerialNumber (0.00s)
    [FAIL] FuzzDecodeSerialNumber/seed#0 (0.00s)
    [FAIL] FuzzDecodeSerialNumber/seed#1 (0.00s)
    [FAIL] FuzzDecodeSerialNumber/seed#2 (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/deck	0.213s
FAIL
```

Byte fixtures are derived by hand from `StreamDeckXL.py` at commit `064f6cf`, as
the test file's comment shows: for a 2,000 byte image, page 0's header is
`02 07 05 00 F8 03 00 00` (1,016 bytes, not last) and page 1's is
`02 07 05 01 D8 03 01 00` (984 bytes, last). A correction from experiment 1
changes one table row and one fixture.

### Fuzzing

```text
FuzzDecodeKeyStates: elapsed: 30s, execs: 19180236 (0/sec), new interesting: 0 (total: 6)
FuzzDecodeSerialNumber: elapsed: 30s, execs: 18918151 (0/sec), new interesting: 1 (total: 34)
```

No failures. CI now runs both deck fuzz targets for 30 seconds on each runner.

### Reproduction report, 11 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| `go test -race -count=1 ./...` | every package ok; 240 tests and subtests passed, 0 failed, 0 skipped |
| `golangci-lint config verify`, then `run ./...` | valid; 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Fuzz | as above |
| Memory analysis | race detector, clean |
| Warnings | 0 |
| Not run | CI on either runner; anything on the Dell; anything on a deck |
| Outstanding | every protocol value awaits experiment 1 (DV1); no author gate for TE5 |

## TE6: transport, watcher and daemon loop

Started 11 November 2026, after rereading datum Part 0 and 0.17.

### Dependency record: github.com/sstallion/go-hid v0.15.0 (datum 0.13)

Written before the dependency is added. Facts from section 4 of this record.

| Item | Record |
|---|---|
| Behaviour provided | Enumerating, opening, reading and writing USB HID devices, through hidapi |
| Why the standard library is insufficient | Go's standard library has no USB or HID access; the platform APIs (IOKit, the Windows HID class driver) are what hidapi wraps |
| Maintenance | Active, not archived; v0.15.0 tagged 23 May 2025, bundling hidapi 0.15.0, the current upstream release. The alternative, karalabe/hid, is archived (TD3) |
| Licence | BSD 2 clause for the wrapper; hidapi taken under its BSD style option |
| Size | the wrapper plus bundled hidapi C sources for each platform |
| Platform support | macOS through IOKit and Windows through the HID class driver, both via cgo; the only cgo in this module, confined to `internal/deck/hidtransport` |
| Security | Device bytes are untrusted: every input report goes through the bounded, fuzzed decoder (TE5). Open issues TF5 (Enumerate can panic on macOS on a bad string) and #15 (transient set report failures on macOS) are guarded: enumeration recovers a panic and reports it, and a failed write is treated as a disconnection, after which the watcher reconnects |
| Test strategy | The adapter runs the same `DeckDevice` and `DeckWatcher` contract suites as the test doubles; on a machine with no deck attached the device half is skipped with that reason, and running it on the deck is part of the TE6 author gate |
| Replacement cost | Contained to `internal/deck/hidtransport` behind the two ports |

Added with `go get github.com/sstallion/go-hid@v0.15.0`, after this record.

### What was built

- **Ports** (`internal/deck/ports.go`): `DeckDevice` moves bytes to and from one
  open deck and understands no protocol; `DeckWatcher` lists attached decks and
  opens one. Every error after a deck goes away wraps `ErrDisconnected`; a deck
  that is listed but will not open is `ErrDeckBusy`.
- **Driver** (`internal/deck/driver.go`): `Deck` composes the pure protocol with a
  `DeckDevice`: open resets the key stream, then key images, brightness, reset,
  key changes from input reports, and the serial number.
- **Connection diffing**: arrivals and departures by path, and filtering to
  supported models.
- **Adapter** (`internal/deck/hidtransport`): go-hid with the bundled hidapi, the
  only cgo package. Enumeration recovers the TF5 panic and reports it; every call
  on a device is serialised; closing twice never reaches hidapi twice; a timed
  read that times out returns nothing rather than an error.
- **Contract suites** (`internal/testsupport/deckcontract.go`): one suite per
  port, run by the doubles and by the adapter.
- **Shared doubles** (`internal/testsupport/deckdoubles.go`, `buttons.go`): a fake
  deck that records writes, replays key presses and can vanish mid repaint; a
  fake watcher with attach, detach and busy; a fake pressable button kind.
- **Brick behaviour** (`internal/brick/behaviour.go`): `Pressable`,
  `ReasonCategorised` with `ReasonCategoryOf`, and `AppearanceOf`.
- **Daemon loop** (`internal/daemon`): one goroutine owns all state; a reader
  goroutine per deck and a goroutine per press report back with a generation
  number so stale events are ignored. It polls for decks, connects with a full
  repaint and brightness, pages with wrap around, presses bricks with a timeout,
  shows a press failure on the key by reason category, repaints on reload, drops
  a deck that fails a read or write and repaints the next one in full, logs each
  connection state once, and resets and closes the deck on shutdown.

### Experiment 4, local half

The adapter's watcher contract runs against the real hidapi through cgo on this
Mac: go-hid v0.15.0 and its bundled hidapi build with Apple clang 21 and
enumerate with no deck attached. **Run**, 11 November 2026. The Windows build and
the runner builds remain (CI, after the author pushes).

### Decisions taken by the agent under DV2

1. **A `daemon` package** holds the main loop. The specification's layout has no
   home for it, and `cmd/` must stay composition only.
2. **The port names stay `DeckDevice` and `DeckWatcher`**, as the specification's
   ports table names them, with the linter's stutter warning suppressed and the
   reason given.
3. **Busy or gone**: when a listed deck will not open, the adapter enumerates
   again; still listed means busy (held by other software), gone means
   disconnected.
4. **The deck is reset on shutdown**, so keys that no longer work are not left
   showing.
5. **The composition root is wired in TE7**, with the configuration location
   port, rather than twice. Poll interval, read timeout and brightness are set
   there.

### Tests first

Each piece was written against stubs and run first (go test prefixes written in
brackets, as above). The deck driver and diffing (the contract tests on the
doubles passed from the start, because the doubles were written with the suite):

```text
[FAIL] TestDeck_SerialNumber_IsReadFromItsFeatureReport (0.00s)
[FAIL] TestSupportedConnections_DropsDevicesOutsideTheModelTable (0.00s)
[FAIL] TestDeck_SetBrightness_SendsTheFeatureReport (0.00s)
[FAIL] TestOpenDeck_RefusesAnUnsupportedModel (0.00s)
[FAIL] TestOpenDeck_ResetsTheKeyStreamBeforeAnythingElse (0.00s)
[FAIL] TestDeck_ReadKeyChanges_ReportsPressAndRelease (0.00s)
[FAIL] TestDiffConnections_ReportsArrivalsAndDepartures_ByPath (0.00s)
[FAIL] TestDeck_ReportsDisconnection_WhenTheDeckGoesAway (0.00s)
[FAIL] TestDeck_SetKeyImage_WritesEveryPageForTheKey (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/deck	0.356s
FAIL
```

The adapter, against a stub watcher:

```text
[SKIP] TestDevice_PassesTheDeckDeviceContract_OnAnAttachedDeck (0.00s)
[FAIL] TestWatcher_PassesTheDeckWatcherContract_WithOrWithoutADeck (0.00s)
    [PASS] TestWatcher_PassesTheDeckWatcherContract_WithOrWithoutADeck/lists_only_supported_models (0.00s)
    [FAIL] TestWatcher_PassesTheDeckWatcherContract_WithOrWithoutADeck/refuses_to_open_a_deck_that_is_not_attached (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered, repanicked]
panic({0x10271b670?, 0x102759750?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport	0.167s
FAIL
```

The brick behaviour:

```text
[FAIL] TestAppearanceOf_ReadsTheEmbeddedAppearance (0.00s)
[FAIL] TestReasonCategoryOf_UsesTheErrorsOwnCategory_EvenWhenWrapped (0.00s)
[FAIL] TestAppearanceOf_ReportsParametersWithoutAnAppearance (0.00s)
[FAIL] TestReasonCategoryOf_FallsBackToFailed_ForAPlainError (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/brick	0.153s
FAIL
```

The daemon, against a stub loop that did nothing:

```text
[FAIL] TestDaemon_PagesForwardAndBack_WithWrapAround (0.00s)
[FAIL] TestDaemon_RepaintsEveryKey_AndSetsBrightness_WhenADeckConnects (0.00s)
[FAIL] TestDaemon_SurvivesDisconnectionMidRepaint_AndRepaintsTheReturningDeck (0.00s)
[FAIL] TestDaemon_RepaintsWhenTheConfigurationReloads (0.00s)
panic: deadlock: all goroutines in bubble are blocked [recovered, repanicked]
panic({0x104d3f548?, 0x72e53eda81b0?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/daemon	0.178s
FAIL
```

One daemon test was corrected before it went green: it assumed the deck always
connects after the first configuration arrives, but either order is valid, so it
now measures a reload as one more repaint of every key.

### Reproduction report, 11 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; Apple clang 21.0.0; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean, including the cgo adapter |
| `go test -race -count=1 ./...` | every package ok; 275 tests and subtests passed, 0 failed, 1 skipped |
| Skipped | `TestDevice_PassesTheDeckDeviceContract_OnAnAttachedDeck`: needs a deck and `THE_FRONT_BUTTONS_DECK_TESTS=1`; part of the TE6 author gate |
| `golangci-lint config verify`, then `run ./...` | valid; 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Memory analysis | race detector, clean; the daemon tests run three times in a row clean |
| Warnings | 0 |
| **Author gate, outstanding** | on both machines, pressing keys, paging both ways, and switching the dock ten times without restarting. The automated ten switch test passes against doubles; that is not the gate and is not claimed as it |
| Not run | the device contract; experiments 1 and 2; CI; anything on the Dell |

## TE7: platform ports and simple bricks

Started 11 November 2026, after rereading datum Part 0 and 0.17.

### What was built

- **Ports** (`internal/platform`): `URLOpener`, `ApplicationLauncher` and
  `ConfigLocation`, the product's names from TD1 written once, and
  `CheckLaunchTarget`, the one rule every launch adapter applies before running
  anything (refuses empty, over 4,096 bytes, a leading hyphen, or a control
  character).
- **Adapters** (`internal/platform/systemadapter`): the decisions in an untagged
  file tested on every machine, with the real runner and shell call in build
  tagged files. macOS runs `/usr/bin/open -u URL`, `open -b BUNDLE_ID` or
  `open -a NAME` as discrete arguments with no shell, discarding output that could
  echo a URL. Windows calls `ShellExecuteW` on a thread with COM initialised,
  never through cmd. One `ConfigLocation` adapter serves both, from
  `os.UserConfigDir`. `ShutdownSignals` adds terminate to interrupt.
- **Bricks**: `open_url` (an https URL) and `open_application` (an identifier the
  adapter interprets), sharing their press behaviour in `internal/bricks/launching`,
  showing `open_failed` or `refused` on the key. `internal/bricks/allbricks`
  registers every kind.
- **Built in presets** (`internal/presets/builtin.go`): `discord_channel`,
  `teams_chat`, `teams_chat_by_id`, `teams_channel`, `teams_private_channel`,
  `teams_shared_channel`, `dndbeyond_character`, all over `open_url`, each tested
  against the documented URL. A `uri_component` template function encodes as
  Microsoft's Teams examples do.
- **Contract suites and doubles** for both launch ports, run by all four
  adapters (with process spies, so nothing launches) and by the doubles.
- **The composition root** (`cmd/the-front-buttons`): builds the model's grid,
  the registry, the presets, the icon set and renderer, the configuration store
  at the per machine path (or `-config`), the HID watcher, the launch adapters
  and the daemon, logs the start up facts, and stops cleanly on interrupt or
  terminate.

### Decisions taken by the agent under DV2

1. **DV3**: Windows packaged applications launch through `shell:AppsFolder`
   rather than COM, because the COM GUIDs were never read from a source.
2. **`open_url` accepts https only**, matching TD13's https presets. An
   `msteams://` desktop link can be added if experiment 3 shows its form.
3. **macOS identifiers**: reverse DNS (two or more dot separated parts, no
   spaces) is a bundle identifier, unless it ends in `.app`; anything else is an
   application name.
4. **Teams encoding**: channel and chat IDs and names are encoded as
   `encodeURIComponent` does (a colon as `%3A`, an at sign as `%40`, a space as
   `%20`), as Microsoft's examples show; the participants list is query encoded,
   so its commas become `%2C`. Whether Teams accepts the encoded commas is
   experiment 3's to confirm.
5. **Three Teams channel presets** (standard, private, shared) rather than one
   with a type field, keeping presets as data with no enumerated field type.
6. **No Roll20 preset**, as TD13 settled, until the author copies one real Launch
   Game link.
7. **Shared launch behaviour** in `internal/bricks/launching`, after the
   duplicate code check found the two bricks repeating it.
8. **Logs go to standard error only** for now. The rotating log file in the state
   folder (specification section 7) is not built yet.
9. **Two earlier tests were edited mechanically**: `brick.Ports{}` literals in
   `internal/brick/registry_test.go` (TE2) and `internal/daemon/daemon_test.go`
   (TE6) now name the two new nil ports, because the struct gained fields. No
   assertion changed.

### Tests first

Each piece was written against stubs and run first (go test prefixes written in
brackets). The launch target rule:

```text
[FAIL] TestCheckLaunchTarget_RefusesTargetsACommandCouldMisread (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/platform	0.207s
FAIL
```

The adapters (the stubs ran nothing, so the first assertion on a recorded
command panicked on an empty record):

```text
ok  	github.com/TheScottBot/the-front-buttons/internal/platform	0.201s
[FAIL] TestMacOSURLOpener_RunsOpenWithTheURLAsOneArgument_AndNoShell (0.00s)
panic: runtime error: index out of range [0] with length 0 [recovered, repanicked]
panic({0x10064e078?, 0xa2167a4000?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter	0.177s
FAIL
```

The bricks and built in presets:

```text
[FAIL] TestOpenApplication_RefusesToConstructWithoutALauncher (0.00s)
panic: reflect: New(nil) [recovered, repanicked]
panic({0x10049cb08?, 0x100468b00?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/bricks/openapplication	0.242s
[FAIL] TestOpenURL_OpensItsURLWhenPressed (0.00s)
panic: reflect: New(nil) [recovered, repanicked]
panic({0x1042e0b00?, 0x1042acb00?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/bricks/openurl	0.239s
[FAIL] TestBuiltIn_ExpandsEveryPresetIntoTheDocumentedURL (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/presets	0.223s
FAIL
```

The shutdown signals:

```text
[FAIL] TestShutdownSignals_IncludeInterruptAndTerminate (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/platform/systemadapter	0.226s
FAIL
```

### Smoke run of the built binary

Built with `go build ./cmd/the-front-buttons`, started with `-config` pointing at a
two key sample configuration, interrupted after two seconds, no deck attached:

```text
{"time":"2026-11-11T23:59:35.477617Z","level":"INFO","msg":"starting","product":"The Front Buttons","version":"v0.0.0-20261111235139-53a6b82eddfa+dirty","config":"SCRATCH/config.yaml","kinds":["open_application","open_url"],"presets":["discord_channel","dndbeyond_character","teams_channel","teams_chat","teams_chat_by_id","teams_private_channel","teams_shared_channel"]}
{"time":"2026-11-11T23:59:35.478899Z","level":"INFO","msg":"configuration reload","result":"accepted"}
{"time":"2026-11-11T23:59:35.483418Z","level":"INFO","msg":"deck","state":"waiting"}
exit 0
```

It starts, accepts the configuration including a preset, waits for a deck, and
exits 0 on interrupt.

### Reproduction report, 12 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; Apple clang 21.0.0; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| Windows type check | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet` over every package except the cgo adapter and the composition root that imports it: clean |
| `go test -race -count=1 ./...` | every package ok; 316 tests and subtests passed, 0 failed, 1 skipped (the TE6 device contract) |
| `golangci-lint config verify`, then `run ./...` | valid; 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens (after the fix above) |
| Memory analysis | race detector, clean |
| Warnings | 0 |
| **Author gate, outstanding** | every preset and both bricks working on each machine; with DV3, launching new Teams by its app ID on the Dell |
| Not run | experiment 3; anything on the Dell; CI |

## TE8: live state and pipeline_status

Started 12 November 2026, after rereading datum Part 0 and 0.17.

### Dependency record: github.com/zalando/go-keyring v0.2.8 (datum 0.13)

Written before the dependency is added (TD12). Facts from evaluation 6.3 and
`gh api`, read 12 November 2026: tag v0.2.8 is commit
`2fb288e584191da8306e42b9f86a697742fca71e`, 23 March 2026.

| Item | Record |
|---|---|
| Behaviour provided | Reading and writing named secrets in the macOS Keychain and the Windows Credential Manager |
| Why the standard library is insufficient | Go has no access to either store; hand writing both would reimplement Keychain access (TD12) |
| Maintenance | Maintained; last commit 24 July 2026 |
| Licence | MIT |
| Requires | `github.com/danieljoos/wincred` v1.2.3 (Windows), `github.com/godbus/dbus/v5` v5.2.2 (Linux only, never built here), `golang.org/x/sys` |
| Platform support | macOS without cgo, by running `/usr/bin/security -i` with the secret written to its standard input, never its arguments; Windows through `wincred` and `CredWriteW`/`CredReadW` |
| Security | The secret never enters configuration, logs or the editor's responses (specification section 2). On macOS it passes through a child process's standard input, which other users cannot read. Windows refuses secrets over 2,560 bytes; the adapter bounds secrets below that first |
| Test strategy | The adapter runs the one `CredentialStore` contract suite only with the author's consent (`THE_FRONT_BUTTONS_CREDENTIAL_TESTS=1`), because it writes to the real store; the in memory double runs it always |
| Replacement cost | Contained to one file of `internal/platform/systemadapter` behind the port |

Added with `go get github.com/zalando/go-keyring@v0.2.8`, after this record. It
brought `github.com/danieljoos/wincred` v1.2.3 and `github.com/godbus/dbus/v5`
v5.2.2 as stated.

### What was built

- **Credentials** (`internal/platform/credentials.go`,
  `internal/platform/systemadapter/credentials.go`): the `CredentialStore` port;
  one rule every adapter applies (a name of 1 to 64 bytes, a secret of 1 to 2,048
  bytes, no control characters, and errors that never contain the secret); the
  go-keyring adapter filing secrets under the product's service name; an in
  memory double; one contract suite.
- **Diagnostics** (`internal/diagnostics`): named counters and readings, safe
  across goroutines, read by snapshot. Counted so far: key presses by kind, brick
  errors by kind and reason, deck connects, disconnects and busy, pipeline polls
  by result, and GitHub's remaining rate limit.
- **The pipeline port** (`internal/pipeline`): provider neutral run states, the
  query, the result with GitHub's requested wait, and `PollError` carrying a
  reason category and a retry wait.
- **The GitHub Actions client** (`internal/githubactions`): the documented
  endpoint, headers and API version (TD8); the query checked with the same
  format functions the configuration uses before any request; the token read
  from the credential store per poll and never put in an error; the ETag sent
  back and a 304 answered from the cached run; rate limiting recognised as GitHub
  describes it, with retry after, reset time and the one minute floor; every
  other failure categorised; the response bounded before parsing (DV4 for its
  extra fields); TD10's state mapping and press target.
- **New formats** (`internal/brick/field.go`): `repository`, `token` and
  `branch`, with `CheckFormat` as the one check per format for schema and adapters
  alike.
- **Live state** (`internal/brick/live.go`, `internal/live`): `LiveSource`,
  `LivePressable` and `LiveState`; the hub running one watch per identity,
  caching each identity's latest state, restarting only changed identities on
  reload, and stopping cleanly.
- **`pipeline_status`** (`internal/bricks/pipelinestatus`): repository, workflow,
  branch, credential (a secret field naming the stored token), interval (30 seconds
  to an hour, a minute if left out) and API base URL (api.github.com if left out);
  publishes each run state, waits longer when GitHub asks, opens the run from its
  latest state on press, and keeps its identity across a relabel.
- **The daemon** now runs the hub (with a non blocking change set, so the hub and
  the loop can never deadlock), repaints live keys on change, draws cached state
  the moment a page is shown, presses live keys with their latest state, logs each
  live error reason change once, names each configured credential that is
  missing, and counts presses, errors and connections.
- **The composition root** wires the keyring store, the GitHub client with a 20
  second request bound, and the counters.

### Decisions taken by the agent under DV2

1. **DV4**: GitHub's response fields beyond those read are ignored, not refused.
2. **The token is read from the store on every poll** rather than cached in
   memory, so a credential updated in the editor (TE9) takes effect on the next
   poll with no restart, and the secret is held only for the request.
3. **A press failure outranks a live error on the key**, as the newer news.
4. **The secret field of `pipeline_status` is required.** GitHub allows
   unauthenticated reads of public repositories, but at 60 requests an hour; a
   required token keeps the rate limit arithmetic of TD8 true.
5. **`startup_failure`**, a conclusion seen in GitHub's newer documentation and
   not in TD10's list, falls to unknown, as TD10 settled for anything
   unrecognised.

### Tests first

Each piece was written against stubs and run first (go test prefixes written in
brackets). The credential rule:

```text
[FAIL] TestCheckCredential_RefusesBadNamesAndSecrets (0.00s)
panic: runtime error: invalid memory address or nil pointer dereference [recovered, repanicked]
panic({0x100ab29a8?, 0x100aed000?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/platform	0.215s
FAIL
```

The credential adapter has no failing run of its own: its contract runs only with
consent, so it was skipped against the stub as it is now; the double's run of the
same suite passed, because the double was written with it.

The counters:

```text
[FAIL] TestCounters_SnapshotIsACopy (0.00s)
[FAIL] TestCounters_IncrementAndSet_AreSafeFromManyGoroutines (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/diagnostics	0.155s
FAIL
```

The GitHub client and mapping:

```text
[FAIL] TestPressTarget_OpensTheLastFailure_WhileAFixIsRunning (0.00s)
[FAIL] TestStateOf_MapsEveryGitHubStatusAndConclusionAsTD10Settled (0.00s)
[FAIL] TestLatestRun_SendsTheDocumentedRequest (0.00s)
panic: runtime error: index out of range [-1] [recovered, repanicked]
panic({0x1047b8760?, 0x7c7157bf2168?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/githubactions	0.281s
FAIL
```

The formats:

```text
[FAIL] TestCheckFormat_RepositoryWorkflowAndBranch (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/brick	0.314s
FAIL
```

The live hub:

```text
[FAIL] TestHub_RunsOneWatchPerIdentity_SoTwoKeysOnOneWorkflowMakeOnePoll (0.00s)
[FAIL] TestHub_NotifiesEachPublishedState_AndStopWaitsForEveryWatch (0.00s)
[FAIL] TestHub_KeepsTheLatestStatePerIdentity (0.00s)
[FAIL] TestHub_RestartsOnlyChangedSources_AndStopsRemovedOnes (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/live	0.195s
FAIL
```

`pipeline_status`:

```text
[FAIL] TestPipelineStatus_IsALiveSourceAndALivePressable (0.00s)
[FAIL] TestPipelineStatus_WaitsLongerWhenGitHubAsks (0.00s)
[FAIL] TestPipelineStatus_IdentityChangesWithAnyPartOfTheDefinition (0.00s)
[FAIL] TestPipelineStatus_PollsAtTheDefaultInterval_AgainstTheDefaultHost (0.00s)
[FAIL] TestPipelineStatus_PublishesEachRunStateAsAKeyStatus (0.00s)
[FAIL] TestPipelineStatus_PressOpensTheRunFromItsState (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/bricks/pipelinestatus	0.180s
FAIL
```

The daemon's live integration:

```text
[FAIL] TestDaemon_RunsLiveSourcesWithNoDeck_AndSharesThemBetweenKeys (0.00s)
[FAIL] TestDaemon_LogsAConfiguredCredentialThatIsMissing (0.00s)
[FAIL] TestDaemon_CountsPressesByKind (0.04s)
[FAIL] TestDaemon_PressesALiveKeyWithItsLatestState (0.04s)
[FAIL] TestDaemon_RepaintsALiveKey_WhenItsSourcePublishes (0.05s)
[FAIL] TestDaemon_ShowsAnOffPageUpdate_TheMomentThePageIsShown (0.06s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/daemon	0.410s
FAIL
```

**Not tests first:** `TestComposition_NeverLogsTheToken`, the specification's
"no token appears in any log line" check, was written after the wiring it
exercises. It was therefore proven the datum's way for a check instead: a
realistic leak was planted in a scratch copy (the brick showing the error's text
as its reason, and the client's 401 error carrying the Authorization header), and
the test failed on it. It was then reordered so a leak is reported as a leak
before anything else, and failed on the plant again:

```text
##### planted: token leak (error text shown as the reason, token in the 401 error)
$ go test -run TestComposition_NeverLogsTheToken ./cmd/the-front-buttons/
[FAIL] TestComposition_NeverLogsTheToken (10.02s)
    main_test.go:115: want the refused token reported by reason, got:
        {"time":"2026-11-12T00:11:13.213967Z","level":"INFO","msg":"deck","state":"waiting"}
        {"time":"2026-11-12T00:11:13.21718Z","level":"WARN","msg":"live brick error","kind":"pipeline_status","reason":"pipeline poll failed: unauthorised: HTTP 401 with Bearer github_pat_MUST_NEVER_BE_LOGGED_0123456789"
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/cmd/the-front-buttons	10.278s
FAIL

##### rerun after ordering the leak check first
$ go test -run TestComposition_NeverLogsTheToken ./cmd/the-front-buttons/
[FAIL] TestComposition_NeverLogsTheToken (10.03s)
    main_test.go:116: the token appeared in the log:
        {"time":"2026-11-12T00:11:37.267934Z","level":"INFO","msg":"deck","state":"waiting"}
        {"time":"2026-11-12T00:11:37.271262Z","level":"WARN","msg":"live brick error","kind":"pipeline_status","reason":"pipeline poll failed: unauthorised: HTTP 401 with Bearer github_pat_MUST_NEVER_
FAIL
```

### Reproduction report, 12 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; Apple clang 21.0.0; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| Windows type check | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet` over every package except the cgo adapter and the composition root: clean |
| `go test -race -count=1 ./...` | every package ok; 365 tests and subtests passed, 0 failed, 2 skipped |
| Skipped | the TE6 device contract (needs a deck); `TestKeyringCredentialStore_PassesTheCredentialStoreContract_WithConsent` (writes to the real Keychain or Credential Manager; needs `THE_FRONT_BUTTONS_CREDENTIAL_TESTS=1`) |
| `golangci-lint config verify`, then `run ./...` | valid; 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Memory analysis | race detector, clean |
| Warnings | 0 |
| **Author gate, outstanding** | a real workflow turning red then green on the deck at work, with a press opening the failing run; the credential contract run with consent on both machines |
| Not run | anything against real GitHub; anything on the Dell; CI |

## TE9: editor

Started 12 November 2026, after rereading datum Part 0 and 0.17.

### What was built

- **The guard** (`internal/editor/server.go`): every request must name the
  editor's own address (or `localhost` on the same port) as its Host, which
  defeats DNS rebinding; any Origin must be the editor's own, and every change
  must carry one; every change must carry the token minted for this launch (256
  bits, compared in constant time); bodies are bounded at 64 KiB before parsing.
  Each refusal is counted and logged by reason, method and path, never body or
  query. Every response carries a strict Content Security Policy (only the
  editor's own files), no framing, no sniffing, no referrer and no caching.
- **Pages** (`internal/editor/pages.go`, `templates/`): the page list and the key
  grid, each key a link named by row, column and contents, the two navigation
  keys shown as reserved; adding, renaming and deleting pages; the key form, its
  type list (every kind and every preset, from the registry and the preset table)
  and its controls generated from the schema, with labels, required marks,
  bounds, help text and errors linked by `aria-describedby`; removing a key;
  write only credential entry and a list of the credentials the configuration
  names, each Stored or Missing; the diagnostics counters and the last refused
  configuration; a preview of each key from the renderer.
- **Saving** goes through the store's `Save`, which validates by encoding and
  decoding and writes atomically; the watcher then reloads it like any other
  change. The store gained `OnDisk`, which decodes the file without consuming the
  change, so the editor shows what it just saved.
- **htmx** 2.0.11, vendored with its digest (`PROVENANCE.md`), only switches the
  key form when its type changes. Every form works by keyboard with no JavaScript.
- **`open_editor`** (TD9): a key that opens the editor at its current address.
- **The composition root** opens the editor on a loopback port, prints its
  address, gives it to `open_editor`, and shuts it down with the daemon.

### Decisions taken by the agent under DV2

1. **A new port every launch.** The operating system picks the port, so it never
   collides with something already running; the printed address and the
   `open_editor` key always have the current one. A bookmark does not survive a
   restart.
2. **Editing waits while the file on disk is invalid.** Saving from the editor
   then would overwrite the hand edit with the last good configuration plus the
   change; instead the problems are listed and the edit is refused.
3. **Saving from the editor drops hand written comments**, the question 11.3 left
   for TE9. Keeping them through the YAML node tree is a contained change in
   `internal/config` if the author wants it.
4. **There is no drag and drop**, so the keyboard alternative the specification
   asks for is the whole interface: a grid of links reached with Tab and followed
   with Enter, and forms.
5. **The preview shows the key as last saved**, without live state or unsaved
   edits; a live preview of the form is a later refinement.
6. **Text fields are trimmed** of surrounding spaces, which pasted URLs and IDs
   often carry.
7. **`brick.Ports` joins the exhaustive struct exemptions**, with the reason in
   the configuration: every kind refuses to construct without its ports.

### Tests first

Written against stubs and run first (go test prefixes written in brackets).
`OnDisk`:

```text
[FAIL] TestStore_OnDisk_ReportsAnInvalidFile (0.00s)
[FAIL] TestStore_OnDisk_ReadsTheFileWithoutChangingWhatIsRunning (0.00s)
panic: runtime error: index out of range [1] with length 0 [recovered, repanicked]
panic({0x102dd4fb8?, 0x67795e29f1b8?})
	/opt/homebrew/Cellar/go/1.27.3/libexec/src/runtime/panic.go:859 +0x120
FAIL	github.com/TheScottBot/the-front-buttons/internal/config	0.246s
FAIL
```

The editor, against a handler that answered everything with 404 (the label test
passed against it trivially, finding no controls):

```text
[FAIL] TestEditor_AddingAKindAddsItsForm_WithNoEditorChange (0.00s)
[FAIL] TestEditor_SetsProtectiveHeaders (0.00s)
[FAIL] TestEditor_AddsRenamesAndDeletesPages (0.00s)
[FAIL] TestEditor_ShowsAValidationErrorAtItsField (0.00s)
[FAIL] TestEditor_RefusesAChangeWithNoOrigin (0.00s)
[FAIL] TestEditor_RefusesARequestFromAnotherOrigin (0.00s)
[FAIL] TestEditor_RefusesAChangeWithAMissingOrWrongToken (0.00s)
[FAIL] TestEditor_DeletesAKey (0.00s)
[FAIL] TestEditor_SavesAPresetKey_ByItsPresetAndFields (0.00s)
[FAIL] TestEditor_RefusesAKeyOnANavigationPosition (0.00s)
[FAIL] TestEditor_RefusesAnOversizedRequest (0.01s)
[FAIL] TestEditor_GeneratesAFormForEveryKindAndPreset_FromTheirSchemas (0.00s)
[FAIL] TestEditor_StoresACredential_AndNeverShowsItBack (0.00s)
[FAIL] TestEditor_GridIsOperableFromTheKeyboard (0.00s)
[FAIL] TestEditor_EveryPageHasASkipLinkAMainLandmarkAndALiveRegion (0.00s)
[FAIL] TestEditor_ShowsTheGrid_WithTheNavigationKeysReserved (0.00s)
[FAIL] TestEditor_RefusesARequestForAnotherHost (0.00s)
[FAIL] TestEditor_PreviewsAKeyWithTheRenderer (0.00s)
[FAIL] TestEditor_ExplainsAHandEditTheStoreRefused (0.00s)
[FAIL] TestEditor_ShowsTheDiagnosticCounters (0.00s)
[FAIL] TestEditor_SavesAKindKey_AndTheFileRoundTrips (0.00s)
[FAIL] TestEditor_StylesShowFocusRespectReducedMotionAndNeverUseColourAlone (0.00s)
FAIL
FAIL	github.com/TheScottBot/the-front-buttons/internal/editor	0.270s
FAIL
```

`open_editor`, against a stub kind that constructed an empty value:

```text
[FAIL] TestOpenEditor_OpensTheEditorsAddressWhenPressed (0.00s)
FAIL	github.com/TheScottBot/the-front-buttons/internal/bricks/openeditor	0.182s
FAIL
```

One editor test was corrected before it went green: it assumed a control's
`name` attribute came before its `id`, which is no requirement; it now finds the
control by name and reads its id from the same tag.

### Smoke run against the live editor

The built daemon was started with a sample configuration and probed with curl
on the port it printed:

```text
editor printed: http://127.0.0.1:(a port the operating system chose)/
GET /page, own host: 200
GET /page, forged host: 403
POST /page/add, own origin, no token: 403
POST /page/add, hostile origin: 403
daemon exit 0
```

Each refusal was also logged by reason (three `editor request refused` lines).

### Reproduction report, 12 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; Apple clang 21.0.0; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| Windows type check | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet` over every package except the cgo adapter and the composition root: clean |
| `go test -race -count=1 ./...` | every package ok; 393 tests and subtests passed, 0 failed, 2 skipped (the device and real credential contracts) |
| `golangci-lint config verify`, then `run ./...` | valid; 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Memory analysis | race detector, clean |
| Warnings | 0 |
| **Author gate, outstanding** | building a full page from scratch in the editor on each machine, including once using the keyboard only. Automated label, landmark, live region, focus and keyboard checks pass; they are not the gate |
| Not run | a screen reader; anything on the Dell; CI |

## TE10: start at login, reference and documents

The last phase, built under DV1 and committed under DV2.

### What was built

- `platform.LaunchAgentPlist` and `platform.ScheduledTaskXML`: the start at login
  definitions, generated in Go and printed by the binary with
  `-print-launch-agent` and `-print-scheduled-task -user`, with `-print-name` for
  the task's name. Every path is XML escaped.
- `scripts/install-macos.sh` and `scripts/uninstall-macos.sh`: a per user login
  agent through `launchctl bootstrap gui/UID`, removing only the agent they wrote.
- `scripts/install-windows.ps1` and `scripts/uninstall-windows.ps1`: a per user
  logon task through `Register-ScheduledTask -Xml`, no administrator rights.
- `diagnostics.RotatingFile`: the log file the specification's section 7 asks
  for, in the state folder, owner only.
- `internal/reference`: the README's brick reference, generated from every kind's
  schema and every preset, with a test that fails when the README drifts.
- The prose check now reads a Go string literal as its value, so a Markdown table
  separator built in Go is not read as a dash run.
- LICENSE, CHANGELOG.md, SECURITY.md, PRIVACY.md and HARDWARE_COMPATIBILITY.md;
  README.md rewritten; TESTING.md updated.

### Decisions taken by the agent under DV2

1. **ProcessType Interactive** on the macOS agent: launchd throttles background
   agents, and a key press must feel immediate.
2. **KeepAlive with SuccessfulExit false**: restarted after a crash, left stopped
   after a clean shutdown, so stopping it on purpose sticks.
3. **Task Scheduler schema version 1.3**, the lowest that carries every element
   used, per Microsoft's task schema.
4. **The scripts take the definition from the binary**, so it is written once, in
   Go, and tested there; the scripts only validate the path and register.
5. **LICENSE: MIT, held by TheScottBot.** The author's to change.
6. **The log rotates at 1 MiB, keeping 3 older files**, so it never passes 4 MiB.
   Output before the log opens goes to a launchd log beside it on macOS.
7. **The prose check reads Go string values** rather than exempting test files.
8. **SECURITY.md's accepted risks are the agent's statement**, marked as awaiting
   the author's own words.

### Tests first

The first run, before the start up definitions, the rotating file and the
generator existed:

```text
[FAIL] TestLaunchAgentPlist_StartsTheDaemonAtLoginAndRestartsItOnlyAfterACrash (0.00s)
[FAIL] TestLaunchAgentPlist_EscapesAPathThatLooksLikeMarkup (0.00s)
FAIL	github.com/TheScottBot/the-front-buttons/internal/platform	0.250s
[FAIL] TestRotatingFile_RotatesAtItsBound_AndKeepsOnlyTheNewestFiles (0.00s)
[FAIL] TestRotatingFile_AppendsToAnExistingFile (0.00s)
FAIL	github.com/TheScottBot/the-front-buttons/internal/diagnostics	0.250s
[FAIL] TestMarkdown_DescribesEveryKindFieldAndPreset_FromTheirSchemas (0.00s)
FAIL	github.com/TheScottBot/the-front-buttons/internal/reference	0.163s
```

The scheduled task, before it existed:

```text
[FAIL] TestScheduledTaskXML_StartsTheDaemonAtThisUsersLogon_WithoutElevation (0.00s)
FAIL	github.com/TheScottBot/the-front-buttons/internal/platform	0.168s
```

The prose check, before it read Go string values:

```text
[FAIL] TestScanProse_AllowsATableSeparatorRowInsideAGoStringLiteral (0.00s)
FAIL	github.com/TheScottBot/the-front-buttons/internal/repositorychecks	0.194s
```

One test was corrected before it went green: the property list reader in the
launch agent test ignored values nested deeper than the top dictionary, so it
read KeepAlive as `SuccessfulExit` without its `false`. It now flattens nested
values. The corrected test was then seen to fail with the agent's KeepAlive
planted as true:

```text
[FAIL] TestLaunchAgentPlist_StartsTheDaemonAtLoginAndRestartsItOnlyAfterACrash (0.00s)
    startup_test.go:105: KeepAlive: want "SuccessfulExitfalse", got "SuccessfulExittrue"
```

The generator was also found to leave no blank line before the README's next
heading; it now ends with one, and the README was regenerated.

### Planted violations

| Planted | Result |
|---|---|
| macOS install script given a path that does not exist | `error: ... is not an executable file`, exit 2 |
| macOS install script given a file that is not executable | exit 2 |
| macOS uninstall script given a path that does not exist | exit 2 |
| README brick reference drifted from the registry | `TestREADME_BrickReferenceMatchesTheRegistry` fails, naming the command to regenerate |
| The agent's KeepAlive set to true | the launch agent test fails, as above |

The Windows scripts could not be run here (no PowerShell on this Mac); their
refusal of a missing binary is checked only by the CI step on windows-2025.

### Reproduction report, 12 November 2026, this Mac only

| Item | Value |
|---|---|
| Platform | macOS 27.2 (26B7124), arm64; Go go1.27.3 darwin/arm64; Apple clang 21.0.0; golangci-lint v2.15.0 |
| `go build ./...` then `go vet ./...` | clean |
| Windows type check | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet` over every package except the cgo adapter and the composition root: clean |
| `go test -race -count=1 ./...` | every package ok; 404 tests and subtests: 402 passed, 0 failed, 2 skipped (the device and real credential contracts) |
| `golangci-lint config verify`, then `run ./...` | valid; 0 issues |
| `scripts/check-duplicate-code.sh` | no duplicate code at or above 100 tokens |
| Warnings | 0 |
| **Author gate, outstanding** | on each machine, install with the script, log out and in, and find the daemon running and the deck live without a terminal open |
| Not run | either install script for real (it would change this login); the Windows scripts at all; anything on the Dell; CI |

## Deck testing, 12 November 2026, the MacBook

The deck arrived. Everything below ran on the MacBook with the author at the
deck; nothing has been run on the Dell.

### What the deck reported

| Item | Reading |
|---|---|
| USB | Stream Deck XL, vendor 0x0FD9, product 0x008F (the XL V2 row of the model table), firmware 0x0130 |
| HID | usage page 0x0C, usage 0x01; input reports up to 512 bytes, output 1,024, feature 32 |
| Elgato software | not running |

### Defect 1: two watchers aborted the process

The first run of the device contract aborted the test process with SIGABRT
inside `hid_init`: the watcher contract and the device contract ran in parallel,
each with its own watcher, and each watcher initialised hidapi behind its own
`sync.Once`. hidapi's process wide state is not safe from two threads. The
daemon has one watcher, so it may never have met this, but nothing prevented it.

Test first: `TestWatcher_SeveralWatchersAtOnce_DoNotCrashHIDAPI` enumerates from
eight watchers at once. It needs no deck, so it runs in CI. Before the fix it
aborted three runs out of three:

```text
SIGABRT: abort
FAIL	github.com/TheScottBot/the-front-buttons/internal/deck/hidtransport	0.169s
```

Fix: one process wide lock held around every hidapi call that touches its global
state (initialise, enumerate, open, close), with initialisation once per process.
Then three runs out of three passed under the race detector, and the device
contract passed on the deck.

### Experiment 1

A temporary probe, not committed, painted each key with its index and logged
every raw input report while the author pressed keys:

```text
REPORT length=512 header=01002000 pressed=[0]
REPORT length=512 header=01002000 pressed=[7]
REPORT length=512 header=01002000 pressed=[24]
REPORT length=512 header=01002000 pressed=[31]
REPORT length=512 header=01002000 pressed=[3 4]
```

| Question | Answer |
|---|---|
| Key order | row by row from the top left, as built: top left 0, top right 7, bottom left 24, bottom right 31 |
| Several keys at once | reported together |
| Image orientation | upright, by the author's eye |
| Input header | 01 00 20 00 on every report: report ID 1, then the key count, 32, little endian. Not interpreted, as built |
| Input length | 512 bytes, zero after the 36th. The driver asks for 36 and hidapi returns those 36, so the exact length check holds; seen working in the daemon |
| Brightness and the macOS feature read | accepted, and the serial number decodes (device contract) |
| Replug | not yet done |

### Defect 2: paging took seconds and drew keys one at a time

The author reported that paging took a few seconds each way, that keys drew one
at a time with each press, and that presses seemed lost. The daemon's log showed
about twenty `write failed` disconnects in thirty seconds, each dropping the half
drawn page and reconnecting. A temporary probe painted full pages (128 writes)
and timed them:

| | Full page | Write failures |
|---|---|---|
| Writing alone | 36 ms | 1 in 1,280 |
| With a reader running, as the daemon does | 3.0 s | 6 in 384 |

Every failure read `IOHIDDeviceSetReport failed: (0xE00002BC) (iokit/common)
general error`, with the deck still attached.

Two causes. The adapter held the device's lock for the whole of each timed read,
so every write waited out a 50 millisecond read. And the daemon took any write
error for an unplugging.

Tests first. A new device contract clause, "a write is not held up by a read in
progress", which the test double already met and the real adapter did not:

```text
deckcontract.go:52: want a write within 100ms while a read waits, took 484.11725ms
[FAIL] TestDevice_PassesTheDeckDeviceContract_OnAnAttachedDeck/a_write_is_not_held_up_by_a_read_in_progress (0.51s)
```

and three unit tests for a retry rule, which failed to build before it existed.

Fix, decided by the agent and described here for the author's review: a timed
read now asks hidapi without waiting, holding the lock only for that, and sleeps
5 milliseconds unlocked between asks, as the reference library polls. A write is
tried up to three times, 2 milliseconds apart, before it counts as failed. The
retry covers output writes only; feature reports were not seen to fail.

After the fix:

| | Full page | Write failures |
|---|---|---|
| Writing alone | 37 ms | 0 in 1,280 (one write took 2.7 ms: a retry that succeeded) |
| With a reader running | 33 ms | 0 in 384 |

The device contract, now six clauses, passes on the deck. The author paged both
ways and pressed keys and reported paging as snappy; the daemon logged no
disconnects in that run.

### Replug and legibility

With the daemon running, the author unplugged and replugged the deck four times
and then pressed a key. The daemon logged each one and needed no restart:

```text
19:32:32.459 deck disconnected, reason read failed
19:32:33.452 deck waiting
19:32:35.455 deck connected, Stream Deck XL V2
19:32:37.548 deck disconnected, reason read failed
19:32:42.455 deck connected, Stream Deck XL V2
19:32:47.698 deck disconnected, reason read failed
19:32:50.455 deck connected, Stream Deck XL V2
19:32:55.375 deck disconnected, reason read failed
19:32:59.454 deck connected, Stream Deck XL V2
19:33:02.369 key press, open_url, page 0, row 2, column 6
```

An unplugging is seen at once as a failed read; the deck reconnects on the
watcher's next one second poll after it reappears. The author reported the
deck coming back as expected, and the labels, icons, chevrons and page
indicator legible at arm's length in normal desk light.

### Start at login, installed

With the author's go ahead, the binary was built to `~/.local/bin`, the test
configuration copied to the default location, and `scripts/install-macos.sh`
run. launchd reported the agent running and never exited, and the daemon's own
log showed it starting from the default configuration and the deck connected,
with no terminal involved. The log out and log in that clears the gate waits
for the author.

### Defect 3: every plain save from the editor was Forbidden

Adding a key in the editor, in the author's own browser, gave Forbidden. The log
read `editor request refused, reason origin, POST /key`. The editor sent
`Referrer-Policy: no-referrer`, under which a browser sends `Origin: null` on a
plain form submit even to the same site, and the guard rightly refuses null.
Every earlier probe and test set the Origin by hand, so none could see it.

Test first, `TestEditor_ReferrerPolicyLetsTheBrowserSendItsOwnOrigin`, pinning
the policy exactly:

```text
guard_test.go:128: want Referrer-Policy same-origin, got "no-referrer"
```

Fix: `same-origin`, which keeps the editor's own origin on its own requests and
sends nothing to other sites. The guard still refuses null. The author then
added a Discord key from the editor and saw it on the deck.

### Defect 4: reinstalling the login agent failed

Running `scripts/install-macos.sh` over a running agent failed with
`Bootstrap failed: 5: Input/output error`, leaving no agent loaded until the next
login: `launchctl bootout` returns before launchd has removed the service. The
script now waits up to ten seconds for the old copy to go, and fails plainly if
it does not. After the change one fresh install and two reinstalls in a row all
succeeded.

### Defect 5: icons sat high on their keys

The author's photographs showed icons close to the top edge. A plain icon key
used the status key's layout, which keeps a line free for a status word it never
shows. Test first, measuring the space above and below the drawn content:

```text
render_test.go:272: {Label:Build Icon:test-circle ...}: 5 pixels above the content and 23 below, want within 4 of each other
```

Fix, decided by the agent: with no status word, the icon and label are centred
as a group, the icon starting at 15 hundredths and the label's baseline at 84;
an icon with no label is centred alone. Status, error, label only and navigation
keys are unchanged, and their golden images are identical. Seven golden images
of icon keys were regenerated. The author's verdict on the deck: "Muuuuuch
better".

### TD13 amended by the author: Discord opens the app

The author's Discord preset key opened the channel in the browser, as TD13's
trade off foresaw ("Discord `https://` only"). With the author's go ahead,
experiment 3 was run for Discord from a shell:

```text
$ open "discord://discord.com/channels/(server ID)/(channel ID)"
exit 0
```

The Discord app registers the `discord` scheme (`CFBundleURLSchemes` in its
Info.plist), and the author reported the app switching to the right channel.
The author then chose the app form ("Yes, try it", then "That worked"), which
amends TD13: the `discord_channel` preset now emits
`discord://discord.com/channels/{server}/{channel}`.

`open_url`'s field moves from `https_url` to a new format, `link_url`: an https
URL, or a `discord://` link whose host is exactly discord.com with no user
information. Every other scheme stays refused. The GitHub API base address keeps
`https_url`. Tests first: the new format's eleven cases, the preset's expected
link and the field's format, all failing before the change. The author pressed
the key on the deck and the Discord app opened.

Not yet seen: the `discord://` form on the Dell.

### A keyboard only dry run of the editor, before the author's TE9 gate

The agent ran a throwaway daemon on a scratch configuration (the login agent
kept the deck, so the throwaway served only an editor) and, in the desktop app's
browser pane, added a page and a Discord preset key by keyboard: Tab, Enter and
typing. The skip link, the labelled grid of keys, the "(required)" marks and the
saved message all worked. This is not the gate, which stays the author's.

The browser pane cannot drive a native drop down on macOS by key, so the type
list was set directly; it is a plain `<select>`, which the browser makes
keyboard operable, and the author's run covers it.

Two findings, each with a test first:

- The grid's caption read "Page 0 plus one of 1": the template held the words
  where the sum belonged. `TestEditor_CaptionCountsPagesFromOne` failed on it;
  the caption now reads "Page 2 of 2: Games", seen in the pane.
- A refused save opened with focus at the top of the page and an unchanged
  title, so nothing told a keyboard or screen reader user that it had failed or
  where. `TestEditor_TakesARejectedFormToItsFirstError` failed on both. Now the
  title starts "Error:" and the first field in error, only that one, carries
  `autofocus`, so it reads its label and then its error. In the pane, the title
  was seen and the served page marks only the server ID field; the pane was
  hidden, so focus actually landing was not seen and is for the author's run.

### TE8 on the deck: the pipeline key against this repository's own CI

The author merged `working-title` into `main` and pushed, which started the
first CI run. A `pipeline_status` key watching `ci.yml` on `main` was added to
the author's configuration, and the author stored a fine grained token (Actions
read only, this repository only) through the editor's credentials page. The key
showed `credential_missing` until the token was stored.

**Defect 6: a press error hid newer live news.** The author pressed the key
before the token was stored; the press failed with `no_run`, and that error
stayed on the key after the poll found the failed run, because a press error
outranked the live state and nothing cleared it. Test first,
`TestDaemon_NewerLiveNewsReplacesAnOlderPressError` (the fake live key now
fails an early press as the real one does):

```text
live_test.go:284: want the published failure shown over the older press error
```

Fix: a source's publish clears the press errors of its keys on every page. The
author then saw the key show the failure, and a press open the failed run.

**The first CI run failed on Windows only**, at the linter configuration step:
`go: -modfile=tools/go: file does not have .mod extension`. PowerShell, the
Windows default shell, split the argument at its dot. Every step now runs under
Bash unless it names its own shell. The second run passed build and the race
detector tests on Windows for the first time, then failed lint on two findings
in `system_windows.go`, which only builds for Windows and so was never linted on
the Mac. Both are fixed, a Windows lint of every non cgo package from the Mac
now reports none, and TESTING.md gives the command.

**TD10 amended by the author, with two further decisions.** During the second
run the key said in progress, and a press opened the previous failed run, as
TD10 part b specified; to the author that read as the key reporting the wrong
run. And with `fail-fast: false`, a run stays in progress until its last job
ends, hiding a known failure for minutes, made worse by a minute between polls.
The author chose, from the agent's options:

1. A press opens the newest run, the one the key shows (amending TD10 part b).
   The client now asks for one run, not twenty.
2. A failed job shows as a failure at once: while the newest run is queued or
   in progress, the client also reads that run's jobs, and any job concluded
   failure or timed out makes the key a failure. If the jobs cannot be read,
   the run's own state stands.
3. A key polls every 15 seconds while its run is queued or in progress, and at
   its interval otherwise; GitHub's own longer waits still win. This goes below
   the interval field's 30 second minimum only while a run is active.

Tests first, each seen failing:

```text
client_test.go:397: want run 9 in progress and opened, got {State:in_progress TargetURL:.../runs/8}
client_test.go:421: failure: want run 9 shown as failed already, got {State:in_progress ...}
client_test.go:425: want the run's jobs asked for, got /repos/octo-org/hello-world/actions/workflows/ci.yml/runs
pipelinestatus_test.go:272: want polls at 0, 15, 30, 45 seconds and 1:45, got 3
```

and a third, that a finished run costs one request only. The test that opened
the last failure while a fix ran is removed with the rule.

### CI green on both runners, and the key followed it

The third run, on `905cd27`, passed on macos-26 and windows-2025: the first
fully green run (run 43299936830, completed 20:30:34). Its only annotation was
GitHub's notice of longer queue times for macOS arm64 runners. The author
reported the key showing in progress and then turning green as the run
finished. With the earlier failed run shown and opened by a press, the key has
now been seen red, in progress and green against a real workflow, which is what
TE8's gate asks; recording the gate cleared is the author's.

### The credential contract on the Keychain

With the author's go ahead, the `CredentialStore` contract ran against the real
macOS Keychain (`THE_FRONT_BUTTONS_CREDENTIAL_TESTS=1`): reads back what was
written with the latest write winning, reports a missing name as missing, and
refuses an oversized secret without echoing it, all passing. Afterwards
`security find-generic-password` found no `contract-test-entry` under
`the-front-buttons`, and the author's `github-front-buttons` entry still
present; no secret was printed, and the pipeline key logged no new credential
error.

### The editor's port, fixed by the author's decision

TE9's first decision gave the editor a new port at every launch. Each
reinstall today restarted the daemon, and the author found the address
"cycling all the time"; the author chose a fixed port. Rather than a new
configuration field, which every hand built configuration in the tests would
have had to carry, the agent made it the default: the editor tries 127.0.0.1
port 48080 first and, only if another program holds it, takes any free port
and logs `editor port taken; using another` with the address chosen.
`-editor-port` names another port, and 0 keeps the old behaviour; a port
outside 0 to 65535 is refused with exit status 2. The Host, Origin and token
checks protect the editor, not an unpredictable port.

Tests first (they failed to build against the old signature): a free preferred
port is used, a taken one falls back to another loopback port and reports it,
zero lets the system choose, and the default is 48080. After two reinstalls in
a row the log read `http://127.0.0.1:48080/` both times and the page answered
200. README and SECURITY.md, including its note for IT, now name the port.

### The D&D Beyond preset on the deck

The author added three `dndbeyond_character` keys from the editor, one per
character, and pressed each: all three opened the right character sheet. With
the Discord app link, that is experiment 3 run for both presets the author
uses on the Mac; Teams is owed.

### The icon gallery and keys' own icons, at the author's request

The author asked to use the D&D Beyond logo. The agent declined to bundle it:
it is Wizards of the Coast's trademark and artwork, and this repository is
public under the MIT licence. Two features followed instead.

**The icon gallery.** The author had "no clue what icons are available". The
editor gains an Icons page listing every bundled icon (5,247), each drawn by the
key renderer as it would be on a key, filtered by a search that needs every
word typed to appear in the name, showing at most 120 at a time with the count,
and linked from every key form's Icon field. Only a known icon's name is drawn;
any other name is not found. Tests first, failing to build before the page
existed: the list in order with previews and a count, the search, the preview's
404 for an unknown or path like name, and the link from the Icon field. Seen in
the pane on the real set: a search for "dice" showed seven icons.

**`icon_file`.** TD5 settled that a person's own icons are PNG, and the renderer
could already decode one within bounds, but nothing connected a key to a file:
that half of TD5 had never been built. Every key's appearance gains
`icon_file`, a plain PNG file name (format `png_file`: dot separated parts of
letters, digits, underscores and hyphens, ending `.png`; no slash, no leading
dot, no empty part), used instead of `icon`. The file is read only from the
`icons` folder beside the configuration file, through `os.Root`, so neither a
name nor a link in the folder can reach another file; at most one byte past the
1 MB bound is read, and the existing decoder checks the size and dimensions
before decoding. A file that cannot be loaded draws the missing icon marker,
exactly as an unknown bundled icon does, and is logged once per configuration
with the reason. The daemon and the editor's preview share the reader.

Tests first, each failing before the code: the format's twelve accept and
refuse cases; the folder reading a PNG, refusing traversal and non PNG names
before touching the disk (with a planted file outside the folder left unread),
reporting a missing file, and refusing an oversized one; the renderer drawing
the missing marker for an unloadable file identically to an unknown icon; and
the daemon drawing a key's own PNG, or the marker when its file is gone. The
README's generated brick reference now lists `icon_file` on every kind and
preset.

### Moving and copying keys in the editor

The author asked to move the Editor key without re-entering it. At the
author's request the work went to a subagent, briefed with the datum's rules
(tests first, no git writes, no documents, the running daemon left alone),
while the agent finished `icon_file`. An existing key's page gains a "Move or
copy this key" form (destination page, row and column, labelled, starting at
the key's place) posting to `POST /key/move` through the same guard and save
path as every other change. The destination must be a free tile on the grid,
not a paging position and not the key's own place; anything else is refused
with a 422 naming why, the title starting "Error:" and focus on the destination
page. A move or a copy carries the key's kind, parameters and preset over
exactly, as a deep copy (`config.Key.Clone`, now also used by the configuration
copy), so a copy and its original share nothing.

The subagent's report gave its failing runs before the code (the route absent:
404s, and no form on the page) and nine tests, covering a move, an independent
copy, a preset moved across pages, every refusal, the token guard and a broken
file on disk. The agent then reran every check independently (all tests, 0 lint
issues, no duplicate code, no new lint exceptions), read the checks and the
copy, and tried it in the browser pane against a scratch configuration: the
Editor key moved from row 1, column 3 to row 0, column 4 in the file, and moving
a key onto an occupied tile was refused with "that destination already has a
key". The native selects were set directly, as the pane cannot drive them by
key on macOS.

**Correction: the form offered tiles that could not be used.** The author
found that the move form "doesn't take the reserved keys into account". The
server refused the paging tiles, but the form's separate page, row and column
lists still offered them, as well as occupied tiles and the key's own place,
so a refusal came only after submitting. The three lists became one,
"Destination tile", grouped by page, listing only tiles on the grid that paging
does not hold, no key uses and that are not the key's own place; a page with
none is left out, and with none anywhere the form says so instead. Its value is
the page, row and column joined by dots, and anything malformed is refused as
not on the deck. The server's checks stay as they were.

Tests first: `TestEditor_MoveListOffersOnlyFreeTiles` (two pages, 59 tiles
offered, the four paging tiles and the key's own place absent, the second
page's group named) and `TestEditor_RefusesAMalformedDestination` were added,
the existing tests changed to post the one value, and all failed before the
change. On the author's own configuration the Editor key's page offered 80
tiles over three pages, none of them a paging tile, an occupied tile or its own
place.

### Dragging keys on the grid

The author asked to drag a key to a new tile, preferably in TypeScript. Asked
how to build it, the author chose TypeScript compiled by esbuild run as a
pinned Go tool, over plain checked JavaScript or TypeScript with tsc and Node.

#### Dependency record: github.com/evanw/esbuild v0.28.2, tools module only (datum 0.13)

Written after the tool was added, not before as datum 0.13 asks; the agent
recorded the version check first and wrote this record late. Facts read 12
November 2026 from the module proxy and the module's own files.

| Item | Record |
|---|---|
| Behaviour provided | Compiling `internal/editor/web/drag.ts` to the JavaScript the editor serves |
| Why the standard library is insufficient | Go cannot compile TypeScript; the alternatives were no TypeScript (plain JavaScript) or Node and npm for tsc |
| Version | v0.28.2, published 8 August 2026, the latest release |
| Licence | MIT |
| Requires | `golang.org/x/sys` only, at a 2022 version esbuild holds back on purpose for old Go releases; in the tools module, so it never enters the daemon's build |
| Security | Runs only at development time and in CI, never in the daemon; its output is committed and reviewed like any other file |
| Limit | esbuild strips types without checking them; type checking would need tsc and Node |
| Test strategy | `scripts/check-editor-script.sh` rebuilds the script and fails if the served file differs; proven on a planted hand edit |
| Replacement cost | One script, `scripts/build-editor-script.sh`; the TypeScript itself is plain and would build with tsc unchanged |

Added with `go get -tool github.com/evanw/esbuild/cmd/esbuild@v0.28.2` in the
tools module.

#### What was built

`internal/editor/web/drag.ts` is compiled by `scripts/build-editor-script.sh`
(run by `go generate ./internal/editor/`) to `internal/editor/static/drag.js`,
embedded and served under the editor's existing content security policy. Each
grid cell carries `data-tile` (page, row and column) and `data-state` (key,
empty or reserved); only a key's link is draggable and only an empty tile takes
a drop, so a paging or occupied tile never does. A drop posts the same `POST
/key/move` the move form sends, with the page's token, through the same guard
and checks; Option or Alt held at the drop copies instead. A move that works
loads the destination page with the saved message; a refusal is read from the
server's alert into the page's live region. A hint above the grid is hidden
unless the script runs, and names the move form as the keyboard way to do the
same; the dragged key and the tile under the pointer are marked with outlines,
not fading, so contrast holds.

Tests first, failing before the change: the grid marking every tile and only a
key draggable, the script tag, the hidden hint, and the script served as
JavaScript. The drift check started as a Go test, which the repository's
operating system check refused (only the adapters may start programs), so it
became a script beside the duplicate code check, run in CI on both runners.

In the browser pane on a scratch configuration, a real mouse drag of the Editor
key onto a free tile moved it in the file and reloaded the page with the saved
message, and dragging another key onto the forward paging tile did nothing and
sent no request. Copying with Option held was not tried by hand, since the
pane cannot hold a key during a drag; it is the same request with `action`
copy, which the move tests cover.

### Editing a key in a popup over the grid

The author asked for a key to open "in a pseudo popup instead of punching out
to another screen". The page gains an empty modal `<dialog>`, named by its
heading and with a close button; the script fetches the key's own page, lifts
its `#key-editor` part into the dialog and shows it modal, so the browser keeps
focus inside, closes it on Escape and leaves the grid inert. Every form inside
is sent from the script and its answer lands in the dialog: a refusal shows its
errors with focus on the field in error, and a change the server accepts
answers with its usual redirect, which is followed, so the grid reloads with
the saved message. htmx is handed the new content so the key type list still
swaps the fields in place. A click with a modifier, or no script at all, still
opens the key's own page. Closing returns focus to the tile and restores the
page title, which htmx otherwise takes from the swapped key page.

The script became one bundle, `web/editor.ts`, importing `web/drag.ts` and
`web/dialog.ts`, served as `static/editor.js`; the build and drift scripts and
the CI step were renamed to match. Restructuring it, the agent's own edit
emptied `drag.ts`; esbuild then only warned that the import was undefined,
which would have shipped a dead drag feature, so the build now makes that
warning an error, and `drag.ts` was restored from the last commit and edited
again. The agent also ran `git rm --cached` on the old `static/drag.js`,
staging its deletion: a git write the datum forbids, reported to the author.

Tests first, failing before the change: the page holds the dialog with its
name, heading, close button and empty body, the grid's links still go to each
key's page, and the editor script is served. In the browser pane on a scratch
configuration: a click on a tile opened its editor over the grid with the
address unchanged and focus on the type list; a bad URL kept the dialog open
with the error shown and focus on the field; Escape closed it with focus back
on the tile; Enter on the tile reopened it; changing the type to a preset
swapped the fields in the dialog; and saving wrote the key and reloaded the
grid with the saved message.

Seen in passing and not changed: a format error reads like "must be link_url",
naming the format rather than saying what is allowed.

### Presets named by their action

The grid named a preset key by its target kind, `open_url`, which told the
author nothing; the author asked for names such as `open_server` and
`open_character`, keeping `pipeline_status`. At the author's request the work
went to a subagent, briefed with the datum's rules and kept clear of the
popup's files, while the agent built the popup.

Each preset now declares its action with its definition: `Define` takes it and
refuses an empty or non lower snake case name, so the built in and test
presets are covered by one check. discord_channel is `open_server`,
dndbeyond_character `open_character`, teams_chat and teams_chat_by_id
`open_chat`, and the three Teams channel presets `open_channel`. The grid's kind
line and accessible name, the key form's type list and the README's generated
reference all name the action; a key without a preset is unchanged, and a
preset the table no longer knows falls back to its kind.

The subagent's report gave its failing runs before the code and six new or
extended tests. The agent reran every check independently (all tests, 0 lint
issues, no duplicate code, no new lint exceptions), read the lookup, and after
reinstalling read the author's own grid: Main as `open_server`, Yeku, Uri and
Foli as `open_character`, CI as `pipeline_status` and plain links as
`open_url`.

### An app picker, on both systems

The author asked for an app picker, "multi plat of course". The
`open_application` key's Application field gains a list of the installed
applications by name, under the field, labelled "Or choose an installed
application"; choosing one fills the field with the identifier the launcher
takes, and the field can still be typed into. A field asks for a list through
a new schema tag, `suggest:"applications"`, the only list so far; an unknown
list, or one on a field that is not text, is a declaration fault.

The list comes through a new port, `platform.ApplicationCatalogue`, with an
adapter per system. The parts that interpret what the system says are built
everywhere and tested on any machine; only the commands are per system.

- macOS: the bundles in `/Applications`, `/System/Applications` and
  `~/Applications` and the folders directly inside them, never inside a bundle,
  following symbolic links (macOS links `/Applications/Safari.app` elsewhere,
  which the first scan of the author's Mac missed), each named from its
  Info.plist through `/usr/bin/plutil`, which reads the XML and binary forms,
  and identified by its bundle identifier, or its name when it has none.
- Windows: `Get-StartApps` through Windows PowerShell, each entry stored as a
  `shell:AppsFolder\` target, which the launcher already passes straight to
  ShellExecute (DV3), so desktop and packaged applications open alike without
  a launcher change. Not run on a Windows machine: the author is testing on
  macOS only for now. It is vetted and linted for Windows from the Mac.

The editor keeps the list for ten minutes and fetches it in the background,
started when the grid page is shown, so a key's page rarely waits; a failed
listing keeps the last list. A first version started the fetch when the editor
was built, which `contextcheck` rightly flagged; the fetch now starts from a
request and outlives it without its cancellation.

Tests first, each failing before its code: the suggest tag and its faults, the
Start menu parser (an array, a single object, duplicates, blanks, a hyphen led
AppID refused, output that is not JSON), the bundle scan with a stand in
reader (nested folders, bundles inside bundles left alone, unreadable bundles
left out, a linked bundle followed), and the picker on the application field
only. On the author's Mac the scan offered 91 applications, Safari among them,
and in the browser pane choosing Spotify filled the field with
`com.spotify.client`. That check found the script was loaded only on the grid
page, so the picker did nothing on a key's own page; the script now loads from
the shared layout on every page, with a test.

### Paging through the icons

At the author's request a subagent, kept to the gallery's own files, added
pages to the Icons page: 120 icons a page in the same order, previous and next
links and "Page N of M" in a navigation landmark inside the live results, the
search kept in the links, a new search starting on page 1, and out of range or
malformed page numbers clamped. Its report gave its failing runs, from a
scratch copy because the app picker's half written tests stopped the package
building, and five tests. The agent reran every check and looked in the
browser pane on the real set: page 2 showed icons 121 to 240 of 5,247, "Page 2
of 44", with working links.

### Parked: reading mute and solo from a RØDECaster Pro II

The author asked for keys showing a RØDECaster Pro II channel's mute and solo,
then parked it on the agent's findings. RØDE documents MIDI control only, with
no state read back. Two MIT licensed community projects have reverse
engineered the USB HID protocol RØDE's own app uses: rodey's protocol notes
(github.com/seanheiney/rodey) and a Bitfocus Companion module
(github.com/gcdkgrnvle/companion-module-rode-rodecaster2), with a Rust library,
rodecaster-protocol. A session opened with two fixed messages gets a full state
dump and then every change pushed; mute is `channelOutputMute`, inverted (false
when muted), and cue is less certain (`channelCueEnable` in one, `soloPressed`
in the other). The author's desk enumerated on vendor 0x19F7, product 0x0092,
with a vendor HID interface on usage page 0xFF00 and 256 byte reports; the
projects saw product 0x0037 on firmware 1.7.3. The risks that parked it: a wrong
byte on the mode report can enter firmware update mode or start a flash, and a
reader that stops reading while a session is open freezes the desk's physical
controls until it is replugged. Were it resumed, the first step would be a
read only experiment with the author at the desk, sending only the two
documented session messages.

### A key that presses a key

The author asked for a key that sends a momentary press of a key, the extended
function keys first. The `press_key` brick takes `key`, a new `function_key`
format accepting F13 to F24 exactly, and presses it through a new
`platform.KeyPresser` port; failures show on the key as `permission`,
`unsupported_key` or `key_failed`.

The key codes were read from source, not memory: `kVK_F13` to `kVK_F20` from
the macOS SDK's HIToolbox `Events.h`, which defines no F21 to F24, and `VK_F13`
to `VK_F24` (0x7C to 0x87) from golang.org/x/sys v0.49.0, the version pinned.
Both tables are pure Go and tested on any machine.

- macOS: CoreGraphics key down and key up posted at the HID level, through a
  small cgo file in the system adapter. macOS lets only a process trusted for
  Accessibility do that; without it the press fails as `permission` and, once
  per run, has macOS show its prompt. F21 to F24 fail as `unsupported_key`.
  macOS ties the permission to the exact binary, so a rebuilt binary may need
  it granted again.
- Windows: `SendInput` with a key down and a key up. The 64 bit `INPUT` layout
  is 40 bytes, checked at compile time. Not run on a Windows machine; vetted
  and linted for Windows from the Mac.

Tests first, each failing before its code: the format, the brick pressing its
key and naming each failure, its required field and port, and both tables. The
agent added an F13 key to the author's configuration for the author to press;
its first choice of tile was one the author had since used, and the daemon
refused the file as `duplicate_position` and kept the last good configuration,
so the key went on the first free tile instead.

On the deck the key first showed `permission` even after the author granted
Accessibility: the running daemon had started before the grant, and macOS
reads the permission when a program starts. Restarting the agent, without
rebuilding, fixed it, and the author's test received F13. The error and the
README now say to restart after granting. The binary's ad hoc signature
carried Go's generic identifier `a.out`, which made it hard to find in the
Accessibility list; at the author's request `scripts/install-macos.sh` now
signs it again, still ad hoc, with the agent's label,
`io.github.thescottbot.the-front-buttons`. After reinstalling, `codesign -dv`
showed that identifier and `codesign --verify --strict` passed. macOS still
ties Accessibility to the exact build, so the author granted it again: an entry
added by hand with the plus button, for this bare binary, did not match it (a
press at 22:11:46, after a restart, still read `permission`), while the entry
macOS added itself from the daemon's own prompt, switched on and followed by a
restart, did (presses at 22:13:20 and 22:13:28 went through). The README gives
those steps. On this macOS the list is titled Device Control and Data Access.

### Application icons, a key list, and restarting for a permission

**Use app icon.** At the author's request a subagent added `use_app_icon` to
`open_application`: the key draws the application's own icon, and its
background takes the colour of the icon's visible edge (render.EdgeColour,
scanning in from each side to the first opaque pixel, taking the largest colour
group), unless a background is set, which wins. A new
`platform.ApplicationIcons` port reads the icon through AppKit on macOS (a cgo
Objective C file drawing NSWorkspace's icon into a 256 pixel bitmap, so icons
kept in an asset catalogue work); on Windows it returns
`platform.ErrIconUnavailable` and the key shows the missing marker, since shell
COM cannot be tried from the Mac. Its report gave failing runs and tests,
including a real AppKit test over Finder, Safari, Mail and Notes. The agent
reran every check (0 lint findings on macOS and Windows, one targeted exception
with its reason), and read the C side's memory handling: a manual retain and
release bitmap released on every path inside an autorelease pool, the PNG
bytes copied into memory of their own and freed after Go copies at most the
decoder's bound plus one, and the C string freed.

The agent made a slip worth recording: reinstalling for the signing change
built the whole tree while the subagent was part way through, so the deck ran
its unfinished code for a while. The author found the result, a check box
reading "Stub" that did nothing for Claude or Spotify. After review, previews
of the finished code drew Claude's icon on its orange edge and Spotify's on
black, each with a label in the contrasting colour, and the clean build was
installed.

**Set aside while ticked.** The author asked for Use app icon to disable the
Icon fields. A new schema tag, `replaces:"icon,icon_file"`, lets a check box
name the fields it stands in for; a field it does not declare, or the tag on
anything but a check box, is a declaration fault. The editor marks those
fields read only and dashed while it is ticked, not disabled, since a disabled
field is not sent and its saved value would be lost. Tests first; in the
browser, ticking set both aside and unticking restored them.

**A list for the key.** The press_key key field is chosen from F13 to F24
rather than typed: `brick.FormatChoices` lists a fixed format's values and the
editor shows any such field as a list with the saved value chosen. The schema
still checks what is sent; a posted F25 is refused. Tests first.

**Restarting for a permission.** A running daemon does not see Accessibility
granted. After a press is refused, the macOS key presser now asks a fresh copy
of the binary (`-accessibility-check`, exit zero when trusted) every two
seconds, and when it says yes the daemon stops cleanly and exits with 75,
which the login agent's KeepAlive restarts. The watcher is tested with a stand
in check (told once, one loop however often started, silent once stopped). On
the deck, after a clean install: a press refused at 22:25:24, the entry
switched on, the log reading "restarting to use the Accessibility permission
just granted" at 22:25:28 with the new daemon starting the same second, and the
author's F13 then working with no manual restart. The README's steps no longer
mention restarting.

### A bin to drag keys to, with undo

The author asked for a bin to drag a key to, appearing only while dragging,
"somewhere obvious but not in the way so you accidentally bin it". The page
carries a hidden bin drawn like a key, the bundled trash icon on black with a
dashed yellow edge and the words "Drop here to remove". The drag script shows
it when a key is picked up, fixed in the window's bottom right corner, or its
top right corner when the grid reaches the bottom one, so it never sits over a
tile; over it, it turns yellow and adds "(release to remove)". A drop sends
the same POST /key/delete as a key's Remove button, which stays the keyboard
way to remove.

The guard against a mistaken drop is undo rather than a confirmation: the
editor remembers the last removed key, deep copied, and the page a removal
returns to says "Removed" with an Undo button (POST /key/undo, guarded like
every change), which puts the key back exactly if its page exists and its tile
is still free, and is refused otherwise or when there is nothing to undo. Only
the last removal is kept, in memory.

Tests first, failing before the change: the hidden bin on the page, a removal
returning to the removed message with an undo form carrying the token, undo
restoring the key exactly and refusing a second time, refusing while the tile
is taken, and needing the token. In the browser pane, a recorded real mouse
drag fired dragstart on the tile and dragenter, dragover and drop on the bin,
which had appeared top right because the grid reached the bottom corner; the
key left the file and the page offered Undo, and Undo put the key back. Two
first attempts missed: one aimed before the saved message moved the grid down,
and one checked the file before the removal had saved.

### TE9 in part: a key made by keyboard alone

The author made a GoLand key using only the keyboard: an `open_application`
key on page 1, row 4, column 7, with the application `com.jetbrains.goland`,
the identifier the app picker fills in, and Use app icon ticked. The daemon
accepted the file and logged no icon failure. TE9's gate is a full page built
from scratch on each machine by keyboard; this is one key of it, recorded as
the author reported it, who found it smooth.

### A black strip on light keys, and keys framed in black

On light coloured keys the author saw a black strip at the bottom and right
only. The bytes sent were complete: a solid key encoded by the real encoder
decoded to 96 by 96 pixels of its colour to every edge. Probes on the deck,
with the daemon paused, then showed:

- With a one pixel line on each edge, only the bottom and right lines showed,
  with black beyond them, more below than to the right; the top and left lines
  were hidden. Frames one pixel apart put the first visible ring two to three
  pixels in at the top and left.
- Images of any width other than 96 drew skewed, with lines at angles, so the
  deck reads each row as exactly 96 pixels.
- Taller images, 96 by 104 to 96 by 120 with the extra rows at either end,
  changed nothing seen straight on: in the author's photograph every row showed
  the same strip, so the deck draws 96 by 96 and nothing more. An earlier
  impression that one helped was the viewing angle.

So the strip is the deck's own: each key's 96 pixel image is drawn a little up
and left of its window, and the window's edges bow. No image can fill it. The
author chose to frame every key in black so the edges look even and the bow is
hidden. `render.NewFramedRenderer` draws a key's content at a smaller size and
places it in a black square, for keys, navigation keys and blanks alike, so the
deck and the editor's previews agree; `render.Inset` gives the square's size
and place, refused if it does not fit or is too small to draw in. The XL's
values sit in the model table: an 84 pixel square, 8 from the left and 10 from
the top, thicker at the top and left that the deck hides. They are estimates
from the probes, to be adjusted on the deck. Tests first, both failing before
the code. The probe file was temporary and deleted; the probes ended at the
author's request, late at night.

On the deck the frame showed the cause of the bowing: the domed key caps act
as lenses and pull any straight edge near the rim in at its middle, so the
square's edges bowed in turn. The content is now shaped to cancel it: each
edge bulges outward on a parabola, three pixels at its middle and nothing at
the corners, and the corners are rounded with a radius of six. The square
became 82 pixels at 8, 9, so the bulge fits inside the frame on every side;
`render.Inset` gains `Bulge` and `CornerRadius`, refused if the bulge would
leave the key or the radius exceeds half the square. Tests first, failing
before the code; an enlarged rendering showed the intended shape before it
was installed.

On the deck three pixels overcorrected: the edges then bowed outward, and the
hard one pixel steps of the curve showed through the cap as a clear staircase.
The bulge is now a pixel and a half, `Bulge` a fraction, and each pixel the
edge crosses is sampled at sixteen points and blended with black by how much
of it lies inside, so the edge is smooth rather than stepped. The test that
checks the blend failed against a planted hard edge (one sample a pixel). The
author liked the look of a smooth, rounded tile: little buttons. Installed,
the author looked on the deck and approved it.

### Dark mode, after Thyme

At the author's request, late at night, the editor gained a dark mode in the
colours of the author's Thyme clock (its theme package): the panel #12131c,
the time text #f894f8 and the outline #f59ef3, with lilacs for quieter text
and shadows. It follows the system's setting; there is no switch of its own.

A plain swap of the punk palette could not work: ink is both the text and
the background of tags, tiles and buttons, and yellow is both a background
under ink text and text on ink. So every rule now names a role (page, text,
block, text on a block, highlight, notice, field, rule, accent and so on),
the light :root sets each role to the punk colour it had, and the dark
:root sets Thyme's. The tilts become none and the heading's misprint a glow.

Tests first, all three failing before the stylesheet changed: dark mode's
page, text and rule are Thyme's colours; every text pair meets 4.5 to 1 and
focus (outline or halo) 3 to 1 on every surface, in both palettes; and no
rule outside :root names a palette colour. Planted violations (a dim dark
text, a rule naming ink directly) failed them. In the browser, with the
colour scheme emulated, the pages grid and the key dialog read in both
modes, and light looks as it did.

The author then asked for a switch. The script is deferred, so a choice kept
only in the browser would be applied after the page is drawn: a dark choice on
a light system would flash pink on every page, at night. The switch instead
writes a cookie, `theme`, and the editor reads it as each page is served,
writing `data-theme` on the html element; only light and dark are taken. The
dark palette is set both for the system's dark (unless light is chosen) and
for a chosen dark, the two blocks kept identical by a test. The switch is a
button with the switch role and aria-checked, hidden until the script shows
it, as the drag hint is; without the script the system's setting holds.

Tests first, failing before the code: the switch on every page, hidden and
with no theme chosen without the cookie; the cookie's light and dark drawn by
the server and anything else ignored; the chosen dark matching the system's.
Planted violations (a cookie read taking any value, the two dark blocks
drifting apart) failed them. In the browser the switch turned the page dark,
set the cookie, and the next page arrived dark from the server; switching
back gave light.

### Brightness and off keys

The author asked for a key showing a brightness percentage, set by digit and
slider, that toggles the deck between its usual brightness and that level;
and an off key that turns every other key off and the deck down low.

Brightness is the whole deck's, and the daemon owns the deck, so these bricks
reach for no port: `brick.DeckControl` names a request (toggle brightness to a
percentage, or toggle off at one), and the daemon carries it out in its loop.
Both kinds live in `internal/bricks/deckcontrol`, levels 1 to 100: nought
would leave a deck too dark to find the key that brings it back.

While off, every key but the off key is drawn blank, including navigation and
any live key whose source publishes. A press of any key only wakes the deck,
since what it would do cannot be seen, and waking restores the brightness in
force before, dimmed or not. A new configuration wakes the deck, as the off
key may have moved. The chosen brightness is set again when the deck returns
from the dock. Pressing a brightness key at the level it set goes back to the
usual; at another level, goes to that one. A key without a label shows its
percentage, or Off.

In the editor any bounded integer gets a slider beside its number box: the
number is sent, the slider has no name, and the script keeps the two in step
and shows the slider, as it does in the key dialog.

Tests first, failing before the code: the bricks' requests, default and
bounds; the daemon dimming and restoring, blanking the other keys and keeping
a live key blank, waking on any press without pressing, restoring a dimmed
level, and a returning deck keeping it; the slider's markup and the script.
Planted violations failed them: no wake on press, a live key drawn while off,
a returning deck at the usual brightness, the wrong default. In the browser
the slider and number moved each other both ways.

On trying them the author asked for a new key's level to start somewhere
useful: 50 percent for brightness, 20 for off. The brick schema gained a
`default` tag, for numbers only and within their bounds, both checked when the
kind is registered; it is placed after bounds in the tag order. A new key's
form starts there, number and slider alike; a refused form keeps what was
typed. The tag only starts a form: a value left out of the configuration file
is still the brick's to handle, so the off key's own default rose from 10 to
20 to match. Tests first, failing before the code, and planted violations (no
bounds check on a default, no prefill) failed them; in the browser both new
key forms opened at their levels.

### Copying a page

The author asked to copy a page, both to a new page and onto an existing one.
Each page now has both under "Copy this page's keys": to a new page, at the
end and named as given, or onto any other page, listed by number and name
and never itself. Every key is cloned, as a single key copy is, and the
change goes through the store like any other save.

Copying onto a page keeps its keys. As with a single key, nothing is ever
silently overwritten: if any copied key would land on a key, the whole copy is
refused, the page is left unchanged, and the refusal names each clashing tile
counting from one. A page that no longer exists, or a page copied onto itself,
is refused in words too.

Tests first, failing before the code: a copy to a new page, a nameless one
refused, a copy onto another page beside its keys, a clash refused with its
tiles named and the file untouched, bad targets refused, and the page offering
both forms. A planted missing clash check failed them. A test that the copy's
parameters were not shared with the original was dropped: it read the keys
back from the file, which decodes fresh values, so it could never fail.

Trying it, the author's copy onto page two was refused for a clash at row
1, column 1: each page is marked there by a key, an open_url to a placeholder
address labelled with the page. The author asked for the marker to be a brick
of its own that copying leaves out. `page_marker` takes only an appearance,
does nothing when pressed, and without a label shows "Page" and its page's
number, the daemon knowing each key's page. Copying a page never copies a
marker, onto an existing page (which keeps its own, so markers never clash)
or to a new one (where a copied label would name the wrong page). Its kind
name lives beside `brick.PageMarker` so the editor can leave it out without
depending on the brick. Tests first, failing before the code; planted
violations (a marker copied, no page number on an unlabelled marker) failed
them. The author's existing markers are still open_url keys until changed.

### Brick drawer

The author asked for a drawer in the left gutter beside the pages that opens
on a click and offers bricks to drag: dropped on an empty tile, the brick opens
the normal key editor; an existing key is not replaced this way, and shows a
subtle cross to say so.

The drawer is an aside fixed to the window's left edge: a vertical "Bricks"
tab, a button with aria-expanded, that opens a panel listing every kind and
preset from the same list the key form chooses from. Dragging needs the
script, so the drawer is hidden until the script shows it; the grid's empty
tiles stay the way to add a key from the keyboard. While a brick is carried
the grid is marked placing: keys and paging tiles are dimmed and crossed in
the rule colour and refuse the drop, and only an empty tile takes it. The
panel fades and lets the drag through to tiles it covers on a narrow window,
found trying it in the browser pane. A drop opens that tile's key editor in
the dialog with the brick chosen, through an export the dialog gained; nothing
is saved until that form is. The existing key drag ignores a carried brick.

At the same time the author asked, through a subagent, for the page copy
forms to sit in a drawer too: closed by default, with a down chevron. They
are now in a details element with the chevron drawn in CSS, turning up when
open, its rotation stopped for reduced motion.

Tests first, failing before the code: the drawer's markup, hidden, with every
kind and preset; none without a page; the script's drop handling; the crosses
and the pass through in the stylesheet; the copy forms inside a closed
details. Planted violations (the drawer shown without the script; the copy
drawer open) failed them. In the browser pane, with simulated drags: an
occupied tile refused the brick and showed dimmed, an empty one took it and
opened its editor with the brick chosen and its level at 50, and the copy
drawer showed closed with its chevron.

### Drawers by size, and keys placed down a column

The author asked for the brick drawer to hold drawers of its own: 1x1 with
every existing brick, and a 1x2 and a 1x3 that add a D&D Beyond character
sheet and a Roll20 page, and a Discord server for the 1x3, down a column. A
group carried over the grid shows the tiles it would fill numbered 1 and 2, or
1 to 3; dropped, it opens one form for every part's details, whose parts can be
reordered.

The drawer's panel now holds three details elements. The 1x2 and 1x3 groups
need that many empty tiles from the drop tile down its column, each on the
deck and none kept for paging; carried, the tiles are outlined and numbered,
and a column without room refuses the drop. Dropped, the group's form opens in
the key dialog from GET /group: each part's label and icon (starting at Sheet
and wand, Roll20 and dice-5, Discord and brand-discord) and its preset's own
IDs, each field named for its part. Up and Down buttons, shown by the script,
move a part, its number following and focus staying on the button; the parts'
hidden order fields go in the order they sit. POST /group places every key or
none: an order that is not the group's parts, a field in error, a page gone, or
a tile off the deck, kept for paging or holding a key is refused in words, the
form kept as typed. Each part becomes an ordinary preset key, built as the key
form builds one, so it is re-edited like any other.

Roll20 gained a preset, roll20_campaign (open_campaign), at the campaign launch
address the author's own Roll20 keys use; the preset table had waited for a
known Roll20 address, and those keys are it.

Tests first, failing before the code: the Roll20 expansion and action; the
group form's parts, in order, with Discord only in the 1x3; an unknown size
refused; a group placed down the column in the chosen order as preset keys;
each refusal, with the file untouched and the form kept; the drawers by size;
and the script's stack marking. Planted violations (no check for a key in the
way, no check on the order) failed them; the first first passed, as the store
also refuses a key on a key, so the test now asks for the tile named. A
variable rename that caught the word "at" in two messages was caught by the
same test. In the browser pane, with simulated drags, a 1x3 numbered three free
tiles down column 3, refused a column without room, and opened its form, where
moving the sheet down put Roll20 first and renumbered the parts.

### Still owed

On the MacBook: logging out and in to see the daemon start at login (TE10);
the keyboard only editor run (TE9). On Windows, when the author turns to the
Dell: the app picker's Start menu listing, and launching a picked application. Everything on the Dell, including real
dock switches between the two machines.
