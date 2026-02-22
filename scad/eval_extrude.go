package scad

import (
	"fmt"
	"math"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleLinearExtrude(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if childUnion.Kind != ShapeSolid2D {
		return ShapeRep{}, fmt.Errorf("linear_extrude() requires 2D children")
	}
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "height", Aliases: []string{"h"}, Pos: 0, Default: Num(1.0)},
		{Name: "center", Pos: 1, Default: Bool(false)},
		{Name: "twist", Pos: 2, Default: Num(0.0)},
		{Name: "scale", Pos: 3, Default: Num(1.0)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	height, err := argNum(args, "height", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	center, err := argBool(args, "center", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	twist, err := argNum(args, "twist", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	scaleV, ok := args["scale"]
	if !ok {
		return ShapeRep{}, fmt.Errorf("missing parameter \"scale\"")
	}
	scale, err := scaleV.AsVec2(st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid3D(linearExtrude(childUnion.S2, height, center, twist, scale)), nil
}

func linearExtrude(s model2d.Solid, height float64, center bool, twist float64, scale [2]float64) model3d.Solid {
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
	maxScale := maxAbsScale(scale)
	r := maxCornerRadius(min2, max2) * maxScale
	min := model3d.XYZ(-r, -r, z0)
	max := model3d.XYZ(r, r, z1)
	return model3d.CheckedFuncSolid(min, max, func(c model3d.Coord3D) bool {
		if c.Z < z0 || c.Z > z1 {
			return false
		}
		t := 0.0
		if height > 0 {
			t = (c.Z - z0) / height
		}
		x, y, ok := inverseExtrudeTransform(c.X, c.Y, t, twist, scale)
		if !ok {
			return false
		}
		return s.Contains(model2d.XY(x, y))
	})
}

func handleRotateExtrude(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if childUnion.Kind != ShapeSolid2D {
		return ShapeRep{}, fmt.Errorf("rotate_extrude() requires 2D children")
	}
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "angle", Pos: 0, Default: Num(360.0)},
		{Name: "start", Pos: 1, Default: Num(0.0)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	angle, err := argNum(args, "angle", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	start, err := argNum(args, "start", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	solid, err := rotateExtrude(childUnion.S2, angle, start)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid3D(solid), nil
}

func rotateExtrude(s model2d.Solid, angleDeg float64, startDeg float64) (model3d.Solid, error) {
	min2 := s.Min()
	max2 := s.Max()
	if min2.X < 0 && max2.X > 0 {
		return nil, fmt.Errorf("rotate_extrude() shape crosses the Y axis")
	}
	sign := 1.0
	if max2.X <= 0 {
		sign = -1.0
	}
	rMax := math.Max(math.Abs(min2.X), math.Abs(max2.X))
	min := model3d.XYZ(-rMax, -rMax, min2.Y)
	max := model3d.XYZ(rMax, rMax, max2.Y)

	angle := angleDeg
	start := normalizeAngleDeg(startDeg)
	full := math.Abs(angle) >= 360-1e-9

	return model3d.CheckedFuncSolid(min, max, func(c model3d.Coord3D) bool {
		r := math.Hypot(c.X, c.Y)
		if !full {
			theta := math.Atan2(c.Y, c.X) * 180 / math.Pi
			if angle >= 0 {
				delta := angleDistanceCCW(start, theta)
				if delta > angle+1e-9 {
					return false
				}
			} else {
				delta := angleDistanceCW(start, theta)
				if delta > -angle+1e-9 {
					return false
				}
			}
		}
		return s.Contains(model2d.XY(sign*r, c.Z))
	}), nil
}

func normalizeAngleDeg(a float64) float64 {
	a = math.Mod(a, 360)
	if a < 0 {
		a += 360
	}
	return a
}

func angleDistanceCCW(from, to float64) float64 {
	return normalizeAngleDeg(to - from)
}

func angleDistanceCW(from, to float64) float64 {
	return normalizeAngleDeg(from - to)
}

func inverseExtrudeTransform(x, y, t, twist float64, scale [2]float64) (float64, float64, bool) {
	sx := 1 + t*(scale[0]-1)
	sy := 1 + t*(scale[1]-1)
	if sx == 0 || sy == 0 {
		return 0, 0, false
	}
	angle := twist * math.Pi / 180 * t
	cosA := math.Cos(angle)
	sinA := math.Sin(angle)
	rx := x*cosA - y*sinA
	ry := x*sinA + y*cosA
	return rx / sx, ry / sy, true
}

func maxAbsScale(scale [2]float64) float64 {
	maxScale := 1.0
	for _, s := range []float64{1, scale[0], scale[1]} {
		if s < 0 {
			s = -s
		}
		if s > maxScale {
			maxScale = s
		}
	}
	return maxScale
}

func maxCornerRadius(min, max model2d.Coord) float64 {
	corners := [4]model2d.Coord{
		model2d.XY(min.X, min.Y),
		model2d.XY(min.X, max.Y),
		model2d.XY(max.X, min.Y),
		model2d.XY(max.X, max.Y),
	}
	maxR2 := 0.0
	for _, c := range corners {
		r2 := c.X*c.X + c.Y*c.Y
		if r2 > maxR2 {
			maxR2 = r2
		}
	}
	return math.Sqrt(maxR2)
}
