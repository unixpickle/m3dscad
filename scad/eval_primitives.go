package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleSphere(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	sphere, err := parseSphere(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid3D(sphere), nil
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

func handleCylinderSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	cyl, err := parseCylinder(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF3D(cyl), nil
}

func handleCircle(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	circle, err := parseCircle(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(circle), nil
}

func handleCircleSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	circle, err := parseCircle(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF2D(circle), nil
}

func handleSquare(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	rect, err := parseSquare(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(rect), nil
}

func handleSquareSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	rect, err := parseSquare(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF2D(rect), nil
}

func handlePolygon(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "points", Pos: 0, Required: true},
		{Name: "paths", Pos: 1, Default: Value{}},
		{Name: "convexity", Pos: 2, Default: Num(1)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	points, err := parsePolygonPoints(args["points"], st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	if len(points) < 3 {
		return ShapeRep{}, fmt.Errorf("polygon(): need at least 3 points")
	}
	paths, err := parsePolygonPaths(args["paths"], len(points), st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	if len(paths) == 0 {
		paths = [][]int{defaultPolygonPath(len(points))}
	}
	primary, err := polygonPathSolid(points, paths[0], st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	if len(paths) == 1 {
		return shapeSolid2D(primary), nil
	}
	holes := make([]model2d.Solid, 0, len(paths)-1)
	for _, p := range paths[1:] {
		s, err := polygonPathSolid(points, p, st.pos())
		if err != nil {
			return ShapeRep{}, err
		}
		holes = append(holes, s)
	}
	return shapeSolid2D(model2d.Subtract(primary, model2d.JoinedSolid(holes))), nil
}

func parseSphere(e *env, st *CallStmt) (*model3d.Sphere, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "r", Pos: 0, Default: Num(1.0)},
	})
	if err != nil {
		return nil, err
	}
	r, err := argNum(args, "r", st.pos())
	if err != nil {
		return nil, err
	}
	return &model3d.Sphere{Radius: r}, nil
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
	vec, err := sizeV.AsVec3(st.pos())
	if err != nil {
		return nil, err
	}
	center, err := argBool(args, "center", st.pos())
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

func parseCylinder(e *env, st *CallStmt) (*model3d.Cylinder, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "h", Pos: 0, Default: Num(1.0)},
		{Name: "r", Pos: 1, Default: Num(1.0)},
		{Name: "center", Pos: 2, Default: Bool(false)},
	})
	if err != nil {
		return nil, err
	}
	h, err := argNum(args, "h", st.pos())
	if err != nil {
		return nil, err
	}
	r, err := argNum(args, "r", st.pos())
	if err != nil {
		return nil, err
	}
	center, err := argBool(args, "center", st.pos())
	if err != nil {
		return nil, err
	}
	z0 := 0.0
	z1 := h
	if center {
		z0 = -h / 2
		z1 = h / 2
	}
	return &model3d.Cylinder{
		P1:     model3d.XYZ(0, 0, z0),
		P2:     model3d.XYZ(0, 0, z1),
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
	r, err := argNum(args, "r", st.pos())
	if err != nil {
		return nil, err
	}
	return &model2d.Circle{Radius: r}, nil
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
	vec, err := sizeV.AsVec2(st.pos())
	if err != nil {
		return nil, err
	}
	center, err := argBool(args, "center", st.pos())
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

func parsePolygonPoints(val Value, pos Pos) ([]model2d.Coord, error) {
	if val.Kind != ValList {
		return nil, fmt.Errorf("%v: polygon(): points must be a list", pos)
	}
	points := make([]model2d.Coord, 0, len(val.List))
	for _, v := range val.List {
		if v.Kind != ValList {
			return nil, fmt.Errorf("%v: polygon(): points must be a list of [x, y] pairs", pos)
		}
		vec, err := v.AsVec2(pos)
		if err != nil {
			return nil, err
		}
		points = append(points, model2d.XY(vec[0], vec[1]))
	}
	return points, nil
}

func parsePolygonPaths(val Value, numPoints int, pos Pos) ([][]int, error) {
	if val.Kind == ValNull {
		return nil, nil
	}
	if val.Kind != ValList {
		return nil, fmt.Errorf("%v: polygon(): paths must be a list", pos)
	}
	if len(val.List) == 0 {
		return nil, nil
	}
	if val.List[0].Kind != ValList {
		path, err := parsePolygonPath(val.List, numPoints, pos)
		if err != nil {
			return nil, err
		}
		return [][]int{path}, nil
	}
	paths := make([][]int, 0, len(val.List))
	for _, p := range val.List {
		if p.Kind != ValList {
			return nil, fmt.Errorf("%v: polygon(): paths must be a list of lists", pos)
		}
		path, err := parsePolygonPath(p.List, numPoints, pos)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func parsePolygonPath(vals []Value, numPoints int, pos Pos) ([]int, error) {
	if len(vals) < 3 {
		return nil, fmt.Errorf("%v: polygon(): path must have at least 3 points", pos)
	}
	path := make([]int, 0, len(vals))
	for _, v := range vals {
		if v.Kind != ValNum {
			return nil, fmt.Errorf("%v: polygon(): path indices must be numbers", pos)
		}
		idx := int(v.Num)
		if float64(idx) != v.Num {
			return nil, fmt.Errorf("%v: polygon(): path indices must be integers", pos)
		}
		if idx < 0 || idx >= numPoints {
			return nil, fmt.Errorf("%v: polygon(): path index %d out of range", pos, idx)
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

func polygonPathSolid(points []model2d.Coord, path []int, pos Pos) (model2d.Solid, error) {
	if len(path) < 3 {
		return nil, fmt.Errorf("%v: polygon(): path must have at least 3 points", pos)
	}
	segs := make([]*model2d.Segment, 0, len(path))
	for i, idx := range path {
		next := path[(i+1)%len(path)]
		a := points[idx]
		b := points[next]
		segs = append(segs, &model2d.Segment{a, b})
	}
	mesh := model2d.NewMeshSegments(segs)
	return mesh.Solid(), nil
}
