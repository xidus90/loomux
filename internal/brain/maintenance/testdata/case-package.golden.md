---
case: a-2026-09-20-abcd
generated.at: 2026-09-20T08:00:00+02:00
segments: 6
---

## D1 — Diff, # ohne verifizierten Vorzustand: das Paket führt nur den neuen Stand

```
# ohne verifizierten Vorzustand: das Paket führt nur den neuen Stand
--- /dev/null
+++ src/ä eins.go
@@ -0,0 +1 @@
+neu
```

## D2 — Diff, --- src/zwei.go (HEAD)

```
--- src/zwei.go (HEAD)
+++ src/zwei.go
@@ -1 +1 @@
-eins
+zwei
```

## W1 — Wiki, eingerueckt, mit einem einsamen 

```
eingerueckt, mit einem einsamen 
 darin
```

## W2 — Wiki, zweiter Absatz

```
zweiter Absatz
```

## Q1 — Quelle, src/ä eins.go

```
doc_id: d1
resource: src/ä eins.go
```

## Q2 — Quelle, src/zwei.go

```
doc_id: d2
resource: src/zwei.go
```
