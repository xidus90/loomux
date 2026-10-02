# iam_workers

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
# loomux dev bench hooks — 2026-10-02-1038

- system: windows/amd64, AMD64 Family 26 Model 68 Stepping 0, AuthenticAMD
- go: go1.27.0
- loomux: 6.1.0
```

8 verglichen, 7 schneller, 1 langsamer, 0 unklar, 3 neu, 1 weggefallen

## Verglichen

| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| SessionStart | 433.3 ms | 113.0 ms | 3.83× | 160.6 ms | 75.0 ms | 2.14× | 147.6 ms | 75.7 ms | [0] | [0] |
| PreToolUse (Edit on AGENTS.md) | 46.5 ms | 13.5 ms | 3.44× | 65.6 ms | 12.1 ms | 5.42× | 43.0 ms | 12.0 ms | [0] | [2] |
| PostToolUse (Edit on AGENTS.md) | 49.5 ms | 16.5 ms | 3.00× | 68.3 ms | 12.2 ms | 5.58× | 45.5 ms | 12.0 ms | [0] | [0] |
| SubagentStart | 2007.9 ms | 1218.9 ms | 1.65× | 1829.0 ms | 1167.3 ms | 1.57× | 1844.9 ms | 1182.1 ms | [0] | [0] |
| SubagentStop | 1794.4 ms | 1072.1 ms | 1.67× | 1840.9 ms | 8.3 ms | 222.76× | 1745.2 ms | 8.5 ms | [0] | [0] |
| commit-msg | 103.9 ms | 31.6 ms | 3.29× | 150.8 ms | 26.3 ms | 5.74× | 101.5 ms | 25.7 ms | [0] | [0] |
| status | 225.4 ms | 3602.2 ms | 0.06× | 120.6 ms | 3266.2 ms | 0.04× | 108.0 ms | 3307.8 ms | [0] | [0] |
| search (fast) | 259.6 ms | 92.8 ms | 2.80× | 251.4 ms | 73.9 ms | 3.40× | 206.6 ms | 70.7 ms | [0] | [0] |

## Neu

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| Stop | 143.8 ms | 160.0 ms | 164.6 ms |
| graph stats | 11.6 ms | 8.8 ms | 9.0 ms |
| graph check | 103.4 ms | 13.8 ms | 13.5 ms |

## Weggefallen

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| python entry (ultraloom --help) | 255.8 ms | 151.6 ms | 135.5 ms |

## Herkunft

Alt-Seite: `before-merged.json`, die erste Baseline (`baseline/bench-2026-09-29-1904`), darin
`SubagentStart` und `SubagentStop` ganz aus der Nachmessung ohne Last
(`baseline-2/bench-2026-09-29-1945`) übernommen. Der Alt-Stand hatte keinen Stop-Hook.
