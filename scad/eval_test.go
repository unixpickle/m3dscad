package scad

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"

	"github.com/unixpickle/model3d/model2d"
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

func countErrorPositions(err error) int {
	var perr *PosError
	if !errors.As(err, &perr) {
		return 0
	}
	return len(perr.Positions)
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

func assertSDFsEqual3D(t *testing.T, a, b model3d.SDF, min, max model3d.Coord3D, tol float64) {
	t.Helper()
	rng := rand.New(rand.NewSource(1337))
	for i := 0; i < 2000; i++ {
		p := model3d.NewCoord3DRandBounds(min, max, rng)
		av := a.SDF(p)
		bv := b.SDF(p)
		if math.Abs(av-bv) > tol {
			t.Fatalf("SDF mismatch at %v: a=%f b=%f", p, av, bv)
		}
	}
}

func assertSDFsEqual2D(t *testing.T, a, b model2d.SDF, min, max model2d.Coord, tol float64) {
	t.Helper()
	rng := rand.New(rand.NewSource(1337))
	for i := 0; i < 2000; i++ {
		p := model2d.NewCoordRandBounds(min, max, rng)
		av := a.SDF(p)
		bv := b.SDF(p)
		if math.Abs(av-bv) > tol {
			t.Fatalf("SDF mismatch at %v: a=%f b=%f", p, av, bv)
		}
	}
}

func countMesh2DSegments(m *model2d.Mesh) int {
	n := 0
	m.Iterate(func(*model2d.Segment) {
		n++
	})
	return n
}

func countMesh3DTriangles(m *model3d.Mesh) int {
	n := 0
	m.Iterate(func(*model3d.Triangle) {
		n++
	})
	return n
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
			name: "Mirror",
			src: `
				mirror([1,0,0]) translate([1,0,0]) sphere(r=1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(-1.5, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.5, 0, 0),
			},
		},
		{
			name: "Teardrop",
			src: `
				linear_extrude(height=1) teardrop(radius=1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0, -0.9, 0.5),
				model3d.XYZ(0, 1.2, 0.5),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0, 1.5, 0.5),
				model3d.XYZ(1.1, 0, 0.5),
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
			name: "ListComprehensionLetEachTrailComma",
			src: `
				list = [
					for (a = [1:4]) let (
						b = a*a,
						c = 2*b,
					) each [a, b, c] ];
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
			name: "ModuleCallTrailingComma",
			src: `
				cylinder(
					h=2,
					r=1,
				);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 1.9),
				model3d.XYZ(0, 0.9, 1.9),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 2.1),
				model3d.XYZ(0, 1.1, 1.9),
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
		{
			name: "FunctionScope",
			src: `
				function foo(x) = x + y;
				y = 2;

				union() {
					y = 3;
					sphere(r=foo(2));
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(3.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(4.1, 0, 0),
				model3d.XYZ(5.1, 0, 0),
			},
		},
		{
			name: "FunctionArgUsesCallerLocalScope",
			src: `
				function apply(f, v) = f(v);
				module place(k) {
					g = function(x) x + k;
					sphere(r=apply(g, 2));
				}
				place(3);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(4.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(5.1, 0, 0),
			},
		},
		{
			name: "FnSolid3D",
			src: `
				fn_solid([-1,-1,-1], [1,1,1], function(c) norm(c) <= 1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.5, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 0),
				model3d.XYZ(0.8, 0.8, 0.8),
			},
		},
		{
			name: "FnSolid2D",
			src: `
				linear_extrude(height=2)
					fn_solid([-1,-1], [1,1], function(c) c.x*c.x + c.y*c.y <= 1);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0.5, 0, 0.5),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(1.1, 0, 0.5),
				model3d.XYZ(0.8, 0.8, 0.5),
			},
		},
		{
			name: "LineJoinL2",
			src: `
				line_join(points=[[0,0,0], [4,0,0]], r=1, norm="l2");
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(2, 0.8, 0.3),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(2, 1.1, 0),
			},
		},
		{
			name: "LineJoinL1",
			src: `
				line_join(points=[[0,0,0], [4,0,0]], r=1, norm="l1");
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(2, 0.7, 0.2),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(2, 0.8, 0.3),
			},
		},
		{
			name: "ModuleScope",
			src: `
				module foo(x) {
					sphere(r=x + y);
				}
				y = 2;
				union() {
					y = 3;
					foo(2);
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(3.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(4.1, 0, 0),
				model3d.XYZ(5.1, 0, 0),
			},
		},
		{
			name: "ModuleScope2",
			src: `
				module foo(x) {
					function r(x) = x + y;
					sphere(r=r(x));
				}
				y = 2;
				union() {
					y = 3;
					foo(2);
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(3.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(4.1, 0, 0),
				model3d.XYZ(5.1, 0, 0),
			},
		},
		{
			name: "ModuleScope3",
			src: `
				module foo(x) {
					z = y;
					function r(x) = x + z;
					sphere(r=r(x));
				}
				y = 3;
				union() {
					y = 4;
					foo(2);
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(4.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(5.1, 0, 0),
				model3d.XYZ(6.1, 0, 0),
			},
		},
		{
			name: "ModuleScope4",
			src: `
				module foo(x) {
					y = 2;
					function r(x) = x + y;
					union() {
						y = 7;
						sphere(r=r(x));
					}
				}
				y = 4;
				union() {
					y = 3;
					foo(2);
				}
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(3.9, 0, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(4.1, 0, 0),
				model3d.XYZ(5.1, 0, 0),
			},
		},
		{
			name: "ModuleScope5",
			src: `
				y = 2;
				module foo(x) {
					z = y+x;
					sphere(r=z);
					y = 3;
					translate([10, 0, 0]) sphere(r=y+x);
				}
				foo(2);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 3.9),
				model3d.XYZ(10, 0, 4.9),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0, 0, 4.1),
				model3d.XYZ(0, 0, 5.1),
				model3d.XYZ(10, 0, 5.1),
			},
		},
		{
			name: "ModuleArgUsesCallerLocalScope",
			src: `
				module hinge(y) {
					hingeHook(y-2);
					hingeHook(y+2);
				}
				module hingeHook(y) {
					translate([0, y, 0]) sphere(r=1);
				}
				hinge(10);
			`,
			inside: []model3d.Coord3D{
				model3d.XYZ(0, 8, 0),
				model3d.XYZ(0, 12, 0),
			},
			outside: []model3d.Coord3D{
				model3d.XYZ(0, 5, 0),
				model3d.XYZ(0, 15, 0),
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

func TestScaleIgnoresZFor2DShapes(t *testing.T) {
	withZeroZ := mustEvalSolid(t, `
		linear_extrude(height=2)
			scale([1.5, 0.75, 0])
			circle(r=1.2);
	`)
	withNonzeroZ := mustEvalSolid(t, `
		linear_extrude(height=2)
			scale([1.5, 0.75, 99])
			circle(r=1.2);
	`)

	rng := rand.New(rand.NewSource(1337))
	min := model3d.XYZ(-3, -3, -0.5)
	max := model3d.XYZ(3, 3, 2.5)
	for i := 0; i < 2000; i++ {
		p := model3d.NewCoord3DRandBounds(min, max, rng)
		if withZeroZ.Contains(p) != withNonzeroZ.Contains(p) {
			t.Fatalf("z scale changed 2D result at point %v", p)
		}
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

func TestBindArgsStrictErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name:    "DuplicateNamed",
			src:     `sphere(r=1, r=2);`,
			wantErr: `duplicate argument "r"`,
		},
		{
			name:    "PositionalAfterNamed",
			src:     `cube(size=1, true);`,
			wantErr: "positional args cannot follow named args",
		},
		{
			name:    "TooManyPositional",
			src:     `sphere(1, 2);`,
			wantErr: "too many positional args",
		},
		{
			name:    "AliasDuplicate",
			src:     `teardrop(radius=1, r=2);`,
			wantErr: `duplicate argument "r"`,
		},
		{
			name:    "RotateTooManyPositional",
			src:     `rotate(10, [0,0,1], 3) cube(1);`,
			wantErr: "too many positional args",
		},
		{
			name:    "RotateUnknownNamed",
			src:     `rotate(a=10, v=[0,0,1], foo=3) cube(1);`,
			wantErr: `rotate(): unknown argument "foo"`,
		},
		{
			name:    "MirrorZeroAxis",
			src:     `mirror([0,0,0]) cube(1);`,
			wantErr: `mirror(): axis must be non-zero`,
		},
		{
			name:    "UnionTakesNoArgs",
			src:     `union(1) { cube(1); }`,
			wantErr: "too many positional args",
		},
		{
			name:    "DifferenceUnknownNamed",
			src:     `difference(foo=1) { cube(1); cube(0.5); }`,
			wantErr: `difference(): unknown argument "foo"`,
		},
		{
			name:    "IntersectionTakesNoArgs",
			src:     `intersection(1) { cube(1); cube(1); }`,
			wantErr: "too many positional args",
		},
		{
			name:    "SolidTakesNoArgs",
			src:     `solid(1) sphere_sdf(r=1);`,
			wantErr: "too many positional args",
		},
		{
			name:    "MeshToSDFTakesNoArgs",
			src:     `mesh_to_sdf(1) marching_cubes() sphere(r=1);`,
			wantErr: "too many positional args",
		},
		{
			name:    "MetaballTakesNoArgs",
			src:     `metaball(1) sphere_sdf(r=1);`,
			wantErr: "too many positional args",
		},
		{
			name:    "EchoDoesNotTakeChildren",
			src:     `echo(1) a=3;`,
			wantErr: "echo() does not take children",
		},
		{
			name:    "AssertDoesNotTakeChildren",
			src:     `assert(true) a=3;`,
			wantErr: "assert() does not take children",
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
		{
			name:    "UnknownNamedArgTeardropAxis",
			src:     `teardrop(axis=[0,1]);`,
			wantErr: `teardrop(): unknown argument "axis"`,
		},
		{
			name: "MissingRequiredAfterDefault",
			src: `
				function foo(x=2, y) = x+y;
				out = foo(3);
			`,
			wantErr: `missing parameter "y"`,
		},
		{
			name: "FnSolidPreflightUndefinedVar",
			src: `
				fn_solid([-1,-1,-1], [1,1,1], function(c) badvar > 0);
			`,
			wantErr: `undefined variable "badvar"`,
		},
		{
			name: "FnSolidPreflightNonBool",
			src: `
				fn_solid([-1,-1], [1,1], function(c) c.x + c.y);
			`,
			wantErr: `expected bool`,
		},
		{
			name: "FnSolidDimensionMismatch",
			src: `
				fn_solid([-1,-1], [1,1,1], function(c) true);
			`,
			wantErr: `max must have the same dimension as min`,
		},
		{
			name: "LineJoinBadNorm",
			src: `
				line_join(points=[[0,0,0], [1,0,0]], norm="foo");
			`,
			wantErr: `line_join(): norm must be "l2" or "l1"`,
		},
		{
			name: "TransformSolidMissingBounds",
			src: `
				transform(function(c) [c.x, c.y, c.z]) sphere(r=1);
			`,
			wantErr: `missing parameter "max"`,
		},
		{
			name: "TransformMeshRejectsBoundsArgs",
			src: `
				transform([-1,-1], [1,1], function(c) c)
					polygon_mesh(points=[[0,0], [1,0], [0,1]]);
			`,
			wantErr: `too many positional args`,
		},
		{
			name: "TransformSolidMapMustReturnVector",
			src: `
				transform([-1,-1,-1], [1,1,1], function(c) c.x)
					sphere(r=1);
			`,
			wantErr: `expected vector/list`,
		},
		{
			name: "Transform2DSolidBoundsDimMismatch",
			src: `
				transform([-1,-1,-1], [1,1,1], function(c) [c.x, c.y, c.z])
					circle(r=1);
			`,
			wantErr: `transform(): min must be a 2D vector/list`,
		},
		{
			name: "Clip2DRejectsZBounds",
			src: `
				clip(min_z=0) circle(r=1);
			`,
			wantErr: `clip(): min_z/max_z are not supported for 2D shapes`,
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

func TestFunctionCallArgErrorPositions(t *testing.T) {
	t.Run("ExtraPositionalArgUsesCallSite", func(t *testing.T) {
		prog, err := Parse("function foo(x) = [x];\n\ny =\n  foo(2,3);\n")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		_, err = Eval(prog)
		if err == nil {
			t.Fatal("expected error")
		}
		if got, want := err.Error(), `3:1: 4:9: too many positional args`; got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("MissingRequiredArgReferencesDefinition", func(t *testing.T) {
		prog, err := Parse("function foo(x) = [x];\n\ny =\n  foo();\n")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		_, err = Eval(prog)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), `3:1: missing parameter "x"`) {
			t.Fatalf("expected missing-arg error at call site, got %v", err)
		}
		if !strings.Contains(err.Error(), "declared at 1:14") {
			t.Fatalf("missing-arg error should reference declaration site, got %v", err)
		}
	})
}

func TestAssertBuiltin(t *testing.T) {
	t.Run("StatementFormSkipsMessageOnSuccess", func(t *testing.T) {
		prog, err := Parse(`
			assert(condition=true, message=badvar);
			sphere(r=1);
		`)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		if _, err := Eval(prog); err != nil {
			t.Fatalf("eval failed: %v", err)
		}
	})

	t.Run("StatementFormAllowsFollowingStatements", func(t *testing.T) {
		solid := mustEvalSolid(t, `
			assert(true, "still ok");
			sphere(r=1);
		`)
		assertContains(t, solid, model3d.XYZ(0, 0, 0), true)
		assertContains(t, solid, model3d.XYZ(1.1, 0, 0), false)
	})

	t.Run("ExpressionUseIsRejected", func(t *testing.T) {
		prog, err := Parse("out = assert(false, \"boom\");\n")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		_, err = Eval(prog)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), `assert() must be used as a statement`) {
			t.Fatalf("expected statement-only error, got %v", err)
		}
	})

	t.Run("StatementFormFalseIncludesCustomMessage", func(t *testing.T) {
		prog, err := Parse("assert(false, \"boom\");\n")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		_, err = Eval(prog)
		if err == nil {
			t.Fatal("expected error")
		}
		if got := countErrorPositions(err); got != 1 {
			t.Fatalf("expected exactly 1 position(s), got %d: %v", got, err)
		}
		if !strings.Contains(err.Error(), `1:1: assertion failed: boom`) {
			t.Fatalf("expected custom assertion message, got %v", err)
		}
	})

	t.Run("StatementFormFalseUsesDefaultMessage", func(t *testing.T) {
		prog, err := Parse("assert(false);\n")
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		_, err = Eval(prog)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), `1:1: assertion failed`) {
			t.Fatalf("expected default assertion message, got %v", err)
		}
	})
}

func TestScopeCaptureSemantics(t *testing.T) {
	t.Run("FunctionSeesLaterSameScopeAssignment", func(t *testing.T) {
		prog, err := Parse(`
			function f() = a;
			a = 7;
			out = f();
		`)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		e := newEnv(nil)
		if _, err := evalStmts(e, prog.Stmts); err != nil {
			t.Fatalf("eval failed: %v", err)
		}
		got, ok := e.get("out")
		if !ok {
			t.Fatal(`missing variable "out"`)
		}
		if !got.Equal(Num(7)) {
			t.Fatalf("expected out=7, got %#v", got)
		}
	})

	t.Run("FunctionCannotSeeCallSiteChildScope", func(t *testing.T) {
		prog, err := Parse(`
			function f() = a;
			if (true) { a = 7; }
			out = f();
		`)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		_, err = Eval(prog)
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), `undefined variable "a"`) {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("ModuleSeesLaterSameScopeAssignment", func(t *testing.T) {
		prog, err := Parse(`
			module ball() { sphere(r=a); }
			a = 2;
			ball();
		`)
		if err != nil {
			t.Fatalf("parse failed: %v", err)
		}
		_, err = Eval(prog)
		if err != nil {
			t.Fatalf("eval failed: %v", err)
		}
	})
}

func TestRedeclareErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "VariableRedeclare",
			src: `
				a = 1;
				a = 2;
			`,
			wantErr: `cannot redeclare variable "a"`,
		},
		{
			name: "FunctionRedeclare",
			src: `
				function f() = 1;
				function f() = 2;
				out = f();
			`,
			wantErr: `cannot redeclare function "f"`,
		},
		{
			name: "ModuleRedeclare",
			src: `
				module m() { sphere(r=1); }
				module m() { sphere(r=2); }
				m();
			`,
			wantErr: `cannot redeclare module "m"`,
		},
		{
			name: "FunctionThenVariableAllowed",
			src: `
				function f(x) = x;
				f = function(x) x + 1;
				out = f(2);
				sphere(r=out);
			`,
			wantErr: ``,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			_, err = Eval(prog)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestIncompleteLetErrors(t *testing.T) {
	_, err := Parse(`
	x = let (
		y = 3,
		z =
	)
	`)
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = Parse(`
	x = let (
		y = 3,
		z =,
	)
	`)
	if err == nil {
		t.Fatal("expected error")
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
			name: "LetInListComprehension",
			src: `
				input = [2, 3, 5, 7, 11];
				out = [for (a = input) let (
					x=a*a,
					y=x+1
				) y-1];
			`,
			var_: "out",
			want: List([]Value{Num(4), Num(9), Num(25), Num(49), Num(121)}),
		},
		{
			name: "LetInListComprehensionTrailingComma",
			src: `
				input = [2, 3, 5, 7, 11];
				out = [for (a = input) let (
					x=a*a,
					y=x+1,
				) y-1];
			`,
			var_: "out",
			want: List([]Value{Num(4), Num(9), Num(25), Num(49), Num(121)}),
		},
		{
			name: "LetFunctionCallTrailingComma",
			src: `
				function foo(x, y) = x+y;
				out = foo(
					3,
					4,
				);
			`,
			var_: "out",
			want: Num(7),
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
			name: "BuiltinConstantPI",
			src: `
				out = [
					round(PI * 1000),
					round(sin((PI / 2) * 180 / PI)),
					round(cos(PI * 180 / PI)),
				];
			`,
			var_: "out",
			want: List([]Value{Num(3142), Num(1), Num(-1)}),
		},
		{
			name: "VectorXYZAccessors",
			src: `
				v = [11, 22, 33, 44];
				out = [v.x, v.y, v.z, [0:2:10].z];
			`,
			var_: "out",
			want: List([]Value{Num(11), Num(22), Num(33), Num(4)}),
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
			e := newEnv(nil)
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

func TestLookupFunction(t *testing.T) {
	src := `
		table1 = [[-1, 2], [4, 4], [5, 4], [3.1, -1]];
		table2 = [[3.1, -1], [5, 4], [-1, 2], [4, 4]];
		out = [
			lookup(3.2, table1),
			lookup(3.2, table2),
			lookup(3, table1),
			lookup(3, table2),
			lookup(-10, table1),
			lookup(-10, table2),
			lookup(10, table1),
			lookup(10, table2),
		];
	`
	prog, err := Parse(src)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	e := newEnv(nil)
	if _, err := evalStmts(e, prog.Stmts); err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	got, ok := e.get("out")
	if !ok {
		t.Fatal("missing variable \"out\"")
	}
	if got.Kind != ValList {
		t.Fatalf("expected list, got %#v", got)
	}
	want := []float64{
		-0.4444444444444444,
		-0.4444444444444444,
		-0.9268292682926833,
		-0.9268292682926833,
		2,
		2,
		4,
		4,
	}
	if len(got.List) != len(want) {
		t.Fatalf("unexpected result length: got %d want %d", len(got.List), len(want))
	}
	for i, wantNum := range want {
		gotNum, err := got.List[i].AsNum()
		if err != nil {
			t.Fatalf("result %d is not numeric: %v", i, err)
		}
		if math.Abs(gotNum-wantNum) > 1e-9 {
			t.Fatalf("result %d mismatch: got %.12f want %.12f", i, gotNum, wantNum)
		}
	}
}

func TestVectorAccessorErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name:    "UnknownAccessor",
			src:     `v=[1,2,3]; out=v.w;`,
			wantErr: `unknown vector accessor "w"`,
		},
		{
			name:    "AccessorAfterDotNeedsIdentifier",
			src:     `v=[1,2,3]; out=v.;`,
			wantErr: "expected identifier after '.'",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if strings.Contains(tc.wantErr, "expected identifier") {
				if err == nil {
					t.Fatal("expected parse error")
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			_, err = Eval(prog)
			if err == nil {
				t.Fatal("expected eval error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestEvalErrorPositionPropagation(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantCount int
	}{
		{
			name: "TopLevelAssignmentBuiltinArgTypeError",
			src: `
				a = sin([1]);
				sphere(r=1);
			`,
			wantCount: 1,
		},
		{
			name: "TopLevelAssignmentIndexTypeError",
			src: `
				a = [1][true];
				sphere(r=1);
			`,
			wantCount: 1,
		},
		{
			name: "FunctionCallAddsCallerPosition",
			src: `
				function f(x)=badvar;
				a = f(1);
				sphere(r=1);
			`,
			wantCount: 2,
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
				t.Fatal("expected eval error")
			}
			if got := countErrorPositions(err); got != tc.wantCount {
				t.Fatalf("expected exactly %d position(s), got %d: %v", tc.wantCount, got, err)
			}
		})
	}
}

func TestEchoStatementAndFunction(t *testing.T) {
	prog, err := Parse(`
		echo(1, "hi", v=[3,4]);
		out = echo([1,2,3].z);
	`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	var msgs []string
	e := newEnv(func(msg string) {
		msgs = append(msgs, msg)
	})
	if _, err := evalStmts(e, prog.Stmts); err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 echo messages, got %d", len(msgs))
	}
	if got, want := msgs[0], "3"; got != want {
		t.Fatalf("unexpected function echo: got %q want %q", got, want)
	}
	if got, want := msgs[1], `1, "hi", v = [3, 4]`; got != want {
		t.Fatalf("unexpected statement echo: got %q want %q", got, want)
	}
	out, ok := e.get("out")
	if !ok {
		t.Fatal(`missing variable "out"`)
	}
	if out.Kind != ValNull {
		t.Fatalf("expected out to be undef/null, got kind=%v", out.Kind)
	}
}

func TestStrFunction(t *testing.T) {
	prog, err := Parse(`
		number = 2;
		echo(str("This is ", number, 3, " and that's it."));
	`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	var msgs []string
	e := newEnv(func(msg string) {
		msgs = append(msgs, msg)
	})
	if _, err := evalStmts(e, prog.Stmts); err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	want := []string{`"This is 23 and that's it."`}
	if !reflect.DeepEqual(msgs, want) {
		t.Fatalf("echo mismatch:\n got: %#v\nwant: %#v", msgs, want)
	}
}

func TestTypePredicateFunctions(t *testing.T) {
	prog, err := Parse(`
		assert(is_list([]));
		assert(is_list([1]));
		assert(is_list([1,2]));
		assert(is_list([true]));
		assert(is_list([1,2,[5,6],"test"]));
		assert(!is_list(1));
		assert(!is_list(1/0));
		assert(!is_list(((1/0)/(1/0))));
		assert(!is_list("test"));
		assert(!is_list(true));
		assert(!is_list(false));

		assert(is_num(0.1));
		assert(is_num(1));
		assert(is_num(10));
		assert(is_num(+1/0));
		assert(is_num(-1/0));
		assert(!is_num(0/0));
		assert(!is_num((1/0)/(1/0)));
		assert(!is_num([]));
		assert(!is_num([1]));
		assert(!is_num("test"));
		assert(!is_num(false));

		assert(is_bool(true));
		assert(is_bool(false));
		assert(!is_bool([]));
		assert(!is_bool([1]));
		assert(!is_bool("test"));
		assert(!is_bool(0.1));
		assert(!is_bool(1));
		assert(!is_bool(10));
		assert(!is_bool(0/0));
		assert(!is_bool((1/0)/(1/0)));
		assert(!is_bool(1/0));
		assert(!is_bool(-1/0));

		assert(is_string(""));
		assert(is_string("test"));
		assert(!is_string(0.1));
		assert(!is_string(1));
		assert(!is_string(10));
		assert(!is_string([]));
		assert(!is_string([1]));
		assert(!is_string(false));
		assert(!is_string(0/0));
		assert(!is_string((1/0)/(1/0)));
		assert(!is_string(1/0));
		assert(!is_string(-1/0));

		assert(is_function(function(x) x*x));
		func = function(x) x+x;
		assert(is_function(func));
	`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if _, err := evalStmts(newEnv(nil), prog.Stmts); err != nil {
		t.Fatalf("eval failed: %v", err)
	}
}

func TestTypePredicateFunctionArityErrors(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "MissingArg",
			src: `
				a = is_list();
				sphere(r=1);
			`,
			want: "function is_list needs exactly 1 argument",
		},
		{
			name: "ExtraArg",
			src: `
				a = is_function(1, 2);
				sphere(r=1);
			`,
			want: "function is_function needs exactly 1 argument",
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
				t.Fatal("expected eval error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestModuleDefaultArgUsesCapturedScope(t *testing.T) {
	prog, err := Parse(`
		module foo(y=baz(z)) {
			echo(y);
			sphere(r=0.01);
		}

		z = 3;
		function baz(x) = x * 2;
		union() {
			z = 2;
			function baz(x) = x / 2;
			foo();
		}
	`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	var msgs []string
	e := newEnv(func(msg string) {
		msgs = append(msgs, msg)
	})
	if _, err := evalStmts(e, prog.Stmts); err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	want := []string{"6"}
	if !reflect.DeepEqual(msgs, want) {
		t.Fatalf("echo mismatch:\n got: %#v\nwant: %#v", msgs, want)
	}
}

func TestAnonymousFunctions(t *testing.T) {
	prog, err := Parse(`
		x = function(y) 3+y;
		echo(x(2));

		a = 1;
		selector = function (which)
			which == "add"
			? function (x) x + x + a
			: function (x) x * x + a;

		echo(selector("add"));
		echo(selector("add")(5));
		echo(selector("mul"));
		echo(selector("mul")(5));
	`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	var msgs []string
	e := newEnv(func(msg string) {
		msgs = append(msgs, msg)
	})
	if _, err := evalStmts(e, prog.Stmts); err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	want := []string{
		"5",
		`function(x) ((x + x) + a)`,
		"11",
		`function(x) ((x * x) + a)`,
		"26",
	}
	if !reflect.DeepEqual(msgs, want) {
		t.Fatalf("echo mismatch:\n got: %#v\nwant: %#v", msgs, want)
	}
}

func TestFunctionValueShadowsNamedFunctionInCalls(t *testing.T) {
	prog, err := Parse(`
		x = function (y) 3+y;
		function x(y) = y;
		function d(y) = y;

		function z(f) = f(2)+1;
		echo(z(x));
		echo(x(2));
		echo(d(2));

		function manip(f) = function(x) 1+f(x);
		echo((manip(x))(4));
	`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	var msgs []string
	e := newEnv(func(msg string) {
		msgs = append(msgs, msg)
	})
	if _, err := evalStmts(e, prog.Stmts); err != nil {
		t.Fatalf("eval failed: %v", err)
	}
	want := []string{"6", "5", "2", "8"}
	if !reflect.DeepEqual(msgs, want) {
		t.Fatalf("echo mismatch:\n got: %#v\nwant: %#v", msgs, want)
	}
}

func TestPassingNamedFunctionAsValueErrors(t *testing.T) {
	prog, err := Parse(`
		function d(y) = y;
		function manip(f) = function(x) 1+f(x);
		echo((manip(d))(4));
	`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, err = Eval(prog)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), `undefined variable "d"`) {
		t.Fatalf("expected undefined variable error, got %v", err)
	}
}

func TestRotateExtrudeSDFTorusAgreement(t *testing.T) {
	shape := mustEvalShape(t, `
		rotate_extrude()
			translate([3, 0, 0])
			circle_sdf(r=1);
	`)
	if shape.Kind != ShapeSDF3D {
		t.Fatalf("expected ShapeSDF3D, got %v", shape.Kind)
	}
	got := shape.SDF3
	want := &model3d.Torus{
		Center:      model3d.Coord3D{},
		Axis:        model3d.Z(1),
		OuterRadius: 3,
		InnerRadius: 1,
	}
	min := want.Min().AddScalar(-0.5)
	max := want.Max().AddScalar(0.5)
	rng := rand.New(rand.NewSource(1337))
	maxErr := 0.0
	for i := 0; i < 10000; i++ {
		c := model3d.NewCoord3DRandBounds(min, max, rng)
		errAbs := math.Abs(got.SDF(c) - want.SDF(c))
		if errAbs > maxErr {
			maxErr = errAbs
		}
		if errAbs > 1e-7 {
			t.Fatalf("sdf mismatch at %v: got=%f want=%f err=%e", c, got.SDF(c), want.SDF(c), errAbs)
		}
	}
}

func TestRootLevelUnion(t *testing.T) {
	code := `
	dual_contour(0.05) sphere(r=1);
	translate([3, 0, 0]) dual_contour(0.05) sphere(r=1);
	`
	shape := mustEvalShape(t, code)
	if shape.Kind != ShapeMesh3D {
		t.Fatalf("expected ShapeMesh3D, got %v", shape.Kind)
	}
	got := shape.M3
	if math.Abs(got.Volume()-8.0/3.0*math.Pi) > 0.01 {
		t.Fatalf("unexpected volume: %f", got.Volume())
	}
}

func TestUnionMeshesConcatenates(t *testing.T) {
	t.Run("2D", func(t *testing.T) {
		a := mustEvalShape(t, `polygon_mesh(points=[[0,0],[1,0],[1,1],[0,1]]);`)
		b := mustEvalShape(t, `translate([3,0,0]) polygon_mesh(points=[[0,0],[1,0],[1,1],[0,1]]);`)
		u := mustEvalShape(t, `
			union() {
				polygon_mesh(points=[[0,0],[1,0],[1,1],[0,1]]);
				translate([3,0,0]) polygon_mesh(points=[[0,0],[1,0],[1,1],[0,1]]);
			}
		`)
		if a.Kind != ShapeMesh2D || b.Kind != ShapeMesh2D || u.Kind != ShapeMesh2D {
			t.Fatalf("expected ShapeMesh2D outputs, got %v %v %v", a.Kind, b.Kind, u.Kind)
		}
		want := countMesh2DSegments(a.M2) + countMesh2DSegments(b.M2)
		got := countMesh2DSegments(u.M2)
		if got != want {
			t.Fatalf("segment count mismatch: got %d want %d", got, want)
		}
	})

	t.Run("3D", func(t *testing.T) {
		a := mustEvalShape(t, `linear_extrude(height=1) polygon_mesh(points=[[0,0],[1,0],[1,1],[0,1]]);`)
		b := mustEvalShape(t, `translate([3,0,0]) linear_extrude(height=1) polygon_mesh(points=[[0,0],[1,0],[1,1],[0,1]]);`)
		u := mustEvalShape(t, `
			union() {
				linear_extrude(height=1) polygon_mesh(points=[[0,0],[1,0],[1,1],[0,1]]);
				translate([3,0,0]) linear_extrude(height=1) polygon_mesh(points=[[0,0],[1,0],[1,1],[0,1]]);
			}
		`)
		if a.Kind != ShapeMesh3D || b.Kind != ShapeMesh3D || u.Kind != ShapeMesh3D {
			t.Fatalf("expected ShapeMesh3D outputs, got %v %v %v", a.Kind, b.Kind, u.Kind)
		}
		want := countMesh3DTriangles(a.M3) + countMesh3DTriangles(b.M3)
		got := countMesh3DTriangles(u.M3)
		if got != want {
			t.Fatalf("triangle count mismatch: got %d want %d", got, want)
		}
	})
}

func TestTransformModifier(t *testing.T) {
	t.Run("MirrorSolid3D", func(t *testing.T) {
		solid := mustEvalSolid(t, `
			mirror([1,0,0]) translate([1,0,0]) sphere(r=1);
		`)
		assertContains(t, solid, model3d.XYZ(-1.5, 0, 0), true)
		assertContains(t, solid, model3d.XYZ(1.5, 0, 0), false)
	})

	t.Run("MirrorSolid2D", func(t *testing.T) {
		shape := mustEvalShape(t, `
			mirror([1,0]) translate([1,0,0]) circle(r=1);
		`)
		if shape.Kind != ShapeSolid2D {
			t.Fatalf("expected ShapeSolid2D, got %v", shape.Kind)
		}
		if !shape.S2.Contains(model2d.XY(-1.5, 0)) {
			t.Fatalf("expected mirrored 2D solid to contain point")
		}
		if shape.S2.Contains(model2d.XY(1.5, 0)) {
			t.Fatalf("expected mirrored 2D solid to exclude point")
		}
	})

	t.Run("MirrorSDF3D", func(t *testing.T) {
		shape := mustEvalShape(t, `
			mirror([0,1,0]) translate([0,1,0]) sphere_sdf(r=1);
		`)
		if shape.Kind != ShapeSDF3D {
			t.Fatalf("expected ShapeSDF3D, got %v", shape.Kind)
		}
		want := model3d.TransformSDF(
			model3d.Mirror(model3d.Y(1)),
			&model3d.Sphere{Center: model3d.Y(1), Radius: 1},
		)
		assertSDFsEqual3D(
			t,
			shape.SDF3,
			want,
			model3d.XYZ(-2, -2, -2),
			model3d.XYZ(2, 2, 2),
			1e-8,
		)
	})

	t.Run("Solid3D", func(t *testing.T) {
		solid := mustEvalSolid(t, `
			transform([-2,-2,-2], [2,2,2], function(c) [c.x/2, c.y/2, c.z/2])
				sphere(r=1);
		`)
		assertContains(t, solid, model3d.XYZ(1.5, 0, 0), true)
		assertContains(t, solid, model3d.XYZ(2.1, 0, 0), false)
	})

	t.Run("Solid2D", func(t *testing.T) {
		shape := mustEvalShape(t, `
			transform([-2,-2], [2,2], function(c) [c.x/2, c.y/2])
				circle(r=1);
		`)
		if shape.Kind != ShapeSolid2D {
			t.Fatalf("expected ShapeSolid2D, got %v", shape.Kind)
		}
		if !shape.S2.Contains(model2d.XY(1.5, 0)) {
			t.Fatalf("expected transformed 2D solid to contain point")
		}
		if shape.S2.Contains(model2d.XY(2.1, 0)) {
			t.Fatalf("expected transformed 2D solid to exclude point")
		}
	})

	t.Run("SDF3D", func(t *testing.T) {
		shape := mustEvalShape(t, `
			transform([-2,-2,-2], [2,2,2], function(c) [c.x/2, c.y/2, c.z/2])
				sphere_sdf(r=1);
		`)
		if shape.Kind != ShapeSDF3D {
			t.Fatalf("expected ShapeSDF3D, got %v", shape.Kind)
		}
		if d := shape.SDF3.SDF(model3d.XYZ(1.5, 0, 0)); d <= 0 {
			t.Fatalf("expected inside SDF value > 0, got %f", d)
		}
		if d := shape.SDF3.SDF(model3d.XYZ(2.1, 0, 0)); d >= 0 {
			t.Fatalf("expected outside SDF value < 0, got %f", d)
		}
	})

	t.Run("SDF2D", func(t *testing.T) {
		shape := mustEvalShape(t, `
			transform([-2,-2], [2,2], function(c) [c.x/2, c.y/2])
				circle_sdf(r=1);
		`)
		if shape.Kind != ShapeSDF2D {
			t.Fatalf("expected ShapeSDF2D, got %v", shape.Kind)
		}
		if d := shape.SDF2.SDF(model2d.XY(1.5, 0)); d <= 0 {
			t.Fatalf("expected inside SDF value > 0, got %f", d)
		}
		if d := shape.SDF2.SDF(model2d.XY(2.1, 0)); d >= 0 {
			t.Fatalf("expected outside SDF value < 0, got %f", d)
		}
	})

	t.Run("Mesh2D", func(t *testing.T) {
		shape := mustEvalShape(t, `
			transform(function(c) [c.x+3, c.y-1])
				polygon_mesh(points=[[0,0], [1,0], [0,1]]);
		`)
		if shape.Kind != ShapeMesh2D {
			t.Fatalf("expected ShapeMesh2D, got %v", shape.Kind)
		}
		min := shape.M2.Min()
		max := shape.M2.Max()
		if min != model2d.XY(3, -1) || max != model2d.XY(4, 0) {
			t.Fatalf("unexpected mesh bounds: min=%v max=%v", min, max)
		}
	})

	t.Run("Mesh3D", func(t *testing.T) {
		shape := mustEvalShape(t, `
			transform(function(c) [c.x+2, c.y-1, c.z+4])
				linear_extrude(height=1)
				polygon_mesh(points=[[0,0], [1,0], [0,1]]);
		`)
		if shape.Kind != ShapeMesh3D {
			t.Fatalf("expected ShapeMesh3D, got %v", shape.Kind)
		}
		min := shape.M3.Min()
		max := shape.M3.Max()
		if min != model3d.XYZ(2, -1, 4) || max != model3d.XYZ(3, 0, 5) {
			t.Fatalf("unexpected mesh bounds: min=%v max=%v", min, max)
		}
	})
}

func TestClipModifier(t *testing.T) {
	t.Run("Solid3DMatchesIntersection", func(t *testing.T) {
		clipped := mustEvalSolid(t, `
			clip(min_x=-0.4, max_x=0.8, min_y=-0.7, max_y=0.2, min_z=-0.3, max_z=0.9)
				sphere(r=1.2);
		`)
		intersected := mustEvalSolid(t, `
			intersection() {
				sphere(r=1.2);
				translate([-0.4, -0.7, -0.3])
					cube(size=[1.2, 0.9, 1.2], center=false);
			}
		`)
		assertSolids3DEqual(t, clipped, intersected)
	})

	t.Run("Solid3DPartiallyUnboundedMatchesIntersection", func(t *testing.T) {
		clipped := mustEvalSolid(t, `
			clip(min_x=-0.4, max_y=0.2)
				sphere(r=1.2);
		`)
		intersected := mustEvalSolid(t, `
			intersection() {
				sphere(r=1.2);
				translate([-0.4, -10000, -10000])
					cube(size=[10000.4, 10000.2, 20000], center=false);
			}
		`)
		assertSolids3DEqual(t, clipped, intersected)
	})

	t.Run("Solid2DMatchesIntersection", func(t *testing.T) {
		clipped := mustEvalShape(t, `
			clip(min_x=-0.4, max_x=0.8, min_y=-0.7, max_y=0.2)
				circle(r=1.2);
		`)
		intersected := mustEvalShape(t, `
			intersection() {
				circle(r=1.2);
				translate([-0.4, -0.7, 0])
					square(size=[1.2, 0.9], center=false);
			}
		`)
		if clipped.Kind != ShapeSolid2D || intersected.Kind != ShapeSolid2D {
			t.Fatalf("expected 2D solids, got %v and %v", clipped.Kind, intersected.Kind)
		}
		assertSolids2DEqual(t, clipped.S2, intersected.S2)
	})

	t.Run("Solid2DPartiallyUnboundedMatchesIntersection", func(t *testing.T) {
		clipped := mustEvalShape(t, `
			clip(max_x=0.8, min_y=-0.7)
				circle(r=1.2);
		`)
		intersected := mustEvalShape(t, `
			intersection() {
				circle(r=1.2);
				translate([-10000, -0.7, 0])
					square(size=[10000.8, 10000.7], center=false);
			}
		`)
		if clipped.Kind != ShapeSolid2D || intersected.Kind != ShapeSolid2D {
			t.Fatalf("expected 2D solids, got %v and %v", clipped.Kind, intersected.Kind)
		}
		assertSolids2DEqual(t, clipped.S2, intersected.S2)
	})

	t.Run("SDF3DMatchesIntersection", func(t *testing.T) {
		clipped := mustEvalShape(t, `
			clip(min_x=-0.4, max_x=0.8, min_y=-0.7, max_y=0.2, min_z=-0.3, max_z=0.9)
				sphere_sdf(r=1.2);
		`)
		intersected := mustEvalShape(t, `
			intersection() {
				sphere_sdf(r=1.2);
				translate([-0.4, -0.7, -0.3])
					cube_sdf(size=[1.2, 0.9, 1.2], center=false);
			}
		`)
		if clipped.Kind != ShapeSDF3D || intersected.Kind != ShapeSDF3D {
			t.Fatalf("expected 3D SDFs, got %v and %v", clipped.Kind, intersected.Kind)
		}
		assertSDFsEqual3D(t, clipped.SDF3, intersected.SDF3, model3d.XYZ(-1.5, -1.5, -1.5), model3d.XYZ(1.5, 1.5, 1.5), 1e-8)
	})

	t.Run("SDF3DPartiallyUnboundedMatchesIntersection", func(t *testing.T) {
		clipped := mustEvalShape(t, `
			clip(max_x=0.8, min_z=-0.3)
				sphere_sdf(r=1.2);
		`)
		intersected := mustEvalShape(t, `
			intersection() {
				sphere_sdf(r=1.2);
				translate([-10000, -10000, -0.3])
					cube_sdf(size=[10000.8, 20000, 10000.3], center=false);
			}
		`)
		if clipped.Kind != ShapeSDF3D || intersected.Kind != ShapeSDF3D {
			t.Fatalf("expected 3D SDFs, got %v and %v", clipped.Kind, intersected.Kind)
		}
		assertSDFsEqual3D(t, clipped.SDF3, intersected.SDF3, model3d.XYZ(-1.1, -1.1, -0.2), model3d.XYZ(0.7, 1.1, 1.1), 1e-8)
	})

	t.Run("SDF2DMatchesIntersection", func(t *testing.T) {
		clipped := mustEvalShape(t, `
			clip(min_x=-0.4, max_x=0.8, min_y=-0.7, max_y=0.2)
				circle_sdf(r=1.2);
		`)
		intersected := mustEvalShape(t, `
			intersection() {
				circle_sdf(r=1.2);
				translate([-0.4, -0.7, 0])
					square_sdf(size=[1.2, 0.9], center=false);
			}
		`)
		if clipped.Kind != ShapeSDF2D || intersected.Kind != ShapeSDF2D {
			t.Fatalf("expected 2D SDFs, got %v and %v", clipped.Kind, intersected.Kind)
		}
		assertSDFsEqual2D(t, clipped.SDF2, intersected.SDF2, model2d.XY(-1.5, -1.5), model2d.XY(1.5, 1.5), 1e-8)
	})

	t.Run("SDF2DPartiallyUnboundedMatchesIntersection", func(t *testing.T) {
		clipped := mustEvalShape(t, `
			clip(min_x=-0.4, max_y=0.2)
				circle_sdf(r=1.2);
		`)
		intersected := mustEvalShape(t, `
			intersection() {
				circle_sdf(r=1.2);
				translate([-0.4, -10000, 0])
					square_sdf(size=[10000.4, 10000.2], center=false);
			}
		`)
		if clipped.Kind != ShapeSDF2D || intersected.Kind != ShapeSDF2D {
			t.Fatalf("expected 2D SDFs, got %v and %v", clipped.Kind, intersected.Kind)
		}
		assertSDFsEqual2D(t, clipped.SDF2, intersected.SDF2, model2d.XY(-0.3, -1.1), model2d.XY(1.1, 0.1), 1e-8)
	})

	t.Run("SDFRectMatchesAlreadyClippedRect", func(t *testing.T) {
		cases := []struct {
			name      string
			clipArgs  string
			wantMin   model3d.Coord3D
			wantMax   model3d.Coord3D
			sampleMin model3d.Coord3D
			sampleMax model3d.Coord3D
		}{
			{
				name:      "NoClip",
				clipArgs:  "",
				wantMin:   model3d.XYZ(0, 0, 0),
				wantMax:   model3d.XYZ(4, 3, 2),
				sampleMin: model3d.XYZ(-1, -1, -1),
				sampleMax: model3d.XYZ(5, 4, 3),
			},
			{
				name:      "LowerAndUpper",
				clipArgs:  "min_x=1.5, max_y=2.1, max_z=1.1",
				wantMin:   model3d.XYZ(1.5, 0, 0),
				wantMax:   model3d.XYZ(4, 2.1, 1.1),
				sampleMin: model3d.XYZ(-1, -1, -1),
				sampleMax: model3d.XYZ(5, 4, 3),
			},
			{
				name:      "BothSidesAllAxes",
				clipArgs:  "min_x=0.4, max_x=1.6, min_y=1, max_y=2.5, min_z=0.2, max_z=1.8",
				wantMin:   model3d.XYZ(0.4, 1, 0.2),
				wantMax:   model3d.XYZ(1.6, 2.5, 1.8),
				sampleMin: model3d.XYZ(-1, -1, -1),
				sampleMax: model3d.XYZ(5, 4, 3),
			},
			{
				name:      "OutsideLowerBoundsClampsToOriginal",
				clipArgs:  "min_x=-2, min_y=-5, min_z=-3, max_x=2.5",
				wantMin:   model3d.XYZ(0, 0, 0),
				wantMax:   model3d.XYZ(2.5, 3, 2),
				sampleMin: model3d.XYZ(-1, -1, -1),
				sampleMax: model3d.XYZ(5, 4, 3),
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				clipCall := "clip()"
				if tc.clipArgs != "" {
					clipCall = "clip(" + tc.clipArgs + ")"
				}
				clipped := mustEvalShape(t, clipCall+` cube_sdf(size=[4, 3, 2], center=false);`)
				if clipped.Kind != ShapeSDF3D {
					t.Fatalf("expected ShapeSDF3D, got %v", clipped.Kind)
				}

				size := tc.wantMax.Sub(tc.wantMin)
				wantSrc := fmt.Sprintf(`
					translate([%g, %g, %g])
						cube_sdf(size=[%g, %g, %g], center=false);
				`, tc.wantMin.X, tc.wantMin.Y, tc.wantMin.Z, size.X, size.Y, size.Z)
				want := mustEvalShape(t, wantSrc)
				if want.Kind != ShapeSDF3D {
					t.Fatalf("expected ShapeSDF3D, got %v", want.Kind)
				}

				assertSDFsEqual3D(t, clipped.SDF3, want.SDF3, tc.sampleMin, tc.sampleMax, 1e-8)
			})
		}
	})
}
