package scad

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func TestModuleChildren(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		inside  []model3d.Coord3D
		outside []model3d.Coord3D
	}{
		{
			name: "Union",
			src: `
				module wrapper() {
					assert($children == 2);
					translate([0, 5, 0]) children();
				}
				wrapper() {
					sphere(1);
					translate([4, 0, 0]) sphere(1);
				}
			`,
			inside:  []model3d.Coord3D{model3d.XYZ(0, 5, 0), model3d.XYZ(4, 5, 0)},
			outside: []model3d.Coord3D{model3d.Origin, model3d.XYZ(2, 5, 0)},
		},
		{
			name: "Indexed",
			src: `
				module cutout() {
					difference() {
						children(index=$children - 1);
						children(0);
					}
				}
				cutout() {
					sphere(1);
					cube(4, center=true);
				}
			`,
			inside:  []model3d.Coord3D{model3d.XYZ(1.5, 0, 0)},
			outside: []model3d.Coord3D{model3d.Origin, model3d.XYZ(2.1, 0, 0)},
		},
		{
			name: "CallerScope",
			src: `
				r = 1;
				module wrapper() {
					r = 20;
					function count() = $children;
					assert(count() == 1);
					children();
				}
				{
					r = 2;
					module ball() { sphere(r); }
					wrapper() ball();
				}
				assert(r == 1);
			`,
			inside:  []model3d.Coord3D{model3d.XYZ(1.9, 0, 0)},
			outside: []model3d.Coord3D{model3d.XYZ(2.1, 0, 0)},
		},
		{
			name: "ChildBlockDeclarationsAndRepeatedSelection",
			src: `
				r = 10;
				module twice() {
					assert($children == 1);
					children(0);
					translate([5, 0, 0]) children();
				}
				twice() {
					module ball() { sphere(radius()); }
					function radius() = r;
					ball();
					r = 2;
				}
				assert(r == 10);
			`,
			inside:  []model3d.Coord3D{model3d.XYZ(1.9, 0, 0), model3d.XYZ(6.9, 0, 0)},
			outside: []model3d.Coord3D{model3d.XYZ(2.5, 0, 0), model3d.XYZ(7.1, 0, 0)},
		},
		{
			name: "StatementCountAndLazySelection",
			src: `
				module select() {
					assert($children == 4);
					children(0);
					children(2);
				}
				select() {
					if (false) sphere(99);
					assert(false, "unselected child");
					for (x = [0, 4]) translate([x, 0, 0]) sphere(1);
					unknown_module();
				}
			`,
			inside:  []model3d.Coord3D{model3d.Origin, model3d.XYZ(4, 0, 0)},
			outside: []model3d.Coord3D{model3d.XYZ(2, 0, 0)},
		},
		{
			name: "NestedForwarding",
			src: `
				module inner() {
					assert($children == 1);
					translate([0, 5, 0]) children();
				}
				module outer() {
					assert($children == 2);
					inner() children($children - 1);
					assert($children == 2);
					children(0);
				}
				outer() {
					sphere(1);
					translate([4, 0, 0]) sphere(1);
				}
			`,
			inside:  []model3d.Coord3D{model3d.Origin, model3d.XYZ(4, 5, 0)},
			outside: []model3d.Coord3D{model3d.XYZ(4, 0, 0), model3d.XYZ(0, 5, 0)},
		},
		{
			name: "EmptyAndUnusedChildren",
			src: `
				module empty() { children(); }
				module fallback() {
					module nested() {
						assert($children == 0);
						children();
					}
					assert($children == 1);
					nested();
					assert($children == 1);
					sphere(1);
				}
				empty();
				empty() {}
				empty() if (false) sphere(99);
				fallback() assert(false, "unused child");
			`,
			inside:  []model3d.Coord3D{model3d.Origin},
			outside: []model3d.Coord3D{model3d.XYZ(1.1, 0, 0)},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			solid := mustEvalSolid(t, tc.src)
			for _, p := range tc.inside {
				assertContains(t, solid, p, true)
			}
			for _, p := range tc.outside {
				assertContains(t, solid, p, false)
			}
		})
	}
}

func TestModuleChildrenShapeKinds(t *testing.T) {
	t.Run("2D", func(t *testing.T) {
		shape := mustEvalShape(t, `
			module wrapper() { children(); }
			wrapper() {
				circle(1);
				translate([4, 0]) circle(1);
			}
		`)
		if shape.Kind != ShapeSolid2D {
			t.Fatalf("expected 2D solid, got %v", shape.Kind)
		}
		if !shape.S2.Contains(model2d.XY(0, 0)) || !shape.S2.Contains(model2d.XY(4, 0)) ||
			shape.S2.Contains(model2d.XY(2, 0)) {
			t.Fatal("unexpected 2D child union")
		}
	})
	t.Run("SDF", func(t *testing.T) {
		shape := mustEvalShape(t, `
			module wrapper() { children(0); }
			wrapper() sphere_sdf(2);
		`)
		if shape.Kind != ShapeSDF3D {
			t.Fatalf("expected 3D SDF, got %v", shape.Kind)
		}
		assertSDFsEqual3D(t, shape.SDF3, &model3d.Sphere{Radius: 2},
			model3d.Ones(-3), model3d.Ones(3), 1e-8)
	})
}

func TestModuleChildrenFromUsedFile(t *testing.T) {
	prog, err := ParseFile("main.scad", `
		use <lib.scad>
		r = 2;
		wrapper() sphere(r);
	`)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: fstest.MapFS{
		"lib.scad": {Data: []byte(`
			r = 10;
			module wrapper() {
				assert($children == 1);
				translate([r, 0, 0]) children();
			}
		`)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, shape.S3, model3d.XYZ(11.9, 0, 0), true)
	assertContains(t, shape.S3, model3d.XYZ(12.1, 0, 0), false)
}

func TestModuleChildrenErrors(t *testing.T) {
	tests := []struct {
		name    string
		call    string
		wantErr string
	}{
		{"NegativeIndex", "children(-1);", "index out of range"},
		{"LargeIndex", "children(1);", "index out of range"},
		{"FractionalIndex", "children(0.5);", "index must be an integer"},
		{"NonFiniteIndex", "children(sqrt(-1));", "index must be an integer"},
		{"StringIndex", `children("0");`, "expected number"},
		{"ExtraArgument", "children(0, 1);", "too many positional args"},
		{"UnknownArgument", "children(foo=0);", `unknown argument "foo"`},
		{"DuplicateArgument", "children(index=0, index=0);", `duplicate argument "index"`},
		{"ChildrenOfChildren", "children() sphere(1);", "children() does not take children"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse("module wrapper() { " + tc.call + " } wrapper() sphere(1);")
			if err != nil {
				t.Fatal(err)
			}
			_, err = Eval(prog, Hooks{})
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
			if countErrorPositions(err) < 2 {
				t.Fatalf("expected child call and wrapper call positions, got %v", err)
			}
		})
	}
}

func TestModuleChildrenBindingsDoNotLeak(t *testing.T) {
	for _, stmt := range []string{"echo($children);", "children();"} {
		t.Run(stmt, func(t *testing.T) {
			prog, err := Parse(`module wrapper() { children(); } wrapper() sphere(1); ` + stmt)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Eval(prog, Hooks{})
			if err == nil || (!strings.Contains(err.Error(), "undefined variable") &&
				!strings.Contains(err.Error(), "unknown module/primitive")) {
				t.Fatalf("expected undefined children binding, got %v", err)
			}
		})
	}
}
