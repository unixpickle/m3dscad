package scad

import (
	"errors"
	"fmt"
	"testing"
)

func TestWithPosStacksPositions(t *testing.T) {
	base := errors.New("boom")
	err := WithPos(base, Pos{Line: 2, Col: 3})
	err = WithPos(err, Pos{Line: 1, Col: 7})

	var perr *PosError
	if !errors.As(err, &perr) {
		t.Fatalf("expected PosError, got %T", err)
	}
	if len(perr.Positions) != 2 {
		t.Fatalf("expected 2 positions, got %d", len(perr.Positions))
	}
	if perr.Positions[0] != (Pos{Line: 1, Col: 7}) {
		t.Fatalf("unexpected outer position: %+v", perr.Positions[0])
	}
	if perr.Positions[1] != (Pos{Line: 2, Col: 3}) {
		t.Fatalf("unexpected inner position: %+v", perr.Positions[1])
	}
	if !errors.Is(err, base) {
		t.Fatalf("expected wrapped base error")
	}
	if got, want := err.Error(), "1:7: 2:3: boom"; got != want {
		t.Fatalf("unexpected error string: got %q want %q", got, want)
	}
}

func TestPosErrorfWrapThenWithPos(t *testing.T) {
	inner := PosErrorf(Pos{Line: 3, Col: 9}, "parse failed: %w", fmt.Errorf("bad token"))
	err := WithPos(inner, Pos{Line: 1, Col: 1})
	if got, want := err.Error(), "1:1: 3:9: parse failed: bad token"; got != want {
		t.Fatalf("unexpected error string: got %q want %q", got, want)
	}
}
