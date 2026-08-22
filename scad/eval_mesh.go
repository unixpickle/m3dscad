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

func handleDecimate(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	bound, err := bindArgsDetailed(e, st.Call, []ArgSpec{
		{Name: "triangles_fraction", Aliases: []string{"triangle_fraction"}, Pos: 0},
		{Name: "triangle_count", Pos: -1},
		{Name: "min_det", Pos: 1, Default: Num(model3d.QEFDecimatorDefaultMinDet)},
		{Name: "min_normal_dot_product", Pos: 2, Default: Num(0)},
		{Name: "tikhonov_regularization", Pos: 3, Default: Num(1e-5)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	if bound.Provided["triangles_fraction"] == bound.Provided["triangle_count"] {
		return ShapeRep{}, fmt.Errorf("decimate(): provide exactly one of triangles_fraction or triangle_count")
	}
	minDet, err := argNum(bound.Values, "min_det")
	if err != nil {
		return ShapeRep{}, err
	}
	minNormalDotProduct, err := argNum(bound.Values, "min_normal_dot_product")
	if err != nil {
		return ShapeRep{}, err
	}
	tikhonovRegularization, err := argNum(bound.Values, "tikhonov_regularization")
	if err != nil {
		return ShapeRep{}, err
	}
	if minDet < 0 {
		return ShapeRep{}, fmt.Errorf("decimate(): min_det must be >= 0")
	}
	if minNormalDotProduct < -1 || minNormalDotProduct > 1 {
		return ShapeRep{}, fmt.Errorf("decimate(): min_normal_dot_product must be between -1 and 1")
	}
	if tikhonovRegularization < 0 {
		return ShapeRep{}, fmt.Errorf("decimate(): tikhonov_regularization must be >= 0")
	}
	if childUnion.Kind != ShapeMesh3D {
		return ShapeRep{}, fmt.Errorf("decimate(): requires a 3D mesh")
	}
	if childUnion.M3.NeedsRepair() || len(childUnion.M3.SingularVertices()) != 0 {
		return ShapeRep{}, fmt.Errorf("decimate(): mesh must be manifold")
	}
	triangleCount := 0
	if bound.Provided["triangles_fraction"] {
		fraction, err := argNum(bound.Values, "triangles_fraction")
		if err != nil {
			return ShapeRep{}, err
		}
		if fraction < 0 || fraction > 1 {
			return ShapeRep{}, fmt.Errorf("decimate(): triangles_fraction must be between 0 and 1")
		}
		triangleCount = int(float64(childUnion.M3.NumTriangles()) * fraction)
	} else {
		count, err := argNum(bound.Values, "triangle_count")
		if err != nil {
			return ShapeRep{}, err
		}
		if float64(int(count)) != count {
			return ShapeRep{}, fmt.Errorf("decimate(): triangle_count must be an integer")
		}
		if count < 0 {
			return ShapeRep{}, fmt.Errorf("decimate(): triangle_count must be >= 0")
		}
		triangleCount = int(count)
	}
	return shapeMesh3D(model3d.QEFDecimate(childUnion.M3, triangleCount, &model3d.QEFDecimatorOptions{
		MinDet:                 minDet,
		MinNormalDotProduct:    minNormalDotProduct,
		TikhonovRegularization: tikhonovRegularization,
	})), nil
}
