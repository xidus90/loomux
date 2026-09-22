---
literal: |
  line one
  line two
kept: |+
  kept

stripped: |-
  stripped
folded: >
  folded
  text
space_break: "trailing space \nnext"
break_space: "a\n  b"
blank_lines: "a\n\nb"
leading_break: "\nlead"
trailing_breaks: "end\n\n"
only_break: "\n"
unicode_breaks: "a\u2028b\x85c"
mixed_breaks: "a\n\u2028b"
list:
  - "one\ntwo"
  - |
    three
    four
---
