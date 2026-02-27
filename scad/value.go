package scad

import (
	"fmt"
)

type ValueKind int

const (
	ValNull ValueKind = iota
	ValNum
	ValBool
	ValString
	ValList
)

type Value struct {
	Kind ValueKind
	Num  float64
	Bool bool
	Str  string
	List []Value
}

func Num(v float64) Value   { return Value{Kind: ValNum, Num: v} }
func Bool(v bool) Value     { return Value{Kind: ValBool, Bool: v} }
func String(v string) Value { return Value{Kind: ValString, Str: v} }
func List(v []Value) Value  { return Value{Kind: ValList, List: v} }

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
