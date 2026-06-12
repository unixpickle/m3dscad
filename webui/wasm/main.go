//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"

	"github.com/unixpickle/m3dscad/scad"
	shapekernel "github.com/unixpickle/webgpu-meshes/shapekernel"
)

type meshBackend string

const (
	meshBackendCPU        meshBackend = "cpu"
	meshBackendGPUFloat32 meshBackend = "gpu_f32"
	meshBackendGPUFixed64 meshBackend = "gpu_fixed64"
)

func main() {
	js.Global().Set("m3dscadCompile", js.FuncOf(compile))
	select {}
}

func compile(_ js.Value, args []js.Value) any {
	if len(args) < 2 {
		return newPromise(func() (js.Value, error) {
			return js.Null(), fmt.Errorf("compile(code, gridSize, options?) requires at least 2 arguments")
		})
	}
	code := args[0].String()
	gridSize := args[1].Int()
	if gridSize < 2 {
		return newPromise(func() (js.Value, error) {
			return js.Null(), fmt.Errorf("gridSize must be >= 2")
		})
	}
	backend := meshBackendCPU
	if len(args) >= 3 && args[2].Type() == js.TypeObject {
		if opt := args[2].Get("meshBackend"); opt.Type() == js.TypeString {
			backend = meshBackend(opt.String())
		} else if opt := args[2].Get("useWebGPU"); opt.Type() == js.TypeBoolean && opt.Bool() {
			backend = meshBackendGPUFloat32
		}
	}
	if !backend.Valid() {
		return newPromise(func() (js.Value, error) {
			return js.Null(), fmt.Errorf("unknown mesh backend %q", backend)
		})
	}

	return newPromise(func() (res js.Value, err error) {
		defer func() {
			if rec := recover(); rec != nil {
				err = fmt.Errorf("%s", panicMessage(rec))
			}
		}()
		logWASMMessage(fmt.Sprintf("[m3dscad] compile start: grid=%d backend=%s", gridSize, backend))
		prog, err := scad.Parse(code)
		if err != nil {
			return js.Null(), err
		}
		hooks := wasmHooks(backend)
		shape, err := scad.Eval(prog, hooks)
		if err != nil {
			return js.Null(), err
		}

		mesh, err := shapeToMesh(shape, gridSize, hooks)
		if err != nil {
			return js.Null(), err
		}
		return meshResponse(mesh), nil
	})
}

func wasmEchoHandler(msg string) {
	echoMsg := js.Global().Get("Object").New()
	echoMsg.Set("type", "echo")
	echoMsg.Set("message", msg)
	js.Global().Call("postMessage", echoMsg)
}

func logWASMMessage(msg string) {
	logMsg := js.Global().Get("Object").New()
	logMsg.Set("type", "log")
	logMsg.Set("message", msg)
	js.Global().Call("postMessage", logMsg)
}

func (b meshBackend) Valid() bool {
	return b == meshBackendCPU || b == meshBackendGPUFloat32 || b == meshBackendGPUFixed64
}

func (b meshBackend) UseWebGPU() bool {
	return b != meshBackendCPU
}

func (b meshBackend) Numerics() shapekernel.Numerics {
	if b == meshBackendGPUFixed64 {
		return shapekernel.Fixed64Numerics
	}
	return shapekernel.NativeFloat32Numerics
}

func wasmHooks(backend meshBackend) scad.Hooks {
	return scad.Hooks{
		Numerics: backend.Numerics(),
		Echo:     wasmEchoHandler,
		MarchingSquares: func(obj scad.ShapeRep, delta float64, iters int) (*model2d.Mesh, error) {
			return mesh2DWithHooks(obj, delta, iters, backend)
		},
		MarchingCubes: func(obj scad.ShapeRep, delta float64, iters int) (*model3d.Mesh, error) {
			return mesh3DWithMarchingCubes(obj, delta, iters, backend)
		},
		DualContour: func(obj scad.ShapeRep, delta float64, repair, clip bool) (*model3d.Mesh, error) {
			return mesh3DWithDualContour(obj, delta, repair, clip, backend)
		},
	}
}

func cpuMarchingSquares(obj scad.ShapeRep, delta float64, iters int) (*model2d.Mesh, error) {
	return model2d.MarchingSquaresSearch(obj.S2, delta, iters), nil
}

func cpuMarchingCubes(obj scad.ShapeRep, delta float64, iters int) (*model3d.Mesh, error) {
	return model3d.MarchingCubesSearch(obj.S3, delta, iters), nil
}

func cpuDualContour(obj scad.ShapeRep, delta float64, repair, clip bool) (*model3d.Mesh, error) {
	return model3d.DualContour(obj.S3, delta, repair, clip), nil
}

func shapeToMesh(shape scad.ShapeRep, gridSize int, hooks scad.Hooks) (*model3d.Mesh, error) {
	switch shape.Kind {
	case scad.ShapeMesh3D:
		logWASMMessage("[m3dscad] preview path: existing mesh output (no meshing hook)")
		return shape.M3, nil
	case scad.ShapeSolid3D:
		delta, err := marchingDelta(shape.S3, gridSize)
		if err != nil {
			return nil, err
		}
		return hooks.DualContour(shape, delta, true, false)
	case scad.ShapeSDF3D:
		solid := scad.SDFToSolid(hooks.Numerics, shape)
		delta, err := marchingDelta(solid.S3, gridSize)
		if err != nil {
			return nil, err
		}
		return hooks.DualContour(solid, delta, true, false)
	case scad.ShapeSolid2D, scad.ShapeMesh2D, scad.ShapeSDF2D:
		logWASMMessage(fmt.Sprintf("[m3dscad] preview path: unsupported 2D output kind=%v", shape.Kind))
		return nil, fmt.Errorf("2D outputs are not supported in the 3D preview")
	default:
		logWASMMessage(fmt.Sprintf("[m3dscad] preview path: unsupported output kind=%v", shape.Kind))
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

func newPromise(f func() (js.Value, error)) js.Value {
	executor := js.FuncOf(func(_ js.Value, args []js.Value) any {
		resolve := args[0]
		reject := args[1]
		go func() {
			result, err := f()
			if err != nil {
				reject.Invoke(err.Error())
				return
			}
			resolve.Invoke(result)
		}()
		return nil
	})
	defer executor.Release()
	return js.Global().Get("Promise").New(executor)
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
