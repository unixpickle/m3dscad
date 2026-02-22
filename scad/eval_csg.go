package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleUnion(_ *env, _ *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if childUnion == nil {
		return ShapeRep{}, fmt.Errorf("union() requires children")
	}
	return *childUnion, nil
}

func handleDifference(_ *env, _ *CallStmt, children []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	if len(children) == 0 {
		return ShapeRep{}, fmt.Errorf("difference() had no solids")
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return ShapeRep{}, fmt.Errorf("difference(): %w", err)
	}
	if len(children) == 1 {
		return children[0], nil
	}
	subUnion, err := unionAll(children[1:])
	if err != nil {
		return ShapeRep{}, err
	}
	switch kind {
	case ShapeSolid3D:
		return shapeSolid3D(model3d.Subtract(children[0].S3, subUnion.S3)), nil
	case ShapeSolid2D:
		return shapeSolid2D(model2d.Subtract(children[0].S2, subUnion.S2)), nil
	case ShapeSDF3D:
		return shapeSDF3D(sdfSubtract3D(children[0], subUnion)), nil
	case ShapeSDF2D:
		return shapeSDF2D(sdfSubtract2D(children[0], subUnion)), nil
	case ShapeMesh2D, ShapeMesh3D:
		return ShapeRep{}, fmt.Errorf("difference() not supported for meshes")
	default:
		return ShapeRep{}, fmt.Errorf("difference(): unknown shape kind")
	}
}

func handleIntersection(_ *env, _ *CallStmt, children []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	if len(children) == 0 {
		return ShapeRep{}, fmt.Errorf("intersection() had no solids")
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return ShapeRep{}, fmt.Errorf("intersection(): %w", err)
	}
	switch kind {
	case ShapeSolid3D:
		solids := make([]model3d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S3)
		}
		return shapeSolid3D(model3d.IntersectedSolid(solids)), nil
	case ShapeSolid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S2)
		}
		return shapeSolid2D(model2d.IntersectedSolid(solids)), nil
	case ShapeSDF3D:
		return shapeSDF3D(sdfIntersect3D(children)), nil
	case ShapeSDF2D:
		return shapeSDF2D(sdfIntersect2D(children)), nil
	case ShapeMesh2D, ShapeMesh3D:
		return ShapeRep{}, fmt.Errorf("intersection() not supported for meshes")
	default:
		return ShapeRep{}, fmt.Errorf("intersection(): unknown shape kind")
	}
}
