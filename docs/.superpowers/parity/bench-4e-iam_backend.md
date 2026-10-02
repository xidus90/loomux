# iam_backend

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
# loomux dev bench hooks — 2026-10-02-1035

- system: windows/amd64, AMD64 Family 26 Model 68 Stepping 0, AuthenticAMD
- go: go1.27.0
- loomux: 6.1.0
```

6 verglichen, 5 schneller, 1 langsamer, 0 unklar, 5 neu, 0 weggefallen

## Verglichen

| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| PreToolUse (Edit on README.md) | 53.5 ms | 16.8 ms | 3.18× | 42.1 ms | 13.3 ms | 3.16× | 41.5 ms | 13.0 ms | [0] | [2] |
| PostToolUse (Edit on README.md) | 80.0 ms | 33.0 ms | 2.42× | 79.6 ms | 29.2 ms | 2.72× | 64.5 ms | 30.0 ms | [0] | [0] |
| Stop | 216.7 ms | 201.2 ms | 1.08× | 273.0 ms | 164.8 ms | 1.66× | 260.8 ms | 156.2 ms | [1] | [0] |
| commit-msg | 569.4 ms | 28.5 ms | 19.95× | 103.0 ms | 24.5 ms | 4.21× | 98.9 ms | 24.1 ms | [0] | [0] |
| status | 90.1 ms | 5293.5 ms | 0.02× | 109.0 ms | 3109.2 ms | 0.04× | 105.1 ms | 3191.8 ms | [0] | [0] |
| search (fast) | 226.2 ms | 5182.6 ms | 0.04× | 236.1 ms | 97.1 ms | 2.43× | 208.0 ms | 97.6 ms | [0] | [0] |

## Neu

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| SessionStart | 147.2 ms | 103.0 ms | 99.7 ms |
| SubagentStart | 1234.0 ms | 1168.1 ms | 1178.1 ms |
| SubagentStop | 1121.7 ms | 9.7 ms | 9.5 ms |
| graph stats | 34.5 ms | 33.5 ms | 34.5 ms |
| graph check | 1620.3 ms | 117.2 ms | 113.5 ms |

## Weggefallen

keine

## Herkunft

Alt-Seite: `before-merged.json`, die erste Baseline (`baseline/bench-2026-09-29-1903`), darin
`Stop` ganz aus der Nachmessung ohne Last (`baseline-2/bench-2026-09-29-1944`) übernommen. Die
Baseline hat keine Sitzungs- und Subagent-Fälle, weil das vendorte Python fehlte (Messung 7);
sie erscheinen als „neu“.
