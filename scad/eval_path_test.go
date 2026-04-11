package scad

import (
	"math"
	"math/rand"
	"testing"

	"github.com/unixpickle/model3d/model3d"
)

func TestPathMeshAndSDFParity(t *testing.T) {
	srcPath := `
		path("M0 0 L4 0 L4 4 L0 4 Z", segments=200);
	`
	srcPathMesh := `
		solid() path_mesh("M0 0 L4 0 L4 4 L0 4 Z", segments=200);
	`
	srcPathSDF := `
		solid() path_sdf("M0 0 L4 0 L4 4 L0 4 Z", segments=200);
	`

	shapeSolid := mustEvalShape(t, srcPath)
	shapeMesh := mustEvalShape(t, srcPathMesh)
	shapeSDF := mustEvalShape(t, srcPathSDF)
	if shapeSolid.Kind != ShapeSolid2D || shapeMesh.Kind != ShapeSolid2D || shapeSDF.Kind != ShapeSolid2D {
		t.Fatalf("expected 2D solids, got %v, %v, %v", shapeSolid.Kind, shapeMesh.Kind, shapeSDF.Kind)
	}

	assertSolids2DEqual(t, shapeSolid.S2, shapeMesh.S2)
	assertSolids2DEqual(t, shapeSolid.S2, shapeSDF.S2)
}

func TestPathMeshAndSDFOutputKinds(t *testing.T) {
	meshShape := mustEvalShape(t, `
		path_mesh("M0 0 L1 0 L1 1 Z");
	`)
	if meshShape.Kind != ShapeMesh2D {
		t.Fatalf("expected ShapeMesh2D, got %v", meshShape.Kind)
	}

	sdfShape := mustEvalShape(t, `
		path_sdf("M0 0 L1 0 L1 1 Z");
	`)
	if sdfShape.Kind != ShapeSDF2D {
		t.Fatalf("expected ShapeSDF2D, got %v", sdfShape.Kind)
	}
}

func TestPathMeshDefaultSegments(t *testing.T) {
	meshShape := mustEvalShape(t, `
		path_mesh("M0 0 L1 0 L1 1 Z");
	`)
	if meshShape.Kind != ShapeMesh2D {
		t.Fatalf("expected ShapeMesh2D, got %v", meshShape.Kind)
	}
	if got := meshShape.M2.NumSegments(); got != 1000 {
		t.Fatalf("expected 1000 segments by default, got %d", got)
	}
}

func TestPathMultipleSubpaths(t *testing.T) {
	srcExpected := `
		union() {
			square([1, 1], center=false);
			translate([2, 0, 0]) square([1, 1], center=false);
		}
	`
	srcPath := `
		path("M0 0 L1 0 L1 1 L0 1 Z M2 0 L3 0 L3 1 L2 1 Z", segments=200);
	`
	srcPathMesh := `
		solid() path_mesh("M0 0 L1 0 L1 1 L0 1 Z M2 0 L3 0 L3 1 L2 1 Z", segments=200);
	`
	srcPathSDF := `
		solid() path_sdf("M0 0 L1 0 L1 1 L0 1 Z M2 0 L3 0 L3 1 L2 1 Z", segments=200);
	`

	shapeExpected := mustEvalShape(t, srcExpected)
	shapePath := mustEvalShape(t, srcPath)
	shapePathMesh := mustEvalShape(t, srcPathMesh)
	shapePathSDF := mustEvalShape(t, srcPathSDF)
	if shapeExpected.Kind != ShapeSolid2D || shapePath.Kind != ShapeSolid2D ||
		shapePathMesh.Kind != ShapeSolid2D || shapePathSDF.Kind != ShapeSolid2D {
		t.Fatalf(
			"expected 2D solids, got %v, %v, %v, %v",
			shapeExpected.Kind,
			shapePath.Kind,
			shapePathMesh.Kind,
			shapePathSDF.Kind,
		)
	}

	assertSolids2DEqual(t, shapeExpected.S2, shapePath.S2)
	assertSolids2DEqual(t, shapeExpected.S2, shapePathMesh.S2)
	assertSolids2DEqual(t, shapeExpected.S2, shapePathSDF.S2)
}

func TestPathMatchesPrimitive(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "Square",
			a: `
				linear_extrude(height=1) square(2, center=true);
			`,
			b: `
				linear_extrude(height=1) path("M-1,-1 h2 v2 h-2 z");
			`,
		},
		{
			name: "Circle",
			a: `
				linear_extrude(height=1) circle(1);
			`,
			b: `
				linear_extrude(height=1) path("M1,0 A1,1 0 1 0 -1,0 A1,1 0 1 0 1,0");
			`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			solidA := mustEvalSolid(t, tc.a)
			solidB := mustEvalSolid(t, tc.b)

			deltaA := marchingDelta(solidA, openscadTestMaxGridSide)
			deltaB := marchingDelta(solidB, openscadTestMaxGridSide)
			delta := math.Max(deltaA, deltaB)
			if delta <= 0 {
				t.Fatalf("invalid marching delta: %v", delta)
			}

			meshA := model3d.MarchingCubesSearch(solidA, delta, openscadTestMCIters)
			meshB := model3d.MarchingCubesSearch(solidB, delta, openscadTestMCIters)

			threshold := math.Max(3*delta, 0.02)
			rng := rand.New(rand.NewSource(int64(len(tc.name)) * 131))
			compareMeshes(t, "a_vs_b", meshA, meshB, threshold, rng)
			compareMeshes(t, "b_vs_a", meshB, meshA, threshold, rng)
		})
	}
}
