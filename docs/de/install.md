# loomux installieren

[Übersicht](README.md) · [Ersteinrichtung](getting-started.md) →

loomux ist ein einziges Go-Binary ohne Python- oder Node-Laufzeit. Windows ist die getestete Plattform; Release-Binaries gibt es auch für Linux und macOS, aber mehrere Teile laufen nur unter Windows. Diese Seite behandelt, was du brauchst, wie du ein Release-Binary installierst und prüfst, wie es sich selbst aktualisiert, was unter Linux und macOS anders ist, wie du aus dem Quellcode baust und wie du die Version prüfst. Ist loomux installiert, führt dich die [Ersteinrichtung](getting-started.md) durch dein Projekt.

## Voraussetzungen

- **Windows (x64)** ist die getestete Plattform. Zum Rest siehe [Linux und macOS](#linux-und-macos).
- **[GitHub CLI](https://cli.github.com/)** (`gh`), angemeldet per `gh auth login`. Sie übernimmt den Download und später die Selbstaktualisierung.
- **Git** 2.31 oder neuer.
- **Go** 1.26 oder neuer, nur zum Bauen aus dem Quellcode (die Toolchain des Moduls ist go1.27.0).

## Schnellstart unter Windows

Führe das in PowerShell aus. Es lädt das neueste stabile Release mit `gh`, prüft es gegen `SHA256SUMS` und installiert es als `loomux.exe`:

```powershell
$bin = "$env:LOCALAPPDATA\loomux\bin"
New-Item -ItemType Directory -Force $bin | Out-Null
gh release download --repo xidus90/loomux --pattern 'loomux_*_windows_amd64.exe' --pattern SHA256SUMS --dir $bin --clobber
$exe = Get-ChildItem $bin -Filter 'loomux_*_windows_amd64.exe' | Select-Object -First 1
if (-not $exe) { throw 'download failed: see the gh error above' }
$want = ((Select-String -Path "$bin\SHA256SUMS" -Pattern $exe.Name -SimpleMatch).Line -split '\s+')[0]
if ((Get-FileHash $exe.FullName -Algorithm SHA256).Hash -eq $want) { Move-Item -Force $exe.FullName "$bin\loomux.exe"; Remove-Item "$bin\SHA256SUMS" } else { throw "checksum mismatch: $($exe.Name)" }
```

Ohne Tag lädt `gh release download` das neueste stabile Release. Der Vergleich mit `-eq` ignoriert die Groß- und Kleinschreibung, denn `Get-FileHash` liefert Großbuchstaben und `SHA256SUMS` Kleinbuchstaben.

Danach führst du das in einer beliebigen Sitzung in deinem eigenen Projekt aus (`cd <projekt>`):

```powershell
& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" init
& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" status
```

Liegt das Binary dort schon, lädt `init` kein zweites. Die vollen Pfade sind nötig, weil nichts den Ordner auf deinen `PATH` setzt. Willst du das, nimm ihn einmal in den Benutzer-`PATH` auf:

```powershell
[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ";$env:LOCALAPPDATA\loomux\bin", 'User')
```

Das wirkt erst in einer neuen Sitzung. Der Einzeiler schreibt den Benutzer-`Path` als einfachen String (REG_SZ) und löst dabei `%VAR%`-Einträge auf; wer solche Einträge behalten will, ergänzt den Ordner lieber über „Umgebungsvariablen bearbeiten“. Weiter geht es mit der [Ersteinrichtung](getting-started.md).

## Einen Download prüfen

Binaries gibt es auf der [Releases-Seite](https://github.com/xidus90/loomux/releases): `loomux_<version>_<os>_<arch>` für `windows/amd64` (`.exe`), `linux/amd64`, `linux/arm64`, `darwin/amd64` und `darwin/arm64`. Geprüft wird gegen `SHA256SUMS` aus demselben Release (der Schnellstart unter Windows tut das für dich). Unter Linux, macOS oder in Git Bash:

```sh
sha256sum --check --ignore-missing SHA256SUMS
```

Neben ihnen trägt jedes Release `NOTICE.md`, ebenfalls in `SHA256SUMS` geführt: die Lizenz jedes fremden Teils im Binary — die Standardbibliothek von Go, jedes gelinkte Modul, jede gelinkte Grammatik von tree-sitter und die deutsche Worthäufigkeitstabelle unter CC BY-SA 4.0 samt ihren Quellen.

## Das maschinenweite Binary

MCP-Brücke und `loomux serve` laufen aus einem Binary je Rechner, `%LOCALAPPDATA%\loomux\bin\loomux.exe`, das aus einem Release stammt und nie aus einem Checkout. Der Schnellstart oben installiert es; lass den MCP-Eintrag darauf zeigen:

```powershell
claude mcp add loomux -s user -- "$env:LOCALAPPDATA\loomux\bin\loomux.exe" mcp --channel local
```

`loomux init` nimmt die ersten Schritte ab: Sein Teil `binary` holt das neueste Release an denselben Ort, wenn dort keines liegt, und sein Teil `mcp-json` schreibt die `.mcp.json` des Projekts, wenn der Nutzerbereich keinen Server `loomux` kennt (siehe die [CLI-Referenz](cli-reference.md#11-projekt-einrichten-loomux-init)).

## Updates und Kanäle

Danach hält `serve` das Binary aktuell: eine Minute nach dem Start und danach täglich holt es über `gh` das höchste Release des eigenen Kanals, prüft es gegen `SHA256SUMS` und seine `--version` und tauscht die Datei. Die nächste Brücke ersetzt den laufenden Dienst. `loomux upgrade` tut dasselbe von Hand. Was der letzte Durchlauf fand, steht in `update.json` im Zustandsverzeichnis; der Sitzungsstart warnt, wenn `serve` woanders läuft oder der Durchlauf gescheitert ist. Vorerst nur unter Windows.

Eine Maschine folgt dem Beta-Kanal, solange die Datei `channel` in ihrem Zustandsverzeichnis `beta` enthält: Sie nimmt jedes neuere Release, Beta oder stabil. Ohne die Markierung nimmt sie nur stabile Releases. `loomux upgrade --beta` nimmt das neueste Release samt Betas und setzt die Markierung; `--stable` nimmt das neueste stabile Release, auch wenn es älter ist als die laufende Beta, und löscht sie; `--version <x.y.z>` installiert genau diesen Tag, auch eine Beta oder einen Rückschritt. Ein Rückschritt hält nicht: `serve` hebt ihn innerhalb von 24 Stunden wieder auf. Ein Binary der Versionszählung vor 1.0.0 nimmt ohne die Markierung die Releases dieser Zählung und die stabilen, nie eine neue Beta, und steht unter jedem späteren Release.

## Linux und macOS

Release-Binaries gibt es für `linux/amd64`, `linux/arm64`, `darwin/amd64` und `darwin/arm64`, und sie entstehen aus demselben Code. Getestet werden sie dort nicht; getestet wird unter Windows. Lade das Binary für dein System von der [Releases-Seite](https://github.com/xidus90/loomux/releases) und prüfe es wie oben beschrieben.

Nur unter Windows gibt es:

- das maschinenweite Binary und damit den Teil `binary` von `loomux init`;
- die Selbstaktualisierung durch `serve` und `loomux upgrade`;
- die Worktree-Spiegel (Junctions).

`loomux init` ist unter Linux und macOS nicht vorgesehen. Sein Teil `binary` scheitert, und `.mcp.json`, die Hook-Einträge und die Git-Hooks werden nicht geschrieben, sondern als fehlgeschlagen gemeldet, weil sie `${LOCALAPPDATA}/loomux/bin/loomux.exe` aufriefen. Wer es dort trotzdem nutzt, setzt das Modul `hooks` auf `none` und wählt im Modul `base` mit `each` die Teile `binary` und `mcp-json` ab. Das geht nur interaktiv: Für `base` gibt es kein Flag, also hilft auch `--yes` nicht.

Dort läuft die CLI selbst: `check`, `graph`, `brain` und `serve`.

## Aus dem Quellcode bauen

```sh
git clone https://github.com/xidus90/loomux.git
cd loomux
```

Unter Windows, für die Arbeit an loomux selbst:

```powershell
go run ./cmd/loomux init --yes   # baut bin/loomux.exe und schaltet die Git-Hooks scharf
```

Unter Linux und macOS:

```sh
go build -o loomux ./cmd/loomux
```

`init` ist unter Linux und macOS nicht vorgesehen (siehe [Linux und macOS](#linux-und-macos)). `go install github.com/xidus90/loomux/cmd/loomux@latest` geht nicht, weil `go.mod` eine `replace`-Direktive hat.

## Die Version prüfen

```sh
loomux --version
# Ausgabe: loomux 1.0.0 (ein stabiles Release), loomux 1.1.0-beta.1 (eine Beta)
# oder loomux 0.0.0-dev (aus dem Quellcode gebaut)
```

Liegt der Ordner nicht auf deinem `PATH`, rufe das Binary mit vollem Pfad auf: `& "$env:LOCALAPPDATA\loomux\bin\loomux.exe" --version`.

[Übersicht](README.md) · [Ersteinrichtung](getting-started.md) →
