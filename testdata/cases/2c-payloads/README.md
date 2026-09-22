# Nutzlasten der Sitzungshooks, gemessen

Aufgenommen am 2026-09-20 mit Claude Code 2.1.276 unter Windows 11, in einer
laufenden Sitzung dieses Worktrees: ein Rekorder-Hook in
`.claude/settings.local.json` (nicht eingecheckt) schrieb jede Nutzlast nach
`.loomux/probe/`, während ein Explore-Subagent lief. `claude-stop.json` kam am
Ende einer Runde derselben Sitzung dazu, gemessen mit demselben Hook.

Gekürzt sind nur die Werte, die eine Maschine nennen oder eine Sitzung
identifizieren: `session_id` wird `s1`, `agent_id` wird `a1`, `prompt_id` wird
`p1`, die Pfade werden `{{WORLD}}`, und `last_assistant_message` wird `done`.
Jeder andere Schlüssel steht so da, wie der Wirt ihn geschickt hat.

## Was die Messung belegt

Die drei Annahmen, auf denen `subagent-start` und `subagent-stop` bauen, halten:

1. **Beide Ereignisse tragen dieselbe `agent_id`** (`adec5f5875befdfa3` in der
   Aufnahme). Der Snapshot des Starts ist am Ende also wiederfindbar.
2. **Beide tragen die `session_id` des Hauptagenten**, nicht eine eigene des
   Subagenten. Die Agent-Datei liegt damit unter der Sitzung, deren
   Rundenende den Befund zustellt.
3. **Beide tragen `agent_type`** (`Explore`).

Dazu, ungefragt mitgemessen: `SubagentStop` trägt `stop_hook_active`,
`agent_transcript_path` und `last_assistant_message`; `SubagentStart` trägt
keines davon und auch kein `permission_mode`.

## Was gemessen ist und was gebaut

Gemessen sind `claude-subagent-start.json`, `claude-subagent-stop.json` und
`claude-stop.json`. Abgeleitet, damit Stufe 2c die beiden Fehlerfälle
aufzeichnen kann, und als solche zu lesen:

- `subagent-no-agent.json` — die gemessene Startnutzlast ohne `agent_id`, für
  den Fall, den Python mit Exit 1 abweist.
- `not-json.txt` — kein JSON, für den Fall der kaputten Nutzlast.

`claude-stop.json` trägt gegenüber der Subagenten-Nutzlast kein Agentenfeld,
dafür `effort`. Weder `stop.py` noch `hosts.Read` liest eines dieser Felder;
die acht `hook-stop`-Fälle sind gegen die gemessene Nutzlast neu aufgezeichnet
und kamen bis aufs Byte gleich heraus.

## Antigravity-Nutzlasten

Nachgemessen am 2026-09-22 mit Antigravity 1.2.2 (`agy.exe`):
`agy-stop.json`, `agy-inv.json`, `agy-pre.json` und `agy-post.json`.
Antigravity überträgt `conversationId` (in protojson camelCase) für die Sitzung.
Bei `PreToolUse` werden `stepIdx` und `toolCall` übergeben; `PostToolUse` liefert
nur `stepIdx`, `error` und `conversationId`. Eine gemeinsame `agent_id`
existiert bei Antigravity nicht; `AgentID` bleibt leer und `subagent-start`/`-stop`
verweigern bei Antigravity mit Exit 1.

