package scad

import (
	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	"github.com/unixpickle/webgpu-mesh/shapekernel"
)

func rect2DSolidKernel(rect *model2d.Rect) *shapekernel.ShapeKernel {
	return rect2DKernel(rect, false)
}

func rect2DSDFKernel(rect *model2d.Rect) *shapekernel.ShapeKernel {
	return rect2DKernel(rect, true)
}

func rect3DSolidKernel(rect *model3d.Rect) *shapekernel.ShapeKernel {
	return rect3DKernel(rect, false)
}

func rect3DSDFKernel(rect *model3d.Rect) *shapekernel.ShapeKernel {
	return rect3DKernel(rect, true)
}

func rect2DKernel(rect *model2d.Rect, sdf bool) *shapekernel.ShapeKernel {
	min := rect.Min()
	max := rect.Max()
	size := shapekernel.Vec2{float32(max.X - min.X), float32(max.Y - min.Y)}
	center := shapekernel.Vec2{float32((min.X + max.X) / 2), float32((min.Y + max.Y) / 2)}

	var k shapekernel.ShapeKernel
	if sdf {
		k = shapekernel.Rect2DSDF(size)
	} else {
		k = shapekernel.Rect2DSolid(size)
	}
	if center[0] != 0 || center[1] != 0 {
		k = shapekernel.Translate(k, center)
	}
	return asPtr(k)
}

func rect3DKernel(rect *model3d.Rect, sdf bool) *shapekernel.ShapeKernel {
	min := rect.Min()
	max := rect.Max()
	size := shapekernel.Vec3{float32(max.X - min.X), float32(max.Y - min.Y), float32(max.Z - min.Z)}
	center := shapekernel.Vec3{
		float32((min.X + max.X) / 2),
		float32((min.Y + max.Y) / 2),
		float32((min.Z + max.Z) / 2),
	}

	var k shapekernel.ShapeKernel
	if sdf {
		k = shapekernel.Rect3DSDF(size)
	} else {
		k = shapekernel.Rect3DSolid(size)
	}
	if center[0] != 0 || center[1] != 0 || center[2] != 0 {
		k = shapekernel.Translate(k, center)
	}
	return asPtr(k)
}

func primitiveSolidKernel3D(shape any) *shapekernel.ShapeKernel {
	switch s := shape.(type) {
	case *model3d.Rect:
		return rect3DSolidKernel(s)
	case *model3d.Sphere:
		return asPtr(shapekernel.SphereSolid(float32(s.Radius)))
	case *model3d.Capsule:
		return asPtr(shapekernel.Capsule3DSolid(coordToVec3(s.P1), coordToVec3(s.P2), float32(s.Radius)))
	case *model3d.Cylinder:
		return asPtr(shapekernel.CylinderSolid(coordToVec3(s.P1), coordToVec3(s.P2), float32(s.Radius)))
	case *model3d.Cone:
		return asPtr(shapekernel.ConeSolid(coordToVec3(s.Tip), coordToVec3(s.Base), float32(s.Radius)))
	case *model3d.ConeSlice:
		return asPtr(shapekernel.ConeSliceSolid(coordToVec3(s.P1), coordToVec3(s.P2), float32(s.R1), float32(s.R2)))
	default:
		return nil
	}
}

func primitiveSDFKernel3D(shape any) *shapekernel.ShapeKernel {
	switch s := shape.(type) {
	case *model3d.Rect:
		return rect3DSDFKernel(s)
	case *model3d.Sphere:
		return asPtr(shapekernel.SphereSDF(float32(s.Radius)))
	case *model3d.Capsule:
		return asPtr(shapekernel.Capsule3DSDF(coordToVec3(s.P1), coordToVec3(s.P2), float32(s.Radius)))
	case *model3d.Cylinder:
		return asPtr(shapekernel.CylinderSDF(coordToVec3(s.P1), coordToVec3(s.P2), float32(s.Radius)))
	case *model3d.Cone:
		return asPtr(shapekernel.ConeSDF(coordToVec3(s.Tip), coordToVec3(s.Base), float32(s.Radius)))
	case *model3d.ConeSlice:
		return asPtr(shapekernel.ConeSliceSDF(coordToVec3(s.P1), coordToVec3(s.P2), float32(s.R1), float32(s.R2)))
	default:
		return nil
	}
}

func primitiveSolidKernel2D(shape any) *shapekernel.ShapeKernel {
	switch s := shape.(type) {
	case *model2d.Circle:
		return asPtr(shapekernel.CircleSolid(float32(s.Radius)))
	case *model2d.Rect:
		return rect2DSolidKernel(s)
	default:
		return nil
	}
}

func primitiveSDFKernel2D(shape any) *shapekernel.ShapeKernel {
	switch s := shape.(type) {
	case *model2d.Circle:
		return asPtr(shapekernel.CircleSDF(float32(s.Radius)))
	case *model2d.Rect:
		return rect2DSDFKernel(s)
	default:
		return nil
	}
}

func meshSolidKernel2D(mesh *model2d.Mesh) *shapekernel.ShapeKernel {
	return asPtr(shapekernel.Mesh2DSolid(mesh))
}

func meshSDFKernel2D(mesh *model2d.Mesh) *shapekernel.ShapeKernel {
	return asPtr(shapekernel.Mesh2DSDF(mesh))
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
	return shapekernel.Vec2{float32(c.X), float32(c.Y)}
}

func coordToVec3(c model3d.Coord3D) shapekernel.Vec3 {
	return shapekernel.Vec3{float32(c.X), float32(c.Y), float32(c.Z)}
}

func sliceToVec3(c []float64) shapekernel.Vec3 {
	return shapekernel.Vec3{float32(c[0]), float32(c[1]), float32(c[2])}
}

func sliceToVec2(c []float64) shapekernel.Vec2 {
	return shapekernel.Vec2{float32(c[0]), float32(c[1])}
}
