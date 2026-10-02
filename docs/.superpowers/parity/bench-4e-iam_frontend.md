# iam_frontend

Gemessen am 2026-10-02 mit der installierten Binary `loomux 6.1.0 (beta)`, `-n 5` (ein
Kaltlauf, fünf Warmläufe). Alt-Seite vom 2026-09-29 mit `ulguard`, `brain` (`~/go/bin`) und
dem vendorten Python, gemessen von `bin/loomux.exe` `0.0.0-dev`. Erläuterung der Exit-Codes:
`stufe-4e.md`, Messung 10.

Umgebung: Windows 11 Pro 10.0.26200, AMD64 Family 26 Model 68; qmd-Dienst auf Port 8765
lief (wie bei der Baseline), dazu `loomux serve --foreground` und zehn `loomux mcp` offener
Sitzungen; ein fremder `strata.exe --serve` hielt rund 45 GB Arbeitsspeicher, CPU-Last vor dem
ersten Lauf 24 %. Gemessen seriell, ein Ziel nach dem anderen, 12:35–12:40 Ortszeit.

Kopf der Nachher-Messung (`benchHead`):

```
# loomux dev bench hooks — 2026-10-02-1037

- system: windows/amd64, AMD64 Family 26 Model 68 Stepping 0, AuthenticAMD
- go: go1.27.0
- loomux: 6.1.0
```

9 verglichen, 8 schneller, 1 langsamer, 0 unklar, 2 neu, 1 weggefallen

## Verglichen

| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| SessionStart | 558.0 ms | 135.6 ms | 4.12× | 411.0 ms | 82.3 ms | 4.99× | 221.4 ms | 81.0 ms | [0] | [0] |
| PreToolUse (Edit on AGENTS.md) | 136.1 ms | 16.9 ms | 8.04× | 41.4 ms | 13.3 ms | 3.11× | 41.0 ms | 13.6 ms | [0] | [2] |
| PostToolUse (Edit on AGENTS.md) | 58.0 ms | 19.0 ms | 3.05× | 47.1 ms | 14.9 ms | 3.16× | 46.9 ms | 15.0 ms | [0] | [0] |
| SubagentStart | 1705.2 ms | 1378.0 ms | 1.24× | 2033.0 ms | 1101.4 ms | 1.85× | 2135.1 ms | 1098.1 ms | [0] | [0] |
| SubagentStop | 1989.5 ms | 1036.6 ms | 1.92× | 1909.6 ms | 9.6 ms | 198.48× | 1898.3 ms | 9.6 ms | [0] | [0] |
| Stop | 346.9 ms | 27559.8 ms | 0.01× | 239.8 ms | 182.0 ms | 1.32× | 232.8 ms | 178.7 ms | [1] | [0] |
| commit-msg | 105.9 ms | 39.0 ms | 2.72× | 120.6 ms | 30.2 ms | 3.99× | 103.7 ms | 29.6 ms | [0] | [0] |
| status | 119.9 ms | 3173.5 ms | 0.04× | 118.1 ms | 3177.2 ms | 0.04× | 101.0 ms | 3135.5 ms | [0] | [0] |
| search (fast) | 241.3 ms | 168.0 ms | 1.44× | 253.3 ms | 67.8 ms | 3.74× | 249.0 ms | 67.6 ms | [0] | [0] |

## Neu

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| graph stats | 10.6 ms | 8.0 ms | 8.0 ms |
| graph check | 15.0 ms | 13.6 ms | 13.5 ms |

## Weggefallen

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| python entry (ultraloom --help) | 391.3 ms | 230.3 ms | 223.4 ms |

## Herkunft

Alt-Seite: `before-merged.json`, die erste Baseline (`baseline/bench-2026-09-29-1903`), darin
`SubagentStart`, `SubagentStop` und `Stop` ganz aus der Nachmessung ohne Last
(`baseline-2/bench-2026-09-29-1945`) übernommen.
