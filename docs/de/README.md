# Loomux Dokumentation

Willkommen in der Dokumentations-Suite von Loomux. Loomux ist ein vollwertiges Entwickler-Betriebssystem für KI-Coding-Agenten, das deterministisches Code-Graph-Retrieval, persistentes Projektgedächtnis und Zero-Overhead-Schreibschranken in einem einzigen Go-Binary vereint.

---

## Dokumentations-Übersicht

| Handbuch | Beschreibung |
|---|---|
| 🚀 **[Erste Schritte](getting-started.md)** | Installation, 3-Minuten-Schnellstart und Anbindung an Agenten-Harnesses (Claude Code, Antigravity, Cursor). |
| 🏛️ **[Architektur & Konzepte](architecture.md)** | Das theoretische Fundament: Andrej Karpathys LLM OS, Googles Knowledge Items (KI), Grafts AST-GraphRank und der Schreibschranken-Kernel. |
| ⚙️ **[Konfigurations-Referenz](configuration.md)** | Vollständige Referenz für `.loomux/config.toml` (`[modules]`, `[verify]`, `[policy]`, `[commit]`, `[worktree]`, `[area]`, `[index]`, `[privacy]`, `[wiki]`). |
| 📖 **[CLI-Referenzhandbuch](cli-reference.md)** | Detailliertes Handbuch aller Befehle, Flags, stdin-JSON-Nutzlasten und Exit-Codes. |
| 🪝 **[Hook-Lebenszyklus & Integration](hooks.md)** | Technische Spezifikation des 4-Phasen-Hook-Zyklus, der Host-Formate und des entkoppelten SSE-Ereignisstroms. |
| 🗺️ **[Migrationsplan](migration.md)** | Jede Migrationsstufe und jede in der Fusion übernommene oder gebaute Funktion: Herkunft, Stand, Abhängigkeiten und Priorität. Was danach kommt, steht in der [Roadmap](../../README.de.md#roadmap). |
| ⏱️ **[Leistungs-Benchmarks](benchmarks.md)** | Chronologische Messungen gegenüber den Vorläufer-Programmen und verbindliche Latenzbudgets. |
| 🌐 **[Open-Source-Matrix](open-source-matrix.md)** | Ein Katalog quelloffener GitHub-Projekte nach Sprache, Framework und Sterne-Klasse. |

---

## Kern-Philosophie

1. **Deterministisches Retrieval statt stochastischem Raten**: Lexik schlägt Kandidaten vor; der strukturelle AST-Aufrufgraph entscheidet, welche Komponenten wirklich relevant sind.
2. **Kuratierte Ground Truth statt roher Text-Dumps**: Second-Brain-Knowledge-Items liefern verifizierte architektonische Absichten und Entscheidungsregister (ADRs).
3. **Strikter Speicherschutz**: Sub-35ms-Wächter stellen sicher, dass Coding-Agenten niemals unbemerkt sensible Dateien modifizieren oder zerstörerische Befehle ausführen.
