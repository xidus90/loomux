# Benchmark & Lücken-Audit: loomux

- [← Zurück zur Gesamt-Matrix](../matrix.md)
- **Beispieldatei:** `cmd/loomux/main.go`

## 1. Performance & Latenzen

| Komponente | Kalt (1. Lauf) | Warmer Median | Warm Min | Warm Max | Status |
|---|---:|---:|---:|---:|---|
| **pre-tool-use** | 11.1 ms | 9.0 ms | 8.5 ms | 11.0 ms | [0] |
| **post-tool-use** | 557.6 ms | 509.4 ms | 507.1 ms | 530.0 ms | [0] |
| **graph build** | 259.8 ms | 252.7 ms | 250.1 ms | 293.4 ms | [0] |
| **Gesamt** | 828.4 ms | 770.6 ms | 768.2 ms | 832.4 ms | [0] |

- **Baseline Claude Hook:** 564.6 ms vs. loomux hooks 518.1 ms (Speedup: 1.1x, Status: [0])

## 2. Test- & Lücken-Audit (Gap Analysis)

| Werkzeug | Kategorie | Nativ im Projekt | Loomux-Lane | Im PATH | Status |
|---|---|---|---|---|---|
| `go-vet` | lint | `go.mod` | `go vet ./...` | Ja | ✅ Aktiv |
| `go-test` | test | `go.mod` | `go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile=coverage.out` | Ja | ✅ Aktiv |

- **Abdeckungsquote:** **100.0 %**

## 3. Erkannte Stacks & Lanes

- **Stacks:** `go, shell`
- **Lanes:** `shellcheck **/*.sh, go vet ./..., set -eu, cd "$(git rev-parse --show-toplevel)", if [ "$(git branch --show-current)" = master ]; then, echo "pre-commit: no commits on master; create a branch and open a pull request" >&2, exit 1, fi, unstaged=$(git diff --name-only -- '*.go' go.mod go.sum testdata .githooks ci), untracked=$(git ls-files --others --exclude-standard -- '*.go' go.mod go.sum testdata .githooks ci), if [ -n "$unstaged$untracked" ]; then, echo "pre-commit: inputs differ from the index; stage or stash them:" >&2, [ -z "$unstaged" ] || echo "$unstaged" >&2, [ -z "$untracked" ] || echo "$untracked" >&2, exit 1, fi, sh ci/gate.sh, set -eu, cd "$(git rev-parse --show-toplevel)", unformatted=$(gofmt -l cmd internal), if [ -n "$unformatted" ]; then, echo "gofmt: these files are not formatted:" >&2, echo "$unformatted" >&2, exit 1, fi, go vet ./..., go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile=coverage.out, go run ./cmd/loomux dev covergate --profile coverage.out, mkdir -p bin, go build -o bin/loomux.new.exe ./cmd/loomux, go run ./cmd/loomux dev swap-binary --dir bin`
