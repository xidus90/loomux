# Search bench 2026-09-26-1200 — profile `fast`

- qmd: 2.8.3
- models: embedding=embeddinggemma, query_expansion=qmd-query-expansion, rerank=qwen3-reranker
- qmd backbone: unknown
- system: windows/amd64, Test CPU
- loomux: 1.2.3
- search path: daemon
- scope: knowledge
- indexed documents: 12
- question set: C:/sets/questions.yaml

This run is **purely vectorial** (`qmd vsearch`) and therefore measures **both** the embedding model's share of the language bridge and whether `fast` carries across languages.

## Hit quality

| Sort | Hits |
|---|---|
| exakt | 1/1 |
| umschreibung | 0/1 |
| gemischt | 0/0 |
| sprachuebergreifend | 0/1 |
| total | 1/3 |

Median hits: 12 ms
Median misses: 36 ms

## Misses

- c02 (umschreibung): rank 5 — wie heißt das Ding | mit dem Strich
- c03 (sprachuebergreifend): not found — how does the index stay fresh

## Findings

Messages of the search chain during the run. They belong beside the numbers: a miss with a finding may be no measurement at all but a silent failure of the engine.

- qmd: 1 document skipped (unreadable)
