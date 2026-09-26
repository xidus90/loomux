---
titel: Dynamic Host Configuration Protocol
quelle: https://de.wikipedia.org/wiki/Dynamic_Host_Configuration_Protocol
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

## Aufbau eines DHCP-Paketes

op (1 Byte): Information, ob es sich um eine Anforderung (request = 1) oder eine Antwort (reply = 2) handelt
htype (1 Byte): Netztyp (z. B. 1 = Ethernet, 6 = IEEE 802 Netzwerke oder 7 = ARCNET)
hlen (1 Byte): Länge der physikalischen Netzadresse in Bytes (z. B. 6 = MAC/Ethernet-Adresse)
hops (1 Byte, optional): Anzahl der DHCP-Relay-Agents auf dem Datenpfad
xid (4 Byte): ID der Verbindung zwischen Client und Server
secs (2 Byte): Zeit in Sekunden seit dem Start des Clients
flags (2 Byte): Z. Zt. wird nur das erste Bit verwendet (zeigt an, ob der Client noch eine gültige IP-Adresse hat), die restlichen Bits sind für spätere Protokollerweiterungen reserviert
ciaddr (4 Byte): Client-IP-Adresse
yiaddr (4 Byte): eigene IP-Adresse
siaddr (4 Byte): Server-IP-Adresse
giaddr (4 Byte): Relay-Agent-IP-Adresse
chaddr (16 Byte): Client-MAC-Adresse
sname (64 Byte): Name des DHCP-Servers, falls ein bestimmter gefordert wird (enthält C-String), Angabe optional
file (128 Byte): Name einer Datei (z. B. System-Kernel), die vom Server per TFTP an den Client gesendet werden soll (enthält C-String), Angabe optional
options (variabel, optional): DHCP-Parameter und -Optionen (Beschreibung in RFC 2132) – Die Optionen können bis zu 312 Bytes lang sein, so dass ein IP-Paket von bis zu 576 Bytes (236 Bytes DHCP-Header + 312 Bytes DHCP-Options + 8 Bytes UDP-Header + 20 Bytes IPv4-Header) Länge auftreten kann. Eine größere maximale Byteanzahl kann zwischen Server und Client ausgehandelt werden.
