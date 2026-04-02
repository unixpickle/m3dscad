package scad

import (
	"strings"
	"testing"

	"github.com/unixpickle/model3d/toolbox3d"
)

func TestInsetExtrudeMatchesToolbox3D(t *testing.T) {
	tests := []struct {
		name   string
		src    string
		sdfSrc string
		minZ   float64
		maxZ   float64
		want   toolbox3d.InsetFunc
	}{
		{
			name: "Defaults",
			src: `
				inset_extrude(h=3)
					circle_sdf(r=2);
			`,
			sdfSrc: `circle_sdf(r=2);`,
			minZ:   0,
			maxZ:   3,
			want: toolbox3d.InsetFuncSum(
				&toolbox3d.ChamferInsetFunc{BottomRadius: 0},
				&toolbox3d.ChamferInsetFunc{TopRadius: 0},
			),
		},
		{
			name: "CenteredMixedInsetOutset",
			src: `
				inset_extrude(height=4, center=true, bottom=0.5, top=-0.25, bottom_fn="fillet", top_fn="chamfer")
					square_sdf(size=[4, 3], center=true);
			`,
			sdfSrc: `square_sdf(size=[4, 3], center=true);`,
			minZ:   -2,
			maxZ:   2,
			want: toolbox3d.InsetFuncSum(
				&toolbox3d.FilletInsetFunc{BottomRadius: 0.5},
				&toolbox3d.ChamferInsetFunc{TopRadius: 0.25, Outwards: true},
			),
		},
		{
			name: "NegativeHeightUsesAlias",
			src: `
				inset_extrude(h=-2, bottom=-0.2, top=0.4, top_fn="fillet")
					circle_sdf(r=1.5);
			`,
			sdfSrc: `circle_sdf(r=1.5);`,
			minZ:   0,
			maxZ:   2,
			want: toolbox3d.InsetFuncSum(
				&toolbox3d.ChamferInsetFunc{BottomRadius: 0.2, Outwards: true},
				&toolbox3d.FilletInsetFunc{TopRadius: 0.4},
			),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			shape := mustEvalShape(t, tc.src)
			if shape.Kind != ShapeSolid3D {
				t.Fatalf("expected ShapeSolid3D, got %v", shape.Kind)
			}

			sdf := mustEvalShape(t, tc.sdfSrc)
			if sdf.Kind != ShapeSDF2D {
				t.Fatalf("expected ShapeSDF2D, got %v", sdf.Kind)
			}

			want := toolbox3d.Extrude(sdf.SDF2, tc.minZ, tc.maxZ, tc.want)
			assertSolids3DEqual(t, shape.S3, want)
		})
	}
}

func TestInsetExtrudeErrors(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantErr string
	}{
		{
			name: "RequiresSDF",
			src: `
				inset_extrude(height=1)
					circle(r=1);
			`,
			wantErr: "inset_extrude() requires 2D SDF children",
		},
		{
			name: "InvalidBottomFn",
			src: `
				inset_extrude(height=1, bottom_fn="bevel")
					circle_sdf(r=1);
			`,
			wantErr: `inset_extrude(): bottom_fn must be "chamfer" or "fillet"`,
		},
		{
			name: "InvalidTopFn",
			src: `
				inset_extrude(height=1, top_fn="round")
					circle_sdf(r=1);
			`,
			wantErr: `inset_extrude(): top_fn must be "chamfer" or "fillet"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := Parse(tc.src)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			_, err = Eval(prog)
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}
