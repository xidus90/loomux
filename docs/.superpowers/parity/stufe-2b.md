# Abweichungsliste Stufe 2b

**Quelle:** `ultraloom commit-msg` (`src/ultraloom/commit/`, `tests/commit/`),
aufgezeichnet am 2026-09-19 mit `uv run ultraloom commit-msg` gegen die Welten
unter `testdata/cases/2b-source/check-commit-msg/`.

**Stand:** 2026-09-19. 19 Fälle, neun Abweichungen. Alle 19 bestehen: Die Übersetzung bildet den
Exit-Code einer Ablehnung über `[[exit]]` in `testdata/cases/2b-map.toml` von
Pythons 2 auf loomux' 1 ab (Eintrag 5), sodass jeder Fall einen einzigen
erwarteten Code trägt. Eine „genehmigte Abweichung“, die bloß irgendwie
abweichen muss, gibt es hier nicht: Sie würde auch einen falschen Code
durchlassen.

Die Spalte „Fall / Test“ nennt den Fall aus `testdata/cases/2b/` oder den
Test in `internal/verify/commit/` bzw. `internal/cli/`, der das Verhalten
festhält.

| Nr. | Fall / Test | Alt (Python) | Neu (loomux 2b) | Begründung |
|---|---|---|---|---|
| 1 | `TestPolicyDefaultWhenNoConfigFile`, `TestPolicyDefaultWhenNoCommitSection` | Ohne `[commit]` lief keine Prüfung: Exit 0, die Nachricht wurde nicht gelesen. | Ohne `[commit]` gilt die Vorgabe `language = "en"`, `threshold = 2`, `conventional = true`. | Ein Tor, das ohne Konfiguration nichts prüft, schützt nichts. Ein Agent darf `.loomux/config.toml` nicht schreiben, also muss die Vorgabe tragen. |
| 2 | `TestStopwordsEN`, `TestScanVariantBDeviations` | Für das Ziel `en` speisten nur die deutsche und die romanische Quelle die Stoppwortliste. | `goSource` ist eine vierte Quelle: 82 deutsche Entwicklerwörter (`fehler`, `datei`, `behebe`, `aktualisiere`, `hinzu` …) aus der alten Go-Prüfung, gefiltert gegen `ordinary["en"]`. Sie zählen als Treffer gegen Englisch. | Gemessen an 1.821 Commits fing die Python-Liste 0 von 8 kurzen deutschen Betreffzeilen; mit dieser Quelle sind es 7. |
| 3 | `TestScanUmlautsFoundAlthoughListIsASCII`, `TestScanFoldsCapitalSharpS`, `TestScanVariantBDeviations` | Umlaute wurden nur gefaltet und gegen die ASCII-Liste geprüft; ein Wort mit Umlaut war für sich kein Treffer. | Für das Ziel `en` ist jedes Wort mit `ä ö ü Ä Ö Ü ß ẞ` ein Treffer, höchstens einer je Wort (`für` zählt als Stoppwort `fuer`, nicht zusätzlich). Gemeldet wird es so, wie es getippt wurde. | Umlaute sind das eindeutigste Merkmal deutscher Prosa. Kosten: zwei Eigennamen mit Umlaut in einer Zeile lehnen sie ab; Ausweg ist `[[commit.allow]]`. |
| 4 | `TestCheckConventionalHeaders`, `TestCheckConventionalDisabled` | Keine Header-Prüfung. | `[commit].conventional` (Vorgabe `true`) prüft die Kopfzeile über `internal/conventional`. | loomux verlangt Conventional Commits; ein Projekt ohne diese Regel schaltet den Schlüssel ab. |
| 5 | `check-commit-msg/refuse-german`, `…/trailer-in-line-1`, `…/non-latin-cjk`, `…/paragraph-break-span` | Ablehnung endete mit Exit 2, interne Fehler mit 1. | Ablehnung, Konfig- und Lesefehler enden mit 1; 2 bleibt dem falschen Aufruf. | Git unterscheidet nur 0 und nicht 0. loomux' bestehende Codes bleiben, das erspart einen Bruch (`release:major`). |
| 6 | `TestCheckCommitMsgCalibrate`, `TestCheckCommitMsgCalibrateErrors` | `--calibrate` ohne `--language` und ohne `[commit]` brach mit Exit 1 ab. | Die Sprache kommt aus dem Flag, sonst aus `[commit]`, sonst aus der Vorgabe `en`. | Folgt aus Eintrag 1: Wo eine Vorgabe gilt, gibt es keine ungewählte Regel mehr. |
| 7 | `TestCheckReadsInvalidUTF8` | Ungültiges UTF-8 endete in `UnicodeDecodeError` und Exit 1. | Go liest die Bytes und prüft den Text, der da ist. | Eine Nachricht, die git eben geschrieben hat, soll am Inhalt gemessen werden, nicht an ihrer Kodierung. |
| 8 | `TestScanNonLatinScripts` | Die Schrift kam aus dem Präfix von `unicodedata.name()`. | Die Schrift kommt aus `unicode.Scripts`. Han, Hiragana, Katakana und U+30FC sind ein Lauf `CJK`; Latin, Common und Inherited zählen nicht. Gemessene Folge: Zeichen, deren Name eine Schrift nennt, die Go als `Common` führt (`MODIFIER LETTER …`, `FEMININE ORDINAL INDICATOR`), sind in Go kein Lauf und in Python einer. | Go hat keine Zeichennamen. Die Tabellen sind die Quelle, aus der Pythons Namen selbst stammen. |
| 9 | `TestPolicySchemaErrors`, `TestPolicyAllowArrayAnyItemError`, `check-commit-msg/allow-broken-regex`, `…/allow-match-key` | `allow`-Muster waren Pythons `re`, mit Lookahead und Rückverweisen. | `allow`-Muster sind Go-RE2; was dort nicht übersetzt, verweigert das Laden der Konfiguration. | RE2 ist linear und hat kein Lookahead. Dieselbe Regel gilt schon für `[[policy.commands.rules]]`. |

**Nicht in dieser Liste, weil kein Unterschied:** Gos `\b`, `\s`, `\w` und
seine Zeilentrennung sind ASCII, Pythons sind Unicode. Nachgebaut sind daher
`splitLines` (alle Grenzen von `str.splitlines`, auch in `Subject`), die
Wortgrenze des Namenspartikels über `[^\p{L}\p{N}_]` und das Wortmuster
`[\p{L}\p{Nl}\p{No}]+`; danach urteilen beide gleich
(`TestScanSplitsLinesLikePython`, `TestScanSplitsCarriageReturns`,
`TestScanWordIncludesNonDecimalNumbers`,
`TestScanNameParticleWithNonASCIISurname`). Ohne den Nachbau lehnte
`von Müller` eine englische Zeile ab.

Ein kaputtes `[commit]`
blockiert in beiden Fassungen mit Exit 1 (`check-commit-msg/config-unknown-key`,
`…/config-missing-reason`). Pythons Kommentar in `commit/cli.py:18–19`, Exit 1
blockiere den Commit nicht, ist falsch — git bricht bei jedem Exit ungleich 0
ab —, das Verhalten selbst stimmt aber überein.
