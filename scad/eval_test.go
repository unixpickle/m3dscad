package scad

import (
	"fmt"
	"reflect"
	"strings"
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
			name: "CylinderPositionalR1R2",
			src: `
				cylinder(2, 1, 0);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.7, 0, 0.2),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0.4, 0, 1.6),
			},
		},
		{
			name: "CylinderDiameterD1D2",
			src: `
				cylinder(h=4, d1=2, d2=4, center=true);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.9, 0, -1.9),
				model3d.XYZ(1.7, 0, 1.9),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.2, 0, -1.9),
				model3d.XYZ(2.1, 0, 1.9),
			},
		},
		{
			name: "CylinderNamedD",
			src: `
				cylinder(h=3, d=2);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.9, 0, 1.5),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 1.5),
			},
		},
		{
			name: "CapsuleCenterAlongZ",
			src: `
				translate([2,3,4]) capsule(h=4, r=1, center=true);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(2, 3, 6.9),
				model3d.XYZ(2.9, 3, 4),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(2, 3, 7.1),
				model3d.XYZ(3.1, 3, 4),
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
			name: "DefinitionsThenAssignmentsThenCalls",
			src: `
				module b() {
					cylinder(r=a, h=2);
				}
				b();
				d=3;
				a=d;
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(2.9, 0, 1),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(3.1, 0, 1),
				model3d.XYZ(0, 0, 2.1),
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
			name: "RotateExtrudeCrossYAxisUsesBothSides",
			src: `
					rotate_extrude()
						square(size=[2,2], center=true);
				`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.5, 0, 0),
				model3d.XYZ(-0.5, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 0),
				model3d.XYZ(0, 0, 1.1),
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
		{
			name: "IfWithoutElse",
			src: `
				if (true) sphere(r=1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 0),
			},
		},
		{
			name: "IfElse",
			src: `
				if (false) sphere(r=1); else sphere(r=2);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(1.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(2.1, 0, 0),
			},
		},
		{
			name: "IfElseIf",
			src: `
				if (false) sphere(r=1);
				else if (true) sphere(r=2);
				else sphere(r=3);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(1.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(2.1, 0, 0),
				model3d.XYZ(2.9, 0, 0),
			},
		},
		{
			name: "IfBranchScope",
			src: `
				r = 1;
				if (true) r = 2;
				sphere(r=r);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 0),
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

func TestCylinderArgErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name:    "PositionalAfterNamed",
			src:     `cylinder(h=2, 1);`,
			wantErr: "positional args cannot follow named args",
		},
		{
			name:    "MixedUniformAndSpecific",
			src:     `cylinder(h=2, r=1, r2=2);`,
			wantErr: "cannot combine r/d with r1/r2/d1/d2",
		},
		{
			name:    "RAndDConflict",
			src:     `cylinder(h=2, r=1, d=2);`,
			wantErr: "cannot provide both r and d",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			_, err = Eval(prog)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestLinearExtrudeArgErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "MeshScaleUnsupported",
			src: `
				linear_extrude(height=1, scale=2)
					path_mesh("M0,0 L1,0 L1,1 Z");
			`,
			wantErr: "scale != 1 is unsupported for Mesh",
		},
		{
			name: "MeshTwistUnsupported",
			src: `
				linear_extrude(height=1, twist=10)
					path_mesh("M0,0 L1,0 L1,1 Z");
			`,
			wantErr: "twist != 0 is unsupported for Mesh",
		},
		{
			name: "SDFScaleUnsupported",
			src: `
				linear_extrude(height=1, scale=2)
					circle_sdf(r=1);
			`,
			wantErr: "scale != 1 is unsupported for SDF",
		},
		{
			name: "SDFTwistUnsupported",
			src: `
				linear_extrude(height=1, twist=10)
					circle_sdf(r=1);
			`,
			wantErr: "twist != 0 is unsupported for SDF",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			_, err = Eval(prog)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestMissingArgError(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "UnknownNamedArgFunction",
			src: `
				function add_one(x) = x + 1;
				out = add_one(y=2);
			`,
			wantErr: `unknown named argument "y"`,
		},
		{
			name: "UnknownNamedArgModule",
			src: `
				module ball(r) {
					sphere(r=r);
				}
				ball(radius=2);
			`,
			wantErr: `unknown named argument "radius"`,
		},
		{
			name:    "UnknownNamedArgBuiltin",
			src:     `text("hi", foo=3);`,
			wantErr: `text(): unknown argument "foo"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			_, err = Eval(prog)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
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
		{
			name: "IfElseAssignmentScope",
			src: `
				x = 7;
				if (false) x = 10; else x = 20;
			`,
			var_: "x",
			want: Num(7),
		},
		{
			name: "BuiltinFunctions",
			src: `
					out = [
					abs(-3),
					sign(-3), sign(0), sign(2),
					round(sin(90)),
					round(cos(180)),
					round(tan(45)),
					round(asin(1)),
					round(acos(0)),
					round(atan(1)),
					round(atan2(1,0)),
					floor(5.9),
					round(-5.5),
					ceil(5.1),
					round(ln(exp(2))),
					round(log(1000)),
					pow(2,3),
					sqrt(9),
					min([8,3,4,5]),
					max(8,3,4,5),
					norm([3,4]),
					cross([2,1],[0,4]),
					round(lookup(-35, [[-50,20],[-20,18]]) * 10),
					len(rands(0,1,4,123)),
					abs(rands(0,1,3,123)[0]-rands(0,1,3,123)[0]),
					cross([2,3,4],[5,6,7]),
				];
			`,
			var_: "out",
			want: List([]Value{
				Num(3),
				Num(-1), Num(0), Num(1),
				Num(1),
				Num(-1),
				Num(1),
				Num(90),
				Num(90),
				Num(45),
				Num(90),
				Num(5),
				Num(-6),
				Num(6),
				Num(2),
				Num(3),
				Num(8),
				Num(3),
				Num(3),
				Num(8),
				Num(5),
				Num(8),
				Num(190),
				Num(4),
				Num(0),
				List([]Value{Num(-3), Num(6), Num(-3)}),
			}),
		},
		{
			name: "ComparisonSemantics",
			src: `
					out = [
						1 < 2, 1 <= 1, 2 > 1, 2 >= 3, 1 == 1, 1 != 2,
						"ab" > "aa", "aa" > "a", "a" == "a", "a" != "b",
						true > false, true >= 1, false < 1, true < 2, 2 > false,
						true < "a", true < [1],
						[1,2] == [1,2], [1,2] != [2,1], [1] < [2], [1] > [0],
						[1] == 1, [1] != 1, "1" == 1, "1" != 1,
						true == 1, true != 1
					];
				`,
			var_: "out",
			want: List([]Value{
				Bool(true), Bool(true), Bool(true), Bool(false), Bool(true), Bool(true),
				Bool(true), Bool(true), Bool(true), Bool(true),
				Bool(true), Bool(true), Bool(true), Bool(true), Bool(true),
				Bool(false), Bool(false),
				Bool(true), Bool(true), Bool(false), Bool(false),
				Bool(false), Bool(true), Bool(false), Bool(true),
				Bool(false), Bool(true),
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
