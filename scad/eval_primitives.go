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

func handlePolygon(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	solid, err := parsePolygonSolid(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(solid), nil
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
	primary, err := polygonPathSolid(points, paths[0], st.pos())
	if err != nil {
		return nil, err
	}
	if len(paths) == 1 {
		return primary, nil
	}
	holes := make([]model2d.Solid, 0, len(paths)-1)
	for _, p := range paths[1:] {
		s, err := polygonPathSolid(points, p, st.pos())
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
		pathMesh, err := polygonPathMesh(points, path, st.pos())
		if err != nil {
			return nil, err
		}
		mesh.AddMesh(pathMesh)
	}
	return mesh, nil
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
	points, err := parsePolygonPoints(args["points"], st.pos())
	if err != nil {
		return nil, nil, err
	}
	if len(points) < 3 {
		return nil, nil, fmt.Errorf("polygon(): need at least 3 points")
	}
	paths, err := parsePolygonPaths(args["paths"], len(points), st.pos())
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

type SolidSDF interface {
	model3d.SDF
	model3d.Solid
}

func parseCylinder(e *env, st *CallStmt) (SolidSDF, error) {
	h, r1, r2, center, err := parseCylinderArgs(e, st)
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

func parseCylinderArgs(e *env, st *CallStmt) (h, r1, r2 float64, center bool, err error) {
	h = 1
	r1 = 1
	r2 = 1
	center = false

	named := map[string]Value{}
	positional := make([]Value, 0, len(st.Call.Args))
	seenNamed := false
	for _, a := range st.Call.Args {
		v, evalErr := evalExpr(e, a.Expr)
		if evalErr != nil {
			err = evalErr
			return
		}
		if a.Name != "" {
			seenNamed = true
			if _, ok := named[a.Name]; ok {
				err = fmt.Errorf("cylinder(): duplicate argument %q", a.Name)
				return
			}
			named[a.Name] = v
			continue
		}
		if seenNamed {
			err = fmt.Errorf("cylinder(): positional args cannot follow named args")
			return
		}
		positional = append(positional, v)
	}
	if len(positional) > 4 {
		err = fmt.Errorf("cylinder(): too many positional args")
		return
	}

	allowed := map[string]bool{
		"h": true, "center": true,
		"r": true, "r1": true, "r2": true,
		"d": true, "d1": true, "d2": true,
	}
	for k := range named {
		if !allowed[k] {
			err = fmt.Errorf("cylinder(): unknown argument %q", k)
			return
		}
	}

	if len(positional) >= 1 {
		h, err = positional[0].AsNum(st.pos())
		if err != nil {
			return
		}
	}
	if len(positional) >= 2 {
		r1, err = positional[1].AsNum(st.pos())
		if err != nil {
			return
		}
	}
	if len(positional) >= 3 {
		r2, err = positional[2].AsNum(st.pos())
		if err != nil {
			return
		}
	}
	if len(positional) >= 4 {
		center, err = positional[3].AsBool(st.pos())
		if err != nil {
			return
		}
	}

	if v, ok := named["h"]; ok {
		h, err = v.AsNum(st.pos())
		if err != nil {
			return
		}
	}
	if v, ok := named["center"]; ok {
		center, err = v.AsBool(st.pos())
		if err != nil {
			return
		}
	}

	usesUniform := false
	usesSpecific := len(positional) >= 2 || len(positional) >= 3

	if _, ok := named["r"]; ok {
		usesUniform = true
	}
	if _, ok := named["d"]; ok {
		usesUniform = true
	}
	if _, ok := named["r1"]; ok {
		usesSpecific = true
	}
	if _, ok := named["r2"]; ok {
		usesSpecific = true
	}
	if _, ok := named["d1"]; ok {
		usesSpecific = true
	}
	if _, ok := named["d2"]; ok {
		usesSpecific = true
	}

	if usesUniform && usesSpecific {
		err = fmt.Errorf("cylinder(): cannot combine r/d with r1/r2/d1/d2")
		return
	}

	if usesUniform {
		if rv, ok := named["r"]; ok {
			r, convErr := rv.AsNum(st.pos())
			if convErr != nil {
				err = convErr
				return
			}
			r1 = r
			r2 = r
		}
		if dv, ok := named["d"]; ok {
			if _, ok := named["r"]; ok {
				err = fmt.Errorf("cylinder(): cannot provide both r and d")
				return
			}
			d, convErr := dv.AsNum(st.pos())
			if convErr != nil {
				err = convErr
				return
			}
			r1 = d / 2
			r2 = d / 2
		}
	} else {
		if rv, ok := named["r1"]; ok {
			r1, err = rv.AsNum(st.pos())
			if err != nil {
				return
			}
		}
		if rv, ok := named["r2"]; ok {
			r2, err = rv.AsNum(st.pos())
			if err != nil {
				return
			}
		}
		if dv, ok := named["d1"]; ok {
			if _, hasR1 := named["r1"]; hasR1 {
				err = fmt.Errorf("cylinder(): cannot provide both r1 and d1")
				return
			}
			d, convErr := dv.AsNum(st.pos())
			if convErr != nil {
				err = convErr
				return
			}
			r1 = d / 2
		}
		if dv, ok := named["d2"]; ok {
			if _, hasR2 := named["r2"]; hasR2 {
				err = fmt.Errorf("cylinder(): cannot provide both r2 and d2")
				return
			}
			d, convErr := dv.AsNum(st.pos())
			if convErr != nil {
				err = convErr
				return
			}
			r2 = d / 2
		}
	}

	if h < 0 {
		err = fmt.Errorf("cylinder(): h must be non-negative")
		return
	}
	if r1 < 0 || r2 < 0 {
		err = fmt.Errorf("cylinder(): radii must be non-negative")
		return
	}
	return
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
	mesh, err := polygonPathMesh(points, path, pos)
	if err != nil {
		return nil, err
	}
	return mesh.Solid(), nil
}

func polygonPathMesh(points []model2d.Coord, path []int, pos Pos) (*model2d.Mesh, error) {
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
	return model2d.NewMeshSegments(segs), nil
}
