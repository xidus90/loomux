# Protokoll neueste zuerst

Ziel: `log.md` folgt OKF §9 — „a flat list of date-grouped entries, newest
first“, je Tag eine Überschrift `## YYYY-MM-DD`. Heute hängt `loomux approve`
jede Zeile ans Ende an (`Append`, wie die Python-Referenz), und das Protokoll
von `project/loomux` hat keine Datumsüberschriften.

## Entscheidungen

- Die Zeile selbst bleibt die der Referenz (`LogLine`, samt Datum). Nur ihr
  Platz ändert sich. So bleiben die Goldens von `LogLine` gültig, und jede
  Zeile trägt ihr Datum auch außerhalb ihrer Gruppe.
- Neu ist `LogInsert(existing, now, line)`: Die Zeile kommt vor den ersten
  Eintrag — die erste Zeile, die mit `## `, `- ` oder `* ` beginnt. Ist das
  die Überschrift des heutigen Tages (UTC), wird die Zeile deren erster
  Eintrag; sonst entsteht davor eine neue Tagesgruppe. Ein Protokoll ohne
  Eintrag bekommt die Gruppe hinten angehängt, hinter Titel und Vorspann.
  Ein Protokoll im alten Format ohne Überschriften bekommt die neue Gruppe
  vor die alten Zeilen.
- `audit.md` bleibt bei `Append`: OKF regelt die Datei nicht.
- Die Aufzeichnungen unter `testdata/cases/3b` bleiben unberührt (Belege der
  Referenz). `approve/success`, `amend`, `rebase` und `no-repo` weichen in
  `repo-a/wiki/log.md` ab; `expected3b` nennt die Abweichung, und der Test
  hält die geschriebene Datei gegen die OKF-Form.

## Schritte

1. Rot: Tabellentest für `LogInsert` in `protocol_test.go`.
2. Grün: `LogInsert` in `protocol.go`; `place.appendProtocol` auf einen
   gemeinsamen Lese- und Schreibweg `writeProtocol` gestellt, `approve`
   ruft für `log.md` `LogInsert`.
3. `approve_test.go`: erwartetes Protokoll mit Tagesüberschrift.
4. `cases_3b_test.go`: Abweichung in `log.md` erwartet und geprüft;
   `parity/stufe-3b.md` nennt sie.
5. `docs/wiki/log.md` einmal in die OKF-Form gebracht, der Vorspann sagt
   „neueste zuerst“. `bundleLog` bleibt, wie er ist: `wiki init` ist in
   Stufe 3c gegen die Referenz aufgezeichnet, und der Vorspann trägt keine
   Regel, die `LogInsert` braucht.
6. Tor: `sh ci/gate.sh`.
