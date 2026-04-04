package scad

import (
	"fmt"
	"math"
	"math/rand"
	"testing"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func TestSolidConversions(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "LinearExtrudeCircleSDF",
			a: `
				linear_extrude(height=3) solid() circle_sdf(r=3);
			`,
			b: `
				linear_extrude(height=3) circle(r=3);
			`,
		},
		{
			name: "TranslatedSphereSDF",
			a: `
				solid() translate([1, 2, 3]) sphere_sdf(r=3);
			`,
			b: `
				translate([1, 2, 3]) sphere(r=3);
			`,
		},
		{
			name: "LinearExtrudeSquareSDF",
			a: `
				linear_extrude(height=2) solid() square_sdf(size=[2, 4], center=true);
			`,
			b: `
				linear_extrude(height=2) square(size=[2, 4], center=true);
			`,
		},
		{
			name: "TranslatedCylinderSDF",
			a: `
				solid() translate([0, 1, -2]) cylinder_sdf(h=4, r=1.5, center=true);
			`,
			b: `
				translate([0, 1, -2]) cylinder(h=4, r=1.5, center=true);
			`,
		},
		{
			name: "RotatedScaledSphereSDF",
			a: `
				solid() rotate([30, 45, 60]) scale([2, 2, 2]) sphere_sdf(r=1.25);
			`,
			b: `
				rotate([30, 45, 60]) scale([2, 2, 2]) sphere(r=1.25);
			`,
		},
		{
			name: "ReflectedSphereSDF",
			a: `
				solid() scale([-2, 2, -2]) sphere_sdf(r=1.25);
			`,
			b: `
				scale([-2, 2, -2]) sphere(r=1.25);
			`,
		},
		{
			name: "TranslatedCapsuleSDF",
			a: `
				solid() translate([2, -1, 3]) capsule_sdf(h=4, r=1.5, center=true);
			`,
			b: `
				translate([2, -1, 3]) capsule(h=4, r=1.5, center=true);
			`,
		},
		{
			name: "MirroredSphereSDF",
			a: `
				solid() mirror([1, 0, 0]) translate([2, 0, 0]) sphere_sdf(r=1.25);
			`,
			b: `
				scale([-1, 1, 1]) translate([2, 0, 0]) sphere(r=1.25);
			`,
		},
		{
			name: "RotatedCircleExtrudeSDF",
			a: `
				linear_extrude(height=4)
					solid() rotate([0, 0, 25]) scale([1.5, 1.5, 0]) circle_sdf(r=2.1);
			`,
			b: `
				linear_extrude(height=4)
					rotate([0, 0, 25]) scale([1.5, 1.5, 0]) circle(r=2.1);
			`,
		},
		{
			name: "ReflectedCircleExtrudeSDF",
			a: `
				linear_extrude(height=4)
					solid() scale([-1.5, 1.5, 0]) circle_sdf(r=2.1);
			`,
			b: `
				linear_extrude(height=4)
					scale([-1.5, 1.5, 0]) circle(r=2.1);
			`,
		},
		{
			name: "MirroredCircleExtrudeSDF",
			a: `
				linear_extrude(height=4)
					solid() scale([-1, 1, 1]) translate([1.5, 0, 0]) circle_sdf(r=2.1);
			`,
			b: `
				linear_extrude(height=4)
					mirror([1, 0, 0]) translate([1.5, 0, 0]) circle(r=2.1);
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

func TestHull2DConversions(t *testing.T) {
	hullShape := mustEvalShape(t, `
		union() {
			cirlce_hull(r=0);
			translate([2, 0, 0]) circle_hull(r=1);
		}
	`)
	if hullShape.Kind != ShapeHull2D {
		t.Fatalf("expected ShapeHull2D, got %v", hullShape.Kind)
	}
	if len(hullShape.H2.Circles) != 2 {
		t.Fatalf("expected 2 hull circles, got %d", len(hullShape.H2.Circles))
	}
	if hullShape.H2.Circles[0].Radius != 0 {
		t.Fatalf("expected zero-radius circle to be preserved, got %v", hullShape.H2.Circles[0].Radius)
	}
	if hullShape.H2.Circles[1].Center != model2d.XY(2, 0) {
		t.Fatalf("expected translated hull circle center, got %v", hullShape.H2.Circles[1].Center)
	}

	want := model2d.NewArcHull([]*model2d.Circle{
		{Center: model2d.XY(0, 0), Radius: 0},
		{Center: model2d.XY(2, 0), Radius: 1},
	})

	solidShape := mustEvalShape(t, `
		hull_solid() union() {
			cirlce_hull(r=0);
			translate([2, 0, 0]) circle_hull(r=1);
		}
	`)
	if solidShape.Kind != ShapeSolid2D {
		t.Fatalf("expected ShapeSolid2D, got %v", solidShape.Kind)
	}
	assertSolids2DEqual(t, solidShape.S2, want)

	sdfShape := mustEvalShape(t, `
		hull_sdf() union() {
			cirlce_hull(r=0);
			translate([2, 0, 0]) circle_hull(r=1);
		}
	`)
	if sdfShape.Kind != ShapeSDF2D {
		t.Fatalf("expected ShapeSDF2D, got %v", sdfShape.Kind)
	}
	assertSDFsEqual2D(t, sdfShape.SDF2, want, want.Min(), want.Max(), 1e-8)
}

func TestHull2DZeroRadiusFallback(t *testing.T) {
	t.Run("ConvexHullMesh", func(t *testing.T) {
		wantMesh := model2d.ConvexHullMesh([]model2d.Coord{
			model2d.XY(0, 0),
			model2d.XY(2, 0),
			model2d.XY(0, 1),
		})
		wantSolid := wantMesh.Solid()
		wantSDF := model2d.MeshToSDF(wantMesh)

		solidShape := mustEvalShape(t, `
			hull_solid() union() {
				circle_hull(r=0);
				translate([2, 0, 0]) circle_hull(r=0);
				translate([0, 1, 0]) circle_hull(r=0);
			}
		`)
		if solidShape.Kind != ShapeSolid2D {
			t.Fatalf("expected ShapeSolid2D, got %v", solidShape.Kind)
		}
		assertSolids2DEqual(t, solidShape.S2, wantSolid)

		sdfShape := mustEvalShape(t, `
			hull_sdf() union() {
				circle_hull(r=0);
				translate([2, 0, 0]) circle_hull(r=0);
				translate([0, 1, 0]) circle_hull(r=0);
			}
		`)
		if sdfShape.Kind != ShapeSDF2D {
			t.Fatalf("expected ShapeSDF2D, got %v", sdfShape.Kind)
		}
		assertSDFsEqual2D(t, sdfShape.SDF2, wantSDF, wantSDF.Min(), wantSDF.Max(), 1e-8)
	})

	t.Run("SinglePoint", func(t *testing.T) {
		solidShape := mustEvalShape(t, `hull_solid() circle_hull(r=0);`)
		if solidShape.Kind != ShapeSolid2D {
			t.Fatalf("expected ShapeSolid2D, got %v", solidShape.Kind)
		}
		if !solidShape.S2.Contains(model2d.Coord{}) {
			t.Fatal("expected degenerate hull solid to contain its only center")
		}
		if solidShape.S2.Contains(model2d.XY(1e-3, 0)) {
			t.Fatal("expected degenerate hull solid to exclude other points")
		}

		sdfShape := mustEvalShape(t, `hull_sdf() circle_hull(r=0);`)
		if sdfShape.Kind != ShapeSDF2D {
			t.Fatalf("expected ShapeSDF2D, got %v", sdfShape.Kind)
		}
		if d := sdfShape.SDF2.SDF(model2d.Coord{}); d != 0 {
			t.Fatalf("expected zero SDF at the hull point, got %v", d)
		}
		if d := sdfShape.SDF2.SDF(model2d.XY(1, 0)); d >= 0 {
			t.Fatalf("expected negative SDF away from the hull point, got %v", d)
		}
	})
}

func TestHull2DTriangleEquivalentToPolygon(t *testing.T) {
	polygonShape := mustEvalShape(t, `
		polygon(points=[[0,0], [2,0], [0.5,1.5]]);
	`)
	hullShape := mustEvalShape(t, `
		hull_solid() union() {
			translate([0, 0, 0]) circle_hull(r=0);
			translate([2, 0, 0]) circle_hull(r=0);
			translate([0.5, 1.5, 0]) circle_hull(r=0);
		}
	`)
	if polygonShape.Kind != ShapeSolid2D || hullShape.Kind != ShapeSolid2D {
		t.Fatalf("expected ShapeSolid2D outputs, got %v and %v", polygonShape.Kind, hullShape.Kind)
	}
	assertSolids2DEqual(t, polygonShape.S2, hullShape.S2)
}

func TestHull2DRoundedTriangleEquivalentToOutsetPolygon(t *testing.T) {
	polygonShape := mustEvalShape(t, `
		solid() outset_sdf(0.1)
			polygon_sdf(points=[[0,0], [2,0], [0.5,1.5]]);
	`)
	hullShape := mustEvalShape(t, `
		hull_solid() union() {
			translate([0, 0, 0]) circle_hull(r=0.1);
			translate([2, 0, 0]) circle_hull(r=0.1);
			translate([0.5, 1.5, 0]) circle_hull(r=0.1);
		}
	`)
	if polygonShape.Kind != ShapeSolid2D || hullShape.Kind != ShapeSolid2D {
		t.Fatalf("expected ShapeSolid2D outputs, got %v and %v", polygonShape.Kind, hullShape.Kind)
	}
	assertSolids2DEqual(t, polygonShape.S2, hullShape.S2)
}

func TestHull2DRoundedTriangleSDFEquivalentToOutsetPolygon(t *testing.T) {
	polygonShape := mustEvalShape(t, `
		outset_sdf(0.1)
			polygon_sdf(points=[[0,0], [2,0], [0.5,1.5]]);
	`)
	hullShape := mustEvalShape(t, `
		hull_sdf() union() {
			translate([0, 0, 0]) circle_hull(r=0.1);
			translate([2, 0, 0]) circle_hull(r=0.1);
			translate([0.5, 1.5, 0]) circle_hull(r=0.1);
		}
	`)
	if polygonShape.Kind != ShapeSDF2D || hullShape.Kind != ShapeSDF2D {
		t.Fatalf("expected ShapeSDF2D outputs, got %v and %v", polygonShape.Kind, hullShape.Kind)
	}

	min := polygonShape.SDF2.Min().Min(hullShape.SDF2.Min()).AddScalar(-0.2)
	max := polygonShape.SDF2.Max().Max(hullShape.SDF2.Max()).AddScalar(0.2)
	rng := rand.New(rand.NewSource(1337))
	for i := 0; i < 2000; i++ {
		p := model2d.NewCoordRandBounds(min, max, rng)
		a := polygonShape.SDF2.SDF(p)
		b := hullShape.SDF2.SDF(p)
		if math.Abs(a-b) > 1e-5 {
			t.Fatalf("SDF mismatch at %v: a=%f b=%f", p, a, b)
		}
	}
}

func TestPolygonHullEquivalentToTriangleHull(t *testing.T) {
	shapeA := mustEvalShape(t, `
		hull_solid() polygon_hull(points=[[0,0], [2,0], [0.5,1.5]]);
	`)
	shapeB := mustEvalShape(t, `
		hull_solid() union() {
			translate([0, 0, 0]) circle_hull(r=0);
			translate([2, 0, 0]) circle_hull(r=0);
			translate([0.5, 1.5, 0]) circle_hull(r=0);
		}
	`)
	if shapeA.Kind != ShapeSolid2D || shapeB.Kind != ShapeSolid2D {
		t.Fatalf("expected ShapeSolid2D outputs, got %v and %v", shapeA.Kind, shapeB.Kind)
	}
	assertSolids2DEqual(t, shapeA.S2, shapeB.S2)
}

func TestMeshToHullEquivalentToTriangleHull(t *testing.T) {
	shapeA := mustEvalShape(t, `
		hull_solid() mesh_to_hull()
			polygon_mesh(points=[[0,0], [2,0], [0.5,1.5]]);
	`)
	shapeB := mustEvalShape(t, `
		hull_solid() union() {
			translate([0, 0, 0]) circle_hull(r=0);
			translate([2, 0, 0]) circle_hull(r=0);
			translate([0.5, 1.5, 0]) circle_hull(r=0);
		}
	`)
	if shapeA.Kind != ShapeSolid2D || shapeB.Kind != ShapeSolid2D {
		t.Fatalf("expected ShapeSolid2D outputs, got %v and %v", shapeA.Kind, shapeB.Kind)
	}
	assertSolids2DEqual(t, shapeA.S2, shapeB.S2)
}

func TestCapsuleEquivalentToCylinderWithSpheres(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
	}{
		{
			name: "CenterFalse",
			a: `
				capsule(h=6, r=1.25, center=false);
			`,
			b: `
				union() {
					cylinder(h=6, r=1.25, center=false);
					translate([0, 0, 0]) sphere(r=1.25);
					translate([0, 0, 6]) sphere(r=1.25);
				}
			`,
		},
		{
			name: "CenterTrue",
			a: `
				capsule(h=6, r=1.25, center=true);
			`,
			b: `
				union() {
					cylinder(h=6, r=1.25, center=true);
					translate([0, 0, -3]) sphere(r=1.25);
					translate([0, 0, 3]) sphere(r=1.25);
				}
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
			rng := rand.New(rand.NewSource(int64(len(tc.name)) * 173))
			compareMeshes(t, "capsule_vs_constructed", meshA, meshB, threshold, rng)
			compareMeshes(t, "constructed_vs_capsule", meshB, meshA, threshold, rng)
		})
	}
}

func TestTeardropEquivalentToCircleWithTriangle(t *testing.T) {
	tests := []struct {
		name   string
		radius float64
	}{
		{
			name:   "RadiusOne",
			radius: 1.0,
		},
		{
			name:   "RadiusFractional",
			radius: 1.25,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srcA := fmt.Sprintf(`
				linear_extrude(height=2) teardrop(radius=%v);
			`, tc.radius)
			srcB := fmt.Sprintf(`
				linear_extrude(height=2)
					union() {
						circle(r=%v);
						polygon(points=[
							[-%v/sqrt(2), %v/sqrt(2)],
							[%v/sqrt(2), %v/sqrt(2)],
							[0, %v*sqrt(2)]
						]);
					}
			`, tc.radius, tc.radius, tc.radius, tc.radius, tc.radius, tc.radius)

			solidA := mustEvalSolid(t, srcA)
			solidB := mustEvalSolid(t, srcB)

			deltaA := marchingDelta(solidA, openscadTestMaxGridSide)
			deltaB := marchingDelta(solidB, openscadTestMaxGridSide)
			delta := math.Max(deltaA, deltaB)
			if delta <= 0 {
				t.Fatalf("invalid marching delta: %v", delta)
			}

			meshA := model3d.MarchingCubesSearch(solidA, delta, openscadTestMCIters)
			meshB := model3d.MarchingCubesSearch(solidB, delta, openscadTestMCIters)

			threshold := math.Max(3*delta, 0.02)
			rng := rand.New(rand.NewSource(int64(len(tc.name)) * 181))
			compareMeshes(t, "teardrop_vs_constructed", meshA, meshB, threshold, rng)
			compareMeshes(t, "constructed_vs_teardrop", meshB, meshA, threshold, rng)
		})
	}
}

func TestLinearExtrudeMeshAndSDF2D(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		wantKind ShapeKind
	}{
		{
			name: "Mesh2D",
			a: `
				linear_extrude(height=2, center=true)
					polygon_mesh(points=[[-1,-1], [1,-1], [1,1], [-1,1]]);
			`,
			b: `
				linear_extrude(height=2, center=true) square(2, center=true);
			`,
			wantKind: ShapeMesh3D,
		},
		{
			name: "SDF2D",
			a: `
				linear_extrude(height=3) circle_sdf(r=3);
			`,
			b: `
				linear_extrude(height=3) circle(r=3);
			`,
			wantKind: ShapeSDF3D,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			shapeA := mustEvalShape(t, tc.a)
			if shapeA.Kind != tc.wantKind {
				t.Fatalf("expected kind %v, got %v", tc.wantKind, shapeA.Kind)
			}
			solidA, err := shapeToSolid3D(shapeA)
			if err != nil {
				t.Fatalf("shapeToSolid3D(a): %v", err)
			}
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
			rng := rand.New(rand.NewSource(int64(len(tc.name)) * 157))

			compareMeshes(t, "a_vs_b", meshA, meshB, threshold, rng)
			compareMeshes(t, "b_vs_a", meshB, meshA, threshold, rng)
		})
	}
}

func TestPolygonMeshAndSDFParity(t *testing.T) {
	srcPolygon := `
		polygon(
			points=[[0,0], [4,0], [4,4], [0,4], [1,1], [3,1], [3,3], [1,3]],
			paths=[[0,1,2,3], [4,5,6,7]]
		);
	`
	srcPolygonMesh := `
		solid() polygon_mesh(
			points=[[0,0], [4,0], [4,4], [0,4], [1,1], [3,1], [3,3], [1,3]],
			paths=[[0,1,2,3], [4,5,6,7]]
		);
	`
	srcPolygonSDF := `
		solid() polygon_sdf(
			points=[[0,0], [4,0], [4,4], [0,4], [1,1], [3,1], [3,3], [1,3]],
			paths=[[0,1,2,3], [4,5,6,7]]
		);
	`

	shapeSolid := mustEvalShape(t, srcPolygon)
	shapeMesh := mustEvalShape(t, srcPolygonMesh)
	shapeSDF := mustEvalShape(t, srcPolygonSDF)
	if shapeSolid.Kind != ShapeSolid2D || shapeMesh.Kind != ShapeSolid2D || shapeSDF.Kind != ShapeSolid2D {
		t.Fatalf("expected 2D solids, got %v, %v, %v", shapeSolid.Kind, shapeMesh.Kind, shapeSDF.Kind)
	}

	assertSolids2DEqual(t, shapeSolid.S2, shapeMesh.S2)
	assertSolids2DEqual(t, shapeSolid.S2, shapeSDF.S2)
}

func TestPolygonMeshAndSDFOutputKinds(t *testing.T) {
	meshShape := mustEvalShape(t, `
		polygon_mesh(points=[[0,0], [1,0], [0,1]]);
	`)
	if meshShape.Kind != ShapeMesh2D {
		t.Fatalf("expected ShapeMesh2D, got %v", meshShape.Kind)
	}

	sdfShape := mustEvalShape(t, `
		polygon_sdf(points=[[0,0], [1,0], [0,1]]);
	`)
	if sdfShape.Kind != ShapeSDF2D {
		t.Fatalf("expected ShapeSDF2D, got %v", sdfShape.Kind)
	}
}

func assertSolids3DEqual(t *testing.T, a, b model3d.Solid) {
	t.Helper()
	min := a.Min().Min(b.Min())
	max := a.Max().Max(b.Max())
	rng := rand.New(rand.NewSource(1337))
	for i := 0; i < 2000; i++ {
		p := model3d.NewCoord3DRandBounds(min, max, rng)
		av := a.Contains(p)
		bv := b.Contains(p)
		if av != bv {
			t.Fatalf("contains mismatch at %v: %v != %v", p, av, bv)
		}
	}
}
