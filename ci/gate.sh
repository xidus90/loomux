#!/bin/sh
# The gate of loomux: `loomux check precommit` over the [verify] table in
# .loomux/config.toml -- gofmt and vet, the tests measuring coverage, and the
# per-function coverage gate. The pre-commit hook and every CI forge run
# exactly this script.
set -eu
cd "$(git rev-parse --show-toplevel)"
go run ./cmd/loomux check precommit
