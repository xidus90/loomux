# Pilot-Binary außerhalb des Checkouts

Stand 2026-11-09. Branch `fix/dev-binary-outside-checkout`.

## Anlass

Ein Agenten-Sandbox-Werkzeug (vermutlich Codex, nicht belegt) hat auf
`C:\Users\micro\Documents\#GIT\loomux` drei explizite ACL-Einträge gesetzt:

- `Jeder:(CI)(DENY)(DC)`
- `S-1-4-575732716-727994981:(OI)(CI)(W,D,DC)`
- `Niedrige Verbindlichkeitsstufe:(OI)(CI)(NW)`

Das Low-Label erbt jede Datei im Checkout, auch `bin/loomux.exe`. Windows
startet ein Programm mit Low-Label als Low-Prozess. Ein Low-Prozess schreibt
nur in Ordner, die selbst Low sind. Folgen, am 2026-11-09 gemessen:

- Die Go-Lanes melden `GOTMPDIR … is not writable`.
- `loomux hook stop` kann `.git/loomux-stop-index-*` nicht anlegen.
- Git for Windows scheitert unter Low:
  `sh.exe … NtCreateDirectoryObject … 0xC0000022`.
- Junctions verhalten sich unter Low anders.

Dasselbe Binary, in einen Ordner außerhalb des Repos kopiert, läuft normal (Medium). Eine
Kopie erbt die Rechte ihres Zielordners. Eine verschobene Datei behält ihr
Label.

Der Mensch hat das Label von Hand entfernt. Das Werkzeug kann es wieder
setzen. Gewünscht ist ein Fix, der **keine Rechte setzen** muss.

## Ziel

Jeder Prozess, den ein Host-Hook des loomux-Checkouts startet, läuft aus
einem Binary außerhalb des Checkouts. Die Git-Hooks bleiben, wie sie sind
(siehe unten). Ein Label auf dem
Checkout erreicht ihn dann nicht. Läuft loomux trotzdem als Low-Prozess,
nennt es die Ursache.

## Nicht-Ziele

- Keine Änderung an den Rechten eines Ordners, weder von Hand noch durch
  loomux.
- Keine Änderung für Projekte, die nicht loomux selbst sind. Sie rufen schon
  heute das installierte `${LOCALAPPDATA}/loomux/bin/loomux.exe`
  (`hostfile.Canonical`).
- Kein Erkennen des Werkzeugs, das das Label setzt.

## Wer heute `bin/loomux.exe` startet

| Starter | Datei | Form |
|---|---|---|
| 8 Claude-Hooks (SessionStart ×2, PreToolUse, PostToolUse, Stop, SubagentStart, SubagentStop, Worktree-Unlink) | `.claude/settings.json`, geschrieben aus `hostfile.Checkout` | `"${CLAUDE_PROJECT_DIR}/bin/loomux.exe" …` |
| commit-msg | `.githooks/commit-msg` | `bin/loomux.exe check commit-msg`, sonst `go run` |
| pre-commit | `.githooks/pre-commit` | baut `bin/loomux.new.exe`, ruft `dev swap-binary --dir bin` |
| git-Hooks aus `init` | `setup.checkoutGitBinary` | `./bin/loomux.exe` |
| post-merge | `.githooks/post-merge` | schon das installierte Binary, nicht betroffen |

`hostfile.BinaryOf` erkennt einen loomux-Checkout an einem Eintrag, der
`${CLAUDE_PROJECT_DIR}/bin/loomux.exe` ruft, und gibt dann `Checkout`
zurück. Das ist der einzige Ort, an dem die Form entschieden wird.

## Entwurf

### Ort des Pilot-Binarys

```
${LOCALAPPDATA}/loomux/dev/<Ordnername des Checkouts>/loomux.exe
```

Der Ordnername ist der letzte Teil des Checkout-Pfads, beim Hauptcheckout
`loomux` und bei einem Worktree dessen Ordner (`compat-version-breadth`).
Das ist der Ordner statt Hash oder Branch im Dateinamen. So bleibt der Name
fest, und es sammeln sich keine alten Builds.

Nachtrag nach dem Review: Claude Code übergibt auch einer Sitzung in einem
Worktree den Hauptcheckout als `CLAUDE_PROJECT_DIR`
(`docs/wiki/topics/schreibschranke.md`). Ihre Hooks laufen deshalb aus
`dev/loomux`, so wie sie vorher aus dem `bin/` des Hauptcheckouts liefen.
Ein Binary je Branch gibt es nur für eine Sitzung, deren Projekt der
Worktree selbst ist.

`${LOCALAPPDATA}` ist schon heute die Wurzel von `hostfile.Canonical`. Die
Form ist also nicht weniger portabel als das, was jedes andere Projekt
bekommt.

**Grenze (bewusst):** Zwei Checkouts mit gleichem Ordnernamen teilen sich
einen Platz, etwa zwei Klone namens `loomux`. Der zuletzt getauschte
gewinnt. Die Doku nennt das.

### Hook-Zeile

`hostfile.Checkout` wird zu:

```
"${LOCALAPPDATA}/loomux/dev/$(basename "${CLAUDE_PROJECT_DIR}")/loomux.exe"
```

Gemessen am 2026-11-09: `basename` in Git Bash schneidet auch Pfade mit
Backslash richtig. `shellwords.Split` liefert als erstes Wort einen Pfad, der
auf `/loomux.exe` endet, `Owned` erkennt die Zeile also als eigene. Das
ist reines POSIX-sh: `basename` und `$( )`, keine Bashismen. Claude
Code führt Hooks unter Windows in Git Bash aus. `BinaryOf` erkennt die neue
Form und für den Übergang auch die alte. `init` schreibt eine alte Zeile
nicht um, denn Merge ändert grundsätzlich keine eigene Zeile. Es nennt sie
mit der neuen Form zum Ändern von Hand.

### `dev swap-binary`

Der Ablauf heute: `<dir>/loomux.new.exe` wird mit Rotation der alten
Generationen zu `<dir>/loomux.exe`.

Neu kommt danach ein zweiter Schritt hinzu. Die Datei wird **kopiert**,
nicht verschoben, als neue Datei (`os.Create` + `io.Copy`) nach
`${LOCALAPPDATA}/loomux/dev/<Ordnername>/loomux.new.exe`. Dort läuft
dieselbe Rotation. Den Ordnernamen nimmt der Befehl vom Wurzelverzeichnis
des Checkouts (`git rev-parse --show-toplevel` oder `--root`).

`bin/loomux.exe` bleibt bestehen, für Aufrufe von Hand
(`bin\loomux.exe check …`).

### Git-Hooks

Bleiben, wie sie sind (Entscheidung des Autors). `commit-msg` ruft
`bin/loomux.exe check commit-msg`, und das liest nur. Lesen darf auch ein
Low-Prozess. `pre-commit` startet `go`, das außerhalb des Checkouts liegt
und Medium läuft. Erst wenn ein Git-Hook unter Low scheitert, zieht er nach.

### Low-Erkennung

Unter Windows liest loomux die Integritätsstufe des eigenen Tokens
(`golang.org/x/sys/windows`, schon über `internal/child` im Modul), träge
und nicht beim Paket-Init. `TestStartDoesNoWorkInPackageInit` erlaubt 500
Allokationen.

Ist die Stufe Low, ersetzt die Meldung von `verify.writable` den Text
„grant it in the sandbox“ durch:

> loomux runs at Low integrity because its binary `<pfad>` carries a Low label (inherited from its folder); run a copy from outside that folder

Unter anderen Systemen gibt es diese Prüfung nicht. Die Meldung bleibt dort wie heute.

### Bootstrap

Die neue Hook-Zeile ruft ein Binary, das erst nach dem ersten
`swap-binary` existiert. `loomux init` im Checkout baut schon heute das
Binary (`binary-build`). Der Schritt endet künftig mit dem Tausch an beide
Orte. Fehlt das Binary trotzdem, scheitert der Hook laut, wie heute bei
fehlendem `bin/loomux.exe`.

## Erwogene Alternative: Weiterleitung über das installierte Binary

Die Hook-Zeile ruft `${LOCALAPPDATA}/loomux/bin/loomux.exe`, also das
installierte Binary, das immer Medium läuft. Es sucht den Dev-Build über
einen Hash des Wurzelpfads und startet ihn.

- **Vorteile:** keine Namenskollision, die Logik liegt in Go statt in sh und
  lässt sich testen.
- **Nachteile:** ein zusätzlicher Prozessstart je Hook, gegen den
  Hook-Bench zu messen. Ein Bootstrap-Problem, wenn das installierte Binary
  älter ist als die Weiterleitung. Außerdem prüft jeder Hook dann mit der
  Logik zweier Versionen.

**Entscheidung des Autors:** die Shell-Form. Sie fügt keinen Prozess hinzu und
hängt nicht von der installierten Version ab. Die Kollision ist eine
benannte, seltene Grenze. Der Mensch kann umentscheiden.

## Abnahme

- `TestTheRepositorysOwnSettingsNeedNoChange` ist grün mit der neuen Zeile
  in `.claude/settings.json`.
- Ein Test von `swap-binary` zeigt: Das Ziel außerhalb wird als neue Datei
  angelegt, nicht umbenannt. Die Rotation läuft dort wie in `bin/`.
- Ein Windows-Test von `writable` zeigt: Bei Low nennt die Meldung das
  Label, bei Medium bleibt sie wie heute. Die Stufe wird für den Test
  injiziert.
- Von Hand auf diesem Rechner gemessen: Mit wieder gesetztem Low-Label auf
  dem Checkout läuft der Stop-Hook grün.
- Doku: `cli-reference` (`dev swap-binary`, `init`), `configuration`
  (Go-Lanes), en und de, sowie die Wiki-Seite zum Pilot-Binary mit
  `log.md`-Eintrag.
