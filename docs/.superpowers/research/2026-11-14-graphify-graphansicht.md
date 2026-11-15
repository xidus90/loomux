# graphify: was die Graphansicht (Schritt 2) übernehmen kann

Stand 2026-11-14. Quelle: <https://github.com/Graphify-Labs/graphify>, v0.9.109
(Release vom 2026-11-14), Python, Apache-2.0, aktiv gepflegt (tägliche
Releases, rund 1 600 offene Issues). Gelesen: `graphify/exporters/html.py`,
`graphify/cluster.py`, `graphify/analyze.py`, `graphify/callflow_html.py`,
`docs/how-it-works.md`, `README.md`.

## Wie graphify den Graphen zeigt

- `graph.html`, eine einzelne Datei mit vis-network 9.1.6 von unpkg (die
  aktuelle Version ist 10.1.5). Die Daten sind als JSON eingebettet.
- Ein Kraftlayout, das nach 200 Iterationen anhält; danach ist die Physik aus.
- Knotengröße nach Grad, Farbe nach Community (Leiden, sonst Louvain).
- Seitenleiste: Suche, Infofeld zum angeklickten Knoten mit einer klickbaren
  Nachbarliste, Legende der Communities mit Haken zum Ausblenden und Zähler.
- **Knotengrenze 5 000:** Ist der Graph größer, zeichnet graphify einen
  Metagraphen. Darin ist jede Community ein Knoten, dessen Größe die Zahl
  ihrer Mitglieder ist, und die Kanten sind die Kanten zwischen Communities.
- Community-Namen ohne LLM: Jede Community heißt nach ihrem Mitglied mit dem
  höchsten Grad (`label_communities_by_hub`).
- Daneben: `path A B` (kürzester Weg, Konfidenz des schwächsten Schritts),
  God-Nodes, „überraschende“ Kanten zwischen Communities, Importzyklen,
  ein Mermaid-Export als statische Architekturseite.

## Abgleich mit loomux

| graphify | loomux heute | Vorschlag |
|---|---|---|
| Knotengrenze plus Metagraph | Code-Graph von loomux: 13 987 Knoten, 47 837 Kanten, 16 MB | übernehmen; zusammengefasst nach Paket oder Verzeichnis, aus `Path` deterministisch und ohne neuen Algorithmus |
| Größe nach Grad | `internal/code/pagerank` | übernehmen, aber mit PageRank statt Grad |
| Seitenleiste: Suche, Info, Nachbarn, Legende mit Haken | – | übernehmen |
| Konfidenz an jeder Kante | vier Konfidenzen in `model.Edge` | übernehmen, als Filter neben der Relation |
| Hub-Name für Gruppen | – | übernehmen, falls Gruppen keine Pakete sind |
| Physik nach Stabilisierung aus | – | übernehmen |
| `path A B` | `graph trace` / `blast` als CLI | später, eigene Scheibe |
| Leiden-Communities | – | erst einmal nicht; Pakete reichen als Gruppen. Später möglich über gonum (Louvain) |
| eingebettete Daten, Skript vom CDN | Insel-Regel: Daten über JSON-Endpunkt, Assets per `embed` | nicht übernehmen |
| Doku, PDF, Video im selben Graphen (LLM) | getrennte Graphen für Wiki und Code | nicht übernehmen |
| Mermaid-Callflow, Bericht, Fragenvorschläge | – | nicht in Schritt 2 |

## Bibliothek (Pflegestand laut npm am 2026-11-14)

vis-network 10.1.5, cytoscape 3.35.0, sigma 4.0.6 (WebGL), force-graph
1.52.0 und 3d-force-graph 1.80.2 werden alle gepflegt. Die Wahl gehört zu den
Ansätzen in der Spec von Schritt 2. Für 3D spricht das Paar force-graph und
3d-force-graph: Beide haben dieselbe API.
