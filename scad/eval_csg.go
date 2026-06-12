package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	shapekernel "github.com/unixpickle/webgpu-meshes/shapekernel"
)

func handleUnion(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
	if childUnion == nil {
		return ShapeRep{}, fmt.Errorf("union() requires children")
	}
	return *childUnion, nil
}

func handleDifference(e *env, st *CallStmt, children []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
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
	subUnion, err := unionAll(e.hooks.Numerics, children[1:])
	if err != nil {
		return ShapeRep{}, err
	}
	switch kind {
	case ShapeSolid3D:
		var k *shapekernel.ShapeKernel
		if children[0].Kernel != nil && subUnion.Kernel != nil {
			k = asPtr(shapekernel.SubtractSolid(e.hooks.Numerics, *children[0].Kernel, *subUnion.Kernel))
		}
		return shapeSolid3D(model3d.Subtract(children[0].S3, subUnion.S3), k), nil
	case ShapeSolid2D:
		var k *shapekernel.ShapeKernel
		if children[0].Kernel != nil && subUnion.Kernel != nil {
			k = asPtr(shapekernel.SubtractSolid(e.hooks.Numerics, *children[0].Kernel, *subUnion.Kernel))
		}
		return shapeSolid2D(model2d.Subtract(children[0].S2, subUnion.S2), k), nil
	case ShapeSDF3D:
		var k *shapekernel.ShapeKernel
		if children[0].Kernel != nil && subUnion.Kernel != nil {
			k = asPtr(shapekernel.SubtractSDF(e.hooks.Numerics, *children[0].Kernel, *subUnion.Kernel))
		}
		return shapeSDF3D(model3d.SubtractSDF(children[0].SDF3, subUnion.SDF3), k), nil
	case ShapeSDF2D:
		var k *shapekernel.ShapeKernel
		if children[0].Kernel != nil && subUnion.Kernel != nil {
			k = asPtr(shapekernel.SubtractSDF(e.hooks.Numerics, *children[0].Kernel, *subUnion.Kernel))
		}
		return shapeSDF2D(model2d.SubtractSDF(children[0].SDF2, subUnion.SDF2), k), nil
	case ShapeMetaball2D:
		return ShapeRep{
			Kind: ShapeMetaball2D,
			MB2:  children[0].MB2.Join(subUnion.MB2.Scale(-1)),
		}, nil
	case ShapeMetaball3D:
		return ShapeRep{
			Kind: ShapeMetaball3D,
			MB3:  children[0].MB3.Join(subUnion.MB3.Scale(-1)),
		}, nil
	case ShapeMesh2D, ShapeMesh3D:
		return ShapeRep{}, fmt.Errorf("difference() not supported for meshes")
	default:
		return ShapeRep{}, fmt.Errorf("difference(): unknown shape kind")
	}
}

func handleIntersection(e *env, st *CallStmt, children []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
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
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.IntersectSolids(e.hooks.Numerics, kernels))
		}
		return shapeSolid3D(model3d.IntersectedSolid(solids), k), nil
	case ShapeSolid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S2)
		}
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.IntersectSolids(e.hooks.Numerics, kernels))
		}
		return shapeSolid2D(model2d.IntersectedSolid(solids), k), nil
	case ShapeSDF3D:
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.IntersectSDFs(e.hooks.Numerics, kernels))
		}
		return shapeSDF3D(sdfIntersect3D(children), k), nil
	case ShapeSDF2D:
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.IntersectSDFs(e.hooks.Numerics, kernels))
		}
		return shapeSDF2D(sdfIntersect2D(children), k), nil
	case ShapeMesh2D, ShapeMesh3D:
		return ShapeRep{}, fmt.Errorf("intersection() not supported for meshes")
	default:
		return ShapeRep{}, fmt.Errorf("intersection(): unknown shape kind")
	}
}
