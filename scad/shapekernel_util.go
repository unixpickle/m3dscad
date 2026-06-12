package scad

import (
	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	"github.com/unixpickle/webgpu-mesh/shapekernel"
)

func rect2DSolidKernel(n shapekernel.Numerics, rect *model2d.Rect) *shapekernel.ShapeKernel {
	return rect2DKernel(n, rect, false)
}

func rect2DSDFKernel(n shapekernel.Numerics, rect *model2d.Rect) *shapekernel.ShapeKernel {
	return rect2DKernel(n, rect, true)
}

func rect3DSolidKernel(n shapekernel.Numerics, rect *model3d.Rect) *shapekernel.ShapeKernel {
	return rect3DKernel(n, rect, false)
}

func rect3DSDFKernel(n shapekernel.Numerics, rect *model3d.Rect) *shapekernel.ShapeKernel {
	return rect3DKernel(n, rect, true)
}

func rect2DKernel(n shapekernel.Numerics, rect *model2d.Rect, sdf bool) *shapekernel.ShapeKernel {
	min := rect.Min()
	max := rect.Max()
	size := shapekernel.Vec2{max.X - min.X, max.Y - min.Y}
	center := shapekernel.Vec2{(min.X + max.X) / 2, (min.Y + max.Y) / 2}

	var k shapekernel.ShapeKernel
	if sdf {
		k = shapekernel.Rect2DSDF(n, size)
	} else {
		k = shapekernel.Rect2DSolid(n, size)
	}
	if center[0] != 0 || center[1] != 0 {
		k = shapekernel.Translate(n, k, center)
	}
	return asPtr(k)
}

func rect3DKernel(n shapekernel.Numerics, rect *model3d.Rect, sdf bool) *shapekernel.ShapeKernel {
	min := rect.Min()
	max := rect.Max()
	size := shapekernel.Vec3{max.X - min.X, max.Y - min.Y, max.Z - min.Z}
	center := shapekernel.Vec3{
		(min.X + max.X) / 2,
		(min.Y + max.Y) / 2,
		(min.Z + max.Z) / 2,
	}

	var k shapekernel.ShapeKernel
	if sdf {
		k = shapekernel.Rect3DSDF(n, size)
	} else {
		k = shapekernel.Rect3DSolid(n, size)
	}
	if center[0] != 0 || center[1] != 0 || center[2] != 0 {
		k = shapekernel.Translate(n, k, center)
	}
	return asPtr(k)
}

func primitiveSolidKernel3D(n shapekernel.Numerics, shape any) *shapekernel.ShapeKernel {
	switch s := shape.(type) {
	case *model3d.Rect:
		return rect3DSolidKernel(n, s)
	case *model3d.Sphere:
		return asPtr(shapekernel.SphereSolid(n, s.Radius))
	case *model3d.Capsule:
		return asPtr(shapekernel.Capsule3DSolid(n, coordToVec3(s.P1), coordToVec3(s.P2), s.Radius))
	case *model3d.Cylinder:
		return asPtr(shapekernel.CylinderSolid(n, coordToVec3(s.P1), coordToVec3(s.P2), s.Radius))
	case *model3d.Cone:
		return asPtr(shapekernel.ConeSolid(n, coordToVec3(s.Tip), coordToVec3(s.Base), s.Radius))
	case *model3d.ConeSlice:
		return asPtr(shapekernel.ConeSliceSolid(n, coordToVec3(s.P1), coordToVec3(s.P2), s.R1, s.R2))
	default:
		return nil
	}
}

func primitiveSDFKernel3D(n shapekernel.Numerics, shape any) *shapekernel.ShapeKernel {
	switch s := shape.(type) {
	case *model3d.Rect:
		return rect3DSDFKernel(n, s)
	case *model3d.Sphere:
		return asPtr(shapekernel.SphereSDF(n, s.Radius))
	case *model3d.Capsule:
		return asPtr(shapekernel.Capsule3DSDF(n, coordToVec3(s.P1), coordToVec3(s.P2), s.Radius))
	case *model3d.Cylinder:
		return asPtr(shapekernel.CylinderSDF(n, coordToVec3(s.P1), coordToVec3(s.P2), s.Radius))
	case *model3d.Cone:
		return asPtr(shapekernel.ConeSDF(n, coordToVec3(s.Tip), coordToVec3(s.Base), s.Radius))
	case *model3d.ConeSlice:
		return asPtr(shapekernel.ConeSliceSDF(n, coordToVec3(s.P1), coordToVec3(s.P2), s.R1, s.R2))
	default:
		return nil
	}
}

func primitiveSolidKernel2D(n shapekernel.Numerics, shape any) *shapekernel.ShapeKernel {
	switch s := shape.(type) {
	case *model2d.Circle:
		return asPtr(shapekernel.CircleSolid(n, s.Radius))
	case *model2d.Rect:
		return rect2DSolidKernel(n, s)
	default:
		return nil
	}
}

func primitiveSDFKernel2D(n shapekernel.Numerics, shape any) *shapekernel.ShapeKernel {
	switch s := shape.(type) {
	case *model2d.Circle:
		return asPtr(shapekernel.CircleSDF(n, s.Radius))
	case *model2d.Rect:
		return rect2DSDFKernel(n, s)
	default:
		return nil
	}
}

func meshSolidKernel2D(n shapekernel.Numerics, mesh *model2d.Mesh) *shapekernel.ShapeKernel {
	return asPtr(shapekernel.Mesh2DSolid(n, mesh))
}

func meshSDFKernel2D(n shapekernel.Numerics, mesh *model2d.Mesh) *shapekernel.ShapeKernel {
	return asPtr(shapekernel.Mesh2DSDF(n, mesh))
}

func insetExtrudeKernelFunc(name string) (shapekernel.InsetFunction, bool) {
	switch name {
	case "chamfer":
		return shapekernel.InsetExtrudeChamfer, true
	case "fillet":
		return shapekernel.InsetExtrudeFillet, true
	default:
		return "", false
	}
}

func concatKernels(shapes []ShapeRep) ([]shapekernel.ShapeKernel, bool) {
	kernels := []shapekernel.ShapeKernel{}
	for _, ch := range shapes {
		if ch.Kernel == nil {
			return nil, false
		}
		kernels = append(kernels, *ch.Kernel)
	}
	return kernels, true
}

func asPtr[T any](sk T) *T {
	return &sk
}

func coordToVec2(c model2d.Coord) shapekernel.Vec2 {
	return shapekernel.Vec2{c.X, c.Y}
}

func coordToVec3(c model3d.Coord3D) shapekernel.Vec3 {
	return shapekernel.Vec3{c.X, c.Y, c.Z}
}

func sliceToVec3(c []float64) shapekernel.Vec3 {
	return shapekernel.Vec3{c[0], c[1], c[2]}
}

func sliceToVec2(c []float64) shapekernel.Vec2 {
	return shapekernel.Vec2{c[0], c[1]}
}
