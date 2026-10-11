package scad

import (
	"strings"
	"testing"
)

func TestTrailingCommas(t *testing.T) {
	// Exercise every kind of comma-separated language construct, including
	// required/default parameters and comments between the comma and closer.
	src := `
module object(size, center=true, /* trailing */) {
  cube(size, center=center,);
}
function add(x, y=2,) = x+y;
f = function(x, y=3,) x*y;
assert(add(1,) == 3);
assert(f(2,) == 6);
assert((function(x,) x+1)(2,) == 3);
assert(let (x=2, y=3,) x+y == 5);
a = [for (i=[1,2,],) i*2,];
assert(a == [2,4,]);
for (i=[0,],) object(1,);
intersection_for (i=[0,],) object(1,);
`
	mustEvalShape(t, src)
	formatted, err := Format(src)
	if err != nil {
		t.Fatal(err)
	}
	mustEvalShape(t, formatted)
	for _, invalid := range []string{
		"module f(,) {}", "module f(x,,) {}", "function f(x,,) = x;",
		"f = function(,) 1;", "cache f(x,) {}", "cube(,);", "x=[,];",
	} {
		if _, err := Parse(invalid); err == nil {
			t.Errorf("accepted %q", invalid)
		}
	}
}

func TestParserLexerErrors(t *testing.T) {
	for _, src := range []string{"x=1; y=2; @", "x=1; y=2; /* unclosed", "cube(); // ok\n/* unclosed"} {
		if _, err := ParseFile("input.scad", src); err == nil || !strings.Contains(err.Error(), "input.scad:") {
			t.Errorf("ParseFile(%q): %v", src, err)
		}
	}
}
