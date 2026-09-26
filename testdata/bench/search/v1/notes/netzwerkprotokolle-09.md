---
titel: Border Gateway Protocol
quelle: https://de.wikipedia.org/wiki/Border_Gateway_Protocol
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

### Arten von BGP-Nachrichten

BGP verwendet vier verschiedene Arten von Nachrichten im Protokoll:
OPEN
Wird nur zu Beginn einer Verbindung gesendet und muss mit einer KEEPALIVE-Nachricht beantwortet werden. Bei der OPEN-Nachricht werden die Parameter BGP-Version, AS-Nummer, Hold Timer, BGP Identifier sowie optionale Parameter mitgeschickt. Danach werden die Routeninformationen zwischen den Routern ausgetauscht.
UPDATE
Teilt eine Pfadänderung mit. Es können pro UPDATE-Nachricht gleichzeitig mehrere Pfade hinzugefügt und mehrere entfernt werden. UPDATE-Nachrichten sind das Kernstück von BGP.
NOTIFICATION
Beendet eine Verbindung und gibt Fehler- bzw. Statuscodes an. Alle Pfade, die über diese beendete Verbindung empfangen wurden, müssen nun gelöscht werden. Über ein BGP-Update würde dann verbreitet werden, dass diese Route nicht mehr verfügbar ist.
KEEPALIVE
Bestätigt die OPEN-Anfrage. Zur regelmäßigen Überprüfung, ob der verbundene Router noch online ist oder ob die Verbindung unterbrochen ist und die Pfade über den verbundenen Router somit ungültig geworden sind. Die Router, welche gerade eine BGP-Session aufgebaut haben, senden sich gegenseitig in regelmäßigen Abständen eine KEEPALIVE-Nachricht. Diese besteht nur aus dem Message Header. Im Attribut Hold Time einer OPEN-Nachricht wird die maximale Zeit angegeben, in der ein BGP-Router eine KEEPALIVE-Nachricht vom BGP-Partner der Session erwartet. Kommt innerhalb der Hold Time keine KEEPALIVE-Nachricht an, wird die BGP-Session mit einer NOTIFICATION beendet.
