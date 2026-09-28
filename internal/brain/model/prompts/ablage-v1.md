<!-- version: ablage-v1
Die Ausgabeform wird am Endpunkt erzwungen (`format`-Schema), nicht hier
erbeten -- Spec Paragraf 5. Der Prompt nennt sie trotzdem, weil das Schema
allein nicht sagt, welcher Scope gemeint ist.
-->
Ordne den folgenden Text genau einem der Bereiche zu. Waehle **nur** aus
dieser Liste; erfinde keinen Bereich:

{scopes}

Antworte als JSON mit den Feldern `scope` (ein Eintrag aus der Liste) und
`grund` (ein kurzer deutscher Satz).

Der Text:

{text}
