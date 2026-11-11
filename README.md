This is for the lions living in the wiry frames of my peripherals' bodies

# The Front Buttons

The Front Buttons is a Go daemon that drives an Elgato Stream Deck XL directly
over USB HID, with no Elgato software, on macOS (Apple silicon) and Windows
(amd64). Each key is an instance of a building block, called a brick, placed by
per machine configuration.

It is being built in phases. The current state of each phase is in
`IMPLEMENTATION_LOG.md`, and the evidence it is built on is in
`docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md`.

## Building

From this directory, with Go 1.27.3 and a C compiler (cgo is used for USB HID):

```bash
go build ./...
```

## Checking

```bash
go test ./...
```

```bash
go tool -modfile=tools/go.mod golangci-lint run ./...
```

```bash
scripts/check-duplicate-code.sh
```

The first command includes the repository checks (prose and typography, operating
system use, and this file's first line). The second runs the linters, including
the import layering rules. The third looks for duplicated code across every
package. Checking tools are pinned in their own module under `tools/` so they
never enter the daemon's dependencies.

## Where it came from

Written from the project specification and the author's engineering datum,
version 2. The Stream Deck protocol is read from the open source
python-elgato-streamdeck library; the commit read is recorded in the evaluation
document.
