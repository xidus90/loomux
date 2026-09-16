---
type: Entity
title: Open Knowledge Format (OKF)
description: Das Format, dem die Wiki-Bundles folgen — und die Stelle, an der dieses Projekt es erweitert.
open_conflicts: 0
sources:
  - id: architektur-spec
    resource: brain://project/ultra-brain/docs/.superpowers/specs/2026-08-18-ultra-brain-architektur-design.md
    doc_id: 01M0QGS594F2KCWTWK9XV07M05
    content_hash: "sha256:c82f573ca64a6b8c53d0f158cb83847865cc8b32c99f710d37c6050fb11ac3ff"
    revision: 3
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
