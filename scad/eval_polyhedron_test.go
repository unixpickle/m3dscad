package scad

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/unixpickle/model3d/model3d"
	shapekernel "github.com/unixpickle/webgpu-meshes/shapekernel"
)

const polyhedronBoxData = `
	points = [[0,0,0], [3,0,0], [3,2,0], [0,2,0],
	          [0,0,1], [3,0,1], [3,2,1], [0,2,1]];
	faces = [[0,1,2,3], [4,5,1,0], [7,6,5,4],
	         [5,6,2,1], [6,7,3,2], [7,4,0,3]];
`

func TestPolyhedronRepresentations(t *testing.T) {
	want := &model3d.Rect{MinVal: model3d.Origin, MaxVal: model3d.XYZ(3, 2, 1)}
	for _, tc := range []struct {
		module string
		kind   ShapeKind
	}{
		{"polyhedron", ShapeSolid3D},
		{"polyhedron_mesh", ShapeMesh3D},
		{"polyhedron_sdf", ShapeSDF3D},
	} {
		t.Run(tc.module, func(t *testing.T) {
			for _, args := range []string{
				`points, faces`,
				`points=points, faces=faces, convexity=10`,
				`points, faces, 4`,
				`points=points, triangles=faces`,
			} {
				shape := mustEvalShape(t, polyhedronBoxData+tc.module+"("+args+");")
				if shape.Kind != tc.kind {
					t.Fatalf("got kind %v, want %v", shape.Kind, tc.kind)
				}
				solid, err := shapeToSolid3D(shape)
				if err != nil {
					t.Fatal(err)
				}
				if solid.Min() != want.Min() || solid.Max() != want.Max() {
					t.Fatalf("unexpected bounds: %v %v", solid.Min(), solid.Max())
				}
				assertSolids3DEqual(t, solid, want)
				assertContains(t, solid, model3d.XYZ(-0.1, 1, 0.5), false)
				switch shape.Kind {
				case ShapeMesh3D:
					if shape.M3.NumTriangles() != 12 || shape.M3.NeedsRepair() {
						t.Fatal("expected a closed box with 12 triangles")
					}
					center := want.Max().Scale(0.5)
					shape.M3.Iterate(func(tri *model3d.Triangle) {
						if tri.Normal().Dot(tri[0].Sub(center)) <= 0 {
							t.Fatal("expected outward-facing triangle normals")
						}
					})
				case ShapeSDF3D:
					assertSDFsEqual3D(t, shape.SDF3, want, want.Min().AddScalar(-1), want.Max().AddScalar(1), 1e-8)
					if shape.Kernel == nil || shape.Kernel.Kind != shapekernel.SDF3D {
						t.Fatal("expected an SDF GPU kernel")
					}
				case ShapeSolid3D:
					if shape.Kernel == nil || shape.Kernel.Kind != shapekernel.Solid3D {
						t.Fatal("expected a solid GPU kernel")
					}
				}
			}
		})
	}
}

func TestPolyhedronConcaveFaces(t *testing.T) {
	data := `
		points = [[0,0,0], [3,0,0], [3,1,0], [1,1,0], [1,3,0], [0,3,0],
		          [0,0,2], [3,0,2], [3,1,2], [1,1,2], [1,3,2], [0,3,2]];
		faces = [[0,1,2,3,4,5], [11,10,9,8,7,6], [6,7,1,0], [7,8,2,1],
		         [8,9,3,2], [9,10,4,3], [10,11,5,4], [11,6,0,5]];
	`
	want := model3d.JoinedSolid{
		&model3d.Rect{MinVal: model3d.Origin, MaxVal: model3d.XYZ(3, 1, 2)},
		&model3d.Rect{MinVal: model3d.Origin, MaxVal: model3d.XYZ(1, 3, 2)},
	}
	for _, module := range []string{"polyhedron", "polyhedron_mesh", "polyhedron_sdf"} {
		t.Run(module, func(t *testing.T) {
			shape := mustEvalShape(t, data+module+"(points, faces);")
			solid, err := shapeToSolid3D(shape)
			if err != nil {
				t.Fatal(err)
			}
			assertSolids3DEqual(t, solid, want)
			assertContains(t, solid, model3d.XYZ(2, 2, 1), false)
			if shape.Kind == ShapeMesh3D && (shape.M3.NumTriangles() != 20 || shape.M3.NeedsRepair()) {
				t.Fatal("expected a closed concave prism with 20 triangles")
			}
		})
	}
}

func TestPolyhedronTriangles(t *testing.T) {
	data := `
		points = [[0,0,0], [2,0,0], [0,2,0], [0,0,2]];
		faces = [[0,1,2], [0,3,1], [0,2,3], [1,3,2]];
	`
	for _, module := range []string{"polyhedron", "polyhedron_mesh", "polyhedron_sdf"} {
		t.Run(module, func(t *testing.T) {
			solid := mustEvalSolid(t, data+module+"(points, faces);")
			assertContains(t, solid, model3d.XYZ(0.2, 0.3, 0.4), true)
			assertContains(t, solid, model3d.XYZ(1.5, 1.5, 1.5), false)
		})
	}
}

func TestPolyhedronOpenMeshAndClosingVertex(t *testing.T) {
	// Repeated coordinates in the points array are allowed. A face can repeat
	// its first vertex to close the loop, including via a different point index.
	shape := mustEvalShape(t, `polyhedron_mesh(
		[[0,0,0], [2,0,0], [2,1,0], [0,1,0], [0,0,0]], [[0,1,2,3,4]]
	);`)
	if shape.M3.NumTriangles() != 2 {
		t.Fatalf("expected 2 triangles, got %d", shape.M3.NumTriangles())
	}
	area := 0.0
	shape.M3.Iterate(func(tri *model3d.Triangle) {
		area += tri.Area()
		if tri.Normal().Z >= 0 {
			t.Fatal("expected downward-facing triangles")
		}
	})
	if math.Abs(area-2) > 1e-8 {
		t.Fatalf("area: got %v, want 2", area)
	}
}

func TestPolyhedronConversionsAndTransforms(t *testing.T) {
	want := mustEvalSolid(t, `translate([1,2,3]) scale(2) cube([3,2,1]);`)
	for _, call := range []string{
		`polyhedron(points, faces)`,
		`solid() polyhedron_mesh(points, faces)`,
		`solid() polyhedron_sdf(points, faces)`,
		`solid() mesh_to_sdf() polyhedron_mesh(points, faces)`,
	} {
		got := mustEvalSolid(t, polyhedronBoxData+`translate([1,2,3]) scale(2) `+call+`;`)
		assertSolids3DEqual(t, got, want)
	}
}

func TestPolyhedronErrors(t *testing.T) {
	for _, module := range []string{"polyhedron", "polyhedron_mesh", "polyhedron_sdf"} {
		for _, tc := range []struct {
			args string
			want string
		}{
			{``, `missing parameter "points"`},
			{`points=points`, `missing parameter "faces"`},
			{`points=1, faces=faces`, "points must be a list"},
			{`points=[], faces=faces`, "points must be a list"},
			{`points=[[0,0],[1,0],[0,1]], faces=[[0,1,2]]`, "points[0] must be a 3D vector"},
			{`points=[[0,0,0,1],[1,0,0],[0,1,0]], faces=[[0,1,2]]`, "points[0] must be a 3D vector"},
			{`points=[[0,0,"x"],[1,0,0],[0,1,0]], faces=[[0,1,2]]`, "coordinates must be finite numbers"},
			{`points=[[0,0,sqrt(-1)],[1,0,0],[0,1,0]], faces=[[0,1,2]]`, "coordinates must be finite numbers"},
			{`points=points, faces=3`, "faces must be a non-empty list"},
			{`points=points, faces=[]`, "faces must be a non-empty list"},
			{`points=points, faces=[0,1,2]`, "must contain at least 3 point indices"},
			{`points=points, faces=[[0,1]]`, "must contain at least 3 point indices"},
			{`points=points, faces=[[0,1,1.5]]`, "indices must be finite integers"},
			{`points=points, faces=[[0,1,"2"]]`, "indices must be finite integers"},
			{`points=points, faces=[[0,1,sqrt(-1)]]`, "indices must be finite integers"},
			{`points=points, faces=[[0,1,-1]]`, "index -1 out of range"},
			{`points=points, faces=[[0,1,8]]`, "index 8 out of range"},
			{`points=points, faces=[[0,1,1,2]]`, "face contains a repeated vertex"},
			{`points=points, faces=[[0,1,0]]`, "at least 3 distinct vertices"},
			{`points=[[0,0,0],[1,0,0],[2,0,0]], faces=[[0,1,2]]`, "zero or non-finite area"},
			{`points=[[0,0,0],[1,0,0],[2,0,0],[3,0,0]], faces=[[0,1,2,3]]`, "cannot triangulate face"},
			{`points=points, faces=faces, convexity="x"`, "convexity: expected number"},
			{`points=points, faces=faces, triangles=faces`, "duplicate argument"},
		} {
			t.Run(fmt.Sprintf("%s/%s", module, tc.args), func(t *testing.T) {
				prog, err := ParseFile("polyhedron.scad", polyhedronBoxData+module+"("+tc.args+");")
				if err != nil {
					t.Fatal(err)
				}
				_, err = Eval(prog, Hooks{})
				if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "polyhedron.scad:") {
					t.Fatalf("expected positioned error containing %q, got %v", tc.want, err)
				}
			})
		}
	}
}
