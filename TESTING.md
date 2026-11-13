# Testing

Every suite, its command, and where it runs. Commands run from the repository
root with Go 1.27.4; nothing here needs a Stream Deck attached.

## Suites

| Suite | Command | Runs where |
|---|---|---|
| Unit tests, with the race detector | `go test -race ./...` | locally; CI on macos-26 and windows-2025 |
| Repository checks: prose and typography (Go string literals read as their values), operating system use, README first line; files named in `.gitignore` are skipped, and an ignore pattern the checks cannot read fails | included in the command above, as `internal/repositorychecks` | locally; CI on both runners |
| Linters: every golangci-lint linter, import layering (depguard), formatting (gofmt, gofumpt, goimports) | `go tool -modfile=tools/go.mod golangci-lint run ./...` | locally; CI on both runners |
| Linter configuration verified against golangci-lint's schema, so a misspelt setting cannot silently switch a rule off | `go tool -modfile=tools/go.mod golangci-lint config verify` | locally; CI on both runners |
| Build with vet | `go build ./...` then `go vet ./...` | locally; CI on both runners |
| Fuzzing the configuration decoder: never panics, and anything accepted survives encode then decode unchanged | `go test -run '^$' -fuzz FuzzDecode -fuzztime 30s ./internal/config/` | locally; CI on both runners, 30 seconds per run |
| Fuzzing the deck's input report and feature text decoders: never panic, accept only exact lengths | `go test -run '^$' -fuzz '^FuzzDecodeKeyStates$' -fuzztime 30s ./internal/deck/`, and the same for `^FuzzDecodeSerialNumber$` | locally; CI on both runners, 30 seconds each |
| README brick reference matches the registry and preset table; regenerate with `UPDATE_README=1 go test ./internal/reference/` | included in `go test ./...`, as `internal/reference` | locally; CI on both runners |
| Start at login: the macOS install script refuses a missing binary and the printed agent passes `plutil -lint`; the Windows install script exits 2 on a missing binary and the printed task parses as XML | the two steps in `.github/workflows/ci.yml`; locally on macOS, `go run ./cmd/the-front-buttons -print-launch-agent \| plutil -lint -` | CI on both runners; the Windows script only in CI |
| The editor's script (`static/editor.js`) matches its TypeScript in `web/`: rebuilt with the pinned esbuild and compared, line endings ignored | `scripts/check-editor-script.sh`; regenerate with `go generate ./internal/editor/` | locally; CI on both runners, under Bash |
| Duplicate code across packages: standalone mibk/dupl, threshold 100 tokens | `scripts/check-duplicate-code.sh` | locally; CI on both runners, under Bash |

golangci-lint v2.15.0, mibk/dupl v1.1.0 and esbuild v0.28.2 are pinned in `tools/go.mod`, a
module of its own, so checking tools never enter the daemon's dependencies.

## How the checks are trusted

Each check was seen to fail on a planted violation in a scratch copy of the tree
before it was allowed to pass on the real one (engineering datum 0.4). The record
of those runs is in `docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md`, under TE1.
The repository checks also carry permanent unit tests that plant each violation
in memory, so a regression in a check fails the build.

Code built only for Windows (`internal/platform/systemadapter/system_windows.go`)
is invisible to the linters run on a Mac; CI lints it on windows-2025. To lint
it from the Mac first, build the linter for the Mac, then point it at Windows,
leaving out the two cgo packages, which have no Windows only code:

```bash
go build -modfile=tools/go.mod -o bin/golangci-lint github.com/golangci/golangci-lint/v2/cmd/golangci-lint
```

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 bin/golangci-lint run $(go list ./... | grep -v -e hidtransport -e cmd/the-front-buttons | sed 's#github.com/TheScottBot/the-front-buttons#.#')
```

## Testing deviations

- The `DeckDevice` contract against the real adapter runs only with a deck
  attached and `THE_FRONT_BUTTONS_DECK_TESTS=1`; elsewhere, including CI, it is
  skipped with that reason. The watcher half of the adapter's contract runs
  everywhere. Run it on the deck with:

  ```bash
  THE_FRONT_BUTTONS_DECK_TESTS=1 go test -count=1 -run Contract ./internal/deck/hidtransport/
  ```

- The `CredentialStore` contract against the real Keychain or Credential
  Manager runs only with `THE_FRONT_BUTTONS_CREDENTIAL_TESTS=1`, because it writes
  one entry, `contract-test-entry`, and removes it afterwards. Run it with:

  ```bash
  THE_FRONT_BUTTONS_CREDENTIAL_TESTS=1 go test -count=1 -run Credential ./internal/platform/systemadapter/
  ```

- The deck experiments run only with `THE_FRONT_BUTTONS_EXPERIMENT=1`, a deck
  attached and a person at it, and with the daemon stopped, since they open the
  deck themselves. Experiment 1 paints every key with its index (each row its
  own colour) and logs every raw input report for two minutes, or for the
  duration in `THE_FRONT_BUTTONS_EXPERIMENT_FOR`, while the person presses keys
  and checks the order and orientation by eye. The write experiment times full
  pages with and without a reader running and fails on any write that fails
  after the transport's retries. Compare the results with the MacBook's in the
  evaluation, under deck testing. Run them with:

  ```bash
  THE_FRONT_BUTTONS_EXPERIMENT=1 go test -count=1 -v -run Experiment ./internal/deck/hidtransport/
  ```

- Bringing an application to the front on macOS is not unit tested, since it
  would take over the person's screen; the wait for it to come forward, and the
  daemon's use of it, are tested with fakes, and it was verified on the deck.

## Author gates

- TE1: the author reproduces the automated criteria on both machines, and CI is
  green on both runners after the author pushes.
- TE4: legibility on the physical deck at arm's length, both machines.
- TE6: the device contract above on both machines; pressing keys, paging both
  ways, and ten dock switches without restarting the daemon.
- TE7: every preset and both bricks on each machine, including new Teams by its
  app ID on the Dell (DV3).
- TE8: a real workflow turning red then green on the deck at work, with a press
  opening the failing run; the credential contract above on both machines.
- TE9: building a full page from scratch in the editor on each machine, once
  using the keyboard only.
- TE10: after installing with the script and logging out and in, the daemon is
  running and the deck is live on each machine without a terminal open.
