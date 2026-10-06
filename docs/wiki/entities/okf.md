---
type: Entity
title: Open Knowledge Format (OKF)
description: Das Format, dem die Wiki-Bundles folgen — und die Stelle, an der dieses Projekt es erweitert.
open_conflicts: 0
sources:
  - id: cli-referenz
    resource: brain://project/loomux/docs/de/cli-reference.md
    doc_id: 01M47W91R5FRSPN6BE643656ZE
    content_hash: "sha256:2e0f9455c00bbd2cd943d2eed65463448522d33232f032358addb56c2bae1255"
    revision: 1
---

Fremdes Format in Version 0.2, übernommen als Muster für die
Verdichtungsschicht: Frontmatter-Familien, `index.md` und `log.md` auf jeder
Ebene, die Actor-Konvention und die Trust-Tiers.

**Die Reichweite ist bewusst begrenzt:** Das Wiki ist ein strenges OKF-Bundle,
Rohquellen bleiben frei. Genutzt werden `type` (Pflicht), `title`,
`description`, `resource`, `tags`, `sources`, `generated`, `verified`, `status`
und `stale_after`.

## Wo dieses Projekt erweitert

OKF erlaubt produzenteneigene Erweiterungen ausdrücklich. Drei werden genutzt:
die Identitätsfelder in `sources[]` (`doc_id`, `content_hash`, `revision`), die
Konfliktzählung `open_conflicts`, und `realization` samt `implemented_in` auf
Projektseiten. Letztere schließen die Lücke zwischen der Reife der Seite und dem
Umsetzungsgrad des Beschriebenen
([Die Wiki-Schicht](../topics/wiki-schicht.md)).

Bereichsübergreifende Verweise tragen Pfad und Kennung als absolute URI und sind
damit von OKF gedeckt
([Datenmodell und Bereiche](../topics/datenmodell-und-bereiche.md)).
