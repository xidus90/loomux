---
titel: Address Resolution Protocol
quelle: https://de.wikipedia.org/wiki/Address_Resolution_Protocol
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

## Paketformat

Das ARP-Paket schließt sich an den Ethernet-MAC-Header an. Das Typfeld im Ethernet-Frame wird auf 0x0806 (2054) gesetzt. Diese Nummer ist für das ARP-Protokoll reserviert. Dadurch lassen sich ARP-Pakete von Paketen anderer Protokolle wie beispielsweise IP unterscheiden.
Da das Paket sehr kurz ist, müssen in der Regel im Ethernet-Frame zwischen ARP-Paket und CRC zusätzliche Bytes eingefügt werden (Padding), um die minimale Framelänge von 64 Bytes zu erreichen.
Obwohl ARP ursprünglich für IPv4 und MAC-Adressen entwickelt wurde, sind im Paket Adresstypen und Protokollgrößenfelder vorgesehen. Dadurch ist ARP auch für andere Protokolle geeignet. Für IPv6 könnten die Protokolladressgröße statt auf vier auf 16 Bytes gesetzt und die Adressfelder auf 128 Bits (= 16 Byte) verlängert werden, jedoch wird ARP für IPv6 durch das Neighbor Discovery Protocol (NDP) ersetzt, welches auf ICMPv6 basiert.

Hardwareadresstyp (2 Byte) enthält den Typ der MAC-Adresse im Paket (für Ethernet: 1).
Protokolladresstyp (2 Byte) enthält den Protokolltyp, der für die MAC-Adresse angefordert wird (für IPv4-Adressen: 0x0800 (2048)).
Hardwareadressgröße (1 Byte) enthält die Größe der MAC-Adresse (für Ethernet: 6).
Protokolladressgröße (1 Byte) enthält die Größe des Protokolls (für IPv4: 4).
Operation (2 Byte) enthält den Wert, der angibt, welche Operation ausgeführt werden soll (1 für ARP-Anforderung, 2 für ARP-Antwort).
Quell-MAC-Adresse (6 Byte) enthält in einer ARP-Anforderung die MAC-Adresse des Senders. In einer ARP-Antwort enthält es die MAC-Adresse des antwortenden Hosts oder Next-Hop-Routers.
Quell-IP-Adresse (4 Bytes bei IPv4) enthält bei einer ARP-Anforderung die IP-Adresse des anfragenden Hosts. In einer ARP-Antwort enthält es die IP-Adresse des antwortenden Hosts oder Next-Hop-Routers.
Ziel-MAC-Adresse (6 Byte) wird in einer ARP-Anforderung ignoriert (meist 00:00:00:00:00:0016). In einer ARP-Antwort enthält es die MAC-Adresse des anfragenden Hosts.
Ziel-IP-Adresse (4 Bytes bei IPv4) ist bei einer ARP-Anforderung die IP-Adresse des gesuchten Hosts. In einer ARP-Antwort enthält es die IP-Adresse des anfragenden Hosts.
