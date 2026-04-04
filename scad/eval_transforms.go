package scad

import (
	"fmt"
	"math"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleTranslate(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "v", Pos: 0, Default: List([]Value{Num(0), Num(0), Num(0)})},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	vec, err := argVec3(args, "v")
	if err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind.Dimension() {
	case 2:
		xf, err := translateTransform2D(vec)
		if err != nil {
			return ShapeRep{}, err
		}
		return applyTransform2D("translate", *childUnion, xf)
	case 3:
		return applyTransform3D("translate", *childUnion, translateTransform3D(vec))
	default:
		return ShapeRep{}, fmt.Errorf("translate(): unsupported shape kind")
	}
}

func handleScale(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "v", Pos: 0, Default: List([]Value{Num(0), Num(0), Num(0)})},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	vec, err := argVec3(args, "v")
	if err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind.Dimension() {
	case 2:
		return applyTransform2D("scale", *childUnion, scaleTransform2D(vec))
	case 3:
		return applyTransform3D("scale", *childUnion, scaleTransform3D(vec))
	default:
		return ShapeRep{}, fmt.Errorf("scale(): unsupported shape kind")
	}
}

func handleRotate(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	spec, err := parseRotateSpec(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind.Dimension() {
	case 2:
		xf, err := rotateTransform2D(spec)
		if err != nil {
			return ShapeRep{}, err
		}
		return applyTransform2D("rotate", *childUnion, xf)
	case 3:
		xf, err := rotateTransform3D(spec)
		if err != nil {
			return ShapeRep{}, err
		}
		return applyTransform3D("rotate", *childUnion, xf)
	default:
		return ShapeRep{}, fmt.Errorf("rotate(): unsupported shape kind")
	}
}

func handleMirror(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "v", Pos: 0, Required: true},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	vec, err := argVec3(args, "v")
	if err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind.Dimension() {
	case 2:
		xf, err := mirrorTransform2D(vec)
		if err != nil {
			return ShapeRep{}, err
		}
		return applyTransform2D("mirror", *childUnion, xf)
	case 3:
		xf, err := mirrorTransform3D(vec)
		if err != nil {
			return ShapeRep{}, err
		}
		return applyTransform3D("mirror", *childUnion, xf)
	default:
		return ShapeRep{}, fmt.Errorf("mirror(): unsupported shape kind")
	}
}

func translateTransform2D(vec [3]float64) (model2d.Transform, error) {
	if vec[2] != 0 {
		return nil, fmt.Errorf("translate(): z component not supported for 2D shapes")
	}
	return &model2d.Translate{Offset: model2d.XY(vec[0], vec[1])}, nil
}

func translateTransform3D(vec [3]float64) model3d.Transform {
	return &model3d.Translate{Offset: model3d.XYZ(vec[0], vec[1], vec[2])}
}

func scaleTransform2D(vec [3]float64) model2d.Transform {
	scale := model2d.XY(vec[0], vec[1])
	if math.Abs(scale.X) == math.Abs(scale.Y) {
		return &uniformVecScale2D{
			VecScale:  model2d.VecScale{Scale: scale},
			DistScale: math.Abs(scale.X),
		}
	}
	return &model2d.VecScale{Scale: scale}
}

func scaleTransform3D(vec [3]float64) model3d.Transform {
	scale := model3d.XYZ(vec[0], vec[1], vec[2])
	if math.Abs(scale.X) == math.Abs(scale.Y) && math.Abs(scale.Y) == math.Abs(scale.Z) {
		return &uniformVecScale3D{
			VecScale:  model3d.VecScale{Scale: scale},
			DistScale: math.Abs(scale.X),
		}
	}
	return &model3d.VecScale{Scale: scale}
}

func rotateTransform2D(spec rotateSpec) (model2d.Transform, error) {
	angle, err := rotateAngle2D(spec)
	if err != nil {
		return nil, err
	}
	return model2d.Rotation(angle), nil
}

func mirrorTransform2D(vec [3]float64) (model2d.Transform, error) {
	if vec[2] != 0 {
		return nil, fmt.Errorf("mirror(): z component not supported for 2D shapes")
	}
	if vec[0] == 0 && vec[1] == 0 {
		return nil, fmt.Errorf("mirror(): axis must be non-zero")
	}
	return model2d.Mirror(model2d.XY(vec[0], vec[1])), nil
}

func applyTransform2D(opName string, shape ShapeRep, xf model2d.Transform) (ShapeRep, error) {
	switch shape.Kind {
	case ShapeSolid2D:
		return shapeSolid2D(model2d.TransformSolid(xf, shape.S2)), nil
	case ShapeMesh2D:
		return shapeMesh2D(shape.M2.Transform(xf)), nil
	case ShapeSDF2D:
		sdf, err := applySDFTransform2D(opName, shape.SDF2, xf)
		if err != nil {
			return ShapeRep{}, err
		}
		return shapeSDF2D(sdf), nil
	case ShapeMetaball2D:
		mb, err := applyMetaballTransform2D(opName, shape.MB2, xf)
		if err != nil {
			return ShapeRep{}, err
		}
		return ShapeRep{Kind: ShapeMetaball2D, MB2: mb}, nil
	case ShapeHull2D:
		hull, err := applyHullTransform2D(opName, shape.H2, xf)
		if err != nil {
			return ShapeRep{}, err
		}
		return shapeHull2D(hull), nil
	default:
		return ShapeRep{}, fmt.Errorf("%s(): unsupported shape kind", opName)
	}
}

func applyTransform3D(opName string, shape ShapeRep, xf model3d.Transform) (ShapeRep, error) {
	switch shape.Kind {
	case ShapeSolid3D:
		return shapeSolid3D(model3d.TransformSolid(xf, shape.S3)), nil
	case ShapeMesh3D:
		return shapeMesh3D(shape.M3.Transform(xf)), nil
	case ShapeSDF3D:
		sdf, err := applySDFTransform3D(opName, shape.SDF3, xf)
		if err != nil {
			return ShapeRep{}, err
		}
		return shapeSDF3D(sdf), nil
	case ShapeMetaball3D:
		mb, err := applyMetaballTransform3D(opName, shape.MB3, xf)
		if err != nil {
			return ShapeRep{}, err
		}
		return ShapeRep{Kind: ShapeMetaball3D, MB3: mb}, nil
	default:
		return ShapeRep{}, fmt.Errorf("%s(): unsupported shape kind", opName)
	}
}

func applySDFTransform2D(opName string, sdf model2d.SDF, xf model2d.Transform) (model2d.SDF, error) {
	if _, ok := xf.(*model2d.VecScale); ok {
		return nil, fmt.Errorf("%s(): non-uniform scaling not supported for SDFs", opName)
	}
	distXf, ok := xf.(model2d.DistTransform)
	if !ok {
		return nil, fmt.Errorf("%s(): transform not supported for SDFs", opName)
	}
	return model2d.TransformSDF(distXf, sdf), nil
}

func applySDFTransform3D(opName string, sdf model3d.SDF, xf model3d.Transform) (model3d.SDF, error) {
	if _, ok := xf.(*model3d.VecScale); ok {
		return nil, fmt.Errorf("%s(): non-uniform scaling not supported for SDFs", opName)
	}
	distXf, ok := xf.(model3d.DistTransform)
	if !ok {
		return nil, fmt.Errorf("%s(): transform not supported for SDFs", opName)
	}
	return model3d.TransformSDF(distXf, sdf), nil
}

func applyMetaballTransform2D(opName string, mb *Metaball2D, xf model2d.Transform) (*Metaball2D, error) {
	if vecScale, ok := xf.(*model2d.VecScale); ok {
		return mb.Map(func(m model2d.Metaball) model2d.Metaball {
			return model2d.VecScaleMetaball(m, vecScale.Scale)
		}), nil
	}
	distXf, ok := xf.(model2d.DistTransform)
	if !ok {
		return nil, fmt.Errorf("%s(): transform not supported for metaballs", opName)
	}
	return mb.Map(func(m model2d.Metaball) model2d.Metaball {
		return model2d.TransformMetaball(distXf, m)
	}), nil
}

func applyMetaballTransform3D(opName string, mb *Metaball3D, xf model3d.Transform) (*Metaball3D, error) {
	if vecScale, ok := xf.(*model3d.VecScale); ok {
		return mb.Map(func(m model3d.Metaball) model3d.Metaball {
			return model3d.VecScaleMetaball(m, vecScale.Scale)
		}), nil
	}
	distXf, ok := xf.(model3d.DistTransform)
	if !ok {
		return nil, fmt.Errorf("%s(): transform not supported for metaballs", opName)
	}
	return mb.Map(func(m model3d.Metaball) model3d.Metaball {
		return model3d.TransformMetaball(distXf, m)
	}), nil
}

func applyHullTransform2D(opName string, hull *Hull2D, xf model2d.Transform) (*Hull2D, error) {
	if _, ok := xf.(*model2d.VecScale); ok {
		return nil, fmt.Errorf("%s(): non-uniform scaling not supported for hulls", opName)
	}
	distXf, ok := xf.(model2d.DistTransform)
	if !ok {
		return nil, fmt.Errorf("%s(): transform not supported for hulls", opName)
	}
	return hull.Map(func(c *model2d.Circle) *model2d.Circle {
		circle := *c
		circle.Center = xf.Apply(circle.Center)
		circle.Radius = distXf.ApplyDistance(circle.Radius)
		return &circle
	}), nil
}

type uniformVecScale2D struct {
	model2d.VecScale
	DistScale float64
}

func (u *uniformVecScale2D) Inverse() model2d.Transform {
	return &uniformVecScale2D{
		VecScale:  model2d.VecScale{Scale: u.Scale.Recip()},
		DistScale: 1 / u.DistScale,
	}
}

func (u *uniformVecScale2D) ApplyDistance(d float64) float64 {
	return d * u.DistScale
}

type uniformVecScale3D struct {
	model3d.VecScale
	DistScale float64
}

func (u *uniformVecScale3D) Inverse() model3d.Transform {
	return &uniformVecScale3D{
		VecScale:  model3d.VecScale{Scale: u.Scale.Recip()},
		DistScale: 1 / u.DistScale,
	}
}

func (u *uniformVecScale3D) ApplyDistance(d float64) float64 {
	return d * u.DistScale
}

func handleTransform(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	switch childUnion.Kind {
	case ShapeSolid3D:
		min, max, fn, err := parseTransformBoundsArgs(e, st, 3)
		if err != nil {
			return ShapeRep{}, err
		}
		min3 := model3d.XYZ(min[0], min[1], min[2])
		max3 := model3d.XYZ(max[0], max[1], max[2])
		return shapeSolid3D(model3d.CheckedFuncSolid(min3, max3, func(c model3d.Coord3D) bool {
			mapped, err := evalFnCoordMap(e, fn, []float64{c.X, c.Y, c.Z}, 3, false)
			if err != nil || mapped == nil {
				return false
			}
			return childUnion.S3.Contains(model3d.XYZ(mapped[0], mapped[1], mapped[2]))
		})), nil
	case ShapeSolid2D:
		min, max, fn, err := parseTransformBoundsArgs(e, st, 2)
		if err != nil {
			return ShapeRep{}, err
		}
		min2 := model2d.XY(min[0], min[1])
		max2 := model2d.XY(max[0], max[1])
		return shapeSolid2D(model2d.CheckedFuncSolid(min2, max2, func(c model2d.Coord) bool {
			mapped, err := evalFnCoordMap(e, fn, []float64{c.X, c.Y}, 2, false)
			if err != nil || mapped == nil {
				return false
			}
			return childUnion.S2.Contains(model2d.XY(mapped[0], mapped[1]))
		})), nil
	case ShapeSDF3D:
		min, max, fn, err := parseTransformBoundsArgs(e, st, 3)
		if err != nil {
			return ShapeRep{}, err
		}
		min3 := model3d.XYZ(min[0], min[1], min[2])
		max3 := model3d.XYZ(max[0], max[1], max[2])
		return shapeSDF3D(model3d.FuncSDF(min3, max3, func(c model3d.Coord3D) float64 {
			mapped, err := evalFnCoordMap(e, fn, []float64{c.X, c.Y, c.Z}, 3, false)
			if err != nil || mapped == nil {
				return -1
			}
			return childUnion.SDF3.SDF(model3d.XYZ(mapped[0], mapped[1], mapped[2]))
		})), nil
	case ShapeSDF2D:
		min, max, fn, err := parseTransformBoundsArgs(e, st, 2)
		if err != nil {
			return ShapeRep{}, err
		}
		min2 := model2d.XY(min[0], min[1])
		max2 := model2d.XY(max[0], max[1])
		return shapeSDF2D(model2d.FuncSDF(min2, max2, func(c model2d.Coord) float64 {
			mapped, err := evalFnCoordMap(e, fn, []float64{c.X, c.Y}, 2, false)
			if err != nil || mapped == nil {
				return -1
			}
			return childUnion.SDF2.SDF(model2d.XY(mapped[0], mapped[1]))
		})), nil
	case ShapeMesh3D:
		oldMin := childUnion.M3.Min()
		oldMax := childUnion.M3.Max()
		fn, err := parseTransformMeshArgs(
			e, st, 3,
			[]float64{oldMin.X, oldMin.Y, oldMin.Z},
			[]float64{oldMax.X, oldMax.Y, oldMax.Z},
		)
		if err != nil {
			return ShapeRep{}, err
		}
		return shapeMesh3D(childUnion.M3.MapCoords(func(c model3d.Coord3D) model3d.Coord3D {
			mapped, err := evalFnCoordMap(e, fn, []float64{c.X, c.Y, c.Z}, 3, false)
			if err != nil || mapped == nil {
				return c
			}
			return model3d.XYZ(mapped[0], mapped[1], mapped[2])
		})), nil
	case ShapeMesh2D:
		oldMin := childUnion.M2.Min()
		oldMax := childUnion.M2.Max()
		fn, err := parseTransformMeshArgs(
			e, st, 2,
			[]float64{oldMin.X, oldMin.Y},
			[]float64{oldMax.X, oldMax.Y},
		)
		if err != nil {
			return ShapeRep{}, err
		}
		return shapeMesh2D(childUnion.M2.MapCoords(func(c model2d.Coord) model2d.Coord {
			mapped, err := evalFnCoordMap(e, fn, []float64{c.X, c.Y}, 2, false)
			if err != nil || mapped == nil {
				return c
			}
			return model2d.XY(mapped[0], mapped[1])
		})), nil
	default:
		return ShapeRep{}, fmt.Errorf("transform(): unsupported shape kind")
	}
}

type clipSpec struct {
	MinX float64
	MaxX float64
	MinY float64
	MaxY float64
	MinZ float64
	MaxZ float64
}

func handleClip(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	switch childUnion.Kind {
	case ShapeSolid3D:
		spec, err := parseClipSpec(e, st, 3)
		if err != nil {
			return ShapeRep{}, err
		}
		min, max, empty := clipBounds3D(childUnion.S3.Min(), childUnion.S3.Max(), spec)
		if empty {
			emptySolid := model3d.CheckedFuncSolid(min, min, func(model3d.Coord3D) bool { return false })
			return shapeSolid3D(emptySolid), nil
		}
		rect := model3d.NewRect(min, max)
		return shapeSolid3D(model3d.IntersectedSolid{childUnion.S3, rect}), nil
	case ShapeSolid2D:
		spec, err := parseClipSpec(e, st, 2)
		if err != nil {
			return ShapeRep{}, err
		}
		min, max, empty := clipBounds2D(childUnion.S2.Min(), childUnion.S2.Max(), spec)
		if empty {
			emptySolid := model2d.CheckedFuncSolid(min, min, func(model2d.Coord) bool { return false })
			return shapeSolid2D(emptySolid), nil
		}
		rect := model2d.NewRect(min, max)
		return shapeSolid2D(model2d.IntersectedSolid{childUnion.S2, rect}), nil
	case ShapeSDF3D:
		spec, err := parseClipSpec(e, st, 3)
		if err != nil {
			return ShapeRep{}, err
		}
		min, max, empty := clipBounds3D(childUnion.SDF3.Min(), childUnion.SDF3.Max(), spec)
		if empty {
			emptySDF := model3d.FuncSDF(min, min, func(model3d.Coord3D) float64 { return -1 })
			return shapeSDF3D(emptySDF), nil
		}
		clipRect := model3d.NewRect(min, max)
		return shapeSDF3D(sdfIntersect3D([]ShapeRep{
			*childUnion,
			shapeSDF3D(clipRect),
		})), nil
	case ShapeSDF2D:
		spec, err := parseClipSpec(e, st, 2)
		if err != nil {
			return ShapeRep{}, err
		}
		min, max, empty := clipBounds2D(childUnion.SDF2.Min(), childUnion.SDF2.Max(), spec)
		if empty {
			emptySDF := model2d.FuncSDF(min, min, func(model2d.Coord) float64 { return -1 })
			return shapeSDF2D(emptySDF), nil
		}
		clipRect := model2d.NewRect(min, max)
		return shapeSDF2D(sdfIntersect2D([]ShapeRep{
			*childUnion,
			shapeSDF2D(clipRect),
		})), nil
	default:
		return ShapeRep{}, fmt.Errorf("clip(): requires a solid or SDF")
	}
}

func parseClipSpec(e *env, st *CallStmt, dim int) (clipSpec, error) {
	bound, err := bindArgsDetailed(e, st.Call, []ArgSpec{
		{Name: "min_x", Pos: 0, Default: Num(math.Inf(-1))},
		{Name: "max_x", Pos: 1, Default: Num(math.Inf(1))},
		{Name: "min_y", Pos: 2, Default: Num(math.Inf(-1))},
		{Name: "max_y", Pos: 3, Default: Num(math.Inf(1))},
		{Name: "min_z", Pos: 4, Default: Num(math.Inf(-1))},
		{Name: "max_z", Pos: 5, Default: Num(math.Inf(1))},
	})
	if err != nil {
		return clipSpec{}, err
	}
	if dim == 2 && (bound.Provided["min_z"] || bound.Provided["max_z"]) {
		return clipSpec{}, fmt.Errorf("clip(): min_z/max_z are not supported for 2D shapes")
	}
	minX, err := argNum(bound.Values, "min_x")
	if err != nil {
		return clipSpec{}, err
	}
	maxX, err := argNum(bound.Values, "max_x")
	if err != nil {
		return clipSpec{}, err
	}
	minY, err := argNum(bound.Values, "min_y")
	if err != nil {
		return clipSpec{}, err
	}
	maxY, err := argNum(bound.Values, "max_y")
	if err != nil {
		return clipSpec{}, err
	}
	minZ, err := argNum(bound.Values, "min_z")
	if err != nil {
		return clipSpec{}, err
	}
	maxZ, err := argNum(bound.Values, "max_z")
	if err != nil {
		return clipSpec{}, err
	}
	return clipSpec{
		MinX: minX,
		MaxX: maxX,
		MinY: minY,
		MaxY: maxY,
		MinZ: minZ,
		MaxZ: maxZ,
	}, nil
}

func clipBounds2D(min, max model2d.Coord, spec clipSpec) (model2d.Coord, model2d.Coord, bool) {
	min = min.Max(model2d.XY(spec.MinX, spec.MinY))
	max = max.Min(model2d.XY(spec.MaxX, spec.MaxY))
	if min.X > max.X || min.Y > max.Y {
		return min, min, true
	}
	return min, max, false
}

func clipBounds3D(min, max model3d.Coord3D, spec clipSpec) (model3d.Coord3D, model3d.Coord3D, bool) {
	min = min.Max(model3d.XYZ(spec.MinX, spec.MinY, spec.MinZ))
	max = max.Min(model3d.XYZ(spec.MaxX, spec.MaxY, spec.MaxZ))
	if min.X > max.X || min.Y > max.Y || min.Z > max.Z {
		return min, min, true
	}
	return min, max, false
}

func parseTransformBoundsArgs(
	e *env,
	st *CallStmt,
	dim int,
) ([]float64, []float64, *FuncClosure, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "min", Pos: 0, Required: true},
		{Name: "max", Pos: 1, Required: true},
		{Name: "fn", Pos: 2, Required: true},
	})
	if err != nil {
		return nil, nil, nil, err
	}
	min, err := argCoordStrict(args, "min")
	if err != nil {
		return nil, nil, nil, err
	}
	if len(min) != dim {
		return nil, nil, nil, fmt.Errorf("transform(): min must be a %dD vector/list", dim)
	}
	max, err := argCoordStrict(args, "max")
	if err != nil {
		return nil, nil, nil, err
	}
	if len(max) != dim {
		return nil, nil, nil, fmt.Errorf("transform(): max must be a %dD vector/list", dim)
	}
	fn, err := argFunc(args, "fn")
	if err != nil {
		return nil, nil, nil, err
	}
	if err := preflightTransformFn(e, fn, dim, min, max); err != nil {
		return nil, nil, nil, err
	}
	return min, max, fn, nil
}

func parseTransformMeshArgs(
	e *env,
	st *CallStmt,
	dim int,
	min []float64,
	max []float64,
) (*FuncClosure, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "fn", Pos: 0, Required: true},
	})
	if err != nil {
		return nil, err
	}
	fn, err := argFunc(args, "fn")
	if err != nil {
		return nil, err
	}
	if err := preflightTransformFn(e, fn, dim, min, max); err != nil {
		return nil, err
	}
	return fn, nil
}

func preflightTransformFn(e *env, fn *FuncClosure, dim int, min, max []float64) error {
	mid := make([]float64, dim)
	for i := range mid {
		mid[i] = (min[i] + max[i]) / 2
	}
	for _, c := range [][]float64{min, max, mid} {
		if _, err := evalFnCoordMap(e, fn, c, dim, true); err != nil {
			return err
		}
	}
	return nil
}

func evalFnCoordMap(
	e *env,
	fn *FuncClosure,
	coord []float64,
	dim int,
	strict bool,
) ([]float64, error) {
	vec := make([]Value, 0, len(coord))
	for _, x := range coord {
		vec = append(vec, Num(x))
	}
	arg := List(vec)
	v, err := evalClosureCallValues(e, fn, []Value{arg})
	if err != nil {
		if strict {
			return nil, err
		}
		return nil, nil
	}
	out, err := valueCoordStrict(v, dim)
	if err != nil {
		if strict {
			return nil, err
		}
		return nil, nil
	}
	return out, nil
}

func valueCoordStrict(v Value, dim int) ([]float64, error) {
	if v.Kind != ValList {
		return nil, fmt.Errorf("expected vector/list")
	}
	if len(v.List) != dim {
		return nil, fmt.Errorf("expected %dD vector/list", dim)
	}
	out := make([]float64, dim)
	for i := range out {
		n, err := v.List[i].AsNum()
		if err != nil {
			return nil, err
		}
		out[i] = n
	}
	return out, nil
}

type rotateSpec struct {
	AxisAngle bool
	Angles    [3]float64
	Axis      [3]float64
	AngleDeg  float64
}

func parseRotateSpec(e *env, st *CallStmt) (rotateSpec, error) {
	bound, err := bindArgsDetailed(e, st.Call, []ArgSpec{
		{Name: "a", Pos: 0, Default: Value{}},
		{Name: "v", Pos: 1, Default: Value{}},
	})
	if err != nil {
		return rotateSpec{}, err
	}

	aVal := bound.Values["a"]
	aOK := bound.Provided["a"]
	if !aOK {
		return rotateSpec{}, fmt.Errorf("rotate(): missing parameter \"a\"")
	}

	vVal := bound.Values["v"]
	vOK := bound.Provided["v"]

	if vOK {
		if aVal.Kind != ValNum {
			return rotateSpec{}, fmt.Errorf("rotate(): expected numeric angle for \"a\"")
		}
		if vVal.Kind != ValList {
			return rotateSpec{}, fmt.Errorf("rotate(): expected vector for \"v\"")
		}
		axis, err := vVal.AsVec3()
		if err != nil {
			return rotateSpec{}, err
		}
		return rotateSpec{AxisAngle: true, Axis: axis, AngleDeg: aVal.Num}, nil
	}

	if aVal.Kind == ValList {
		angles, err := aVal.AsVec3()
		if err != nil {
			return rotateSpec{}, err
		}
		return rotateSpec{Angles: angles}, nil
	}
	if aVal.Kind == ValNum {
		return rotateSpec{Angles: [3]float64{0, 0, aVal.Num}}, nil
	}
	return rotateSpec{}, fmt.Errorf("rotate(): expected numeric or vector \"a\"")
}

func handleInsetSDF(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	delta, err := parseInsetDelta(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return insetSDF("inset_sdf", childUnion, delta)
}

func handleOutsetSDF(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	delta, err := parseInsetDelta(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return insetSDF("outset_sdf", childUnion, -delta)
}

func rotateAngle2D(spec rotateSpec) (float64, error) {
	if spec.AxisAngle {
		if spec.Axis[0] != 0 || spec.Axis[1] != 0 {
			return 0, fmt.Errorf("rotate(): only Z rotation supported for 2D shapes")
		}
		if spec.Axis[2] == 0 {
			return 0, fmt.Errorf("rotate(): axis must be non-zero")
		}
		sign := 1.0
		if spec.Axis[2] < 0 {
			sign = -1.0
		}
		return sign * spec.AngleDeg * math.Pi / 180, nil
	}
	if spec.Angles[0] != 0 || spec.Angles[1] != 0 {
		return 0, fmt.Errorf("rotate(): only Z rotation supported for 2D shapes")
	}
	return spec.Angles[2] * math.Pi / 180, nil
}

func parseInsetDelta(e *env, st *CallStmt) (float64, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "delta", Pos: 0, Required: true},
	})
	if err != nil {
		return 0, err
	}
	return argNum(args, "delta")
}

func insetSDF(opName string, childUnion *ShapeRep, delta float64) (ShapeRep, error) {
	switch childUnion.Kind {
	case ShapeSDF2D:
		min, max := insetBounds2D(childUnion.SDF2.Min(), childUnion.SDF2.Max(), delta)
		return shapeSDF2D(model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
			return childUnion.SDF2.SDF(c) - delta
		})), nil
	case ShapeSDF3D:
		min, max := insetBounds3D(childUnion.SDF3.Min(), childUnion.SDF3.Max(), delta)
		return shapeSDF3D(model3d.FuncSDF(min, max, func(c model3d.Coord3D) float64 {
			return childUnion.SDF3.SDF(c) - delta
		})), nil
	default:
		return ShapeRep{}, fmt.Errorf("%s(): requires an SDF", opName)
	}
}

func insetBounds2D(min, max model2d.Coord, delta float64) (model2d.Coord, model2d.Coord) {
	min = min.AddScalar(delta)
	max = max.AddScalar(-delta)
	if min.X > max.X {
		mid := (min.X + max.X) / 2
		min.X, max.X = mid, mid
	}
	if min.Y > max.Y {
		mid := (min.Y + max.Y) / 2
		min.Y, max.Y = mid, mid
	}
	return min, max
}

func insetBounds3D(min, max model3d.Coord3D, delta float64) (model3d.Coord3D, model3d.Coord3D) {
	min = min.AddScalar(delta)
	max = max.AddScalar(-delta)
	if min.X > max.X {
		mid := (min.X + max.X) / 2
		min.X, max.X = mid, mid
	}
	if min.Y > max.Y {
		mid := (min.Y + max.Y) / 2
		min.Y, max.Y = mid, mid
	}
	if min.Z > max.Z {
		mid := (min.Z + max.Z) / 2
		min.Z, max.Z = mid, mid
	}
	return min, max
}

func rotateTransform3D(spec rotateSpec) (model3d.Transform, error) {
	if spec.AxisAngle {
		axis := model3d.XYZ(spec.Axis[0], spec.Axis[1], spec.Axis[2])
		norm := axis.Norm()
		if norm == 0 {
			return nil, fmt.Errorf("rotate(): axis must be non-zero")
		}
		axis = axis.Scale(1 / norm)
		return model3d.Rotation(axis, spec.AngleDeg*math.Pi/180), nil
	}
	return model3d.JoinedTransform{
		model3d.Rotation(model3d.XYZ(1, 0, 0), spec.Angles[0]*math.Pi/180),
		model3d.Rotation(model3d.XYZ(0, 1, 0), spec.Angles[1]*math.Pi/180),
		model3d.Rotation(model3d.XYZ(0, 0, 1), spec.Angles[2]*math.Pi/180),
	}, nil
}

func mirrorTransform3D(vec [3]float64) (model3d.Transform, error) {
	axis := model3d.XYZ(vec[0], vec[1], vec[2])
	if axis.Norm() == 0 {
		return nil, fmt.Errorf("mirror(): axis must be non-zero")
	}
	return model3d.Mirror(axis), nil
}
