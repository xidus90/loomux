---
type: Topic
title: Die Stufen und ihre Abnahme
description: Wie loomux in Stufen baut — die fünf Bedingungen einer fertigen Stufe, der Stand jeder Stufe und der Paritätsnachweis.
open_conflicts: 0
realization: in_progress
sources:
  - id: architektur-spec
    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M39G4J14CK311B66GRSAK7HQ
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
  - id: fusion-spec
    resource: brain://project/loomux/docs/.superpowers/specs/2026-09-14-loomux-fusion-design.md
    doc_id: 01M39G4J1486TZBN5SM5489MZS
    content_hash: "sha256:6fb778dd75759468370010fea63889b110dde93d03b7f0986894f5c371c65e00"
    revision: 3
---

loomux baut in **Stufen**, und **jede Stufe endet grün und wird einzeln
übergeben**, mit eigenem Plan und, sobald sie fertig ist, eigener
Paritätsakte. Die Herkunft: ultra-brain zerlegte seinen Bau in acht
„Scheiben" (0 bis 7) mit je einem Fertig-Kriterium, das ohne Codelektüre
prüfbar war; mit dem Umzug nach loomux ersetzen die Stufen der Fusions-Spec
diese Zerlegung.

## Eine Stufe ist fertig, wenn

1. alle übersetzten Fälle der Stufe grün sind oder freigegeben in der
   Abweichungsliste stehen,
2. die Coverage 100 % ist, jeder Ausschluss begründet,
3. die Mutationsrunde der Stufe gelaufen ist und ihre Überlebenden
   dokumentiert sind — ab Stufe 1b, weil `loomux dev mutants` dort entsteht;
   die Runde von 1b schließt die Entscheidungspakete aus 1a ein,
4. die Zielwerte der Stufe gemessen und in `docs/en/benchmarks.md` und
   `docs/de/benchmarks.md` eingetragen sind,
5. das loomux-Repo die Funktionen der Stufe selbst benutzt.

**Fehlt eine der fünf Bedingungen, ist die Stufe nicht fertig** — auch dann
nicht, wenn der Code steht.

## Die Stufen

Zwei Spuren laufen nebeneinander: die Fusion der beiden Altrepos und der
Code-Graph (Säule 3), der **neben** ihr statt hinter ihr gebaut wird, weil
keine seiner Stufen auf eine Fusions-Stufe wartet (G5a zog die bisher einzige
Abhängigkeit ein, `gotreesitter`). 1b, 2 und 3 sind
je in drei Teilstufen zerfallen, weil jede ihren eigenen Plan und ihre
eigene Abnahme brauchte; 3 lief parallel zu 2b und 2c. Stand nach `docs/de/migration.md`:

| Stufe | Stand | Inhalt |
|---|---|---|
| 1a | ✅ | Pilot: Repo-Gerüst, Tore, vereinter Wächter, Post-Edit-Lanes |
| 1b-1 | ✅ | die lesenden Brain-Befehle mit Parität zur Python-Referenz |
| 1b-2 | ✅ | `serve` mit MCP und die stdio-Brücke |
| 1b-3 | ✅ | Wiki und Dokumentation umgezogen |
| 2a | ✅ | die Prüfkette `[verify]`, `loomux check <profil>` |
| 2b | ✅ | commit-msg mit `--language`, `--calibrate`, `[commit]` |
| 2c | ✅ | Stop-Tor, `subagent-start`/`-stop`, Host-Adapter |
| 3a | ✅ | Erkennen: `reindex`, `embed`, `reconcile`, `area add` |
| 3b | ✅ | Entscheiden: `cases`, `case`, `approve` |
| 3c | ✅ | Pflegen: `brain check`, `lint --scope`, `wiki init\|types\|retype` |
| 4a-1 | 🚧 | Schema und `loomux config`; gebaut, Schritte des Menschen offen |
| 4a-2 | 🚧 | `loomux init`; gebaut, Schritte des Menschen offen |
| 4c-1 | 🚧 | das lokale Modell; gebaut, Selbstnutzung offen |
| 4c-2 | 🚧 | die Suchmessung `dev bench search`; gebaut, Alltagsqualität offen |
| 4d | offen | `convert` und `fetch` |
| 4e | offen | Umstellung der Wirte, eine Checkliste ohne Code |
| G1–G4b | ✅ | Rang, Blast-Radius, Extraktor, Abfrage, Navigation, Diff-Blast |
| G4c | offen | Stop-Hook mit Blast-Logik |
| G5a | ✅ | Extraktor-Schnittstelle, Tree-sitter-Kern auf `gotreesitter`, Python; abgenommen an `iam_backend` und `ultra-brain` |
| G5b–G5d | offen | TypeScript/TSX, GDScript, C++ |

Eine Teilstufe 4b gibt es nicht. `loomux migrate` fällt weg: den
Maschinenzustand hat die Selbstnutzung seit 3a schon umgezogen, die Wirte
richtet `init` neu ein. Flow und das Web-OS (W1–W5) sind Folgeprojekte mit
eigener Spec.

## Reihenfolge der offenen Stufen

Drei Regeln, der Reihe nach: **zuerst, was seine Abhängigkeiten schon
zulassen; dann, was loomux an sich selbst benutzt; dann die Größe.** Daraus
folgt G4c auf Priorität 2, Stufe 4 auf 3 (in sich 4a-1 → 4a-2 → 4c-1 → 4c-2 → 4d,
dann 4e), Flow auf 4, W1–W5 auf 5 und G5 auf 6; G5a hat der Nutzer am
2026-09-26 vorgezogen. Ohne Stufe 4 bleiben die
alten Repos im Dienst, und die Umstellung der Wirte braucht vorher einen
Remote für `brain-knowledge`.

## Der Paritätsnachweis

Was eine der beiden Altseiten konnte, kann loomux am Ende auch, **oder die
Abweichung steht mit Begründung und Freigabe in einer Liste**. Vor jeder Stufe
wird das Verhalten der alten Form von einem getaggten Commit aufgezeichnet —
Erfolgs-, Ablehnungs- und Fehlerfälle — und unter `testdata/cases/` als
Fallkorpus übersetzt; der Originalfall bleibt als Beleg daneben.

- **Daten** (`search`, `catalog`, `read`, `neighbors`, `status`): stdout
  exakt.
- **Meldungen** (Wächter, Hooks, `check`): Exit-Code und Dateiwelt exakt,
  Text frei.

Die Abweichungsliste liegt je Stufe unter `docs/.superpowers/parity/`; jeder
Eintrag nennt Fall, altes und neues Verhalten, Begründung und Freigabe.

## Bau- und Qualitätsregeln

TDD je Task; **100 % Coverage je Funktion**, jeder Ausschluss mit Begründung
im Code. Umgezogene Go-Pakete bringen ihre Tests mit und werden beim Umzug auf
100 % gehoben oder bekommen begründete Ausschlüsse; neuer Code entsteht
test-first. Golden-Dateien halten die Antwortformen von Claude und
Antigravity fest; `loomux dev mutants` fährt je Stufe eine Mutationsrunde
über die Entscheidungspakete, und `loomux dev bench hooks` misst die
Zielwerte jeder Stufe mit demselben Werkzeug. **Externe Programme** — qmd,
Ollama, `pdftotext`, `yt-dlp`, Git-Remotes — werden an der Prozessgrenze
durch Stubs ersetzt, deren Antworten aus einem echten, einmal
aufgezeichneten Lauf stammen.

Die Sprache ist Go ≥ 1.25; Python bleibt nur in den Projekten, die loomux
prüft, nicht im Produkt. Die Tore von loomux selbst fahren
`loomux check precommit` — `gofmt`, `go vet`, `go test` mit 100 % Coverage.

Siehe auch [Suche, Profile und Messwerte](suche-und-profile.md),
[Die Wiki-Schicht](wiki-schicht.md) und
[Grundsätze und Vertrauenskette](architektur-grundsaetze.md); Quellen sind
die Fusions-Spec und, für die Herkunft, das Architektur-Design ultra-brain.
