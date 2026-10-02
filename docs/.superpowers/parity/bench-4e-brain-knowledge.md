# brain-knowledge

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

3 verglichen, 2 schneller, 1 langsamer, 0 unklar, 7 neu, 0 weggefallen

## Verglichen

| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| status | 71.4 ms | 3514.4 ms | 0.02× | 43.8 ms | 3133.6 ms | 0.01× | 43.0 ms | 3207.6 ms | [0] | [0] |
| search (fast) | 213.3 ms | 74.5 ms | 2.86× | 204.8 ms | 81.6 ms | 2.51× | 202.6 ms | 82.4 ms | [0] | [0] |
| wiki lint | 137.1 ms | 41.4 ms | 3.31× | 46.2 ms | 40.9 ms | 1.13× | 47.0 ms | 41.4 ms | [0] | [1] |

## Neu

| Fall | kalt | warm Ø | Median |
|---|---:|---:|---:|
| SessionStart | 43.0 ms | 9.0 ms | 9.0 ms |
| PreToolUse (Edit on AGENTS.md) | 13.5 ms | 12.7 ms | 12.4 ms |
| PostToolUse (Edit on AGENTS.md) | 22.7 ms | 20.5 ms | 20.4 ms |
| SubagentStart | 1219.5 ms | 1170.2 ms | 1163.6 ms |
| SubagentStop | 1131.2 ms | 10.8 ms | 11.0 ms |
| Stop | 222.9 ms | 188.4 ms | 189.0 ms |
| commit-msg | 30.0 ms | 26.2 ms | 25.9 ms |

## Weggefallen

keine

## Herkunft

Alt-Seite: `baseline/bench-2026-09-29-1904`. Der Vault hatte vorher keine Hooks; alle Hooks
und `commit-msg` erscheinen als „neu“.
