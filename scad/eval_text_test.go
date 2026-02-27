package scad

import (
	"math/rand"
	"strings"
	"testing"

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

func TestTextUnsupportedFont(t *testing.T) {
	prog, err := Parse(`text("Hello", font="Arial");`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, err = Eval(prog)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unsupported font") {
		t.Fatalf("expected unsupported font error, got: %v", err)
	}
}

func assertSolids2DEqual(t *testing.T, a, b model2d.Solid) {
	t.Helper()
	min := a.Min().Min(b.Min())
	max := a.Max().Max(b.Max())
	rng := rand.New(rand.NewSource(1337))
	for i := 0; i < 2000; i++ {
		p := model2d.XY(
			min.X+rng.Float64()*(max.X-min.X),
			min.Y+rng.Float64()*(max.Y-min.Y),
		)
		av := a.Contains(p)
		bv := b.Contains(p)
		if av != bv {
			t.Fatalf("contains mismatch at %v: %v != %v", p, av, bv)
		}
	}
}
