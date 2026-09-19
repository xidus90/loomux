#!/bin/sh
# The gate of loomux: format, vet, tests with per-function coverage. The
# pre-commit hook and every CI forge run exactly this script.
set -eu
cd "$(git rev-parse --show-toplevel)"
unformatted=$(gofmt -l cmd internal)
if [ -n "$unformatted" ]; then
	echo "gofmt: these files are not formatted:" >&2
	echo "$unformatted" >&2
	exit 1
fi
go vet ./...
# -count=1: a package served from the test cache adds no counts to the
# -coverpkg profile, and check gocover then reports its unchanged functions at 0%.
# The pattern names the module, not ./...: a directory pattern also takes in
# third_party/toml, which go.mod replaces the parser with, and that copy is
# not held to our coverage rule.
go test ./... -count=1 -covermode=set -coverpkg=github.com/xidus90/loomux/... -coverprofile=coverage.out
go run ./cmd/loomux check gocover --profile coverage.out
