package scad

import (
	"fmt"
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
