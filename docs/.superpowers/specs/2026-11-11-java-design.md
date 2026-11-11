# Java im Code-Graphen, in der Prüfkette und in der Kompatibilitätsliste

**Stand:** 2026-11-11 · Teilprojekt 3 von
`2026-11-10-graph-sprachen-programm.md` · nutzt `Result.Enclosing` nicht:
Java hat ein Paket je Datei (`Result.Package`)

## Ziel

Java steht in Graph, Prüfkette und Kompatibilitätsliste auf der Stufe von
Python, für jede von Eclipse Temurin gepflegte Version: 8, 11, 17, 21, 25
und 27.

## Nicht-Ziele

- Keine Typinferenz für lokale Variablen (`obj.m()` bleibt ohne Kante).
- Kein Lesen von `pom.xml` oder `build.gradle` für Quellwurzeln; das Paket
  steht in der Datei.
- Kein Kotlin, kein Groovy.

## Extraktor `internal/code/extract/java`

- **Grunddaten:** Sprache `java`, Endung `.java`, Version `java/1@<Parser>`,
  Grammatik `gotreesitter/grammars/java`.
- **Knoten:**
  - Typen `class`, `interface`, `enum`, `record` und `@interface`. Die Art
    ist `interface` für Schnittstellen und Annotationstypen, sonst `class`.
  - Verschachtelte Typen heißen `Outer.Inner`.
  - Methoden und Konstruktoren (Konstruktor heißt wie der Typ) auch in
    `enum_body_declarations`. Kompakte Konstruktoren eines Records heißen
    wie der Record.
  - Methoden auf oberster Ebene (kompakte Quelldatei, Java 25) sind Art
    `function`.
- **Weitere Felder:**
  - `Exported`: `public` oder `protected`, oder Mitglied einer Schnittstelle
  - `Result.Package`: der Name aus `package a.b;`, sonst `""`
- **Imports** (`extract.Import` plus `imports`-Rohkante):
  - `import a.b.C;`
  - `import a.b.*;`
  - `import static a.b.C.m;` und `import static a.b.C.*;` (Alias `static`
    oder das Feld, das C# dafür nutzt)
  - `import module m;`
- **Rohkanten:**
  - `extends` je `superclass` und je Eintrag in `super_interfaces` bzw.
    `extends_interfaces`. Der Resolver entscheidet zwischen `extends` und
    `implements` nach der Art des Ziels.
  - `calls`:
    - `m()`
    - `this.m()`, `super.m()`
    - `X.m()` und `a.b.X.m()` (Receiver als Kette)
    - `new X()` mit `Member "new"`
    - `obj.m()` (Receiver unbekannt)
- **Fehlertoleranz:** wie C#. Kopf in einem ERROR heißt kein Knoten, der
  Rest der Datei wird gelesen.
- **Grammatik-Probe vom 2026-11-11** (gotreesitter v0.57.0): Java 8 bis 21
  parsen ohne Fehler. Java 25 hat zwei Lücken:
  - `import module java.base;` gibt einen ERROR im Import, der Rest kommt an.
  - Eine Anweisung vor `super(x)` im Konstruktor (flexible
    Konstruktorrümpfe) gibt einen ERROR im Rumpf; die Mitglieder danach
    kommen an.
  - Beide Lücken stehen in `support.md`. Das Syntax-Fixture 25 verlangt
    nur, was ankommt.

## Resolver `internal/code/resolve/java.go`

**Typauflösung** eines Namens N in Datei F (Paket P), in dieser Folge:
1. verschachtelte Typen des umgebenden Typs und seiner äußeren Typen
2. Typen der Datei
3. Einzel-Typ-Importe (`import a.b.N;`)
4. Typen des Pakets P (alle Dateien mit demselben Paketnamen)
5. Typen der `*`-Importe
6. Voll qualifizierte Namen

Ein Treffer ergibt eine Kante, mehrere auf einer Stufe `ReasonAmbiguous`.

**Kanten:**
- **`imports`:**
  - Einzel-Typ-Import → die Datei, die den Typ deklariert
  - `*`-Import → Vertreterdatei des Pakets (kleinster Pfad unter den
    Nicht-Testdateien)
  - statischer Import → die Datei des Typs
  - fremdes Paket → keine Kante
  - eine Kante je Paar, kleinste Zeile
- **`extends`/`implements`:** nach der Art des Ziels. `Unresolved` gibt es
  nur hier.
- **`calls`:**
  - `m()`: Typ, seine Basen (zyklenfest), die umgebenden Typen, dann
    statische Importe
  - `this.m()`, `super.m()` (erst die Basen)
  - `X.m()` über die Typauflösung
  - `new X()`: Konstruktor, sonst der Typ
  - Überladung: die erste Deklaration

## Restliche Graph-Stellen

- **`blast.IsTestPath`:**
  - Segment `src/test/`
  - oder ein Dateiname nach `*Test.java`, `*Tests.java`, `*IT.java` oder
    `Test*.java`
  - Negativfall `Contest.java`.
- **`repomap`, `isFileQuery`:** `.java` steht schon in `isFileQuery`, also
  nachsehen.
- **`corpus.pruneDirs`:** `target` und `build` stehen drin.

## Prüfkette: Stack `java`

**Erkennung:**
- `pom.xml` → `java` mit Variante `maven`
- `build.gradle`, `build.gradle.kts`, `settings.gradle` oder
  `settings.gradle.kts` → `java` mit Variante `gradle`
- Liegen beide vor, gewinnt Maven, mit Hinweis.
- Ein `build.gradle.kts` ohne `.java`-Datei (reines Kotlin) bleibt
  unerkannt.

**Lanes (Maven):**
- lint: `mvn -B -q checkstyle:check`, nur wenn das Projekt das
  Checkstyle-Plugin im `pom.xml` konfiguriert (`when_found`). Sonst gibt es
  keine Lint-Lane.
  - Ein Aufruf von außen mit festem Checkstyle geht nicht: Checkstyle 9.3
    (der letzte Stand für Java 8) liest die Syntax von Java 21 nicht, und
    14.4 braucht Java 21 zum Laufen. Die Version gehört darum dem Projekt.
  - Die Fixtures pinnen sie je Laufzeit: 9.3 für 8 und 11, 12.x für 17
    (im Plan nachschlagen), 14.4.0 ab 21.
- types: `mvn -B -q -DskipTests compile test-compile`
- test: `mvn -B -q test`
- coverage: JaCoCo-Plugin (0.8.16, exakt) mit `prepare-agent`, `test` und
  `report`, dann `{loomux} check linecov --report target/site/jacoco/jacoco.xml`
  (linecov kennt `jacoco` schon)
- graph und rules wie Python

**Lanes (Gradle):** `./gradlew` wenn ein Wrapper da ist, sonst `gradle`.
- lint: `check -x test`
- types: `compileJava compileTestJava`
- test: `test`
- coverage: nur wenn `jacoco` im Build steht (`when_found`):
  `test jacocoTestReport`, Bericht unter `build/reports/jacoco/test/jacocoTestReport.xml`

**Weiteres:**
- `carrierRank`: `… csharp, cpp, java`
- `IsSource` und `contentExts` + `.java`
- `support.md` (en, de) mit der Zeile `java`
- Werkzeug-Pins in `[verify.java].tools`, wenn die Platzhalter-Mechanik das
  trägt; sonst fest im Preset (im Plan nachsehen)

## Kompatibilität

- `runtime = "java"`, `runtimeOf["java"] = "java"`
- **Einträge:**

  | Version | Fixture | Arten |
  |---|---|---|
  | 8, 11 | `java-maven-junit5` | lint, types, test, coverage, graph |
  | 17 | `java-maven-junit6-17` (Checkstyle 12.x) | lint, types, test, coverage, graph |
  | 21, 25, 27 | `java-maven-junit6` (Checkstyle 14.4.0) | lint, types, test, coverage, graph |
  | 21 | `java-gradle-junit6` | lint, types, test, coverage, graph |
  | 8, 11, 17, 21, 25, 27 | `java-syntax-<v>` | graph |

  JUnit 6 verlangt Java 17, darum JUnit 5 für 8 und 11. Die Versionen
  werden im Plan recherchiert und exakt gepinnt. Das gilt auch für Maven,
  Gradle und die Untergrenzen der Plugins; ein Werkzeug, das 8 nicht mehr
  trägt, bekommt für 8 eine ältere, noch gepflegte Version oder fällt mit
  Begründung weg.
- **Syntax-Fixtures:** die Formen der Recherche.
  - 27 hat keine neue finale Syntax. Das Fixture 27 belegt die Formen von
    25 auf 27.
- **Scout:**
  - Releases aus `https://api.adoptium.net/v3/info/available_releases`
  - Vorabversion ist die nächste Feature-Version mit EA-Builds
  - Supportende über endoflife.date `eclipse-temurin`
- **CI:** `actions/setup-java` (`distribution: temurin`, SHA-gepinnt) mit
  der Version des Eintrags. Maven aus dem Runner, sonst über die Action;
  Gradle über den Wrapper des Fixtures.

## Tests und Abnahme

Wie bei C#:
- Extraktor-, Resolver- und Spiegel-Tests
- Probelauf auf guava und spring-petclinic: kein Absturz, ParseErrors wie in
  der Parse-Prüfung
- Der Graph dieses Repos bleibt bis auf den Stempel gleich.
