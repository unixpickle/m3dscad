package scad

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/unixpickle/model3d/model3d"
)

func mustEvalShape(t *testing.T, src string) ShapeRep {
	t.Helper()
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	shape, err := Eval(prog)
	if err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	return shape
}

func mustEvalSolid(t *testing.T, src string) model3d.Solid {
	t.Helper()
	shape := mustEvalShape(t, src)
	solid, err := shapeToSolid3D(shape)
	if err != nil {
		t.Fatalf("eval to solid failed: %v", err)
	}
	return solid
}

func shapeToSolid3D(shape ShapeRep) (model3d.Solid, error) {
	switch shape.Kind {
	case ShapeSolid3D:
		return shape.S3, nil
	case ShapeMesh3D:
		return shape.M3.Solid(), nil
	case ShapeSDF3D:
		return model3d.SDFToSolid(shape.SDF3, 0), nil
	case ShapeSolid2D, ShapeMesh2D, ShapeSDF2D:
		return nil, fmt.Errorf("2D output not supported")
	default:
		return nil, fmt.Errorf("unsupported output kind")
	}
}

func assertContains(t *testing.T, s model3d.Solid, p model3d.Coord3D, want bool) {
	t.Helper()
	got := s.Contains(p)
	if got != want {
		t.Fatalf("contains(%v) = %v, want %v", p, got, want)
	}
}

func TestSolidsIntegration(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		inside  []model3d.Coord3D
		outside []model3d.Coord3D
	}{
		{
			name: "Union",
			src: `
				union() {
					sphere(r=1);
					translate([2,0,0]) sphere(r=1);
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.5, 0, 0),
				model3d.XYZ(2.5, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(-1.1, 0, 0),
				model3d.XYZ(3.1, 0, 0),
			},
		},
		{
			name: "Difference",
			src: `
				difference() {
					cube(size=[2,2,2], center=true);
					sphere(r=1);
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.9, 0.9, 0.9),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 0),
				model3d.XYZ(2, 0, 0),
			},
		},
		{
			name: "Intersection",
			src: `
				intersection() {
					cube(size=[2,2,2], center=true);
					sphere(r=1);
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 0.5),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 0),
				model3d.XYZ(0, 0, 1.1),
			},
		},
		{
			name: "TranslateRotate",
			src: `
				translate([1,2,3]) rotate([0,0,90]) cube(size=[2,4,2], center=true);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(1+1.5, 2, 3),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1, 2+1.5, 3),
			},
		},
		{
			name: "Scale",
			src: `
				scale([2,1,1]) sphere(r=1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(1.5, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0, 1.1, 0),
			},
		},
		{
			name: "Module",
			src: `
				module make_ring(r, h) {
					difference() {
						cylinder(h=h, r=r, center=true);
						cylinder(h=h, r=r-0.5, center=true);
					}
				}
				make_ring(2, 2);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(1.75, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.2, 0, 0),
				model3d.XYZ(0, 0, 2.1),
			},
		},
		{
			name: "Function",
			src: `
				function r(x) = x + 1;
				module place(z) {
					translate([0,0,z]) sphere(r=r(1));
				}
				place(1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 2.9),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0, 0, -1.1),
				model3d.XYZ(0, 0, 4.2),
			},
		},
		{
			name: "NestedScopes",
			src: `
				r = 1;
				translate([5,0,0]) {
					r = 2;
					sphere(r=r);
					translate([0,0,5]) sphere(r=1);
				}
				sphere(r=r);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(5, 0, 0),
				model3d.XYZ(5, 0, 6),
				model3d.XYZ(0.5, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(2.1, 0, 0),
				model3d.XYZ(5, 0, 7.1),
			},
		},
		{
			name: "LinearExtrudeCircle",
			src: `
				linear_extrude(height=2) circle(r=1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.5, 0, 0.5),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 0.5),
				model3d.XYZ(0, 0, 2.1),
			},
		},
		{
			name: "LinearExtrudeSquareTransforms",
			src: `
				linear_extrude(height=3, center=true)
					translate([1,2,0]) rotate([0,0,90]) square(size=[2,4], center=true);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(1, 2, 0),
				model3d.XYZ(2.5, 2, 1),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1, 4.1, 0),
				model3d.XYZ(1, 2, 2.0),
			},
		},
		{
			name: "BinaryExprInArgs",
			src: `
				sphere(3-2);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 0),
			},
		},
		{
			name: "PowFunctionInArgs",
			src: `
				sphere(r=pow(2, 3));
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(7.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(8.1, 0, 0),
			},
		},
		{
			name: "TrigFunctionsUseDegrees",
			src: `
				union() {
					sphere(r=sin(90)*2);
					translate([5,0,0]) sphere(r=abs(cos(180)));
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(1.9, 0, 0),
				model3d.XYZ(5.8, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(2.1, 0, 0),
				model3d.XYZ(6.1, 0, 0),
			},
		},
		{
			name: "RangeLen",
			src: `
				sphere(r=len([0:2:10]));
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(5.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(6.1, 0, 0),
			},
		},
		{
			name: "ConcatWithRange",
			src: `
				vals = concat([0,1], [2:4], [5]);
				sphere(r=len(vals));
				translate([0, 0, 100]) sphere(r=vals[5]);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(5.9, 0, 0),
				model3d.XYZ(4.9, 0, 100),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(6.1, 0, 0),
				model3d.XYZ(5.1, 0, 100),
			},
		},
		{
			name: "DirectRangeIndex",
			src: `
				translate([0, 0, 50]) sphere(r=[0:2:10][3]);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(5.9, 0, 50),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(6.1, 0, 50),
			},
		},
		{
			name: "ForStatementRange",
			src: `
				for (a = [1:3])
					translate([a*4, 0, 0]) sphere(r=1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(4.5, 0, 0),
				model3d.XYZ(8.5, 0, 0),
				model3d.XYZ(12.5, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 0),
				model3d.XYZ(16, 0, 0),
			},
		},
		{
			name: "ForStatementMultipleBindings",
			src: `
				for (a = [0:1], b = [0:1])
					translate([a*5, b*5, 0]) sphere(r=1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.5, 0.5, 0),
				model3d.XYZ(5.5, 0.5, 0),
				model3d.XYZ(0.5, 5.5, 0),
				model3d.XYZ(5.5, 5.5, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(3, 3, 0),
			},
		},
		{
			name: "ListComprehensionLetEach",
			src: `
				list = [ for (a = [1:4]) let (b = a*a, c = 2*b) each [a, b, c] ];
				sphere(r=list[5]);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(7.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(8.1, 0, 0),
			},
		},
		{
			name: "IntersectionFor",
			src: `
				intersection_for (n = [1:6]) {
					rotate([0, 0, n * 60])
						translate([5,0,0])
						sphere(r=12);
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(20, 0, 0),
			},
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

func TestExpressionAssignments(t *testing.T) {
	tests := []struct {
		name string
		src  string
		var_ string
		want Value
	}{
		{
			name: "ListComprehensionMap",
			src: `
				input = [2, 3, 5, 7, 11];
				out = [for (a = input) a * a];
			`,
			var_: "out",
			want: List([]Value{Num(4), Num(9), Num(25), Num(49), Num(121)}),
		},
		{
			name: "EachFlattenRange",
			src: `
				a = [-2, each [1:2:5], each [6:-2:0], -1];
			`,
			var_: "a",
			want: List([]Value{Num(-2), Num(1), Num(3), Num(5), Num(6), Num(4), Num(2), Num(0), Num(-1)}),
		},
		{
			name: "ForLetEach",
			src: `
				list = [ for (a = [1:4]) let (b = a*a, c = 2*b) each [a, b, c] ];
			`,
			var_: "list",
			want: List([]Value{
				Num(1), Num(1), Num(2),
				Num(2), Num(4), Num(8),
				Num(3), Num(9), Num(18),
				Num(4), Num(16), Num(32),
			}),
		},
		{
			name: "MultiBindFlat",
			src: `
				flat = [ for (a = [0:2], b = [0:2]) a == b ? 1 : 0 ];
			`,
			var_: "flat",
			want: List([]Value{
				Num(1), Num(0), Num(0),
				Num(0), Num(1), Num(0),
				Num(0), Num(0), Num(1),
			}),
		},
		{
			name: "NestedMatrix",
			src: `
				identity = [ for (a = [0:2]) [ for (b = [0:2]) a == b ? 1 : 0 ] ];
			`,
			var_: "identity",
			want: List([]Value{
				List([]Value{Num(1), Num(0), Num(0)}),
				List([]Value{Num(0), Num(1), Num(0)}),
				List([]Value{Num(0), Num(0), Num(1)}),
			}),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			e := newEnv()
			if _, err := evalStmts(e, prog.Stmts); err != nil {
				t.Fatalf("eval failed: %v", err)
			}
			got, ok := e.get(tc.var_)
			if !ok {
				t.Fatalf("missing variable %q", tc.var_)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("value mismatch for %q:\n got: %#v\nwant: %#v", tc.var_, got, tc.want)
			}
		})
	}
}
