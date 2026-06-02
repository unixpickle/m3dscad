package main

import (
	"fmt"
	"math"
)

func shapeKernelBindingWGSLType(elemType string) (string, error) {
	switch elemType {
	case "f32", "u32", "i32":
		return fmt.Sprintf("array<%s>", elemType), nil
	default:
		return "", fmt.Errorf("unsupported ShapeKernel buffer WGSL type %q", elemType)
	}
}

func float32SliceFromBits(values []uint32) []float32 {
	result := make([]float32, len(values))
	for i, x := range values {
		result[i] = math.Float32frombits(x)
	}
	return result
}

func int32SliceFromUint32(values []uint32) []int32 {
	result := make([]int32, len(values))
	for i, x := range values {
		result[i] = int32(x)
	}
	return result
}
