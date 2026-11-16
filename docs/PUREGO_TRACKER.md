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

## Checks

Run from the worktree, before signing:

| Name | Command |
| --- | --- |
| race | `go test -race ./<owned packages>/...` |
| lint | `golangci-lint run ./...` (the coordinator's copy is in the session scratchpad) |
| nocgo | `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./...` (from gate G0 on) |
| windows | `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./...` and the Windows lint run |
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
| C1 | `platform_bridge` setting | none | `internal/config/` setting code and tests | open | | |
| C2 | Bridge type, constructors, stubs, build lines, agreement helper | none | systemadapter: `bridge.go`, `cgo_absent_darwin.go`, `agreement_darwin_test.go`, the seven `*_purego_darwin.go` stubs, the build line in each cgo file, the constructors' signatures in every `*_windows.go` and cgo file | open | | |
| C3 | HID driver interface under the watcher | none | `internal/deck/hidtransport/` except the drivers H1 to H3 own | open | | |
| C4 | objcbridge package and the purego module | none | `internal/platform/objcbridge/`, `go.mod`, `go.sum`, `.golangci.yml` | open | | |
| W1 | Daemon reads the setting and passes it on | C1, C2, C3 | `cmd/the-front-buttons/main.go` and its tests | open | | |
| A1 | Clipboard through purego | C2, C4 | `clipboard_purego_darwin.go`, `clipboard_agreement_darwin_test.go` | open | | |
| A2 | Quit through purego | C2, C4 | `quit_purego_darwin.go`, `quit_agreement_darwin_test.go` | open | | |
| A3 | Keys and Accessibility trust through purego | C2, C4 | `keys_purego_darwin.go`, `keys_agreement_darwin_test.go` | open | | |
| A4 | Activation through purego | C2, C4 | `activation_purego_darwin.go`, `activation_agreement_darwin_test.go` | open | | |
| A5 | Badges through purego | C2, C4 | `badges_purego_darwin.go`, `badges_agreement_darwin_test.go` | open | | |
| A6 | Window cycling through purego | C2, C4 | `windows_purego_darwin.go`, `windows_agreement_darwin_test.go` | open | | |
| A7 | App icons through purego | C2, C4 | `appicons_purego_darwin.go`, `appicons_agreement_darwin_test.go` | open | | |
| X1 | objcbridge helpers asked for during wave 2 | C4 | `internal/platform/objcbridge/` (from G0 on) | open | | |
| H1 | IOKit driver: enumerate, open, reports out, features, close | C3, C4 | `iokit_darwin.go`, `iokit_darwin_test.go` | open | | |
| H2 | IOKit driver: input reports and removal | H1 | `iokit_input_darwin.go`, `iokit_input_darwin_test.go`, `reportqueue.go`, `reportqueue_test.go` | open | | |
| H3 | Windows HID driver | C3 | `winhid_windows.go`, `winhid_windows_test.go` | open | | |
| D1 | Docs for the setting and the new code | C1 | the files D1 owns above | open | | |
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

## Integration log

Kept by the coordinator. Each line: date, task, worker, what was integrated,
whole tree checks run.
