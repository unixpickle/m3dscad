package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleLinearExtrude(e *env, st *CallStmt, _ []SolidValue, childUnion *SolidValue) (SolidValue, error) {
	if childUnion.Kind != Solid2D {
		return SolidValue{}, fmt.Errorf("linear_extrude() requires 2D children")
	}
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "height", Aliases: []string{"h"}, Pos: 0, Default: Num(1.0)},
		{Name: "center", Pos: 1, Default: Bool(false)},
	})
	if err != nil {
		return SolidValue{}, err
	}
	height, err := argNum(args, "height", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	center, err := argBool(args, "center", st.pos())
	if err != nil {
		return SolidValue{}, err
	}
	return solid3D(linearExtrude(childUnion.Solid2, height, center)), nil
}

func linearExtrude(s model2d.Solid, height float64, center bool) model3d.Solid {
	if height < 0 {
		height = -height
	}
	z0 := 0.0
	z1 := height
	if center {
		z0 = -height / 2
		z1 = height / 2
	}
	min2 := s.Min()
	max2 := s.Max()
	min := model3d.XYZ(min2.X, min2.Y, z0)
	max := model3d.XYZ(max2.X, max2.Y, z1)
	return model3d.CheckedFuncSolid(min, max, func(c model3d.Coord3D) bool {
		if c.Z < z0 || c.Z > z1 {
			return false
		}
		return s.Contains(model2d.XY(c.X, c.Y))
	})
}
