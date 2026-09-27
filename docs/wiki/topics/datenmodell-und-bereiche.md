---
type: Topic
title: Datenmodell und Bereiche
description: Drei Orte für Daten, Manifest und zentrale Registrierung, Identitätsregister, geteilte Bereiche und Scoping.
open_conflicts: 0
realization: implemented
implemented_in: 6b610d7
sources:
  - id: architektur-spec
    resource: brain://project/loomux/docs/.superpowers/specs-ub/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M39G4J14CK311B66GRSAK7HQ
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
---

## Die tragende Aufteilung

**Verdichtetes liegt an einem Ort, Rohquellen liegen dort, wo sie entstehen.**

| Ort | Inhalt | Versioniert |
|---|---|---|
| Wissens-Vault | eigene Notizen plus **sämtliche** Wikis plus alle Prüffälle | eigenes Git-Repo |
| Code-Repos | **nur** Rohquellen: README, CHANGELOG, ADRs, Spec, Plan, Manifest | vorhandenes Repo |
| Zustandsverzeichnis | Index, Modelle, Zustandsdatenbank | **nie** |

Vier Gründe stehen hinter der Zusammenlegung aller Wikis im Vault: ein Ort zum
Lesen; Code-Repos bleiben veröffentlichbar (ein Wiki im Repo ginge bei einer
Veröffentlichung mit — samt offener Konflikte und Prüfprotokoll); keine
Wiki-Commits in der Code-Historie; und der vermeintliche Versionsvorteil des
Repos existiert gar nicht, weil der Abgleich ausdrücklich nicht gegen
Code-Inhalte läuft.

**Es gibt genau eine Betriebsart.** Bis zum Umzug am 2026-09-16 war ein Wiki im
Code-Repo keine Option, auch nicht als Schalter. Daraus folgt eine Eigenschaft, die im Code sichtbar bleiben
muss: **Wiki-Ort und Rohquellen-Ort sind zwei unabhängige Eingaben; der eine
wird nirgends aus dem anderen abgeleitet.**

## Der Pfad entscheidet, nicht das Frontmatter

Ob eine Datei zur KI-Schicht gehört, wird **ausschließlich** über den Pfad
entschieden. „Rohquellen bleiben frei" heißt *unbeschränkt*, nicht *ohne
Frontmatter*: Ein gewachsener Obsidian-Vault steckt voller handgeschriebener
Frontmatter-Blöcke, und wer Rohquellen am fehlenden `type` erkennen wollte,
bricht am Altbestand.

## Manifest und Registrierung

Das Manifest `.brain.toml` beschreibt **nur, was der eigene Bereich mitbringt** —
Scope, Ein- und Ausschlüsse, Datenschutzmodus, `readonly`. Wo etwas liegt, weiß
allein die zentrale Registrierung im Zustandsverzeichnis; sie ist die einzige
Stelle, die Bereichsnamen auf Pfade abbildet.

`.brain.toml` ist der Name bis zum Umzug am 2026-09-16. loomux liest
`.loomux/config.toml` vor den beiden Altnamen `.ultra-brain/config.toml` und
`.brain.toml` — bis Stufe 4 und nur, wenn die neue Datei eine `[area]`-Tabelle
trägt, sonst gilt weiter das Altmanifest —, hält die Registrierung unter
`LOOMUX_STATE_DIR` (`%LOCALAPPDATA%\loomux`) und lässt Artefakte und
Reconcile-Stempel bis Stufe 3 unter `LOOMUX_LEGACY_BRAIN_DIR`.

Drei Feinheiten mit Messgeschichte:

- **`never` wirkt doppelt** — solche Pfade werden nicht indexiert *und* `read`
  verweigert sie zusätzlich. Die frühere Fassung versprach beides zugleich in
  unvereinbarer Form: Was im Index liegt, kommt als Trefferausschnitt zurück.
- **`unsearched` trennt „indiziert" von „durchsucht".** Spec und Plan eines
  Projekts stehen im Katalog und beantworten `read`, sind aber nicht Gegenstand
  der Bedeutungssuche. Ohne diese Angabe stünden 61 Dokumente im Index, die die
  Suche nie zurückgeben kann, und `status` meldete sie als Rückstand.
- **Die eigenen Artefaktnamen fallen nur im eigenen Baum aus dem Index.** Der
  pauschale Ausschluss kostete in `iam_wiki` **22 von Hand geschriebene,
  eingecheckte `index.md`** — in einem Wiki genau die Navigationsebene.

## Identitätsregister

`_identities.tsv` je Bereich, im versionierten Baum: `doc_id`, Pfad,
Inhaltshash, Revision. Es ist **Wahrheit, nicht Ableitung** — die einzige
Ausnahme neben dem Markdown selbst — und wird ausschließlich von Code
geschrieben, nie von einem Modell.

- **Umbenennungen** erkennt der Abgleich am Inhaltshash: verschwundener Pfad
  plus neuer Pfad mit gleichem Hash gilt als Umbenennung, die `doc_id` bleibt.
- **Gleichzeitiges Umbenennen und Ändern** wird nicht geraten, sondern als
  „Quelle fehlt" plus „neue Quelle" vorgelegt.
- **`content_hash` wird über auf LF normalisierten Inhalt gebildet.** Sonst
  meldete die Wartungsschicht nach einem Klon auf einem zweiten Rechner *jede
  einzelne Seite* als geändert.
- **`revision` wächst um 1 je festgestellter Inhaltsänderung**, nicht je
  Ereignis: Der Abgleich sieht nur den Unterschied zweier Stände und behauptet
  nichts über den Weg dazwischen.

## Drei Bereichsfamilien und die Leserichtung

`knowledge`, `project/<name>`, `engineering/<name>`. Der Kern kennt nur
„Bereiche"; die Familie steckt allein im Namen. Die dritte Familie schließt eine
Lücke: Programmierwissen entsteht in einem Projekt, gilt aber für alle.

**Projekte lesen geteilte Bereiche. Geteilte Bereiche lesen keine Projekte.**
Der einzige Weg hinein ist die Beförderung durch menschliche Hand, bei der ein
Modell eine projektfreie Fassung vorschlägt. Dabei wird **nicht kopiert** — die
geteilte Seite ist die Wahrheit, das Projekt verweist. Diese Richtung setzt
der Lint durch: `loomux lint` meldet mit `wrong-direction` einen Fehler, wenn
eine Seite in einem geteilten Bereich eine Projektseite als Quelle nennt.

Verweise tragen Pfad **und** `doc_id` und **heilen sich damit selbst**: Läuft
ein Pfad ins Leere, sucht der Kern die Kennung über alle Bereiche. Ein Umschnitt
der Bereichsgrenzen ist damit eine Verschiebeoperation statt einer Reparatur —
und genau das entschärft, dass die richtigen Grenzen heute noch nicht bekannt
sind.

## Scoping

Jede Anfrage trifft eine **Liste** von Bereichen und eine Schicht (`raw`,
`wiki`, `both`). Technisch gilt: **eine Sammlung je Bereich und Schicht.** Der
Scope wählt Sammlungen aus, statt Treffer nachträglich auszusortieren — was
nicht in der Sammlung liegt, kann nicht versehentlich in einer Antwort
auftauchen. Es bleibt eine Vorgabe, keine Sperre: Ein Modell darf bewusst
weiten, soll das Weiten aber benennen.

Siehe auch [Datenschutz und Kanäle](datenschutz-und-kanaele.md),
[Die Wiki-Schicht](wiki-schicht.md) und, wie Registry und Manifest beim
Schreiben gelesen werden, [Die Schreibschranke](schreibschranke.md); Quelle ist
Architektur-Design ultra-brain.
