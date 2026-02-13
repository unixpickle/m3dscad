package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleUnion(_ *env, st *CallStmt, _ []SolidValue, childUnion *SolidValue) (SolidValue, error) {
	if childUnion == nil {
		return SolidValue{}, fmt.Errorf("union() requires children")
	}
	return *childUnion, nil
}

func handleDifference(_ *env, st *CallStmt, children []SolidValue, _ *SolidValue) (SolidValue, error) {
	if len(children) == 0 {
		return SolidValue{}, fmt.Errorf("difference() had no solids")
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return SolidValue{}, fmt.Errorf("difference(): %w", err)
	}
	if len(children) == 1 {
		return children[0], nil
	}
	subUnion, err := unionAll(children[1:])
	if err != nil {
		return SolidValue{}, err
	}
	switch kind {
	case Solid3D:
		return solid3D(model3d.Subtract(children[0].Solid3, subUnion.Solid3)), nil
	case Solid2D:
		return solid2D(model2d.Subtract(children[0].Solid2, subUnion.Solid2)), nil
	default:
		return SolidValue{}, fmt.Errorf("difference(): unknown solid kind")
	}
}

func handleIntersection(_ *env, st *CallStmt, children []SolidValue, _ *SolidValue) (SolidValue, error) {
	if len(children) == 0 {
		return SolidValue{}, fmt.Errorf("intersection() had no solids")
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return SolidValue{}, fmt.Errorf("intersection(): %w", err)
	}
	switch kind {
	case Solid3D:
		solids := make([]model3d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.Solid3)
		}
		return solid3D(model3d.IntersectedSolid(solids)), nil
	case Solid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.Solid2)
		}
		return solid2D(model2d.IntersectedSolid(solids)), nil
	default:
		return SolidValue{}, fmt.Errorf("intersection(): unknown solid kind")
	}
}
