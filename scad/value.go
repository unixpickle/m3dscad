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
	ValEach
	ValList
)

type Value struct {
	Kind ValueKind
	Num  float64
	Bool bool
	Str  string
	Rng  Range
	Each *Value
	List []Value
}

func Num(v float64) Value   { return Value{Kind: ValNum, Num: v} }
func Bool(v bool) Value     { return Value{Kind: ValBool, Bool: v} }
func String(v string) Value { return Value{Kind: ValString, Str: v} }
func RangeValue(v Range) Value {
	return Value{Kind: ValRange, Rng: v}
}
func EachValue(v Value) Value {
	vCopy := v
	return Value{Kind: ValEach, Each: &vCopy}
}
func List(v []Value) Value { return Value{Kind: ValList, List: v} }

type Range struct {
	Start float64
	End   float64
	Step  float64
}

func (v Value) AsNum() (float64, error) {
	if v.Kind != ValNum {
		return 0, fmt.Errorf("expected number")
	}
	return v.Num, nil
}

func (v Value) AsBool() (bool, error) {
	if v.Kind != ValBool {
		return false, fmt.Errorf("expected bool")
	}
	return v.Bool, nil
}

func (v Value) AsString() (string, error) {
	if v.Kind != ValString {
		return "", fmt.Errorf("expected string")
	}
	return v.Str, nil
}

func (v Value) AsVec3() ([3]float64, error) {
	// Accept [x,y,z] or [x,y] (z=0) or scalar -> [s,s,s]
	if v.Kind == ValNum {
		return [3]float64{v.Num, v.Num, v.Num}, nil
	}
	if v.Kind != ValList {
		return [3]float64{}, fmt.Errorf("expected vector/list")
	}
	if len(v.List) == 0 {
		return [3]float64{}, fmt.Errorf("expected non-empty vector")
	}
	var out [3]float64
	for i := 0; i < 3; i++ {
		if i < len(v.List) {
			n, err := v.List[i].AsNum()
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

func (v Value) AsVec2() ([2]float64, error) {
	// Accept [x,y] or [x] (y=0) or scalar -> [s,s]
	if v.Kind == ValNum {
		return [2]float64{v.Num, v.Num}, nil
	}
	if v.Kind != ValList {
		return [2]float64{}, fmt.Errorf("expected vector/list")
	}
	if len(v.List) == 0 {
		return [2]float64{}, fmt.Errorf("expected non-empty vector")
	}
	var out [2]float64
	for i := 0; i < 2; i++ {
		if i < len(v.List) {
			n, err := v.List[i].AsNum()
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

func (v Value) IterableElems() ([]Value, error) {
	switch v.Kind {
	case ValList:
		return append([]Value(nil), v.List...), nil
	case ValRange:
		return v.Rng.Values()
	case ValEach:
		if v.Each == nil {
			return nil, fmt.Errorf("invalid each value")
		}
		return v.Each.IterableElems()
	default:
		return nil, fmt.Errorf("expected vector or range")
	}
}

func (v Value) ElemAt(idx int) (Value, error) {
	if idx < 0 {
		return Value{}, fmt.Errorf("index out of range")
	}
	switch v.Kind {
	case ValList:
		if idx >= len(v.List) {
			return Value{}, fmt.Errorf("index out of range")
		}
		return v.List[idx], nil
	case ValRange:
		return v.Rng.ValueAt(idx)
	case ValEach:
		if v.Each == nil {
			return Value{}, fmt.Errorf("invalid each value")
		}
		return v.Each.ElemAt(idx)
	default:
		return Value{}, fmt.Errorf("expected vector or range")
	}
}

func (v Value) Len() (int, error) {
	switch v.Kind {
	case ValList:
		return len(v.List), nil
	case ValRange:
		return v.Rng.Len()
	case ValEach:
		if v.Each == nil {
			return 0, fmt.Errorf("invalid each value")
		}
		return v.Each.Len()
	default:
		return 0, fmt.Errorf("expected vector or range")
	}
}

func (r Range) Values() ([]Value, error) {
	if r.Step == 0 {
		return nil, fmt.Errorf("range step cannot be zero")
	}
	var out []Value
	const eps = 1e-9
	cur := r.Start
	if r.Step > 0 {
		for cur <= r.End+eps {
			out = append(out, Num(cur))
			cur += r.Step
			if len(out) > 1_000_000 {
				return nil, fmt.Errorf("range produced too many elements")
			}
		}
	} else {
		for cur >= r.End-eps {
			out = append(out, Num(cur))
			cur += r.Step
			if len(out) > 1_000_000 {
				return nil, fmt.Errorf("range produced too many elements")
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

func (r Range) ValueAt(idx int) (Value, error) {
	if idx < 0 {
		return Value{}, fmt.Errorf("index out of range")
	}
	if r.Step == 0 {
		return Value{}, fmt.Errorf("range step cannot be zero")
	}
	const eps = 1e-9
	val := r.Start + r.Step*float64(idx)
	if r.Step > 0 {
		if val > r.End+eps {
			return Value{}, fmt.Errorf("index out of range")
		}
	} else {
		if val < r.End-eps {
			return Value{}, fmt.Errorf("index out of range")
		}
	}
	if math.Abs(val-r.End) < eps {
		val = r.End
	}
	return Num(val), nil
}

func (r Range) Len() (int, error) {
	if r.Step == 0 {
		return 0, fmt.Errorf("range step cannot be zero")
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

func (v Value) Equal(other Value) bool {
	if v.Kind != other.Kind {
		return false
	}
	switch v.Kind {
	case ValNull:
		return true
	case ValNum:
		return v.Num == other.Num
	case ValBool:
		return v.Bool == other.Bool
	case ValString:
		return v.Str == other.Str
	case ValList:
		if len(v.List) != len(other.List) {
			return false
		}
		for i := range v.List {
			if !v.List[i].Equal(other.List[i]) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// CompareOrder returns ordering for relational operators.
// It returns (-1, true), (0, true), or (1, true) when comparable;
// otherwise (0, false).
func (v Value) CompareOrder(other Value) (int, bool) {
	switch {
	case v.Kind == ValNum && other.Kind == ValNum:
		return compareFloat(v.Num, other.Num), true
	case v.Kind == ValString && other.Kind == ValString:
		return compareString(v.Str, other.Str), true
	case v.Kind == ValBool && other.Kind == ValBool:
		return compareFloat(boolAsNum(v.Bool), boolAsNum(other.Bool)), true
	case v.Kind == ValBool && other.Kind == ValNum:
		return compareFloat(boolAsNum(v.Bool), other.Num), true
	case v.Kind == ValNum && other.Kind == ValBool:
		return compareFloat(v.Num, boolAsNum(other.Bool)), true
	default:
		return 0, false
	}
}

func boolAsNum(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func compareFloat(a, b float64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareString(a, b string) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}
