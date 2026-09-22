---
1: int key
true: bool key
~: null key
2026-01-01: date key
2026-01-01 10:00:00: datetime key
1.5: float key
1e3: exponent without dot
.inf: inf key
"": empty key
"multi\nline key": multi
kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk: key of 122
kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk: key of 123
'yes': quoted yes key
nested:
  1: int
  true: bool
  ~: null
  2026-01-01: date
  1.5: float
  "": empty
  kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk: key of 123
  "multi\nline": multi
---
