---
titel: Domain Name System
quelle: https://de.wikipedia.org/wiki/Domain_Name_System
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

### Protokoll

DNS-Anfragen werden normalerweise per UDP Port 53 zum Namensserver gesendet. Der DNS-Standard fordert aber auch die Unterstützung von TCP für Fragen, deren Antworten zu groß für UDP-Übertragungen sind. Ursprünglich betrug die maximal zulässige Länge einer DNS-Nachricht über UDP 512 Bytes. Mit Extended DNS (EDNS) wurde diese Größenbeschränkung aufgehoben und kann variabel zwischen Client und Server gewählt werden. Beim DNS Flag Day 2020, einer Informationsinitiative von DNS-Software- und DNS-Service-Anbietern, wurde eine standardmäßige Maximallänge von 1232 Bytes empfohlen. Die maximal mögliche Nachrichtenlänge wird durch die Maximum Transmission Unit begrenzt. Der Einsatz von IP-Fragmentierung ist zwar möglich, wird aber nicht empfohlen.
Überlange Antworten werden abgeschnitten übertragen, sodass sie die maximal mögliche Nachrichtenlänge des Antwortenden nicht übersteigen, und mit dem Header-Flag Truncated (TC) als solches markiert. Der Anfragende kann daraufhin die Anfrage über TCP wiederholen. Bei TCP beträgt die maximale Nachrichtenlänge 65.535 Bytes. Die Verwendung von persistenten Verbindungen und Pipelining ist möglich. Zonentransfers werden stets über TCP durchgeführt, wobei die Nachrichtenlängenbeschränkung dafür nicht relevant ist.
