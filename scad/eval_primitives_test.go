package scad

import (
	"math"
	"math/rand"
	"testing"

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
