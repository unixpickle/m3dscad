package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	shapekernel "github.com/unixpickle/webgpu-meshes/shapekernel"
)

func handleMarchingSquares(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if childUnion.Kind != ShapeSolid2D {
		return ShapeRep{}, fmt.Errorf("marching_squares(): requires a 2D solid")
	}
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "delta", Pos: 0, Default: Num(0.02)},
		{Name: "subdiv", Pos: 1, Default: Num(8)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	delta, err := argNum(args, "delta")
	if err != nil {
		return ShapeRep{}, err
	}
	subdiv, err := argNum(args, "subdiv")
	if err != nil {
		return ShapeRep{}, err
	}
	if delta <= 0 {
		return ShapeRep{}, fmt.Errorf("marching_squares(): delta must be > 0")
	}
	if subdiv < 1 {
		return ShapeRep{}, fmt.Errorf("marching_squares(): subdiv must be >= 1")
	}
	mesh, err := e.hooks.MarchingSquares(*childUnion, delta, int(subdiv))
	if err != nil {
		return ShapeRep{}, fmt.Errorf("marching_squares(): %w", err)
	}
	return shapeMesh2D(mesh), nil
}

func handleMarchingCubes(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if childUnion.Kind != ShapeSolid3D {
		return ShapeRep{}, fmt.Errorf("marching_cubes(): requires a 3D solid")
	}
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "delta", Pos: 0, Default: Num(0.02)},
		{Name: "subdiv", Pos: 1, Default: Num(8)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	delta, err := argNum(args, "delta")
	if err != nil {
		return ShapeRep{}, err
	}
	subdiv, err := argNum(args, "subdiv")
	if err != nil {
		return ShapeRep{}, err
	}
	if delta <= 0 {
		return ShapeRep{}, fmt.Errorf("marching_cubes(): delta must be > 0")
	}
	if subdiv < 1 {
		return ShapeRep{}, fmt.Errorf("marching_cubes(): subdiv must be >= 1")
	}
	mesh, err := e.hooks.MarchingCubes(*childUnion, delta, int(subdiv))
	if err != nil {
		return ShapeRep{}, fmt.Errorf("marching_cubes(): %w", err)
	}
	return shapeMesh3D(mesh), nil
}

func handleDualContour(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if childUnion.Kind != ShapeSolid3D {
		return ShapeRep{}, fmt.Errorf("dual_contour(): requires a 3D solid")
	}
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "delta", Pos: 0, Default: Num(0.02)},
		{Name: "repair", Pos: 1, Default: Bool(true)},
		{Name: "clip", Pos: 2, Default: Bool(false)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	delta, err := argNum(args, "delta")
	if err != nil {
		return ShapeRep{}, err
	}
	repair, err := argBool(args, "repair")
	if err != nil {
		return ShapeRep{}, err
	}
	clip, err := argBool(args, "clip")
	if err != nil {
		return ShapeRep{}, err
	}
	if delta <= 0 {
		return ShapeRep{}, fmt.Errorf("dual_contour(): delta must be > 0")
	}
	mesh, err := e.hooks.DualContour(*childUnion, delta, repair, clip)
	if err != nil {
		return ShapeRep{}, fmt.Errorf("dual_contour(): %w", err)
	}
	return shapeMesh3D(mesh), nil
}

func handleMeshToSDF(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind {
	case ShapeMesh2D:
		return shapeSDF2D(
			model2d.MeshToSDF(childUnion.M2),
			asPtr(shapekernel.Mesh2DSDF(e.hooks.Numerics, childUnion.M2)),
		), nil
	case ShapeMesh3D:
		tris := childUnion.M3.TriangleSlice()
		model3d.GroupTriangles(tris)
		return shapeSDF3D(
			model3d.GroupedTrianglesToSDF(tris),
			asPtr(shapekernel.GroupedTrianglesSDF(e.hooks.Numerics, tris)),
		), nil
	default:
		return ShapeRep{}, fmt.Errorf("mesh_to_sdf(): requires a mesh")
	}
}

func handleMeshToHull(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
	if childUnion.Kind != ShapeMesh2D {
		return ShapeRep{}, fmt.Errorf("mesh_to_hull(): requires a 2D mesh")
	}
	vertices := childUnion.M2.VertexSlice()
	circles := make([]*model2d.Circle, 0, len(vertices))
	for _, v := range vertices {
		circles = append(circles, &model2d.Circle{Center: v})
	}
	return shapeHull2D(&Hull2D{Circles: circles}), nil
}

func handleSmoothSq(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "iters", Pos: 0, Required: true},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	iters, err := argNum(args, "iters")
	if err != nil {
		return ShapeRep{}, err
	}
	if float64(int(iters)) != iters {
		return ShapeRep{}, fmt.Errorf("smooth_sq(): iters must be an integer")
	}
	if iters < 0 {
		return ShapeRep{}, fmt.Errorf("smooth_sq(): iters must be >= 0")
	}
	if childUnion.Kind != ShapeMesh2D {
		return ShapeRep{}, fmt.Errorf("smooth_sq(): requires a 2D mesh")
	}
	return shapeMesh2D(childUnion.M2.SmoothSq(int(iters))), nil
}
