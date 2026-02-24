package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
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
	delta, err := argNum(args, "delta", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	subdiv, err := argNum(args, "subdiv", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	if delta <= 0 {
		return ShapeRep{}, fmt.Errorf("marching_squares(): delta must be > 0")
	}
	if subdiv < 1 {
		return ShapeRep{}, fmt.Errorf("marching_squares(): subdiv must be >= 1")
	}
	mesh := model2d.MarchingSquaresSearch(childUnion.S2, delta, int(subdiv))
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
	delta, err := argNum(args, "delta", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	subdiv, err := argNum(args, "subdiv", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	if delta <= 0 {
		return ShapeRep{}, fmt.Errorf("marching_cubes(): delta must be > 0")
	}
	if subdiv < 1 {
		return ShapeRep{}, fmt.Errorf("marching_cubes(): subdiv must be >= 1")
	}
	mesh := model3d.MarchingCubesSearch(childUnion.S3, delta, int(subdiv))
	return shapeMesh3D(mesh), nil
}

func handleDualContour(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if childUnion.Kind != ShapeSolid3D {
		return ShapeRep{}, fmt.Errorf("dual_contour(): requires a 3D solid")
	}
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "delta", Pos: 0, Default: Num(0.02)},
		{Name: "repair", Pos: 1, Default: Bool(false)},
		{Name: "clip", Pos: 2, Default: Bool(false)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	delta, err := argNum(args, "delta", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	repair, err := argBool(args, "repair", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	clip, err := argBool(args, "clip", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	if delta <= 0 {
		return ShapeRep{}, fmt.Errorf("dual_contour(): delta must be > 0")
	}
	mesh := model3d.DualContour(childUnion.S3, delta, repair, clip)
	return shapeMesh3D(mesh), nil
}

func handleMeshToSDF(_ *env, _ *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	switch childUnion.Kind {
	case ShapeMesh2D:
		return shapeSDF2D(model2d.MeshToSDF(childUnion.M2)), nil
	case ShapeMesh3D:
		return shapeSDF3D(model3d.MeshToSDF(childUnion.M3)), nil
	default:
		return ShapeRep{}, fmt.Errorf("mesh_to_sdf(): requires a mesh")
	}
}
