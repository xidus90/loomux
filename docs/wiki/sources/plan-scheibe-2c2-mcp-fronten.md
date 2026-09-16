---
type: Source
title: Plan Scheibe 2c-2 — MCP-Fronten
description: Der Kanal wird zur Adresse statt zum Argument — und was der Plan bewusst nicht enthält.
open_conflicts: 0
realization: implemented
implemented_in: cba7c8b
sources:
  - id: plan-scheibe-2c2
    resource: brain://project/loomux/docs/.superpowers/plans-ub/2026-08-22-scheibe-2c2-mcp-fronten.md
    doc_id: 01M0QGS594D9D3M90MV6A1WFX9
    content_hash: "sha256:179c1cc297ff0f27e92be6b41d84f1bed023cb975c6af624d49b5befc3c9c0aa"
    revision: 1
---

Eine stdio-Front für MCP-Wirte auf den Daemon — **und der Kanal wird dabei zur
Adresse statt zum Argument.**

## Die tragende Idee

Der Daemon lauscht auf **zwei Adressen statt auf einer**, jede fest an ihren
Kanal gebunden. Der Kanal steht damit in keiner Nachricht, sondern **ist die
Pipe, auf der die Verbindung ankam**. Ein Klient kann ihn nicht mehr behaupten;
die Gegenprobe der Messung schickt ihn ausdrücklich mit und wird unverändert
verweigert.

Die Front selbst ist ein dünner Umleiter, der Handschlag und Werkzeugliste aus
eigener Kraft beantwortet, **damit kein Wirt in seinen Start-Timeout läuft** —
sie wartet nicht auf die Wärme darunter.

## Zwei Regeln der Bauform

- **Drei Module bleiben unberührt.** *Muss eines davon geändert werden, ist die
  Naht nicht die, für die wir sie halten — anhalten und die Spec berichtigen,
  nicht umgehen.*
- **Zwei Fehlersorten dürfen nie verschmelzen:** Werkzeugfehler sind ein
  Fehlerergebnis des Werkzeugs, Transportfehler sind Protokollfehler.

## Wie gemessen wird

- **Jede Zahl, die eine Rate ist, nennt ihre Läufe.** Sechs Läufe ergaben in der
  Vorgängerscheibe einmal 6/6 und hätten einen echten Fund beerdigt; fünfzehn
  zeigten 3/15.
- **Vor jedem rückgratberührenden Lauf wird der Daemon gestoppt**, sonst bekommt
  die Messung das Rückgrat eines laufenden Daemons statt des angeforderten.
- **Wird der Preis der Umleitung zweistellig, ist das ein Befund, keine
  Fußnote** — dann kommt er als offener Punkt in die Spec, nicht in eine
  Wegdiskussion.
- **Weicht etwas von der Erwartung ab, wird die Spec berichtigt, nicht die
  Messung gerundet.**
- Der Nachweis, dass der Code keine Protokollrevision kennt, wird nicht behauptet,
  sondern gesucht: Ein Durchsuchen der Quellen nach den Revisionsdaten darf außer
  Kommentaren nichts finden.
- Eine **Falle der Plattform** steht ausdrücklich im Plan: Das Beenden des
  Prozesses trifft den Batch-Shim, nicht den Node-Prozess mit dem Modell —
  danach mit der Prozessliste prüfen, dass kein verwaister Daemon steht. Die
  Vorgängerscheibe hatte nach ihrem ersten Messlauf vier.

Vor der Installation eines fremden Werkzeugs auf dem Rechner des Nutzers wird
**ausdrücklich gefragt** — es ist eine Installation auf seinem System, keine
Projektabhängigkeit.

## Was der Plan bewusst nicht enthält

Die HTTP-Front (eigene Scheibe), den kompilierten Mini-Client (*die Zahl steht,
die Entscheidung nicht*), die Untersuchung des verfehlten Suchbudgets
(*eigene Untersuchung, keine stille Anpassung*) und den Fehlerbericht an das
fremde Projekt — eine Veröffentlichung, und die liegt beim Menschen.

Siehe [Datenschutz und Kanäle](../topics/datenschutz-und-kanaele.md) und
[Der brain-Daemon](../entities/brain-daemon.md).
