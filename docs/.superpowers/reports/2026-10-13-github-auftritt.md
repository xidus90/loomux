# Bericht: Agent-Team-Lauf „GitHub-Auftritt“

- Plan: `docs/.superpowers/plans/2026-10-10-github-auftritt.md`
- Spec: `docs/.superpowers/specs/2026-10-10-github-auftritt-design.md`
- Lauf: `.team-runs/20261011-112138`, Generationen 1–3 (11.–13.10.2026)
- Feature-Branch: `feat/github-auftritt`
- Abschluss-Gate (`sh ci/gate.sh`) auf dem Feature-Branch: Exit 0
  (lint/go, test/go, coverage/go, lint/wiki, lint/wiki-sources ok)

## Ergebnis

Alle sieben Plan-Tasks sind umgesetzt und gemergt. Die Bug-Hunts fanden
40 Befunde (B1–B40), dazu kam B41 aus dem Re-Review von B37. Zwölf Bugs
bekamen einen eigenen Fix und sind gemergt: B1, B3, B8, B9, B16, B17,
B22, B24, B36, B37, B40, B41. B2 ist im Fix von B1 mitbehoben, B28 ein
Duplikat von B24. B19 und B20 sind geparkt. Die übrigen sind als `low`
bestätigt oder als Duplikat erkannt. Die Hunts endeten nach Runde 4 auf
Wunsch des Menschen: „Nach den nächsten bug fixes hören wir mit der hunt
auf“.

## Vor dem Pull Request

- **Release-Label:** Der Plan sieht `release:none` ohne Changelog vor.
  Der Branch enthält aber `fix:`-Commits für Fehler, die schon auf
  `master` standen (B16, B17, B22, B36, B37, B41). Der PR braucht
  deshalb `release:patch` mit `## Changelog`.
- **Commits nach Themen:** Vor dem Push wurden die 44 Commits des Laufs
  zu 13 Commits nach Themen neu gebaut, ein Commit je Änderung. Folge-
  Korrekturen, etwa B37 Runde 2 und 3 und der B24-Fix, stecken im Commit
  der Änderung, die sie korrigieren. Der Baum ist derselbe wie vorher.

## Wurzeln

Eine Runde ist jedes `[impl]` und jedes `[fix]` ohne `:conflict`.

| Wurzel | Inhalt | Zustand | Runden | Letztes Urteil |
|---|---|---|---|---|
| T1 | `install.md` en/de | gemergt | 1 | verify:rebase pass |
| T2 | `releases.md` en/de | gemergt | 2 | verify:review pass |
| T3 | README.md / README.de.md | gemergt | 4 | verify:rebase pass |
| T4 | Navigationsseiten `docs/*/README.md` | gemergt | 1 | verify:rebase pass |
| T5 | Verweise (AGENTS, CONTRIBUTING, PR-Vorlage) | gemergt | 1 | verify:review pass |
| T6 | Social-Preview-SVG | gemergt | 1 | verify:review pass |
| T7 | Ersteinrichtung und Konfiguration | gemergt | 3 | verify:review pass |
| B1 | Aussagen der Social Preview begrenzt | gemergt | 2 | verify:review pass |
| B3 | Befehlsregel- und `bash -c`-Sätze in Index und Hooks | gemergt | 3 | verify:rebase pass |
| B8 | vollständige Grenzenliste des Guards | gemergt | 5 | verify:review pass |
| B9 | Datenschutz-Abschnitt und `never`-Ausnahme der Graph-Befehle | gemergt | 2 | verify:review pass |
| B16 | `config list` zählt Inline-Rules-Arrays | gemergt | 1 | verify:review pass |
| B17 | `init` über Inline-Rules-Arrays | gemergt | 1 | verify:rebase pass |
| B19 | Pfadschreibweisen mit Geräte-Präfix / 8.3 im Guard | **geparkt** | 3 | verify:fix pass |
| B22 | Git 2.31 als Minimum; `worktree remove` löscht nicht durch Junctions | gemergt | 2 | verify:rebase pass |
| B24 | `worktree remove` lehnt gesperrten Worktree vor dem Link-Sweep ab | gemergt | 1 | verify:review pass |
| B36 | `config set` neben einer Inline-Nachbartabelle | gemergt | 1 | verify:rebase pass |
| B37 | endendes `/**` trifft jede Tiefe, linear in der Tiefe | gemergt | 3 | verify:rebase pass |
| B40 | Argumente von `graph_file_api` & Co. in der CLI-Referenz | gemergt | 1 | verify:review pass |
| B41 | Folge führender `**/` als eines lesen | gemergt | 1 | verify:review pass |

B20 hatte nie einen eigenen Branch. Er wartete auf B19 und wurde mit ihm
geparkt.

Bemerkenswert:

- **B37:** drei Runden. Runde 1 machte den Matcher korrekt, aber
  quadratisch in der Pfadtiefe. Eine 24K-Bash-Zeile brauchte 16 s, und
  Claude Code lässt einen Hook nach 15 s Timeout durch. Runde 2 machte ihn
  linear, übersprang aber Vorfahren bei `/` in einer Zeichenklasse
  (Fail-open). Runde 3 behob das; ein Differenztest über 153 Mio. Aufrufe
  zeigt keine Abweichung. Die quadratische Laufzeit kam erst mit Runde 1
  auf diesen Branch; auf `master` stand nur der Tiefenfehler, den Runde 1
  behob.
- **B36, B37, B40, B41:** Diese Fehler stammen nicht aus diesem Branch,
  sie stehen schon in v1.4.0 bzw. auf `master`. Sie wurden trotzdem hier
  behoben.
- **B41:** Mehrere führende `**/` in einer Projektregel ließen den Guard
  mit O(d^k) laufen. `**/**/**/secrets/**` lehnte eine 2-KB-Bash-Zeile
  erst nach 36,6 s ab, und Claude Code lässt einen Hook nach 15 s Timeout
  durch. Nebenbei schützte im strikten Modus eine Regel `**/**/x` gar
  nichts, weil die Vorprüfung nur ein `**/` abschnitt. Beides ist behoben.
- **Repin durch den Menschen:** B24 und B40 brauchten je ein
  `loomux wiki repin` im Worktree, das nur ein Mensch ausführen darf.

## Bug-Hunts

| Runde | Partitionen | Befunde | neu bestätigt ≥ medium |
|---|---|---|---|
| R1 | README, install-and-releases, setup-config-wiki, index-contrib-assets | 13 | ja |
| R2 | README, install-and-releases, setup-config-wiki, hooks-index-contrib | 10 | ja (B16, B17, B19, B20, B22) |
| R3 | user-docs, config-and-init, worktree, hooks-index-contrib | 7 (B24–B30) | ja (B24) |
| R4 | user-docs, config-and-init, worktree, hooks-index-contrib | 10 (B31–B40) | ja (B36, B37, B40) |

Danach endete das Jagen auf Wunsch des Menschen, obwohl Runde 4 noch neue
Befunde ab `medium` brachte. Ein weiterer Hunt würde bei den Bereichen
`internal/pathkey`, `internal/config/edit` und `worktree remove`
ansetzen, wo die letzten Fixes liegen.

### Befunde der Runden 3 und 4

| Bug | Befund | Urteil |
|---|---|---|
| B24 | gesperrter Worktree verliert eigene Junctions | medium, behoben |
| B25 | Junction `x.` neben Datei `x` | low (Durchlöschen widerlegt) |
| B26 | schreibgeschützte Junction stoppt `remove` | low |
| B27 | gemischte Zeilenenden in `AppendInline` | low, Duplikat von B17 |
| B28 | Doku-Sätze zum Link-Sweep | Duplikat von B24 |
| B29 | deutscher Registry-Satz in getting-started §5 | low |
| B30 | getting-started Schritt 2 ohne Modul `base` | low |
| B31 | Worktree ohne `.git`-Datei verliert Junctions | low |
| B32 | Worktree-Liste ohne `-z` gelesen | low |
| B33 | Wiki: „kein `upgrade`-Befehl“ | low |
| B34 | `(git push)` in der Grenzenliste | low, Duplikat von B8 |
| B35 | Pre-commit prüft Doku-Eingaben nicht gegen den Index | low |
| B36 | `config set` neben Inline-Tabelle | medium, behoben |
| B37 | endendes `/**` nach Wildcard nur eine Ebene | medium, behoben |
| B38 | brain-Befehle „überspringen still“ | low |
| B39 | falsche Meldung für config ohne `[area]` | low |
| B40 | `graph_file_api` mit `file_path` dokumentiert | medium, behoben |
| B41 | mehrere führende `**/` (aus dem Re-Review von B37) | medium, behoben |

## Parked roots

- **B19** (Branch `team/20261011-112138/B19`, ungemergt): Pfadangaben mit
  Geräte-Präfix (`\\?\`, `\\.\`) oder 8.3-Kurznamen umgehen die
  Pfadregeln des Guards. Frage an den Menschen: „Das Security-Review für
  B19 (Task 72) wurde von einer Sicherheitsprüfung gestoppt: Der Reviewer
  sollte Pfad-Schreibweisen aufzählen, die am loomux-Wächter vorbeikommen,
  und darf das nicht weiter erzeugen. […] Parallel hat das Code-Review
  einen bestätigungsbedürftigen medium-Befund (Kopieren in einen `\\?\`-
  oder `\\localhost\C$`-Ordner umgeht die Regeln); ein Fix wäre Runde 4 von
  B19. Wie weiter?“ Antwort: Park B19.
- **B20** (kein eigener Branch; `team/20261011-112138/B20` existiert
  nicht): Antigravity schreibt über `::$DATA` an `.env`, `armed.toml` und
  `no-verify` vorbei (bestätigt medium). Frage an den Menschen: „B20 […]
  wartete auf den Merge des jetzt geparkten B19. Wie weiter mit B20?“
  Antwort: Park B20.

Beide Lücken bleiben offen und brauchen eine eigene Arbeit am Guard.

## Deferred findings (low)

Ein Eintrag pro Befund, den ein Verify-Urteil als `low` bestätigt hat:
Id, Wurzel, Aussage.

- B3-sec-F2 (B3): „writes to secrets such as `.env`“ deutet eine Klasse an, der Guard kennt aber eine feste Liste; `id_ed25519`, `.netrc`, `secrets.yaml` gehen durch.
- B3-code-F2 (B3): Überschrift „Strict Memory Protection“ passt nicht mehr zum Inhalt.
- B3-code-F3 (B3): „entscheidet in 35 ms“ liest sich als Garantie; der Benchmark misst bis 351 ms bei löschendem `find`.
- B3-code-F4 (B3): hooks.md §5 sagt noch, die Interpreter-Regel lese den String „wie die übrigen Befehlsregeln“.
- B3-code-F5 (B3): Wiki-Log sagt, nur loomux' eigene Befehlsregeln läsen `bash -c`; die Interpreter-Regel tut es auch.
- B1-ver-F3 (B1): „Guard decisions within 35 ms“ verliert das Wort „budget“.
- B1-ver-F4 (B1): Die Social Preview lässt „One binary“ fallen, das die Spec vorgab.
- T1-sec-F1 (T1): `sha256sum --check --ignore-missing` ist grün, wenn das Binary umbenannt wurde.
- T1-sec-F2 (T1): Nach einem Prüfsummenfehler bleibt die Datei liegen und blockiert spätere Läufe (Vorläufer von B23).
- T2-sec-F3 (T2): Setup sagt nicht, den privaten Release-Schlüssel nach `gh secret set` zu löschen.
- T3-ver-F8 (T3): Stellung der Klammer `(run --no-model)` weicht vom Plan ab.
- T3-code-F2 (T3): Features-Zeile nennt alle Lanes nach jeder Bearbeitung; das Edit-Profil ist nur lint/types.
- T3-sec-F2 (T3): Features-Zeile verliert die Grenze „Schreibzugriffe, die der Guard aus einer Shell-Zeile liest“.
- T3-ver-F3 (T3): Der falsche Netzwerksatz steht noch in Spec und Plan.
- T3-ver-F5 (T3): Doku-Zeile nennt „4-phase lifecycle“ und „SSE streaming“; hooks.md hat drei Phasen, der Stream ist geplant.
- T3-code-F5 (T3): Check-Chain-Zeile „über den neuen Inhalt der Runde“ ist ungenau.
- T4-code-F1 (T4): Zeile „Benchmark matrix“ beschreibt sich wie die Zeile „Open-source matrix“.
- T7-ver-F7 (T7): „init schreibt nur, was von den Vorgaben abweicht“ gilt nicht für die Bereichsdeklaration.
- T7-ver-F8 / T7-sec-F2 / T7-code-F1 (T7): „bleiben lokal“ steht direkt unter den Cloud-Modi, deren Inhalte der Cloud-Kanal ausgibt.
- T7-code-F2 (T7): code-graph.md nennt noch „offene Stufen“ der README-Roadmap.
- T7-ver-F2 (T7, auch B29/R3-P1-F1): Der Registry-Satz in getting-started §5 gilt nur, solange kein Baum das Zustandsverzeichnis umschließt; der deutsche Satz ist zweideutig.
- B8-ver-F2 / B8-ver-F6 (B8): Der Junction-Punkt liest sich, als lehne der strikte Modus auch das Anlegen des Links ab.
- B8-code-F1 (B8, auch B34/R4-P4-F2): `(git push)` geht laut Liste durch; mit der Regel des Repos endet es mit Exit 2.
- B8-code-F2 (B8): Der deutsche Satz zu Shell-Zeilen und MCP-Dateiwerkzeugen ist verschachtelt.
- R1-P4-F4 (B4), R2-P1-F2 (B15, Duplikat von B4): „vollständige Referenz“-Zeilen nennen nicht alle Abschnitte von configuration.md.
- R1-P2-F1 (B5): „serve hebt es binnen 24 Stunden auf“ stimmt im Mechanismus nicht.
- R1-P2-F2 (B6): Beta-Marker beim Festnageln eines Tags ist in install.md nur für `--beta`/`--stable` beschrieben.
- R1-P2-F3 (B7): `CheckCommits` überspringt die Formprüfung für `Merge`/`fixup!`/`Revert`-Köpfe.
- B10-ver-F1 (B10): Mit genau den Beispielregeln schreibt `python -c` an `.loomux/config.toml` vorbei.
- B11-ver-F1 (B11): Die 35-ms-Aussage gilt nicht für ein löschendes `find` über viele Einträge.
- B12-ver-F1 (B12): Die Go-Lint-Lane geht bei neuerer `go`-Version im `go.mod` online.
- B13-ver-F1 (B13): Edit-Lanes ohne `on_file`-Form laufen über das ganze Projekt.
- R2-P1-F1 (B14): Zehn `!windows`-Funktionen sind weder gemessen noch ausgenommen, trotz „100 % Coverage“.
- R2-P3-F3 (B18): `status` druckt Matcher ohne `MultiEdit`.
- R2-P4-F3 (B21): PR-Vorlage übergibt `--commits` als Array statt als Datei.
- R2-P2-F2 (B23): Windows-Quickstart bei laufendem `loomux.exe`: Move-Item scheitert still, später falscher Prüfsummenalarm.
- B9-code-F1 (B9): Der never-Test ignoriert Fehler beim Anlegen der Datei.
- B9-sec-F2 (B9): Ein never-`.go` mit Syntaxfehler verrät Pfad und Token über die Rebuild-Meldung.
- B9-sec-F3 (B9): `blast` nennt Aufrufer aus never-Pfaden.
- B9-sec-F4 (B9): `loomux lint` gibt Dateinamen aus never-Wikipfaden aus.
- B9-ver-F5 (B9): Die Liste der Graph-Ausgaben lässt `blast` aus.
- B9-ver-F6 (B9): getting-started behält die unbegrenzte never-Zeile.
- B16-code-F1 (B16): Feldkommentar `Entry.Count` ist veraltet.
- B16-sec-F1 (B16): `config list` zählt auch Nicht-Tabellen in einem Rules-Array.
- B17-ver-F4 (B17, auch B27/R3-P2-F1): `AppendInline` setzt das Komma hinter ein `\r` bei gemischten Zeilenenden.
- B19-ver-F16 / B19-ver-F18 (B19, geparkt): Freigaben dieses Rechners unter unbekanntem Namen; strikter Modus lockerer als Vorgabe.
- B22-sec-F1 (B22): Unter Git 2.30 landen Hooks nirgends falsch, nur das Abschalten außerhalb Windows ist geplant.
- B22-ver-F3 (B22): `dropLinks` vor Git (durch B24 für gesperrte Worktrees behoben, Rest siehe B31).
- B22-sec-F2 / B22-sec-F3 / B22-sec-F4 (B22): Ersetzter Worktree-Ordner, späte Junction und fremde Reparse-Tags beim Remove.
- B22-ver-F1 (B22): Kommentar über Gits Ablehnung war seit `dropLinks` falsch (durch B24 nachgezogen).
- B24-ver-F8 / B24-sec-F1 (B24): Unlock-Hinweis nennt den Pfad ohne Anführungszeichen.
- B24-ver-F9 (B24): Zwei Testkommentare veraltet.
- B24-code-F1 (B24): Kommentar „the first two rest on os.SameFile“ ist zu schwach.
- R3-P3-F2 (B25): Junction `x.` neben Datei `x` löscht die Datei; Git löscht nicht durch.
- R3-P3-F3 (B26): Schreibgeschützte Junction stoppt `worktree remove`.
- R3-P1-F2 (B30): getting-started Schritt 2 nennt das Modul `base` nicht.
- R4-P3-F1 (B31): Fehlt die `.git`-Datei, verliert `remove` selbst angelegte Junctions.
- R4-P3-F2 (B32): Worktree-Liste ohne `-z`; Pfad mit Zeilenumbruch erzeugt Schein-Einträge (`-z` erst ab Git 2.36).
- R4-P4-F1 (B33): Wiki code-graph.md behauptet „kein `upgrade`-Befehl“.
- R4-P4-F3 (B35): Pre-commit prüft die Eingaben der neuen Doku-Tests nicht gegen den Index.
- R4-P1-F1 (B38): getting-started §5: brain-Befehle sind nicht nachsichtiger als die Schranke.
- R4-P1-F2 (B39): getting-started §5 nennt für config ohne `[area]` die falsche Meldung.
- B36-ver-F4 (B36): Ein gepunkteter Nachbarschlüssel (`roles.describe`) blockiert `config set model.name` weiter.
- B36-code-F1 (B36): Kommentar in edit.go:118-121 nennt den falschen Grund der Ablehnung.
- B36-ver-F3 (B36): `TestRefreshWaitsForAReleasedLock` ist unter Windows bei Last flaky (Sharing-Violation).
- B37-code-F1 / B37-ver-F3 (B37): Testdatei `pathrule_depth_repro_test.go` ist nach Herkunft benannt.
- B37-sec-F2 (B37): Die Laufzeit wächst quadratisch mit der Zahl der `[` im Präfix (256 Klassen: 2 s bei 8K Zeichen); nur aus `[policy.paths]`, die ein Mensch schreibt.
- B37-code-F2 (B37): Grenzkommentar in glob.go:41-47 ist korrekt, aber schwer lesbar.
- B37-ver-F2 (B37): Ein für sich unlesbares Präfix vor `/**` liefert den Syntaxfehler nur noch bei genug tiefen Namen.
- B37-ver-F6 / B37-ver-F7 (B37): Präfix, das auf einen einzelnen Backslash oder auf `[` + `/` ohne `]` endet, liefert jetzt `ErrBadPattern`. Der Guard lehnt ab, `tailMayReach` im strikten Modus verwirft den Fehler.
- B37-ver-F2 aus g3-57 (B37): Ein Glob `build\/**` lädt, lehnt aber jeden Schreibzugriff im Projekt ab (fail-closed).
- B40-ver-F4 (B40): Der Wiki-Log-Eintrag nennt `schreibschranke.md` statt `warum-fast-die-vorgabe-bleibt.md`.
- B40-ver-F5 (B40): Der neue Referenztest prüft nur Argumentnamen, keine Pflichtfelder oder Defaults.
- B41-code-F1 (B41): configuration.md:94 en/de sagt nicht, dass mehrere führende `**/` wie eines gelesen werden; man liest sonst `**/*/x`.
- B41-sec-F1 (B41): Eine Regel wie `**/**/key.pem` steht bei `git clean -fdx` oder `rm -rf .` nicht mehr unter den Gründen der Ablehnung; die Ablehnung selbst bleibt durch die eingebauten Regeln.

## Weitere Hinweise

- Der Testhelfer `payload()` in `internal/hooks/pretool_test.go` setzt
  keine Escapes vor Anführungszeichen. Eine Eingabe mit `"` scheitert dann
  am JSON und sieht wie eine Regelablehnung aus. Das ist ein Nebenfund aus
  B34, nicht als Befund geführt.
- Der B24-Implementer hat für seine RED-Läufe den Guard kurz abgeschaltet.
  `verify:fix` B24 hat geprüft, dass davon nichts zurückblieb.
- Im Haupt-Tree liegen seit Beginn von Generation 3 zwei unversionierte
  Einträge, `good` und `tampered`. Sie stammen nicht aus diesem Lauf und
  sind unberührt.
- Ein Write auf einen 32K-Zeichen-Pfad ohne geschütztes Element dauert
  schon im Basis-Binary je nach Last 4–36 s (Nebenfund aus
  `verify:review` B37). Er verzögert nur einen erlaubten Aufruf; ein
  Kandidat für einen späteren Hunt.
- Mehrere Verdikte von B37 (g3-55, g3-56, g3-57) wurden nach dem
  Abschluss ihrer Tasks neu geschrieben, g3-58 von einem zweiten Verifier
  mit gleichem Inhalt überschrieben. Alle Fassungen tragen `pass` bzw. nur
  Befunde `low`, außer dem neuen Befund, der B41 wurde.
- Ein erstes `[verify:final]` (g3-60) auf `ca1994cf` scheiterte an diesem
  Bericht (Zahl der Fixes, Herkunft der Laufzeit, Release-Label). Diese
  Fassung behebt das.
- Kosten sind nicht bekannt.
