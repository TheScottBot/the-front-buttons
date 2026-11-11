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
