# loomux-Schreibschranke: verknüpfte Worktrees eines Workspace

**Datum:** 2026-09-15
**Stand:** umgesetzt, gemessen und freigegeben 2026-09-15 (Plan `2026-09-15-loomux-schranke-worktrees.md`)
**Bezug:** [Fusions-Spec](2026-09-14-loomux-fusion-design.md), Schreibschranke aus Stufe 1a
(`internal/brain/guard`). Eigene kleine Stufe, eingeschoben vor
[Stufe 1b-1](2026-09-15-loomux-stufe-1b-1-design.md).
**Messgrundlage:** Messungen und Dateibefunde vom 2026-09-15, unten mit Zahlen.

## Ziel

Ein verknüpfter Git-Worktree eines Repos, das die Registry als `workspace = true` führt, ist ohne
eigenen `[[area]]`-Eintrag beschreibbar. Heute verweigert die Schranke dort jeden Write
(`lies outside every writable tree`), und jeder Worktree braucht einen von Hand gepflegten Block in
`%LOCALAPPDATA%\loomux\registry.toml`.

## Ausgangslage im Code

- `writableRoots` (`internal/brain/guard/guard.go`) nimmt bei `workspace = true` genau den
  eingetragenen `path` als Wurzel, sonst nichts.
- `sameRepository` (`internal/brain/guard/path.go`) erkennt Worktrees bereits, wird aber nur von
  `declaredWikiRoot` für Wiki-Manifeste gerufen, nicht für Workspaces.
- `loomux worktree link|unlink|remove` fasst die Registry nicht an.
- Das Verhalten ist Parität zu Pythons `_writable_roots`; die Änderung ist eine Abweichung.

## Entscheidungen

| Frage | Entscheidung |
|---|---|
| Umfang | Nur die Worktree-Lücke. Allow-List, Read-only-Zonen, Manifest-Schutz, Memory und Scratchpad bleiben unverändert |
| Welche Bereiche | Nur `workspace = true`. Das Feld `wiki` wird nicht auf Worktrees übertragen; Wiki-Bundles im Worktree deckt `declaredWikiRoot` über ihr Manifest schon ab. `readonly` spielt für die Workspace-Hälfte keine Rolle, wie heute in `writableRoots` |
| Erkennung | Git-Verwaltungsdateien direkt lesen, kein `git`-Prozess (Ansatz A) |
| Zeitpunkt und Ort | Vor 1b-1, umgesetzt im Worktree `loomux-sdd-1b1` auf eigenem Zweig |

### Verworfene Ansätze

Gemessen am 2026-09-15 im Worktree `loomux-sdd-1b1`, je 10 warme Läufe, Git 2.54.0:

| Aufruf | Median |
|---|---|
| `git rev-parse --git-common-dir` | 42 ms |
| `git worktree list --porcelain` | 40 ms |
| `loomux hook pre-tool-use` heute (`docs/en/benchmarks.md`) | 24,5 ms warm, Zielwert 72 ms |

- **B — `sameRepository` wiederverwenden:** zwei `git rev-parse` je Write im Worktree, rund +84 ms,
  also etwa 108 ms. Über dem Zielwert.
- **C — `git worktree list --porcelain` je Workspace-Bereich:** rund +40 ms je Bereich, heute etwa
  65 ms. Knapp unter dem Zielwert und wächst mit jedem Bereich.
- **Allow-List ganz entfernen** oder **Schranke aus dem Hook nehmen:** vom Nutzer verworfen; beides
  öffnet fremde Repos, Nutzerdateien und Vault-Quellen.

## Verhalten

Liegt ein aufgelöstes Write-Ziel unter keiner Wurzel und nicht in Memory oder Scratchpad, sucht die
Schranke den nächsten verknüpften Worktree darüber. Gehört er zum selben Repository wie ein
`workspace`-Bereich, wird sein Wurzelverzeichnis eine weitere beschreibbare Wurzel.

### Einbau in `Decide`

- Neue Funktion `linkedWorktreeRoot(target string, areas []area) string`.
- Sie läuft dort, wo `Decide` heute die Ziele `outside` sammelt: **nach** dessen früher Rückkehr
  bei leerem `outside` und **vor** `reviewCentre`, einmal für alle noch offenen Ziele. `outside`
  enthält nur Ziele unter keiner Wurzel und nicht in Memory oder Scratchpad; ein Write im
  Hauptcheckout kehrt vorher zurück und liest keine Datei mehr als heute.
- Ein Fund kommt in `roots`, die von ihm gedeckten Ziele fallen aus `outside`. Die Ablehnungsmeldung
  eines gemischten Aufrufs listet ihn dadurch unter „writing is allowed only below“ mit.
- Die Reihenfolge von `Decide` bleibt: Manifest, Memory, Registry, Read-only-Zonen, Allow-List. Eine
  Read-only-Zone schlägt eine Worktree-Wurzel, weil die Zonenprüfung vorher und gegen die Ziele
  läuft. Die Prüfung „keine Wurzel überhaupt“ davor ändert sich nicht: Eine Worktree-Wurzel setzt
  einen `workspace`-Bereich voraus, und der ist schon selbst eine Wurzel.

### Erkennung

`linkedWorktreeRoot` geht `parents(target)` durch, nächstes Verzeichnis zuerst, und gibt das erste
Verzeichnis `dir` zurück, für das alle fünf Bedingungen gelten:

1. `dir/.git` ist eine **reguläre Datei** (per `Lstat`, kein Symlink: ein Symlink auf die `.git`
   eines echten Worktrees würde dessen Rückverweis borgen), ihr Inhalt beginnt mit `gitdir: `. Der
   Rest, um nachgestellten Leerraum gekürzt, ist das Verwaltungsverzeichnis `admin`; ein relativer
   Pfad gilt ab `dir`.
2. `admin/commondir` ist eine reguläre Datei. Ihr Inhalt, gekürzt und relativ ab `admin`, aufgelöst
   mit `resolvePath`, ist das gemeinsame Git-Verzeichnis `common`. Submodule und
   `--separate-git-dir` haben keine `commondir` und scheiden hier aus.
3. **Rückverweis:** `admin/gitdir` ist eine reguläre Datei. Ihr Inhalt, gekürzt und relativ ab
   `admin`, aufgelöst mit `resolvePath`, ist nach `pathsEqual` gleich `resolvePath(dir/.git)`.
4. `common` ist nach `pathsEqual` gleich dem gemeinsamen Verzeichnis eines `workspace`-Bereichs.
5. **Lage:** `admin` liegt direkt unter `<common>/worktrees`, d. h. `filepath.Dir(admin)` ist nach
   `pathsEqual` gleich `common/worktrees`. Git legt es immer dort an. Ohne diese Bedingung könnte ein
   Verwaltungsverzeichnis an anderer Stelle — etwa das eines verschachtelten fremden Repos im
   Workspace, dessen `commondir` ein Schreibwerkzeug umschreiben darf — auf das Repo des Workspace
   zeigen und einen Baum außerhalb öffnen (Befund des Abschluss-Reviews, per Probe bestätigt).

Das gemeinsame Verzeichnis eines Bereichs mit Pfad `p`:

- `p/.git` ist ein Verzeichnis: `resolvePath(p/.git)`.
- `p/.git` ist eine Datei: Bedingungen 1–3 und 5 mit `dir = p`, das Ergebnis ist `common`.
- sonst: keins, der Bereich nimmt am Vergleich nicht teil.

Die gemeinsamen Verzeichnisse der Bereiche werden beim ersten Bedarf berechnet, höchstens einmal je
Aufruf von `Decide`.

Passt ein Verzeichnis nicht, klettert die Suche weiter, wie `declaredWikiRoot`. Ein verschachteltes
fremdes Repo im Worktree beendet die Suche nicht; das gleicht dem Hauptcheckout, wo derselbe Pfad
ebenfalls unter der Workspace-Wurzel liegt.

Beleg des Layouts auf dieser Maschine (`git worktree add`, Git 2.54.0):
`loomux-sdd-1b1/.git` = `gitdir: C:/Users/micro/Documents/#GIT/loomux/.git/worktrees/loomux-sdd-1b1`;
dort `commondir` = `../..`, `gitdir` = `C:/Users/micro/Documents/#GIT/loomux-sdd-1b1/.git`. Mit
`worktree.useRelativePaths` schreibt Git beide Verweise relativ; die Erkennung liest beide Formen.

### Fehlerverhalten

Jede Lese- oder Auflösungsstörung heißt „kein Worktree“. Das Ziel bleibt außerhalb und bekommt die
bestehende Ablehnung; es gibt keine neue Meldung. Grund wie bei `gitCommonDir`: Eine Schranke, die
bei Unklarheit öffnet, ist keine.

### Folgen

- **Verschobener Worktree** ohne `git worktree repair`: Der Rückverweis stimmt nicht, der Baum bleibt
  gesperrt. Gewollt; `git worktree repair` behebt es.
- **Registrierter Pfad ist selbst ein verknüpfter Worktree:** Seine Geschwister-Worktrees öffnen
  sich, der Hauptcheckout nicht — er hat keine `.git`-Datei, sondern ein Verzeichnis. Wer den
  Hauptcheckout will, registriert ihn.
- **Kein `git`-Prozess:** Die geerbte `GIT_DIR`-Umgebung, gegen die `askGit` sich schützt, spielt
  hier keine Rolle.
- **Verwaltungsverzeichnis ohne Worktree** (gelöscht, nicht gepruned): Das Verzeichnis existiert
  nicht, es gibt nichts zu beschreiben.
- **Restrisiko:** Wer außerhalb aller Wurzeln eine `.git`-Datei anlegt, die auf ein selbst
  angelegtes Verwaltungsverzeichnis unter `.git/worktrees/` des Workspace zeigt, öffnet sich jenes
  Verzeichnis. Die Verwaltungsdateien darf ein Schreibwerkzeug im Workspace anlegen, die
  `.git`-Datei außerhalb aber verweigert die Schranke jedem Schreibwerkzeug; sie gelingt nur über
  ein Werkzeug, das sie nicht prüft (etwa Bash). Das ist dieselbe Klasse wie heute.

## Nicht im Umfang

- Scope und Zustandsverzeichnis einer Sitzung im Worktree.
- `loomux worktree link|unlink|remove` und die Ausgabe von `loomux status`.
- Die Übertragung des Registry-Felds `wiki` auf Worktrees.

## Tests

TDD, 100 % je Funktion.

**Einheit, ohne Git** — Verwaltungsdateien von Hand in `t.TempDir()`, je Zweig ein Test:

- `gitdir` absolut; `gitdir` relativ; `commondir` und `gitdir` im Verwaltungsverzeichnis relativ
- `commondir` fehlt (Submodul-Form)
- Rückverweis zeigt woanders hin (geliehene `.git`-Datei)
- `.git` ist ein Verzeichnis, kein verknüpfter Worktree
- Präfix `gitdir: ` fehlt
- `.git`-Datei, `commondir` oder `gitdir` unlesbar oder nicht auflösbar
- Bereich ohne `workspace`; Bereich ohne `.git`
- registrierter Pfad ist selbst ein verknüpfter Worktree
- verschachteltes fremdes Repo im Worktree: die Suche klettert weiter
- Worktree eines nicht registrierten Repos bleibt gesperrt
- Read-only-Zone im Worktree schlägt die Worktree-Wurzel
- Write im Hauptcheckout: `linkedWorktreeRoot` wird nicht gerufen
- Verwaltungsverzeichnis eines verschachtelten Repos im Workspace, `commondir` auf den Workspace
  umgeschrieben: bleibt gesperrt
- selbst angelegtes Verwaltungsverzeichnis (`gitdir: .`) außerhalb: bleibt gesperrt
- `.git` als Symlink auf die `.git`-Datei eines echten Worktrees: bleibt gesperrt (übersprungen
  ohne Symlink-Recht)

**Integration mit echtem Git:** Temp-Repo, `git worktree add`, Registry kennt nur das Haupt-Repo als
`workspace`. `Decide` erlaubt einen Write im Worktree und verweigert einen Write in einem fremden
Repo daneben mit der bestehenden Meldung.

## Doku

- **Paritätsliste:** neue Datei `docs/.superpowers/parity/schranke-worktrees.md`, eine Zeile: Alt
  „brain guard: verweigert im verknüpften Worktree ohne eigenen Eintrag“, Neu „erlaubt, wenn der
  Worktree zu einem `workspace`-Bereich gehört“. Braucht die Freigabe des Nutzers.
- **Benchmarks** in `docs/en/benchmarks.md` und `docs/de/benchmarks.md`: `hook pre-tool-use`,
  Write im Worktree ohne Registry-Eintrag vor der Änderung (Ablehnung) gegen danach (Erlauben), dazu
  ein Write im Hauptcheckout vorher und nachher. Kalt und warm.
- **`docs/en/getting-started.md`, `docs/de/getting-started.md`:** beim Punkt „Write barrier“ ein
  Absatz zur Registry und dazu, dass verknüpfte Worktrees eines Workspace mitzählen.
- **READMEs** (`README.md`, `README.de.md`): eine Zeile, falls sie die Schranke nennen.
- **Registry-Vorlage** in `%LOCALAPPDATA%\loomux\registry.toml`: Die Kommentarzeile „Ein
  Git-Worktree ist ein eigener Pfad und braucht einen eigenen Eintrag“ wird falsch. Der Agent schlägt
  den Ersatztext vor, der Nutzer schreibt ihn.

## Ablauf

1. Spec auf `master` im Hauptcheckout committen; der Nutzer prüft sie. Dann Plan mit
   `writing-plans`, ebenfalls auf `master`.
2. Im Worktree `loomux-sdd-1b1`: Zweig `sdd-1b-1` in `barrier-worktrees` umbenennen und auf den
   Stand von `master` bringen.
3. **Nutzer:** Registry-Block für `loomux-sdd-1b1` eintragen, letztmalig. Probe aus Task 0 Step 4
   des 1b-1-Plans: Exit 0.
4. Umsetzung im Worktree nach Plan, Tor grün.
5. Merge nach `master` nach ausdrücklichem Ja des Nutzers, danach `bin/loomux.exe` im Hauptcheckout
   neu bauen — ein Fast-Forward löst das pre-commit-Tor nicht aus.
6. **Nutzer:** Registry-Block entfernen. **Abnahme:** dieselbe Probe liefert ohne Eintrag Exit 0.
7. 1b-1: neuer Zweig `sdd-1b-1` von `master` im selben Worktree. Task 0 Step 4 wird per erster
   Ruling im 1b-1-Ledger zu „Probe ohne Eintrag“.
