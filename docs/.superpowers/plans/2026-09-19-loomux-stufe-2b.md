# loomux Stufe 2b — Implementierungsplan: check commit-msg mit [commit], --calibrate und --language

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Portierung von `ultraloom commit-msg` nach Go in `internal/verify/commit/` und CLI-Verdrahtung in `internal/cli/check.go`. Vollständige Ablösung der bisherigen Wortlistenprüfung (`ValidateCommitMessage`) durch `Check` mit Schema `[commit]`, Schwellwertprüfung über alle Zeilen, Span- und Trailer-Erkennung, Umlaut- und Go-Wortliste (Variante B), `--calibrate N`, `--language`, Paritätskorpus 2b und zweisprachige Dokumentation.

**Architecture:**
- `internal/verify/commit/words.go`: `_ORDINARY`, Quellen German, English, Romance, Go-Liste; `Stopwords(lang)` mit `sync.OnceValue`.
- `internal/verify/commit/scan.go`: `Scan(text, lang, threshold, allow) []Finding` mit Spans, Scissors, Trailern, Pfaden, Namen, Fremdschrift-Läufen und Umlautregel.
- `internal/verify/commit/policy.go`: `ReadPolicy(root) (Policy, error)` liest `.loomux/config.toml` (`[commit]`-Tabelle).
- `internal/verify/commit/calibrate.go`: `Calibrate`, `ReadMessages` (über `child.Run` mit 60s Frist und Naht) und `Render`.
- `internal/verify/commit/check.go`: `Check(msg, policy, w io.Writer) error` verbindet `Scan`, Ablehnungsausgabe und Conventional-Commits-Header.
- `internal/cli/check.go`: `loomux check commit-msg` (Dateiprüfung und `--calibrate`).
- `internal/dev/importcases/importcases.go`: `.ultraloom/config.toml` `[commit]` wird nach `.loomux/config.toml` übertragen.
- `testdata/cases/2b`: Paritätskorpus und Fall-Runner `internal/cli/cases_2b_test.go`.

**Tech Stack:** Go (Toolchain 1.27.0, `go.mod` 1.25.0), Standardbibliothek, `golang.org/x/text` (norm NFKD/NFC), `third_party/toml`. Keine neuen Abhängigkeiten.

**Spec:** `docs/.superpowers/specs/2026-09-19-loomux-stufe-2b-design.md`.

## Global Constraints

- **Arbeitsort:** Worktree `C:\Users\micro\Documents\#GIT\loomux\.claude\worktrees\2b-planung-4d6dcc`, Branch `claude/2b-planung-4d6dcc`.
- **Modulpfad:** `github.com/xidus90/loomux`.
- **Sprachen:** Code, Bezeichner, Kommentare, Fehlermeldungen und Commit-Nachrichten englisch. Plan, Spec und `docs/.superpowers/parity/` deutsch. `docs/en/**` englisch, `docs/de/**` deutsch.
- **Kein `init()`, keine Paketvariable, die Daten parst.** Lazy-Loading via `sync.OnceValue`.
- **Coverage 100 % je Funktion.** Ausnahme nur mit `//coverage:exempt <reason>` direkt über `func`.
- **`.loomux/config.toml` schreibt kein Agent.** Policy-Regeln für Menschen vorschlagen.
- **Commits:** Conventional Commits, Mensch als Autor und Committer (`Christoph Wübbels <christoph.wuebbels@gmail.com>`). Vor jedem Commit `git branch --show-current` und `git log -1 --format=%h` prüfen. Niemand außer dem Menschen pusht.

## Vektorklassifikation (Python vs. Variante B)

### Paritätsvektoren (1:1 aus Python übernommen)
- Englische Nachricht ohne Befund: `test_an_english_message_is_clean`
- Deutsche Prosa wird erkannt: `test_german_prose_is_found`
- Zählung je Zeile, nicht je Nachricht: `test_the_threshold_counts_per_line_not_per_message`
- Ein Treffer reicht nicht: `test_one_hit_in_a_line_is_not_enough`
- Zitierter Satz (`"..."`): `test_a_quoted_sentence_does_not_count`
- Code-Span (Backticks): `test_a_code_span_does_not_count`
- Pfad-Token (`wiki/.../file.md`): `test_a_path_does_not_count`
- Trailer (`Co-Authored-By:` ab Z. 2): `test_a_trailer_does_not_count`
- Namenspartikel (`von Neumann`): `test_a_name_particle_does_not_count`
- `von` ohne Großschreibung zählt: `test_von_without_a_capitalised_name_still_counts`
- Scissors (`# --- >8 ---`) beendet Scan: `test_the_diff_below_the_scissors_is_ignored`
- Kommentarzeilen (`#`): `test_comment_lines_are_ignored`
- Gegenrichtung `de`: `test_the_other_direction_finds_english_in_german`, `test_a_german_message_is_clean_under_de`
- `allow`-Muster überspringt Zeile: `test_an_allow_pattern_drops_the_whole_line`
- Finding trägt Zeile und Treffer: `test_a_finding_carries_the_line_and_its_hits`
- Deutsche Wörter, die auch englisch sind, zählen nie: `test_german_words_that_are_also_english_never_count`
- ASCII-transkribierte vs. gefaltete Umlaut-Stoppwörter: `test_umlauts_are_found_although_the_list_is_ascii`
- Hyphenated / Listed Trailer: `test_a_hyphenated_trailer_does_not_count`, `test_a_listed_unhyphenated_trailer_does_not_count`
- Conventional-Commit-Header ist kein Trailer: `test_a_conventional_commit_subject_is_not_a_trailer`
- Englisches Wort `fest`: `test_an_english_fest_is_not_a_finding`
- Kein Trailer-Exempt auf Zeile 1: `test_no_trailer_is_exempt_on_the_first_line`
- `BREAKING CHANGE:` Trailer ab Zeile 2: `test_a_breaking_change_footer_does_not_count`
- Deutsches Wort `still` unter `de`: `test_a_german_still_is_not_a_finding`
- Umbrechende Backtick-Spans: `test_the_opening_line_of_a_wrapped_span_is_exempt`, `test_the_tail_of_a_wrapped_code_span_is_exempt`, `test_a_code_span_wrapping_three_lines_is_exempt`
- Schließender Backtick ohne Öffner: `test_a_closing_backtick_with_no_opener_leaves_its_text_scored`
- Ausgeglichene Spans: `test_a_balanced_span_is_left_alone`
- Einzelner abschließender Backtick: `test_a_lone_trailing_backtick_strips_nothing`
- Anführungszeichen brechen nicht um: `test_a_quoted_span_does_not_wrap_across_lines`, `test_a_quoted_span_within_one_line_is_still_exempt`
- Einzelnes Anführungszeichen (Messung `80"`): `test_a_lone_quote_is_punctuation_not_a_span_opener`
- Apostroph kein Begrenzer: `test_an_apostrophe_is_not_a_quote_delimiter`
- Git-Hinweise verschieben keine Flags: `test_a_git_hint_line_does_not_move_the_span_flags`
- Ausgenommene Zeile trägt Span weiter: `test_an_exempted_line_still_carries_its_span_onward`
- Absatzgrenze (Leerzeile) schließt Spans: `test_an_unpaired_quote_does_not_outlive_its_paragraph`, `test_an_unpaired_backtick_does_not_outlive_its_paragraph`, `test_a_span_does_not_wrap_across_a_blank_line`
- Pfad/Partikel in/außerhalb Spans: `test_a_path_and_a_name_particle_inside_a_wrapped_span`, `test_a_path_and_a_name_particle_outside_a_span`
- Stray Quote / Backtick im Betreff: `test_a_stray_quote_with_no_blank_line_anywhere_does_not_disable_the_gate`, `test_a_stray_backtick_in_the_subject_does_not_silence_the_body`
- Ersatz durch Leerzeichen verhindert Wortverschmelzung: `test_a_removed_span_leaves_a_separator_behind`
- Spanschwanz kann kein Trailer sein: `test_the_tail_of_a_carried_span_cannot_pass_as_a_trailer`
- Fremdschrift-Läufe (CJK, Kyrillisch, Arabisch, Hebräisch, Griechisch, Devanagari, Thai, Hangul): `test_two_words_of_a_non_latin_script_are_a_finding`, `test_each_covered_script_produces_hits`
- Ein zitiertes Fremdschrift-Wort unter Schwelle: `test_a_single_quoted_term_stays_below_the_threshold`, `test_one_japanese_word_stays_under_the_threshold`
- Fremdschrift-Lauf zählt einmal unabhängig von Länge: `test_a_script_run_counts_once_however_long_it_is`
- Latein mit Diakritika kein Fremdschrift-Treffer: `test_latin_with_diacritics_is_not_a_script_hit`
- Span/Allow für Fremdschrift: `test_a_script_hit_obeys_the_span_exemption`, `test_a_script_hit_obeys_an_allow_rule`
- Romanische Sprachen (Spanisch, Portugiesisch, Französisch) unter `en`: `test_spanish_prose_is_refused_where_commits_are_english`, `test_portuguese_prose_is_refused_where_commits_are_english`, `test_french_prose_is_refused_where_commits_are_english`
- Alltägliche englische Wörter überleben: `test_ordinary_english_survives_the_merged_list`
- Joiner (`-`, `_`): `test_a_list_word_inside_an_identifier_is_not_evidence`, `test_a_hyphen_on_one_side_alone_is_enough_to_exempt`
- Fullwidth / Mathematical Latin: `test_fullwidth_and_mathematical_latin_are_latin`
- U+30FC Katakana-Hiragana Prolonged Sound Mark ist Teil von CJK: `test_one_japanese_word_stays_under_the_threshold`
- Calibrate: `test_a_higher_threshold_refuses_fewer`, `test_the_calibrated_default_refuses_only_the_prose`, `test_an_allow_pattern_takes_a_message_out_of_every_count`, `test_the_table_names_the_first_line_of_every_refused_message`, `test_the_subject_skips_the_blank_lines_above_it`
- Config Schema: unbekannte Schlüssel, ungültige Regex, ungültige Typen, `match` statt `regex`.

### Abweichungsvektoren (unter B gekippt)
1. **Go-Liste & Umlautregel:**
   - `"feat: füge neue sprachprüfung hinzu"`: Python 0 Treffer (akzeptiert). B: `füge` (Go-Liste), `neue` (Python-Liste), `sprachprüfung` (Umlaut `ü`), `hinzu` (Go-Liste) -> 4 Treffer -> abgelehnt!
   - `"korrigiere fehler in der verifikation"`: Python 1 Treffer (`der`). B: `korrigiere` (Go), `fehler` (Go), `der` (Python) -> 3 Treffer -> abgelehnt!
   - `"aktualisiere dokumentation und beispiele"`: Python 1 Treffer (`und`). B: `aktualisiere` (Go), `dokumentation` (Go), `und` (Python), `beispiele` (Go) -> 4 Treffer -> abgelehnt!
   - `"WIP: ändere dateien"`: Python 0 Treffer. B: `ändere` (Go/Umlaut), `dateien` (Go) -> 2 Treffer -> abgelehnt!
   - `"Verbessere Performance für Windows"`: Python 1 Treffer (`für` als Stoppwort `fuer`). B: `verbessere` (Go), `für` (Stoppwort `fuer`) -> 2 Treffer -> abgelehnt!
   - `"entferne ungenutzte importe"`: B hat 1 Treffer (`entferne`). Bei Schwelle 2 nicht abgelehnt (Header scheitert separat an Conventional).
2. **Fehlende Konfig:** Python prüft ohne `[commit]` nicht; loomux prüft mit Vorgaben (`en`, threshold 2, conventional true).
3. **Exit-Codes:** Bei Ablehnung Exit 1 (Python: Exit 2).
4. **Calibrate ohne Sprache:** loomux nimmt Vorgabe `en` (Python verweigert).
5. **Conventional-Header:** loomux prüft Conventional Commits, wenn `conventional = true` (Vorgabe).

---

## File Structure

| Datei | Verantwortung |
|---|---|
| `internal/verify/commit/words.go` | `_ORDINARY`, `_GERMAN_SOURCE`, `_ENGLISH_SOURCE`, `_ROMANCE_SOURCE`, `_GO_SOURCE`, `Stopwords(lang)` via `sync.OnceValue` |
| `internal/verify/commit/words_test.go` | Tests der Stoppwortmengen, ordinary-Filterung, Disjunktheit |
| `internal/verify/commit/scan.go` | `Finding`, `Scan(text, lang, threshold, allow) []Finding`, Spans, Tokenisierung, Fremdschriften, Umlautregel |
| `internal/verify/commit/scan_test.go` | Paritäts- und Abweichungstests für `Scan` |
| `internal/verify/commit/policy.go` | `Policy`, `ReadPolicy(root string) (Policy, error)`, Validierung von `[commit]` und `[[commit.allow]]` |
| `internal/verify/commit/policy_test.go` | Tests aller Konfigurationsfälle und Fehlermeldungen |
| `internal/verify/commit/check.go` | `Check(msg string, policy Policy, w io.Writer) error`, Formatierung der Ablehnung und Header-Fehler |
| `internal/verify/commit/check_test.go` | Tests von `Check`, Fehlertexten und Einzügen |
| `internal/verify/commit/calibrate.go` | `Calibrate`, `ReadMessages`, `Render`, Seam für `child.Run` |
| `internal/verify/commit/calibrate_test.go` | Tests von `--calibrate`, Schwellwertzählung und Tabellenformatierung |
| `internal/cli/check.go` (ändern) | `loomux check commit-msg` und `loomux check commit-msg --calibrate N` |
| `internal/cli/check_test.go` (ändern/ergänzen) | Tests der CLI-Aufrufe, Exit-Codes und Flags |
| `internal/dev/importcases/importcases.go` (ändern) | `[commit]` aus `.ultraloom/config.toml` in `.loomux/config.toml` übertragen |
| `internal/dev/importcases/importcases_test.go` | Test für den Import von `[commit]` |
| `testdata/cases/2b-worlds/`, `2b-source/`, `2b-map.toml`, `2b/` | Paritätsfälle für `check commit-msg` |
| `internal/cli/cases_2b_test.go` | Fall-Runner für Stufe 2b |
| `docs/.superpowers/parity/stufe-2b.md` | Abweichungsliste der Stufe 2b |
| Dokumentation (ändern) | `README.md`, `README.de.md`, `docs/{en,de}/*.md`, Fusions-Spec |

---

## Tasks

- [ ] **Task 0: Arbeitsort und Basis prüfen**
  - Git branch, HEAD und status prüfen
  - Gate fahren (`sh ci/gate.sh`)

- [x] **Task 1: `words.go` und `words_test.go`**
  - Map-Literale für Ordinary, German, English, Romance und Go-Liste
  - `Stopwords(lang string) map[string]struct{}` mit `sync.OnceValue`
  - Tests für Mengenfilterung und Disjunktheit schreiben und verifizieren

- [x] **Task 2: `scan.go` und `scan_test.go`**
  - Spans (Backticks über Zeilen, Quotes je Zeile, Absatzgrenze schließt Spans, Scissors)
  - Trailer- und Name-Particle-Filterung, Pfadtoken-Prädikat
  - Faltung (`ä→ae, ö→oe, ü→ue, ß→ss`) und Stoppwort-Matching mit Joiner-Regel (`-`, `_`)
  - Umlaut-Regel (für Ziel `en`): reine Umlautwörter erfassen
  - Fremdschrift-Läufe (NFKD, Scripts-Klassifikation, U+30FC CJK, NFC, Kappen auf 12 Runen)
  - Alle portierten Paritäts- und Abweichungsvektoren testen

- [x] **Task 3: `policy.go` und `policy_test.go`**
  - `Policy`-Struktur (`Language`, `Threshold`, `Conventional`, `Allow`)
  - `ReadPolicy(root string) (Policy, error)` mit TOML-Parsing
  - Validierung: Typen, bekannte Schlüssel, `threshold > 0`, `allow` mit `regex` und `reason`, Ablehnung von `match`
  - Vorgaben bei fehlender Sektion: `Language: "en"`, `Threshold: 2`, `Conventional: true`
  - Tests für alle gültigen und fehlerhaften Konfigurationen

- [x] **Task 4: `check.go` und `check_test.go` (Ablösung von `ValidateCommitMessage`)**
  - `Check(msg string, policy Policy, w io.Writer) error`
  - Ablehnungstext auf stderr formatieren (Einzug entsprechend Zeilennummer, `hits: ...`)
  - Conventional Commits Header-Prüfung integrieren (Header-Fehler als eigene Zeile)
  - `language.go` und `language_test.go` entfernen bzw. ersetzen
  - Unit-Tests für `Check`

- [x] **Task 5: `calibrate.go` und `calibrate_test.go`**
  - `Calibrate(messages []string, lang string, thresholds []int, allow []*regexp.Regexp)`
  - `ReadMessages(root string, count int) ([]string, error)` über `child.Run` (`git log -z --format=%B -n N`)
  - `Render(messages []string, lang string, thresholds []int, w io.Writer, allow []*regexp.Regexp)`
  - Seam für `child.Run` zur Testbarkeit ohne echtes Git-Repository
  - Tests für Formatierung, Sortierung und Fehlerbehandlung

- [x] **Task 6: CLI-Verdrahtung in `internal/cli/check.go`**
  - `checkCommitMsg(args []string, stdout, stderr io.Writer) int`
  - Aufruf `loomux check commit-msg <file>`
  - Aufruf `loomux check commit-msg --calibrate N [--language en|de]`
  - Flag-Prüfungen: `--language` verboten bei Dateiprüfung (Exit 2); `--calibrate` ohne Datei, `N >= 1`
  - Tests in `internal/cli/check_test.go`

- [x] **Task 7: `importcases` für `[commit]` erweitern**
  - `importcases.go`: `config["commit"]` in `result["commit"]` übernehmen
  - Test in `internal/dev/importcases/importcases_test.go`

- [x] **Task 8: Paritätsfälle `2b-source` aufzeichnen und nach `2b` importieren**
  - `2b-map.toml` anlegen (`ultraloom commit-msg ` -> `loomux check commit-msg `)
  - Aufzeichnung ausgewählter Fälle aus `ultraloom commit-msg` via `loomux dev record-case`
  - Übersetzung nach `testdata/cases/2b` via `loomux dev import-cases`
  - Fall-Runner `internal/cli/cases_2b_test.go` implementieren und ausführen

- [x] **Task 9: Abweichungsliste `docs/.superpowers/parity/stufe-2b.md`**
  - Dokumentation der 10 Abweichungen und Testabdeckung

- [x] **Task 10: Dokumentation und Fusions-Spec nachziehen**
  - `README.md` und `README.de.md` aktualisieren
  - `docs/en/cli-reference.md` und `docs/de/cli-reference.md` aktualisieren
  - `docs/en/configuration.md` und `docs/de/configuration.md` aktualisieren
  - `docs/en/hooks.md` und `docs/de/hooks.md` aktualisieren
  - Fusions-Spec `docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md` aktualisieren (2b auf ✅)
  - Spec `docs/.superpowers/specs/2026-09-19-loomux-stufe-2b-design.md` Stand auf `umgesetzt`

- [ ] **Task 11: Endabnahme und Gate-Verifikation**
  - `go test ./...`
  - `loomux check commit-msg --calibrate 300` über loomux-Historie: 0 Ablehnungen bei Schwelle 2
  - `sh ci/gate.sh`
  - 100 % Coverage je Funktion sicherstellen
