package scad

import (
	"strings"
	"testing"

	"github.com/unixpickle/model3d/model3d"
)

func TestMetaballPrimitiveKindsAndDefaultSign(t *testing.T) {
	shape3D := mustEvalShape(t, `sphere_metaball(r=1);`)
	if shape3D.Kind != ShapeMetaball3D {
		t.Fatalf("expected ShapeMetaball3D, got %v", shape3D.Kind)
	}
	if shape3D.MB3 == nil || len(shape3D.MB3.Balls) != 1 || len(shape3D.MB3.Weights) != 1 || shape3D.MB3.Weights[0] != 1 {
		t.Fatalf("expected one 3D metaball with weight 1")
	}

	shape2D := mustEvalShape(t, `circle_metaball(r=1);`)
	if shape2D.Kind != ShapeMetaball2D {
		t.Fatalf("expected ShapeMetaball2D, got %v", shape2D.Kind)
	}
	if shape2D.MB2 == nil || len(shape2D.MB2.Balls) != 1 || len(shape2D.MB2.Weights) != 1 || shape2D.MB2.Weights[0] != 1 {
		t.Fatalf("expected one 2D metaball with weight 1")
	}

	capsuleShape := mustEvalShape(t, `capsule_metaball(h=2, r=1, center=true);`)
	if capsuleShape.Kind != ShapeMetaball3D {
		t.Fatalf("expected ShapeMetaball3D, got %v", capsuleShape.Kind)
	}
	if capsuleShape.MB3 == nil || len(capsuleShape.MB3.Balls) != 1 || len(capsuleShape.MB3.Weights) != 1 || capsuleShape.MB3.Weights[0] != 1 {
		t.Fatalf("expected one 3D metaball with weight 1")
	}
}

func TestMetaballFromSDF(t *testing.T) {
	shape := mustEvalShape(t, `metaball() sphere_sdf(r=2);`)
	if shape.Kind != ShapeMetaball3D {
		t.Fatalf("expected ShapeMetaball3D, got %v", shape.Kind)
	}
	if shape.MB3 == nil || len(shape.MB3.Balls) != 1 || len(shape.MB3.Weights) != 1 || shape.MB3.Weights[0] != 1 {
		t.Fatalf("expected one 3D metaball with weight 1")
	}
}

func TestScaleMetaball(t *testing.T) {
	shape := mustEvalShape(t, `weight_metaball(-1) sphere_metaball(r=1);`)
	if shape.Kind != ShapeMetaball3D {
		t.Fatalf("expected ShapeMetaball3D, got %v", shape.Kind)
	}
	if shape.MB3 == nil || len(shape.MB3.Balls) != 1 || len(shape.MB3.Weights) != 1 || shape.MB3.Weights[0] != -1 {
		t.Fatalf("expected one 3D metaball with weight -1")
	}
}

func TestMetaballSolidDefaultsToQuartic(t *testing.T) {
	solid := mustEvalSolid(t, `metaball_solid(1) sphere_metaball(r=1);`)
	if !solid.Contains(model3d.XYZ(1.9, 0, 0)) {
		t.Fatalf("expected point to be inside metaball solid")
	}
	if solid.Contains(model3d.XYZ(2.1, 0, 0)) {
		t.Fatalf("expected point to be outside metaball solid")
	}
}

func TestMetaballSolidWithNegationAndFalloff(t *testing.T) {
	solid := mustEvalSolid(t, `
		metaball_solid(1, "gaussian") {
			sphere_metaball(r=1);
			weight_metaball(-1) translate([3, 0, 0]) sphere_metaball(r=1);
		}
	`)
	if !solid.Contains(model3d.XYZ(0.8, 0, 0)) {
		t.Fatalf("expected positive metaball region to remain")
	}
	if solid.Contains(model3d.XYZ(3, 0, 0)) {
		t.Fatalf("expected negated metaball region to be removed")
	}
}

func TestMetaballSolid2DOutputKind(t *testing.T) {
	shape := mustEvalShape(t, `metaball_solid(1) circle_metaball(r=1);`)
	if shape.Kind != ShapeSolid2D {
		t.Fatalf("expected ShapeSolid2D, got %v", shape.Kind)
	}
}

func TestMetaballSolidUnknownFalloff(t *testing.T) {
	prog, err := Parse(`metaball_solid(1, "nope") sphere_metaball(r=1);`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	_, err = Eval(prog, Hooks{})
	if err == nil {
		t.Fatal("expected eval error")
	}
	if !strings.Contains(err.Error(), "unknown falloff") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMetaballForLoop(t *testing.T) {
	withLoop := `
		metaball_solid(1) {
			for (x = [-1, 1])
				translate([x, 0, 0]) sphere_metaball(r=1);
		}
	`
	withoutLoop := `
		metaball_solid(1) {
			for (x = [-1, 1])
				translate([x, 0, 0]) sphere_metaball(r=1);
		}
	`

	loopSolid := mustEvalShape(t, withLoop)
	withoutLoopSolid := mustEvalShape(t, withoutLoop)
	if loopSolid.Kind != ShapeSolid3D || withoutLoopSolid.Kind != ShapeSolid3D {
		t.Fatalf("expected 3D solids, got %v, %v", loopSolid.Kind, withoutLoopSolid.Kind)
	}

	assertSolids3DEqual(t, loopSolid.S3, withoutLoopSolid.S3)
}
