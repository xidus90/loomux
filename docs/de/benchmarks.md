# Benchmarks

Chronologische Leistungsmessungen für loomux, kalt und warm.

## 2026-09-14 16:05 — Basismessung (Vorläufer-Programme)

Gemessen unter Windows x86_64, warm Median über 5 Läufe. Grundlage für die Zielwerte der Loomux-Fusion.

| Fall | Vorläufer-Laufzeit | Zeit | Loomux-Zielwert |
|---|---|---:|---|
| `ulguard --root` (PreToolUse, Edit) | Go (`ulguard.exe` 6,0 MB) | 33 ms | < 35 ms |
| `brain guard` | Go (`brain.exe` 13,7 MB) | 72 ms | vereint in `hook pre-tool-use` |
| Beide Wächter parallel (wie Edit sie sieht) | Go + Go | 73 ms | < 35 ms (vereint im Prozess) |
| Beide Wächter sequenziell (CPU-Zeit) | Go + Go | 114 ms | < 35 ms |
| Go-Startboden (`brain version`, `ulinit --version`) | Go (`ulinit.exe` 8,5 MB) | 32–34 ms | ~32 ms |
| `ulguard hook session-start` | Go | 67 ms | < 100 ms (Stufe 2) |
| `ultraloom hook subagent-start` | Python (mit/ohne `uv run`) | 706–730 ms | < 100 ms (Go-Port) |
| `brain-mcp --help` | Python | 898 ms | < 35 ms (Go-Port) |
| `brain-mcp status` / `brain status` | Python / Go | 3.113 ms / 36 ms | < 35 ms |
| `brain-mcp search` / `brain search` | Python / Go | 7.773 ms / 242 ms | < 50 ms |

### Wichtigste Befunde
1. Die Binärgröße bestimmt die Startzeit nicht: `brain.exe` (13,7 MB) startet in 32 ms, `ulinit.exe` (8,5 MB) in 34 ms, `ulguard.exe` (6,0 MB) in 33 ms.
2. Der Startboden wird ausschließlich durch Paket-`init()`-Funktionen und das Parsen eingebetteter Daten bestimmt, nicht durch die Dateigröße.
3. Python-Hooks und CLI-Brücken verursachten 700–3.000 ms Verzögerung, die durch das reine Go-Binary vollständig entfallen.
