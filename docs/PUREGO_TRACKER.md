# Purego implementation tracker

The working record for [PUREGO_PLAN.md](PUREGO_PLAN.md). Several workers
build it at once, each taking a task from the table below, and this file is
how they share out the work, wait on each other and sign work off. The plan
says what is built and why; its Contracts section is what every task is
written against.

There is one copy of this file: the one in the main checkout, at
`docs/PUREGO_TRACKER.md` under the checkout's root. The coordinator gives each
worker that full path when starting it; a worker in a worktree edits that
copy, never the copy in its own worktree.

## Roles

- **Coordinator:** the main session. Starts the workers, accepts contract
  changes, integrates finished work into the main checkout, runs the checks
  on the whole tree, installs, and keeps the integration log.
- **Worker:** one task at a time, in its own git worktree. Signs with the
  name it was given when started, such as `worker-3`.

### Where worktrees live

Outside the checkout, in a folder beside it named
`the-front-buttons-worktrees`, one detached worktree per task named after
the task (`C1`, `A4`), made by the coordinator from the last gate's commit
before the worker starts. Never inside the checkout: the repository checks
scan everything under the checkout's root and read only its own
`.gitignore`, so a whole copy of the tree inside it fails them. Workers are
started without the agent tool's own worktree option, which puts worktrees
under `.claude/`, and are told their worktree's path. The coordinator removes
a worktree once its task is integrated.
- **Person:** commits at each gate, and does whatever needs hands: pressing
  keys on the deck, granting Accessibility, using the build day to day.

## How a worker works

1. **Read** the plan, this file, and the files the task owns.
2. **Pick** the first task whose status is `open` and whose dependencies are
   all `integrated`. Failing that, pick an `open` task whose dependencies are
   only `claimed` or `in progress` and whose row lists work that can be done
   while waiting.
3. **Claim** it with one Edit of its row, changing `open` to
   `claimed: <name> <date>`. Edit only replaces text that still matches, so
   if two workers try at once, one Edit fails: that worker reads the file
   again and picks another task. Never claim a task by rewriting the whole
   file.
4. **Work** in the worktree, on the files the task owns and no others. Tests
   come first, and each test is proven against a planted fault before the
   code it guards is trusted.
5. **Report progress** in the row's Progress cell (a short phrase and a
   percentage) at each milestone: tests written, plants caught, code passing,
   checks passing.
6. **Finish** by running the task's checks, then setting the status to
   `done` and signing the row (see Signing). The coordinator integrates the
   work and sets `integrated`.
7. **Hand back** a report to the coordinator: what changed, the plants and
   what caught each, anything left for another task, and the worktree's path.

### Blocked

A worker whose task needs another task's work that is not yet `integrated`:

1. Sets the status to `blocked on <task>` and keeps the claim.
2. Does the task's "while waiting" work from the task details below, which
   never needs the missing piece: tests against the contract, fakes,
   declarations of the C functions to be called, pure Go parts.
3. When that runs out, waits, checking the dependency's row every few
   minutes (Monitor with an until loop, not sleep) for up to 30 minutes.
4. If the dependency is still not `integrated`, writes what it finished in
   the Progress cell, sets the status to `paused: <name>`, and hands back.
   The coordinator starts it again once the dependency lands, and the same
   or another worker resumes from the Progress cell and the worktree.

### Signing

A row is signed when its task is done, in the Signed cell:

`<name>, <date>; tests <n> new; plants <caught>/<planted>; checks: race, lint, nocgo, windows, dupl`

Each check named was run and passed. Leave out a check that does not apply
(windows for a darwin only file) and add `deck` when the task needed the
deck and used it. Never sign for a check that was not run.

### Contract changes

Contracts are in the plan. A worker that finds one wrong or short adds a line
to the contract log below and carries on with what it can. The coordinator
accepts or refuses it, edits the plan, and notes which claimed tasks it
touches. A worker never changes a contract by itself.

## Rules every task keeps

- No git commits, pushes, branch changes or stashes. Work stays uncommitted in
  the worktree for the coordinator to integrate; the person commits at gates.
- Touch only the files the task owns. A file owned by another task, or frozen,
  is read only.
- UK spelling. No em or en dashes, and no runs of two hyphens in prose or in Go
  comments. Comments and docs say what the code does, never who asked or
  decided.
- Tests first, proven against planted faults. Only claim a check that was run.
- Never touch `~/Library/LaunchAgents`, the installed daemon or the install
  scripts. Only the coordinator installs.
- The deck and the desktop are shared (see Locks).
- Secrets never in config, logs, test output or reports.
- Scratch files (plant scripts, backups, built binaries) go in a folder of the
  worker's own name inside the session scratchpad, never at its top level,
  where two workers once overwrote each other's plant script.

## Checks

Run from the worktree, before signing:

| Name | Command |
| --- | --- |
| race | `go test -race ./<owned packages>/...` |
| lint | `golangci-lint run ./...` (the coordinator's copy is in the session scratchpad) |
| nocgo | `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./...` (from gate G0 on) |
| windows | with `GOOS=windows GOARCH=amd64 CGO_ENABLED=0` set for every command: `go list ./...` piped through `xargs go vet`, and the same list into the lint run. Take the list under GOOS=windows, so darwin only packages such as objcbridge are left out. |
| dupl | `scripts/check-duplicate-code.sh` |

## Locks

Claimed and released with one Edit, as tasks are.

| Lock | Guards | Holder |
| --- | --- | --- |
| deck | Opening or drawing on the physical deck. The installed daemon must be stopped by the person first. | free |
| desktop | Tests that act on the desktop: activating or quitting apps, posting keys, writing the clipboard, raising windows. Read only agreement tests need no lock. | free |

## Gates

Workers start each wave from the last gate's commit, so everything in it is
in their worktree.

| Gate | When | Then |
| --- | --- | --- |
| G0 | C1, C2, C3, C4 and W1 integrated; whole tree checks pass | person commits; wave 2 starts |
| G1 | every A task, H1 to H3, X1 and D1 integrated; agreement tests pass | person commits; H4 and V1 start |
| G2 | V1 decided in favour of purego | person tags v0.1.0 if wanted; R1 starts |

Between gates, a worker that needs work integrated since the last commit asks
the coordinator, who copies the main checkout's uncommitted changes into that
worktree.

## Files nobody else touches

| File | Owner |
| --- | --- |
| `go.mod`, `go.sum`, `.golangci.yml` | C4 |
| `cmd/the-front-buttons/main.go` | W1 |
| `internal/deck/hidtransport/hidtransport.go` | C3, then frozen |
| `internal/platform/ports.go` | frozen |
| the seven cgo files in systemadapter | C2 adds the build line, then frozen until R1 |
| `CHANGELOG.md`, `SECURITY.md`, `docs/CONFIGURATION.md`, `docs/SETUP.md`, the evaluation | D1 (other tasks leave notes for D1 in their Progress cell) |
| `docs/PUREGO_PLAN.md` | coordinator |

## Waves

```
wave 1   C1 ─┐
         C2 ─┼─ W1 ── G0
         C3 ─┤
         C4 ─┘
         H3 (from C3's contract)   D1 (drafting)

wave 2   A1 A2 A3 A4 A5 A6 A7   (each needs C2, C4)
         H1 ── H2               (need C3, C4)
         H3 continues   X1 as asked   D1 continues
                                                   ── G1
wave 3   H4 (deck)   V1 (a week of use)            ── G2
wave 4   R1
```

The longest chain is C4, H1, H2, H4: about six working days from the start,
against 13 to 15 days with one worker. Wave 1 runs four or five workers and
wave 2 five or six; more than that spends more on coordination than it saves.
A sensible wave 2 split: A1, A2 and A3 to one worker; A4 and A5 to one; A6;
A7; H1 then H2; H3.

## Tasks

| ID | Task | Depends on | Owns | Status | Progress | Signed |
| --- | --- | --- | --- | --- | --- | --- |
| C1 | `platform_bridge` setting | none | `internal/config/` setting code and tests | integrated | 100%: setting, 6 tests, plants 4/4 caught; `PlatformBridge: ""` added to the 21 literals (accepted ask), eight wrapped for line length. D1 note: optional top level `platform_bridge: cgo` or `purego`, absent means cgo, any other value refused (`unknown_bridge`, wrong type `wrong_type`), kept on editor saves, written only when set | worker-1, 2026-11-16; tests 6 new; plants 4/4; checks: race, lint, windows, dupl |
| C2 | Bridge type, constructors, stubs, build lines, agreement helper | none | systemadapter: `bridge.go`, `cgo_absent_darwin.go`, `agreement_darwin_test.go`, the seven `*_purego_darwin.go` stubs, the build line in each cgo file, the constructors' signatures in every `*_windows.go` and cgo file | integrated | 100%; main.go passes BridgeCgo everywhere for W1 to replace, CgoAvailable for W1's log; for D1: a CGO_ENABLED=0 macOS build gives the purego stubs (ErrBridgeUnavailable) until A1 to A7 land; see contract log | worker-2, 2026-11-16; tests 8 new; plants 7/7; checks: race, lint, nocgo, windows, dupl |
| C3 | HID driver interface under the watcher | none | `internal/deck/hidtransport/` except the drivers H1 to H3 own | integrated | driver interface, hidapi driver and contract in; checks passing, 100%. For D1: no fallback files, drivers add themselves to `builtDrivers` (see contract log); `Watcher.DriverChoice()` says asked and used driver for W1 to log; a `CGO_ENABLED=0` build of hidtransport now compiles, with no driver until H1 and H3 | worker-3, 2026-11-16; tests 15 new; plants 7/7; checks: race, lint, nocgo, windows, dupl |
| C4 | objcbridge package and the purego module | none | `internal/platform/objcbridge/`, `go.mod`, `go.sum`, `.golangci.yml` | integrated | all checks pass, 100%; additions in the contract log; for D1: SECURITY.md could name purego and the libraries objcbridge opens (CoreFoundation, Foundation, AppKit, ApplicationServices, libobjc; IOKit later), not demanded by its test | worker-4, 2026-11-16; tests 38 new; plants 6/6; checks: race, lint, nocgo (objcbridge test and vet only; whole tree from G0), windows, dupl |
| W1 | Daemon reads the setting and passes it on | C1, C2, C3 | `cmd/the-front-buttons/main.go` and its tests | integrated | 100%: bridge read once at start (`OnDisk`, missing or unreadable file gives cgo), passed to every constructor, `NewWatcher` and the `-accessibility-check` one shot; start line `platform bridge` (asked, platform, hid, fall backs); reload line once per change; new files `cmd/the-front-buttons/bridge.go` and `bridge_test.go`. D1 note: log lines `platform bridge` (INFO, WARN on a fall back), `platform bridge not read from the configuration file; using cgo`, `platform bridge change takes effect at the next start` (running, file) | worker-5, 2026-11-16; tests 9 new; plants 11/11; checks: race, lint, nocgo, windows, dupl |
| A1 | Clipboard through purego | C2, C4 | `clipboard_purego_darwin.go`, `clipboard_agreement_darwin_test.go` | integrated | 100%: adapter (`setString:forType:` on the general pasteboard, as cgo: text ends at a NUL, not UTF-8 refused after clearing); non acting tests write a uniquely named pasteboard; two acting agreement tests (write and read back through each bridge, restoring the clipboard) wait for the desktop lock. X1 helpers `Bind`, `Class`, `ClassSelector`, `InstanceSelector` (`binding_darwin.go`) and `RunningApplications`, `RunningApplicationsWithBundleID` (`runningapplications_darwin.go`), counted here; WriteClipboard and QuitApplication entries removed from `bridge_darwin_test.go` as accepted. Plant not caught: an owned string left unreleased (here and in the helper). For D1: a purego clipboard loads AppKit | worker-7, 2026-11-16; tests 14 new; plants 8/10; checks: race, lint, nocgo, windows, dupl |
| A2 | Quit through purego | C2, C4 | `quit_purego_darwin.go`, `quit_agreement_darwin_test.go` | integrated | 100%: adapter (by bundle ID, or by name or bundle file name with `isEqualToString:`, then `terminate`; NUL and not UTF-8 as cgo); non acting tests list the candidates without terminating (the Dock, and every running app whose name and bundle file name differ); two acting agreement tests (unknown app quits nothing; Calculator opened and quit by bundle ID, name and name.app through each bridge) wait for the desktop lock. Plant not caught: an owned string left unreleased; `forceTerminate` for `terminate` only an acting run could tell | worker-7, 2026-11-16; tests 8 new; plants 5/6; checks: race, lint, nocgo, windows, dupl |
| A3 | Keys and Accessibility trust through purego | C2, C4 | `keys_purego_darwin.go`, `keys_agreement_darwin_test.go` | integrated | 100%: adapter (AXIsProcessTrusted, the prompt through AXIsProcessTrustedWithOptions with a released dictionary, CGEventCreateKeyboardEvent, CGEventSetFlags, CGEventPost at the HID tap, both events released; a failed binding answers ErrBridgeUnavailable without asking or posting); trust agreement read only; events and prompt options checked through recorders in place of CGEventPost and the prompt; one acting test (F20 and Shift F20 through each bridge, only when trusted) waits for the desktop lock. Uses worker-9's `CFBoolean`. Plants not caught: the session tap for the HID tap (acting only, and even then unseen); trust always true is caught only on a Mac that is not trusted. PressKey and PressShortcut entries, the RequestAccessibility call and the stub trust test removed from `bridge_darwin_test.go`. For R1: `pureFreshProcessTrusted` and `pureAccessibilityAskInterval` repeat the cgo file's, to drop with it. For D1: a purego key presser loads ApplicationServices | worker-7, 2026-11-16; tests 8 new; plants 9/10; checks: race, lint, nocgo, windows, dupl |
| A4 | Activation through purego | C2, C4 | `activation_purego_darwin.go`, `activation_agreement_darwin_test.go` | integrated | 100%: adapter through objcbridge (AX focused application, AXFrontmost, AXUIElementGetPid; NSRunningApplication by bundle ID, by process, activateWithOptions:), Accessibility calls behind a seam so non acting tests count retains with real elements; activation and badges entries removed from C2's stub test; uses worker-7's binding and running applications files and worker-9's cfboolean file, copied unchanged. Not exposed, so not compared: the frontmost read through cgo (static C). The two acting tests compile and skip. For D1: nothing new to document | worker-8, 2026-11-16; tests 8 new (2 acting, skipped); plants 11/11; checks: race, lint, nocgo, windows, dupl |
| A5 | Badges through purego | C2, C4 | `badges_purego_darwin.go`, `badges_agreement_darwin_test.go` | integrated | 100%: adapter through objcbridge (NSRunningApplication for the Dock, AXChildren of the Dock and each list, AXStatusLabel and AXURL per item, CFGetTypeID against CFURLGetTypeID, NSBundle bundleWithURL: and bundleIdentifier); trust from pureAccessibilityTrusted, so until A3 lands purego reports ErrBadgesNotPermitted and the agreement test fails on a trusted machine; this test process is not trusted, so both bridges agree on ErrBadgesNotPermitted and the purego Dock read with trust forced gives "the Dock could not be read". Reader proven against a stand in Dock tree of real CF objects with retain counts. For D1: nothing new to document | worker-8, 2026-11-16; tests 7 new; plants 14/14; checks: race, lint, nocgo, windows, dupl |
| A6 | Window cycling through purego | C2, C4 | `windows_purego_darwin.go`, `windows_agreement_darwin_test.go` | integrated | 100%: adapter in; non acting tests on a fake desktop (made up windows on elements of unused process IDs) plus an SDK header check of every AX name; real window agreement tests skip here, as Accessibility is not granted to the test binary; raise agreement test acting, not run. CycleWindows entry removed from the stubs test (accepted rule). X1 files: CFBoolean (own), worker-7's binding and running application files copied unchanged. Misses: frontmostApplication swapped for menuBarOwningApplication (same answer in practice), processIdentifier read as int64 (harmless on arm64). For D1: purego window cycling uses the purego Accessibility check, the stub's false until A3 lands | worker-9, 2026-11-16; tests 14 new; plants 18/20; checks: race, lint, nocgo, windows, dupl |
| A7 | App icons through purego | C2, C4 | `appicons_purego_darwin.go`, `appicons_agreement_darwin_test.go` | integrated | 100%: icons agree pixel for pixel through both bridges for 9 installed apps (by bundle ID, name and path) and give the same not found error for 5 missing ones; uses worker-7's `Bind`, `Class`, `ClassSelector`, `InstanceSelector` (copy of `binding_darwin.go` in the worktree, worker-7's to keep). `TestPuregoStubs_ReportTheBridgeUnavailable` now fails on its Icon entry, as worker-7's open ask says; run with `-skip` until accepted, entry left alone. Not caught: the bitmap's release left out (a leak no read shows). D1 note: purego icons draw with AppKit in the calling thread, no cgo; only arm64 run, Intel (NSRect on the stack) unrun | worker-10, 2026-11-16; tests 7 new; plants 8/8; checks: race, lint, nocgo, windows, dupl |
| X1 | objcbridge helpers asked for during wave 2 | C4 | `internal/platform/objcbridge/` (from G0 on) | integrated | | |
| H1 | IOKit driver: enumerate, open, reports out, features, close | C3, C4 | `iokit_darwin.go`, `iokit_darwin_test.go` | integrated | 100%: listing, open seized (kIOHIDOptionsTypeSeizeDevice, as go-hid's hidapi opens, per the accepted decision; closed with the same option), reports out, features, close; parity equal (deck 1 listing; every vendor 36 listings of 24 devices, hidapi's one listing per usage pair mirrored); untagged core `iokitcore.go` with `iokitcore_internal_test.go`; binds through X1's `objcbridge.Bind` (`binding_darwin.go` copied unchanged from A123 to compile). `experiment_test.go` (allowed for this only): the experiments take `THE_FRONT_BUTTONS_EXPERIMENT_DRIVER` (cgo or purego, default cgo) and fail rather than fall back. Opening not run: H4 on the deck. D1 note: both drivers seize the deck; TESTING.md could name the new variable | worker-11, 2026-11-16; tests 27 new; plants 9/9; checks: race, lint, nocgo, windows, dupl |
| H2 | IOKit driver: input reports and removal | H1 | `iokit_input_darwin.go`, `iokit_input_darwin_test.go`, `reportqueue.go`, `reportqueue_test.go` | integrated | 100%: a run loop thread per open device (locked goroutine, CFRunLoop kept alive by an empty source), input reports into a queue of 30 read without waiting, removal marks the handle gone and stops the loop, close unschedules, stops, waits for the thread, then IOHIDDeviceClose; callbacks made once per process, devices found by a context number. Thread, registry and run loop outcome in `iokitcore.go`; real CFRunLoop and both callbacks through the C ABI tested on the Mac with no device. Reports from a real device and unplugging are for H4 | worker-11, 2026-11-16; tests 17 new; plants 6/6; checks: race, lint, nocgo, windows, dupl |
| H3 | Windows HID driver | C3 | `winhid_windows.go`, `winhid_windows_test.go` | integrated | 100%: driver in; pure logic (padding, report ID, error mapping, paths, struct layouts, the handle's pending read and removal state) in untagged `winhidcore.go` and `winhidcore_internal_test.go`, run on macOS. Device tests in `winhid_windows_test.go` skip without THE_FRONT_BUTTONS_EXPERIMENT and wait for the Dell (with the contract test's purego run). For D1: SECURITY.md should name hid.dll and setupapi.dll (kernel32 through x/sys) | worker-6, 2026-11-16; tests 30 new; plants 9/9; checks: race, lint, nocgo, windows, dupl |
| D1 | Docs for the setting and the new code | C1 | the files D1 owns above | integrated | 100%: CONFIGURATION.md (Platform bridge section), SECURITY.md (System libraries without cgo, seized opening, limitations), CHANGELOG.md (three entries), TESTING.md (experiment driver variable, as allowed) and the evaluation's "Purego, 16 November 2026" section; read only agreement tests rerun for the evaluation (icons pixel for pixel pass; window lists and 7 acting tests skip). SETUP.md unchanged (still right for the default build) | worker-12, 2026-11-16; tests 0 new (docs only); plants 0/0; checks: repositorychecks (prose, security profile), reference, go test ./... |
| H4 | Both drivers on the deck: contract, experiments, parity | H1, H2, W1; deck lock | the evaluation's deck section (with D1) | open | | |
| V1 | A week of use on purego, then decide | G1 | the evaluation's verdict | open | | |
| R1 | Remove cgo, go-hid and the setting | G2 | everything cgo; a schema migration | open | | |

## Task details

Each lists what done means, the plants to prove its tests, and what can be
done while waiting.

### C1: the setting
- **Done:** `platform_bridge` decodes, encodes, is absent from a file saved
  without it, defaults to cgo, refuses any other value with the field named;
  the editor's saves keep it.
- **Plants:** default purego; any value accepted; the setting dropped on save.
- **While waiting:** nothing to wait for.

### C2: the bridge in systemadapter
- **Done:** everything under "The bridge in systemadapter" in the contracts;
  `CGO_ENABLED=0 go build ./...` passes for darwin with the cgo files left
  out; every stub returns `ErrBridgeUnavailable`; the agreement helper skips
  acting tests without `THE_FRONT_BUTTONS_ACTING`.
- **Plants:** a cgo file without its build line (the nocgo build must fail);
  a constructor ignoring the bridge; the acting guard never skipping.
- **While waiting:** nothing to wait for.

### C3: the driver interface
- **Done:** go-hid moved into `hidapi_driver.go` behind the interface with no
  change in behaviour; every existing hidtransport test passes unchanged; the
  driver contract test passes against a fake driver.
- **Plants:** the write retry skipped; `errHandleGone` not reported as
  disconnected; the panic recovery lost.
- **While waiting:** nothing to wait for.

### C4: objcbridge
- **Done:** every function in the contract, each with a test that runs on the
  Mac; purego added to `go.mod`; depguard keeps purego to objcbridge and
  hidtransport; `unsafe` allowed only there.
- **Plants:** a string conversion losing non ASCII text; Release not
  releasing (counted with `CFGetRetainCount`); a missing constant not
  reported; depguard letting a brick import purego.
- **While waiting:** nothing to wait for.

### W1: the daemon
- **Done:** the bridge is read once at start from the file and passed to
  every constructor and `NewWatcher`; the start is logged with the bridge;
  a change while running is logged as taking effect at the next start.
- **Plants:** the setting ignored; the log line missing.
- **While waiting:** tests for the log lines against a fake configuration.

### A1 to A7: the adapters
- **Done:** the stub replaced; the adapter's agreement test passes, giving
  the same answer through both bridges (read only parts without a lock,
  acting parts under the desktop lock); its existing tests pass through
  purego.
- **Plants:** at least one per binding: a wrong selector, a wrong argument
  type, a result not released, and for A7 NSSize passed for NSRect.
- **While waiting (on C4):** write the agreement test; declare each C
  function and selector the adapter calls, against the objcbridge contract,
  so it compiles the moment C4 lands; work out what each call returns from
  the cgo file and Apple's documentation.
- **Shared code:** a helper two adapters would both write goes to X1, never
  into an adapter.

### X1: helpers asked for
- **Done:** each helper asked for in the contract log, added with its test.
  Claimed afresh for each batch of asks; the asking worker is blocked on it
  only for that helper.

### H1: IOKit driver, out
- **Done:** as "The macOS HID driver" in the plan, apart from input reports
  and removal; enumeration gives the same connections and paths as the hidapi
  driver (a parity test that only lists, needing no lock).
- **Plants:** report types swapped; the report ID byte dropped; the IOReturn
  code lost from an error.
- **While waiting (on C4):** declare the IOKit functions; write the parity
  and contract tests.

### H2: IOKit driver, in
- **Done:** a run loop thread per open device; input reports queued and read
  without waiting; removal makes every later call return `errHandleGone`;
  closing ends the thread.
- **Plants:** the queue never filled; the queue unbounded; removal ignored;
  the thread left running after close.
- **While waiting (on H1):** the bounded queue in `reportqueue.go` and its
  tests, pure Go and needing nothing else.

### H3: Windows driver
- **Done:** setupapi enumeration and hid.dll reports with overlapped reads,
  behind the interface; lint and vet for Windows pass; tests that need a
  device are skipped off Windows and recorded for the Dell.
- **Plants:** the same three as H1, checked by unit tests on the path and
  report building, which run anywhere.
- **While waiting:** nothing blocks it once C3's contract is fixed; it can
  start in wave 1 against the contract text.

### D1: docs
- **Done:** the configuration guide describes `platform_bridge`; the
  security profile names purego and what it loads; the changelog and
  evaluation record the work; SETUP.md loses nothing until R1.
- **While waiting:** draft from the plan, then fill in from the Progress
  notes of finished tasks.

### H4: on the deck
- **Done:** the driver contract and both experiments run with each driver on
  the physical deck, under the deck lock, with the person at it; the numbers
  recorded side by side in the evaluation.

### V1: a week of use
- **Done:** the person uses `platform_bridge: purego` daily for a week, and
  the log, the experiments and the agreement tests are compared; the verdict
  recorded.

### R1: removal
- **Done:** as phase 7 in the plan.

## Contract log

Append only. Each line: date, worker, contract, change asked for, why;
then the coordinator's `accepted` or `refused`, with the tasks it touches.

- 2026-11-16, coordinator, files nobody else touches: C2 and C3 change
  constructor signatures that `cmd/the-front-buttons/main.go` and its tests
  call, so each may change those call sites only, passing
  `systemadapter.BridgeCgo` or `hidtransport.DriverHIDAPI`, so the tree
  builds; W1 then owns the file. Accepted; touches C2, C3, W1.
- 2026-11-16, worker-1, the setting (C1) and C1's Owns cell: the new
  `Configuration.PlatformBridge` field makes exhaustruct report 21 literals
  outside `internal/config/`: 15 in `internal/daemon/*_test.go`, and in the
  editor `deckrender.go` (2), `reorder.go`, `pages.go` and
  `icons_pages_test.go` (2). Asked: let C1 add `PlatformBridge: ""` to those
  literals only, with no other change, as the C2 and C3 call site allowance
  does; the other way, an exhaustruct ignore for `config.Configuration`,
  would need C4's `.golangci.yml` and lose the check for every later field.
  Why: lint on `./...` cannot pass otherwise. Accepted: those literals
  only, adding `PlatformBridge: ""` and nothing else; touches C1, and W1
  later where it builds a Configuration.
- 2026-11-16, worker-4, the bridge helpers (C4): additions to the contract,
  nothing in it changed. (1) Exported library paths `PathCoreFoundation`,
  `PathFoundation`, `PathAppKit`, `PathApplicationServices`, `PathIOKit`
  for `Framework`, so no two adapters spell a path. (2) `type AXError int32`
  (an error; `Error()` names the code) with `ErrAXIllegalArgument`,
  `ErrAXCannotComplete`, `ErrAXAttributeUnsupported`,
  `ErrAXActionUnsupported`, `ErrAXAPIDisabled`, `ErrAXNoValue`; every AX
  helper wraps one, read with `errors.As`, and a nil element gives
  `ErrAXIllegalArgument` without calling macOS, a nil value from a
  successful copy `ErrAXNoValue`. (3) `Function` panics on a missing
  function, as purego's `RegisterLibFunc` does; `Constant` is the call that
  reports `ErrSymbolMissing`. (4) Note, no change: AX attribute and action
  names such as `kAXFocusedApplicationAttribute` are `CFSTR` macros, not
  exported symbols, so `Constant` cannot read them; adapters pass the
  string ("AXFocusedApplication") to `AXCopy`. Real exported constants such
  as `kAXTrustedCheckOptionPrompt` do read. (5) depguard also permits
  `internal/platform/systemadapter`, since adapters send messages with
  purego's objc package directly; the plan's layout row names only
  objcbridge and hidtransport. (6) `objcbridge` added to the repository
  check's permitted operating system directories in
  `internal/repositorychecks/operatingsystem.go`, with a test; without it
  `TestRepository_HasNoViolations` fails on every `_darwin.go` file.
  Accepted, all six. On (3): a missing function must never panic the
  daemon, so each adapter and the IOKit driver bind their functions once,
  in one place that recovers a panic and then answers
  `ErrBridgeUnavailable`; touches A1 to A7, H1, H2.
- 2026-11-16, worker-2, the bridge in systemadapter (C2): additions, and
  one reading of the contract. (1) `newPure<Name>` takes the same arguments
  as its constructor less the bridge: `newPureKeyPresser(watchContext,
  granted)` and `newPureWindowCycler(ask)`, as the real adapters need the
  watcher and the prompt; the other five take none. The key presser stub
  also has `pureAccessibilityTrusted() bool`, and its `ForThisPlatform` and
  `DescribeShortcut`, which reach no system call and return no error,
  answer as the cgo adapter does. (2) The cgo files' constructors became
  unexported `cgo<Name>` helpers returning their concrete types, and the
  eight exported constructors dispatch in a new `bridge_darwin.go`, which
  compiles with and without cgo; `cgo_absent_darwin.go` gives the helpers
  in a build without cgo. (3) `CgoAvailable` is a const in two new files,
  `cgo_present.go` (`cgo`) and `cgo_missing.go` (`!cgo`), so it exists on
  every system for W1's log. (4) `AskingKeyPresser` in `bridge.go` is the
  key presser interface (KeyPresser, ShortcutPresser,
  RequestAccessibility). (5) The Windows constructors take the bridge and
  keep their concrete return types, which satisfy the same ports. (6) Note,
  no change: Go leaves out a file importing "C" when cgo is off whatever
  its build line, so the nocgo build cannot catch a cgo file missing
  `//go:build darwin && cgo`; `bridge_darwin_test.go` (new) checks the
  line by parsing the package's files. Touches W1, A1 to A7. Accepted, all
  six.
- 2026-11-16, worker-3, the HID driver (C3): two additions and one
  departure, no signature in the contract changed. (1) No
  `driver_absent_*.go` files: each driver's own file adds itself from one
  init line, `func init() { builtDrivers[DriverPurego] = &iokitDriver{} }
  //nolint:gochecknoinits // registers the driver`, and `chooseDriver` in
  `drivers.go` picks the one asked for or else the first built in of
  `DriverHIDAPI`, `DriverPurego`, or `ErrNoDriver`. Why: absent files keyed
  by build constraints would need H1 and H3 each to edit the same shared
  file's constraint; this way they add only their own file. (2)
  `Watcher.DriverChoice() DriverChoice` (`Asked`, `Using`, `Unavailable`,
  `FellBack()`) so the daemon can log a fall back, as the plan asks;
  NewWatcher has no logger. With no driver, `Enumerate` fails with
  `ErrEnumerationFailed` and `Open` with `deck.ErrDisconnected`, both
  wrapping `ErrNoDriver`. (3) `Close` reports `errHandleGone` as
  `deck.ErrDisconnected`; hidapi never returns it, so nothing changes today.
  Touches H1, H3 (the registration line), W1 (logging the choice).
  Accepted, all three.
- 2026-11-16, worker-7, X1 helper (running applications), for A2 with A4,
  A5 and A6 calling the same messages: in
  `internal/platform/objcbridge/runningapplications_darwin.go`,
  `func RunningApplications() []objc.ID` (`[[NSWorkspace sharedWorkspace]
  runningApplications]`) and `func RunningApplicationsWithBundleID(bundleID
  string) []objc.ID` (`[NSRunningApplication
  runningApplicationsWithBundleIdentifier:]`), each NSRunningApplication
  not retained, so both are called inside `WithAutoreleasePool` and the
  items used before it drains; an empty or unknown bundle ID gives none.
  NSArray needs no helper: it is toll free bridged, so `CFArrayItems`
  lists one. Why: quit, activation, badges and windows send these same
  messages, and dupl fails on a second copy.
- 2026-11-16, worker-7, the stubs' test (C2's `bridge_darwin_test.go`):
  `TestPuregoStubs_ReportTheBridgeUnavailable` calls every purego method
  expecting `ErrBridgeUnavailable`, and
  `TestAccessibilityTrusted_IsFalse_ThroughTheStub` expects false. Once a stub is replaced those calls act:
  WriteClipboard writes "text" to the clipboard, QuitApplication quits
  Finder, RequestAccessibility shows the Accessibility prompt and starts the
  watcher, PressKey posts F13 when trusted; and the tests fail. Asked: let
  each A task remove its own adapter's entries (and A3 the
  RequestAccessibility call and the stub trust test) from that file, nothing
  else, so `go test ./internal/platform/...` never acts on the desktop.
  Until accepted, worker-7 runs the systemadapter tests with
  `-skip 'TestPuregoStubs_ReportTheBridgeUnavailable|TestAccessibilityTrusted_IsFalse_ThroughTheStub'`.
  Touches A1 to A7.
- 2026-11-16, worker-7, X1 helper (the recovering binder every adapter
  needs, from worker-4's (3) above), in
  `internal/platform/objcbridge/binding_darwin.go`:
  `var ErrBindingFailed = errors.New("binding failed")`;
  `func Bind[T any](bind func() (T, error)) func() (T, error)` runs bind
  once, on the first call, recovers a panic (Function's missing function)
  into an error wrapping ErrBindingFailed, wraps any error bind returns
  the same way, and gives that same result to every later call;
  `func Class(name string) (objc.Class, error)` (no such class wraps
  ErrBindingFailed); `func ClassSelector(class objc.Class, name string)
  (objc.SEL, error)` and `func InstanceSelector(class objc.Class, name
  string) (objc.SEL, error)` register the selector and check, with
  `respondsToSelector:` and `instancesRespondToSelector:`, that the class
  or its instances answer it, so a misspelt or missing selector is an
  ErrBindingFailed at bind time rather than an unrecognised selector
  exception that aborts the daemon. An adapter answers
  `fmt.Errorf("%w: %w", ErrBridgeUnavailable, bindError)`. Why: A1 to A7,
  H1 and H2 each need it, and dupl fails on a second copy.
- 2026-11-16, worker-9, X1 helper (Core Foundation booleans), for A6 with
  A3 (the trust prompt's dictionary) and A4 (AXFrontmost) setting the same
  values: in `internal/platform/objcbridge/cfboolean_darwin.go`,
  `func CFBoolean(value bool) (CFTypeRef, error)`, giving kCFBooleanTrue or
  kCFBooleanFalse, read once with `Constant`; not owned, never released; a
  missing constant gives `ErrSymbolMissing`, never a panic. A6 also uses
  worker-7's `RunningApplications() []objc.ID` from
  `runningapplications_darwin.go` with that exact signature and file name.
  Why: three adapters pass these constants, and each would otherwise read
  them itself.
- 2026-11-16, worker-10, X1 helper (binding once without a panic), for A7
  with every A task and H1 needing the same, as the accepted C4 entry (3)
  asks: in `internal/platform/objcbridge/bindonce_darwin.go`,
  `func BindOnce[T any](bind func() (T, error)) func() (T, error)`. The
  returned function runs bind on its first call only and gives every call
  the same result; a panic inside bind (such as `Function`'s for a missing
  C function) is recovered and given as an error wrapping
  `ErrSymbolMissing`. The adapter wraps any error from it in
  `ErrBridgeUnavailable`. Why: each adapter must bind once in one place
  that recovers a panic, and dupl fails on a second copy of that guard.
- 2026-11-16, worker-10: the line above is withdrawn; it repeats
  worker-7's `Bind`, logged first. A7 uses worker-7's `Bind`, `Class`,
  `ClassSelector`, `InstanceSelector` and `ErrBindingFailed` from
  `binding_darwin.go` with those exact signatures.
- 2026-11-16, coordinator: worker-7's three entries (RunningApplications,
  the stubs' test, Bind and its selector checks) and worker-9's CFBoolean
  are accepted; worker-10's BindOnce stands withdrawn in favour of Bind. On
  the stubs' test: each A task removes its own adapter's entries from
  `TestPuregoStubs_ReportTheBridgeUnavailable` (and A3 the
  RequestAccessibility call and the stub trust test) and nothing else, so no
  test run without `THE_FRONT_BUTTONS_ACTING` ever acts on the desktop.
  Until a task has done so, its worker skips that test rather than running
  it against a real adapter. Touches A1 to A7.
- 2026-11-16, coordinator, the macOS HID driver (H1): go-hid opens the deck
  seized (kIOHIDOptionsTypeSeizeDevice), so the IOKit driver opens it the
  same way, for a like for like comparison; the plan's "without seizing"
  was wrong. And `experiment_test.go` takes its driver from
  `THE_FRONT_BUTTONS_EXPERIMENT_DRIVER` (cgo or purego, default cgo), so H4
  runs both drivers on one build. Accepted; both given to worker-11 under H1.

## Integration log

Kept by the coordinator. Each line: date, task, worker, what was integrated,
whole tree checks run.

- 2026-11-16, wave 1 started from `abac687`: worktrees C1 to C4 made beside
  the checkout.
- 2026-11-16, C1, worker-1: the setting, its tests and the 21 literals, copied
  into the main checkout; whole tree race tests, lint and dupl pass.
- 2026-11-16, C4, worker-4: purego in go.mod, objcbridge, the depguard rule
  and the permitted directory, copied into the main checkout; whole tree race
  tests, objcbridge without cgo, lint, Windows vet and lint, and dupl pass.
- 2026-11-16, C2, worker-2, and C3, worker-3: the bridge, the stubs and the
  build lines, and the driver interface with the hidapi driver behind it,
  copied into the main checkout. Whole tree checks with C1 to C4 together:
  race tests, lint, dupl, and for the first time the whole tree without cgo
  (`CGO_ENABLED=0 go build ./...` and `go vet ./...`, cmd included), and
  Windows vet and lint with hidtransport included; all pass.
- 2026-11-16, worktrees W1 and H3 made from `abac687` with the main checkout's
  uncommitted C1 to C4 work copied in.
- 2026-11-16, W1, worker-5: the daemon reads the setting at start, passes it
  to every constructor and the watcher, and logs it; copied into the main
  checkout. Whole tree race tests, lint, dupl, nocgo, Windows vet, lint and
  build all pass. Installed on the arm64 Mac: the log reads `platform
  bridge` asked cgo, platform cgo, hid cgo, and the deck connected. Gate G0
  reached; waiting on the person's commit. For A3: `-accessibility-check`,
  spawned by the daemon without `-config`, reads the default file's bridge.
- 2026-11-16, gate G0 committed as `67ffac5`.
- 2026-11-16, H3, worker-6: the Windows HID driver (`winhid_windows.go`,
  `winhidcore.go` and their tests), copied into the main checkout; deck race
  tests, lint, dupl, nocgo, Windows vet, lint, test build and daemon build
  pass. No Windows call has run yet: the Dell runs the WinHID tests, the
  on the deck tests under `THE_FRONT_BUTTONS_EXPERIMENT`, and an unplug by
  hand.
- 2026-11-16, wave 2 started from `67ffac5` with H3's files copied in:
  worktrees A123 (A1, A2, A3), A45 (A4, A5), A6, A7 and H12 (H1 then H2).
  X1: a worker needing a helper two adapters share adds it to objcbridge
  in its own file named for the helper, after logging its signature in the
  contract log; the coordinator keeps one copy at integration.
- 2026-11-16, A1 to A7 and X1 (worker-7, worker-8, worker-9, worker-10): the
  seven purego adapters with their agreement tests, and the shared helpers
  (`binding_darwin.go`, `runningapplications_darwin.go` from worker-7,
  `cfboolean_darwin.go` from worker-9, one copy each), copied into the main
  checkout. With every adapter real, `TestPuregoStubs_ReportTheBridgeUnavailable`
  and `TestAccessibilityTrusted_IsFalse_ThroughTheStub` had no entries left
  and are removed. Whole tree race tests, platform tests without cgo, lint,
  dupl, nocgo, Windows vet and lint pass. No acting test run yet; the read
  only agreement tests that need Accessibility skipped (not granted to the
  test binary).
- 2026-11-16, H1 and H2, worker-11: the IOKit driver, opening the deck
  seized as go-hid does, its input thread and report queue, and the
  experiments choosing a driver by `THE_FRONT_BUTTONS_EXPERIMENT_DRIVER`,
  copied into the main checkout. Listing parity with hidapi: equal paths
  for all 24 devices. Whole tree race tests, deck and platform tests
  without cgo, lint, dupl, nocgo, Windows vet, lint and build pass. The
  deck was never opened; H4 runs the contract and experiments on it.
- 2026-11-16, D1, worker-12: the configuration guide, security profile,
  changelog, testing guide and evaluation, copied into the main checkout;
  repository and reference checks pass.
- 2026-11-16, installed on the arm64 Mac with every wave 2 task in: the log
  reads `platform bridge` asked cgo, platform cgo, hid cgo, and the deck
  connected. Gate G1 reached apart from the agreement tests that need
  Accessibility and acting; waiting on the person's commit.
