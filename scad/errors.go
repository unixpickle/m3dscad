package scad

import (
	"fmt"
	"strings"
)

// PosError is an error annotated with one or more source positions.
type PosError struct {
	Positions []Pos
	Err       error
}

func (p *PosError) Error() string {
	if p == nil {
		return "<nil>"
	}
	if len(p.Positions) == 0 {
		if p.Err == nil {
			return ""
		}
		return p.Err.Error()
	}

	var b strings.Builder
	for i, pos := range p.Positions {
		if i > 0 {
			b.WriteString(": ")
		}
		b.WriteString(pos.String())
	}
	if p.Err != nil {
		b.WriteString(": ")
		b.WriteString(p.Err.Error())
	}
	return b.String()
}

func (p *PosError) Unwrap() error {
	if p == nil {
		return nil
	}
	return p.Err
}

// WithPos annotates err with pos. If err is already a PosError, the position is prepended.
func WithPos(err error, pos Pos) error {
	if err == nil {
		return nil
	}
	if p, ok := err.(*PosError); ok {
		positions := append([]Pos{pos}, p.Positions...)
		return &PosError{Positions: positions, Err: p.Err}
	}
	return &PosError{Positions: []Pos{pos}, Err: err}
}

// PosErrorf creates an error with fmt.Errorf and annotates it with pos.
func PosErrorf(pos Pos, format string, args ...any) error {
	return WithPos(fmt.Errorf(format, args...), pos)
}
