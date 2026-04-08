package main

import (
	"errors"
	"testing"
)

func TestPanicMessage(t *testing.T) {
	testCases := []struct {
		Name     string
		Value    any
		Expected string
	}{
		{
			Name:     "String",
			Value:    "boom",
			Expected: "boom",
		},
		{
			Name:     "Error",
			Value:    errors.New("kaboom"),
			Expected: "kaboom",
		},
		{
			Name:     "Nil",
			Value:    nil,
			Expected: "panic: <nil>",
		},
		{
			Name:     "Number",
			Value:    42,
			Expected: "42",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if actual := panicMessage(tc.Value); actual != tc.Expected {
				t.Fatalf("expected %q but got %q", tc.Expected, actual)
			}
		})
	}
}
