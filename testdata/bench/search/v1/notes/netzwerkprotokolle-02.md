---
titel: User Datagram Protocol
quelle: https://de.wikipedia.org/wiki/User_Datagram_Protocol
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

## UDP-Datagramm

Neben den zu übertragenden Nutzdaten werden weitere Informationen mitgesendet, die sich immer am Anfang einer UDP-Botschaft befinden, im sogenannten Header. Der UDP-Header besteht aus vier Datenfeldern, die alle jeweils 16 Bit groß sind:

Quell-Port
gibt die Port-Nummer des sendenden Prozesses an. Diese Information wird benötigt, damit der Empfänger auf das Paket antworten kann. Da UDP verbindungslos ist, ist der Quell-Port optional und kann auf den Wert „0“ gesetzt werden (für den Fall, dass keine Antwortpakete erwartet werden und nur Pakete zum Empfänger gesendet werden sollen).
Ziel-Port
gibt an, welcher Prozess das Paket empfangen soll.
 Längenfeld
gibt die Länge des Datagramms, bestehend aus den Daten und dem Header, in Oktetten an. Der kleinstmögliche Wert sind 8 Oktette (bzw. Byte). Das Längenfeld legt eine theoretische Obergrenze von 216−1 = 65.535 Bytes (8 Byte Header + 65.527 Bytes Nutzdaten) fest. Die tatsächlich verfügbare Länge der Nutzdaten ist bedingt durch das zugrundeliegende IP-Protokoll jedoch auf 65.507 Bytes (65.535 – 8 Byte UDP Header – 20 Byte IP Header) bei Verwendung von IPv4 und 65.487 Bytes (65.535 – 8 Bytes UDP Header – 40 Bytes IPv6 Header) bei Nutzung von IPv6 beschränkt.
Prüfsummenfeld
es kann eine 16 Bit große Prüfsumme mitgesendet werden. Die Prüfsumme wird über den sogenannten Pseudo-Header, den UDP-Header und die Daten gebildet. Die Prüfsumme ist optional, wird aber in der Praxis fast immer benutzt. Wird die Prüfsumme nicht genutzt, wird diese auf „0“ gesetzt.
Datenfeld
es enthält die eigentlichen Nutzdaten, auch Payload genannt. Das Feld ist optional und kann theoretisch auch komplett fehlen, was in der Praxis aber eigentlich nie vorkommt. Das Datenfeld besteht immer aus einer geraden Anzahl Oktette. Am Ende freibleibende Oktette werden mit Nullen aufgefüllt.
