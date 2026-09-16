# loomux-Schreibschranke: `sameRepository` ohne `git rev-parse`

**Datum:** 2026-09-16
**Stand:** entworfen, nicht umgesetzt
**Bezug:** [Worktree-Spec](2026-09-15-loomux-schranke-worktrees-design.md), deren Dateibefund
(`internal/brain/guard/worktree.go`) hier wiederverwendet wird.
**Paritätsliste:** [schranke-worktrees.md](../parity/schranke-worktrees.md), Zeile
„`sameRepository` über Dateibefund".
**Messgrundlage:** `docs/en/benchmarks.md`, Eintrag „The Write Barrier in a Linked Worktree"
(2026-09-15 15:39, Commit `d8bfad2`).

## Befund

Jeder Write in einem verknüpften Worktree, über dem ein eingechecktes `.loomux/config.toml` einen
registrierten Scope nennt, startet zwei `git rev-parse --git-common-dir`-Prozesse:

`Decide` → `declaredWikiRoot` (läuft für jedes Ziel) → `sameRepository` → `gitCommonDir` → `askGit`,
einmal für den Kandidaten, einmal für den registrierten Pfad.

Im Hauptcheckout nicht: dort ist das Manifestverzeichnis der registrierte Pfad, und `pathsEqual`
kehrt vorher zurück.

Gemessen (warm, Median, n = 20): Worktree 65,3 ms, Hauptcheckout 31,0 ms, Zielwert 72 ms, Luft
6,7 ms. Die Zuschreibung der Lücke von rund 34 ms an die beiden git-Aufrufe folgt aus dem Codeweg
und ist **nicht** getrennt gemessen. Die Messung dieser Stufe prüft sie mit.

## Ziel

`sameRepository` beantwortet dieselbe Frage ohne Kindprozess, aus den Dateien, die git selbst liest.
Ein Write im Worktree kostet danach ungefähr so viel wie im Hauptcheckout.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Erkennung | `repositoryCommon` aus `worktree.go`, kein `git`-Prozess und kein Rückfall darauf |
| Kandidatenseite | Streng: der Kandidat muss selbst Checkout-Wurzel sein (`.git`-Verzeichnis oder geprüfte `.git`-Datei) |
| Registrierte Seite | Steigend: `registered` selbst, dann `parents(registered)`, bis zum ersten `.git`-Eintrag gleich welcher Art; dessen `repositoryCommon` ist die Antwort. Nur, wenn `registered` existiert |
| Vergleich | `pathsEqual`, nicht `==` |
| Reihenfolge in `Decide` | Unverändert. Getauscht wird nur das Innere von `sameRepository` |
| Toter Code | `askGit`, `gitCommonDir`, `gitCommonDirTimeout` und der `gitenv`-Import in `path.go` entfallen |

### Verworfene Ansätze

- **Beide Seiten streng.** Einfacher, aber eine echte Verengung: registrierte Bereiche sind nicht
  immer Checkout-Wurzeln (die Tests registrieren etwa `<tmp>/vault/demo`), und ein solcher Bereich
  gälte zu keinem Worktree mehr als dasselbe Repository. Vom Nutzer verworfen.
- **Dateibefund zuerst, `git rev-parse` als Rückfall.** Kein Verhaltensverlust, aber zwei
  Wahrheiten im Code, und der langsame Pfad bliebe für jeden Fall, den die Dateien nicht erklären.
  Vom Nutzer verworfen.
- **`declaredWikiRoot` hinter die Worktree-Suche ziehen.** Nicht verhaltensneutral: ein kaputtes
  Manifest über dem Ziel verweigert heute auch innerhalb eines Workspace
  (`guard.go`, Schleife über `declaredWikiRoot`). Nicht Gegenstand dieser Stufe.

## Verhalten

```go
func sameRepository(candidate, registered string) bool {
	here, hereErr := resolvePath(candidate)
	there, thereErr := resolvePath(registered)
	if hereErr != nil || thereErr != nil {
		return false
	}
	if pathsEqual(here, there) {
		return true
	}
	common := repositoryCommon(candidate)
	if common == "" {
		return false
	}
	return pathsEqual(common, registeredCommon(registered))
}
```

Die Skizze legt die Form fest, nicht den Wortlaut. Eine leere Antwort von `registeredCommon`
braucht keinen eigenen Zweig: `components("")` ist leer, `pathsEqual(common, "")` für ein
nicht-leeres `common` also `false`. Ein Test hält das fest, weil genau diese Stelle der Grund war,
warum `gitCommonDir` die leere Antwort ausdrücklich abfing.

### Kandidatenseite

`repositoryCommon(candidate)` antwortet nur dort, wo der Kandidat ein eigenes `.git` trägt:

- `.git` ist ein Verzeichnis (`os.Stat`, folgt also einem Symlink auf ein Verzeichnis — wie git und
  Python auch): das aufgelöste Verzeichnis.
- `.git` ist eine reguläre Datei: `linkedCommon` mit seinen drei Wächtern (kein Symlink,
  Rückverweis `gitdir`, Verwaltungsverzeichnis direkt unter `<common>/worktrees`).
- Sonst `""`.

Damit ist die bisherige Forderung „der Kandidat trägt ein eigenes `.git`" in `repositoryCommon`
enthalten; die separate `os.Stat(filepath.Join(candidate, ".git"))` entfällt.
`TestASubdirectoryOfTheRegisteredRepositoryIsNoWorktreeOfIt` behält seinen Zahn: das
Unterverzeichnis trägt kein `.git`, `repositoryCommon` antwortet `""`.

### Registrierte Seite: `registeredCommon`

Neue Funktion in `worktree.go`:

1. Existiert `registered` nicht, `""`.
2. Für `registered` selbst und dann jedes Verzeichnis aus `parents(registered)`: trägt es einen
   `.git`-Eintrag (`os.Lstat`, gleich ob Verzeichnis, Datei oder Link), ist
   `repositoryCommon(verzeichnis)` die Antwort — auch wenn sie `""` ist.
3. Kein `.git` bis zur Wurzel ⇒ `""`.

`parents` beginnt bei `filepath.Dir(path)` und lässt den Pfad selbst aus, daher steht `registered`
vor der Schleife.

**Warum Schritt 1.** `git -C <pfad> rev-parse` scheitert an einem Pfad, den es nicht gibt, und
`gitCommonDir` antwortet dann `""`. Ein reines Steigen fände dagegen das Repository eines noch
vorhandenen Vorfahren und wäre an dieser Stelle weiter als heute. Ein `os.Stat` hält die Grenze.

**Warum steigend.** `git rev-parse --git-common-dir` antwortet aus jedem Unterverzeichnis eines
Checkouts. Die registrierte Seite war nie eine Wurzelprüfung, und sie wird keine.

**Warum am ersten `.git` halten, nicht am ersten erkannten Repository.** git hält bei seiner Suche
ebenfalls am nächsten `.git`. Liegt der registrierte Bereich in einem Submodul oder einem
`--separate-git-dir`-Checkout, versteht `repositoryCommon` dessen `.git`-Datei nicht und antwortet
`""`. Stiege die Schleife dann weiter, fände sie das Superprojekt, und dessen Worktrees gälten als
dasselbe Repository wie ein Bereich, für den git ein anderes gemeinsames Verzeichnis nennt: die
Änderung würde öffnen statt verengen. `os.Lstat` statt `os.Stat`, damit auch ein verwaister
`.git`-Link die Suche beendet, statt sie nach oben durchzulassen.

### Auflösung und Vergleich

`repositoryCommon` liefert über `resolvedOrEmpty` aufgelöste Pfade, dieselbe Form wie `resolvePath`
in `gitCommonDir`. Verglichen wird mit `pathsEqual` wie in `linkedWorktreeRoot`, damit beide
Aufrufer desselben Befunds gleich vergleichen. Eine Verhaltensänderung gegen das bisherige `==` ist
das nicht: `resolvePath` schließt die Schreibweise jeder existierenden Komponente, und ein `common`,
das `linkedCommon` durchlässt, existiert (sein `worktrees`-Verzeichnis enthält das gelesene
Verwaltungsverzeichnis).

### Was unverändert bleibt

- Der Hauptcheckout: `pathsEqual(here, there)` kehrt vor jedem Dateizugriff zurück.
- Die Reihenfolge in `Decide` und in `declaredWikiRoot`; ein kaputtes Manifest verweigert weiter.
- Die übrigen Aufrufer von `gitenv` (`wiki/gate.go`, `gitwork`, `hooks/worktree.go`,
  `worktree/topo`).
- Ein nicht auflösbarer Pfad, ein Kreis in den Zeigerdateien, eine fehlende oder gefälschte
  Zeigerdatei: jede dieser Antworten ist `""` und heißt „nicht dasselbe Repository".

## Abweichung von Python

Pythons `_same_repository` fragt `git rev-parse --git-common-dir` auf beiden Seiten. Die neue Form
antwortet anders, wo git etwas liest, das nicht in den Zeigerdateien steht:

- **`--separate-git-dir`-Checkout und Submodul** (`.git`-Datei ohne `gitdir`/`commondir` im Ziel):
  git findet das Repository, der Dateibefund nicht. Beim Submodul gilt das auch für einen
  registrierten Bereich darin: die Suche hält an dessen `.git` und steigt nicht ins Superprojekt.
- **`core.worktree`** und andere Konfiguration, die die Lage des Arbeitsbaums verschiebt.
- **`GIT_COMMON_DIR`, `GIT_DIR` und Verwandte:** `askGit` hat sie schon über `gitenv.Environ()`
  entfernt; der Dateibefund liest sie gar nicht. Keine Verhaltensänderung, aber jetzt ohne Kindprozess
  garantiert.

In jedem dieser Fälle antwortet die neue Form „nicht dasselbe Repository", und der Aufrufer liest das
als „dieses Manifest erklärt nichts". Die Abweichung verengt also nur; sie öffnet keinen Baum, den
Python verschlossen hielt. Der Fall „registrierter Pfad existiert nicht" ist über Schritt 1 in
Parität gehalten.

Die Abweichung braucht eine Freigabe in der Paritätsliste, bevor die Stufe als fertig gilt.

## Tests

TDD, jede Änderung erst rot. Jeder `askGit`-Stub kodiert eine Eigenschaft; portiert wird die
Eigenschaft, nicht der Stub.

| Heute | Neu |
|---|---|
| `TestTheRealGitIsAskedWithTheEnvironmentCleaned` | Behält `t.Setenv("GIT_DIR", foreign)`. Prüft das Ergebnis: echter `git worktree add` ⇒ `sameRepository` true; ein unabhängiges `git init` daneben ⇒ false. Wer wieder einen git-Aufruf ohne `gitenv` einbaut, fällt durch |
| `TestAGitThatCannotAnswerMakesNoTwoTreesOneRepository` | Stub entfällt; Fixture (`.git` mit `gitdir: elsewhere`) und `deny` bleiben, die Datei läuft durch `linkedCommon` |
| `TestABrokenPayloadCannotSlipThroughAPanic` | Panik über `readGitFile` statt `askGit`; Gegenstand bleibt der `recover` in `answer` |
| `TestGitAnsweringWithACircleIsNoCommonDirectory` | Ein `commondir`, das im Kreis führt, ergibt `""` |
| `TestGitCommonDirRefusesRatherThanGuesses`, `TestGitCommonDirJoinsARelativeAnswerAndKeepsAnAbsoluteOne` | Entfallen mit der Funktion |

Neu:

- `registeredCommon`: registrierte Wurzel; Unterverzeichnis einer Wurzel (steigt); nicht vorhandener
  Pfad unter einem Repository (`""`); kein Repository darüber (`""`); registrierter Pfad selbst
  verknüpfter Worktree; Bereich in einem Submodul eines Superprojekts (`""`, steigt nicht weiter).
- Zähltest über `readGitFile`: ein Write im Hauptcheckout liest keine Zeigerdatei.
- Ein Kandidat mit gültigem `.git`, dessen registrierte Seite in keinem Repository liegt, ist nicht
  dasselbe Repository (die leere Antwort von `registeredCommon` gegen ein nicht-leeres `common`).

Unverändert und grün zu halten: `TestASubdirectoryOfTheRegisteredRepositoryIsNoWorktreeOfIt`,
`TestAWorktreeOfTheRegisteredRepositoryDeclaresItsWiki` (Integrationsprobe mit echtem git) und
alle Tests in `worktree_test.go`.

Abdeckung 100 % je Funktion, ohne neuen `//coverage:exempt`.

## Messung

`loomux dev bench-hooks testdata/bench/barrier-worktrees.json -n 20`, Fallsdatei unverändert:

- `before.exe` aus `e4e0dc2`, `after.exe` aus dem Änderungscommit, beide nach
  `%TEMP%/loomux-barrier/`, gebaut mit derselben Go-Version. `before.exe` wird vor der ersten Codeänderung aus dem
  Arbeitsbaum gebaut, nachdem `git diff e4e0dc2 --stat -- cmd internal go.mod go.sum` leer
  geantwortet hat; kein temporärer Worktree, kein `git stash`.
- Der Worktree `loomux-sdd-1b1` steht auf `e4e0dc2`; die stdin-Dateien zeigen dorthin. Steht er bei
  der Messung woanders, wird das im Eintrag vermerkt.
- `LOOMUX_STATE_DIR` wie am 2026-09-15: eine Registry-Kopie, die nur den Hauptcheckout mit
  `workspace = true` führt.

Die Erwartung, die die Messung prüft und nicht bestätigen soll: der Worktree-Fall rückt an den
Hauptcheckout heran, der Hauptcheckout bleibt gleich. Bleibt die Lücke, war die Zuschreibung an die
git-Aufrufe falsch, und das steht so im Eintrag.

Einträge chronologisch in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`: Datum und Uhrzeit,
Gegenstand, Ausgangsbasis gegen Änderung, kalt und warm. Die älteren Absätze, die `git rev-parse`
als Ursache nennen, bekommen einen Verweis auf den neuen Eintrag und werden nicht umgeschrieben.

## Dokumentation

`README.md` und `README.de.md` nennen `git rev-parse` nicht. Die Zeile „Unified Pre-Tool Guard"
nennt ein Budget von <35 ms; ob der Worktree-Write es nach der Messung trifft, wird dort nachgetragen.

## Nicht Gegenstand

- Die Reihenfolge von `declaredWikiRoot` gegen die Worktree-Suche.
- Die Zeitzone auf dem Startpfad (eigener Eintrag in `benchmarks.md`).
- Ein Rückfall auf git für `--separate-git-dir` oder `core.worktree`.
