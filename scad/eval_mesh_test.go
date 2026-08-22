package scad

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func TestSmoothSq(t *testing.T) {
	source := `polygon_mesh(points=[[0,0], [4,0], [5,2], [3,5], [0,3]]);`
	input := mustEvalShape(t, source)
	result := mustEvalShape(t, `smooth_sq(3) `+source)
	if input.Kind != ShapeMesh2D || result.Kind != ShapeMesh2D {
		t.Fatalf("expected ShapeMesh2D, got %v and %v", input.Kind, result.Kind)
	}
	assertMeshes2DEqual(t, result.M2, input.M2.SmoothSq(3))
}

func TestImageSmoothSqPipeline(t *testing.T) {
	files := fstest.MapFS{
		"pixel.png": {Data: testBitmapPNG(t)},
	}
	prog, err := ParseFile("main.scad", `smooth_sq(5) image("pixel.png");`)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	if shape.Kind != ShapeMesh2D {
		t.Fatalf("expected ShapeMesh2D, got %v", shape.Kind)
	}
}

func TestSmoothSqErrors(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr string
	}{
		{
			name:    "solid input",
			source:  `smooth_sq(3) square(1);`,
			wantErr: "smooth_sq(): requires a 2D mesh",
		},
		{
			name:    "3D mesh input",
			source:  `smooth_sq(3) dual_contour(delta=0.5) sphere(1);`,
			wantErr: "smooth_sq(): requires a 2D mesh",
		},
		{
			name:    "missing iterations",
			source:  `smooth_sq() polygon_mesh(points=[[0,0], [1,0], [0,1]]);`,
			wantErr: `missing parameter "iters"`,
		},
		{
			name:    "fractional iterations",
			source:  `smooth_sq(1.5) polygon_mesh(points=[[0,0], [1,0], [0,1]]);`,
			wantErr: "smooth_sq(): iters must be an integer",
		},
		{
			name:    "negative iterations",
			source:  `smooth_sq(-1) polygon_mesh(points=[[0,0], [1,0], [0,1]]);`,
			wantErr: "smooth_sq(): iters must be >= 0",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog, err := Parse(test.source)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Eval(prog, Hooks{})
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func TestDecimate(t *testing.T) {
	tests := []struct {
		name      string
		arguments string
		target    func(int) int
	}{
		{
			name:      "positional fraction",
			arguments: "0.25",
			target:    func(inputCount int) int { return int(float64(inputCount) * 0.25) },
		},
		{
			name:      "named fraction and options",
			arguments: "triangles_fraction=0.3, min_det=1e-7, min_normal_dot_product=-0.5, tikhonov_regularization=1e-4",
			target:    func(inputCount int) int { return int(float64(inputCount) * 0.3) },
		},
		{
			name:      "named triangle count",
			arguments: "triangle_count=80",
			target:    func(int) int { return 80 },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := model3d.NewMeshPolar(nil, 12)
			inputTriangles := input.NumTriangles()
			target := test.target(inputTriangles)
			prog, err := Parse("decimate(" + test.arguments + ") dual_contour() sphere();")
			if err != nil {
				t.Fatal(err)
			}
			shape, err := Eval(prog, Hooks{
				DualContour: func(ShapeRep, float64, bool, bool) (*model3d.Mesh, error) {
					return input, nil
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			if shape.Kind != ShapeMesh3D {
				t.Fatalf("expected ShapeMesh3D, got %v", shape.Kind)
			}
			if got := shape.M3.NumTriangles(); got == 0 || got > target {
				t.Fatalf("triangle count: got %d, want 1..%d", got, target)
			}
			if shape.M3.NeedsRepair() || len(shape.M3.SingularVertices()) != 0 {
				t.Fatal("decimated mesh is not manifold")
			}
			if got := input.NumTriangles(); got != inputTriangles {
				t.Fatalf("input triangle count changed: got %d want %d", got, inputTriangles)
			}
		})
	}
}

func TestDecimateErrors(t *testing.T) {
	validMesh := model3d.NewMeshPolar(nil, 6)
	openMesh := model3d.NewMeshTriangles([]*model3d.Triangle{{
		model3d.XYZ(0, 0, 0),
		model3d.XYZ(1, 0, 0),
		model3d.XYZ(0, 1, 0),
	}})
	tests := []struct {
		name    string
		source  string
		mesh    *model3d.Mesh
		wantErr string
	}{
		{name: "solid input", source: `decimate(10) sphere();`, wantErr: "decimate(): requires a 3D mesh"},
		{name: "2D mesh input", source: `decimate(10) polygon_mesh(points=[[0,0],[1,0],[0,1]]);`, wantErr: "decimate(): requires a 3D mesh"},
		{name: "missing target", source: `decimate() dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): provide exactly one of triangles_fraction or triangle_count"},
		{name: "both targets", source: `decimate(0.5, triangle_count=10) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): provide exactly one of triangles_fraction or triangle_count"},
		{name: "negative fraction", source: `decimate(-0.1) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): triangles_fraction must be between 0 and 1"},
		{name: "fraction too large", source: `decimate(1.1) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): triangles_fraction must be between 0 and 1"},
		{name: "fractional triangle count", source: `decimate(triangle_count=10.5) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): triangle_count must be an integer"},
		{name: "negative triangle count", source: `decimate(triangle_count=-1) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): triangle_count must be >= 0"},
		{name: "negative min determinant", source: `decimate(0.5, min_det=-1) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): min_det must be >= 0"},
		{name: "normal dot product too small", source: `decimate(0.5, min_normal_dot_product=-1.1) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): min_normal_dot_product must be between -1 and 1"},
		{name: "normal dot product too large", source: `decimate(0.5, min_normal_dot_product=1.1) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): min_normal_dot_product must be between -1 and 1"},
		{name: "negative regularization", source: `decimate(0.5, tikhonov_regularization=-1) dual_contour() sphere();`, mesh: validMesh, wantErr: "decimate(): tikhonov_regularization must be >= 0"},
		{name: "non-manifold mesh", source: `decimate(0.5) dual_contour() sphere();`, mesh: openMesh, wantErr: "decimate(): mesh must be manifold"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog, err := Parse(test.source)
			if err != nil {
				t.Fatal(err)
			}
			hooks := Hooks{}
			if test.mesh != nil {
				hooks.DualContour = func(ShapeRep, float64, bool, bool) (*model3d.Mesh, error) {
					return test.mesh, nil
				}
			}
			_, err = Eval(prog, hooks)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func assertMeshes2DEqual(t *testing.T, actual, expected *model2d.Mesh) {
	t.Helper()
	actualSegments := actual.SegmentSlice()
	expectedSegments := expected.SegmentSlice()
	if len(actualSegments) != len(expectedSegments) {
		t.Fatalf("segment count: got %d want %d", len(actualSegments), len(expectedSegments))
	}
	matched := make([]bool, len(expectedSegments))
	for _, actualSegment := range actualSegments {
		found := false
		for i, expectedSegment := range expectedSegments {
			if matched[i] {
				continue
			}
			forward := actualSegment[0].Dist(expectedSegment[0]) < 1e-8 &&
				actualSegment[1].Dist(expectedSegment[1]) < 1e-8
			reverse := actualSegment[0].Dist(expectedSegment[1]) < 1e-8 &&
				actualSegment[1].Dist(expectedSegment[0]) < 1e-8
			if forward || reverse {
				matched[i] = true
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("no matching expected segment for %v", actualSegment)
		}
	}
}
