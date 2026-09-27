# loomux hält sein maschinenweites Binary selbst aktuell

Stand 2026-09-23, **umgesetzt** (`internal/selfupdate`, `loomux self-update`). Entscheidung in der Fusion-Spec
(`2026-09-14-loomux-fusion-design.md`, Tabelle „Entscheidungen“ und Abschnitt
`loomux serve`).

## Anlass

Am 2026-09-23 liefen MCP-Brücke und `loomux serve` seit Tagen aus
`loomux-sdd-1b-2\bin\loomux.exe`, einem Build des Zweigs `sdd-1b-2` vom
2026-09-19. `project/loomux` war nie indiziert, der letzte Reconcile älter als
24 h. Neu bauen hätte nichts geholfen: Der User-MCP-Eintrag zeigte in einen
Entwicklungs-Checkout, den niemand mehr anfasste.

Der Mechanismus gegen alte Stände existiert schon (`internal/bridge/connect.go`,
`State.OlderThan`): Eine Brücke mit neuerem Binary ersetzt einen älteren
`serve`. Er griff nie, weil sich nur dieses eine alte Binary verband.

Von Hand behoben am selben Tag: Release v2.7.0 per `gh release download` nach
`%LOCALAPPDATA%\loomux\bin\loomux.exe`, Prüfsumme gegen `SHA256SUMS`, MCP-Eintrag
(`claude mcp … -s user`) dorthin umgehängt. Die erste Brücke aus dem neuen
Binary hat `serve` ersetzt, ohne dass ein Prozess beendet werden musste. Diese
Spec macht aus dem Handgriff den Normalfall.

## Ziel

- Das maschinenweite Binary liegt an **einem** festen Ort und kommt aus einem
  Release, nie aus einem Checkout.
- `serve` hält es aktuell, ohne dass jemand daran denkt.
- Liegt der Dienst doch woanders oder scheitert das Update, sagt es der
  Sitzungsstart.

Nicht Ziel: die `bin/loomux.exe` der Checkouts. Die baut weiter das
Pre-Commit-Gate, und `staleBinary` im Sitzungsstart wacht weiter über sie.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Kanonischer Ort | `<Zustandsverzeichnis>/bin/loomux.exe`, also `%LOCALAPPDATA%\loomux\bin\loomux.exe` (`config.StateDir()`) |
| Wer aktualisiert | `serve`, automatisch; `loomux self-update` stößt denselben Ablauf von Hand an |
| Quelle | GitHub-Release von `xidus90/loomux` über die `gh`-CLI. Das Repo ist privat; `gh` bringt die Anmeldung mit, loomux fasst keinen Token an |
| Kanal | Der des laufenden Binarys (`cli.Channel`). `beta` nimmt jedes Release, `stable` nur solche ohne Prerelease-Markierung. Gemessen am 2026-09-23: `release.sh` setzt ohne `RELEASE_CHANNEL` den Kanal `beta` und markiert dann jedes Release als Prerelease; v2.3.0 bis v2.7.0 sind alle Prereleases, und `gh release view` ohne Tag antwortet `release not found` |
| Prüfung | SHA-256 gegen das Asset `SHA256SUMS` des Releases, dann `--version` des neuen Binarys |
| Tausch | `swap.Swap` (heute `internal/dev/swap`, verschoben nach `internal/swap`) |
| Aktivierung | Keine eigene: Die nächste Brücke startet aus der neuen Datei, `OlderThan` ersetzt `serve` |
| Plattform | Windows. Auf POSIX ist der Update-Schritt aus und meldet das in `update.json`; `swap` kennt nur `loomux.exe`, und „Windows zuerst“ gilt laut Fusion-Spec |
| Nie überschrieben | Ein Binary außerhalb des kanonischen Orts, darunter jeder Build mit Version `0.0.0-dev` dort. Ein Entwicklungsbuild am kanonischen Ort wird dagegen ersetzt wie ein altes Release: Er kann nur von Hand dorthin gekommen sein, und nach der ursprünglichen Regel hätte ihn kein Lauf je wieder entfernt (Nachtrag 2026-09-25, beobachtet mit zwei von Hand getauschten Builds am selben Tag) |

## Aufbau

### Paket `internal/selfupdate` (neu)

Abhängigkeiten: `config`, `swap`. Importiert nie `serve`, `hooks` oder `cli`,
damit alle drei es benutzen dürfen. Die laufende Version (`cli.Version`) reicht
der Aufrufer in den Optionen herein: `cli` importiert `selfupdate` für den
Befehl, der umgekehrte Import wäre ein Zyklus.

- **`Canonical() string`** — `filepath.Join(config.StateDir(), "bin", "loomux.exe")`.
  Ob ein Binary dort liegt, entscheidet `os.SameFile` gegen `os.Executable()`,
  nicht ein Stringvergleich: Groß- und Kleinschreibung und Pfadform dürfen
  abweichen.
- **`Latest(ctx, channel) (Release, error)`** — `gh release list --repo
  xidus90/loomux --exclude-drafts --limit 30 --json tagName,isPrerelease`,
  gefiltert nach Kanal (siehe Entscheidungen), dann das höchste Tag nach
  semver, nicht das zuletzt veröffentlichte. **Nicht** `gh release view` ohne
  Tag: Das kennt nur das „Latest“-Release, und das gibt es bei lauter
  Prereleases nicht.
- **`Newer(tag, running string) bool`** — semver-Vergleich von `vX.Y.Z` gegen
  die laufende Version. Nicht parsebar heißt nicht neuer. Nie ein Downgrade.
- **`Fetch(ctx, rel, dir) (string, error)`** — `gh release download <tag>
  --pattern loomux_<ver>_windows_amd64.exe --pattern SHA256SUMS --dir <tmp>`,
  `<tmp>` ein frisches Verzeichnis unter `dir`. Prüft die Summe, prüft
  `<datei> --version` auf genau `loomux <ver>` am Zeilenanfang, setzt ihre
  Änderungszeit mit `os.Chtimes` auf jetzt, legt sie als `dir/loomux.new.exe`
  ab und räumt `<tmp>` weg. Die Änderungszeit ist die Aktivierung: `OlderThan`
  vergleicht nur sie. `gh` setzt sie heute auf die Downloadzeit (gemessen
  2026-09-23: 22:08:43, veröffentlicht war v2.7.0 um 09:26 UTC), aber darauf
  verlässt sich der Tausch nicht.
- **`Run(ctx, opts) Result`** — der ganze Ablauf unter `update.lock`:
  Ort und Version prüfen, `Latest`, `Newer`, `Fetch`, `swap.Swap(dir)`,
  `update.json` schreiben. `Newer` vergleicht mit der höheren von laufender
  Version und dem, was die Datei unter `Canonical()` auf `--version` meldet
  (bei Fehler oder unlesbarer Antwort nur die laufende): Ein `serve`, das
  keine Brücke ersetzt hat, läuft weiter als alte Version und lüde sonst jeden
  Tag dasselbe Release erneut herunter.
- **`ReadStatus() (Status, error)`** — liest `update.json` für den Hook.

`gh` läuft über eine Runner-Funktion im Options-Struct (Muster:
`benchProcessRunner` in `internal/cli/dev.go`). Jeder `gh`-Aufruf hat
2 Minuten Frist.

### `internal/swap` (verschoben)

`internal/dev/swap` zieht unverändert nach `internal/swap`, mit Tests;
`loomux dev swap-binary` importiert es von dort. Der Grund ist der Name: `dev`
ist Werkzeug für die Arbeit am Repo, `serve` soll davon nichts importieren.
Die Slots für gesperrte Altbinaries (`loomux.old.exe`, `loomux.old.<n>.exe`)
decken genau den Fall ab, dass ältere MCP-Brücken offener Sitzungen noch
laufen.

### `serve`

Eine eigene Schleife neben dem übrigen Dienst, **nicht** in der Upkeep. Die
Upkeep hält die erste Antwort an, bis der Reconcile durch ist; ein Update darf
keine Antwort anhalten. Beide laufen nebeneinander in `serve.Run`, und `Run`
wartet vor dem Freigeben der Sperre auf beide.

- Erster Lauf 1 Minute nach dem Start, danach alle 24 h, solange der Prozess
  lebt. Kein Zeitplan, nichts bleibt zurück, wenn `serve` endet.
- Läuft `serve` nicht aus `Canonical()` oder mit `0.0.0-dev`, schreibt jeder
  Durchlauf `update.json` mit `result = "skipped"` und dem Grund, und tut
  sonst nichts.
- Nach einem Tausch läuft `serve` weiter aus dem alten Image, bis die nächste
  Brücke es ersetzt.
- `gh` braucht `PATH` und seine Konfiguration (`APPDATA`, `USERPROFILE` oder
  `GH_CONFIG_DIR`). `serve` erbt die volle Umgebung der Brücke, die es startet
  (`childEnv` in `spawn.go` baut auf `os.Environ()` auf und ersetzt nur
  `LOOMUX_STATE_DIR` und `LOOMUX_BROKE_AWAY`). Das bleibt so; ein Test hält es
  fest.
- Unter Windows läuft `serve` ohne Konsole (`DETACHED_PROCESS`), deshalb
  startet jeder externe Aufruf des Durchlaufs (`gh`, `--version`) mit
  `CREATE_NO_WINDOW`; sonst öffnete Windows für jeden ein sichtbares
  Konsolenfenster.

### `loomux self-update`

Ruft `selfupdate.Run` und schreibt eine Zeile: `already current (v2.7.0)`,
`updated to v2.8.0; serve switches on the next bridge`, oder den Fehler. Exit 0
bei `current` und `updated`, 1 bei `failed`, 2 bei `skipped` (falscher Ort
oder Dev-Build). Kein Flag für einen anderen Ort: Wer ein Checkout-Binary
aktualisieren will, baut es. Ein ausgelassener Durchlauf von Hand schreibt
kein `update.json`: Er sagt nichts über das maschinenweite Binary und würde
den Eintrag von `serve` — gescheitert oder von woanders gelaufen — bis zu
dessen nächstem Durchlauf verdecken.

### Sitzungsstart

Neben `staleBinary` eine Funktion `updateWarnings`, die nur `update.json`
liest — kein Netz, keine Prozesse, kein Import von `serve`:

1. `source` ist `serve` und `executable` in `update.json` ist nicht
   `Canonical()` →
   `loomux serve runs from <pfad>, not from <kanonisch>; point the MCP entry at <kanonisch>`.
   Nur ein Durchlauf von `serve` sagt, woher der Dienst läuft; ein
   `loomux self-update` von Hand darf aus einem Checkout laufen. Und nur
   unter Windows: Anderswo lässt der Durchlauf alles aus, bevor er den Pfad
   ansieht, und `executable` ist dort nie `Canonical()` — die Warnung stünde
   in jeder Sitzung, ohne dass etwas falsch ist. `updateWarnings` bekommt
   dafür `goos` übergeben, damit ein Test beide Zweige auf jeder Plattform
   erreicht.
2. `result = "failed"` → `loomux self-update failed at <zeit>: <fehler>`.

Fehlt `update.json`, schweigt der Hook: Dann lief noch kein `serve` mit dieser
Fassung, und ein Rechner ohne Dienst ist kein Fehler.

Eine Warnung nach Alter („seit 48 h nicht geprüft“) gibt es bewusst nicht.
`serve` prüft erst eine Minute nach dem Start; nach zwei freien Tagen stünde
sie in der ersten Sitzung, ohne dass etwas kaputt ist.

Die Warnungen erreichen nur Sitzungen, deren Hooks loomux rufen: heute dieses
Repo, die übrigen Wirte nach der Umstellung in Stufe 4. Stufe 4 bringt auch
`loomux init` mit „dem Binary auf dem `PATH`“; `init` legt es an den
kanonischen Ort dieser Spec.

### `update.json`

Im Zustandsverzeichnis, atomar geschrieben wie `serve.json`
(`WriteState`: Tempdatei, dann `Rename`).

```json
{
  "source": "serve",
  "checked_at": "2026-09-24T08:00:00Z",
  "executable": "C:\\Users\\…\\AppData\\Local\\loomux\\bin\\loomux.exe",
  "running": "2.7.0",
  "result": "updated",
  "version": "2.8.0",
  "error": ""
}
```

`source` sagt, wer den Durchlauf gestartet hat: `serve` oder `cli`
(`loomux self-update`). `result` ist `current`, `updated`, `skipped` oder
`failed`.

## Fehlerverhalten

Grundsatz: Ein misslungenes Update lässt `loomux.exe` unangetastet und steht in
`update.json`. `serve` bricht nie ab.

| Fall | Verhalten |
|---|---|
| `gh` nicht auf dem PATH | `failed`, `gh not found; install GitHub CLI and run gh auth login` |
| `gh` nicht angemeldet oder ohne Zugriff aufs Repo | `failed`, die erste Zeile von `gh`s stderr |
| Kein Release im Kanal | `failed`, `no release in channel <kanal>`. Ein Repo mit Releases, das keins liefert, ist ein Fehler, kein Stand |
| Höchstes Tag nicht neuer | `current` |
| Ein Tag nicht semver | übergangen; bleibt keins übrig, wie „kein Release im Kanal“ |
| Asset `loomux_<ver>_windows_amd64.exe` fehlt | `failed`, mit dem erwarteten Namen |
| `SHA256SUMS` fehlt, hat keine Zeile fürs Asset, oder die Summe weicht ab | `failed`, Download verworfen |
| Neues Binary meldet bei `--version` nicht `loomux <ver>` | `failed`, verworfen |
| Alle 16 Slots für Altbinaries belegt | `failed` mit der Meldung von `swap` |
| Umbenennen schlägt fehl | `swap` hat das alte Binary noch an seinem Platz oder im Slot; `failed` nennt beide Pfade |
| `update.lock` gehalten | `self-update`: Exit 1, `update in progress`; `serve`: diesen Lauf auslassen, ohne `update.json` zu schreiben |
| `gh` hängt | Frist 2 Minuten, dann `failed`, `gh timed out` |
| `serve` hält während eines Durchlaufs an | `serve` bricht ihn ab und wartet, bis er zurückkehrt; der abgebrochene Durchlauf schreibt kein `update.json` |

## Externe Programme

`gh` kommt zur Liste in der Fusion-Spec („Externe Programme zur Laufzeit“),
optional: Ohne `gh` fällt nur das Update aus, und der Sitzungsstart sagt es.

## Tests

- `gh` über die Runner-Funktion; die Tests liefern JSON und Dateien aus
  `testdata`, kein Test braucht Netz oder `gh`.
- Jede Zeile der Fehlertabelle ein Test gegen ein Temp-Verzeichnis mit
  `LOOMUX_STATE_DIR`.
- Der `--version`-Aufruf des geladenen Binarys geht durch dieselbe
  Runner-Funktion wie `gh`; kein Test baut ein Binary.
- `Newer` tabellengetrieben: gleich, älter, neuer, Prerelease-Suffix, kaputt.
- `Latest` je Kanal: nur Prereleases (der heutige Stand) unter `beta` und
  unter `stable`, gemischt, höchstes Tag nicht das zuletzt veröffentlichte,
  leere Liste.
- Nach `Fetch` ist die Änderungszeit von `loomux.new.exe` jünger als die des
  laufenden Binarys, auch wenn die heruntergeladene Datei eine alte trug.
- `childEnv` reicht `PATH`, `APPDATA` und `GH_CONFIG_DIR` unverändert durch.
- `swap` behält seine Tests samt `swap_windows_test.go`.
- `updateWarnings` mit geschriebenen `update.json`: fehlt, fremder Ort,
  derselbe Ort in anderer Schreibweise, `failed`, alles gut.
- Die Schleife in `serve` mit injizierter Uhr und `after`-Funktion, ohne echte
  Wartezeit.
- 100 % pro Funktion; Ausnahmen nur für OS-Fehlerzweige mit
  `//coverage:exempt <Grund>`.

## Doku

- `README.md` / `README.de.md`: Installation aus dem Release an den
  kanonischen Ort, MCP-Eintrag darauf, `gh auth login` als Voraussetzung.
- `docs/{en,de}/cli-reference.md`: `loomux self-update`.
- Fusion-Spec: Zeile in „Entscheidungen“, Punkt in `loomux serve`, `gh` in
  „Externe Programme“, `selfupdate` und `swap` in „Pakete“, Abhängigkeitsregeln
  `hooks` → `selfupdate` und `serve` → `selfupdate`, `bin/`, `update.json` und
  `update.lock` in „Konfiguration und Zustand“.
- Migrationsplan (`docs/{en,de}/migration.md`): eine Zeile „Self-Update“ unter
  den Funktionen, Herkunft `new`, keine Stufe. Sie wird ✅, wenn der Code
  gemergt ist.
