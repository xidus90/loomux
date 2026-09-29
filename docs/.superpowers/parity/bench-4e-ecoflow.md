# ecoflow

3 verglichen, 1 schneller, 2 langsamer, 0 unklar, 8 neu, 0 weggefallen

## Verglichen

| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| status | 49.5 ms | 6308.6 ms | 0.01× | 67.6 ms | 5196.1 ms | 0.01× | 43.0 ms | 5238.4 ms | [0] | [0] |
| search (fast) | 206.4 ms | 2739.9 ms | 0.08× | 250.7 ms | 145.6 ms | 1.72× | 236.0 ms | 121.7 ms | [0] | [0] |
| wiki lint | 47.0 ms | 156.4 ms | 0.30× | 49.5 ms | 162.4 ms | 0.30× | 48.5 ms | 158.3 ms | [0] | [0] |

## Neu

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| SessionStart | 213.5 ms | 13.3 ms | 13.0 ms |
| PreToolUse (Edit on README.md) | 27.5 ms | 45.2 ms | 19.5 ms |
| PostToolUse (Edit on README.md) | 22.0 ms | 45.6 ms | 21.8 ms |
| SubagentStart | 292.5 ms | 678.4 ms | 568.9 ms |
| SubagentStop | 292.5 ms | 13.0 ms | 13.0 ms |
| Stop | 806.3 ms | 1042.5 ms | 827.1 ms |
| graph stats | 123.3 ms | 218.0 ms | 134.0 ms |
| graph check | 28.0 ms | 24.0 ms | 23.5 ms |

## Weggefallen

keine
