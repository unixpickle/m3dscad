package scad

import (
	"math/rand"
	"strings"
	"testing"
	"testing/fstest"

	m3dfonts "github.com/unixpickle/m3dscad/fonts"
	"github.com/unixpickle/model3d/model2d"
)

func TestTextAndTextMeshParity(t *testing.T) {
	srcText := `
		text("Hello", size=4, halign="center", valign="center", spacing=1.05, segments=8);
	`
	srcTextMesh := `
		solid() text_mesh("Hello", size=4, halign="center", valign="center", spacing=1.05, segments=8);
	`
	shapeA := mustEvalShape(t, srcText)
	shapeB := mustEvalShape(t, srcTextMesh)
	if shapeA.Kind != ShapeSolid2D || shapeB.Kind != ShapeSolid2D {
		t.Fatalf("expected 2D solids, got %v and %v", shapeA.Kind, shapeB.Kind)
	}
	assertSolids2DEqual(t, shapeA.S2, shapeB.S2)
}

func TestTextAndTextSDFParity(t *testing.T) {
	srcText := `
		text("Hello", size=4, halign="center", valign="center", spacing=1.05, segments=8);
	`
	srcTextSDF := `
		solid() text_sdf("Hello", size=4, halign="center", valign="center", spacing=1.05, segments=8);
	`
	shapeA := mustEvalShape(t, srcText)
	shapeB := mustEvalShape(t, srcTextSDF)
	if shapeA.Kind != ShapeSolid2D || shapeB.Kind != ShapeSolid2D {
		t.Fatalf("expected 2D solids, got %v and %v", shapeA.Kind, shapeB.Kind)
	}
	assertSolids2DEqual(t, shapeA.S2, shapeB.S2)
}

func TestTextFontFile(t *testing.T) {
	files := fstest.MapFS{
		"assets/custom.ttf": {Data: m3dfonts.LiberationSansRegularTTF},
	}
	prog, err := ParseFile(
		"models/main.scad",
		`text("Hello", size=4, font="../assets/custom.ttf", halign="center", valign="center");`,
	)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	embedded := mustEvalShape(t, `text("Hello", size=4, halign="center", valign="center");`)
	if shape.Kind != ShapeSolid2D || embedded.Kind != ShapeSolid2D {
		t.Fatalf("expected 2D solids, got %v and %v", shape.Kind, embedded.Kind)
	}
	assertSolids2DEqual(t, shape.S2, embedded.S2)
}

func TestTextFontRelativeToUsedModule(t *testing.T) {
	files := fstest.MapFS{
		"lib/label.scad": {Data: []byte(`
module label() {
    text("Hello", font="../assets/custom.ttf");
}
`)},
		"assets/custom.ttf": {Data: m3dfonts.LiberationSansRegularTTF},
	}
	prog, err := ParseFile("main.scad", `use <lib/label.scad> label();`)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	if shape.Kind != ShapeSolid2D {
		t.Fatalf("expected ShapeSolid2D, got %v", shape.Kind)
	}
}

func TestTextFontFileErrors(t *testing.T) {
	tests := []struct {
		name    string
		font    string
		files   fstest.MapFS
		noFS    bool
		wantErr string
	}{
		{
			name:    "missing hook",
			font:    "custom.ttf",
			noFS:    true,
			wantErr: "Hooks.FS is not configured",
		},
		{
			name:    "missing file",
			font:    "custom.ttf",
			files:   fstest.MapFS{},
			wantErr: `read font "models/custom.ttf"`,
		},
		{
			name:    "malformed font",
			font:    "bad.ttf",
			files:   fstest.MapFS{"models/bad.ttf": {Data: []byte("not a font")}},
			wantErr: `parse font "models/bad.ttf"`,
		},
		{
			name:    "path escapes root",
			font:    "../../bad.ttf",
			files:   fstest.MapFS{},
			wantErr: `invalid font path "../../bad.ttf"`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			prog, err := ParseFile(
				"models/main.scad",
				`text("Hello", font="`+test.font+`");`,
			)
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

func assertSolids2DEqual(t *testing.T, a, b model2d.Solid) {
	t.Helper()
	min := a.Min().Min(b.Min())
	max := a.Max().Max(b.Max())
	rng := rand.New(rand.NewSource(1337))
	for i := 0; i < 2000; i++ {
		p := model2d.NewCoordRandBounds(min, max, rng)
		av := a.Contains(p)
		bv := b.Contains(p)
		if av != bv {
			t.Fatalf("contains mismatch at %v: %v != %v", p, av, bv)
		}
	}
}
