package scad

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/unixpickle/model3d/model2d"
)

func TestImageMesh(t *testing.T) {
	files := fstest.MapFS{
		"assets/pixel.png": {Data: testBitmapPNG(t)},
	}
	prog, err := ParseFile("models/main.scad", `image("../assets/pixel.png");`)
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
	if got, want := shape.M2.NumSegments(), 4; got != want {
		t.Fatalf("segment count: got %d want %d", got, want)
	}
	if got, want := shape.M2.Min(), model2d.XY(1, 1); got != want {
		t.Fatalf("min: got %v want %v", got, want)
	}
	if got, want := shape.M2.Max(), model2d.XY(2, 2); got != want {
		t.Fatalf("max: got %v want %v", got, want)
	}
}

func TestImageRelativeToUsedModule(t *testing.T) {
	files := fstest.MapFS{
		"lib/picture.scad": {Data: []byte(`
module picture() {
    image("../assets/pixel.png");
}
`)},
		"assets/pixel.png": {Data: testBitmapPNG(t)},
	}
	prog, err := ParseFile("main.scad", `use <lib/picture.scad> picture();`)
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

func TestImageErrors(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		files   fstest.MapFS
		noFS    bool
		wantErr string
	}{
		{
			name:    "missing hook",
			source:  `image("pixel.png");`,
			noFS:    true,
			wantErr: "Hooks.FS is not configured",
		},
		{
			name:    "missing file",
			source:  `image("pixel.png");`,
			files:   fstest.MapFS{},
			wantErr: `open image "models/pixel.png"`,
		},
		{
			name:    "malformed image",
			source:  `image("bad.png");`,
			files:   fstest.MapFS{"models/bad.png": {Data: []byte("not an image")}},
			wantErr: `decode image "models/bad.png"`,
		},
		{
			name:    "path escapes root",
			source:  `image("../../bad.png");`,
			files:   fstest.MapFS{},
			wantErr: `invalid image path "../../bad.png"`,
		},
		{
			name:    "children",
			source:  `image("pixel.png") square(1);`,
			noFS:    true,
			wantErr: "image() does not take children",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog, err := ParseFile("models/main.scad", test.source)
			if err != nil {
				t.Fatal(err)
			}
			var hooks Hooks
			if !test.noFS {
				hooks.FS = test.files
			}
			_, err = Eval(prog, hooks)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("expected error containing %q, got %v", test.wantErr, err)
			}
		})
	}
}

func testBitmapPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			img.Set(x, y, color.White)
		}
	}
	img.Set(1, 1, color.Black)
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}
