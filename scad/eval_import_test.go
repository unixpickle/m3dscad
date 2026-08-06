package scad

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/unixpickle/model3d/model3d"
)

func TestImportSTLAsMesh(t *testing.T) {
	input := model3d.NewMeshRect(model3d.XYZ(1, 2, 3), model3d.XYZ(2, 4, 6))
	files := fstest.MapFS{
		"assets/box.stl": {Data: input.EncodeSTL()},
	}
	prog, err := ParseFile("models/main.scad", `import("../assets/box.stl");`)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	if shape.Kind != ShapeMesh3D {
		t.Fatalf("expected ShapeMesh3D, got %v", shape.Kind)
	}
	if got, want := shape.M3.NumTriangles(), input.NumTriangles(); got != want {
		t.Fatalf("triangle count: got %d want %d", got, want)
	}
	if got, want := shape.M3.Min(), input.Min(); got != want {
		t.Fatalf("min: got %v want %v", got, want)
	}
	if got, want := shape.M3.Max(), input.Max(); got != want {
		t.Fatalf("max: got %v want %v", got, want)
	}
}

func TestImportSTLSolidScale(t *testing.T) {
	input := model3d.NewMeshRect(model3d.XYZ(1, 2, 3), model3d.XYZ(2, 4, 6))
	files := fstest.MapFS{
		"box.stl": {Data: input.EncodeSTL()},
	}
	prog, err := ParseFile("main.scad", `solid() scale(3) import("box.stl");`)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	if shape.Kind != ShapeSolid3D {
		t.Fatalf("expected ShapeSolid3D, got %v", shape.Kind)
	}
	if got, want := shape.S3.Min(), model3d.XYZ(3, 6, 9); got != want {
		t.Fatalf("min: got %v want %v", got, want)
	}
	if got, want := shape.S3.Max(), model3d.XYZ(6, 12, 18); got != want {
		t.Fatalf("max: got %v want %v", got, want)
	}
}

func TestImportSTLRelativeToUsedModule(t *testing.T) {
	input := model3d.NewMeshRect(model3d.XYZ(-1, -2, -3), model3d.XYZ(1, 2, 3))
	files := fstest.MapFS{
		"lib/importer.scad": {Data: []byte(`module imported() { import(file="asset.stl", convexity=4); }`)},
		"lib/asset.stl":     {Data: input.EncodeSTL()},
	}
	prog, err := ParseFile("main.scad", `use <lib/importer.scad> imported();`)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	if shape.Kind != ShapeMesh3D {
		t.Fatalf("expected ShapeMesh3D, got %v", shape.Kind)
	}
	if got, want := shape.M3.NumTriangles(), input.NumTriangles(); got != want {
		t.Fatalf("triangle count: got %d want %d", got, want)
	}
}

func TestImportSTLErrors(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		files   fstest.MapFS
		noFS    bool
		wantErr string
	}{
		{
			name:    "missing hook",
			source:  `import("mesh.stl");`,
			noFS:    true,
			wantErr: "Hooks.FS is not configured",
		},
		{
			name:    "missing file",
			source:  `import("missing.stl");`,
			files:   fstest.MapFS{},
			wantErr: `open imported file "missing.stl"`,
		},
		{
			name:    "unsupported extension",
			source:  `import("mesh.obj");`,
			files:   fstest.MapFS{},
			wantErr: "unsupported file type",
		},
		{
			name:    "malformed stl",
			source:  `import("broken.stl");`,
			files:   fstest.MapFS{"broken.stl": {Data: []byte("not an STL")}},
			wantErr: `decode imported STL "broken.stl"`,
		},
		{
			name:    "children",
			source:  `import("mesh.stl") cube(1);`,
			files:   fstest.MapFS{},
			wantErr: "import() does not take children",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog, err := ParseFile("main.scad", test.source)
			if err != nil {
				t.Fatal(err)
			}
			hooks := Hooks{FS: test.files}
			if test.noFS {
				hooks.FS = nil
			}
			_, err = Eval(prog, hooks)
			if err == nil || !strings.Contains(err.Error(), test.wantErr) {
				t.Fatalf("unexpected error: %v", err)
			}
			if !strings.Contains(err.Error(), "main.scad:1:1") {
				t.Fatalf("error lacks filename: %v", err)
			}
		})
	}
}
