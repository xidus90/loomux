# loomux-Dokumentation

Ein Go-Binary, das KI-Coding-Agenten bewacht: Hooks und eine Schreibschranke, eine Prüfkette, ein Code-Graph und ein Projekt-Wiki.

## Loslegen

1. [loomux installieren](install.md) — Download, Prüfsumme, das maschinenweite Binary, Updates; Linux und macOS.
2. [Ersteinrichtung](getting-started.md) — `loomux init`, die Konfiguration prüfen, den Agenten anbinden.
3. [Konfiguration](configuration.md) — `.loomux/config.toml`, beginnend mit dem ersten Tag.

## Referenz

| Handbuch | Inhalt |
|---|---|
| [CLI-Referenz](cli-reference.md) | Detailliertes Handbuch aller Befehle, Flags, stdin-JSON-Nutzlasten und Exit-Codes. |
| [Konfiguration](configuration.md) | Vollständige Referenz für `.loomux/config.toml` (`[modules]`, `[verify]`, `[policy]`, `[commit]`, `[worktree]`, `[area]`, `[index]`, `[privacy]`, `[wiki]`, `[agent]`, `[flow]`). |
| [Hooks](hooks.md) | Technische Spezifikation des 4-Phasen-Hook-Zyklus, der Host-Formate und des entkoppelten SSE-Ereignisstroms. |

## Konzepte

| Handbuch | Inhalt |
|---|---|
| [Architektur](architecture.md) | Das theoretische Fundament: Andrej Karpathys LLM OS, Googles Knowledge Items (KI), AST-GraphRank und der Schreibschranken-Kernel. |
| [Flows](flows.md) | Flows als Daten: das Ordnerformat, Rollen und Modelle, der Katalog und das Überschreiben, einen Flow beitragen und warum ein Tor einem Menschen gehört. |

## Projekt

| Handbuch | Inhalt |
|---|---|
| [Releases und Pull Requests](releases.md) | Wie ein gemergter Pull Request zum Release wird, Beta-Releases, einen Pull Request öffnen und die Release-Mechanik einrichten. |
| [Benchmarks](benchmarks.md) | Chronologische Leistungsmessungen mit Datum, Basis und Änderung, kalt und warm. |
| [Benchmark-Matrix](benchmarks/matrix.md) | Open-Source-Matrix über Top-Sprachen hinweg mit Detailberichten pro Sprache und Repository. |
| [Open-Source-Matrix](open-source-matrix.md) | Ein Katalog quelloffener GitHub-Projekte nach Sprache, Framework und Sterne-Klasse. |
| [GDScript-Frameworks](stacks/gdscript.md) | Welche Frameworks von Drittanbietern loomux für GDScript fährt, woran es sie erkennt und was jedes trägt. |

## Kern-Philosophie

1. **Deterministisches Retrieval statt stochastischem Raten**: Lexik schlägt Kandidaten vor; der strukturelle AST-Aufrufgraph entscheidet, welche Komponenten wirklich relevant sind.
2. **Kuratierte Ground Truth statt roher Text-Dumps**: Second-Brain-Knowledge-Items liefern verifizierte architektonische Absichten und Entscheidungsregister (ADRs).
3. **Strikter Speicherschutz**: Der Wächter entscheidet innerhalb eines Budgets von 35 ms. Er weist Schreibzugriffe auf Geheimnisse wie `.env` ab, und seine Befehlsregeln in `[policy.commands]` verweigern eine Befehlszeile, auf die ein regulärer Ausdruck so passt, wie sie geschrieben steht. Beides gilt innerhalb der Grenzen, die unter [Hooks](hooks.md#7-der-entscheidungsweg-von-pre-tool-use) aufgelistet sind.
4. **Startzeit-Disziplin**: Keine Paketvariable und kein `init()` darf eingebettete Daten parsen oder Netzwerk-I/O durchführen.
5. **Strikte Isolierung**: Hook-Pfade laufen im Prozess und hängen niemals von einem laufenden `serve`-Daemon ab.
