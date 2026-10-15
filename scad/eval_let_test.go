package scad

import (
	"strings"
	"testing"

	"github.com/unixpickle/model3d/model3d"
)

func TestLetModule(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
	}{
		{"SingleChild", `let(size = 2) cube(size);`},
		{"SequentialBindings", `let(a = 1, size = a + 1,) cube(size);`},
		{"ShadowOuter", `size = 1; let(size = size + 1) cube(size); assert(size == 1);`},
		{"Nested", `let(size = 1) let(size = size + 1) cube(size);`},
		{"Block", `let(size = 2) { assert(size == 2); cube(size); }`},
		{"ChildScope", `let(size = 1, original = size) { size = 2; assert(original == 1); cube(size); }`},
		{"EmptyBindings", `let() cube(2);`},
		{"Loop", `for(size = [1]) let(size = size + 1) cube(size);`},
		{"Conditional", `let(size = 2) if(size == 2) cube(size); else cube(10);`},
		{"ModuleArgument", `module box(size) { cube(size); } let(size = 2) box(size);`},
		{"Children", `module wrapper() { children(); } let(size = 2) wrapper() cube(size);`},
		{"EnclosingChildren", `module wrapper() { let(size = 2) scale(size) children(); } wrapper() cube(1);`},
		{"MultipleChildren", `let(size = 2) { cube([1, size, size]); translate([1, 0, 0]) cube([1, size, size]); }`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			solid := mustEvalSolid(t, tc.src)
			if solid.Min() != model3d.Origin || solid.Max() != model3d.XYZ(2, 2, 2) {
				t.Fatalf("unexpected bounds: %v to %v", solid.Min(), solid.Max())
			}
			assertContains(t, solid, model3d.XYZ(0.5, 1, 1), true)
			assertContains(t, solid, model3d.XYZ(1.5, 1, 1), true)
			assertContains(t, solid, model3d.XYZ(2.1, 1, 1), false)
		})
	}
}

func TestLetModule2D(t *testing.T) {
	shape := mustEvalShape(t, `let(size = 2) square(size);`)
	if shape.Kind != ShapeSolid2D || shape.S2.Max().X != 2 || shape.S2.Max().Y != 2 {
		t.Fatalf("unexpected 2D shape: %v", shape)
	}
}

func TestLetModuleNoGeometry(t *testing.T) {
	for _, src := range []string{
		`let();`,
		`let(size = 2);`,
		`let(size = 2) {}`,
		`let(size = 2) assert(size == 2);`,
		`let(size = 2) if(false) cube(size);`,
	} {
		t.Run(src, func(t *testing.T) {
			prog, err := Parse(src)
			if err != nil {
				t.Fatal(err)
			}
			shapes, err := evalStmts(newEnv(Hooks{}), prog.Stmts)
			if err != nil {
				t.Fatal(err)
			}
			if len(shapes) != 0 {
				t.Fatalf("expected no geometry, got %d shapes", len(shapes))
			}
		})
	}
}

func TestLetModuleErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		src  string
		want string
	}{
		{"Positional", `let(2) cube(1);`, "let() requires named bindings"},
		{"Duplicate", `let(size = 1, size = 2) cube(size);`, `cannot redeclare variable "size"`},
		{"ForwardReference", `let(a = b, b = 2) cube(a);`, `undefined variable "b"`},
		{"BindingDoesNotLeak", `let(size = 2) cube(size); cube(size);`, `undefined variable "size"`},
		{"ChildDoesNotLeak", `let() { size = 2; cube(size); } cube(size);`, `undefined variable "size"`},
		{"InvalidChild", `let(size = 2) cube(missing);`, `undefined variable "missing"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Eval(prog, Hooks{})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}
