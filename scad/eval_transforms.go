package scad

import (
	"fmt"
	"math"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleTranslate(e *env, st *CallStmt, _ []SolidValue, childUnion *SolidValue) (SolidValue, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "v", Pos: 0, Default: List([]Value{Num(0), Num(0), Num(0)})},
	})
	if err != nil {
		return SolidValue{}, err
	}
	vec, err := argVec3(args, "v", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	switch childUnion.Kind {
	case Solid3D:
		xf := &model3d.Translate{Offset: model3d.XYZ(vec[0], vec[1], vec[2])}
		return solid3D(model3d.TransformSolid(xf, childUnion.Solid3)), nil
	case Solid2D:
		if vec[2] != 0 {
			return SolidValue{}, fmt.Errorf("translate(): z component not supported for 2D solids")
		}
		xf := &model2d.Translate{Offset: model2d.XY(vec[0], vec[1])}
		return solid2D(model2d.TransformSolid(xf, childUnion.Solid2)), nil
	default:
		return SolidValue{}, fmt.Errorf("translate(): unsupported solid kind")
	}
}

func handleScale(e *env, st *CallStmt, _ []SolidValue, childUnion *SolidValue) (SolidValue, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "v", Pos: 0, Default: List([]Value{Num(0), Num(0), Num(0)})},
	})
	if err != nil {
		return SolidValue{}, err
	}
	vec, err := argVec3(args, "v", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	switch childUnion.Kind {
	case Solid3D:
		xf := &model3d.VecScale{Scale: model3d.XYZ(vec[0], vec[1], vec[2])}
		return solid3D(model3d.TransformSolid(xf, childUnion.Solid3)), nil
	case Solid2D:
		if vec[2] != 0 {
			return SolidValue{}, fmt.Errorf("scale(): z component not supported for 2D solids")
		}
		xf := &model2d.VecScale{Scale: model2d.XY(vec[0], vec[1])}
		return solid2D(model2d.TransformSolid(xf, childUnion.Solid2)), nil
	default:
		return SolidValue{}, fmt.Errorf("scale(): unsupported solid kind")
	}
}

func handleRotate(e *env, st *CallStmt, _ []SolidValue, childUnion *SolidValue) (SolidValue, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "v", Pos: 0, Default: List([]Value{Num(0), Num(0), Num(0)})},
	})
	if err != nil {
		return SolidValue{}, err
	}
	vec, err := argVec3(args, "v", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	switch childUnion.Kind {
	case Solid3D:
		xf := model3d.JoinedTransform{
			model3d.Rotation(model3d.XYZ(1, 0, 0), vec[0]*math.Pi/180),
			model3d.Rotation(model3d.XYZ(0, 1, 0), vec[1]*math.Pi/180),
			model3d.Rotation(model3d.XYZ(0, 0, 1), vec[2]*math.Pi/180),
		}
		return solid3D(model3d.TransformSolid(xf, childUnion.Solid3)), nil
	case Solid2D:
		if vec[0] != 0 || vec[1] != 0 {
			return SolidValue{}, fmt.Errorf("rotate(): only Z rotation supported for 2D solids")
		}
		xf := model2d.Rotation(vec[2] * math.Pi / 180)
		return solid2D(model2d.TransformSolid(xf, childUnion.Solid2)), nil
	default:
		return SolidValue{}, fmt.Errorf("rotate(): unsupported solid kind")
	}
}
