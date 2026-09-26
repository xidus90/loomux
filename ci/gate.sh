#!/bin/sh
# The gate of loomux: `loomux check precommit` over the [verify] table in
# .loomux/config.toml -- gofmt and vet, the tests measuring coverage, and the
# per-function coverage gate. The pre-commit hook and every CI forge run
# exactly this script.
set -eu
cd "$(git rev-parse --show-toplevel)"
# CGo-free gate: loomux must build without a C toolchain.
cgo_gate="$(mktemp -d)"
CGO_ENABLED=0 go build -o "$cgo_gate/loomux" ./cmd/loomux
rm -rf "$cgo_gate"
go run ./cmd/loomux check precommit
