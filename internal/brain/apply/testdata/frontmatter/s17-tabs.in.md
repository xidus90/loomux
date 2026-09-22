---
single: 'x	y'
double: "x	y"
escaped: "a\"b\\	c"
comment: b # c	d
# 	full comment line
literal: |
  x	y
  	z

  after blank
header_comment: | # c	d
  x
folded: >
  x	y
multi_double: "x
	y"
multi_single: 'x
	y'
quote_in_single: 'it''s	ok'
explicit: |2
   	x
last: |
  end	line
---
