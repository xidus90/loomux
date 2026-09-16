# Was in den Hook-Payloads wirklich steht

Gemessen am 2026-08-25 mit Claude Code **2.1.241** unter Windows 11, im
Hauptcheckout `C:/Users/micro/Documents/#GIT/ultraloom`. Aufgezeichnet hat ein
Skript außerhalb des Repos, das nichts tut außer stdin nach JSON zu schreiben
und mit 0 zu enden. Die vier Rekorder-Einträge in `.claude/settings.json` waren
vorübergehend und sind wieder entfernt.

Provoziert wurde über headless Sitzungen (`claude -p …`) im Hauptcheckout —
eine mit Schreibvorgang und Subagent, eine zweite mit einem Stop-Hook, der beim
ersten Aufruf Exit 2 gibt, damit ein **zweiter** `Stop` ankommt, und eine
dritte, die zusätzlich `SubagentStart` registriert hatte.

Sitzungs-IDs und Pfade sind unten gekürzt; die Feldnamen sind wörtlich.

## Die drei Fragen der Spec

1. **`stop_hook_active` gibt es.** Es steht in jeder `Stop`- **und** jeder
   `SubagentStop`-Payload. Beim ersten Aufruf `false`, beim zweiten — dem, den
   ein vorangegangenes Exit 2 erzwungen hat — `true`.
2. **`agent_id` und `agent_type` gibt es**, beide bei `SubagentStop`. Dazu
   `agent_transcript_path`. `agent_id` ist eine 17-stellige Hex-Zeichenkette,
   `agent_type` der Name des Subagenten (`general-purpose`).
3. **Die Sitzungskennung heißt `session_id`** und ist über alle Aufrufe einer
   Sitzung hinweg identisch — `SessionStart`, `PostToolUse`, `SubagentStart`,
   `SubagentStop` und beide `Stop`-Aufrufe trugen dieselbe UUID. Eine neue
   headless Sitzung bekam eine neue.

## Zwei Befunde, die nicht gefragt waren

- **`SubagentStart` existiert** und trägt dieselbe `agent_id` wie das
  zugehörige `SubagentStop`. Der Schnappschuss für `subagent-stop` kann also je
  Subagent genommen werden, nicht nur je Sitzung.
- **Mehrere Einträge desselben Ereignisses laufen gleichzeitig, nicht
  nacheinander.** Zwei `Stop`-Einträge mit je zwei Sekunden Verweildauer
  starteten 2 ms auseinander und liefen vollständig überlappend. Ein Zähler,
  den zwei Hooks desselben Ereignisses fortschreiben, ist damit nicht
  verlässlich — er gehört in **einen** Eintrag.
- **`prompt_id`** ist über einen Zug stabil und bleibt auch beim zweiten `Stop`
  derselbe; er wechselt mit dem nächsten Nutzerbeitrag.

## `SessionStart`

Fünf Felder, mehr nicht:

| Feld | Beispielwert |
| --- | --- |
| `cwd` | `C:\Users\micro\Documents\#GIT\ultraloom` |
| `hook_event_name` | `SessionStart` |
| `session_id` | `df304b39-…-fd1a11d72986` |
| `source` | `startup` |
| `transcript_path` | `C:\Users\micro\.claude\projects\C--Users-…\df304b39-….jsonl` |

Kein `permission_mode`, kein `effort`, kein `prompt_id` — die Sitzung hat noch
keinen Zug.

## `PostToolUse`

| Feld | Beispielwert |
| --- | --- |
| `cwd` | `C:\Users\micro\Documents\#GIT\ultraloom` |
| `duration_ms` | `8` |
| `effort` | `{"level": "low"}` |
| `hook_event_name` | `PostToolUse` |
| `permission_mode` | `auto` |
| `prompt_id` | `07bca88d-…-da332d067a1c` |
| `session_id` | `df304b39-…-fd1a11d72986` |
| `tool_input` | `{"content": "hallo\n", "file_path": "C:\…\probe.txt"}` |
| `tool_name` | `Write` |
| `tool_response` | `{"content": …, "filePath": …, "originalFile": null, "structuredPatch": [], "type": "create", "userModified": false}` |
| `tool_use_id` | `toolu_015GZ3BSdJx8hKn6Z52LQAGT` |
| `transcript_path` | `C:\Users\micro\.claude\projects\C--Users-…\df304b39-….jsonl` |

Bemerkenswert für `post-edit`: `tool_input.file_path` kommt mit
**Backslashes**, `tool_response.filePath` mit Schrägstrichen. Beide meinen
denselben Pfad. Der Entwurf liest `tool_input`, und `pathlib.Path` verträgt
unter Windows beides.

## `Stop`

| Feld | Beispielwert (erster Aufruf) |
| --- | --- |
| `background_tasks` | `[]` |
| `cwd` | `C:\Users\micro\Documents\#GIT\ultraloom` |
| `effort` | `{"level": "low"}` |
| `hook_event_name` | `Stop` |
| `last_assistant_message` | `EINS` |
| `permission_mode` | `auto` |
| `prompt_id` | `8b027eb4-…-46c4ed446230` |
| `session_crons` | `[]` |
| `session_id` | `607ea60f-…-19f08429d5ba` |
| `stop_hook_active` | `false` |
| `transcript_path` | `C:\Users\micro\.claude\projects\C--Users-…\607ea60f-….jsonl` |

Beim **zweiten** Aufruf derselben Sitzung — erzwungen durch ein Exit 2 des
ersten — ist die Feldliste dieselbe, und es ändern sich zwei Werte:

- `stop_hook_active`: `true`
- `last_assistant_message`: `ZWEITE` (die Antwort auf das, was der blockende
  Hook auf stderr geschrieben hatte)

`session_id` und `prompt_id` bleiben gleich.

## `SubagentStop`

Wie `Stop`, plus drei Felder:

| Feld | Beispielwert |
| --- | --- |
| `agent_id` | `a53a0c59d7bb63527` |
| `agent_transcript_path` | `C:\Users\…\<session-id>\subagents\agent-a53a0c59d7bb63527.jsonl` |
| `agent_type` | `general-purpose` |
| `background_tasks` | `[]` |
| `cwd` | `C:\Users\micro\Documents\#GIT\ultraloom` |
| `effort` | `{"level": "low"}` |
| `hook_event_name` | `SubagentStop` |
| `last_assistant_message` | `7` |
| `permission_mode` | `auto` |
| `prompt_id` | `9be3725e-…-7563cc82d9bd` |
| `session_crons` | `[]` |
| `session_id` | `4931a4bc-…-f5caf7495ae7` |
| `stop_hook_active` | `false` |
| `transcript_path` | `C:\Users\micro\.claude\projects\C--Users-…\4931a4bc-….jsonl` |

Die `session_id` ist die der **Muttersitzung**, nicht eine eigene des
Subagenten. Wer je Subagent unterscheiden will, braucht `agent_id`.

## `SubagentStart`

Nicht in der Spec verlangt, aber gemessen, weil Task 5 daran hängt:

| Feld | Beispielwert |
| --- | --- |
| `agent_id` | `a53a0c59d7bb63527` |
| `agent_type` | `general-purpose` |
| `cwd` | `C:\Users\micro\Documents\#GIT\ultraloom` |
| `hook_event_name` | `SubagentStart` |
| `prompt_id` | `9be3725e-…-7563cc82d9bd` |
| `session_id` | `4931a4bc-…-f5caf7495ae7` |
| `transcript_path` | `C:\Users\micro\.claude\projects\C--Users-…\4931a4bc-….jsonl` |

Kein `agent_transcript_path` — den gibt es erst am Ende. Die `agent_id` ist
dieselbe wie beim `SubagentStop` desselben Subagenten; das ist der Schlüssel,
den der Schnappschuss braucht.
