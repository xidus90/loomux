# Worktree-Spiegelung gitignorierter Verzeichnisse

Stand 2026-09-07. Entwurf, freigegeben im Gespräch; Umsetzung folgt über
`writing-plans`.

## Das Problem

Ein `git worktree` bekommt vom Haupt-Checkout nur, was Git kennt. Alles
Gitignorierte fehlt — und in diesem Umfeld sind das genau die Verzeichnisse,
ohne die im Worktree nichts läuft:

- `space/.tools` — 4,2 GB Godot-Editor, JDK, Android-SDK, dotnet. Gitignoriert
  über `/.tools/` (`space/.gitignore:10`).
- `.ultraloom/vendor/` — die gepinnte Python-Laufzeit, die ultraloom dorthin
  klont. Gitignoriert über `.ultraloom/vendor/` (`space/.gitignore:68`).

Gemessen am 2026-09-07: `space/.worktrees/okf-bundle-root/.ultraloom/vendor/`
existiert nicht. In diesem Worktree läuft heute **kein** ultraloom-Hook, denn
jeder von ihnen ruft `uv run --project "${CLAUDE_PROJECT_DIR}/.ultraloom/vendor/ultraloom"`
und findet dort nichts. Zwei andere Worktrees derselben Ablage
(`.claude/worktrees/brainstorming`, `.claude/worktrees/huelle`) tragen eine von
Hand gelegte `.tools`-Junction — die Praxis existiert also schon, nur unzuverlässig.

Der Weg, den es *nicht* nehmen darf, ist ebenfalls belegt: `ln -s` legt unter
MSYS ohne `MSYS=winsymlinks` eine echte Kopie an. Das hat hier einmal 3,7 GB
gekostet.

## Was gebaut wird

Ein Kommando in `ulguard`, das gitignorierte Verzeichnisse eines Worktrees als
Windows-Junction auf den Haupt-Checkout zeigt, aufräumt, was es angelegt hat,
und verwaiste Junctions einsammelt.

### Warum `ulguard` und nicht `ulinit`

Semantisch spricht für `ulinit`, dass eine Laufzeit in Position zu bringen
Installationsarbeit ist (`internal/vendoring`). Operativ spricht dagegen, dass
`cmd/init/main.go:37` nur `args[0] == "check"` abzweigt und **alles andere in
den interaktiven Installer** fallen lässt (`internal/interview`, `x/term`). Ein
Binary, dessen Standardpfad ein Interview ist, darf nicht an einem globalen
`SessionStart` hängen: ein Tippfehler im Hook-Eintrag hängt dann jeden
Sitzungsstart auf, statt eine Fehlermeldung zu schreiben.

`ulguard` ist die Gegenseite: `cmd/guard/main.go:15,25` verteilt echte
Subkommandos (`status|explain|doctor`, `post-edit`), der `--root`-Vertrag ist
etabliert, und `cmd/guard/guard.go:99` liest bereits eine TOML-Datei unter
`.ultraloom/`. Die README nennt es „Policy Guard & Hook Dispatcher"; ein
Hook-Kommando gehört dorthin.

`ulinit` bekommt nur, was seiner Rolle entspricht: die Hook-Einträge in
`settings.json` schreiben.

### Konfiguration

Neuer Abschnitt in `.ultraloom/config.toml`:

```toml
[worktree]
mirror = [".tools", ".ultraloom/vendor"]
```

Die Datei ist getrackt (`git ls-files .ultraloom` in `space` listet sie), sie
steht also von sich aus in jedem Worktree. Eine zweite Datei — `.worktree-mirror`
oder ein Schlüssel in `.claude/settings.json` — wäre eine zweite Quelle für
dieselbe Auskunft und nützte außerhalb von Claude Code nichts.

`.ultraloom/vendor` ist Vorgabe und nicht Kür: ohne diesen Eintrag läuft in
einem frischen Worktree kein ultraloom-Hook, und kein Python-Hook könnte sich
diesen Zustand selbst herausbootstrappen.

Drei Fälle enden mit Exit 0 und ohne Ausgabe, und alle drei muss der globale
SessionStart treffen: `.ultraloom/config.toml` fehlt, sie ist da und hat keinen
`[worktree]`-Abschnitt, oder das Verzeichnis liegt in keinem Git-Repository.
Ein Hook, der in jedem fremden Projekt feuert, darf keinen davon als Fehler
melden — was ein Fehler ist, ist eine Junction, die angelegt werden sollte und
nicht angelegt werden konnte.

Der Python-Loader stört nicht: `load_config` liest ausschließlich `[verify]`,
`[agent]` und `[exec]` (`src/ultraloom/config.py:140-142`) und ignoriert eine
unbekannte Tabelle stillschweigend. `space` nimmt den Abschnitt also ohne
Bruch. Umgekehrt gilt: Go **liest** `config.toml` bislang nicht, es rendert sie
nur aus einer Vorlage (`internal/render/render.go:81`). Der Leser ist neuer
Code, kein Import.

### Die drei Kommandos

| Kommando | Aufrufer | Wirkung |
|---|---|---|
| `ulguard worktree-link --root <dir>` | globaler `SessionStart` | Legt fehlende Junctions an; sammelt verwaiste ein. |
| `ulguard worktree-unlink --root <dir>` | globaler `SessionEnd` | Löst die Junctions — nur wenn keine andere Sitzung mehr auf diesem Worktree steht. |
| `ulguard worktree-remove <dir>` | von Hand | Erst lösen, dann `git worktree remove`. |

### Worktree-Erkennung

Über `git worktree list --porcelain`, nicht über einen Vergleich von
`--git-dir` mit `--git-common-dir`. Letzteres verbietet `CLAUDE.md` mit
gemessener Begründung: aus `.claude/worktrees/opus-5-enforcement-57de82`
antwortet Git `C:/Users/micro/Documents/#GIT/ultraloom/.git` und `../../../.git`
— dasselbe Verzeichnis in zwei Schreibweisen, als Text verschieden.

Die Porcelain-Liste beantwortet beides in einem Aufruf: der erste
`worktree`-Eintrag ist der Haupt-Checkout und damit das Junction-Ziel, und ob
`git rev-parse --show-toplevel` weiter unten in der Liste auftaucht, ist die
Worktree-Antwort. Der Sweep braucht die Liste ohnehin.

### Sweep: Suchraum und Eigentum

Verwaiste Worktree-Verzeichnisse stehen per Definition *nicht* in
`git worktree list`, also muss der Sweep scannen. Suchraum sind die beiden
Konventionen, die in `space` beide vorkommen: `.claude/worktrees/*` und
`.worktrees/*` unter dem Haupt-Checkout, jeweils eine Ebene tief.

Als „unsere" Junction gilt **jeder Reparse-Point vom Typ Junction, der an einem
konfigurierten `mirror`-Pfad in einem solchen Verzeichnis liegt und auf den
Haupt-Checkout zeigt**. Kein Ledger: eine Liste angelegter Junctions wäre ein
weiterer Zustand, der von der Wirklichkeit abweichen kann, und nach einem
`Remove-Item -Recurse` auf einen Worktree wäre sie sofort falsch. Der Preis ist
bewusst: die von Hand gelegten Junctions in `brainstorming` und `huelle` werden
damit adoptiert. Sie sind nach Ziel und Ort von einer selbst angelegten nicht
unterscheidbar, und der Sweep greift ohnehin nur in Verzeichnissen, die Git als
Worktree nicht mehr kennt — also in Müll.

### SessionEnd und Fremdsitzungen

`CLAUDE.md` dokumentiert Sitzungen, die sich einen Checkout teilen. Löste
`SessionEnd` bedingungslos, verschwände `.tools` einer noch laufenden Sitzung
oder einem offenen Godot-Editor unter den Füßen. Also zählt das Kommando
zuerst: ultraloom führt Sitzungszustand pro Sitzung
(`.ultraloom/hooks/`, in `space` gitignoriert über `.ultraloom/hooks/`), und
gelöst wird nur, wenn dieser Worktree danach von keiner Sitzung mehr gehalten
wird.

### Löschwege, gemessen

Wegwerf-Fixtures, echte Junction, Git und PowerShell 7 dieser Maschine,
2026-09-07:

| Weg | Ziel im Haupt-Checkout | Rest im Worktree |
|---|---|---|
| `git worktree remove --force` | intakt | `.tools`-Junction bleibt liegen, Exit 0 |
| `Remove-Item -Recurse -Force` | intakt | alles weg |
| `bash rm -rf` auf dem vollen Worktree | intakt | alles weg |
| `bash rm -rf` auf dem Rest nach `git worktree remove` | intakt | alles weg |

Kein Weg löscht durch die Junction hindurch. Das Destroy löst also Müll, nicht
Gefahr — und `git worktree remove` ist der Weg, der Müll erzeugt und dabei
Erfolg meldet. Genau deshalb gibt es den Wrapper.

**Korrigiert am Ende von Task 6.** Die `bash rm -rf`-Zeile stand hier zuerst
als „`.tools`, `.git` und `.gitignore` bleiben". Das **reproduziert nicht**:
dreimal hintereinander mit frischen Fixtures nachgemessen räumt `rm -rf` in
beiden Lagen alles weg, das Ziel jedes Mal intakt. Warum die erste Messung
etwas anderes zeigte, ist **nicht** geklärt — die naheliegende Erklärung, ein
falsch geschriebener `/c`-Pfad, ist ausgeschlossen, denn in dieser Bash gibt es
`/c/Users` und `/C/Users` beide. Es steht hier also nur, was reproduziert, und
keine erfundene Ursache. Sicherheitsrelevant war die Zeile nie: die Aussage,
dass kein Löschweg durch die Junction hindurchgreift, hat in jeder Messung
gehalten, und die Begründung des Wrappers hängt an der Git-Zeile, die inzwischen
viermal bestätigt ist.

## Vorbedingung, beim Schreiben dieser Spec erledigt

Der erste Versuch, diese Datei aus einem Worktree zu committen, blieb am
eigenen Gate hängen — und der Grund gehört zur Sache: Git exportiert `GIT_DIR`
an jeden Hook, und die Variable **schlägt das Arbeitsverzeichnis**. Im
Haupt-Checkout ist der Wert das relative `.git` und löst in einem
Testrepository zufällig richtig auf; aus einem Worktree ist er absolut, und
`.githooks/pre-commit` reichte `go test` damit einen Zeiger auf das Repository,
das gerade committet wird. Drei Tests in `cmd/init` lasen dieses statt ihres
eigenen, `git rev-parse --absurd-flag` meldete Erfolg.

Nachgerechnet:

```
go test ./cmd/init/                                     → ok
GIT_DIR="…/ultraloom/.git" go test -count=1 ./cmd/init/ → dieselben 3 Fehler
```

Nicht bloß ein Testartefakt: `internal/detect/edges.go:23` und
`src/ultraloom/worktree.py` fragen Git mit einem `dir`- bzw. `cwd`-Argument,
und ein gesetztes `GIT_DIR` gewinnt dagegen. `.githooks/commit-msg` ruft
`ulinit check commit-msg` — dieser Weg lief also schon immer mit geliehener
Repository-Auskunft, und im Worktree mit der falschen.

Behoben in vier Stücken, jeweils mit vorher rotem Test:

- `internal/gitenv` (Go) und `src/ultraloom/gitenv.py` (Python) benennen die
  sieben Variablen, die auf ein Repository, einen Index oder einen Objektspeicher
  zeigen. Kein Präfixschnitt über `GIT_*`: `GIT_AUTHOR_NAME`, `GIT_EDITOR` und
  `GIT_TERMINAL_PROMPT` sind Einstellungen des Nutzers.
- `cmd/init/main.go` setzt `command.Env` für seinen einzigen Unterprozess.
- `process.child_env` nimmt sie jedem Kind, das ultraloom startet — damit auch
  `go test` und `pytest` unter dem Gate.
- `worktree.py` nimmt sie jedem eigenen Git-Aufruf.

Dazu die Testseite, die dieselbe Falle hatte: `run3` in `cmd/init` baute seine
Fixtures mit geliehener Umgebung, und ein neues `tests/conftest.py` räumt die
Variablen für die ganze Python-Sitzung weg. Ohne das fielen unter gesetztem
`GIT_DIR` 60 Tests in acht Dateien.

Das Python-Blattmodul liegt neben `process` und nicht darin, weil
`import ultraloom.cli` kein `ctypes` laden darf — `tests/test_cli_imports.py`
prüft genau das, und der erste Entwurf ist dort aufgelaufen.

Zweitens, kleiner: `/ulinit.exe` und `/ulguard.exe` sind gitignoriert
(`.gitignore:33,35`) und fehlen im Worktree, weshalb der Gate-Eintrag
`./ulinit check gofmt` dort stirbt. Ein `go build` im Worktree behebt es. Ein
Binary ist Bauergebnis pro Baum und **kein** Spiegelkandidat — der Plan sollte
es beim Anlegen eines Worktrees bauen, nicht auf den Haupt-Checkout zeigen.

## Offen für die Planungsphase

Go hat keinen Standardbibliotheksaufruf für eine Junction. Entweder
`DeviceIoControl(FSCTL_SET_REPARSE_POINT)` über `x/sys/windows` mit einem
Mount-Point-Reparse-Puffer, oder `cmd /c mklink /J` als Unterprozess.
`os.Symlink` ist **keine** Option: das ergibt einen Symlink und braucht je nach
Richtlinie Rechte. Die Wahl entscheidet, ob der Code außerhalb von Windows
testbar bleibt.

## Teststrecke

`space` — dort liegen die 4,2 GB, beide Worktree-Konventionen und ein Worktree
ohne `vendor/`. Ein eigens angelegter Worktree wird nach dem Test wieder
entfernt, über den neuen `worktree-remove`-Weg, der damit gleich seinen
Nachweis erbringt.
