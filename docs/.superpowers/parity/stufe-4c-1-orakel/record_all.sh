#!/usr/bin/env bash
# Zeichnet auf: alle Fälle unter testdata/cases/4c1-source (7).
# Aufruf:  RECORD_DIR=<dir> bash record_all.sh
#          danach: loomux dev import-cases --map testdata/cases/4c1-map.toml
#                  --from testdata/cases/4c1-source --to testdata/cases/4c1
# Gegen:   ultra-brain, Tag `loomux-3-source` (3cc72d2), über record.sh.
# Welten:  je Fall eine Kopie von 3a-worlds/vault-changed (der Welt von
#          reconcile/changed-source), in der Weltwurzel config.toml mit dem
#          Block [model] und, wo die Attrappe antwortet, ollama-fixture.json.
#          Der Bereich ist local_only, außer in open-area-not-asked.
# Die Attrappe lauscht fest auf 127.0.0.1:11435; die Fälle laufen darum
# nacheinander, und record.sh beendet sie vor dem nächsten.
set -uo pipefail
S="$(cd "$(dirname "$0")" && pwd)"
R() { bash "$S/record.sh" "$@"; }

R reconcile/proposal-kept proposal-kept "" "a local_only area's case gets the model's proposal, which quotes the added line of D1" "brain-mcp reconcile"
R reconcile/proposal-invented proposal-invented "" "the model quotes a line D1 does not hold: the proposal is dropped, the case is manual with the second note" "brain-mcp reconcile"
R reconcile/model-unreachable model-unreachable "" "nobody listens on the endpoint: the case is manual with the second note" "brain-mcp reconcile"
R reconcile/model-off model-off "" "no [model] block: the model is off, the case is manual with the first note" "brain-mcp reconcile"
R reconcile/open-area-not-asked open-area-not-asked "" "an area that is not local_only never asks the model" "brain-mcp reconcile"
R reconcile/endpoint-off-loopback endpoint-off-loopback "" "an endpoint off the loopback stops the run before a case is written" "brain-mcp reconcile"
R reconcile/broken-model-block broken-model-block "" "a [model] that is not a table stops the run before a case is written" "brain-mcp reconcile"
