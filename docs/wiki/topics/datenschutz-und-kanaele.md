---
type: Topic
title: Datenschutz und Kanäle
description: Wie local_only, manual_cloud und never wirken — und warum der Kanal ein Parameter des Kerns ist.
open_conflicts: 0
realization: implemented
implemented_in: 9d23a19
sources:
  - id: konfiguration
    resource: brain://project/loomux/docs/de/configuration.md
    doc_id: 01M47W91R50RT15H2ES8RV7TJT
    content_hash: "sha256:676e0456e1ac9b8df14b826e1f2888ad304bf0f5f7bb9b4b00d9b2f1ddb1bb09"
    revision: 16
  - id: cli-referenz
    resource: brain://project/loomux/docs/de/cli-reference.md
    doc_id: 01M47W91R5FRSPN6BE643656ZE
    content_hash: "sha256:7baaad75eeb7fb3e320ff9d0e8df54d36bcc7a762a0389627bee522105959bd1"
    revision: 14
---

## Der Fund, der die Regel erzwang

Die Datenschutzgatter wirkten ursprünglich nur auf dem Wartungspfad. Damit war
`local_only` im Alltag **wirkungslos**: Claude — ein Cloud-Modell — hätte
denselben Inhalt über `search` und `read` vollständig lesen können. Seitdem
setzt der Kern den Modus **auf jedem ausgehenden Pfad** durch.

| Modus | Kanal `cloud` | Kanal `local` |
|---|---|---|
| `local_only` | **Bereich ist unsichtbar** — keine Treffer, keine Inhalte, kein Katalogeintrag | voller Zugriff |
| `manual_cloud` | voller Zugriff; das Gatter greift beim Vorschlaggeber | voller Zugriff |
| `automatic_cloud` | voller Zugriff | voller Zugriff |

`local_only` bedeutet damit wörtlich, was es sagt. Der Preis ist bewusst
gewählt: In einem solchen Bereich kann Claude Code nicht arbeiten — dort wird
mit CLI, Oberfläche und lokalem Modell gearbeitet.

## `never` gegen `local_only`

`never` schließt einzelne **Pfade** auf jedem Kanal von loomux aus, auch
innerhalb eines sonst offenen Bereichs. `local_only` schließt einen **ganzen
Bereich** gegenüber der Cloud.

Beide begrenzen nur, was die Kanäle von loomux herausgeben (Suche, Lesen, die
MCP-Werkzeuge). Die Befehle `loomux graph` auf der Kommandozeile kennen keinen
Kanal und filtern keinen `never`-Pfad: `graph grep` gibt Zeilen einer solchen
Datei aus, `graph ask --source` ihren Text. Was ein Agent mit den Werkzeugen seines Hosts liest — das
`Read`-Werkzeug, ein `cat` in der Shell —, erreicht sein Modell trotzdem: Der
Wächter beurteilt, was ein Agent schreibt und ausführt, nicht was er liest,
`never` ist keine Regel von ihm, und `Read` gehört nicht zu den Werkzeugen, für
die `loomux init` einen Hook einträgt. Die Konfigurationsreferenz sagt das
ebenso.

## Verschachtelung hebt `local_only` nicht auf

Bereiche können ineinander liegen: Das Wiki eines Hubs kann die Wikis mehrerer
Projekte enthalten. Solange nur der Scope entschied, las der Cloud-Kanal über
den umschließenden Bereich die Seiten eines `local_only`-Projekts, und
`reconcile` eröffnete für dieselbe Seite einen zweiten Fall im Hub, dessen
Paket den Diff der `local_only`-Quelle trug (gefunden 2026-09-29). Seitdem
gilt: Jeder Pfad im Wiki oder Quellbaum eines verborgenen Bereichs bleibt auf
dem Cloud-Kanal verborgen, gleich über welchen Scope er erreicht wird — `read`
antwortet wie bei einer fehlenden Datei, Treffer, Katalogzeilen und Nachbarn
darunter fallen weg. Ob ein Pfad darin liegt, entscheidet das Dateisystem,
nicht die Schreibweise. Und eine Seite gehört dem tiefsten Bereich, dessen
Wiki sie enthält; nur dort entsteht ihr Fall.

## Der Kanal gehört dem Kern

Jede der fünf Werkzeugfunktionen nimmt den Kanal (`local` oder `cloud`) entgegen,
und das Gatter greift dort — **nicht im MCP-Adapter**. Sonst könnte ein zweiter
Klient es vergessen, und der Beweis hinge an der Sorgfalt des jeweils äußersten
Randes. So erbt jeder künftige Zugang die Zusage, statt sie nachzubauen; auf der
CLI ist der Kanal setzbar, damit der Nachweis ohne MCP-Prozess führbar ist.

**Prüfbar ist das ohne Modell:** ein Bereich auf `local_only`, ein `search` über
`all` auf dem Cloud-Kanal — die Treffer dürfen daraus nichts enthalten, und
`read` auf einen bekannten Pfad daraus muss verweigern. Der Bereich wird dabei
gar nicht erst befragt, statt seine Treffer nachträglich zu filtern. Dieser
Nachweis ist Fertig-Kriterium von Scheibe 2a und wird in 2c über den echten
MCP-Kanal wiederholt ([Die Stufen und ihre Abnahme](scheiben-und-abnahme.md)).

## Was es über MCP nicht gibt

Kein `approve` und kein `ingest`. Freigeben ist eine menschliche Entscheidung;
Ingest braucht kein Werkzeug, weil Claude Wiki-Seiten als normale Dateien
schreibt. **Ein Sprachmodell kann über diese Schnittstelle nichts am Wissen
ändern.**

Quellen: Konfigurations- und CLI-Referenz der Nutzerdoku; der
Ort der Bereiche steht unter [Datenmodell und Bereiche](datenmodell-und-bereiche.md).
