# Testing

Every suite, its command, and where it runs. Commands run from the repository
root with Go 1.27.3; nothing here needs a Stream Deck attached.

## Suites

| Suite | Command | Runs where |
|---|---|---|
| Unit tests, with the race detector | `go test -race ./...` | locally; CI on macos-26 and windows-2025 |
| Repository checks: prose and typography, operating system use, README first line; files named in `.gitignore` are skipped, and an ignore pattern the checks cannot read fails | included in the command above, as `internal/repositorychecks` | locally; CI on both runners |
| Linters: every golangci-lint linter, import layering (depguard), formatting (gofmt, gofumpt, goimports) | `go tool -modfile=tools/go.mod golangci-lint run ./...` | locally; CI on both runners |
| Build with vet | `go build ./...` then `go vet ./...` | locally; CI on both runners |
| Duplicate code across packages: standalone mibk/dupl, threshold 100 tokens | `scripts/check-duplicate-code.sh` | locally; CI on both runners, under Bash |

golangci-lint v2.15.0 and mibk/dupl v1.1.0 are pinned in `tools/go.mod`, a
module of its own, so checking tools never enter the daemon's dependencies.

## How the checks are trusted

Each check was seen to fail on a planted violation in a scratch copy of the tree
before it was allowed to pass on the real one (engineering datum 0.4). The record
of those runs is in `docs/evaluation/ACTUAL_CONTRACT_EVALUATION.md`, under TE1.
The repository checks also carry permanent unit tests that plant each violation
in memory, so a regression in a check fails the build.

## Testing deviations

None recorded.

## Author gates

- TE1: the author reproduces the automated criteria on both machines, and CI is
  green on both runners after the author pushes.
