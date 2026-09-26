# Paritätsakte Stufe 3a

**Quelle:** ultra-brain `loomux-3-source` (`3cc72d2`) — derselbe Commit wie
`loomux-1a-source`; das Repo hat sich seit Stufe 1a nicht bewegt. Der Tag ist
am 2026-09-22 auf Weisung des Nutzers gesetzt worden, lokal und nicht
gepusht, vor dem Aufzeichnen der Fälle (Task 16).
**Spec:** `docs/.superpowers/specs/2026-09-19-loomux-stufe-3-design.md`
**Plan:** `docs/.superpowers/plans/2026-09-20-loomux-stufe-3a.md`

## Vorbedingungen, festgestellt am 2026-09-20

| Frage | Antwort | Belegstelle |
|---|---|---|
| Ist 2c gemergt? | **Nein.** `internal/cases/gitworld.go` gibt es auf diesem Zweig nicht | `ls internal/cases/gitworld.go` |
| Stellt `tools/cases.py` eine Welt mit `.git` her? | **Ja.** `stage_world` kopiert den Baum über `shutil.copytree` und schließt nichts aus | `tools/cases_core.py:101-105` |
| Vergleicht es `.git` danach? | **Nein.** `compare_trees` überspringt jeden Pfad mit einem Teil aus `DEFAULT_IGNORE`, und `.git` steht darin | `tools/cases_core.py:98,111,117` |

**Folgerung für 3a: der Rekorder reicht, ohne Umbau.** `reconcile` liest git
nur — `vcs.ShowBlob` für die Grundlinie, `ChangedPaths`/`CommitSubjects` für
die Merge-Belege. Alles, was es schreibt, liegt außerhalb von `.git`
(Fallakten im Prüfzentrum, Statistiken und Stempel unter
`<state>/maintenance/`) und wird darum verglichen. Der blinde Fleck des
Rekorders beginnt erst bei 3b, wo `approve` selbst committet.

**Folgerung für die Git-Welten:** solange 2c nicht gemergt ist, fehlt auf der
loomux-Seite `internal/cases/gitworld.go`. Fälle, die ein echtes Repo in der
`world/` brauchen, bleiben geparkt und stehen unten. Es wird **keine** zweite
Git-Welt daneben gebaut. **Nachtrag Task 16b (2026-09-22):** 2c ist gemergt
(`78604bb`), die drei geparkten Fälle sind aufgezeichnet und abgespielt,
Abschnitt „Entparkte Fälle" unten.

**Nachtrag Task 16 (2026-09-22): aufgezeichnet wurde mit `loomux dev
record-case`, nicht mit `tools/cases.py`.** Beide setzen `BRAIN_STATE_DIR`
auf das gestagte Verzeichnis (`cases.py:138`, `recordcase.go`, `recordEnv`),
aber `tools/cases.py` kennt kein `{{WORLD}}`: `stage_world` ist ein blankes
`copytree`, die Ausgabe wird nicht normalisiert, und `world_after` trägt den
absoluten Temp-Pfad. Eine Registry, die auf Bereiche **in** der Welt zeigt —
jede Welt dieser Stufe —, lässt sich damit nicht aufzeichnen, und der
Übersetzer dieses Repos liest ohnehin `{{WORLD}}`. Der Rekorder von 1b-1 tut
beides; er ist darum der Rekorder von 3a, mit `brain-mcp.exe` aus
`ultra-brain/.venv` als `--argv` (kein `uv`, das in `ultra-brain` schreiben
könnte). Gemessen, dass ein eingebettetes `.git` nicht in `testdata` passt: `git
add` einer Welt mit `world/repo/.git` legt einen Gitlink (`160000`) an, nicht
den Inhalt des Repos — der Grund, aus dem die Merge-Fälle geparkt waren, bis
`gitworld.go` das Repo beim Stagen baut.

## Abweichungen

| Fall | Alt (Python-Referenz) | Neu (loomux) | Begründung | Freigabe |
|---|---|---|---|---|
| Sperre | PID in der Sperrdatei, Übernahme nach Frist (`locking.claim`, `_take_over`). Die Registry sperrt `brain init` über `registry.toml.lock` (`init.py:366`), und die Datei wird nach dem Schreiben weggeräumt | Betriebssystem-Handle, leere Sperrdatei (`internal/lock`). Die Registry sperrt `area add` über `registry.lock` (`internal/config/registrywrite.go:18`), und die leere Datei **bleibt liegen**, auch nach der Ablehnung eines bekannten Scopes | Ein sterbender Prozess gibt das Handle ohne Aufräumen frei; der gehaltene Bytebereich wäre für Leser ohnehin unlesbar, solange er etwas bedeutet. Portiert in Stufe 1a. **Name und Liegenbleiben** zeigte erst die Fallsuite (Task 16, alle `area-add`-Fälle): die Zeile deckte den Mechanismus, nicht die Datei. Folge der zwei Namen: die beiden Werkzeuge schließen sich an **derselben** Registry nicht gegenseitig aus; im Normalbetrieb sind es aber getrennte Registries unter `%LOCALAPPDATA%\loomux` und `%LOCALAPPDATA%\brain` | freigegeben 2026-09-20, erweitert Ruling 2026-09-22 |
| Zustandsort | Artefakte und Stempel unter `%LOCALAPPDATA%\brain` | Geschrieben nach `LOOMUX_STATE_DIR`, das Altverzeichnis nur noch gelesen | Die befristete Ausnahme der 1b-1-Akte lief „bis Stufe 3"; 3a ist der erste Schreiber. `loomux migrate` (Stufe 4) zieht den Bestand um | freigegeben 2026-09-20 |
| `local_only`-Bereich in `reconcile` | `_proposers` baut einen Ollama-Proposer und legt einen Vorschlag neben den Fall | Fall mit `manual = true` und einem `note`, kein Vorschlag, kein Modellaufruf | Das lokale Modell ist Stufe 4. Ein Fall ohne Vorschlag bleibt entscheidbar, und die Feldsemantik ist genau dafür da (`case.py:52-58`: „a closed area with no local proposal"). Fällt mit Stufe 4 weg | freigegeben 2026-09-20; **fällt weg mit 4c-1** (`stufe-4c-1.md`) |
| `--state-dir` an `reindex`, `embed`, `reconcile`, `area add` | An allen vieren angenommen (`cli.py:463-470`, `:596`, `:657`) | `unrecognized arguments: --state-dir …`, Exit 2; der Zustand kommt aus `LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR` | Dieselbe Entscheidung wie in 1b-1 für die fünf brain-Befehle, aus demselben Grund: **ein** Zustandsmodell für alle Befehle, und das ist die Umgebung. Die Übersetzung der Fälle entfernt die Flagge, wie 1b-1 es tat. **Nicht** aus dem Grund, der im Rumpf von `a95a380` steht: dort heißt es, ein `--state-dir` verschöbe nur den halben Zustand und ließe „read from one world and its stock written to another" — das ist am 2026-09-20 gemessen widerlegt, ein `--state-dir X` setzte genau die Variable, die `LOOMUX_STATE_DIR=X` heute setzt, mit byte-gleichem Ergebnis. Die Spaltung „liest aus dem Altverzeichnis, schreibt ins neue" **ist** die Rückfallregel dieser Stufe und tritt ohne jede Flagge auf | Ruling 2026-09-20 |
| `reindex`/`embed` ohne Registry | `_parse` (`registry.py`) fängt nur `TOMLDecodeError`; der `FileNotFoundError` endet im Sammelfänger als `error:`-Zeile, Exit 1 | `no areas registered in <pfad>; nothing to index` bzw. `… to embed` auf stdout, Exit 0. Eine mit `--registry` ausdrücklich **benannte**, fehlende Datei bleibt rot | Eine Maschine, auf der nie ein Bereich angemeldet wurde, hat nichts zu indizieren, und das ist eine Aussage, kein Zusammenbruch. Kein Register rückt dabei vor, es gibt also keinen Weg am Prüftor vorbei. Entschieden in Task 5, festgehalten in Task 14 | Ruling 2026-09-22 |
| `vcs.HooksDirectory` | Teil von `vcs.py` | Zieht in 3a **nicht** mit um | Der einzige Aufrufer ist `merge_events.consenting` (`merge_events.py:286`), die Hook-Verwaltung — und die ist Stufe 4 (Nachtrag #5). Zieht dort mit um | Ruling 2026-09-20 |
| `vcs._reject_operation_in_progress` | Teil von `vcs.py` | Zieht in 3a **nicht** mit um; `internal/brain/vcs` lehnt ein Repo mitten im Rebase, Merge oder Cherry-Pick **nicht** ab | Der Aufgabenbrief der Task 6 behauptete den Lesepfad; `grep -rn "_reject_operation_in_progress" src/` gibt genau einen Aufrufer, und der ist `commit_paths` (`vcs.py:185`), die Schreibseite. Die beiden Belegtests (`test_a_rebase_in_progress_is_refused:414`, `test_an_unfinished_merge_is_refused:427`) fahren ebenfalls `commit_paths`. Der Docstring (`vcs.py:265-272`) begründet mit dem **verwaisten Commit** („orphaned by the branch reset that ends it"), nicht mit einem unlesbaren HEAD — für das Lesen ist HEAD mitten im Rebase sogar der Stand, gegen den der Nutzer arbeitet. Zieht mit `commit_paths` in Stufe 3b um | Ruling 2026-09-20 |
| `vcs.ShowBlob` bei einem git-Fehlschlag | `None` für jeden Weg, auf dem die Fassung fehlen kann | `nil, nil` ebenso; einziger Fehler ist `ErrOutsideRepository` | Keine Abweichung, sondern die Festschreibung gegen den Aufgabenbrief, der „jeder andere Fehlschlag ist ein Fehler" behauptete. `_git_bytes` (`vcs.py:384-388`) faltet den `OSError` ausdrücklich zu `1, b""`: „raising here would take a whole reconciliation down over one unreadable source." Belegt durch `test_show_blob_is_none_without_a_repository:468`, `…_on_an_unborn_branch:476`, `…_when_the_directory_is_gone:488`. Loomux fügt eine Zusage hinzu, die Python geschenkt bekommt: eine committete **leere** Datei kommt als leeres, nicht-nil Slice an, weil `nil` hier „keine Grundlinie" heißt | Ruling 2026-09-20 |
| Zeitstempel eines Merge-Ereignisses ohne Zone | `datetime.fromisoformat` nimmt ihn als **naiven** Wert an; die Zeile ist ein Ereignis (`_parse_event`, `merge_events.py:551`) | `pytext.ParseAwareIsoFormat` gibt `false`, die Zeile ist eine kaputte und wird übersprungen | `MergeEvent.At` ist in Go `time.Time` und kennt kein „naiv"; der einzige Schreiber des Protokolls ist der Hook, und der schreibt ausnahmslos `%Y-%m-%dT%H:%M:%SZ` (`merge_events.py:112`), also erreicht nichts Echtes den Unterschied. Zudem liest `reconcile` das Feld gar nicht (`reconcile.py:511-529` benutzt `repo`, `first`, `last`), es ist reines Zeilentor. Festgehalten durch `TestReadEventsSkipsAStampWithoutAZone`. Python nimmt darüber hinaus verkürzte Formen, Wochendaten, `24:00` und mehr an — dieselbe Paritätsliste, die `pytext.ParseAwareIsoFormat` schon für den Reconcile-Stempel führt | Ruling 2026-09-20 |
| `merge_events.record_event` | Teil von `merge_events.py` | Zieht in 3a **nicht** mit um | Der einzige echte Schreiber des Protokolls ist der `post-merge`-Hook, und dessen Verwaltung ist Stufe 4. Gemessen: `grep -rn "record_event" src/ tests/` in ultra-brain gibt genau die Definition (`merge_events.py:227`) und Aufrufe in `tests/maintenance/test_merge_events.py` und `tests/maintenance/test_reconcile_merge.py` — **kein** Produktivaufrufer. Die Testwelten der Stufe 3a bringen die Protokolldatei fertig mit, statt sie zu schreiben; eine Go-Form hätte damit keinen Nutzer außer ihrem eigenen Test | Ruling 2026-09-20 |
| `write_case` und ein `created` ohne Zone | `CaseError`: „created must be timezone-aware" (`case.py:136-141`) | `WriteCase` lehnt die **Nullzeit** ab, nicht „naiv" | Go kennt die naive Form nicht: jede `time.Time` trägt eine Location, und `pytext.IsoFormat` schreibt ausnahmslos einen Offset. Die Schreibseite hat also nichts zu prüfen; was in den freigewordenen Platz tritt, ist die schwächere Wache gegen einen Fall, dem niemand einen Zeitpunkt gegeben hat. **Die Zonenregel selbst ist nicht verlorengegangen, sondern steht auf der Leseseite** — `zoneless` lehnt ein `created` in einer der drei Zonen `datetime-local`, `date-local` und `time-local` ab, die `third_party/toml/internal/tz.go` beim Offset dieser Maschine baut. Das ist genau die stille Umdeutung, die `read_case` (`case.py:188-193`) meint, und eine handgeschriebene `case.toml` ist der Normalfall | Ruling 2026-09-20 |
| `note`, `superseded_proposal`, `prompt_version` mit einem Wert, der kein String ist | `data.get("note")` reicht jeden Typ ungeprüft durch; das Feld ist `str | None` und hält dann eine Lüge | `ReadCase` lehnt ab: „note must be a string, found an integer" | Ein Go-`string`-Feld hat für einen anderen Typ keinen Platz, und ein stilles `""` wäre die schlechtere Antwort als eine Ablehnung. Die drei Pflichtstrings prüft Python selbst genauso streng (`_require_str`) — die Lücke besteht nur bei den optionalen | Ruling 2026-09-20 |
| `note = ""` in einer Datei, die Python geschrieben hat | `note is not None` heißt „schreib es", also steht `note = ""` in der Datei und kommt als `""` zurück | `""` **ist** hier die Abwesenheit: `ReadCase` gibt `""`, `WriteCase` lässt den Schlüssel dann weg | Die Schnittstelle der Task 8 gibt `Note string` vor, und ein Zeiger brächte die Unterscheidung in jeden Aufrufer für ein Feld, das sie nirgends braucht — dieselbe Entscheidung wie bei `config.Area.WikiPath`. **Die Folge ist ein einmaliges falsches „geändert"**: eine von Python mit `note = ""` geschriebene Fallakte wird beim ersten loomux-Schreiben ohne den Schlüssel neu gerendert und als Änderung gemeldet. Gemessen am 2026-09-20: **kein Produktivaufrufer setzt ein leeres `note`.** `grep -n "note=" src/brain/maintenance/*.py` gibt fünf Treffer, davon **vier Schreiber** — der fünfte ist `case.py:212`, `read_case`s eigenes `note=data.get("note")`, also ein Leser. Von den vieren reicht `reconcile.py:796` `_note_for` durch, das `None` oder eine der beiden Konstanten `_PROPOSAL_REJECTED_NOTE`/`_LOCAL_ONLY_NOTE` gibt (`:836-848`), und `apply.py:929`, `:969` und `:1049` setzen `_MOVED_NOTE`, einen nicht-leeren f-String bzw. `_AMEND_NOTE`/`_REFUSED_NOTE`. Der Unterschied ist damit heute unerreichbar; er wird erreichbar, sobald jemand ein leeres `note` schreibt | Ruling 2026-09-20 |
| `state` und `trigger` auf der **Schreib**seite | `write_case` prüft **keines** von beiden; `Case.state` ist ein blankes `str`, kein `Literal`, und die Vokabulare `STATES`/`TRIGGERS` (`case.py:22-23`) liest nur `read_case:177-184` | `WriteCase` hält beide gegen dieselben geschlossenen Vokabulare wie `ReadCase`, mit demselben Wortlaut (`outsideVocabulary`) | **loomux schließt eine Lücke, die die Referenz hat.** Ohne die Prüfung schreibt Python eine Fallakte, die sein eigener Leser ablehnt — und eine Fallakte ist genau das, was ein Mensch später lesend entscheidet; `read_case` begründet seine Strenge selbst damit, dass ein falsch geschriebener Zustand „must fail loudly, not fall back to a guess" (`case.py:20-21`). Kein Byte-Paritätsrisiko: die Prüfung steht vor der Renderung und ändert an einer angenommenen Akte nichts. Eine von Python geschriebene Akte mit unbekanntem Zustand war schon vorher unlesbar, also verliert niemand einen Bestand | Ruling 2026-09-20, Fix-Runde 1 |
| Segmentart, für die es kein Wort gibt (`RenderPackage`) | `_KIND_LABELS[segment.kind]` endet im `KeyError` (`package.py:156`) | `panic`: „segment D1 has no known kind" | Ein Go-Map-Zugriff gäbe `""` und schriebe `## S1 — , label` — eine Überschrift, die `evidence._SEGMENT_HEADING` weiter als Überschrift liest, also eine Segmentart, der niemand zugestimmt hat, still eingeschmuggelt. Über `BuildPackage` unerreichbar; erreichbar nur über ein von Hand gebautes `Segment`, und dafür ist der laute Ausgang der ehrliche. Kein Fehlerwert, weil die Schnittstelle `RenderPackage` einen blanken `string` gibt | Ruling 2026-09-20 |
| `render_package` und ein `created` ohne Zone | `_generated_at` wirft `ValueError`: „case.created must be timezone-aware" (`package.py:130-135`) | Kein Zweig dafür | Dieselbe Lage wie bei `write_case` eine Zeile höher: Go kennt die naive Form nicht, jede `time.Time` trägt eine Location, und `pytext.IsoFormat` schreibt ausnahmslos einen Offset. Anders als dort tritt hier **nichts** in den freigewordenen Platz — `RenderPackage` gibt keinen Fehler, und die Nullzeit einer Fallakte hat `WriteCase` schon abgelehnt, bevor ein Paket daneben entsteht | Ruling 2026-09-20 |
| Reihenfolge des Rückwärtsindex (`Dependents`) | `dict` über `sorted(edges.items())`: Schlüssel **und** Werte sortiert | `map[string][]string`: Werte sortiert, Schlüssel ungeordnet | Der einzige Aufrufer schlägt nach (`index.get(doc_id, ())`, `reconcile.py:470`) und läuft die Schlüssel nie ab; eine Go-Map kann keine Ordnung halten, und eine sortierte Paarliste hätte für den Nachschlag den falschen Zuschnitt. Die Zusage, auf die es ankommt — zwei Läufe über ein unverändertes Bündel geben dasselbe —, liegt in den sortierten Wertelisten. Fällt auf, sobald jemand den Index selbst rendert | Ruling 2026-09-20 |
| Das Wiki-Verzeichnis selbst in `Dependents` | `wiki_path.rglob("*.md")` läuft, was **darin** liegt, und gibt die Wurzel nie zurück | `filepath.WalkDir` ruft zuerst für die Wurzel zurück; `path == wikiPath` springt darüber | Keine Abweichung, sondern die Angleichung: ohne den Sprung läse ein Bündel, dessen Wiki-Ordner `wiki.md` heißt, sich selbst als erste Seite und scheiterte daran. Gegenstück zur Zeile darunter — ein Verzeichnis **unterhalb** der Wurzel, das auf `.md` endet, wird sehr wohl gelesen, weil `rglob` es ebenfalls liefert und `read_page` daran scheitert. Belegt durch `TestDependentsIsNoPageOfItself` und `TestDependentsAnswersAPageItCannotRead` | Ruling 2026-09-20 |
| `_read_stats` über Bytes, die kein UTF-8 sind | `path.read_text(encoding="utf-8")` wirft `UnicodeDecodeError` und nimmt den ganzen Lauf mit | `readStats` prüft UTF-8 **gar nicht**: `string(data)` trägt die kaputten Bytes wörtlich in den Schlüssel, die Zeile bleibt stehen und die Zeilen um sie herum wirken weiter | **Die Zeile hier stand bis zur Fix-Runde falsch** — sie behauptete, loomux verwerfe die Datei. Gemessen mit einer Sonde direkt auf `readStats` am 2026-09-20: `"pfad\tmtime_ns\tsize\n\xff\xfe.md\t1\t2\nb.md\t3\t4\n"` gibt **zwei** Einträge, den guten und einen mit den kaputten Bytes als Schlüssel. Der kaputte trifft auf keinen Pfad, den die Begehung liefert, also ist er genau eine Hashung wert — der Preis, den `_read_stats` (`reconcile.py:996-1001`) für jede beschädigte Zeile ausdrücklich einkalkuliert, während sein eigener Code für die kaputte **Datei** den ganzen Lauf mitnimmt. Unerreichbar, solange nur `_write_stats`/`writeStats` schreiben: beide schreiben UTF-8. Festgehalten durch `TestReadStatsCarriesBrokenBytesIntoTheKey` | Ruling 2026-09-20, Fix-Runde 1 |
| `_read_stats` über eine vorhandene, unlesbare Datei | `read_text` wirft den `PermissionError` (oder was das Betriebssystem sonst gibt) nach oben | `readStats` gibt einen leeren Zwischenspeicher, der Lauf hasht jede Quelle neu | Dies ist der **zweite** Auslöser, den die Zeile darüber bis zur Fix-Runde mit dem ersten vermengt hatte. Derselbe Grund und derselbe Preis: eine Hashung je Quelle gegen einen abgebrochenen Lauf für eine Datei, die der Lauf gerade neu schreiben würde. Festgehalten durch `TestReadStatsIsEmptyWhenTheFileCannotBeOpened` | Ruling 2026-09-20, Fix-Runde 1 |
| Stempelfeld mit Nicht-ASCII-Ziffern oder jenseits von 2⁶³ | `"٣".isdigit()` ist `True` und `int("٣")` gibt `3`; `int` hat keine Obergrenze | `parseStamp` nimmt nur `0`–`9` und nur, was in ein `int64` passt; sonst ist die Zeile eine beschädigte und fällt weg | Die Kosten sind eine Hashung mehr, genau die, die `_read_stats` für jede beschädigte Zeile ausdrücklich in Kauf nimmt. Die Alternative wäre ein umgeschlagener Stempel, der eine echte Datei aus Versehen als frisch ausweist — das wäre eine **verschwiegene Änderung**, und nur die ist hier teuer. Kein Schreiber erzeugt so eine Zeile: `mtime_ns` und `size` kommen aus `os.stat` | Ruling 2026-09-20 |
| `_note_for`, zweiter Wortlaut | Zwei Notizen: `_LOCAL_ONLY_NOTE` ohne Proposer, `_PROPOSAL_REJECTED_NOTE` nach einem verworfenen Vorschlag (`reconcile.py:836-848`) | Nur der erste. `noteFor` kennt den zweiten nicht | Gegenstück zur Zeile über `_proposers`: ohne lokales Modell gibt es keinen verbrauchten Versuch, den der zweite Wortlaut beschreibt. Der zweite Satz zieht mit Stufe 4 ein, zusammen mit dem Proposer, der ihn auslöst | Ruling 2026-09-20; **fällt weg mit 4c-1** (`stufe-4c-1.md`) |
| `_declared_review` und ein `[layout] review`, das kein String ist | `isinstance(declared, str)`, sonst `ReconcileError` (`reconcile.py:349-353`) | Kein Zweig dafür | `Manifest.LayoutReview` ist ein `string`; `config.ReadDeclaration` hat den Typ schon beim Lesen entschieden. Dieselbe Lage wie bei `write_case` und `render_package` und dem naiven `created` | Ruling 2026-09-20 |
| `_declared_review` und ein absoluter oder gewurzelter Wert | `area.path / declared` **ersetzt** die Wurzel, wenn `declared` absolut ist; `is_relative_to` fängt es danach | `declaredReview` prüft **ausdrücklich** auf einen führenden `/` oder `\` und auf einen Volume-Namen, **bevor** es zusammensetzt | Keine Abweichung, sondern die Angleichung an eine Semantik, die Go nicht hat: `filepath.Join("C:/a", "C:/x")` gibt `C:\a\C:\x` — ein Pfad, der bequem *innerhalb* des Bereichs liegt, während `PureWindowsPath("C:/a") / "C:/x"` `C:/x` gibt. Ohne die Vorprüfung ließe loomux genau den Wert durch, für den die Prüfung da ist. Belegt durch `TestReconcileRefusesAReviewCentreOutsideItsArea` mit vier Werten (`../aside`, `/aside`, `\aside`, `C:/aside`) | Ruling 2026-09-20 |
| `_declared_review` und ein `[layout] review`, das über einen Link aus dem Bereich zeigt | `root.resolve().is_relative_to(area.path.resolve())` — `resolve()` folgt jedem Link, der Wert wird abgelehnt | **Keine Abweichung mehr.** `declaredReview` entscheidet die Eingrenzung über `guard.ResolvePath`/`guard.IsRelativeTo`, also über denselben Auflöser wie die Schreibschranke | **Aufgenommen als Lücke, in Fix-Runde 1 geschlossen.** Vorgeführt am 2026-09-20: eine Junction aus dem Bereich heraus ließ `guard.reviewCentre` `""` antworten (Ausnahme zurückgezogen), während `maintenance.ReviewRoot` den Ort annahm und ein Schreiben **draußen** landete. Zwei Leser eines Feldes, die verschieden auflösen, sind schlimmer als jede der beiden Antworten. `filepath.EvalSymlinks` schließt das **nicht**: Windows meldet eine Junction als `ModeIrregular` statt als Link, `os.Lstat` sieht keinen, und nur `os.Readlink` löst sie auf — dafür trägt `guard` sein `finalPath`/`readlinkDeep`. Darum sind `resolvePath` und `isRelativeTo` zu `ResolvePath` und `IsRelativeTo` **umbenannt** statt mit einer Hülle daneben versehen: eine Funktion, ein Name, und an der Logik der Schranke ist nichts verschoben. Nebenfolge mitgenommen: `supersede` ruft `os.RemoveAll` auf einen verbrauchten Fallordner, und unter einer Junction landet das im Ziel — das Ziel muss jetzt im Bereich liegen. Belegt durch `TestReconcileRefusesAReviewCentreBehindALinkOutOfTheArea` und sein Gegenstück, das eine Junction **innerhalb** des Bereichs annimmt | geschlossen 2026-09-20, Fix-Runde 1 |
| `_standing_case` über eine Fallakte, die nicht zu lesen ist | Fängt nur `CaseError`; ein `OSError` nimmt den ganzen Durchgang mit (`reconcile.py:945-953`) | `standingCase` trägt **jeden** Fehler von `ReadCase` in `Unreadable` ein und geht weiter | `ReadCase` gibt für beides einen blanken `error` und hat keinen eigenen Typ — Task 8 hat das mit der Begründung entschieden, dass nichts auf einem Fallfehler etwas entscheidet außer ihn zu melden. Die nachsichtige Lesart ist zudem die, für die die Liste da ist: „one typo must not stop the pass for every other area". Der Preis ist, dass eine Fallakte, die das Betriebssystem nicht hergibt, als kaputte Akte gemeldet wird statt als kaputtes Zustandsverzeichnis | Ruling 2026-09-20 |
| `_supersede` über einen vorhandenen, unlesbaren Vorschlag | `proposal.exists()` entscheidet, `read_bytes()` wirft danach | `supersede` liest und wertet jeden Lesefehler als „liegt keiner da" | Ein zweiter Zweig für „ist da, geht aber nicht auf" wäre über keine Eingabe erreichbar, die diese Suite bauen kann, und die Folge ist in beiden Fällen dieselbe: der alte Ordner geht, der Vorschlag ist weg. Der Unterschied ist eine Meldung, die niemand bekommt, gegen einen Zweig, den niemand prüft | Ruling 2026-09-20 |
| `difflib.SequenceMatcher` und `isjunk` | `find_longest_match` hat vier Schleifen, die einen gefundenen Block über Junk-Zeilen hinweg wachsen lassen | Nicht portiert | `unified_diff` baut den Matcher ausnahmslos mit `SequenceMatcher(None, a, b)`, also ist `bjunk` bei jedem Aufruf dieses Pakets leer und keine der vier Schleifen kann einen Schritt tun. `autojunk` ist die andere Hälfte von `__chain_b` und **ist** portiert — sie ist voreingestellt an, eine 200-Zeilen-Quelle ist gewöhnlich, und die Mutanten „ohne autojunk", „Schwelle 100" und „ntest um eins daneben" sterben alle drei an `testdata/hunks.golden.json` | Ruling 2026-09-20 |
| `read_last_run` ohne Datei, mit unlesbarem Stempel oder ohne Zone | `None` für alle drei; ob die Datei da ist, entscheidet `path.exists()`, das jeden `stat`-Fehler schluckt (`reconcile.py:1026-1043`) | **Keine Abweichung mehr.** Der einzige Leser ist `search.ReadLastRun` (`internal/brain/search/stamp.go`): kein Stempel (`false`, kein Fehler) für alle drei und für Bytes, die kein UTF-8 sind; ein Fehler nur für eine vorhandene Datei, die sich nicht lesen lässt | **Im Schlussreview gestrichen, was hier bis dahin stand:** `maintenance.ReadLastRun`, ein zweiter Leser derselben Datei ohne Produktivaufrufer, der anders antwortete — die Nullzeit statt `false`, und `os.ReadFile` mit `ErrNotExist` statt `Path.exists`, also ein Fehler, wo Python einen `stat`-Fehler schluckt. `Report.LastRun` las niemand. Beide sind weg; `Reconcile` schreibt den Stempel weiter (`writeLastRun`), gelesen wird er von `search` und `status`. Die Aufholung vor jedem `reindex` liest ihn nicht, sie läuft bedingungslos; ein Leser im Upkeep ist Sache von 3c | Ruling 2026-09-20, Schlussreview 2026-09-22 |
| Entdoppelung der Fälle nach `id` | `reconcile` entdoppelt jeden Lauf (`reconcile.py:216-221`) | **Keine Abweichung mehr.** `dedupedByID` entdoppelt jeden Lauf, erste Fassung gewinnt | **Mit Task 12 eingezogen.** Die Zeile stand hier, solange es nur einen Erzeuger gab: ein Quellfall entsteht je Bereich und Ziel genau einmal, und `CaseID` nimmt den Bereich in seinen Abdruck. Der Merge-Auslöser ist der Fall, den sie braucht — dasselbe Ereignis unter zwei Schreibweisen eines Repositorys hat zwei Schlüssel, löst auf denselben Bereich auf und reicht denselben stehenden Fall zweimal heraus. Belegt durch `TestReconcileDeduplicatesByID` | geschlossen 2026-09-20, Task 12 |
| `_candidates`, Reihenfolge der offenen Seiten | `sorted(area.wiki_path.rglob("*.md"))` sortiert **Pfadobjekte** | `candidates` sortiert die relativen Pfade als Zeichenketten | Dieselbe Unstimmigkeit, die `Dependents` schon führt: `-` steht unter `/`, also ordnen Pfadteile und Zeichenketten `a-x.md` und `a/x.md` verschieden. Hier entscheidet sie nichts — jeder Kandidat bekommt seinen eigenen Fall, und jede `id` kommt aus ihrem eigenen Ziel, nicht aus einer Position in der Liste | Ruling 2026-09-20 |
| `_merge_evidence`, zwei Lesungen und ein Urteil | Ruft `changed_paths` und `commit_subjects` **beide** und prüft danach `paths is None or subjects is None` | `mergeEvidence` ruft beide und prüft `pathsErr != nil || subjectsErr != nil` — dieselbe Form | Keine Abweichung, aber ein Operand ohne Vektor: derselbe Bereich im selben Repository lässt beide Aufrufe gemeinsam scheitern. Gesucht ist ein Objektname, den `git diff` liest und `git log` ablehnt; es ist keiner bekannt. Die beiden Kandidaten am 2026-09-20 mit git 2.54.0 probiert: einen Tree-Namen nehmen beide an; einen Blob-Namen nimmt `git log` hier an, während andere git-Stände ihn `git diff` verweigern (Prüfer, 2026-09-20: exit 129) — er ist damit ein Vektor für den **ersten** Operanden, nie für den zweiten. Der zweite steht da, weil er dieselbe Antwort ist, nicht weil ein Test ihn erreicht | Ruling 2026-09-20 |
| `_absorbable` über ein `package.md`, das nicht zu lesen ist | `is_file()` entscheidet, `read_text()` wirft danach | `absorbable` liest und wertet jeden Lesefehler als „nicht aufsaugbar" | Gegenstück zur Zeile über `_supersede`: beide Antworten bedeuten dasselbe — auf der Platte steht nichts, was zeigt, dass dieser Fall diese Belege trägt —, und die Folge ist in beiden Fällen, dass das Ereignis zurücktritt statt zu landen. Die nachsichtige Lesart ist hier zusätzlich die sichere: ein geworfener Fehler nähme den ganzen Durchgang mit | Ruling 2026-09-20 |
| `_merge_cases`, Fehler beim Ablegen eines verbrauchten Ereignisses | `drop_event` schreibt und wirft, was der `open()` wirft; `_merge_cases` fängt nichts | `mergeCases` gibt den Fehler nach oben und nimmt den Durchgang mit | Keine Abweichung — dieselbe Richtung, nur sichtbar gemacht. Wichtig ist, was sie **nicht** ist: die beiden Stellen darüber, `commonOf` und `mergeEvidence`, schlucken ihren Fehler ausdrücklich und lassen das Ereignis stehen. Ein `return err` dort verlöre einen Merge still und für immer; festgenagelt durch `TestAnUnreadableRangeKeepsTheEvent` und `TestAnUnresolvableRepositoryKeepsTheEvent` | Ruling 2026-09-20 |
| `_common` und die `is_dir()`-Vorprüfung | `if not directory.is_dir(): return None` steht vor dem git-Aufruf | `commonOf` prüft nicht vor und ruft git auch für ein Verzeichnis, das es nicht gibt | Dieselbe Antwort, ein Prozess mehr: `exec` scheitert beim `chdir`, `readPath` gibt den Fehler, `commonOf` faltet ihn auf `""`. Die Vorprüfung ist bewusst weg, weil sie den Fehlerarm sonst unerreichbar machte — und genau dieser Arm ist die Klausel (b) aus Task 6, die nie einen Fehler nach oben geben darf. Ein gespartes `git rev-parse` gegen einen Zweig, den kein Test erreicht, ist der schlechtere Tausch | Ruling 2026-09-20 |
| `_addresses`, Tiefe der Fallsuche | `_case_files` geht mit `rglob("case.toml")` durch den ganzen Baum des Prüfzentrums (`cli.py:1121-1134`) | `caseAddresses` liest genau zwei Ebenen, `<review>/<scope>/*/case.toml` | Ein Fall, den ein Leser tiefer geschoben hat, ist für den Durchgang, der diese Auflistung erzeugt, unsichtbar: `standingCase` sieht nur `<review>/<scope>/*` (`reconcile.go:866-875`) und eröffnet einen frischen Fall daneben. **Ohne Wirkung ist die Abweichung trotzdem nicht** — bis zum Schlussreview stand hier, die tiefere Begehung fände „ausschließlich Fälle, über die dieser Befehl nie berichten kann", und das ist falsch: hat der verschobene Fall dasselbe `(area, target)` wie der frische, konkurrieren beide in `_addresses` um einen Schlüssel, und wo der Pfad des verschobenen später sortiert, druckt die Referenz **dessen** Ordnernamen als Adresse des neuen Falls. loomux druckt die Adresse des Falls, den der Durchgang gerade eröffnet hat. Ein verschobener Fall mit anderem `(area, target)` bleibt ohne Wirkung, weil die Auflistung nur die Fälle des Laufs nachschlägt. Fällt ganz auf, sobald ein Befehl die Warteschlange selbst auflistet — `brain cases` hat in 3a keine Entsprechung | Ruling 2026-09-20, Task 13; Wortlaut korrigiert im Schlussreview 2026-09-22 |
| `area add`, Schlüssel des Merge-Zweigs | `write_manifest` schreibt `[maintenance] merge_branch` (`init.py:152`), der Leser fragt `maintenance.get("branch", "main")` (`manifest.py:41`) | Geschrieben wird `[maintenance] branch`, der Schlüssel, den `declaration.go:81` liest | In der Referenz war `--merge-branch` damit **wirkungslos**: jedes Manifest aus `brain init` wurde mit `"main"` gelesen, auch wo `_detect_branch` `master` erkannt hatte. loomux schreibt, was es liest. Datei-Parität bricht an genau dieser Zeile | Ruling 2026-09-22 |
| `area add --privacy` mit einem unbekannten Modus | Wird ungeprüft ins Manifest geschrieben | Exit 2, `--privacy must be one of automatic_cloud, local_only, manual_cloud` | Der Leser der Referenz selbst (`manifest.py:12`, `_PRIVACY_MODES`, geprüft `:28-32`) weist ab, was ihr Schreiber hineinschreibt; in loomux macht ein solches Manifest die Schreibschranke im ganzen Repo zur Totalverweigerung | Ruling 2026-09-22 |
| `area add --path`, das kein Verzeichnis ist | Ein **fehlender** Pfad wird über `ultra_dir.mkdir(parents=True)` (`init.py:126`) angelegt und registriert; ist der Pfad eine **Datei**, scheitert das `mkdir` mit Traceback, vor der Registrierung | Beides Exit 1, `… is not a directory`, vor dem ersten Schreiben | Ein vertippter Pfad würde sonst zu einem neuen Verzeichnis und einem registrierten Bereich, den niemand gemeint hat | Ruling 2026-09-22 |
| `area add`, Wirtsteil | `configure_mcp` schreibt `.mcp.json`, `configure_agent_hooks` die Hook-Dateien (`init.py:434-435`) | Nicht geschrieben | Wirts-Installation gehört `loomux init` in Stufe 4 (Spec, Nachtrag #5). Der aufgezeichnete Fall wird `.mcp.json` zeigen | Ruling 2026-09-22 |
| `InitBundle`, was als vorhanden gilt | `init_bundle` überspringt jedes `exists()` (`scaffold.py:93`) | Übersprungen wird nur eine **reguläre Datei**; ein Verzeichnis `log.md` ist ein Fehler mit Dateinamen | Ein Verzeichnis dieses Namens ist kein Protokoll; das stille Überspringen hinterließe ein Bundle ohne eines. Belegt durch `TestInitBundleRefusesADirectoryWhereAFileBelongs` | Ruling 2026-09-22 |
| `area add`, Reihenfolge und doppelter Scope | Manifest → Regel → Registry; `_append_area` gibt bei bekanntem Scope `False` zurück, `run_init` läuft weiter und endet mit Exit 0 — der Befehl ist **wiederholbar** | Registry zuerst; ein bekannter Scope ist Exit 1 (`config.AddArea`), Repo unberührt | Mit der Reihenfolge der Referenz bliebe bei einer Ablehnung ein halb eingerichtetes Repo stehen. Der Preis: ein zweiter Lauf als Reparatur ist in loomux **nicht** möglich, der Bereich muss dafür erst aus der Registry | Ruling 2026-09-22 |
| `area add`, Indexlauf | `run_reindex` wird übergeben (`cli.py:844`) und nie gelesen (`init.py:419`); `brain init` indiziert nie, `--no-reindex` ist wirkungslos | `area add` endet mit `loomux reindex`, Auffangdurchgang eingeschlossen; `--no-reindex` schaltet ihn ab | Spec und Brief verlangen den Indexlauf; beide Wege in den Index gehen durch denselben Auffangdurchgang. **Folge für die Fallsuite:** der aufgezeichnete Fall `brain init -y` hat keinen Indexlauf, `<repo>/_identities.tsv`, Kataloge und qmd-Aufrufe fehlen dort | Ruling 2026-09-22 |
| `area add`, altes `.brain.toml` | `write_manifest` gibt ein vorhandenes `.brain.toml` zurück und schreibt kein neues Manifest (`init.py:130-132`) | Kein Rückfall; nur `.loomux/config.toml` zählt | Gedeckt durch den Namensschnitt vom 2026-09-14 (`manifestNames` in `internal/config/manifest.go`) | Ruling 2026-09-22 |
| `area add` über eine vorhandene Konfiguration, die der Deklarationsleser ablehnt | `write_manifest` gibt sie zurück, `run_init` registriert den Bereich | Exit 1 vor dem ersten Schreiben, Datei und Fehler genannt; nichts registriert. Eine Konfiguration **ohne** `[area]` wird weiter behalten und nur gewarnt | Registriert, hielte sie den Auffangdurchgang vor **jedem** Indexlauf und jedes `reconcile` der Maschine an, für alle Bereiche; ein zweites `area add` zur Reparatur scheiterte als Duplikat. Gefunden in der Prüfung von Task 15, belegt durch `TestAreaAddRefusesAKeptConfigurationThatDoesNotRead` (mit Indexlauf) | Fix-Runde 1, Task 15 |
| Leeres `[index] include` | `include` ist `()` (`manifest.py:60`), und `_matches_any` mit leerem Tupel ist falsch (`walk.py:116`, `:153`): `find_files` liefert **keine** Datei. `reconcile` prüft keine Quelle des Bereichs, `reindex` schreibt ein **leeres** Register — **alle Identitäten des Bereichs gehen verloren** —, einen leeren Katalog und einen leeren Graphen, während qmd weiter `**/*.md` bekommt (`cli.py:266`) | `FindFiles` setzt `**/*.md` ein (`internal/brain/index/walk.go:155-157`); `reconcile` prüft, `reindex` behält die Identitäten, beide Indizes sehen dieselben Dateien. Greift **auch bei einem ausdrücklichen `include = []`**, weil beide Formen eine leere Liste ergeben | **Bug der Referenz, bewusst nicht übernommen.** Die Referenz widerspricht sich selbst: derselbe Lauf gibt qmd `**/*.md` und dem eigenen Index nichts. `brain init` schreibt ohnehin immer ein `include` (`init.py:134`), der Weg öffnet sich nur mit einem handgeschriebenen Manifest. Der Vorgabewert stammt schon aus dem Indexport (`3012bed`) und hatte bis hierher keine Zeile. Belegt durch `reconcile/area-without-include` (stdout `2 Quellen` gegen `1 Quelle`, `stats.tsv`) und `reindex/area-without-include` | Ruling 2026-09-22 |
| `graph.json`, Schlüsselreihenfolge | `json.dumps(…, indent=2, sort_keys=True)` (`graph.py:153`): `edges`, `links`, `nodes`, `scope`; je Knoten `id`, `tags`, `title` | Reihenfolge der Go-Struktur (`internal/brain/graph/render.go`): `scope`, `nodes`, `edges`, `links`; je Knoten `id`, `title`, `tags` | Inhalt gleich; die Fallsuite dekodiert beide Seiten und hält sie gleich (`formatOnly`). Anders als die beiden Zeilen darunter wird die Datei **neu geschrieben**: `write_if_changed` vergleicht Text, also kostet jeder Wechsel zwischen den Werkzeugen eine Änderungszeit. `graph.json` liegt bei einem schreibbaren Bereich **im Repo**, und `brain init` trägt sie nicht in `.gitignore` ein — der Wechsel erscheint dort als Diff | Ruling 2026-09-22 |
| qmds `index.yml`, Einrückung | `yaml.safe_dump(…, sort_keys=True)` (`qmd_config.py:151`, `:229`): zwei Leerzeichen, Listen nicht eingerückt | `yaml.v3` `Marshal` (`internal/brain/index/qmdconfig.go:294`): vier Leerzeichen, Listen eingerückt | Inhalt gleich (dekodiert verglichen). Gemessen: loomux schreibt eine von Python erzeugte `index.yml` **nicht** neu, weil beide Seiten dekodiert vergleichen, bevor sie schreiben | Ruling 2026-09-22 |
| `qmd-collections.json`, letzter Zeilenumbruch | `json.dumps(sorted(names), indent=1)` ohne `\n` (`qmd_config.py:205`) | `append(data, '\n')` (`internal/brain/index/qmdconfig.go:196`) | Inhalt gleich (dekodiert verglichen); aus demselben Grund wie bei `index.yml` nicht neu geschrieben | Ruling 2026-09-22 |
| `area add` ohne vorher angelegte `registry.toml` | `_init` liest die Registry vor dem ersten Schreiben: `_registered` → `read_registry` → `_parse` → `path.read_text`, und nirgends auf dem Weg wird die Existenz geprüft (`cli.py:821`); `error: [Errno 2] No such file or directory: '…registry.toml'`, Exit 1, nichts geschrieben | `config.AddArea` legt die Registry an, Exit 0, der Bereich ist angemeldet und indiziert | Mit der Referenz lässt sich der erste Bereich einer Maschine nur anmelden, wenn jemand die Datei vorher von Hand angelegt hat. Belegt durch `area-add/no-registry` (zusätzlich zum Fallsatz der Spec aufgezeichnet) | Ruling 2026-09-22 |
| Globs der Begehung: `include`, `exclude`, Artefaktnamen, `[layout] review` (S1, S2) | `_matches_any` (`walk.py:116-117`): `PurePosixPath(relative).full_match(pattern)`, ungefaltet; `**` deckt auch null Teile | **Keine Abweichung mehr.** `matchesAnyGlob` ruft `privacy.MatchesGlobsUnfolded`, den treuen `glob.translate`-Port aus 1b-1 ohne NFC und `casefold`; `never` bleibt beim gefalteten `MatchesGlobs`, wie `find_files` es begründet (`walk.py:136-139`) | **Geheilt.** Der Indexport (`3012bed`) hatte eine **zweite**, naive Übersetzung desselben Dialekts mitgebracht, `globToRegex`: byteweise, also traf kein Nicht-ASCII-Literal (`95 Prüfzentrum` wurde begangen, samt Paketen mit Quelldiffs), und `docs/**/*.md` verlangte ein Unterverzeichnis. Sie ist **gelöscht**, nicht repariert: zwei Übersetzungen eines Dialekts im Repo sind genau die Fehlerklasse, die beide Befunde erzeugt hat. Nebenfolge, treu zu Python: ein Muster mit `\` wird nicht mehr zu `/` umgeschrieben; gemessen am 2026-09-22 schreibt keines der elf Manifeste der Registry einen Rückstrich. Belegt durch `TestFindFilesExcludesAReviewCentreWithANonASCIIName`, `…ExcludesANonASCIILiteral`, `…IncludesAFileDirectlyBelowATwoStarInclude`, `…ExcludesAFileDirectlyBelowATwoStarExclude`, `TestMatchesGlobsUnfoldedAnswersLikeFullMatch` und den aufgezeichneten Fall `reindex/globs` (Welt `vault-globs`: Prüfzentrum `95 Prüfzentrum`, `docs/**/*.md` und `docs/**/draft.md` gegen Dateien direkt unter `docs`), der gegen die byteweise Übersetzung an zehn Dateien rot wird | Ruling 2026-09-22, geheilt |
| Linkgraph nach Anhang B der ub-Migrationsspec (S4) | `_drop_reason` (`graph.py:80-96`): `external` nur für `http://`, `https://`, `mailto:`; `_resolve` dekodiert, schneidet aber weder `?` noch `#` ab — `README.de.md#policy` ist `unknown_target` | `cleanTarget` (`internal/brain/graph/render.go:16-21`) schneidet ab dem ersten `#` oder `?` ab, bevor `ResolveTarget` (`:35`) dekodiert und auflöst; `DropReason` (`:60-62`) bucht **jedes** RFC-3986-Schema (`uriSchemeRegex`, `:13`) als `external` | Bewusst übernommen aus dem Go-`brain`: Anhang B der ub-Migrationsspec (`specs-ub/2026-09-04-python-nach-go-migration-design.md:241-243`) führt es als Fund der Referenz, „56 fehlende Kanten, 48 Links unter falschem Grund gebucht". Gemessen in der Selbstnutzung: `iam-wiki` 322 gegen 266 aufgelöste Links, `ultra-brain` 100 gegen 4 `external`. Kehrseite: auch ein Windows-Pfad als Linkziel (`C:/x`) hat ein Schema und zählt als `external`. Keine Welt der 3a-Fallsuite trägt einen solchen Link, darum sieht sie nur die Schlüsselreihenfolge von `graph.json` (Zeile oben). Stand bis hier in keiner Akte | Ruling 2026-09-22 |
| `qmd-collections.json` (S5) | Liegt im Zustandsverzeichnis der Referenz (`cli.py:368`) | Gelesen neu zuerst, sonst alt (`config.ArtifactLookup` über `ownershipRecord`, `internal/brain/index/reindex.go`), geschrieben nur neu | **Geheilt.** Die erste Fassung las nur das neue Verzeichnis. Auf dieser Maschine stehen die zehn Sammlungen aber in `%LOCALAPPDATA%\brain\qmd-collections.json`, und qmds `index.yml` führt sie: der **erste echte `loomux reindex`** hätte alle zehn als fremd verweigert („already exists … not created by brain"), mit Exit 1 geendet und qmd nie aktualisiert. Die bisherige Zeile der Selbstnutzung („Schaden null") war falsch. Die Datei rückt, wie in Python (`sync_collections`, `qmd_config.py:109-157`), erst mit dem ersten Lauf ins neue Verzeichnis, der `index.yml` ändert oder dessen Liste eine verwaiste Sammlung nennt — `PruneCollections` schreibt dann, auch wenn `DropCollections` nichts entfernt, weil die Sammlung in `index.yml` schon fehlt; bis dahin wird weiter die alte gelesen. Danach liest loomux nur noch die neue: eine Sammlung, die ein weiterlaufendes Python-`brain` später in die **alte** Liste schreibt, verweigert loomux — dieselbe Kante wie bei `merge-events.done.tsv` unten, `loomux migrate` schließt sie. Belegt durch `TestReindexReadsTheCollectionRecordFromTheLegacyDirectory` | Ruling 2026-09-22, geheilt |
| `stats.tsv` (S5) | Liegt unter `maintenance/` im Zustandsverzeichnis der Referenz | Nur aus dem neuen Zustandsverzeichnis gelesen (`statPath`, `internal/brain/maintenance/scan.go`), **ohne** Rückfall | Bewusst ohne Rückfall: die Datei ist ein Zwischenspeicher, kein Zustand. Die Folge ist ein Kostenpunkt, kein Fehler — der erste `reconcile` hasht jede Quelle (in der Selbstnutzung 915 von 915 gegen 71 von 901) und schreibt den Speicher dann neu | Ruling 2026-09-22 |
| Junctions in der Begehung (S6) | `rglob` folgt jeder Junction, auch unterhalb der Bereichswurzel; `space/.agents/skills` (Junction auf `.claude/skills`) bringt drei `SKILL.md` an `**/.claude/**` vorbei ins Register | **Die Wurzel** wird vor der Begehung mit `guard.ResolvePath` aufgelöst, derselbe Auflöser, den `declaredReview` für `area.Path` benutzt; die Pfade kommen unter der registrierten Wurzel zurück. **Verschachtelte** Junctions werden nicht verfolgt | Die Wurzel ist **geheilt**: Go meldet eine Junction als irregulär, `WalkDir` stieg nicht ab, und eine Wurzel, die selbst eine war, gab `[]` **ohne Fehler** — ein `reindex` schriebe ein leeres Register, alle Identitäten gingen verloren. Aufgelöst statt abgelehnt, weil `rglob` der Wurzel ebenso folgt und eine Ablehnung einen Bereich ausschlösse, den die Referenz indiziert; zurückgegeben unter `area.Path`, weil jeder Aufrufer (`collect`, `Scan`) gegen ihn misst. Eine laufwerksrelative Wurzel (`C:rel`) lehnt der Auflöser jetzt ab. Verschachtelt bleibt es **bewusst** abweichend: nicht zu folgen ist die sichere Richtung, sonst kämen Zwillinge einer ausgeschlossenen Ablage an den Ausschlüssen vorbei (qmd zählt dort wie loomux, 215 auf beiden Seiten). Belegt durch `TestFindFilesWalksAnAreaRootThatIsAJunction` und `TestFindFilesRefusesADriveRelativeRoot` | Ruling 2026-09-22, Wurzel geheilt |
| Verschachtelte Bereiche über einer Junction (`nestedAreas`) | `_nested_areas` (`cli.py:112-119`) löst beide Seiten auf: `root.resolve().is_relative_to(area.path.resolve())` | `nestedAreas` (`internal/brain/index/reindex.go:27-45`) vergleicht **lexikalisch** über `filepath.Rel`, und der Ausschluss in `FindFiles` misst ebenso gegen die registrierten Schreibweisen | Älter als die Heilung von S6, sichtbar erst durch sie: seit `FindFiles` eine Wurzel-Junction auflöst, wird ein Bereich abgegangen, dessen Kindbereich unter dem **Ziel**pfad der Junction registriert sein kann. Dann erkennt `nestedAreas` ihn nicht als Kind, und seine Dateien werden **doppelt** indiziert — im Kind und im Elternbereich, mit zwei `doc_id` für eine Datei. Heute ohne Folge: keiner der 19 registrierten Pfade dieser Maschine (`path` und `wiki` aus beiden Registries) ist eine Junction oder liegt unter einer (gelesen am 2026-09-22). Keine Code-Änderung in dieser Runde; die Heilung wäre, beide Seiten mit `guard.ResolvePath` aufzulösen wie Python | Ruling 2026-09-22 |
| `QmdConfigPath` und `XDG_CONFIG_HOME` | `qmd_config_path` liest `XDG_CONFIG_HOME` selbst, sonst `~/.config` (`src/brain/search/qmd_config.py:69-72`) | `index.QmdConfigPath` (`internal/brain/index/qmdconfig.go:140-151`) ebenso, und zwar **unterhalb** der Kompositionsstelle: `reindex` ruft es selbst (`internal/brain/index/reindex.go:274`), statt den Pfad hereingereicht zu bekommen | **Keine Abweichung, sondern eine benannte Ausnahme** von der Regel, der jeder andere Pfad dieser Stufe folgt: Zustand und Rückfall kommen über `config.ArtifactLookup` herein, weil `internal/serve` zusagt, dass alles am übergebenen Zustandsverzeichnis hängt. Sie bleibt, weil sie die Referenz treu abbildet und heute stimmt: die Aufrufer sind die CLI-Befehle `reindex` und `area add`, und für sie ist die Umgebung die Kompositionsstelle. Auflage an 3c unten | Schlussreview 2026-09-22 |
| Deklaration eines read-only-Bereichs, gelesen von `guard` | `manifest_path` (`src/brain/registry.py:132-138`) fragt im Artefaktverzeichnis `.ultra-brain/config.toml`, sonst `.brain.toml` | `guard.manifestPath` (`internal/brain/guard/registry.go:59-65`) sucht nur `.loomux/config.toml` in `config.ManifestDir` (Namensschnitt vom 2026-09-14, `internal/brain/guard/manifest.go:8-13`). `publish` (`internal/brain/index/staging.go:56-87`) kopiert den Bestand des alten Verzeichnisses aber **wörtlich** nach `<state>/areas/<scope>/`, `.brain.toml` eingeschlossen, und die brain-Leser nehmen ihn dort über `ReadAreaManifestUntilStage4` weiter an | **Zwei Leser einer Deklaration, die verschieden antworten.** Alle drei read-only-Bereiche dieser Maschine tragen ihre Deklaration nur als `.brain.toml` (gelesen am 2026-09-22 unter `%LOCALAPPDATA%\brain\areas\`). `guard` findet dort keine Datei und wertet das als „noch nicht deklariert": eine kaputte Deklaration lässt die Schranke durch, statt jeden Schreibzugriff zu verweigern (`checkDeclaration`), und was die Datei erklärt, sieht sie nicht. Neu ist die Blindheit nicht — vor dem ersten loomux-Lauf liegt unter `%LOCALAPPDATA%\loomux\areas\` gar nichts —, aber `publish` trägt sie in den Umzug hinein, statt sie aufzulösen. Keine Code-Änderung in 3a; Auflage an Stufe 4 unten | Schlussreview 2026-09-22 |
| `area add` über eine vorhandene Konfiguration mit **anderem** Scope | `write_manifest` gibt sie zurück (`init.py:128-129`), `run_init` registriert unter dem Scope der Befehlszeile, ohne ein Wort | Ebenso behalten und registriert, Exit 0; dazu die Warnung `… declares scope "X", the registry now names "Y"` (`warnAboutKeptManifest`, `internal/cli/area.go`) | Die Datei ist Policy und Prüfkette des Repos und bleibt unberührt (Task 15, Frage 2). Registry und Deklaration nennen danach zwei Namen für einen Bereich. Kein brain-Leser dieser Stufe vergleicht die beiden; `guard.declaredWikiRoot` schlägt den **deklarierten** Scope in der Registry nach, findet ihn nicht und öffnet über die Deklaration nichts — ohne Folge, solange `area add` `workspace = true` einträgt und die Schranke den Baum darüber öffnet. **Offen:** ob `area add` hier ablehnen soll | vermerkt im Schlussreview 2026-09-22, nicht entschieden |
| `area add` über eine Konfiguration mit `[area]`, aber ohne `[layout] wiki` | Behalten und registriert; der Wiki-Ort der Registry kommt aus der Erkennung | Ebenso, und **ohne** Warnung: `warnAboutKeptManifest` schweigt, sobald der Scope stimmt | Die Registry nennt dann einen Wiki-Ort (`<repo>/wiki` oder `<repo>/docs/wiki`, `InitBundle` legt ihn an), die Deklaration keinen. `WikiLayout` antwortet `""`: `guard.declaredWikiRoot` öffnet über die Deklaration kein Bündel, und `declaresWikiLayout` (`internal/hooks/post_edit.go`) fügt die Wiki-Lane nicht hinzu; `wiki.Root` findet das Bündel über seine Rückfälle `docs/wiki` und `wiki` trotzdem. **Offen:** ob die Warnung auch diesen Fall nennen soll | vermerkt im Schlussreview 2026-09-22, nicht entschieden |
| Wiki-Ort aus zwei Quellen: `--wiki` gegen `[layout] wiki` | `--wiki` verschiebt Bündel und Registry-Eintrag; `[layout] wiki` im neu geschriebenen Manifest bleibt der erkannte Ort (`cli.py:835-838`) | Ebenso (`planArea`, `internal/cli/area.go:176-190`) | Referenzverhalten, übernommen. Registry und Deklaration laufen damit auseinander: wer `wiki` aus der Registry liest (`WikiPath`), sieht den Ort aus `--wiki`, wer die Deklaration liest (`wiki.Root`, `guard.declaredWikiRoot`, die Wiki-Lane), den erkannten — und dort legt `area add` kein Bündel an. **Offen:** ob `--wiki` auch `[layout] wiki` schreiben soll, wo der Ort im Repo liegt | vermerkt im Schlussreview 2026-09-22, nicht entschieden |

## Auflagen an spätere Stufen

### Umstieg: `project/loomux` braucht ein `[index]`, bevor der erste echte `reindex` läuft (S3)

**Ruling 2026-09-22.** `.loomux/config.toml` dieses Repos hat kein `[index]`,
also gilt `**/*.md` (Zeile „Leeres `[index] include`"). Gemessen am
2026-09-22 auf einer Wegwerfkopie von `c3a1594` (`git archive`, eigenes Repo
unter Temp, `ReindexWithOutput` mit Zustand und `XDG_CONFIG_HOME` unter Temp):

- **345 versionierte Dateien geändert**, alle `index.md` unter
  `testdata/cases/`; 170 davon liegen in `-source`-Aufnahmen, 14 in
  `testdata/cases/1a-source/`, das die Policy sperrt.
- **2002 neue Dateien**: 2000 Kataloge `index.md` (1931 der 2002 unter
  `testdata/`), dazu `_identities.tsv` und `graph.json` an der Wurzel.
- 2504 Dokumente im Register.

Die Zahlen der Selbstnutzung (317, 156, 1489) stammen von der Kopie des
Hauptcheckouts auf `01c2a2a` und sind mit dem Fallkorpus seither gewachsen.

**Vor dem ersten echten `loomux reindex` trägt ein Mensch `[index]` in
`.loomux/config.toml` ein** — kein Agent, AGENTS.md. Vorschlag, auf derselben
Wegwerfkopie gemessen (die Tabelle eingesetzt statt der Datei):

```toml
# Only the wiki is knowledge; testdata/cases holds recordings, and a catalog
# written into them would change evidence.
[index]
include = ["docs/wiki/**/*.md"]
```

Ergebnis damit: **keine** versionierte Datei geändert, 32 Dokumente (alle
`*.md` unter `docs/wiki`), vier neue Dateien — `_identities.tsv`,
`graph.json` und die Kataloge `index.md` und `docs/index.md` an der Wurzel
und über dem Bündel (im Bündel selbst schreibt `WriteCatalogs` keinen,
Decision 43).

**Frage an den Menschen, vor dem Umstieg zu entscheiden: werden die vier
versioniert?** Die Abwägung, am Repo gelesen:

- **`_identities.tsv` spricht fürs Versionieren.** Es ist das Identitätsregister
  des Bereichs, nicht bloß ein Nebenprodukt: `reconcile` misst Änderungen
  dagegen, und ein frischer Klon ohne die Datei vergäbe jeder Seite eine neue
  `doc_id`. `docs/wiki/_identities.tsv` ist heute schon versioniert.
- **`graph.json`, `index.md` und `docs/index.md` sind aus den Quellen
  ableitbar.** Versioniert erscheint jeder Lauf, der sie ändert, als Diff;
  `graph.json` wechselt zudem zwischen Referenz und loomux die
  Schlüsselreihenfolge (Zeile „`graph.json`, Schlüsselreihenfolge").

Vorschlag, falls die drei ableitbaren nicht versioniert werden sollen — an
die Wurzel verankert, damit `docs/wiki/index.md`, der Katalog des Bündels,
versioniert bleibt:

```gitignore
# Generated by `loomux reindex` for the area project/loomux.
/index.md
/docs/index.md
/graph.json
```

Sonst werden alle vier mit dem ersten Lauf committet.


Nur **ein** Muster, weil qmd lediglich das erste
`include` als `pattern` bekommt (`collectionSpec`,
`internal/brain/index/reindex.go`); seit der Heilung von S2 deckt es auch
`docs/wiki/index.md` direkt unter dem Bündel.

### Stufe 4: `migrate` muss `merge-events.done.tsv` und `qmd-collections.json` mitnehmen

**Festgestellt am 2026-09-20 in Task 7.** `ReadEvents` liest beide Dateien des
Protokolls nach der Rückfallregel der Stufe — neu zuerst, sonst alt —,
`DropEvent` schreibt die Ablage aber nur ins **neue** Verzeichnis. Auf einer
Maschine, auf der ein Python-`brain reconcile` bereits abgelegt hat, kippt die
Auflösung mit dem **ersten** Drop von loomux: ab da gewinnt die neue,
einzeilige Ablage, und die Einträge der alten zählen nicht mehr mit.

**Die Folge ist Lärm, kein Verlust.** Jeder in Python abgelegte Merge taucht
genau einmal wieder auf, bekommt seinen Fall erneut und wird dann von loomux
abgelegt. Das ist absichtlich die Richtung, in die dieses Paar fällt
(`_dropped_path`, `merge_events.py:203-209`): ein doppelter Fall ist Lärm, den
ein Prüfer wegwirft, ein vergessener Merge ist Wissen, das niemand
zurückbekommt.

**Eine Vereinigung beider Dateien beim Lesen wäre die falsche Heilung.** Sie
bräche die Regel „neu zuerst, sonst alt", die für jedes andere Artefakt dieser
Stufe gilt, und gäbe der Ablage als einziger Datei eine Sonderbehandlung.
`loomux migrate` kopiert `maintenance/merge-events.done.tsv` (und, solange ein
Python-Hook noch schreibt, `maintenance/merge-events.tsv`) mit um; danach ist
der Rückfall leer und die Frage verschwindet.

**Ebenso `qmd-collections.json` (Nachtrag 2026-09-22, S5).** `reindex` liest
die Liste der eigenen qmd-Sammlungen neu zuerst, sonst alt, und schreibt sie
nur neu. Nimmt `loomux migrate` die Datei **nicht** mit und leert danach den
Rückfall, beginnt die Liste leer, und der erste Lauf verweigert jede Sammlung,
die qmds `index.yml` schon führt — S5 von vorn, mit Exit 1. `migrate` kopiert
`qmd-collections.json` darum mit um, und zwar nur, wenn im neuen Verzeichnis
noch keine liegt: eine dort schon geschriebene ist die jüngere.


### Die vier `brain reindex`-Regeln in `[relevance]` laufen nie — keine Auflage

**Festgestellt im Schlussreview 2026-09-22 als Auflage, am selben Tag
widerlegt.** Vier Repos tragen `brain reindex` in ihrer `.ultraloom/config.toml`:

| Datei | Zeile |
|---|---|
| `iam_backend/.ultraloom/config.toml:18` | `"wiki/**" = ["brain reindex"]` |
| `iam_frontend/.ultraloom/config.toml:18` | `"wiki/**" = ["brain reindex"]` |
| `iam_workers/.ultraloom/config.toml:20` | `"wiki/**" = ["brain reindex"]` |
| `space/.ultraloom/config.toml:18` | `"docs/wiki/**" = ["brain reindex"]` |

Alle vier stehen in der Tabelle **`[[relevance]]`**, und die liest **kein
Hook** — das ist Nachtrag #12 der Fusions-Spec („Der Installer schreibt ihn,
aber kein Hook liest ihn, schon in ultraloom nicht"). Nachgemessen am
2026-09-22: in `ultraloom` kommt `relevance` außerhalb von Tests nur in
`cmd/init/run.go` und `internal/answers/answers.go` vor, also im Installer,
der die Tabelle schreibt; `cmd/guard/` und `src/ultraloom/hooks/` kennen sie
nicht. `ulguard post-edit` führt diese Regeln also **nicht** aus, und der alte
Indexer läuft über sie nie. Die Auflage, die das Schlussreview hier setzte,
entfällt: vor dem Umstieg ist nichts umzustellen. `loomux migrate` (Stufe 4)
lässt `[relevance]` ohnehin fallen und sagt es (#12).

### Umstieg: die Ratschläge `brain reindex` und `brain reconcile` ziehen mit um

**Festgestellt im Schlussreview 2026-09-22.** loomux' Meldungen nennen noch die
Python-Befehle: `ReconcileAdvice` (`internal/brain/search/stamp.go:23`),
`internal/brain/status/status.go:61` und die zwölf Meldungen in
`internal/brain/graph/read.go` (Tabelle „Alte Aufrufe" unten). **Sie bleiben
in 3a**, weil die aufgezeichneten Fälle von 1b-1 und 1b-2 ihren Wortlaut
halten.

**Nach dem ersten loomux-Lauf führen sie ins Leere.** `loomux reconcile`
schreibt `maintenance/last-run.txt` ins neue Zustandsverzeichnis, und
`search.ReadLastRun` liest neu zuerst. Ein `brain reconcile`, wie der
Ratschlag es empfiehlt, schreibt nur noch ins alte: der neue Stempel altert
weiter, und die Warnung vor dem veralteten Abgleich verschwindet nie, so oft
man dem Ratschlag folgt. Für `brain reindex` gilt dasselbe überall, wo loomux
den Bestand aus dem neuen Verzeichnis liest — bei den read-only-Bereichen.

**Auflage: mit dem Umstieg werden die Meldungen auf `loomux reconcile` und
`loomux reindex` umgestellt**, und die berührten Fälle von 1b-1 und 1b-2
bekommen die neue Erwartung. Der Kommentar an `ReconcileAdvice` sagt es
seit dem Schlussreview.

### Stufe 3c: das Upkeep in `serve` bekommt den qmd-Konfigurationspfad hereingereicht

**Festgestellt im Schlussreview 2026-09-22** (Zeile „`QmdConfigPath` und
`XDG_CONFIG_HOME`" oben). `serve.Options` sagt zu, dass alles am übergebenen
Zustandsverzeichnis hängt, „never off a fixed path"
(`internal/serve/serve.go:44-45`) — nur so isoliert ein Test seinen `serve`
vom echten. `index.QmdConfigPath` liest `XDG_CONFIG_HOME` dagegen selbst, und
`reindex` ruft es unterhalb jeder Kompositionsstelle
(`internal/brain/index/reindex.go:274`). Heute ruft `serve` kein `reindex`,
also hält die Zusage.

**Auflage: bevor das Upkeep aus 3c `reindex` ruft, bekommt `reindex` den Pfad
als Argument, und `serve` reicht ihn aus seinen Optionen herein.**
`QmdConfigPath` bleibt dann nur an der Kompositionsstelle der CLI. Sonst
schriebe ein `serve` im Test in die qmd-Konfiguration dessen, der die Suite
laufen lässt, sobald der Test `XDG_CONFIG_HOME` nicht selbst setzt — die
Falle, in die diese Suite schon einmal gelaufen ist
(`internal/cli/maintenance_test.go`).

### Stufe 4: `guard` muss die Deklaration eines read-only-Bereichs finden

**Festgestellt im Schlussreview 2026-09-22** (Zeile „Deklaration eines
read-only-Bereichs, gelesen von `guard`" oben). `guard` liest nur
`.loomux/config.toml`, die brain-Leser nehmen bis Stufe 4 auch `.brain.toml`
und `.ultra-brain/config.toml`, und `publish` kopiert den alten Namen
wörtlich ins neue Zustandsverzeichnis. Für read-only-Deklarationen bleibt die
Schranke damit blind.

**Auflage: `loomux migrate` legt die Deklaration jedes read-only-Bereichs
unter `.loomux/config.toml` ab**, bevor der Rückfall in
`ReadAreaManifestUntilStage4` entfällt — sonst verliert brain die Deklaration
im selben Schritt, in dem `guard` sie nie hatte. Danach lesen beide Leser
dieselbe Datei. Wer das umsetzt, prüft, ob `publish` bis dahin schon den
neuen Namen schreiben soll.

### Stufe 4: der abwesende Bereichsordner nach einem erschlagenen Tausch

**Festgestellt am 2026-09-20 in Task 4.** `internal/lock.ReplaceDir` tauscht
den Bestand eines read-only-Bereichs als Ganzes ein: eine Umbenennung legt
den alten Stand nach `<ziel>.loomux-aside`, eine zweite schiebt den neuen
hinein. Zwischen den beiden ist das Ziel **abwesend**. Ein dort erschlagener
Prozess lässt es abwesend liegen.

**Heute ist das harmlos, aber nicht aus eigener Kraft.**
`config.ResolvedAreaDir` fällt auf ultra-brains Zustandsverzeichnis zurück,
solange unter `<state>/areas/<scope>/` nichts liegt — und liefert von dort
den alten Bestand ganz. Der Tresor antwortet also weiter.

**Ab Stufe 4 ist der Rückfall leer**, weil `loomux migrate` den Bestand
umgezogen hat. Dann ist der abwesende Zielordner eines **registrierten**
Bereichs eine fehlende Deklaration, und `privacy.VisibleAreas` gibt beim
ersten fehlenden Manifest für **alle** Bereiche auf
(`internal/brain/privacy/areas.go:42-47`). Ein einziger erschlagener Tausch
macht den ganzen Tresor unbeantwortbar — genau das, was die Auflage von 3a
verhindern soll.

**Die Heilung gibt es, aber sie hat heute einen einzigen Aufrufer.**
`lock.Recover` steht in `internal/brain/index.recoverStock` und wird aus
`indexArea` gerufen, hinter `os.Stat(area.Path)`. Kein Leser ruft es. Ein
erschlagener Tausch bleibt damit liegen, bis jemand **genau diesen Bereich**
erneut erfolgreich indiziert; verschwindet der Checkout des Bereichs, greift
selbst das nicht mehr, weil `indexArea` dann vorher mit „skipping" abbiegt.

**Zu entscheiden in Stufe 4, bevor der Rückfall entfällt — eines von beiden:**

1. **`Recover` in den Lesepfad nehmen.** `privacy.VisibleAreas` (oder die
   Stelle, die den Bereichsordner auflöst) räumt ein Aside auf, bevor sie
   das Fehlen eines Manifests zum Urteil über alle Bereiche macht. Kostet
   jedem Leser einen `stat` je Bereich und macht einen Leser schreibend.
2. **`loomux migrate` die Asides fegen lassen.** Der Umzug läuft ohnehin
   einmal über jeden Bereich und kann jedes `*.loomux-aside` an seinen Platz
   zurückschieben. Billiger, heilt aber nur einmal: ein Tausch, der nach der
   Migration erschlagen wird, bleibt liegen.

Wer das entscheidet, streicht diesen Absatz und trägt die Entscheidung in die
Abweichungstabelle der Stufe ein, die sie trifft.

## Lesarten der Fallsuite (Task 16)

**Wie „unlesbare Quelle" gelesen wurde.** Die Spec nennt als Fall „unlesbare
Quelle". Eine Quelldatei, die der Durchgang nicht lesen kann, lässt sich in
einer kopierten Welt nicht bauen (Rechte überleben das Stagen nicht, und ein
Verzeichnis mit Quellnamen liefert `find_files` gar nicht erst), und die
Referenz fängt einen solchen Lesefehler nicht (`_scan` ruft `content_hash`
ungeschützt; ein `OSError` endet als `error:`-Zeile). Das einzige, was
`ReconcileReport.unreadable` sammelt, sind unlesbare **Fallakten**
(`_standing_case`). Aufgezeichnet ist darum `reconcile/unreadable-case`: eine
kaputte `case.toml` im Prüfzentrum, `unreadable case:` auf stderr, Exit 1, der
neue Fall trotzdem geschrieben.

## Entparkte Fälle (Task 16b, 2026-09-22)

Geparkt waren drei Fälle, weil sie ein echtes Repo in der Welt brauchen und
ein `.git` in `testdata` zum Gitlink wird. Seit 2c baut
`internal/cases/gitworld.go` das Repo beim Stagen aus einem `git.toml`; die
drei sind aufgezeichnet (`brain-mcp.exe` von `loomux-3-source`), übersetzt
und **exakt** — keiner steht in `expected3a`. Die Fix-Runde hat einen vierten
dazugelegt, den Grundlinien-Fall mit abweichendem HEAD. Die Suite hat damit
28 Fälle.

| Fall | Welt | Was er zeigt | Mutationen, die ihn töten |
|---|---|---|---|
| `reconcile/merge-event` | `vault-merge` | Ein Ereignis `{{COMMIT:1}}..{{COMMIT:3}}` über zwei Commits nach der Basis, die Seite `realization: planned`, die Quelle unverändert: es entsteht ein Fall `trigger = "merge"`, `state = "due"`, das Paket trägt die Pfade `src/Alpha.go`, `src/zeta.go` (angelegt in umgekehrter Reihenfolge) und die Betreffe in der Reihenfolge von `git log` (`add the Alpha part`, `add the zeta part`), das Ereignis landet in `merge-events.done.tsv` | `mergeEvidence` gibt immer einen Fehler: `merge-events.done.tsv`, `case.toml` und `package.md` fehlen, stdout 44 statt 84 Bytes. Betreffe umgekehrt (`slices.Reverse(subjects)`): `content mismatch` am `package.md` |
| `reconcile/merge-behind-source-case` | `vault-merge-standing` | Ein Ereignis `{{COMMIT:1}}..{{COMMIT:2}}`, dazu ein stehender Quellfall `a-2000-01-01-5bd8` auf derselben Seite: `0 Fälle`, der Quellfall bleibt unberührt, das Ereignis bleibt im Protokoll, kein `done.tsv` | `absorbable` gibt immer `true`: der Quellfall ist weg, ein Merge-Fall liegt an seiner Stelle, `done.tsv` ist da, stdout 84 statt 44 Bytes |
| `reconcile/changed-source-baseline` | `vault-changed-baseline` | `vault-changed` mit Geschichte: HEAD hält den Stand, den das Register nennt, der Arbeitsbaum den neuen. Das Paket führt `--- notes/source.md (HEAD)` und einen Hunk `@@ -1,3 +1,3 @@`. `reconcile/changed-source` bleibt daneben stehen, für den Weg ohne Repo | `baselineOf` gibt immer `nil`: `content mismatch` am `package.md` |
| `reconcile/changed-source-stale-head` | `vault-changed-stale-head` | Ein Handcommit zwischen zwei Freigaben: HEAD hält einen Stand, den das Register **nicht** nennt. Er ist keine Grundlinie, das Paket führt „ohne verifizierten Vorzustand" und einen Hunk gegen `/dev/null` | Hashvergleich in `baselineOf` weggelassen (HEAD blind geglaubt): `content mismatch` am `package.md` |

Jede Mutation einzeln gesetzt, mit `git diff` gezeigt, die ganze Suite
gespielt und mit `git checkout -- <datei>` zurückgenommen; jede tötet genau
die genannten Fälle und keinen anderen. **Eine überlebt:** `sort.Strings`
über die Pfade in `mergeEvidence` weggelassen. Die beiden Pfade fallen in
umgekehrter Commit-Reihenfolge an, aber `git diff --name-only` gibt sie
schon in Byte-Reihenfolge aus — das ist die Äquivalenz, die unter
„Überlebende Mutanten" steht. Töten ließe er sich nur über ein
`diff.orderFile` in der Konfiguration des Welt-Repos, und dafür hat
`git.toml` keinen Schlüssel.

**Das Format dafür.** Die Bereiche der Stufe 3a liegen unter
`{{WORLD}}/repo-a`, während die Welt selbst das Zustandsverzeichnis ist.
`git.toml` hat dafür den Schlüssel `dir` bekommen: das Repo entsteht in diesem
Unterverzeichnis, und jeder Pfad der Erklärung (`paths`, `files`, `worktree`)
ist relativ zu ihm; `{{COMMIT:n}}` wird weiter in der **ganzen** Welt
ersetzt, weil das Ereignisprotokoll unter `maintenance/` liegt. Ein `dir`,
das die Welt verlässt, wird abgelehnt. `InfraPath` erkennt `.git` und
`.origin.git` jetzt in **jedem** Pfadsegment, sonst landete `repo-a/.git` im
Baumvergleich und im Korpus. Ohne `dir` baut `BuildGitWorld` wie bisher in
der Wurzel; `TestCases2c` bleibt grün.

**SHAs in `world_after`.** Der Rekorder ersetzt SHAs nicht zurück durch
`{{COMMIT:n}}`; `merge-events.tsv` und `merge-events.done.tsv` im
`world_after` tragen sie ausgeschrieben, ebenso 20 Dateien der 2c-Fälle.
Das trägt, weil `BuildGitWorld` sie deterministisch baut: feste Identität und
Daten, eigene leere Konfiguration und **festes Objektformat**
(`--object-format=sha1` an beiden `init`, Fix-Runde 1). Ohne das nahm git das
Format aus `GIT_DEFAULT_HASH`, das auf keiner Streichliste steht; gemessen
starben mit `GIT_DEFAULT_HASH=sha256` `merge-event` und
`merge-behind-source-case`, und git 3.0 kündigt SHA-256 als Vorgabe an. Mit
der Festlegung sind 2c und 3a unter `GIT_DEFAULT_HASH=sha256` grün
(`TestBuildGitWorldPinsTheObjectFormat`).

**Dieselbe git-Umgebung beim Aufzeichnen und Abspielen.** Die Referenz rief
git bisher mit der Konfiguration des Nutzers, und die dieses Rechners ist
nicht leer: die Systemdatei setzt `core.autocrlf=true` und die LFS-Filter,
`~/.gitconfig` Identität, `core.longpaths` und LFS. Eine Einstellung wie
`diff.orderFile` oder `core.quotepath` hätte die Aufzeichnung geformt.
**`GIT_CONFIG_GLOBAL` und `GIT_CONFIG_SYSTEM` helfen dagegen nicht:** beide
stehen auf der Streichliste der Referenz (`_INHERITED`, `vcs.py:33-62`) und
auf der von `gitenv.Location`, git sieht sie auf keiner Seite. Was beide
Listen durchlassen, ist `GIT_CONFIG_NOSYSTEM` und `HOME`. `cases.GitEnv`
setzt darum `GIT_CONFIG_NOSYSTEM=1` und `HOME={{WORLD}}/.no-git-home`, ein
Verzeichnis, das niemand anlegt; die dritte Stelle, `XDG_CONFIG_HOME/git`,
zeigt in 3a auf beiden Seiten schon in die Welt. Der Rekorder (`recordEnv`),
`TestCases3a` und `TestCases2c` setzen dieselbe Liste; 2c leert dazu
`XDG_CONFIG_HOME`, weil seine Welten keines nennen. Gemessen: `git config
--list --show-origin` unter dieser Umgebung gibt nichts aus
(`TestGitEnvLeavesGitWithoutAConfigurationFile`). **Auf Windows** erreicht
`HOME` nur git: Pythons `Path.home()` und loomux' `os.UserHomeDir` lesen dort
`USERPROFILE`. Auf POSIX lesen beide `HOME`; dort lenkte `GitEnv` auch die
Heimatverzeichnis-Zugriffe von loomux und der Referenz um — geprüft ist das
nicht.

## Umstieg (2026-09-22) — Bedingung 5 erfüllt

Nach dem Merge von 3a (`1253c42`), mit dem Binary des Hauptcheckouts auf
diesem Stand, gegen die echte Registry mit elf Bereichen:

- **`[index]` vorher** (Auflage S3), von Hand eingetragen: `include =
  ["docs/wiki/**/*.md"]` und `unsearched = ["docs/**/**/index.md"]`.
  Vorgeschlagen war `docs/wiki/**/index.md`; heute wirkt beides gleich, weil
  das Register nur Seiten unter `docs/wiki` führt. Wird `include` erweitert,
  blendet das eingetragene Muster jeden Katalog unter `docs/` aus.
- **`loomux reindex`** (die Ausgabe blieb im Terminal des Nutzers, der
  Exit-Code ist nicht mitgeschrieben): im loomux-Repo **keine** versionierte
  Datei geändert, neu `/_identities.tsv` (32 Einträge, die Seiten unter
  `docs/wiki`), `/graph.json`, `/index.md`, `/docs/index.md`. Versioniert wird
  nur das Register; die drei ableitbaren stehen in `.gitignore`, das
  Register mit nur der Kopfzeile unter `docs/wiki/` ist entfernt. Der
  Auffangdurchgang öffnete keinen Fall (in `brain-knowledge/95 Prüfzentrum/`
  ist nichts neuer als der Lauf). `%LOCALAPPDATA%\loomux` hält jetzt `areas/`
  (drei schreibgeschützte Bereiche), `maintenance/` (zwölf Einträge) und
  `qmd-collections.json` (zwölf Sammlungen).
- **Nebenwirkung in fremden Repos,** anders als auf der Kopie (Abschnitt
  darunter): `brain-knowledge` hat drei versionierte Dateien geändert
  (`_identities.tsv`, `graph.json`, `index.md`, +354/−52), `ultraloom` drei
  neue unversionierte (`_identities.tsv`, `docs/index.md`, `graph.json`).
  `space`, `ultra-brain`, `iam_wiki` und `ecoflow` trugen keine Datei mit dem
  Zeitstempel des Laufs. Zurückgesetzt wurde nichts; das entscheidet der
  Nutzer in beiden Repos.
- **`loomux brain status`**: für `project/loomux` keine Divergenz und keine
  unaufgelösten Links, nur drei Hinweise „same content hash under 2 paths“ —
  die Kataloge `sources`, `topics`, `entities` sind byte-gleich mit denen in
  `ultra-brain`.
- **`loomux embed`**: `embedded 11 area(s)`, Exit 0. `loomux brain search
  Schreibschranke --scope project/loomux` liefert Seiten aus `docs/wiki`.
- **`loomux area add`** hatte nichts zu tun: `project/loomux` war schon
  registriert.

## Selbstnutzung (Task 18, 2026-09-22) — auf einer Kopie

**Stand vor dem Umstieg darüber: Bedingung 5 der Stufe war nicht erfüllt, und das war entschieden.** „Das
loomux-Repo fährt `loomux reconcile` und `loomux reindex` auf sich selbst"
gibt es so nicht: beide Befehle haben keinen Bereichsfilter und laufen über
die **ganze** Registry, auf dieser Maschine elf Bereiche. Ein echter Lauf
schriebe in acht Repos (darunter `ultra-brain`, das eingefrorene Quellrepo),
legte die Artefakte der drei schreibgeschützten Bereiche unter
`%LOCALAPPDATA%\loomux\areas\` ab und könnte Fälle in
`brain-knowledge/95 Prüfzentrum/` anlegen, das unversioniert ist und keinen
Remote hat. Der Nutzer hat entschieden: alles gegen eine Kopie; Bedingung 5
bleibt offen bis zum bewussten Umstieg nach dem Merge. Die alten Aufrufe
stehen unten, ersetzt ist keiner.

**Die Kopie.** Die fünf Repos der acht schreibbaren Bereiche (`loomux`,
`brain-knowledge` für `knowledge`, beide `engineering/*` und `hub`,
`ultra-brain`, `ecoflow`, `ultraloom`) samt `.git` und unversionierten
Dateien nach `%TEMP%`, Änderungszeiten auf die Nanosekunde erhalten; ohne
`node_modules`, `.venv`, Python-Caches, `loomux/bin` und die Arbeitsbäume
außer `brain-knowledge/.worktrees/okf-specs`, aus dem das Register von
`knowledge` 62 Quellen liest (dessen `gitdir` auf die Kopie umgebogen). Kein
ausgeschlossener Pfad steht in einem Register der kopierten Bereiche. Die drei
schreibgeschützten Bereiche blieben im Original und wurden nur gelesen. Die
Registry-Kopie zeigt auf die Kopien; qmd lief mit `QMD_CONFIG_DIR`,
`INDEX_PATH` und `XDG_CACHE_HOME` auf Temp (belegt mit `qmd status`, das den
Temp-Index nannte). Zwei frische Kopien: A für loomux (`01c2a2a`), B für die
Referenz (`brain-mcp.exe` aus `ultra-brain/.venv`, `3cc72d2`), B mit
derselben Registry aus elf Bereichen.

| | loomux (A) | Referenz (B) |
|---|---|---|
| `reconcile`, erster Lauf | `915 Quellen geprüft, 915 davon gehasht`, `0 Fälle`, Exit 0 | `901 Quellen geprüft, 71 davon gehasht`, `0 Fälle`, Exit 0 |
| `reindex` | Exit 0; `updated qmd collections:` elf Sammlungen | Exit 0; `skipping project/loomux: …\.brain.toml does not exist`, dann zehn Sammlungen |
| `reconcile` nach dem `reindex` | `3064 Quellen geprüft, 2164 davon gehasht`, `0 Fälle` | `910 Quellen geprüft, 24 davon gehasht`, `0 Fälle` |
| Dokumente in qmd | 2559, davon 2066 in `project-loomux` | 493 |

**Kein Fall ist aufgegangen**, auf keiner Seite. Die zehn gemeinsamen
Sammlungen hält qmd auf beiden Seiten mit derselben Dokumentzahl (3, 3, 3, 44,
6, 94, 27, 215, 83, 15). `index.yml` ist bis auf die Pfade der Kopien und die
Sammlung `project-loomux` gleich. Die Referenz überspringt `project/loomux`,
weil sie nur die alten Manifestnamen kennt (die Zeile „`area add`, altes
`.brain.toml`" oben ist die Kehrseite).

**Außerhalb von `project/loomux` hat loomux die vier anderen Repos
byte-gleich mit dem heutigen Stand der Platte gelassen, die Referenz nicht.**
Die Artefakte auf der Platte entsprechen also loomux' Verhalten, nicht dem der
Referenz; auf dem `PATH` liegt `~/go/bin/brain`,
`ultra-brain (go core) v0.2.0-go (3cc72d2)`, aus dessen `pkg/index` der
Indexport (`3012bed`) stammt, und naheliegend ist, dass sie von dort kommen. Die Befunde unten sind darum ererbt, und die
Fallsuite sieht keinen davon.

### Befunde der Selbstnutzung

**Ruling 2026-09-22.** S1, S2, S5 (`qmd-collections.json`) und die Wurzel von
S6 sind geheilt, S4, S5 (`stats.tsv`) und die verschachtelten Junctions von S6
stehen als Abweichungen in der Tabelle oben, S3 ist eine Auflage an den
Umstieg (unten). Die Tabelle hier hält fest, was die Selbstnutzung gemessen
hat, vor der Heilung.

| # | Befund | Referenz (B) | loomux (A) | Art |
|---|---|---|---|---|
| S1 | `globToRegex` (`internal/brain/index/walk.go:90-118`) übersetzt Byte für Byte: `regexp.QuoteMeta(string(p[0]))` macht aus jedem Byte eines Mehrbytezeichens ein eigenes Zeichen (aus `ü` = `C3 BC` wird `Ã¼`). Ein Nicht-ASCII-Literal in `include`, `exclude` oder `[layout] review` trifft darum nie | `review = "95 Prüfzentrum"` von `knowledge` nimmt das Prüfzentrum aus der Begehung | Das Prüfzentrum wird begangen: `reconcile` prüft 14 `package.md` mehr (915 gegen 901), `reindex` behält sie im Register, der Katalog nennt `95 Prüfzentrum/`, der Graph hat 14 Knoten mehr (121 gegen 107). Mit einer Sonde gegen `FindFiles` bestätigt: `95 Prüfzentrum` schließt nichts aus, `Review` schon. qmd ist nicht betroffen (es bekommt die Ignore-Liste und globt selbst) | Bug, Datenschutz: die Pakete tragen Quelldiffs, auch aus `local_only`-Bereichen (`walk.py:96-103`) |
| S2 | Dieselbe Funktion übersetzt `/**` als `/.*`: `docs/**/*.md` wird `^docs/.*/[^/]*\.md$` und verlangt ein Unterverzeichnis. `full_match` lässt `**` auch null Teile decken | `ultraloom` (`include = ["docs/**/*.md", "README*.md"]`) registriert `docs/benchmarks.md`, `docs/benchmarks.de.md`, `docs/hooks.md`, `docs/hooks.de.md` | Die vier fehlen in Register, Katalog und Graph (73 gegen 77 Knoten) | Bug. Stufe 1b-1 hat für `privacy` schon einen treuen Übersetzer gebaut und den alten `globToRegex` dort gestrichen (`plans/2026-09-15-loomux-stufe-1b-1.md`); der Indexport hat eine zweite Kopie zurückgebracht |
| S3 | `project/loomux` hat kein `[index]`, also gilt `**/*.md` (Ruling B1, Zeile „Leeres `[index] include`") | überspringt den Bereich | `reindex` ändert **317 versionierte Dateien** — alle `index.md` unter `testdata/cases/`, 156 davon in den `-source`-Aufzeichnungen, die die Policy als Beweismittel sperrt — und legt **1489 neue** an (1487 `index.md`, `_identities.tsv`, `graph.json`; 1422 unter `testdata/`); qmd bekommt 2066 Dokumente | Vor dem Umstieg braucht `.loomux/config.toml` ein `[index]`, das die Spec meint („über `docs/wiki`"), mindestens ohne `testdata/**`. Die Datei schreibt ein Mensch (AGENTS.md) |
| S4 | Der Graph folgt Anhang B der ub-Migrationsspec: `cleanTarget` schneidet `?`/`#` vor dem Auflösen ab, und jedes URI-Schema zählt als `external` (`internal/brain/graph/render.go:15-21`, `:59-62`) | Nur `http://`, `https://`, `mailto:` sind extern; `README.de.md#policy` ist `unknown_target` | `iam-wiki`: 322 gegen 266 aufgelöste Links, 282 gegen 266 Kanten; `ultra-brain`: 100 gegen 4 `external`; `space`: 334 gegen 332 Kanten; in `ultraloom` zwei Kanten mehr, die S2 wieder überdeckt (17 gegen 19). In `iam-wiki`, `space` und `ultra-brain` bleibt die Summe der Links gleich | Bewusste Abweichung (Anhang B nennt 56 fehlende Kanten; in `iam-wiki` sind es hier 56 aufgelöste Links mehr), aber in keiner Paritätsakte eingetragen |
| S5 | `stats.tsv` (`scan.go:306`) und `qmd-collections.json` (`reindex.go:263`) liest loomux nur aus dem neuen Zustandsverzeichnis, ohne die Rückfallregel der Stufe | liest beide aus `%LOCALAPPDATA%\brain` | `stats.tsv`: der erste `reconcile` hasht jede Quelle (915 von 915) — ein Kostenpunkt. `qmd-collections.json`: die Liste beginnt leer, und jede Sammlung, die qmds `index.yml` schon führt, gilt als fremd. In der Kopie zeigte es sich nicht (qmd lief dort mit eigener Konfiguration unter Temp, `reindex` meldete elf aktualisierte Sammlungen); auf dieser Maschine führt sie alle zehn, und der **erste echte `reindex` verweigert alle zehn und endet mit Exit 1** (mit einer Sonde bestätigt: eine Sammlung in `index.yml`, aber nicht in der eigenen Liste, ergibt `refused=[knowledge]`). Die frühere Fassung dieser Zeile sagte „Schaden null" — das war falsch | Ruling 2026-09-22: `qmd-collections.json` mit Rückfall (geheilt), `stats.tsv` ohne (Tabelle oben) |
| S6 | `filepath.WalkDir` steigt nicht in eine NTFS-Junction ab (sie meldet sich als `ModeIrregular`), `rglob` schon | `space/.agents/skills` ist eine Junction auf `.claude/skills`; die Referenz registriert drei `SKILL.md` darüber, an `**/.claude/**` vorbei | loomux registriert sie nicht (qmd auch nicht: 215 auf beiden Seiten) | Abweichung ohne Zeile; loomux' Lesart vermeidet Zwillinge einer ausgeschlossenen Ablage |

### Alte Aufrufe, aufgelistet, nicht ersetzt

Gesucht nach `brain reindex`, `brain reconcile`, `uv run brain` und
`brain-mcp reindex|reconcile` im ganzen Worktree. **In Hooks und Konfiguration
steht keiner:** `.claude/`, `.githooks/`, `ci/`, `.loomux/`, `.github/` sind
leer. Was sich auf den Umstieg bezieht:

| Ort | Stellen | Was dort steht |
|---|---:|---|
| `README.md:178-180`, `README.de.md:179-181` | 1 + 1 | `loomux brain reconcile` und `loomux brain embed` — Namen, die es nicht gibt (`loomux brain reconcile` endet mit Exit 2, `invalid choice`); gemeint sind `loomux reconcile` und `loomux embed`. `reindex` und `area add` fehlen |
| `docs/en/cli-reference.md:303,331,336,343`, `docs/de/cli-reference.md:308,336,341,348` | 4 + 4 | Die Ratschläge nennen `brain reindex`/`brain reconcile` „bis Stufe 3 sie umschreibt", ein Abschnitt `loomux brain reconcile`; die vier Befehle von 3a haben keinen Abschnitt |
| `internal/brain/graph/read.go:23,51,64,97,102,105,108,114,122,127,131,136` | 12 | Meldungen „run `brain reindex`" |
| `internal/brain/status/status.go:61` | 1 | „never indexed; run `brain reindex`" |
| `internal/brain/search/stamp.go:23` | 1 | ``ReconcileAdvice = "run `brain reconcile`"`` |
| Tests, die diese Meldungen festhalten | 15 + 4 + 10 + 5 + 2 + 1 | `graph/read_test.go`, `graph/read_python_test.go`, `status/status_test.go`, `cli/brain_test.go`, `search/stamp_test.go`, `search/search_test.go` |

Nicht umzustellen: `internal/brain/maintenance/events.go:127` (ein Kommentar
über die Python-Seite), `docs/wiki/syntheses/gegenpruefung-vor-jeder-designempfehlung.md:33`
(Geschichte), die Rekorder-Argumente in `cli/devmcp_test.go`, `cli/dev_test.go`
und `dev/importcases/importcases_test.go`, und die Pläne, Specs und Berichte
unter `docs/.superpowers/` und `.superpowers/`.

**Nachtrag Schlussreview (2026-09-22): außerhalb dieses Repos.** Gesucht nach
`brain reindex`, `brain reconcile` und `uv run brain` in den Repos aller elf
Bereiche der Registry `%LOCALAPPDATA%\loomux\registry.toml` — acht Wurzeln:
`loomux`, `brain-knowledge` (für `knowledge`, beide `engineering/*` und
`hub`), `ultra-brain`, `space`, `iam_wiki`, `#Obsidian/AI`, `ecoflow`,
`ultraloom` — und in `iam_backend`, `iam_frontend`, `iam_workers`; je in
`.ultraloom/`, `.claude/settings.json`, `.claude/settings.local.json`,
`.agents/hooks.json` und dem Hook-Verzeichnis, das git für das Repo nennt
(`git rev-parse --git-path hooks`, also `core.hooksPath` oder `.git/hooks`).
Nur gelesen. **In `.claude/`, `.agents/` und den git-Hooks steht in keinem
der elf Repos ein Treffer**; die Hooks von `ultra-brain` rufen nur
`install_brain`, `#Obsidian/AI` ist kein git-Repo.

| Ort | Stellen | Was dort steht |
|---|---:|---|
| `iam_backend/.ultraloom/config.toml:18`, `iam_frontend/.ultraloom/config.toml:18`, `iam_workers/.ultraloom/config.toml:20` | 3 | `"wiki/**" = ["brain reindex"]` in `[relevance]` — liest kein Hook (#12), läuft also nie. Abschnitt „Die vier `brain reindex`-Regeln in `[relevance]` laufen nie" oben |
| `space/.ultraloom/config.toml:18` | 1 | `"docs/wiki/**" = ["brain reindex"]`, ebenso unter `docs/wiki/` |
| `iam_backend/.ultraloom/answers.toml:26`, `iam_frontend/.ultraloom/answers.toml:24`, `iam_workers/.ultraloom/answers.toml:25`, `space/.ultraloom/answers.toml:24`, `ultraloom/.ultraloom/answers.toml:25` | 5 | Dieselbe Regel in den Antworten, aus denen `ultraloom init` die Konfiguration schreibt; `space` mit `docs/wiki/**`, die übrigen mit `wiki/**`. In `ultraloom` steht sie **nur** hier, seine `config.toml` hat sie nicht |
| `ultraloom/internal/answers/answers.go:100` | 1 | `"wiki/**": {"brain reindex"}` — die Vorgabe des Installers; jedes neue `ultraloom init` schreibt die Regel wieder |
| `ultraloom/cmd/guard/post_edit.go:430` | 1 | `uv run brain lint` — die eingebaute Wiki-Lane von `ulguard post-edit`. Liest eine Datei und schreibt keinen Zustand; Treffer des Suchmusters `uv run brain`, nicht Teil des Umstiegs von 3a |

Nicht einzeln aufgeführt: die mitgelieferten Quellkopien von `ultraloom`
unter `.ultraloom/vendor/ultraloom/` in `space`, `iam_frontend` und
`iam_workers` (dort samt deren Arbeitsbäumen). Sie tragen dieselbe
`answers.go`, `post_edit.go` und `answers.toml` wie `ultraloom` selbst — Quelltext,
keine Konfiguration, die ein Hook liest.

## Überlebende Mutanten

**Die Runde mit `loomux dev mutants` (Task 19, 2026-09-22).** Die Zeilen bis
`RenderPackage` stammen aus den Handmutanten der Tasks 10 bis 12; alles darunter
aus der Werkzeugrunde über die beiden Pakete der Bedingung 3, gefahren mit
`LOOMUX_STATE_DIR` und `LOOMUX_LEGACY_BRAIN_DIR` auf Temp.

| Paket | erzeugt | nicht übersetzbar | erste Runde: getötet / überlebt | nach den neuen Tests: getötet / stehen |
|---|---:|---:|---:|---:|
| `internal/brain/maintenance` | 834 | 163 | 566 / 62 (siehe unten) | 640 / 31 |
| `internal/config` | 393 | 117 | 268 / 8 | 273 / 3 |

Die erste Runde über `maintenance` ist verunreinigt und wird nur als Suche
gelesen: während sie lief, stand eine Testdatei rund eine Minute lang
unübersetzbar im Baum, 43 Mutanten fielen dadurch fälschlich unter „nicht
übersetzbar“ (206 statt 163), und drei Überlebende zeigten sich erst in der
zweiten, sauberen Runde. Getötet wurden 34 Mutanten in `maintenance` (33 der ersten Runde und einer
der zweiten) und 5 in
`config` durch neue oder geschärfte Tests; der letzte (`candidates`,
`reconcile.go:664`, ohne `page.Realization != nil`) ist nach der zweiten Runde
einzeln per `-overlay` geprüft: `TestAMergePassesOverAPageWithoutARealization`
endet unter ihm in einer nil-Dereferenz. Jeder stehende Mutant hat unten eine
Zeile; wo „äquivalent“ steht, ist es begründet **und** mit einem
Differenzorakel oder einer Wegwerfwelt nachgeprüft.

| Paket | Mutant | Entscheidung |
|---|---|---|
| `internal/brain/maintenance` | `sourceStates`: die `slices.SortFunc` nach `DocID` entfernt | **Äquivalent, stehengelassen.** `sourceCases` läuft `sortedDocIDs(changed)` ab und hängt die Quellen in genau dieser Reihenfolge an, also kommen sie bei `landCase` schon sortiert an. Dieselbe Redundanz hat die Referenz: `_cases:469` läuft `sorted(changed.items())` und `_land_case:731` sortiert trotzdem noch einmal. Der Aufruf bleibt stehen, weil er die Zusage von `landCase` ist und nicht die seines ersten Aufrufers; Task 12 hat den zweiten danebengesetzt, und der reicht gar keine Quelle herein. **Die Zwillingszeile in `segmentsOf` ist es ausdrücklich nicht:** `testdata/case-package.golden.md` reicht die Quellen verkehrt herum hinein, und ohne die Sortierung dort kommen `D` und `Q` andersherum nummeriert heraus — der Mutant stirbt |
| `internal/brain/maintenance` | `mergeEvidence`: die `sort.Strings` über die geänderten Pfade entfernt | **Äquivalent, stehengelassen.** `git diff --name-only` gibt die Namen schon in Byte-Reihenfolge aus; gemessen am 2026-09-20 an den beiden Namen des Paritätsgoldens (`Zeta.txt` vor `alpha.txt`, also `0x5A` vor `0x61` und nicht case-insensitiv) und an jedem Fall dieser Suite. Genau dieselbe Redundanz trägt die Referenz: `_merge_evidence:600` schreibt `sorted(paths)` über dieselbe Ausgabe. Der Aufruf bleibt stehen, und der Grund ist schärfer als „nichts sagt die Reihenfolge zu": `diff.orderFile` in der Benutzer- oder Systemkonfiguration ordnet die Ausgabe von `git diff` um, und `vcs.run` streift nur die `GIT_*`-Umgebung ab, nicht die Konfigurationsdateien. Ohne die Sortierung hinge die Reihenfolge der Belege also an der git-Konfiguration des Rechners, auf dem der Durchgang lief — und ein Paket, das sich je Rechner anders liest, wäre über zwei Läufe nicht mehr vergleichbar |
| `internal/brain/maintenance` | `findLongestMatch`: `if j >= bhi { break }` → `continue` | **Äquivalent, stehengelassen.** `b2j` hält die Vorkommen einer Zeile in aufsteigender Reihenfolge, also ist nach dem ersten `j >= bhi` jedes weitere ebenfalls draußen; `continue` überspringt sie einzeln und kommt zum selben Ergebnis, nur langsamer. `break` steht da, weil `difflib` es schreibt |
| `internal/brain/maintenance` | `Scan`: `sortedRegister` ohne `sort.Strings` | **Nicht beobachtet, stehengelassen — nicht äquivalent.** Die erste Fassung dieser Zeile sagte „äquivalent"; das ist in der Fix-Runde widerlegt. `identity.ReadIdentities` schlüsselt nach `pfad` und weist einen doppelten `doc_id` **nicht** zurück (`identity.go:124`), also ist `changed[entry.DocID]` last-writer-wins und über ein handgeschriebenes Register beobachtbar — ebenso die Reihenfolge, in der zwei gleichzeitig verschwundene Quellen gemeldet würden. Kein Fall dieser Suite sieht es, dreimal gemessen, alle 149 grün. Nicht getötet, weil der Fall mit zwei Quellen unter einem `doc_id` den Mutanten nur mit Wahrscheinlichkeit ½ fängt: ein flatternder Test wäre hier der schlechtere Tausch als ein ehrlich etikettierter Überlebender. `sort.Strings` steht da, weil `_scan:378` `sorted(identities.items())` schreibt |
| `internal/brain/maintenance` | `RenderPackage`: `strings.TrimRight(…, "\n")` → `strings.TrimSpace(…)` | **Äquivalent, stehengelassen.** Das letzte verbundene Element ist immer `""`, davor ein Zaun aus Backticks oder `---`; am Ende steht also genau ein Zeilenumbruch, und am Anfang `---`. Über keine erreichbare Eingabe unterscheiden sich die beiden. `TrimRight` steht trotzdem da, weil `package.py:163` `rstrip("\n")` schreibt: träte je ein Segmentkörper ans Ende, der auf Leerzeichen endet, wären sie nicht mehr dasselbe |
| `internal/brain/maintenance` | `matchingBlocks`, `diff.go:149` und `:152`: die beiden Bedingungen, unter denen ein Teilkasten links oder rechts des Blocks in die Schlange kommt — je `true`, je ein Operand allein, je `<` → `<=` (zehn Mutanten) | **Äquivalent, stehengelassen.** Jeder zusätzlich eingereihte Kasten hat eine leere a- oder b-Spanne, und über einer leeren Spanne antwortet `findLongestMatch` Größe 0: bei leerem a läuft die äußere Schleife nicht und beide Verlängerungen scheitern an `besti > alo` bzw. `besti+bestsize < ahi`; bei leerem b bricht die innere Schleife an `j >= bhi` ab und die Verlängerungen scheitern an `bestj`. Ein Block der Größe 0 wird verworfen. Die Bedingungen sparen Arbeit, sie entscheiden nichts. Nachgerechnet mit einem Differenzorakel: `hunksOf` über 4000 Zufallspaare (Alphabete von 2 bis 7 Zeilen, Längen bis 329, ein Viertel davon über der `autojunk`-Schwelle von 200), aufgezeichnet mit dem echten `diff.go` und unter jedem Mutanten per `-overlay` wiederholt — alle zehn byte-gleich |
| `internal/brain/maintenance` | `matchingBlocks`, `diff.go:165`: das Zusammenlegen benachbarter Blöcke → `false` | **Äquivalent, stehengelassen.** Benachbarte Blöcke entstehen hier nicht: jeder gefundene Block ist in seinem Kasten nach beiden Seiten maximal verlängert (die beiden Schleifen am Ende von `findLongestMatch`), und ein Block im linken Teilkasten, der genau an der Ecke des Elternblocks endete, hätte dessen Rückwärtsverlängerung weitergetragen; rechts ebenso, und induktiv für jede Tiefe. Die eine Stelle, an der die Bedingung doch zuschlägt, ist der Nullblock am Ursprung vor dem ersten Block bei `(0, 0)`, und dort liefern beide Formen denselben Lauf. `difflib` braucht das Zusammenlegen für `isjunk`, das hier `None` ist. Orakel wie oben: byte-gleich |
| `internal/brain/maintenance` | `matchingBlocks`, `diff.go:169` und `:174`: `if run.size != 0` → `true`, in der Schleife und danach | **Äquivalent, stehengelassen.** Einziger Fall, in dem `run` dort leer ist: vor dem ersten Block, wenn der nicht bei `(0, 0)` beginnt. Der Mutant hängt dann einen Block der Größe 0 bei `(0, 0)` vorn an, und `opcodes` macht daraus nichts: weder `i < 0` noch `j < 0`, und Größe 0 gibt kein `equal`. Orakel: byte-gleich |
| `internal/brain/maintenance` | `opcodes`, `diff.go:188`: `i < block.ai` → `<=` und `j < block.bj` → `<=` | **Äquivalent, stehengelassen.** Aus `insert` bzw. `delete` wird `replace` mit einer leeren Seite. `opcodes` hat nur einen Leser, `unifiedDiff`, und der druckt für `replace` die `-`-Zeilen aus `a[i1:i2]` und die `+`-Zeilen aus `b[j1:j2]` — die leere Seite druckt nichts, also dieselben Zeilen wie `insert`/`delete`. `groupedOpcodes` fragt nur nach `equal`. Orakel: byte-gleich |
| `internal/brain/maintenance` | `groupedOpcodes`, `diff.go:241`: `len(group) > 0` weggelassen und → `>= 0` | **Äquivalent, stehengelassen.** Nach der Schleife ist `group` nie leer: die Schleife hängt jeden Opcode an, und `codes` ist nie leer (bei zwei leeren Seiten setzt die Funktion den einen erfundenen `equal`-Opcode, wie Python). Orakel: byte-gleich |
| `internal/brain/maintenance` | `unifiedDiff`, `diff.go:266` und `:271`: die Tag-Prüfung vor den `-`- bzw. `+`-Zeilen → `true` | **Äquivalent, stehengelassen.** `equal` ist davor schon mit `continue` weg; übrig bleibt je ein Tag mit leerer Spanne auf der gefragten Seite (`insert` hat `i1 == i2`, `delete` hat `j1 == j2`), und über eine leere Spanne druckt die Schleife nichts. Orakel: byte-gleich |
| `internal/brain/maintenance` | `splitAtHunkHeads`, `diff.go:363`: `at < len(text)` → `<=` | **Äquivalent, stehengelassen.** Bei `at == len(text)` ist `text[at:]` leer: kein `@@ `, `IndexByte` antwortet `-1`, die Schleife bricht ab. Kein Zugriff über das Ende hinaus. Orakel: byte-gleich. Die drei Nachbarn (`newline < 0` → `<= 0`, `last < len(text)` → `true`/`<=`) waren über `_hunks` ebenso unerreichbar, sind aber beobachtbar, sobald die Funktion das tut, wofür sie steht — `re.split(r"^(?=@@ )", …)` —, und sterben jetzt an `TestSplitAtHunkHeadsAsTheRegexpSplits` |
| `internal/brain/maintenance` | `decode`, `scan.go:227`: der schnelle Weg `if utf8.Valid(folded)` → `false` | **Äquivalent, stehengelassen.** Über gültigem UTF-8 antwortet `DecodeRune` nie `RuneError` mit Größe 1, also schreibt der langsame Weg jedes Zeichen unverändert. Nachgerechnet mit 200 000 Zufallsfolgen aus 1 bis 7 Bytes über einem Vorrat, der jede Bereichsgrenze der Tabelle 3-7 trifft: byte-gleich |
| `internal/brain/maintenance` | `decode`, `scan.go:232`: `\|\| size > 1` weggelassen | **Äquivalent, stehengelassen.** Getroffen wird nur ein wörtliches U+FFFD (`EF BF BD`) in sonst ungültigem Text. Es geht dann durch den Ersatzzweig, der U+FFFD schreibt, und `maximalSubpart` antwortet für `EF BF BD` 3 — dieselben drei Bytes. Orakel: byte-gleich |
| `internal/brain/maintenance` | `maximalSubpart`, `scan.go:261`: `first >= 0xC2` → `>` und `first <= 0xDF` → `<` | **Äquivalent, stehengelassen.** Für eine Zweibytefolge antwortet die Funktion immer 1, wenn sie überhaupt gefragt wird: eine vollständige gültige Folge hat `DecodeRune` schon genommen, also ist das zweite Byte fehlend oder außerhalb von `80..BF`. Fällt `C2` oder `DF` in den `default`-Zweig, antwortet der ebenfalls 1. Orakel: byte-gleich. Die entsprechenden Grenzen der Drei- und Vierbyteführer sind es **nicht** — dort ist eine abgeschnittene Folge zwei oder drei Bytes lang —, sie sterben jetzt an neun neuen Zeilen in `TestDecodeReplacesEachMaximalSubpart`, gegen CPython 3.14.7 geprüft |
| `internal/brain/maintenance` | `maximalSubpart`, `scan.go:286`: `i < total` → `<=` | **Äquivalent, stehengelassen.** Erreicht die Schleife `i == total`, waren alle Folgebytes gültig und das zweite im Bereich seines Führers — eine vollständige gültige Folge, die `decode` nie hereinreicht. Direkt gefragt (`TestMaximalSubpartMeasuresAWholeSequence`) antwortet der Mutant bei `i == total` entweder über `len(raw) <= i` oder nach der Schleife mit `total`, also derselben Zahl. Orakel: byte-gleich |
| `internal/brain/maintenance` | `parseStamp`, `scan.go:356`: `\|\| field[i] > '9'` weggelassen | **Äquivalent, stehengelassen.** Was über `'9'` liegt — Buchstaben, Nicht-ASCII —, lehnt `strconv.ParseInt` zur Basis 10 ab; unter `'0'` fängt weiter die Schleife (`+`, `-`, Leerzeichen, die `ParseInt` sonst nähme). Orakel über dieselben 200 000 Folgen, darin `a`, `+`, `-`, Leerzeichen und die hohen Bytes: gleich |
| `internal/brain/maintenance` | `readLines`, `events.go:191`: `if line != ""` → `true` | **Äquivalent, stehengelassen.** `readLines` hat zwei Leser: `ReadEvents` gibt die Zeile an `parseEvent`, das fünf Felder verlangt, `dropped` verlangt drei; eine leere Zeile hat eines und fällt bei beiden heraus. Der Filter steht da, weil `_lines` ihn schreibt |
| `internal/brain/maintenance` | `FenceFor`, `package.go:90`: `run > longest` → `>=` | **Äquivalent, stehengelassen.** Bei Gleichheit weist der Mutant `longest` den Wert zu, den es schon hat |
| `internal/brain/maintenance` | `byRepository`, `reconcile.go:475`: `if common == ""` → `false` | **Nicht beobachtbar, stehengelassen.** Ein Bereich ohne Repository wird dann unter `""` geführt. Nachgeschlagen wird `""` nur für ein Ereignis, dessen Repository git nicht einordnen kann, und für genau dieses scheitert danach `mergeEvidence` (`git diff` und `git log` brauchen dasselbe Repository, das `rev-parse --git-common-dir` nicht fand): das Ereignis bleibt, kein Fall. Mit einer Wegwerfwelt nachgeprüft — ein Bereich ohne Repository an erster Stelle, drei Ereignisse auf ein verschwundenes, ein leeres und dieses Verzeichnis —: mit und ohne Mutant kein Fall, kein Ereignis abgelegt. Das hält, solange `rev-parse --path-format=absolute --git-common-dir` überall gelingt, wo `diff` und `log` gelingen |
| `internal/brain/maintenance` | `mergeEvidence`, `reconcile.go:528`: `\|\| subjectsErr != nil` weggelassen | **Nicht beobachtbar, stehengelassen.** Gebraucht würde ein Bereich, den `git diff` liest und `git log` ablehnt; gemessen am 2026-09-22 mit git 2.54: ein Baum als Ende liest sich in beiden, ein Blob nur in `log` (`diff` endet mit 129). Der Blob ist darum der Vektor für die **andere** Hälfte, und die stirbt jetzt an `TestARangeOnlyGitLogReadsKeepsTheEvent` (der sich überspringt, wo ein git den Blob auch in `diff` liest) |
| `internal/brain/maintenance` | `isDirectory`, `reconcile.go:680`: `if path == ""` → `false` | **Äquivalent, stehengelassen.** `os.Stat("")` scheitert, also antwortet die Funktion für den leeren Pfad auch ohne die Abkürzung `false`. Die Zeile spart einen Systemaufruf und sagt, was ein fehlender Wiki-Pfad bedeutet |
| `internal/config` | `readManifestAmong`, `manifest.go:202`: `lanes[i].Name < lanes[j].Name` → `<=` | **Äquivalent, stehengelassen.** Die Namen sind die Schlüssel einer Map, also paarweise verschieden; über verschiedenen Schlüsseln geben `<` und `<=` dieselbe Ordnung |
| `internal/config` | `hubEscapes`, `manifest.go:326`: `size > 0` → `>= 0` | **Äquivalent, stehengelassen.** `DecodeRuneInString` antwortet Größe 0 nur für `""`, und dort scheitert schon `len(value) >= size+2` |
| `internal/config` | `RenderRegistry`, `registrywrite.go:77`: `sorted[i].Scope < sorted[j].Scope` → `<=` | **Äquivalent für jede geschriebene Datei, stehengelassen.** Gleiche Bereichsnamen sind die einzige Stelle, an der `<=` die Ordnung von `SliceStable` ändert, und eine Registry mit doppeltem Bereich lehnt `writeRegistryLocked` ab, bevor sie geschrieben wird (`parseRegistry`, doppelter Bereich in `registry.go:137-138`) |
