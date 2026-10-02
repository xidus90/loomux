# iam_wiki

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
# loomux dev bench hooks — 2026-10-02-1039

- system: windows/amd64, AMD64 Family 26 Model 68 Stepping 0, AuthenticAMD
- go: go1.27.0
- loomux: 6.1.0
```

3 verglichen, 1 schneller, 2 langsamer, 0 unklar, 0 neu, 0 weggefallen

## Verglichen

| Fall | kalt alt | kalt neu | × | warm Ø alt | warm Ø neu | × | Median alt | Median neu | Exit alt | Exit neu |
|---|---:|---:|---:|---:|---:|---:|---:|---:|---|---|
| status | 51.4 ms | 3481.7 ms | 0.01× | 68.3 ms | 3206.6 ms | 0.02× | 42.5 ms | 3119.7 ms | [0] | [0] |
| search (fast) | 233.7 ms | 70.8 ms | 3.30× | 198.2 ms | 70.0 ms | 2.83× | 196.7 ms | 68.1 ms | [0] | [0] |
| wiki lint | 43.1 ms | 675.3 ms | 0.06× | 59.8 ms | 81.5 ms | 0.73× | 47.5 ms | 81.1 ms | [0] | [1] |

## Neu

keine

## Weggefallen

keine

## Herkunft

Alt-Seite: `baseline/bench-2026-09-29-1904`. iam_wiki hat weder vorher noch nachher Hooks
(die Umstellung legte nur den Wiki-Bereich an, kein `init`); verglichen sind nur die
Wissensbefehle. Die Fälle sind von Hand aus `extras-new.json` gebaut, weil `dev bench cases`
eine `settings.json` ohne Hook ablehnt („no hook of any event applies to an edit“).
