# Search bench 2026-09-26-1200 — profile `full`

- qmd: 2.8.3
- models: embedding=embeddinggemma, query_expansion=qmd-query-expansion, rerank=qwen3-reranker
- system: windows/amd64, Test CPU
- loomux: 1.2.3
- search path: daemon
- indexed documents: 12
- question set: C:/sets/questions.yaml
- corpus: v1

These numbers measure **regression** against an artificial stock. They say whether the chain got worse than at the last stand — and they can carry **no decision**, neither about the architecture nor about a model. That is what the real question set is for.

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

## Latency

"Warm" here means the chain has already searched in this run: the latency runs after the quality pass.

- document read: `notes/alpha.md`
- query: `how does the index stay fresh`

| Operation | cold | warm median | min | max |
|---|---|---|---|---|
| catalog | 3 ms | 1 ms | 1 ms | 1 ms |
| read | 6 ms | 2 ms | 2 ms | 3 ms |
| keyword | 41 ms | 18 ms | 18 ms | 20 ms |
| vector | 812 ms | 10 ms | 9 ms | 12 ms |
| hybrid | 2411 ms | 656 ms | 640 ms | 701 ms |
