package scad

import (
	"fmt"
	"math"
)

type ValueKind int

const (
	ValNull ValueKind = iota
	ValNum
	ValBool
	ValString
	ValRange
	ValList
)

type Value struct {
	Kind ValueKind
	Num  float64
	Bool bool
	Str  string
	Rng  Range
	List []Value
}

func Num(v float64) Value   { return Value{Kind: ValNum, Num: v} }
func Bool(v bool) Value     { return Value{Kind: ValBool, Bool: v} }
func String(v string) Value { return Value{Kind: ValString, Str: v} }
func RangeValue(v Range) Value {
	return Value{Kind: ValRange, Rng: v}
}
func List(v []Value) Value { return Value{Kind: ValList, List: v} }

type Range struct {
	Start float64
	End   float64
	Step  float64
}

func (v Value) AsNum(pos Pos) (float64, error) {
	if v.Kind != ValNum {
		return 0, fmt.Errorf("%v: expected number", pos)
	}
	return v.Num, nil
}

func (v Value) AsBool(pos Pos) (bool, error) {
	if v.Kind != ValBool {
		return false, fmt.Errorf("%v: expected bool", pos)
	}
	return v.Bool, nil
}

func (v Value) AsString(pos Pos) (string, error) {
	if v.Kind != ValString {
		return "", fmt.Errorf("%v: expected string", pos)
	}
	return v.Str, nil
}

func (v Value) AsVec3(pos Pos) ([3]float64, error) {
	// Accept [x,y,z] or [x,y] (z=0) or scalar -> [s,s,s]
	if v.Kind == ValNum {
		return [3]float64{v.Num, v.Num, v.Num}, nil
	}
	if v.Kind != ValList {
		return [3]float64{}, fmt.Errorf("%v: expected vector/list", pos)
	}
	if len(v.List) == 0 {
		return [3]float64{}, fmt.Errorf("%v: expected non-empty vector", pos)
	}
	var out [3]float64
	for i := 0; i < 3; i++ {
		if i < len(v.List) {
			n, err := v.List[i].AsNum(pos)
			if err != nil {
				return [3]float64{}, err
			}
			out[i] = n
		} else {
			out[i] = 0
		}
	}
	return out, nil
}

func (v Value) AsVec2(pos Pos) ([2]float64, error) {
	// Accept [x,y] or [x] (y=0) or scalar -> [s,s]
	if v.Kind == ValNum {
		return [2]float64{v.Num, v.Num}, nil
	}
	if v.Kind != ValList {
		return [2]float64{}, fmt.Errorf("%v: expected vector/list", pos)
	}
	if len(v.List) == 0 {
		return [2]float64{}, fmt.Errorf("%v: expected non-empty vector", pos)
	}
	var out [2]float64
	for i := 0; i < 2; i++ {
		if i < len(v.List) {
			n, err := v.List[i].AsNum(pos)
			if err != nil {
				return [2]float64{}, err
			}
			out[i] = n
		} else {
			out[i] = 0
		}
	}
	return out, nil
}

func (v Value) IterableElems(pos Pos) ([]Value, error) {
	switch v.Kind {
	case ValList:
		return append([]Value(nil), v.List...), nil
	case ValRange:
		return v.Rng.Values(pos)
	default:
		return nil, fmt.Errorf("%v: expected vector or range", pos)
	}
}

func (v Value) ElemAt(idx int, pos Pos) (Value, error) {
	if idx < 0 {
		return Value{}, fmt.Errorf("%v: index out of range", pos)
	}
	switch v.Kind {
	case ValList:
		if idx >= len(v.List) {
			return Value{}, fmt.Errorf("%v: index out of range", pos)
		}
		return v.List[idx], nil
	case ValRange:
		return v.Rng.ValueAt(idx, pos)
	default:
		return Value{}, fmt.Errorf("%v: expected vector or range", pos)
	}
}

func (v Value) Len(pos Pos) (int, error) {
	switch v.Kind {
	case ValList:
		return len(v.List), nil
	case ValRange:
		return v.Rng.Len(pos)
	default:
		return 0, fmt.Errorf("%v: expected vector or range", pos)
	}
}

func (r Range) Values(pos Pos) ([]Value, error) {
	if r.Step == 0 {
		return nil, fmt.Errorf("%v: range step cannot be zero", pos)
	}
	var out []Value
	const eps = 1e-9
	cur := r.Start
	if r.Step > 0 {
		for cur <= r.End+eps {
			out = append(out, Num(cur))
			cur += r.Step
			if len(out) > 1_000_000 {
				return nil, fmt.Errorf("%v: range produced too many elements", pos)
			}
		}
	} else {
		for cur >= r.End-eps {
			out = append(out, Num(cur))
			cur += r.Step
			if len(out) > 1_000_000 {
				return nil, fmt.Errorf("%v: range produced too many elements", pos)
			}
		}
	}
	if len(out) > 0 {
		last := out[len(out)-1].Num
		if math.Abs(last-r.End) < eps {
			out[len(out)-1] = Num(r.End)
		}
	}
	return out, nil
}

func (r Range) ValueAt(idx int, pos Pos) (Value, error) {
	if idx < 0 {
		return Value{}, fmt.Errorf("%v: index out of range", pos)
	}
	if r.Step == 0 {
		return Value{}, fmt.Errorf("%v: range step cannot be zero", pos)
	}
	const eps = 1e-9
	val := r.Start + r.Step*float64(idx)
	if r.Step > 0 {
		if val > r.End+eps {
			return Value{}, fmt.Errorf("%v: index out of range", pos)
		}
	} else {
		if val < r.End-eps {
			return Value{}, fmt.Errorf("%v: index out of range", pos)
		}
	}
	if math.Abs(val-r.End) < eps {
		val = r.End
	}
	return Num(val), nil
}

func (r Range) Len(pos Pos) (int, error) {
	if r.Step == 0 {
		return 0, fmt.Errorf("%v: range step cannot be zero", pos)
	}
	const eps = 1e-9
	if r.Step > 0 {
		if r.Start > r.End+eps {
			return 0, nil
		}
		n := int(math.Floor((r.End-r.Start)/r.Step+eps)) + 1
		if n < 0 {
			return 0, nil
		}
		return n, nil
	}
	if r.Start < r.End-eps {
		return 0, nil
	}
	step := -r.Step
	n := int(math.Floor((r.Start-r.End)/step+eps)) + 1
	if n < 0 {
		return 0, nil
	}
	return n, nil
}
