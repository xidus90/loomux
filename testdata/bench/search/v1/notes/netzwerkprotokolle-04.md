---
titel: Hypertext Transfer Protocol
quelle: https://de.wikipedia.org/wiki/Hypertext_Transfer_Protocol
abgerufen: 2026-08-21
lizenz: CC BY-SA 4.0
thema: netzwerkprotokolle
---

## HTTP-Statuscodes

Jede HTTP-Anfrage wird vom Server mit einem HTTP-Statuscode beantwortet. Er gibt zum Beispiel Informationen darüber, ob die Anfrage erfolgreich bearbeitet wurde, oder teilt dem Client, also etwa dem Browser, im Fehlerfall mit, wo (zum Beispiel per Umleitung) beziehungsweise wie (zum Beispiel mit Authentifizierung) er die gewünschten Informationen (wenn möglich) erhalten kann.

1xx – Informationen
Die Bearbeitung der Anfrage dauert trotz der Rückmeldung noch an. Eine solche Zwischenantwort ist manchmal notwendig, da viele Clients nach einer bestimmten Zeitspanne (Zeitüberschreitung) automatisch annehmen, dass ein Fehler bei der Übertragung oder Verarbeitung der Anfrage aufgetreten ist, und mit einer Fehlermeldung abbrechen.
2xx – Erfolgreiche Operation
Die Anfrage wurde bearbeitet und die Antwort wird an den Anfragesteller zurückgesendet.
3xx – Umleitung
Um eine erfolgreiche Bearbeitung der Anfrage sicherzustellen, sind weitere Schritte seitens des Clients erforderlich. Das ist zum Beispiel der Fall, wenn eine Webseite vom Betreiber umgestaltet wurde, so dass sich eine gewünschte Datei nun an einem anderen Platz befindet. Mit der Antwort des Servers erfährt der Client im Location-Header, wo sich die Datei jetzt befindet.
4xx – Client-Fehler
Bei der Bearbeitung der Anfrage ist ein Fehler aufgetreten, der im Verantwortungsbereich des Clients liegt. Ein 404 tritt beispielsweise ein, wenn ein Dokument angefragt wurde, das auf dem Server nicht existiert. Ein 403 weist den Client darauf hin, dass es ihm nicht erlaubt ist, das jeweilige Dokument abzurufen. Es kann sich zum Beispiel um ein vertrauliches oder nur per HTTPS zugängliches Dokument handeln.
5xx – Server-Fehler
Es ist ein Fehler aufgetreten, dessen Ursache beim Server liegt. Zum Beispiel bedeutet 501, dass der Server nicht über die erforderlichen Funktionen (das heißt zum Beispiel Programme oder andere Dateien) verfügt, um die Anfrage zu bearbeiten.
Zusätzlich zum Statuscode enthält der Header der Server-Antwort eine Beschreibung des Fehlers in englischsprachigem Klartext. Zum Beispiel ist ein Fehler 404 an folgendem Header zu erkennen:
