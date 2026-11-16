# Plan: calling macOS without cgo

The Front Buttons calls macOS through cgo in two places: the HID library
github.com/sstallion/go-hid, which bundles hidapi's C code, and seven
Objective-C files in `internal/platform/systemadapter/`. Both need a C
compiler, so building needs Xcode's Command Line Tools. This plan replaces
both with github.com/ebitengine/purego, which calls system frameworks from Go
with no C compiler, behind a setting that picks either way at start, so the
two can be compared on the physical deck before the cgo code is removed.

## Why purego, not a Rust port

A Rust build on macOS still links through the system `cc` and the macOS SDK,
both from the Command Line Tools, and on Windows through Visual Studio's Build
Tools or MinGW, so a port would not remove the C toolchain. It would also take
three to five months, bring in 120 to 200 crates against 8 Go modules today,
redraw every key image with different font rendering, and lose
`html/template`'s escaping by context. Go with `CGO_ENABLED=0` links with its
own linker and needs neither.

A spike (`spike/purego/main.go` in a throwaway worktree) built with
`CGO_ENABLED=0` on the arm64 Mac listed the Stream Deck XL through IOKit
(product 0x008F, path `DevSrvsID:4295141440`, the same form hidapi gives) and
read the frontmost application, the running applications and the clipboard
through AppKit. The binary linked only libSystem and libresolv.

## The setting

```yaml
schema_version: 1
platform_bridge: purego   # or cgo, the default while both exist
pages:
  ...
```

- `platform_bridge` is optional. Absent, it means `cgo`, so files written
  today keep today's behaviour and no migration is needed.
- It is read once, when the daemon starts. A change made while the daemon runs
  is logged as taking effect at the next start; restart with
  `launchctl kickstart -k gui/$(id -u)/<label>` or by quitting and opening.
- The daemon logs the bridge it started with, so a log shows which way each
  run went.
- Windows ignores it except for the HID driver (see phase 5).
- The editor does not show it; it is a YAML setting while both exist.

### Build constraints

Go sets the `cgo` build constraint itself when cgo is on. The cgo files gain
`//go:build darwin && cgo` and the hidapi driver `//go:build cgo`.

- A normal build (cgo on) holds both ways, and the setting picks.
- A `CGO_ENABLED=0` build holds purego only. There, `platform_bridge: cgo` is
  logged as unavailable in this build and purego is used. This build is the
  proof that no C compiler is needed, and is checked on every change.

## Why a toggle and not a release

Duplicating code to keep both ways was the condition for cutting a v0.1.0
release instead. It does not arise:

- Every adapter already sits behind a port in `internal/platform/ports.go`,
  and the decisions (which window, which process, which badge) already live in
  untagged Go files tested without a Mac. A purego adapter replaces only the
  thin calling layer, and the cgo version's body is Objective-C, so the
  duplicate code check (dupl, 100 tokens) has nothing to match.
- The HID transport's own logic (the process wide lock, the 0xE00002BC write
  retry, the unlocked read poll, the enumeration panic recovery, busy against
  disconnected) moves above a small driver interface, shared by both drivers
  rather than copied.

So no release is cut now. Before phase 7 removes the cgo code, tagging
v0.1.0 gives a point to go back to; that is decided then.

## Layout

| Package | Change |
| --- | --- |
| `internal/platform/objcbridge` (new, darwin) | The only package besides the HID driver that imports purego. Loads the frameworks once; `Class`, `Selector`, `Send` for objc_msgSend with typed variants for struct arguments (NSRect, NSSize); `CFString` to and from Go strings; `CFNumber`, `CFBoolean`, `CFArray` readers; `Symbol` for constants such as `kAXFocusedApplicationAttribute` read with Dlsym; `WithAutoreleasePool`; `Release` and `Retain`. |
| `internal/platform/systemadapter` | A `Bridge` type with `BridgeCgo` and `BridgePurego`. The eight macOS constructors take it: `NewApplicationIcons`, `NewApplicationActivator`, `NewApplicationBadges`, `NewWindowCycler`, `NewKeyPresser`, `AccessibilityTrusted`, `NewClipboardWriter`, `NewApplicationQuitter`. Each purego adapter is a file `*_purego_darwin.go` beside its cgo file. The Windows constructors take the bridge and ignore it. |
| `internal/deck/hidtransport` | `Watcher` keeps all its logic and calls a private `driver` interface: `enumerate(vendorID) ([]deck.Connection, error)`, `open(path) (handle, error)`, and on the handle `write`, `sendFeature`, `getFeature`, `readWithoutWaiting`, `close`. Drivers: `hidapiDriver` (cgo), `iokitDriver` (darwin, purego), later `winhidDriver` (windows). `NewWatcher(bridge)`. |
| `internal/config` | `PlatformBridge` on `Configuration`, decoded and encoded like `world_clocks`, refusing values other than `cgo` and `purego`. |
| `cmd/the-front-buttons` | Reads the bridge from the file at start and passes it to the constructors and `NewWatcher`. |
| `.golangci.yml` | depguard: purego only in `objcbridge` and `hidtransport`. `unsafe` and gosec G103 suppressions there only, as in the Windows files. |

## The macOS HID driver

The way hidapi's macOS backend does it, through IOKit:

- **Enumerate:** `IOHIDManagerCreate`, `IOHIDManagerSetDeviceMatching` with
  vendor 0x0FD9, `IOHIDManagerCopyDevices`; read vendor, product and the
  registry entry ID of `IOHIDDeviceGetService`; the path is
  `DevSrvsID:<id>`, as hidapi writes it, so a path from either driver opens
  with either.
- **Open:** `IORegistryEntryIDMatching`, `IOServiceGetMatchingService`,
  `IOHIDDeviceCreate`, `IOHIDDeviceOpen` without seizing, as hidapi does.
- **Input reports:** each open device gets a goroutine on a locked OS thread
  running a `CFRunLoop`, with the device scheduled on it and
  `IOHIDDeviceRegisterInputReportCallback` filling a bounded queue (oldest
  dropped past 30, as hidapi does). `readWithoutWaiting` takes from the queue,
  so the transport's 5ms poll stays. Polling `IOHIDDeviceGetReport` is not a
  substitute: it reads over the control pipe.
- **Removal:** `IOHIDDeviceRegisterRemovalCallback` marks the handle gone, so
  every later call fails as disconnected.
- **Writes and features:** `IOHIDDeviceSetReport` with the output or feature
  type and the report ID from the first byte; `IOHIDDeviceGetReport` for a
  feature report. An `IOReturn` other than success becomes an error carrying
  its code, so 0xE00002BC is retried as today.
- **Callbacks:** purego callbacks are never freed and are capped at 2000, so
  the input and removal callbacks are made once per process, and the context
  pointer finds the device in a map.
- **Close:** unschedule, stop the run loop, end the thread, `IOHIDDeviceClose`,
  release.

## The seven AppKit files

None uses blocks, callbacks, delegates or a run loop; every call is
synchronous, as now.

| File | Calls | Work |
| --- | --- | --- |
| clipboard | NSPasteboard generalPasteboard, clearContents, setString:forType: | small |
| quit | NSRunningApplication by bundle ID, terminate, localizedName, bundleURL; NSWorkspace runningApplications | small |
| keys | AXIsProcessTrusted(WithOptions) with a CFDictionary; CGEventCreateKeyboardEvent, CGEventSetFlags, CGEventPost | small; plain C functions |
| activation | NSRunningApplication activateWithOptions:; AXUIElementCreateApplication and SystemWide; AX set and copy of frontmost and focused application; AXUIElementGetPid | medium |
| badges | the Dock's AX children, AXStatusLabel and AXURL; CF type checks; NSBundle bundleIdentifier | medium |
| windows | NSWorkspace frontmost and running applications, activationPolicy; AX windows, subrole, title, minimised, main, raise; CFEqual | medium; keeps AX elements across calls |
| appicons | NSWorkspace icon lookup; NSBitmapImageRep; NSGraphicsContext; NSImage drawInRect:fromRect:operation:fraction: with NSRect structs; PNG representation | largest; struct arguments |

## Tests

Tests first, each proven against a planted fault, as everywhere else.

- **Bridge helpers:** round trips of strings, numbers and booleans; a pool
  that drains; a missing symbol reported, not a crash.
- **Agreement tests (darwin, both bridges in one binary):** each read only
  adapter run through cgo and purego must give the same answer: frontmost
  application, running applications, the Dock's badges, the window list,
  Accessibility trust, and the icon for a fixed application, compared pixel
  for pixel. These are the evaluation in test form. Adapters that act
  (activate, quit, post keys, write the clipboard, raise a window) are checked
  by acting and reading back through the other bridge where macOS allows.
- **HID:** the existing `RunDeckWatcherContract` runs against each driver with
  a deck attached, gated like the experiments by
  `THE_FRONT_BUTTONS_EXPERIMENT`; the experiments themselves take a driver, so
  their numbers can be compared between drivers on the same deck. A parity
  test enumerates through both drivers and wants the same connections and
  paths.
- **Config:** the setting decodes, encodes, defaults to cgo, and refuses any
  other value with the field named.
- **Build:** `CGO_ENABLED=0 go build ./...` and `go vet` for darwin/arm64 and
  windows/amd64 join the checks run before every commit.

Planted faults to prove the tests: a wrong selector, NSSize passed for
NSRect, output and feature report types swapped, the report ID byte dropped,
the input queue never filled, removal not marking the handle gone, the
transient write error losing its code, the setting ignored, and the
`CGO_ENABLED=0` build pulling in a cgo file.

## Phases

| Phase | Work | Days |
| --- | --- | --- |
| 0 | The work machine's fixes, from the handoff | first |
| 1 | `platform_bridge` setting, `Bridge` type, constructors taking it, build constraints, the `CGO_ENABLED=0` build check (purego adapters stubbed as unavailable) | 1 |
| 2 | `objcbridge` helpers and their tests | 1 |
| 3 | The seven adapters through purego, smallest first, each with its agreement test | 5 |
| 4 | HID driver interface under the watcher, then the IOKit driver, then running the contract and both experiments on the deck with each driver | 3 to 4 |
| 5 | Windows HID driver through hid.dll, setupapi and overlapped I/O, so a Windows build needs no C compiler either, and SETUP.md's MSYS2 gcc step goes; tested on the Dell | 2 to 3 |
| 6 | Evaluation on the physical deck: a week of daily use on purego; the log, the experiments and the agreement tests compared; a reinstall to confirm the Accessibility grant holds, as it is tied to the signing identity | in use |
| 7 | Removal: optional v0.1.0 tag first; delete the cgo files, the hidapi driver, go-hid and the setting (a schema migration drops `platform_bridge` from files that hold it); docs | 1 |

Total writing time is about 13 to 15 focused days. Phases 1 to 4 are usable
on the Mac on their own; phase 5 can wait for the Dell.

## Risks

- **No signature checks.** purego cannot see a function's C declaration, so a
  wrong argument type or struct layout crashes or corrupts memory at run time.
  Every binding is covered by a test that runs on this Mac.
- **Run loop threads.** Input reports depend on a CFRunLoop per device. Removal
  and wake from sleep are where hidapi's own bugs were; both are tried on the
  deck in phase 6.
- **Main thread.** AppKit calls, including icon drawing, already run off the
  main thread under cgo; purego does not change that, but it stays a latent
  risk.
- **Dependency.** purego is one module, Apache-2.0, with no dependencies of
  its own. go-hid and hidapi's bundled C go at phase 7, so the count of
  modules does not rise.
- **Outgoing connections.** None are added.
