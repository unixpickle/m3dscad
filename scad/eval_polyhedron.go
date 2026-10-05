package scad

import (
	"fmt"
	"math"

	"github.com/unixpickle/model3d/model3d"
	shapekernel "github.com/unixpickle/webgpu-meshes/shapekernel"
)

func handlePolyhedron(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parsePolyhedronMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	tris := mesh.TriangleSlice()
	model3d.GroupTriangles(tris)
	return shapeSolid3D(
		model3d.NewColliderSolid(model3d.GroupedTrianglesToCollider(tris)),
		asPtr(shapekernel.GroupedTrianglesSolid(e.hooks.Numerics, tris)),
	), nil
}

func handlePolyhedronMesh(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parsePolyhedronMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMesh3D(mesh), nil
}

func handlePolyhedronSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parsePolyhedronMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	tris := mesh.TriangleSlice()
	model3d.GroupTriangles(tris)
	return shapeSDF3D(
		model3d.GroupedTrianglesToSDF(tris),
		asPtr(shapekernel.GroupedTrianglesSDF(e.hooks.Numerics, tris)),
	), nil
}

func parsePolyhedronMesh(e *env, st *CallStmt) (*model3d.Mesh, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "points", Pos: 0, Required: true},
		{Name: "faces", Aliases: []string{"triangles"}, Pos: 1, Required: true},
		{Name: "convexity", Pos: 2, Default: Num(1)},
	})
	if err != nil {
		return nil, err
	}
	// Convexity only affects OpenSCAD's preview, not the geometry.
	if _, err := argNum(args, "convexity"); err != nil {
		return nil, fmt.Errorf("%s(): convexity: %w", st.Call.Name, err)
	}
	pointsArg := args["points"]
	if pointsArg.Kind != ValList || len(pointsArg.List) < 3 {
		return nil, fmt.Errorf("%s(): points must be a list of at least 3 3D points", st.Call.Name)
	}
	points := make([]model3d.Coord3D, len(pointsArg.List))
	for i, v := range pointsArg.List {
		if v.Kind != ValList || len(v.List) != 3 {
			return nil, fmt.Errorf("%s(): points[%d] must be a 3D vector", st.Call.Name, i)
		}
		for _, coord := range v.List {
			if coord.Kind != ValNum || math.IsNaN(coord.Num) || math.IsInf(coord.Num, 0) {
				return nil, fmt.Errorf("%s(): points[%d] coordinates must be finite numbers", st.Call.Name, i)
			}
		}
		points[i] = model3d.XYZ(v.List[0].Num, v.List[1].Num, v.List[2].Num)
	}
	facesArg := args["faces"]
	if facesArg.Kind != ValList || len(facesArg.List) == 0 {
		return nil, fmt.Errorf("%s(): faces must be a non-empty list of index lists", st.Call.Name)
	}
	mesh := model3d.NewMesh()
	for i, face := range facesArg.List {
		if face.Kind != ValList || len(face.List) < 3 {
			return nil, fmt.Errorf("%s(): faces[%d] must contain at least 3 point indices", st.Call.Name, i)
		}
		polygon := make([]model3d.Coord3D, len(face.List))
		for j, v := range face.List {
			if v.Kind != ValNum || math.IsNaN(v.Num) || math.IsInf(v.Num, 0) || math.Trunc(v.Num) != v.Num {
				return nil, fmt.Errorf("%s(): faces[%d] indices must be finite integers", st.Call.Name, i)
			}
			if v.Num < 0 || v.Num >= float64(len(points)) {
				return nil, fmt.Errorf("%s(): faces[%d] index %g out of range", st.Call.Name, i, v.Num)
			}
			polygon[j] = points[int(v.Num)]
		}
		// OpenSCAD also accepts an explicitly repeated closing vertex.
		if polygon[0] == polygon[len(polygon)-1] {
			polygon = polygon[:len(polygon)-1]
		}
		tris, err := triangulatePolyhedronFace(polygon)
		if err != nil {
			return nil, fmt.Errorf("%s(): faces[%d]: %w", st.Call.Name, i, err)
		}
		for _, tri := range tris {
			mesh.Add(tri)
		}
	}
	return mesh, nil
}

// triangulatePolyhedronFace converts model3d's triangulation failures into
// evaluation errors, rather than allowing invalid faces to panic the interpreter.
func triangulatePolyhedronFace(polygon []model3d.Coord3D) (tris []*model3d.Triangle, err error) {
	defer func() {
		if r := recover(); r != nil {
			tris = nil
			err = fmt.Errorf("cannot triangulate face: %v", r)
		}
	}()
	if len(polygon) < 3 {
		return nil, fmt.Errorf("face must have at least 3 distinct vertices")
	}
	seen := map[model3d.Coord3D]bool{}
	for _, p := range polygon {
		if seen[p] {
			return nil, fmt.Errorf("face contains a repeated vertex")
		}
		seen[p] = true
	}
	tris = model3d.TriangulateFace(polygon)
	for _, tri := range tris {
		area := tri.Area()
		if area == 0 || math.IsNaN(area) || math.IsInf(area, 0) {
			return nil, fmt.Errorf("face has zero or non-finite area")
		}
		// OpenSCAD faces are clockwise from outside; model3d uses the
		// opposite winding for outward-facing triangle normals.
		tri[0], tri[2] = tri[2], tri[0]
	}
	return tris, nil
}
