---
titel: Internet Control Message Protocol
quelle: https://de.wikipedia.org/wiki/Internet_Control_Message_Protocol
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

Das Internet Control Message Protocol (ICMP) dient in Rechnernetzwerken dem Austausch von Informations- und Fehlermeldungen über das Internet-Protokoll in der Version 4 (IPv4). Für IPv6 existiert ein ähnliches Protokoll mit dem Namen ICMPv6.
ICMP ist Bestandteil von IPv4, wird aber wie ein eigenständiges Protokoll behandelt. Es wird von jedem Router und jedem Rechner erwartet, dass sie ICMP „verstehen“. Die meisten ICMP-Pakete enthalten Diagnose-Informationen: Sie werden vom Router zur Quelle zurückgeschickt, wenn der Router Pakete verwirft, etwa weil beispielsweise das Ziel nicht erreichbar ist oder die TTL abgelaufen ist. Es gelten folgende Grundsätze:

ICMP benutzt IP als Kommunikationsbasis, indem es sich selbst als Protokoll einer höheren Schicht interpretiert, d. h. ICMP-Nachrichten werden in IP-Paketen gekapselt.
ICMP erkennt einige Fehlerzustände, macht aber IPv4 nicht zu einem zuverlässigen Protokoll.
ICMP analysiert Fehler in jedem IP-Paket, mit Ausnahme solcher, die eine ICMP-Nachricht tragen.
ICMP-Nachrichten werden nicht als Antwort auf Pakete an Zieladressen versendet, bei denen es sich um Multicast- oder Broadcast-Adressen handelt.
ICMP-Nachrichten antworten nur einer eindeutigen Quell-IP-Adresse.
