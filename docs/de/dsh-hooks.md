# Native DSH-Hooks

loomux bewacht eine DeepSeek-Harness-Sitzung (DSH) nativ: das Plugin
`dsh-loomux-hooks` registriert die eigenen Nähte des Harness und führt die
loomux-Hooks des Projekts in ihnen als echte Hook-Prozesse — dieselben
Befehle, dasselbe Protokoll im Claude-Dialekt, und Ablehnung, Blockade,
Lenkung und Kontext als harness-eigene Aktionen statt einer Brücke.

---

## 1. Einrichten

Zwei Stücke gehören zusammen: die Hook-Datei, die loomux schreibt, und das
Plugin, das der Harness lädt.

**Die Hook-Datei.** Im Projektgrund:

```sh
go run ./cmd/loomux init --yes --hosts=dsh
```

`init` schreibt `.loomux/dsh/hooks.json`: einen Eintrag pro Ereignis mit dem
Matcher `.*` und dem absoluten Pfad zum loomux-Binary, im
`settings.json`-Dialekt von Claude. Ist die Datei einmal da, pflegt ein
nacktes `loomux init` sie wie jeden anderen Host-Eintrag.

**Das Plugin.** Installiere das Paket `dsh-loomux-hooks` in ein Profil,
entweder über den `plugin_manager` des Harness (Bündel `dsh-loomux-hooks`
installieren) oder durch den Eintrag in der Bündelliste des Profils samt
seinem Patch. Seine Konfiguration ist durchweg optional:

```yaml
- id: loomux-hooks
  name: dsh-loomux-hooks
  config: {}
  # config:
  #   configPath: C:/arbeit/proj/.loomux/dsh/hooks.json  # überspringt die Kette
  #   defaultTimeoutMs: 60000                            # für Einträge ohne Timeout
```

Nach der Installation ist ein Neustart nötig, und damit eine geänderte
Hook-Datei neu gelesen wird: was ein Verzeichnis auflöst, merkt sich das
Plugin bis zum Neustart des Harness-Prozesses. Eine Sitzung ohne
Hook-Einträge für ihr Verzeichnis läuft ohne Hooks.

## 2. Wie die Hook-Datei gefunden wird

Das Plugin löst pro Sitzungsverzeichnis genau eine Datei auf — das eigene
Verzeichnis des Harness-Prozesses sagt nichts über das laufende Projekt —,
und merkt sich, was es fand, in dieser Reihenfolge:

1. `configPath` aus der Plugin-Konfiguration, wenn gesetzt;
2. `.loomux/dsh/hooks.json` im Arbeitsverzeichnis des Agenten;
3. `.claude/settings.json`;
4. `hooks/hooks.json`, die für den Codex-Host dokumentierte Position.

Nichts gefunden: die Nähte bleiben für dieses Verzeichnis stumm. Eine Datei,
die zwar da, aber nicht lesbar oder nicht parsebar ist: das Plugin warnt
über seinen Logger, und dieses Verzeichnis läuft ohne Hooks — es fällt nicht
auf die nächste Position zurück, denn eine kaputte Datei soll nie
stillschweigend umgangen werden. Eine Datei mit leerem `hooks`-Objekt
unterdrückt Hooks bewusst. Ist ein Matcher keine gültige reguläre
Ausdrucksweise, verwirft das Plugin die ganze Datei.

## 3. Nähte und Entscheidungen

Das Plugin registriert sechs Nähte. Jeder Hook-Befehl ist `loomux hook
<event> --host dsh`; die Nutzlast kommt als JSON auf stdin wie in jedem
anderen Host, und `transcript_path` darf leer sein.

| Harness-Naht | Hook-Ereignis | Was die Entscheidung tut |
|---|---|---|
| `agent/created` | SessionStart | gewartet, bevor die erste Eingabe frei wird; `additionalContext` → als Benutzernachricht injiziert |
| `tools/pre-execute` | PreToolUse | Exit 2 oder `permissionDecision: "deny"` → der Aufruf wird mit dem Grund abgelehnt; `"ask"` → der Harness fragt |
| `tools/post-execute` | PostToolUse | Exit 2 oder deny → das Ergebnis wird blockiert und der Grund zurückgespielt; `additionalContext` → nach dem Aufruf injiziert |
| `agent/turn-stopping` | Stop | Exit 2 oder `decision: "block"` → der Agent wird mit dem Grund weitergesteuert |
| `subagent/start` | SubagentStart | `additionalContext` → in das Kind injiziert |
| `subagent/end` | SubagentStop | die Hooks laufen und werden protokolliert; die Entscheidung wird ignoriert |

Einträge laufen in Dateireihenfolge; die erste entscheidende Antwort einer
Naht gewinnt, und mehrere `additionalContext`-Antworten sammeln sich an.
Nur ein `hookSpecificOutput` mit passendem `hookEventName` zählt — ein
falscher Name macht das JSON zu einem Nichtstun, genau wie in Claude Code.

## 4. Wenn etwas schiefgeht

- **Interne Fehler sperren nie.** Exit 1, ein Timeout, ein fehlendes Binary
  und nicht parsebare Ausgabe lassen den Aufruf durch, an jeder Naht; nur
  Exit 2 und ein korrektes JSON-Votum entscheiden.
- **Ein Timeout tötet den ganzen Prozessbaum**, unter Windows mit
  `taskkill /T /F`, damit ein Hook, der Enkel gestartet hat, keine zurücklässt.
  Das `timeoutMs` des Eintrags gewinnt gegen das `defaultTimeoutMs` des
  Plugins, dessen Vorgabe 60000 ist.
- **Schleifenschutz.** Die erste Stop-Auswertung pro Agentenlauf meldet
  `stop_hook_active: false`, jede spätere `true` — Hook-Autoren können dort
  aussteigen. Davon unabhängig ignoriert das Plugin die Lenkung eines
  Stop-Hooks nach spätestens acht Steuerungen pro Agentenlauf.
- **Hooks stehen außerhalb des Harness-Sandkastens.** Ein Hook-Prozess läuft
  mit den Rechten deines Benutzers und sieht den Arbeitsbereich über
  `CLAUDE_PROJECT_DIR`; die Datei-Richtlinie des Harness begrenzt ihn nicht.
  Schreibschranken-Ablehnungen der Agentenschreibvorgänge bleiben davon
  unberührt.
- **Eine Warnung bricht nie eine Sitzung.** Eine kaputte Hook-Datei, ein
  toter Audit-Kanal und ein bockiger Eintrag werden geloggt; die Sitzung läuft weiter.

## 5. Audit

Jeder Hook-Lauf hängt zwei Datensätze des bekannten Sitzungsprotokoll-Formats
an: `hook/invoked` (Runde, Naht, Dialekt `claude`, Handler-Kennung
`loomux:<Ereignis>:<Index>`, Matcher) und `hook/result` (Votum, Dauer,
Exitcode, stderr-Zusammenfassung von höchstens 500 Zeichen). Ist das
Sitzungsprotokoll nicht erreichbar, schreibt sie der Plugin-Logger; das
Audit ändert nie eine Entscheidung.

## 6. Am echten Harness geprüft

Das Verhalten oben wurde gegen eine installierte DSH bewiesen, bevor es
codiert wurde: Exit 2 lehnt bei `PreToolUse` ab und blockiert bei
`PostToolUse` mit dem stderr-Text als sichtbarem Grund; die Steuerung eines
`Stop`-Hooks kommt genau einmal pro Exit-2 zurück; der
`SessionStart`-Kontext steht als erste Benutzernachricht des neuen Agenten;
die Block-Rückmeldung und die injizierte brauchen keinen Fabrik-Import.
JSON-Vota zählen nur mit passendem `hookEventName`, und ein Hook-Aufruf
kostet etwa eine halbe Sekunde — Matcher und Geltungsbereich sind die
Stellschrauben. Einzelheiten pro Hook: das Paket-README in
`dsh-loomux-hooks/`.
