package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

type SolidKind int

const (
	Solid2D SolidKind = iota
	Solid3D
)

type SolidValue struct {
	Kind   SolidKind
	Solid2 model2d.Solid
	Solid3 model3d.Solid
}

func solid2D(s model2d.Solid) SolidValue {
	return SolidValue{Kind: Solid2D, Solid2: s}
}

func solid3D(s model3d.Solid) SolidValue {
	return SolidValue{Kind: Solid3D, Solid3: s}
}

func unionAll(children []SolidValue) (SolidValue, error) {
	if len(children) == 0 {
		return SolidValue{}, fmt.Errorf("no solids produced")
	}
	if len(children) == 1 {
		return children[0], nil
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return SolidValue{}, err
	}
	switch kind {
	case Solid3D:
		solids := make([]model3d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.Solid3)
		}
		return solid3D(model3d.JoinedSolid(solids)), nil
	case Solid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.Solid2)
		}
		return solid2D(model2d.JoinedSolid(solids)), nil
	default:
		return SolidValue{}, fmt.Errorf("unknown solid kind")
	}
}

func ensureSameKind(children []SolidValue) (SolidKind, error) {
	if len(children) == 0 {
		return Solid3D, fmt.Errorf("no solids produced")
	}
	kind := children[0].Kind
	for _, ch := range children[1:] {
		if ch.Kind != kind {
			return kind, fmt.Errorf("mixed 2D and 3D solids")
		}
	}
	return kind, nil
}
