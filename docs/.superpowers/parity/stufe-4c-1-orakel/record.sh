#!/usr/bin/env bash
# Zeichnet auf: testdata/cases/4c1-source/<verb>/<name>, ein Fall je Aufruf.
# Aufruf:  RECORD_DIR=<dir> bash record.sh <verb/name> <welt> <compare> <notiz> <cmd>
#          record_all.sh macht alle Aufrufe der Stufe.
# Gegen:   ultra-brain, Tag `loomux-3-source` (3cc72d2): brain-mcp.exe aus der
#          .venv des Checkouts in UB, der auf diesem Tag stehen muss.
# Quelle:  stufe-3a-orakel/record.sh mit drei Änderungen: WT ist dieser
#          Checkout (aus dem Ort des Skripts), die Welten kommen aus
#          testdata/cases/4c1-worlds, und um record-case läuft die
#          Ollama-Attrappe im Hintergrund, ihr Log außerhalb jeder Welt.
#          Die Abschottung von 3a bleibt: BRAIN_STATE_DIR ist die gestellte
#          Welt (das setzt der Recorder), qmd liest seine index.yml unter
#          {{WORLD}}/xdg, alles andere, was die Referenz anfassen könnte, zeigt
#          in eine Sandbox unter RECORD_DIR.
set -euo pipefail
# Das Arbeitsverzeichnis einer Aufzeichnung: ein leeres Verzeichnis mit
# bin/loomux.exe (aus diesem Baum gebaut) und fakeqmd/ (die qmd-Attrappe, aus
# internal/dev/fakeqmd/_qmd gebaut). sandbox/ entsteht darunter.
S="${RECORD_DIR:?set RECORD_DIR to the recording directory}"
WT="$(cd "$(dirname "$0")/../../../.." && pwd)"
UB="C:/Users/micro/Documents/#GIT/ultra-brain"
target="$1"; world="$2"; compare="$3"; notes="$4"; cmd="$5"
sandbox="$S/sandbox"
mkdir -p "$sandbox/localappdata" "$sandbox/loomux-state" "$sandbox/loomux-legacy" "$sandbox/xdg-state" "$sandbox/xdg-cache"
out="$WT/testdata/cases/4c1-source/$target"
rm -rf "$out"

# Die Attrappe liest ihre Fixture aus der Quellwelt. Die Fixture ändert sich
# nicht, darum ist es gleich, dass record-case der Referenz eine Kopie stellt.
# Das Log liegt außerhalb jeder Welt, sonst stünde es in world_after.
calls="$S/ollama-calls.log"
: > "$calls"
fixture="$WT/testdata/cases/4c1-worlds/$world/ollama-fixture.json"
fake=""
if [ -f "$fixture" ]; then
  "$S/bin/loomux.exe" dev fake-ollama --fixture "$fixture" --addr 127.0.0.1:11435 --log "$calls" &
  fake=$!
  trap '[ -n "$fake" ] && kill $fake 2>/dev/null' EXIT
  # Erst aufzeichnen, wenn die Attrappe lauscht; ein Verbindungsaufbau ohne
  # Anfrage schreibt keine Zeile ins Log. Lebt die Attrappe danach nicht mehr,
  # war der Port belegt, und ein anderer Prozess hätte geantwortet.
  for _ in $(seq 1 50); do
    if (exec 3<>/dev/tcp/127.0.0.1/11435) 2>/dev/null; then break; fi
    sleep 0.1
  done
  if ! kill -0 "$fake" 2>/dev/null; then
    echo "fake-ollama is not running: port 11435 taken?" >&2
    exit 2
  fi
fi

# Die Zahl der Aufrufe steht erst nach dem Lauf fest: record-case bekommt einen
# Platzhalter, den sed danach in notes.md ersetzt.
LOOMUX_STATE_DIR="$sandbox/loomux-state" LOOMUX_LEGACY_BRAIN_DIR="$sandbox/loomux-legacy" \
  "$S/bin/loomux.exe" dev record-case \
  --argv "\"$UB/.venv/Scripts/brain-mcp.exe\"" \
  --env "XDG_CONFIG_HOME={{WORLD}}/xdg" \
  --env "LOCALAPPDATA=$sandbox/localappdata" \
  --env "XDG_STATE_HOME=$sandbox/xdg-state" \
  --env "XDG_CACHE_HOME=$sandbox/xdg-cache" \
  --path-prepend "$S/fakeqmd" --env "LOOMUX_FAKE_QMD_FIXTURE={{WORLD}}/qmd-fixture.json" \
  --cmd "$cmd" --world "$WT/testdata/cases/4c1-worlds/$world" --out "$out" \
  --notes "loomux-3-source (3cc72d2), brain-mcp over fakeqmd and fake-ollama: $notes; ollama calls: @CALLS@" \
  --compare "$compare"

if [ -n "$fake" ]; then
  kill "$fake" 2>/dev/null || true
  wait "$fake" 2>/dev/null || true
  fake=""
fi
n=$(( $(wc -l < "$calls") ))
sed -i "s/@CALLS@/$n/" "$out/notes.md"
echo "exit=$(cat "$out/exit") calls=$n $target"
