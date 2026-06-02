//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	"github.com/unixpickle/webgpu-mesh/shapekernel"

	"github.com/unixpickle/m3dscad/scad"
)

func mesh2DWithHooks(obj scad.ShapeRep, delta float64, iters int, useWebGPU bool) (*model2d.Mesh, error) {
	if !useWebGPU || obj.Kernel == nil || obj.Kernel.Kind != shapekernel.Solid2D {
		logMeshingBackend("marching_squares", useWebGPUFallbackReason(useWebGPU, obj.Kernel, shapekernel.Solid2D))
		return cpuMarchingSquares(obj, delta, iters)
	}
	logMeshingBackend("marching_squares", "WebGPU")
	req, err := webGPUMeshRequest2D("marching_squares", obj.Kernel, obj.S2.Min(), obj.S2.Max(), delta)
	if err != nil {
		return nil, err
	}
	req.Set("subdiv", iters)
	res, err := requestWebGPUMesh(req)
	if err != nil {
		return nil, err
	}
	return mesh2DFromJS(res)
}

func mesh3DWithMarchingCubes(obj scad.ShapeRep, delta float64, iters int, useWebGPU bool) (*model3d.Mesh, error) {
	if !useWebGPU || obj.Kernel == nil || obj.Kernel.Kind != shapekernel.Solid3D {
		logMeshingBackend("marching_cubes", useWebGPUFallbackReason(useWebGPU, obj.Kernel, shapekernel.Solid3D))
		return cpuMarchingCubes(obj, delta, iters)
	}
	logMeshingBackend("marching_cubes", "WebGPU")
	req, err := webGPUMeshRequest3D("marching_cubes", obj.Kernel, obj.S3.Min(), obj.S3.Max(), delta)
	if err != nil {
		return nil, err
	}
	req.Set("subdiv", iters)
	res, err := requestWebGPUMesh(req)
	if err != nil {
		return nil, err
	}
	return mesh3DFromJS(res)
}

func mesh3DWithDualContour(obj scad.ShapeRep, delta float64, repair, clip bool, useWebGPU bool) (*model3d.Mesh, error) {
	if !useWebGPU || obj.Kernel == nil || obj.Kernel.Kind != shapekernel.Solid3D {
		logMeshingBackend("dual_contour", useWebGPUFallbackReason(useWebGPU, obj.Kernel, shapekernel.Solid3D))
		return cpuDualContour(obj, delta, repair, clip)
	}
	logMeshingBackend("dual_contour", "WebGPU")
	req, err := webGPUMeshRequest3D("dual_contour", obj.Kernel, obj.S3.Min(), obj.S3.Max(), delta)
	if err != nil {
		return nil, err
	}
	req.Set("repair", repair)
	req.Set("clip", clip)
	res, err := requestWebGPUMesh(req)
	if err != nil {
		return nil, err
	}
	return mesh3DFromJS(res)
}

func webGPUMeshRequest2D(method string, kernel *shapekernel.ShapeKernel, min, max model2d.Coord, delta float64) (js.Value, error) {
	req, err := webGPUMeshRequest(method, kernel, 2, delta)
	if err != nil {
		return js.Null(), err
	}
	req.Set("min", jsFloat64Array([]float64{min.X, min.Y}))
	req.Set("max", jsFloat64Array([]float64{max.X, max.Y}))
	return req, nil
}

func webGPUMeshRequest3D(method string, kernel *shapekernel.ShapeKernel, min, max model3d.Coord3D, delta float64) (js.Value, error) {
	req, err := webGPUMeshRequest(method, kernel, 3, delta)
	if err != nil {
		return js.Null(), err
	}
	req.Set("min", jsFloat64Array([]float64{min.X, min.Y, min.Z}))
	req.Set("max", jsFloat64Array([]float64{max.X, max.Y, max.Z}))
	return req, nil
}

func webGPUMeshRequest(method string, kernel *shapekernel.ShapeKernel, dim int, delta float64) (js.Value, error) {
	if kernel == nil {
		return js.Null(), fmt.Errorf("WebGPU meshing requires a non-nil shape kernel")
	}
	serializedKernel, err := serializeShapeKernel(kernel, dim)
	if err != nil {
		return js.Null(), err
	}
	req := js.Global().Get("Object").New()
	req.Set("method", method)
	req.Set("delta", delta)
	req.Set("kernel", serializedKernel)
	return req, nil
}

func serializeShapeKernel(kernel *shapekernel.ShapeKernel, dim int) (js.Value, error) {
	result := js.Global().Get("Object").New()
	result.Set("dimension", dim)
	result.Set("wgsl", kernelWGSL(kernel, dim))
	bindings := js.Global().Get("Array").New(len(kernel.Buffers))
	for i, buffer := range kernel.Buffers {
		bindingWGSLType, err := shapeKernelBindingWGSLType(buffer.WGSLType)
		if err != nil {
			return js.Null(), err
		}
		bindingSource, err := jsShapeKernelBufferSource(buffer)
		if err != nil {
			return js.Null(), err
		}
		binding := js.Global().Get("Object").New()
		binding.Set("name", buffer.Name)
		binding.Set("kind", "storage")
		binding.Set("wgslType", bindingWGSLType)
		binding.Set("source", bindingSource)
		bindings.SetIndex(i, binding)
	}
	result.Set("bindings", bindings)
	return result, nil
}

func kernelWGSL(kernel *shapekernel.ShapeKernel, dim int) string {
	return fmt.Sprintf(
		"%s\nfn solidOccupancy(p: vec%d<f32>) -> bool {\n\treturn %s(p);\n}\n",
		kernel.Code,
		dim,
		kernel.EntrypointName,
	)
}

func requestWebGPUMesh(req js.Value) (js.Value, error) {
	requestFn := js.Global().Get("m3dscadWebGPUMesh")
	if requestFn.Type() != js.TypeFunction {
		return js.Null(), fmt.Errorf("WebGPU bridge is unavailable")
	}
	promise := requestFn.Invoke(req)
	type promiseResult struct {
		value js.Value
		err   error
	}
	resultChan := make(chan promiseResult, 1)
	var thenFn js.Func
	var catchFn js.Func
	thenFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		value := js.Null()
		if len(args) > 0 {
			value = args[0]
		}
		resultChan <- promiseResult{value: value}
		return nil
	})
	catchFn = js.FuncOf(func(_ js.Value, args []js.Value) any {
		errMsg := "WebGPU meshing failed"
		if len(args) > 0 {
			errMsg = jsErrorString(args[0], errMsg)
		}
		resultChan <- promiseResult{err: fmt.Errorf("%s", errMsg)}
		return nil
	})
	defer thenFn.Release()
	defer catchFn.Release()
	promise.Call("then", thenFn)
	promise.Call("catch", catchFn)
	result := <-resultChan
	return result.value, result.err
}

func mesh2DFromJS(value js.Value) (*model2d.Mesh, error) {
	positions := jsFloat64Slice(value.Get("positions"))
	indices := jsIntSlice(value.Get("indices"))
	if len(positions)%2 != 0 {
		return nil, fmt.Errorf("invalid WebGPU 2D mesh positions length: %d", len(positions))
	}
	if len(indices)%2 != 0 {
		return nil, fmt.Errorf("invalid WebGPU 2D mesh indices length: %d", len(indices))
	}
	mesh := model2d.NewMesh()
	for i := 0; i < len(indices); i += 2 {
		a, err := mesh2DCoordAt(positions, indices[i])
		if err != nil {
			return nil, err
		}
		b, err := mesh2DCoordAt(positions, indices[i+1])
		if err != nil {
			return nil, err
		}
		mesh.Add(&model2d.Segment{a, b})
	}
	return mesh, nil
}

func mesh3DFromJS(value js.Value) (*model3d.Mesh, error) {
	positions := jsFloat64Slice(value.Get("positions"))
	indices := jsIntSlice(value.Get("indices"))
	if len(positions)%3 != 0 {
		return nil, fmt.Errorf("invalid WebGPU 3D mesh positions length: %d", len(positions))
	}
	if len(indices)%3 != 0 {
		return nil, fmt.Errorf("invalid WebGPU 3D mesh indices length: %d", len(indices))
	}
	tris := make([]*model3d.Triangle, 0, len(indices)/3)
	for i := 0; i < len(indices); i += 3 {
		a, err := mesh3DCoordAt(positions, indices[i])
		if err != nil {
			return nil, err
		}
		b, err := mesh3DCoordAt(positions, indices[i+1])
		if err != nil {
			return nil, err
		}
		c, err := mesh3DCoordAt(positions, indices[i+2])
		if err != nil {
			return nil, err
		}
		tris = append(tris, &model3d.Triangle{a, b, c})
	}
	return model3d.NewMeshTriangles(tris), nil
}

func mesh2DCoordAt(positions []float64, index int) (model2d.Coord, error) {
	base := index * 2
	if base < 0 || base+1 >= len(positions) {
		return model2d.Coord{}, fmt.Errorf("invalid 2D mesh vertex index: %d", index)
	}
	return model2d.XY(positions[base], positions[base+1]), nil
}

func mesh3DCoordAt(positions []float64, index int) (model3d.Coord3D, error) {
	base := index * 3
	if base < 0 || base+2 >= len(positions) {
		return model3d.Coord3D{}, fmt.Errorf("invalid 3D mesh vertex index: %d", index)
	}
	return model3d.XYZ(positions[base], positions[base+1], positions[base+2]), nil
}

func jsFloat64Slice(value js.Value) []float64 {
	length := value.Get("length").Int()
	result := make([]float64, length)
	for i := 0; i < length; i++ {
		result[i] = value.Index(i).Float()
	}
	return result
}

func jsIntSlice(value js.Value) []int {
	length := value.Get("length").Int()
	result := make([]int, length)
	for i := 0; i < length; i++ {
		result[i] = value.Index(i).Int()
	}
	return result
}

func jsFloat32Array32(values []float32) js.Value {
	arr := js.Global().Get("Float32Array").New(len(values))
	for i, v := range values {
		arr.SetIndex(i, v)
	}
	return arr
}

func jsUint32Array32(values []uint32) js.Value {
	arr := js.Global().Get("Uint32Array").New(len(values))
	for i, v := range values {
		arr.SetIndex(i, v)
	}
	return arr
}

func jsInt32Array32(values []int32) js.Value {
	arr := js.Global().Get("Int32Array").New(len(values))
	for i, v := range values {
		arr.SetIndex(i, v)
	}
	return arr
}

func jsShapeKernelBufferSource(buffer shapekernel.Buffer) (js.Value, error) {
	values := buffer.Constructor()
	switch buffer.WGSLType {
	case "f32":
		return jsFloat32Array32(float32SliceFromBits(values)), nil
	case "u32":
		return jsUint32Array32(values), nil
	case "i32":
		return jsInt32Array32(int32SliceFromUint32(values)), nil
	default:
		return js.Null(), fmt.Errorf("unsupported ShapeKernel buffer WGSL type %q", buffer.WGSLType)
	}
}

func jsErrorString(value js.Value, fallback string) string {
	if value.Type() == js.TypeString {
		return value.String()
	}
	if value.Type() == js.TypeObject {
		msg := value.Get("message")
		if msg.Type() == js.TypeString {
			return msg.String()
		}
	}
	return fallback
}

func logMeshingBackend(method, backend string) {
	msg := js.Global().Get("Object").New()
	msg.Set("type", "log")
	msg.Set("message", fmt.Sprintf("[m3dscad] %s backend: %s", method, backend))
	js.Global().Call("postMessage", msg)
}

func useWebGPUFallbackReason(useWebGPU bool, kernel *shapekernel.ShapeKernel, wantKind shapekernel.ShapeKind) string {
	if !useWebGPU {
		return "CPU fallback (WebGPU disabled)"
	}
	if kernel == nil {
		return "CPU fallback (no kernel available)"
	}
	if kernel.Kind != wantKind {
		return fmt.Sprintf("CPU fallback (kernel kind %v unsupported)", kernel.Kind)
	}
	return "CPU fallback"
}
