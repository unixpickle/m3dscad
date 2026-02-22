package scad

import (
	"fmt"
	"math"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

type ShapeKind int

const (
	ShapeSolid2D ShapeKind = iota
	ShapeSolid3D
	ShapeMesh2D
	ShapeMesh3D
	ShapeSDF2D
	ShapeSDF3D
)

type ShapeRep struct {
	Kind ShapeKind
	S2   model2d.Solid
	S3   model3d.Solid
	M2   *model2d.Mesh
	M3   *model3d.Mesh
	SDF2 model2d.SDF
	SDF3 model3d.SDF
}

func shapeSolid2D(s model2d.Solid) ShapeRep { return ShapeRep{Kind: ShapeSolid2D, S2: s} }
func shapeSolid3D(s model3d.Solid) ShapeRep { return ShapeRep{Kind: ShapeSolid3D, S3: s} }
func shapeMesh2D(m *model2d.Mesh) ShapeRep  { return ShapeRep{Kind: ShapeMesh2D, M2: m} }
func shapeMesh3D(m *model3d.Mesh) ShapeRep  { return ShapeRep{Kind: ShapeMesh3D, M3: m} }
func shapeSDF2D(s model2d.SDF) ShapeRep     { return ShapeRep{Kind: ShapeSDF2D, SDF2: s} }
func shapeSDF3D(s model3d.SDF) ShapeRep     { return ShapeRep{Kind: ShapeSDF3D, SDF3: s} }

func handleSolid(_ *env, _ *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	switch childUnion.Kind {
	case ShapeSolid2D, ShapeSolid3D:
		return *childUnion, nil
	case ShapeMesh2D:
		return shapeSolid2D(childUnion.M2.Solid()), nil
	case ShapeMesh3D:
		return shapeSolid3D(childUnion.M3.Solid()), nil
	case ShapeSDF2D:
		return shapeSolid2D(model2d.SDFToSolid(childUnion.SDF2, 0)), nil
	case ShapeSDF3D:
		return shapeSolid3D(model3d.SDFToSolid(childUnion.SDF3, 0)), nil
	default:
		return ShapeRep{}, fmt.Errorf("solid(): unsupported shape kind")
	}
}

func unionAll(children []ShapeRep) (ShapeRep, error) {
	if len(children) == 0 {
		return ShapeRep{}, fmt.Errorf("no shapes produced")
	}
	if len(children) == 1 {
		return children[0], nil
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return ShapeRep{}, err
	}
	switch kind {
	case ShapeSolid3D:
		solids := make([]model3d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S3)
		}
		return shapeSolid3D(model3d.JoinedSolid(solids)), nil
	case ShapeSolid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S2)
		}
		return shapeSolid2D(model2d.JoinedSolid(solids)), nil
	case ShapeSDF3D:
		return shapeSDF3D(sdfUnion3D(children)), nil
	case ShapeSDF2D:
		return shapeSDF2D(sdfUnion2D(children)), nil
	case ShapeMesh3D, ShapeMesh2D:
		return ShapeRep{}, fmt.Errorf("cannot union meshes")
	default:
		return ShapeRep{}, fmt.Errorf("unknown shape kind")
	}
}

func ensureSameKind(children []ShapeRep) (ShapeKind, error) {
	if len(children) == 0 {
		return ShapeSolid3D, fmt.Errorf("no shapes produced")
	}
	kind := children[0].Kind
	for _, ch := range children[1:] {
		if ch.Kind != kind {
			return kind, fmt.Errorf("mixed shape kinds")
		}
	}
	return kind, nil
}

func sdfUnion2D(children []ShapeRep) model2d.SDF {
	sdfs := make([]model2d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF2)
	}
	min, max := model2d.BoundsUnion(sdfs)
	return model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
		val := sdfs[0].SDF(c)
		for _, s := range sdfs[1:] {
			val = math.Max(val, s.SDF(c))
		}
		return val
	})
}

func sdfUnion3D(children []ShapeRep) model3d.SDF {
	sdfs := make([]model3d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF3)
	}
	min, max := model3d.BoundsUnion(sdfs)
	return model3d.FuncSDF(min, max, func(c model3d.Coord3D) float64 {
		val := sdfs[0].SDF(c)
		for _, s := range sdfs[1:] {
			val = math.Max(val, s.SDF(c))
		}
		return val
	})
}

func sdfIntersect2D(children []ShapeRep) model2d.SDF {
	sdfs := make([]model2d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF2)
	}
	min, max := boundsIntersect2D(sdfs)
	return model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
		val := sdfs[0].SDF(c)
		for _, s := range sdfs[1:] {
			val = math.Min(val, s.SDF(c))
		}
		return val
	})
}

func sdfIntersect3D(children []ShapeRep) model3d.SDF {
	sdfs := make([]model3d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF3)
	}
	min, max := boundsIntersect3D(sdfs)
	return model3d.FuncSDF(min, max, func(c model3d.Coord3D) float64 {
		val := sdfs[0].SDF(c)
		for _, s := range sdfs[1:] {
			val = math.Min(val, s.SDF(c))
		}
		return val
	})
}

func sdfSubtract2D(a, b ShapeRep) model2d.SDF {
	min := a.SDF2.Min()
	max := a.SDF2.Max()
	return model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
		return math.Min(a.SDF2.SDF(c), -b.SDF2.SDF(c))
	})
}

func sdfSubtract3D(a, b ShapeRep) model3d.SDF {
	min := a.SDF3.Min()
	max := a.SDF3.Max()
	return model3d.FuncSDF(min, max, func(c model3d.Coord3D) float64 {
		return math.Min(a.SDF3.SDF(c), -b.SDF3.SDF(c))
	})
}

func boundsIntersect2D[B model2d.Bounder](bounds []B) (min, max model2d.Coord) {
	min = bounds[0].Min()
	max = bounds[0].Max()
	for _, b := range bounds[1:] {
		min = min.Max(b.Min())
		max = max.Min(b.Max())
	}
	return min, max
}

func boundsIntersect3D[B model3d.Bounder](bounds []B) (min, max model3d.Coord3D) {
	min = bounds[0].Min()
	max = bounds[0].Max()
	for _, b := range bounds[1:] {
		min = min.Max(b.Min())
		max = max.Min(b.Max())
	}
	return min, max
}
