# Paritätsliste Schreibschranke: verknüpfte Worktrees

**Quelle:** ultra-brain `loomux-1a-source` (`3cc72d2`), `_writable_roots`.
**Spec:** [2026-09-15-loomux-schranke-worktrees-design.md](../specs/2026-09-15-loomux-schranke-worktrees-design.md)
**Regel:** Jede Zeile braucht eine Freigabe des Nutzers, bevor die Stufe als fertig gilt.

| Fall / Bereich | Alt | Neu | Begründung | Freigabe |
|---|---|---|---|---|
| Write in einem verknüpften Worktree eines `workspace`-Bereichs ohne eigenen Registry-Eintrag | brain guard: verweigert (`lies outside every writable tree`) | erlaubt; Erkennung über `.git`-Datei, `commondir` und Rückverweis `gitdir`, ohne `git`-Prozess | Ein Worktree ist dasselbe Repository; `git rev-parse` kostet 42 ms gegen 24,5 ms Hook (Spec) | offen |
