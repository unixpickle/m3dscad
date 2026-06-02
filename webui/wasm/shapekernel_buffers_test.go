package main

import (
	"math"
	"testing"
)

func TestShapeKernelBindingWGSLType(t *testing.T) {
	testCases := []struct {
		name      string
		elemType  string
		expected  string
		expectErr bool
	}{
		{
			name:     "Float32",
			elemType: "f32",
			expected: "array<f32>",
		},
		{
			name:     "Uint32",
			elemType: "u32",
			expected: "array<u32>",
		},
		{
			name:     "Int32",
			elemType: "i32",
			expected: "array<i32>",
		},
		{
			name:      "Unsupported",
			elemType:  "vec4<f32>",
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := shapeKernelBindingWGSLType(tc.elemType)
			if tc.expectErr {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if actual != tc.expected {
				t.Fatalf("expected %q but got %q", tc.expected, actual)
			}
		})
	}
}

func TestFloat32SliceFromBits(t *testing.T) {
	values := []uint32{
		math.Float32bits(1.25),
		math.Float32bits(float32(-3.5)),
		math.Float32bits(float32(math.Inf(1))),
	}
	expected := []float32{1.25, -3.5, float32(math.Inf(1))}

	actual := float32SliceFromBits(values)
	if len(actual) != len(expected) {
		t.Fatalf("expected %d values but got %d", len(expected), len(actual))
	}
	for i, x := range expected {
		if actual[i] != x {
			t.Fatalf("expected value %d to be %v but got %v", i, x, actual[i])
		}
	}
}

func TestInt32SliceFromUint32(t *testing.T) {
	values := []uint32{0, 17, uint32(math.MaxUint32), uint32(math.MaxInt32) + 1}
	expected := []int32{0, 17, -1, math.MinInt32}

	actual := int32SliceFromUint32(values)
	if len(actual) != len(expected) {
		t.Fatalf("expected %d values but got %d", len(expected), len(actual))
	}
	for i, x := range expected {
		if actual[i] != x {
			t.Fatalf("expected value %d to be %v but got %v", i, x, actual[i])
		}
	}
}
