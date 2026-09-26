---
titel: OSI-Modell
quelle: https://de.wikipedia.org/wiki/OSI-Modell
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

### Schicht 3 – Vermittlungsschicht (Network Layer)

Die Vermittlungsschicht (englisch network layer; auch Paketebene, Netzwerkschicht oder Paketvermittlungsebene) sorgt bei leitungsorientierten Diensten für das Schalten von Verbindungen und bei paketorientierten Diensten für die Weitervermittlung von Datenpaketen sowie die Stauvermeidung (englisch congestion avoidance). Die Datenübertragung geht in beiden Fällen jeweils über das gesamte Kommunikationsnetz hinweg und schließt die Wegsuche (Routing) zwischen den Netzwerkknoten ein. Da nicht immer eine direkte Kommunikation zwischen Absender und Ziel möglich ist, müssen Pakete von Knoten, die auf dem Weg liegen, weitergeleitet werden. Weitervermittelte Pakete gelangen nicht in die höheren Schichten, sondern werden mit einem neuen Zwischenziel versehen und an den nächsten Knoten gesendet.
Zu den wichtigsten Aufgaben der Vermittlungsschicht zählt das Bereitstellen netzwerkübergreifender Adressen, das Routing bzw. der Aufbau und die Aktualisierung von Routingtabellen und die Fragmentierung von Datenpaketen. Aber auch die Aushandlung und Sicherstellung einer gewissen Dienstgüte fällt in den Aufgabenbereich der Vermittlungsschicht.
Neben dem Internet Protocol zählen auch die NSAP-Adressen zu dieser Schicht. Da ein Kommunikationsnetz aus mehreren Teilnetzen unterschiedlicher Übertragungsmedien und -protokolle bestehen kann, sind in dieser Schicht auch die Umsetzungsfunktionen angesiedelt, die für eine Weiterleitung zwischen den Teilnetzen notwendig sind.
Hardware auf dieser Schicht: Router, Layer-3-Switch (BRouter).
Protokolle und Normen: X.25, ISO 8208, ISO 8473 (CLNP), ISO 9542 (ESIS), IP, IPsec, ICMP, VRF-Instanz.
