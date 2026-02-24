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
	vec, err := argVec3(args, "v", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind {
	case ShapeSolid3D:
		xf := &model3d.Translate{Offset: model3d.XYZ(vec[0], vec[1], vec[2])}
		return shapeSolid3D(model3d.TransformSolid(xf, childUnion.S3)), nil
	case ShapeMesh3D:
		xf := &model3d.Translate{Offset: model3d.XYZ(vec[0], vec[1], vec[2])}
		return shapeMesh3D(childUnion.M3.Transform(xf)), nil
	case ShapeSDF3D:
		xf := &model3d.Translate{Offset: model3d.XYZ(vec[0], vec[1], vec[2])}
		return shapeSDF3D(model3d.TransformSDF(xf, childUnion.SDF3)), nil
	case ShapeSolid2D:
		if vec[2] != 0 {
			return ShapeRep{}, fmt.Errorf("translate(): z component not supported for 2D shapes")
		}
		xf := &model2d.Translate{Offset: model2d.XY(vec[0], vec[1])}
		return shapeSolid2D(model2d.TransformSolid(xf, childUnion.S2)), nil
	case ShapeMesh2D:
		if vec[2] != 0 {
			return ShapeRep{}, fmt.Errorf("translate(): z component not supported for 2D shapes")
		}
		xf := &model2d.Translate{Offset: model2d.XY(vec[0], vec[1])}
		return shapeMesh2D(childUnion.M2.Transform(xf)), nil
	case ShapeSDF2D:
		if vec[2] != 0 {
			return ShapeRep{}, fmt.Errorf("translate(): z component not supported for 2D shapes")
		}
		xf := &model2d.Translate{Offset: model2d.XY(vec[0], vec[1])}
		return shapeSDF2D(model2d.TransformSDF(xf, childUnion.SDF2)), nil
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
	vec, err := argVec3(args, "v", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind {
	case ShapeSolid3D:
		xf := &model3d.VecScale{Scale: model3d.XYZ(vec[0], vec[1], vec[2])}
		return shapeSolid3D(model3d.TransformSolid(xf, childUnion.S3)), nil
	case ShapeMesh3D:
		xf := &model3d.VecScale{Scale: model3d.XYZ(vec[0], vec[1], vec[2])}
		return shapeMesh3D(childUnion.M3.Transform(xf)), nil
	case ShapeSDF3D:
		if vec[0] != vec[1] || vec[1] != vec[2] {
			return ShapeRep{}, fmt.Errorf("scale(): non-uniform scaling not supported for SDFs")
		}
		xf := &model3d.Scale{Scale: vec[0]}
		return shapeSDF3D(model3d.TransformSDF(xf, childUnion.SDF3)), nil
	case ShapeSolid2D:
		if vec[2] != 0 {
			return ShapeRep{}, fmt.Errorf("scale(): z component not supported for 2D shapes")
		}
		xf := &model2d.VecScale{Scale: model2d.XY(vec[0], vec[1])}
		return shapeSolid2D(model2d.TransformSolid(xf, childUnion.S2)), nil
	case ShapeMesh2D:
		if vec[2] != 0 {
			return ShapeRep{}, fmt.Errorf("scale(): z component not supported for 2D shapes")
		}
		xf := &model2d.VecScale{Scale: model2d.XY(vec[0], vec[1])}
		return shapeMesh2D(childUnion.M2.Transform(xf)), nil
	case ShapeSDF2D:
		if vec[2] != 0 {
			return ShapeRep{}, fmt.Errorf("scale(): z component not supported for 2D shapes")
		}
		if vec[0] != vec[1] {
			return ShapeRep{}, fmt.Errorf("scale(): non-uniform scaling not supported for SDFs")
		}
		xf := &model2d.Scale{Scale: vec[0]}
		return shapeSDF2D(model2d.TransformSDF(xf, childUnion.SDF2)), nil
	default:
		return ShapeRep{}, fmt.Errorf("scale(): unsupported shape kind")
	}
}

func handleRotate(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	spec, err := parseRotateSpec(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind {
	case ShapeSolid3D:
		xf, err := rotateTransform3D(spec)
		if err != nil {
			return ShapeRep{}, err
		}
		return shapeSolid3D(model3d.TransformSolid(xf, childUnion.S3)), nil
	case ShapeMesh3D:
		xf, err := rotateTransform3D(spec)
		if err != nil {
			return ShapeRep{}, err
		}
		return shapeMesh3D(childUnion.M3.Transform(xf)), nil
	case ShapeSDF3D:
		xf, err := rotateTransform3D(spec)
		if err != nil {
			return ShapeRep{}, err
		}
		return shapeSDF3D(model3d.TransformSDF(xf, childUnion.SDF3)), nil
	case ShapeSolid2D:
		angle, err := rotateAngle2D(spec)
		if err != nil {
			return ShapeRep{}, err
		}
		xf := model2d.Rotation(angle)
		return shapeSolid2D(model2d.TransformSolid(xf, childUnion.S2)), nil
	case ShapeMesh2D:
		angle, err := rotateAngle2D(spec)
		if err != nil {
			return ShapeRep{}, err
		}
		xf := model2d.Rotation(angle)
		return shapeMesh2D(childUnion.M2.Transform(xf)), nil
	case ShapeSDF2D:
		angle, err := rotateAngle2D(spec)
		if err != nil {
			return ShapeRep{}, err
		}
		xf := model2d.Rotation(angle)
		return shapeSDF2D(model2d.TransformSDF(xf, childUnion.SDF2)), nil
	default:
		return ShapeRep{}, fmt.Errorf("rotate(): unsupported shape kind")
	}
}

type rotateSpec struct {
	AxisAngle bool
	Angles    [3]float64
	Axis      [3]float64
	AngleDeg  float64
}

func parseRotateSpec(e *env, st *CallStmt) (rotateSpec, error) {
	named := map[string]Value{}
	positional := make([]Value, 0, len(st.Call.Args))
	for _, a := range st.Call.Args {
		v, err := evalExpr(e, a.Expr)
		if err != nil {
			return rotateSpec{}, err
		}
		if a.Name != "" {
			named[a.Name] = v
		} else {
			positional = append(positional, v)
		}
	}

	aVal, aOK := named["a"]
	if !aOK && len(positional) > 0 {
		aVal = positional[0]
		aOK = true
	}
	if !aOK {
		return rotateSpec{}, fmt.Errorf("rotate(): missing parameter \"a\"")
	}

	vVal, vOK := named["v"]
	if !vOK && len(positional) > 1 {
		vVal = positional[1]
		vOK = true
	}

	if vOK {
		if aVal.Kind != ValNum {
			return rotateSpec{}, fmt.Errorf("rotate(): expected numeric angle for \"a\"")
		}
		if vVal.Kind != ValList {
			return rotateSpec{}, fmt.Errorf("rotate(): expected vector for \"v\"")
		}
		axis, err := vVal.AsVec3(st.pos())
		if err != nil {
			return rotateSpec{}, err
		}
		return rotateSpec{AxisAngle: true, Axis: axis, AngleDeg: aVal.Num}, nil
	}

	if aVal.Kind == ValList {
		angles, err := aVal.AsVec3(st.pos())
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
	return argNum(args, "delta", st.pos())
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

func rotateTransform3D(spec rotateSpec) (model3d.DistTransform, error) {
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
