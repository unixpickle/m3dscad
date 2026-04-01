package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	"github.com/unixpickle/model3d/toolbox3d"
)

func handleFnSolid(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	dim, min, max, fn, err := parseFnSolidArgs(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	if dim == 3 {
		min3 := model3d.XYZ(min[0], min[1], min[2])
		max3 := model3d.XYZ(max[0], max[1], max[2])
		solid := model3d.CheckedFuncSolid(min3, max3, func(c model3d.Coord3D) bool {
			in, err := evalFnSolidBool(e, fn, []float64{c.X, c.Y, c.Z}, false)
			if err != nil {
				return false
			}
			return in
		})
		return shapeSolid3D(solid), nil
	}
	min2 := model2d.XY(min[0], min[1])
	max2 := model2d.XY(max[0], max[1])
	solid := model2d.CheckedFuncSolid(min2, max2, func(c model2d.Coord) bool {
		in, err := evalFnSolidBool(e, fn, []float64{c.X, c.Y}, false)
		if err != nil {
			return false
		}
		return in
	})
	return shapeSolid2D(solid), nil
}

func handleSphere(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	sphere, err := parseSphere(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid3D(sphere), nil
}

func handleSphereMetaball(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	sphere, err := parseSphere(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMetaball3D(sphere), nil
}

func handleSphereSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	sphere, err := parseSphere(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF3D(sphere), nil
}

func handleCube(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	rect, err := parseCube(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid3D(rect), nil
}

func handleCubeMetaball(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	rect, err := parseCube(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMetaball3D(rect), nil
}

func handleCubeSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	rect, err := parseCube(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF3D(rect), nil
}

func handleCylinder(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	cyl, err := parseCylinder(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid3D(cyl), nil
}

func handleCylinderMetaball(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	cyl, err := parseCylinder(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	metaball, ok := cyl.(model3d.Metaball)
	if !ok {
		return ShapeRep{}, fmt.Errorf("cylinder_metaball(): primitive does not implement metaball")
	}
	return shapeMetaball3D(metaball), nil
}

func handleCylinderSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	cyl, err := parseCylinder(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF3D(cyl), nil
}

func handleCapsule(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	capsule, err := parseCapsule(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid3D(capsule), nil
}

func handleCapsuleMetaball(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	capsule, err := parseCapsule(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMetaball3D(capsule), nil
}

func handleCapsuleSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	capsule, err := parseCapsule(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF3D(capsule), nil
}

func handleCircle(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	circle, err := parseCircle(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(circle), nil
}

func handleCircleMetaball(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	circle, err := parseCircle(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMetaball2D(circle), nil
}

func handleCircleSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	circle, err := parseCircle(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF2D(circle), nil
}

func handleCircleHull(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	circle, err := parseCircle(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeHull2D(&Hull2D{Circles: []*model2d.Circle{circle}}), nil
}

func handleHullSolid(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
	if childUnion.Kind != ShapeHull2D {
		return ShapeRep{}, fmt.Errorf("hull_solid(): requires a Hull2D")
	}
	return shapeSolid2D(childUnion.H2.Solid()), nil
}

func handleHullSDF(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
	if childUnion.Kind != ShapeHull2D {
		return ShapeRep{}, fmt.Errorf("hull_sdf(): requires a Hull2D")
	}
	return shapeSDF2D(childUnion.H2.SDF()), nil
}

func handleTeardrop(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	teardrop, err := parseTeardrop(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(teardrop), nil
}

func handleSquare(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	rect, err := parseSquare(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(rect), nil
}

func handleSquareMetaball(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	rect, err := parseSquare(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMetaball2D(rect), nil
}

func handleSquareSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	rect, err := parseSquare(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF2D(rect), nil
}

func handleLineJoin(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "points", Pos: 0, Required: true},
		{Name: "r", Pos: 1, Default: Num(1)},
		{Name: "norm", Pos: 2, Default: String("l2")},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	points, err := parseLineJoinPoints(args["points"])
	if err != nil {
		return ShapeRep{}, err
	}
	if len(points) < 2 {
		return ShapeRep{}, fmt.Errorf("line_join(): need at least 2 points")
	}
	r, err := argNum(args, "r")
	if err != nil {
		return ShapeRep{}, err
	}
	if r < 0 {
		return ShapeRep{}, fmt.Errorf("line_join(): r must be non-negative")
	}
	norm, err := argString(args, "norm")
	if err != nil {
		return ShapeRep{}, err
	}
	segs := make([]model3d.Segment, 0, len(points)-1)
	for i := 0; i < len(points)-1; i++ {
		segs = append(segs, model3d.NewSegment(points[i], points[i+1]))
	}
	switch norm {
	case "l2":
		return shapeSolid3D(toolbox3d.LineJoin(r, segs...)), nil
	case "l1":
		return shapeSolid3D(toolbox3d.L1LineJoin(r, segs...)), nil
	default:
		return ShapeRep{}, fmt.Errorf(`line_join(): norm must be "l2" or "l1"`)
	}
}

func handlePolygon(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	solid, err := parsePolygonSolid(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(solid), nil
}

func handlePolygonHull(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	hull, err := parsePolygonHull(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeHull2D(hull), nil
}

func handlePolygonSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parsePolygonMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF2D(model2d.MeshToSDF(mesh)), nil
}

func handlePolygonMesh(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parsePolygonMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMesh2D(mesh), nil
}

func parsePolygonSolid(e *env, st *CallStmt) (model2d.Solid, error) {
	points, paths, err := parsePolygonData(e, st)
	if err != nil {
		return nil, err
	}
	primary, err := polygonPathSolid(points, paths[0])
	if err != nil {
		return nil, err
	}
	if len(paths) == 1 {
		return primary, nil
	}
	holes := make([]model2d.Solid, 0, len(paths)-1)
	for _, p := range paths[1:] {
		s, err := polygonPathSolid(points, p)
		if err != nil {
			return nil, err
		}
		holes = append(holes, s)
	}
	return model2d.Subtract(primary, model2d.JoinedSolid(holes)), nil
}

func parsePolygonMesh(e *env, st *CallStmt) (*model2d.Mesh, error) {
	points, paths, err := parsePolygonData(e, st)
	if err != nil {
		return nil, err
	}
	mesh := model2d.NewMesh()
	for _, path := range paths {
		pathMesh, err := polygonPathMesh(points, path)
		if err != nil {
			return nil, err
		}
		mesh.AddMesh(pathMesh)
	}
	return mesh, nil
}

func parsePolygonHull(e *env, st *CallStmt) (*Hull2D, error) {
	points, paths, err := parsePolygonData(e, st)
	if err != nil {
		return nil, err
	}
	exterior := paths[0]
	circles := make([]*model2d.Circle, 0, len(exterior))
	for _, idx := range exterior {
		circles = append(circles, &model2d.Circle{Center: points[idx]})
	}
	return &Hull2D{Circles: circles}, nil
}

func parsePolygonData(e *env, st *CallStmt) ([]model2d.Coord, [][]int, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "points", Pos: 0, Required: true},
		{Name: "paths", Pos: 1, Default: Value{}},
		{Name: "convexity", Pos: 2, Default: Num(1)},
	})
	if err != nil {
		return nil, nil, err
	}
	points, err := parsePolygonPoints(args["points"])
	if err != nil {
		return nil, nil, err
	}
	if len(points) < 3 {
		return nil, nil, fmt.Errorf("polygon(): need at least 3 points")
	}
	paths, err := parsePolygonPaths(args["paths"], len(points))
	if err != nil {
		return nil, nil, err
	}
	if len(paths) == 0 {
		paths = [][]int{defaultPolygonPath(len(points))}
	}
	return points, paths, nil
}

func parseSphere(e *env, st *CallStmt) (*model3d.Sphere, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "r", Pos: 0, Default: Num(1.0)},
	})
	if err != nil {
		return nil, err
	}
	r, err := argNum(args, "r")
	if err != nil {
		return nil, err
	}
	return &model3d.Sphere{Radius: r}, nil
}

func parseFnSolidArgs(e *env, st *CallStmt) (int, []float64, []float64, *FuncClosure, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "min", Pos: 0, Required: true},
		{Name: "max", Pos: 1, Required: true},
		{Name: "fn", Pos: 2, Required: true},
	})
	if err != nil {
		return 0, nil, nil, nil, err
	}
	min, err := argCoordStrict(args, "min")
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if len(min) != 2 && len(min) != 3 {
		return 0, nil, nil, nil, fmt.Errorf("min must be a 2D or 3D vector/list")
	}
	max, err := argCoordStrict(args, "max")
	if err != nil {
		return 0, nil, nil, nil, err
	}
	if len(max) != len(min) {
		return 0, nil, nil, nil, fmt.Errorf("max must have the same dimension as min")
	}
	fn, err := argFunc(args, "fn")
	if err != nil {
		return 0, nil, nil, nil, err
	}
	mid := make([]float64, len(min))
	for i := range mid {
		mid[i] = (min[i] + max[i]) / 2
	}
	for _, c := range [][]float64{min, max, mid} {
		if _, err := evalFnSolidBool(e, fn, c, true); err != nil {
			return 0, nil, nil, nil, err
		}
	}
	return len(min), min, max, fn, nil
}

func argFunc(args map[string]Value, name string) (*FuncClosure, error) {
	v, ok := args[name]
	if !ok {
		return nil, fmt.Errorf("missing parameter %q", name)
	}
	if v.Kind != ValFunc || v.Func == nil {
		return nil, fmt.Errorf("expected function")
	}
	return v.Func, nil
}

func argCoordStrict(args map[string]Value, name string) ([]float64, error) {
	v, ok := args[name]
	if !ok {
		return nil, fmt.Errorf("missing parameter %q", name)
	}
	if v.Kind != ValList {
		return nil, fmt.Errorf("expected vector/list")
	}
	out := make([]float64, len(v.List))
	for i := range out {
		n, err := v.List[i].AsNum()
		if err != nil {
			return nil, err
		}
		out[i] = n
	}
	return out, nil
}

func evalFnSolidBool(e *env, fn *FuncClosure, coord []float64, strict bool) (bool, error) {
	vec := make([]Value, 0, len(coord))
	for _, x := range coord {
		vec = append(vec, Num(x))
	}
	arg := List(vec)
	v, err := evalClosureCallValues(e, fn, []Value{arg})
	if err != nil {
		return false, err
	}
	b, err := v.AsBool()
	if err != nil {
		if strict {
			return false, err
		}
		return false, nil
	}
	return b, nil
}

func parseCube(e *env, st *CallStmt) (*model3d.Rect, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "size", Pos: 0, Default: Num(1)},
		{Name: "center", Pos: 1, Default: Bool(false)},
	})
	if err != nil {
		return nil, err
	}
	sizeV, ok := args["size"]
	if !ok {
		return nil, fmt.Errorf("missing parameter \"size\"")
	}
	vec, err := sizeV.AsVec3()
	if err != nil {
		return nil, err
	}
	center, err := argBool(args, "center")
	if err != nil {
		return nil, err
	}
	min := [3]float64{0, 0, 0}
	max := vec
	if center {
		min = [3]float64{-vec[0] / 2, -vec[1] / 2, -vec[2] / 2}
		max = [3]float64{vec[0] / 2, vec[1] / 2, vec[2] / 2}
	}
	return model3d.NewRect(
		model3d.XYZ(min[0], min[1], min[2]),
		model3d.XYZ(max[0], max[1], max[2]),
	), nil
}

type SolidSDF interface {
	model3d.SDF
	model3d.Solid
	model3d.Metaball
}

func parseCylinder(e *env, st *CallStmt) (SolidSDF, error) {
	bound, err := bindArgsDetailed(e, st.Call, []ArgSpec{
		{Name: "h", Pos: 0, Default: Num(1)},
		{Name: "r1", Pos: 1, Default: Num(1)},
		{Name: "r2", Pos: 2, Default: Num(1)},
		{Name: "center", Pos: 3, Default: Bool(false)},
		{Name: "r", Pos: -1, Default: Value{}},
		{Name: "d", Pos: -1, Default: Value{}},
		{Name: "d1", Pos: -1, Default: Value{}},
		{Name: "d2", Pos: -1, Default: Value{}},
	})
	if err != nil {
		return nil, err
	}
	h, err := argNum(bound.Values, "h")
	if err != nil {
		return nil, err
	}
	r1, err := argNum(bound.Values, "r1")
	if err != nil {
		return nil, err
	}
	r2, err := argNum(bound.Values, "r2")
	if err != nil {
		return nil, err
	}
	center, err := argBool(bound.Values, "center")
	if err != nil {
		return nil, err
	}
	usesUniform := bound.NamedProvided["r"] || bound.NamedProvided["d"]
	usesSpecific := bound.Provided["r1"] || bound.Provided["r2"] ||
		bound.NamedProvided["d1"] || bound.NamedProvided["d2"]
	if usesUniform && usesSpecific {
		return nil, fmt.Errorf("cylinder(): cannot combine r/d with r1/r2/d1/d2")
	}

	if usesUniform {
		if bound.NamedProvided["r"] && bound.NamedProvided["d"] {
			return nil, fmt.Errorf("cylinder(): cannot provide both r and d")
		}
		if bound.NamedProvided["r"] {
			r, err := argNum(bound.Values, "r")
			if err != nil {
				return nil, err
			}
			r1 = r
			r2 = r
		}
		if bound.NamedProvided["d"] {
			d, err := argNum(bound.Values, "d")
			if err != nil {
				return nil, err
			}
			r1 = d / 2
			r2 = d / 2
		}
	} else {
		if bound.NamedProvided["d1"] {
			if bound.NamedProvided["r1"] {
				return nil, fmt.Errorf("cylinder(): cannot provide both r1 and d1")
			}
			d1, err := argNum(bound.Values, "d1")
			if err != nil {
				return nil, err
			}
			r1 = d1 / 2
		}
		if bound.NamedProvided["d2"] {
			if bound.NamedProvided["r2"] {
				return nil, fmt.Errorf("cylinder(): cannot provide both r2 and d2")
			}
			d2, err := argNum(bound.Values, "d2")
			if err != nil {
				return nil, err
			}
			r2 = d2 / 2
		}
	}

	if h < 0 {
		return nil, fmt.Errorf("cylinder(): h must be non-negative")
	}
	if r1 < 0 || r2 < 0 {
		return nil, fmt.Errorf("cylinder(): radii must be non-negative")
	}

	z0 := 0.0
	z1 := h
	if center {
		z0 = -h / 2
		z1 = h / 2
	}
	p1 := model3d.XYZ(0, 0, z0)
	p2 := model3d.XYZ(0, 0, z1)
	if r1 == r2 {
		return &model3d.Cylinder{
			P1:     p1,
			P2:     p2,
			Radius: r1,
		}, nil
	}
	if r1 == 0 {
		return &model3d.Cone{
			Tip:    p1,
			Base:   p2,
			Radius: r2,
		}, nil
	}
	if r2 == 0 {
		return &model3d.Cone{
			Tip:    p2,
			Base:   p1,
			Radius: r1,
		}, nil
	}
	return &model3d.ConeSlice{
		P1: p1,
		P2: p2,
		R1: r1,
		R2: r2,
	}, nil
}

func parseCapsule(e *env, st *CallStmt) (*model3d.Capsule, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "h", Pos: 0, Default: Num(1.0)},
		{Name: "r", Pos: 1, Default: Num(1.0)},
		{Name: "center", Pos: 2, Default: Bool(false)},
	})
	if err != nil {
		return nil, err
	}
	h, err := argNum(args, "h")
	if err != nil {
		return nil, err
	}
	if h < 0 {
		return nil, fmt.Errorf("capsule(): h must be non-negative")
	}
	r, err := argNum(args, "r")
	if err != nil {
		return nil, err
	}
	if r < 0 {
		return nil, fmt.Errorf("capsule(): r must be non-negative")
	}
	center, err := argBool(args, "center")
	if err != nil {
		return nil, err
	}
	z0 := 0.0
	z1 := h
	if center {
		z0 = -h / 2
		z1 = h / 2
	}
	p1 := model3d.XYZ(0, 0, z0)
	p2 := model3d.XYZ(0, 0, z1)
	return &model3d.Capsule{
		P1:     p1,
		P2:     p2,
		Radius: r,
	}, nil
}

func parseCircle(e *env, st *CallStmt) (*model2d.Circle, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "r", Pos: 0, Default: Num(1.0)},
	})
	if err != nil {
		return nil, err
	}
	r, err := argNum(args, "r")
	if err != nil {
		return nil, err
	}
	return &model2d.Circle{Radius: r}, nil
}

func parseTeardrop(e *env, st *CallStmt) (*toolbox3d.Teardrop2D, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "radius", Aliases: []string{"r"}, Pos: 0, Default: Num(1.0)},
	})
	if err != nil {
		return nil, err
	}
	r, err := argNum(args, "radius")
	if err != nil {
		return nil, err
	}
	return &toolbox3d.Teardrop2D{
		Center:    model2d.Coord{},
		Radius:    r,
		Direction: model2d.Y(1),
	}, nil
}

func parseSquare(e *env, st *CallStmt) (*model2d.Rect, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "size", Pos: 0, Default: Num(1)},
		{Name: "center", Pos: 1, Default: Bool(false)},
	})
	if err != nil {
		return nil, err
	}
	sizeV, ok := args["size"]
	if !ok {
		return nil, fmt.Errorf("missing parameter \"size\"")
	}
	vec, err := sizeV.AsVec2()
	if err != nil {
		return nil, err
	}
	center, err := argBool(args, "center")
	if err != nil {
		return nil, err
	}
	min := [2]float64{0, 0}
	max := vec
	if center {
		min = [2]float64{-vec[0] / 2, -vec[1] / 2}
		max = [2]float64{vec[0] / 2, vec[1] / 2}
	}
	return model2d.NewRect(
		model2d.XY(min[0], min[1]),
		model2d.XY(max[0], max[1]),
	), nil
}

func parsePolygonPoints(val Value) ([]model2d.Coord, error) {
	if val.Kind != ValList {
		return nil, fmt.Errorf("polygon(): points must be a list")
	}
	points := make([]model2d.Coord, 0, len(val.List))
	for _, v := range val.List {
		if v.Kind != ValList {
			return nil, fmt.Errorf("polygon(): points must be a list of [x, y] pairs")
		}
		vec, err := v.AsVec2()
		if err != nil {
			return nil, err
		}
		points = append(points, model2d.XY(vec[0], vec[1]))
	}
	return points, nil
}

func parseLineJoinPoints(val Value) ([]model3d.Coord3D, error) {
	if val.Kind != ValList {
		return nil, fmt.Errorf("line_join(): points must be a list")
	}
	points := make([]model3d.Coord3D, 0, len(val.List))
	for _, v := range val.List {
		if v.Kind != ValList || len(v.List) != 3 {
			return nil, fmt.Errorf("line_join(): points must be a list of [x, y, z] vectors")
		}
		xyz := [3]float64{}
		for i := range xyz {
			n, err := v.List[i].AsNum()
			if err != nil {
				return nil, err
			}
			xyz[i] = n
		}
		points = append(points, model3d.XYZ(xyz[0], xyz[1], xyz[2]))
	}
	return points, nil
}

func parsePolygonPaths(val Value, numPoints int) ([][]int, error) {
	if val.Kind == ValNull {
		return nil, nil
	}
	if val.Kind != ValList {
		return nil, fmt.Errorf("polygon(): paths must be a list")
	}
	if len(val.List) == 0 {
		return nil, nil
	}
	if val.List[0].Kind != ValList {
		path, err := parsePolygonPath(val.List, numPoints)
		if err != nil {
			return nil, err
		}
		return [][]int{path}, nil
	}
	paths := make([][]int, 0, len(val.List))
	for _, p := range val.List {
		if p.Kind != ValList {
			return nil, fmt.Errorf("polygon(): paths must be a list of lists")
		}
		path, err := parsePolygonPath(p.List, numPoints)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func parsePolygonPath(vals []Value, numPoints int) ([]int, error) {
	if len(vals) < 3 {
		return nil, fmt.Errorf("polygon(): path must have at least 3 points")
	}
	path := make([]int, 0, len(vals))
	for _, v := range vals {
		if v.Kind != ValNum {
			return nil, fmt.Errorf("polygon(): path indices must be numbers")
		}
		idx := int(v.Num)
		if float64(idx) != v.Num {
			return nil, fmt.Errorf("polygon(): path indices must be integers")
		}
		if idx < 0 || idx >= numPoints {
			return nil, fmt.Errorf("polygon(): path index %d out of range", idx)
		}
		path = append(path, idx)
	}
	return path, nil
}

func defaultPolygonPath(n int) []int {
	path := make([]int, n)
	for i := range path {
		path[i] = i
	}
	return path
}

func polygonPathSolid(points []model2d.Coord, path []int) (model2d.Solid, error) {
	mesh, err := polygonPathMesh(points, path)
	if err != nil {
		return nil, err
	}
	return mesh.Solid(), nil
}

func polygonPathMesh(points []model2d.Coord, path []int) (*model2d.Mesh, error) {
	if len(path) < 3 {
		return nil, fmt.Errorf("polygon(): path must have at least 3 points")
	}
	segs := make([]*model2d.Segment, 0, len(path))
	for i, idx := range path {
		next := path[(i+1)%len(path)]
		a := points[idx]
		b := points[next]
		segs = append(segs, &model2d.Segment{a, b})
	}
	return model2d.NewMeshSegments(segs), nil
}
