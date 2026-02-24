//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/unixpickle/model3d/model3d"

	"github.com/unixpickle/m3dscad/scad"
)

func main() {
	js.Global().Set("m3dscadCompile", js.FuncOf(compile))
	select {}
}

func compile(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return jsError("compile(code, gridSize) requires 2 arguments")
	}
	code := args[0].String()
	gridSize := args[1].Int()
	if gridSize < 2 {
		return jsError("gridSize must be >= 2")
	}

	prog, err := scad.Parse(code)
	if err != nil {
		return jsError(err.Error())
	}
	shape, err := scad.Eval(prog)
	if err != nil {
		return jsError(err.Error())
	}

	mesh, err := shapeToMesh(shape, gridSize)
	if err != nil {
		return jsError(err.Error())
	}
	return meshResponse(mesh)
}

func shapeToMesh(shape scad.ShapeRep, gridSize int) (*model3d.Mesh, error) {
	switch shape.Kind {
	case scad.ShapeMesh3D:
		return shape.M3, nil
	case scad.ShapeSolid3D:
		delta, err := marchingDelta(shape.S3, gridSize)
		if err != nil {
			return nil, err
		}
		return model3d.MarchingCubesSearch(shape.S3, delta, 8), nil
	case scad.ShapeSDF3D:
		solid := model3d.SDFToSolid(shape.SDF3, 0)
		delta, err := marchingDelta(solid, gridSize)
		if err != nil {
			return nil, err
		}
		return model3d.MarchingCubesSearch(solid, delta, 8), nil
	case scad.ShapeSolid2D, scad.ShapeMesh2D, scad.ShapeSDF2D:
		return nil, fmt.Errorf("2D outputs are not supported in the 3D preview")
	default:
		return nil, fmt.Errorf("unsupported output kind")
	}
}

func marchingDelta(solid model3d.Solid, gridSize int) (float64, error) {
	min := solid.Min()
	max := solid.Max()
	size := max.Sub(min)
	maxDim := size.Abs().MaxCoord()
	if maxDim == 0 {
		return 0, fmt.Errorf("shape has zero size")
	}
	return maxDim / float64(gridSize), nil
}

func meshResponse(mesh *model3d.Mesh) js.Value {
	tris := mesh.TriangleSlice()
	positions := make([]float64, 0, len(tris)*9)
	normals := make([]float64, 0, len(tris)*9)
	for _, tri := range tris {
		n := tri.Normal()
		for i := 0; i < 3; i++ {
			p := tri[i]
			positions = append(positions, p.X, p.Y, p.Z)
			normals = append(normals, n.X, n.Y, n.Z)
		}
	}
	min := mesh.Min()
	max := mesh.Max()
	res := js.Global().Get("Object").New()
	res.Set("ok", true)
	res.Set("positions", jsFloat32Array(positions))
	res.Set("normals", jsFloat32Array(normals))
	bounds := js.Global().Get("Object").New()
	bounds.Set("min", jsFloat64Array([]float64{min.X, min.Y, min.Z}))
	bounds.Set("max", jsFloat64Array([]float64{max.X, max.Y, max.Z}))
	res.Set("bounds", bounds)
	return res
}

func jsError(msg string) js.Value {
	res := js.Global().Get("Object").New()
	res.Set("ok", false)
	res.Set("error", msg)
	return res
}

func jsFloat32Array(values []float64) js.Value {
	arr := js.Global().Get("Float32Array").New(len(values))
	for i, v := range values {
		arr.SetIndex(i, float32(v))
	}
	return arr
}

func jsFloat64Array(values []float64) js.Value {
	arr := js.Global().Get("Array").New(len(values))
	for i, v := range values {
		arr.SetIndex(i, v)
	}
	return arr
}
