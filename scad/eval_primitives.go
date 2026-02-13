package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleSphere(e *env, st *CallStmt, _ []SolidValue, _ *SolidValue) (SolidValue, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "r", Pos: 0, Default: Num(1.0)},
	})
	if err != nil {
		return SolidValue{}, err
	}
	r, err := argNum(args, "r", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	return solid3D(&model3d.Sphere{Radius: r}), nil
}

func handleCube(e *env, st *CallStmt, _ []SolidValue, _ *SolidValue) (SolidValue, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "size", Pos: 0, Default: Num(1)},
		{Name: "center", Pos: 1, Default: Bool(false)},
	})
	if err != nil {
		return SolidValue{}, err
	}
	sizeV, ok := args["size"]
	if !ok {
		return SolidValue{}, fmt.Errorf("missing parameter \"size\"")
	}
	vec, err := sizeV.AsVec3(st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	center, err := argBool(args, "center", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	min := [3]float64{0, 0, 0}
	max := vec
	if center {
		min = [3]float64{-vec[0] / 2, -vec[1] / 2, -vec[2] / 2}
		max = [3]float64{vec[0] / 2, vec[1] / 2, vec[2] / 2}
	}
	return solid3D(model3d.NewRect(
		model3d.XYZ(min[0], min[1], min[2]),
		model3d.XYZ(max[0], max[1], max[2]),
	)), nil
}

func handleCylinder(e *env, st *CallStmt, _ []SolidValue, _ *SolidValue) (SolidValue, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "h", Pos: 0, Default: Num(1.0)},
		{Name: "r", Pos: 1, Default: Num(1.0)},
		{Name: "center", Pos: 2, Default: Bool(false)},
	})
	if err != nil {
		return SolidValue{}, err
	}
	h, err := argNum(args, "h", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	r, err := argNum(args, "r", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	center, err := argBool(args, "center", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	z0 := 0.0
	z1 := h
	if center {
		z0 = -h / 2
		z1 = h / 2
	}
	return solid3D(&model3d.Cylinder{
		P1:     model3d.XYZ(0, 0, z0),
		P2:     model3d.XYZ(0, 0, z1),
		Radius: r,
	}), nil
}

func handleCircle(e *env, st *CallStmt, _ []SolidValue, _ *SolidValue) (SolidValue, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "r", Pos: 0, Default: Num(1.0)},
	})
	if err != nil {
		return SolidValue{}, err
	}
	r, err := argNum(args, "r", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	return solid2D(&model2d.Circle{Radius: r}), nil
}

func handleSquare(e *env, st *CallStmt, _ []SolidValue, _ *SolidValue) (SolidValue, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "size", Pos: 0, Default: Num(1)},
		{Name: "center", Pos: 1, Default: Bool(false)},
	})
	if err != nil {
		return SolidValue{}, err
	}
	sizeV, ok := args["size"]
	if !ok {
		return SolidValue{}, fmt.Errorf("missing parameter \"size\"")
	}
	vec, err := sizeV.AsVec2(st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	center, err := argBool(args, "center", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	min := [2]float64{0, 0}
	max := vec
	if center {
		min = [2]float64{-vec[0] / 2, -vec[1] / 2}
		max = [2]float64{vec[0] / 2, vec[1] / 2}
	}
	return solid2D(model2d.NewRect(
		model2d.XY(min[0], min[1]),
		model2d.XY(max[0], max[1]),
	)), nil
}
