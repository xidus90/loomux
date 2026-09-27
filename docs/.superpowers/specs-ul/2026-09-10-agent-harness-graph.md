# Agent-Harness: der lokale Entwicklungszyklus als Graph

**Stand:** 2026-09-11 · **Zweig:** `feature/agent-harness` · **Kein Spec.**
Begleitpapier: [Knotenkatalog](2026-09-10-agent-harness-knotenkatalog.md)

Zwei Eingänge, siebzehn Phasen, sechs Schleifen und acht Stellen, an denen ein
Mensch antwortet: vier davon immer und vier nur dann, wenn das Modell nicht
weiterkommt. Auslieferung und Betrieb gehören nicht
in dieses Bild.

## Legende

| Form | Knotenart | Bedeutung |
|---|---|---|
| `[ Rechteck ]` blau | Codeknoten | deterministisch, kostet keine Tokens |
| `( abgerundet )` ocker | Agentenknoten | ein Modellaufruf |
| `[[ doppelt ]]` violett | Prüferfächer | N Linsen parallel, danach zusammengeführt |
| `{{ Sechseck }}` rot | Mensch, unabdinglich | Tor: eine Frage, der Lauf endet und lässt sich wiederaufnehmen |
| `{{ Sechseck }}` grün gestrichelt | Mensch, optional | Notausgang, wenn es nicht weitergeht |
| `([ Stadion ])` grau gestrichelt | Eingang oder Verweis | Start, oder Sprung in einen anderen Abschnitt |

## Überblick

```mermaid
flowchart LR
    P["PLANEN<br/>einmal je Absicht"] --> B["BAUEN und PRÜFEN<br/>je Karte"]
    B --> A["ABSCHLIESSEN<br/>einmal je Absicht"]
    B -->|nächste Karte| B
    B -.->|"Architekturfrage, Umplanung"| P
    A -.->|"Befunde im Abschlussreview"| B
```

## 1. Planen

```mermaid
flowchart TD
    auftrag(["Auftrag von extern<br/>Ticket, Call, Stakeholder"]):::start
    chat(["Nutzer direkt im Chat"]):::start
    intake["0 · Eingang<br/>legt das gelieferte intent.md ins Projekt"]:::code
    clarify("1 · Klärung · Brainstorming<br/>über intent.md und Gespräch"):::agent
    gClarify{{"MENSCH<br/>Abnahme der Klärung"}}:::must
    spec("2 · Spec<br/>schreibt spec.md"):::agent
    vSpec[["LLM-PRÜFUNG<br/>Kern: grounding · references · consistency · open<br/>+ testability · design<br/>+ coverage, nur wenn intent.md vorliegt"]]:::fan
    gSpec{{"MENSCH<br/>Finale Abnahme Spec"}}:::must
    tagger("3 · Einordnung<br/>Marken des Vorhabens"):::agent
    plan("4 · Plan<br/>plan.md mit markierten Aufgaben"):::agent
    vPlan[["LLM-PRÜFUNG — derselbe Block<br/>Kern: grounding · references · consistency · open<br/>+ traceability · ordering · cards · proof"]]:::fan
    gPlan{{"MENSCH<br/>Freigabe Plan"}}:::must
    clarifyPlan("Klärung zum Plan<br/>optional"):::agent
    board["Plan → Board<br/>schneidet Karten, deutet nichts"]:::code
    toPick(["weiter: Bauen und Prüfen"]):::start

    auftrag --> intake --> clarify
    chat --> clarify
    clarify --> gClarify
    gClarify -->|"noch nicht — weiter klären"| clarify
    gClarify -->|abgenommen| spec
    spec --> vSpec
    vSpec -->|"offene Fragen"| clarify
    vSpec -->|"Unstimmigkeit, Verweis, Halluzination → Reviser"| spec
    vSpec -->|sauber| gSpec
    gSpec -->|"etwas offen — zurück ins Brainstorming"| clarify
    gSpec -->|abgenommen| tagger --> plan
    plan --> vPlan
    vPlan -->|"offene Fragen"| clarifyPlan
    vPlan -->|"Befunde → Reviser"| plan
    vPlan -->|sauber| gPlan
    gPlan -->|abgelehnt| clarifyPlan --> plan
    gPlan -->|freigegeben| board --> toPick

    classDef code  fill:#DDE8F0,stroke:#2F6389,stroke-width:1.4px,color:#14181C;
    classDef agent fill:#F3E7D2,stroke:#99621C,stroke-width:1.4px,color:#14181C;
    classDef fan   fill:#E9E1F1,stroke:#6A4B86,stroke-width:1.4px,color:#14181C;
    classDef must  fill:#F7DFDC,stroke:#A8342A,stroke-width:2px,color:#14181C;
    classDef start fill:#FBFAF7,stroke:#4C5765,stroke-width:1.4px,stroke-dasharray:2 3,color:#14181C;
    classDef done  fill:#FBFAF7,stroke:#6B7681,stroke-width:1.2px,color:#4C5765;
```

**Zwei Eingänge, ein Treffpunkt.** Ein Auftrag von extern (Ticket, Call,
Stakeholder) liefert ein `intent.md`, das ins Projekt gelegt wird. Oder du
beginnst direkt im Chat. Beide Wege münden ins Brainstorming, und es läuft so
lange, bis du die Klärung **abnimmst**. Erst dann wird die Spec geschrieben.

Ein eigenes Annahmetor für externe Aufträge gibt es nicht mehr. Die Abnahme der
Klärung übernimmt diese Rolle auf beiden Wegen. Soll ein Auftrag nicht
weiterverfolgt werden, nimmst du die Klärung nicht ab, und der Lauf endet.

**`intent.md` gibt es nur auf dem Auftragsweg.** Beginnst du im Chat, ist die
Spec das erste Artefakt. Ein nachgeschriebenes `intent.md` wäre dort nur eine Kopie
der Spec: Problem, Ziel, Randbedingungen und verworfene Alternativen stehen
ohnehin in ihr. Deshalb läuft auch die Linse `coverage` nur, wenn ein
`intent.md` vorliegt. Sie braucht einen Maßstab, den niemand im Harness
geschrieben hat, und den gibt es nur, wenn die Anforderung von außen kommt.
Beginnst du im Chat, übernimmst du den Abgleich an der finalen Abnahme der Spec,
weil du im Gespräch dabei warst.

**Ein Prüfblock, zweimal eingesetzt.** Spec und Plan durchlaufen
dieselbe Prüfung mit derselben Rolle und derselben Anweisung, nur das Artefakt
davor wechselt. Vier Linsen bilden den gemeinsamen Kern:

| Linse | Fehlerbild |
|---|---|
| `grounding` | Halluzination: eine genannte Datei, Funktion, ein Schalter oder eine Messung, die es nicht gibt. Gegen das Repo nachgerechnet, nicht gelesen |
| `references` | ein Verweis oder Link zeigt ins Leere |
| `consistency` | Widerspruch im Dokument |
| `open` | offene Frage, stehengebliebener Platzhalter, aufgeschobene Entscheidung |

Dazu kommen die Linsen je Artefakt:

| Artefakt | Linse | Fehlerbild |
|---|---|---|
| `spec.md` | `coverage` | Anforderung erfunden, oder das `intent.md` ist nicht gedeckt. Nur auf dem Auftragsweg |
| | `testability` | ein Kriterium, das niemand prüfen kann |
| | `design` | Schnittstellen, Fehlerfälle, Datenmodell, Bedrohung oder Leistung tragen nicht |
| `plan.md` | `traceability` | Aufgabe ohne Bezug zur Spec, oder Spec-Punkt ohne Aufgabe |
| | `ordering` | Abhängigkeit verletzt; zwei Aufgaben fassen dieselbe Datei an |
| | `cards` | das Board kann keine Karten schneiden: Marke fehlt, oder die Aufgabe ist zu groß für einen Zug |
| | `proof` | Aufgabe ohne benannten Beweis, dass sie fertig ist |

Der Kern hat beim Zusammenführen **Vorrang**: meldet er etwas, geht das Artefakt
zurück, bevor ein anderer Befund gewichtet wird. Alle Linsen laufen trotzdem
parallel, denn der Kern besteht meistens.

**Wohin ein Befund geht, hängt davon ab, wer ihn beheben kann.** Eine
Unstimmigkeit, ein toter Verweis oder eine Halluzination lässt sich am Dokument
reparieren; das macht der Reviser. Eine **offene Frage** kann nur der Nutzer
beantworten, deshalb geht sie zurück ins Brainstorming und nicht an den
Reviser, der sie sonst mit einer erfundenen Antwort schließen würde. Nach
derselben Regel geht eine offene Frage im Plan in die Klärung zum Plan.

**Der Mensch sieht nur ein sauberes Artefakt**, weil die Schleife zwischen
Erzeuger und Prüfung vorher ausläuft. Bleibt bei der finalen Abnahme trotzdem
etwas offen, geht es zurück ins Brainstorming.

**Der Plan schneidet die Karten, nicht das Board.** Jede Aufgabe im `plan.md`
trägt schon `type`, `area`, `size`, ihre Abhängigkeiten und ihren Beweis.
`Plan → Board` übernimmt das wörtlich und deutet nichts.

**Kein gezeichneter Verwurf vom Plantor zurück zur Spec.** Ein Tor beendet den
Lauf. Wer den Plan ganz verwirft, antwortet nicht und startet einen neuen Lauf
ab der Spec. Eine gezeichnete Kante würde die Spec auf den Zyklus des Plans
legen, und ihr Besuchsdeckel müsste Spec-Runden mal Plan-Runden abdecken.

## 2. Bauen und Prüfen — je Karte

```mermaid
flowchart TD
    fromBoard(["vom Board"]):::start
    pick["Picker<br/>nächste Karte"]:::code
    toDocs(["weiter: Abschließen"]):::start
    research("5 · Recherche<br/>Digest mit Fundstellen"):::agent
    debug("Debugging-Unterzyklus<br/>Hypothese statt Fix"):::agent
    testw("6 · Test schreiben"):::agent
    red["Rot-Prüfer<br/>Test MUSS fehlschlagen"]:::code
    build("7 · Bau<br/>Implementer"):::agent
    verify["8 · Prüfkette<br/>Satz nach Marke"]:::code
    fCode[["9 · Codereview<br/>bugs · compliance<br/>security · simplicity"]]:::fan
    rework("10 · Nacharbeit<br/>Reviser"):::agent
    adj("11 · Schlichtung<br/>Adjudicator"):::agent
    policy("12 · Regelprüfung<br/>Policy-Guard"):::agent
    book["Booker<br/>Karte verbuchen"]:::code
    gArch{{"MENSCH<br/>Architekturfrage"}}:::opt
    gReplan{{"MENSCH<br/>Umplanung"}}:::opt
    gBlock{{"MENSCH<br/>Blockade"}}:::opt
    toPlan(["zurück: 4 · Plan"]):::start

    fromBoard --> pick
    pick -->|keine Karte offen| toDocs
    pick -->|"kind: research"| research
    pick -->|"type: bug"| debug
    pick -->|sonst| testw

    research --> book
    testw --> red
    debug --> red
    red -->|"grün — der Test taugt nicht"| testw
    red -->|rot| build
    build --> verify
    verify -->|"rot, Karte type: bug"| debug
    verify -->|rot| build
    verify -->|grün| fCode
    debug -->|"3 Fixes gescheitert"| gArch
    gArch --> toPlan

    fCode -->|"Konflikt mit dem Plan"| adj
    fCode -->|wichtig| rework
    fCode -->|sauber| policy
    adj -->|"Plan trägt nicht mehr"| gReplan
    adj --> rework
    gReplan --> toPlan
    rework -->|"Runde 5"| gBlock
    rework --> verify
    gBlock --> pick
    policy -->|Verstoß| rework
    policy -->|ok| book
    book --> pick

    classDef code  fill:#DDE8F0,stroke:#2F6389,stroke-width:1.4px,color:#14181C;
    classDef agent fill:#F3E7D2,stroke:#99621C,stroke-width:1.4px,color:#14181C;
    classDef fan   fill:#E9E1F1,stroke:#6A4B86,stroke-width:1.4px,color:#14181C;
    classDef opt   fill:#D9EDE9,stroke:#1F6F66,stroke-width:1.6px,stroke-dasharray:5 3,color:#14181C;
    classDef start fill:#FBFAF7,stroke:#4C5765,stroke-width:1.4px,stroke-dasharray:2 3,color:#14181C;
```

**Die Reihenfolge der Kanten trägt Bedeutung.** `next_name` nimmt die erste
Kante, deren Bedingung hält. Eine Kante ohne Bedingung hält immer und steht
deshalb zuletzt. Aus `pick` heißt das: zuerst „keine Karte offen", dann
Recherche, dann Fehler, und die unbedingte Kante in den Bau ganz am Ende.
Dasselbe gilt für `verify`: eine rote Fehlerkarte geht zurück ins Debugging,
bevor die allgemeine Kante „rot → Bau" greift.

**Nacharbeit läuft durch die Prüfkette.** Der Reviser ändert Code, also muss die
Änderung erst wieder grün sein, bevor der Codefächer sie erneut liest.

## 3. Abschließen — einmal je Absicht

```mermaid
flowchart TD
    fromPick(["vom Picker: keine Karte offen"]):::start
    docs("13 · Doku<br/>EN und .de.md"):::agent
    fDocs[["Fächer Doku<br/>graph · parity"]]:::fan
    final[["14 · Abschlussreview<br/>Codelinsen, scope: branch"]]:::fan
    toRework(["zurück: 10 · Nacharbeit"]):::start
    commit["15 · Commit<br/>Sprachtor, Nachweis-Prüfer"]:::code
    evals["16 · Übergabe<br/>Eval-Suite"]:::code
    gPush{{"MENSCH<br/>Push"}}:::must
    ende(["Lauf beendet"]):::done

    fromPick --> docs --> fDocs
    fDocs -->|Befunde| docs
    fDocs -->|sauber| final
    final -->|Befunde| toRework
    final -->|sauber| commit --> evals --> gPush
    gPush -->|"ja — Commits gehen aufs Remote"| ende
    gPush -->|"nein — Commits bleiben lokal"| ende

    classDef code  fill:#DDE8F0,stroke:#2F6389,stroke-width:1.4px,color:#14181C;
    classDef agent fill:#F3E7D2,stroke:#99621C,stroke-width:1.4px,color:#14181C;
    classDef fan   fill:#E9E1F1,stroke:#6A4B86,stroke-width:1.4px,color:#14181C;
    classDef must  fill:#F7DFDC,stroke:#A8342A,stroke-width:2px,color:#14181C;
    classDef start fill:#FBFAF7,stroke:#4C5765,stroke-width:1.4px,stroke-dasharray:2 3,color:#14181C;
    classDef done  fill:#FBFAF7,stroke:#6B7681,stroke-width:1.2px,color:#4C5765;
```

## Mensch in der Schleife

Ein Tor stellt **eine** Frage, und der Lauf endet so, dass er sich
wiederaufnehmen lässt. Weitergefahren wird mit `ultraloom resume --answer`, und
die Antwort wählt die Kante. Jedes Tor hat deshalb zwei Ausgänge.

### Unabdinglich

| Tor | wann | warum | bei „nein" |
|---|---|---|---|
| Abnahme der Klärung | Ende des Brainstormings, auf beiden Eingängen | du erklärst die Klärung für fertig; erst dann wird eine Spec geschrieben. Ersetzt das frühere Annahmetor für externe Aufträge | weiter klären; wird nie abgenommen, endet der Lauf |
| Finale Abnahme Spec | die LLM-Prüfung meldet sauber | ob es die *richtige* Absicht war, prüft kein Modell | etwas offen: zurück ins Brainstorming |
| Freigabe Plan | LLM-Prüfung meldet sauber | der letzte Punkt, an dem eine Kurskorrektur billig ist | Klärung zum Plan, dann neuer Plan |
| Push | nach Commit und Eval-Suite | Projektregel: ein Lauf darf committen, über das Remote entscheidet ein Mensch | Commits bleiben lokal |

### Optional — wenn das Modell nicht weiterkommt

| Tor | Auslöser | warum |
|---|---|---|
| Blockade | Runde 5 der Nacharbeit an derselben Karte; ab Runde 4 lief schon ein stärkeres Modell | die Sicherung fällt, statt eine sechste Runde zu versuchen |
| Architekturfrage | 3 gescheiterte Fixes im Debugging | kein vierter Versuch, sondern die Bauart infrage stellen |
| Umplanung | der Schlichter findet, dass der Plan nicht mehr trägt | der Schlichter darf entscheiden, aber nicht den Plan wegwerfen |
| Triage *(nur Anhang)* | ein Detektor schreibt ein `intent.md` | das einzige Artefakt ohne menschlichen Autor; ohne Triage wird aus der Schleife ein Generator |

## Schleifen

`validate()` weist einen Knoten ab, der auf einem Zyklus sitzt und nur einen
Besuch erlaubt. Jede Schleife hier braucht deshalb einen Deckel.

| # | Schleife | Weg | Deckel |
|---|---|---|---|
| L1 | Artefaktschleife | Spec ⇄ LLM-Prüfung · Plan ⇄ LLM-Prüfung · Doku ⇄ Fächer | 5 Runden. Ab Runde 4 ein frischer Arbeiter eine Modellstufe höher; in Runde 5 entscheidet der Schlichter jeden offenen Befund einzeln |
| L2 | Klärungsschleife | Abnahme der Klärung → Klärung · Prüfung (offene Fragen) → Klärung · Finale Abnahme Spec → Klärung · Freigabe Plan → Klärung zum Plan → Plan | `max_visits` über eins an allen drei Toren und an der Klärung, sonst lehnt `validate()` den Graphen ab. Ein Rundendeckel ergibt hier wenig Sinn: die Schleife treibt ein Mensch, nicht das Modell |
| L3 | Rot-Grün | Test → Rot-Prüfer → Bau → Prüfkette → Bau | die Rundenzahl der Karte. Der Rot-Prüfer schickt zurück, wenn der neue Test grün ist, denn dann prüft er nichts |
| L4 | Kartenschleife | Picker → … → Booker → Picker | aus der Kartenzahl gerechnet, je Knoten aus seiner Rolle im Automaten |
| L5 | Debugging | Debugging → Rot-Prüfer → Bau → Prüfkette → Debugging | 3 Fixversuche, dann das Tor Architekturfrage. Keine Reparatur, bevor die Ursache feststeht |
| L6 | Äußere Schleife *(Anhang, optional)* | Detektor → `intent.md` → Triage → Phase 0 | die Bänder: 1σ protokollieren, 2σ mit `read_only` diagnostizieren, 3σ mit `edit` vorschlagen |

## Phasen

| # | Phase | Art | Was sie tut | Ausgabe |
|---|---|---|---|---|
| 0 | Eingang | Code | Ein Auftrag von extern liefert ein `intent.md`, das ins Projekt gelegt wird. Beginnst du direkt im Chat, gibt es nichts abzulegen | `intent.md` oder nichts |
| 1 | Klärung | Agent + Tor | Brainstorming über `intent.md` und Gespräch, bis du die Klärung abnimmst | abgenommene Klärung |
| 2 | Spec | Agent + LLM-Prüfung | Anforderungen und Entwurf in einem Dokument, deshalb keine eigene Entwurfsphase. Enthält auch Problem, Ziel, Randbedingungen und verworfene Alternativen; auf dem Ideenweg ist sie das erste Artefakt | `spec.md` |
| 3 | Einordnung | Agent | Marken des Vorhabens. *Offen: einfalten, weil der Plan die Marken inzwischen selbst vergibt* | Marken |
| 4 | Plan | Agent + LLM-Prüfung | Dateien, Reihenfolge, Risiken, Beweis. Schneidet die Aufgaben selbst, jede mit `type`, `area`, `size`, Abhängigkeiten und Beweis | `plan.md` mit markierten Aufgaben |
| 5 | Recherche | Agent | holt Wissen von außen, jede Behauptung mit Fundstelle; einziger Knoten mit Profil `mcp` | Digest |
| 6 | Test schreiben | Agent + Code | Test zuerst, und der Rot-Prüfer belegt, dass er fehlschlägt | Testdatei, rotes Protokoll |
| 7 | Bau | Agent | einziger Knoten mit `edit` am Produktivcode; liest seinen Aufgabenbrief, nie den ganzen Plan | Diff, Commit |
| 8 | Prüfkette | Code | Lint, Typen, Tests, Coverage, plus was die Marke verlangt: Sichtvergleich und WCAG bei `frontend`, Migrationslauf bei `data`, SAST bei `security` | Urteil |
| 9 | Codereview | Fächer | vier Linsen parallel: `bugs` (falsch), `compliance` (richtig gebaut, aber nicht das Verlangte), `security`, `simplicity`. Danach entdoppeln, nach Schwere ordnen, Kleinkram auf fünf deckeln | Befunde |
| 10 | Nacharbeit | Agent | arbeitet Befunde ein und darf einen begründet ablehnen; die Ablehnung steht im Journal | Änderung |
| 11 | Schlichtung | Agent | entscheidet, wo Befund und Plan sich widersprechen; die Antwort trägt `cost_if_wrong` | Entscheidung |
| 12 | Regelprüfung | Agent | kein `Any`, kein `type: ignore` ohne Begründung, Kommentare englisch, keine Testdatei angefasst, die nicht angefasst werden durfte | Urteil |
| 13 | Doku | Agent + Fächer | englische Fassung plus `.de.md`; der Fächer hält das Mermaid-Diagramm gegen den echten Graphen | Flow-Seite |
| 14 | Abschlussreview | Fächer | dieselben Codelinsen mit `scope: branch` auf dem stärksten Modell; hier wird der abgelegte Kleinkram vorgelegt | Befunde |
| 15 | Commit | Code | Nachricht, Sprachtor, und der Nachweis-Prüfer verlangt die Ausgabe des Prüfbefehls, bevor „grün" behauptet wird | Commit |
| 16 | Übergabe | Code + Tor | die Eval-Suite fährt den Harness gegen sich selbst; sinkt die Bestehensquote, braucht die Änderung eine Freigabe | Bestehensquote |

## Offen

- **Einordnung einfalten?** Der Planschreiber sieht Spec und Absicht und vergibt
  die Marken ohnehin. Ein Knoten, dessen Ausgabe der nächste überschreibt, ist
  ein Modellaufruf ohne Ertrag.
- **Picker:** Code- oder Agentenknoten.
- **Artefaktnamen des Playbooks** (`intent.md`, `spec.md`, `plan.md`) übernehmen?
- **Modell im `FlowContext`:** ohne das gibt es keinen Fächerknoten.
