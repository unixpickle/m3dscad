package scad

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/unixpickle/model3d/model2d"
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
