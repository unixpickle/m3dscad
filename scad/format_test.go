package scad

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	cases := []struct{ name, input, want string }{
		{"empty", " \n\t", ""},
		{"chain", "module m(){\ntranslate([1,2,3])\nrotate(45)\ncube(1);\n}", "module m() {\n  translate([1, 2, 3])\n    rotate(45)\n    cube(1);\n}\n"},
		{"inline chain", "translate ( [1,2,3] )  rotate (45) cube (1);", "translate([1, 2, 3])\n  rotate(45)\n  cube(1);\n"},
		{"loop chain", "for(z=[a,b])clip(z=z)\nother()\ncube();", "for (z = [a, b])\n  clip(z=z)\n    other()\n    cube();\n"},
		{"chain block", "clip()\ndifference(){cube();sphere();}", "clip()\n  difference() {\n    cube();\n    sphere();\n  }\n"},
		{"nested control", "a()\nfor(i=[0:3])b()\nc()\nd();", "a()\n  for (i = [0:3])\n    b()\n      c()\n      d();\n"},
		{"trailing commas", "clip(x=a,y=b,)cube([1,2,]);", "clip(\n  x=a,\n  y=b,\n)\n  cube([\n    1,\n    2,\n  ]);\n"},
		{"params", "module m(a,b=3,){} function f(a,)=a+1;", "module m(\n  a,\n  b=3,\n) {}\nfunction f(\n  a,\n) = a + 1;\n"},
		{"operators", "x=a * b+c; y=a*b; z=[a + 3:b]; q=[0:3:9]; n=foo[len(foo) - 1];", "x = a*b + c;\ny = a * b;\nz = [a+3 : b];\nq = [0:3:9];\nn = foo[len(foo)-1];\n"},
		{"parentheses", "x=(a + b) * (c-d)+e^f*g; y=1-(2-3); z=(a^b)^c;", "x = (a + b)*(c - d) + e^f*g;\ny = 1 - (2 - 3);\nz = (a ^ b) ^ c;\n"},
		{"literals", "x=1.00E-2; s=\"a\\n\\\"b\"; use <a path/foo.scad>;", "x = 1.00E-2;\ns = \"a\\n\\\"b\";\nuse <a path/foo.scad>;\n"},
		{"if else", "if(a){cube();}else if(b) sphere();else {a();}", "if (a) {\n  cube();\n} else if (b)\n  sphere();\nelse {\n  a();\n}\n"},
		{"comments", "// hi  \nx=1;// trailing\n\n/* untouched\n    body\n */\ny=2; // end", "// hi  \nx = 1; // trailing\n\n/* untouched\n    body\n */\ny = 2; // end\n"},
		{"interior comments", "module /*name*/ m(){// start\nx=1/*number*/+2; /* end */\n}", "module /*name*/ m() { // start\n  x = 1 /*number*/ + 2; /* end */\n}\n"},
		{"list comment", "a([1, // one\n2, /* two */]);", "a([\n  1, // one\n  2, /* two */\n]);\n"},
		{"comprehension", "x=[\nfor(i=[0:3])\nlet(a=i+1,b=2,) [a,b]\n];", "x = [\n  for (i = [0:3])\n    let (\n      a = i + 1,\n      b = 2,\n    ) [a, b]\n];\n"},
		{"numeric member", "x=1 .x; y=1. .x;", "x = 1 .x;\ny = 1. .x;\n"},
		{"unary bodies", "f=function() -2; x=each -1; y=each(1);", "f = function () -2;\nx = each -1;\ny = each (1);\n"},
		{"ternary index", "x=a[b?1:2]; y=[(a?1:2):3];", "x = a[b ? 1 : 2];\ny = [(a ? 1 : 2):3];\n"},
		{"list comments without trailing comma", "cube(1, // size\n true); x=[\n// first\n1,2];", "cube(\n  1, // size\n  true\n);\nx = [\n  // first\n  1,\n  2\n];\n"},
		{"nested else", "if(a) if(b) {a();} else {b();} else {c();}", "if (a)\n  if (b) {\n    a();\n  } else {\n    b();\n  }\nelse {\n  c();\n}\n"},
		{"all expressions", "f=function(a=1,) a?b:c; x=let(a=1,)each [for(i=[0:2]) f(i)].x;", "f = function (\n  a=1,\n) a ? b : c;\nx = let (\n  a = 1,\n) each [for (i = [0:2]) f(i)].x;\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Format(c.input)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Fatalf("got:\n%s\nwant:\n%s", got, c.want)
			}
			checkFormatRoundTrip(t, c.input, got)
		})
	}
}

func checkFormatRoundTrip(t *testing.T, input, output string) {
	t.Helper()
	again, err := Format(output)
	if err != nil {
		t.Fatal(err)
	}
	if again != output {
		t.Fatalf("not idempotent:\n%s\nsecond pass:\n%s", output, again)
	}
	// Identical non-whitespace tokens (including comment and literal text) imply
	// identical syntax, even for grouping and dangling-else cases the AST erases.
	tokens := func(s string) []string {
		lx := NewLexer(s)
		lx.KeepComments = true
		var result []string
		for {
			tok, err := lx.Next()
			if err != nil {
				t.Fatal(err)
			}
			if tok.Kind == TokEOF {
				return result
			}
			result = append(result, s[tok.Pos.Offset:tok.End.Offset])
		}
	}
	if !reflect.DeepEqual(tokens(input), tokens(output)) {
		t.Fatal("formatter changed tokens")
	}
}

func TestFormatExamples(t *testing.T) {
	for _, pattern := range []string{"../testdata/formatter/*.scad", "../testdata/openscad_scad/*.scad", "../landing_page/examples/*.scad"} {
		paths, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range paths {
			t.Run(path, func(t *testing.T) {
				src, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				out, err := FormatFile(path, string(src))
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(path, "/formatter/") {
					want, err := os.ReadFile(path + ".golden")
					if err != nil {
						t.Fatal(err)
					}
					if out != string(want) {
						t.Fatalf("format differs from %s.golden:\n%s", path, out)
					}
				}
				checkFormatRoundTrip(t, string(src), out)
			})
		}
	}
}

func TestFormatInvalid(t *testing.T) {
	for _, src := range []string{"x=;", "cube(1", "x=1; /* unclosed", "x=1; @", "x=1; y=2; @", "x=1; y=2; /*"} {
		got, err := FormatFile("bad.scad", src)
		if err == nil || !strings.Contains(err.Error(), "bad.scad:") || got != "" {
			t.Errorf("FormatFile(%q) = %q, %v", src, got, err)
		}
	}
}

func FuzzFormat(f *testing.F) {
	for _, src := range []string{"x=a*b+c;", "a()\nb()\nc();", "x=[1,2,]; // ok", "x=[for(i=[0:2]) i];", "if(a)if(b)a();else b();", "x=a[1?2:3];"} {
		f.Add(src)
	}
	f.Fuzz(func(t *testing.T, src string) {
		out, err := Format(src)
		if err == nil {
			checkFormatRoundTrip(t, src, out)
		}
	})
}
