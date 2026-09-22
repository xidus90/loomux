---
plain0: a퟿b
double0: "a\x01퟿b"
plain1: ab
double1: "a\x01b"
plain2: a�b
double2: "a\x01�b"
plain3: a𐀀b
double3: "a\x01\U00010000b"
plain4: a b
double4: "a\x01 b"
plain5: a~b
double5: "a\x01~b"
plain6: aÿb
double6: "a\x01ÿb"
generated:
  at: '2026-09-22T08:16:27.936837+00:00'
verified:
- by: human:tester
  at: '2026-09-22T08:16:27.936837+00:00'
---
body
