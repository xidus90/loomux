# space

10 verglichen, 8 schneller, 2 langsamer, 0 unklar, 2 neu, 1 weggefallen

## Verglichen

| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| SessionStart | 683.9 ms | 553.7 ms | 1.24× | 552.6 ms | 253.0 ms | 2.18× | 456.8 ms | 251.4 ms | [0] | [0 0] |
| PreToolUse (Edit on AGENTS.md) | 54.0 ms | 12.4 ms | 4.35× | 48.8 ms | 9.8 ms | 4.98× | 48.4 ms | 9.6 ms | [0 0] | [0] |
| PostToolUse (Edit on AGENTS.md) | 52.4 ms | 18.8 ms | 2.79× | 67.5 ms | 13.7 ms | 4.92× | 52.2 ms | 13.2 ms | [0] | [0] |
| SubagentStart | 2093.3 ms | 1735.5 ms | 1.21× | 1837.0 ms | 1127.5 ms | 1.63× | 1765.4 ms | 1112.6 ms | [0] | [0] |
| SubagentStop | 1935.8 ms | 1088.9 ms | 1.78× | 1866.0 ms | 8.1 ms | 230.60× | 1820.6 ms | 8.0 ms | [0] | [0] |
| Stop | 618.9 ms | 1099.8 ms | 0.56× | 664.2 ms | 336.7 ms | 1.97× | 608.8 ms | 408.3 ms | [1 1] | [2] |
| commit-msg | 603.2 ms | 355.8 ms | 1.70× | 587.6 ms | 261.8 ms | 2.24× | 488.1 ms | 262.1 ms | [0] | [0] |
| status | 201.6 ms | 2336.0 ms | 0.09× | 206.7 ms | 2143.4 ms | 0.10× | 208.1 ms | 2113.9 ms | [0] | [0] |
| search (fast) | 267.7 ms | 11506.8 ms | 0.02× | 232.4 ms | 76.1 ms | 3.05× | 197.6 ms | 76.1 ms | [0] | [0] |
| wiki lint | 45.5 ms | 173.6 ms | 0.26× | 47.4 ms | 169.8 ms | 0.28× | 47.8 ms | 169.0 ms | [0] | [1] |

## Neu

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| graph stats | 10.0 ms | 7.5 ms | 7.5 ms |
| graph check | 84.5 ms | 37.0 ms | 36.5 ms |

## Weggefallen

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| python entry (ultraloom --help) | 247.1 ms | 119.1 ms | 116.9 ms |

## Herkunft

Die Alt-Seite ist vom 2026-09-29, die Neu-Seite vom 2026-09-30. Die Alt-Werte
von `SubagentStart`, `SubagentStop` und `Stop` stammen aus der Nachmessung
(`baseline-2`, 21:31), alle übrigen aus der ersten Baseline (21:02): in der
ersten trugen die Nutzlasten keine `session_id`, und die drei Hooks lehnten
sie ab, bevor sie arbeiteten. Die Erklärung der Exit-Codes steht in
`stufe-4e.md`, Messung 9.
