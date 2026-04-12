package scad

import (
	"fmt"
	"math"
)

func evalUnaryArithmetic(op TokenKind, v Value) (Value, error) {
	switch op {
	case TokMinus:
		if v.Kind == ValList {
			return negateList(v)
		}
		n, err := v.AsNum()
		if err != nil {
			return Value{}, err
		}
		return Num(-n), nil
	case TokPlus:
		n, err := v.AsNum()
		if err != nil {
			return Value{}, err
		}
		return Num(n), nil
	default:
		return Value{}, fmt.Errorf("unknown unary arithmetic op")
	}
}

func evalBinaryArithmetic(op TokenKind, lv, rv Value) (Value, error) {
	if lv.Kind == ValNum && rv.Kind == ValNum {
		return evalNumericBinary(op, lv.Num, rv.Num), nil
	}

	switch op {
	case TokPlus, TokMinus:
		if lv.Kind == ValList && rv.Kind == ValList {
			return evalRecursiveListBinary(op, lv, rv)
		}
	case TokStar:
		switch {
		case lv.Kind == ValList && rv.Kind == ValNum:
			return evalRecursiveListScalar(op, lv, rv.Num)
		case lv.Kind == ValNum && rv.Kind == ValList:
			return evalRecursiveListScalar(op, rv, lv.Num)
		case lv.Kind == ValList && rv.Kind == ValList:
			return evalListMultiply(lv, rv)
		}
	case TokSlash:
		if lv.Kind == ValList && rv.Kind == ValNum {
			return evalRecursiveListScalar(op, lv, rv.Num)
		}
	}

	a, err := lv.AsNum()
	if err != nil {
		return Value{}, err
	}
	b, err := rv.AsNum()
	if err != nil {
		return Value{}, err
	}
	return evalNumericBinary(op, a, b), nil
}

func evalNumericBinary(op TokenKind, a, b float64) Value {
	switch op {
	case TokPlus:
		return Num(a + b)
	case TokMinus:
		return Num(a - b)
	case TokStar:
		return Num(a * b)
	case TokSlash:
		return Num(a / b)
	case TokPercent:
		return Num(math.Mod(a, b))
	case TokCaret:
		return Num(math.Pow(a, b))
	default:
		panic("unsupported numeric arithmetic op")
	}
}

func negateList(v Value) (Value, error) {
	if v.Kind != ValList {
		return Value{}, fmt.Errorf("prefix - on vectors requires numeric elements")
	}
	out := make([]Value, len(v.List))
	for i, elem := range v.List {
		switch elem.Kind {
		case ValNum:
			out[i] = Num(-elem.Num)
		case ValList:
			child, err := negateList(elem)
			if err != nil {
				return Value{}, err
			}
			out[i] = child
		default:
			return Value{}, fmt.Errorf("prefix - on vectors requires numeric elements")
		}
	}
	return List(out), nil
}

func evalRecursiveListScalar(op TokenKind, v Value, scalar float64) (Value, error) {
	if v.Kind != ValList {
		return Value{}, fmt.Errorf("operator %s on vectors requires numeric elements", arithmeticOpString(op))
	}
	out := make([]Value, len(v.List))
	for i, elem := range v.List {
		switch elem.Kind {
		case ValNum:
			if op == TokStar {
				out[i] = Num(elem.Num * scalar)
			} else {
				out[i] = Num(elem.Num / scalar)
			}
		case ValList:
			child, err := evalRecursiveListScalar(op, elem, scalar)
			if err != nil {
				return Value{}, err
			}
			out[i] = child
		default:
			return Value{}, fmt.Errorf("operator %s on vectors requires numeric elements", arithmeticOpString(op))
		}
	}
	return List(out), nil
}

func evalRecursiveListBinary(op TokenKind, lv, rv Value) (Value, error) {
	if lv.Kind == ValNum && rv.Kind == ValNum {
		return evalNumericBinary(op, lv.Num, rv.Num), nil
	}
	if lv.Kind != ValList || rv.Kind != ValList {
		return Value{}, fmt.Errorf("operator %s on vectors requires matching numeric/list structure", arithmeticOpString(op))
	}
	n := len(lv.List)
	if len(rv.List) < n {
		n = len(rv.List)
	}
	out := make([]Value, n)
	for i := 0; i < n; i++ {
		child, err := evalRecursiveListBinary(op, lv.List[i], rv.List[i])
		if err != nil {
			return Value{}, err
		}
		out[i] = child
	}
	return List(out), nil
}

func evalListMultiply(lv, rv Value) (Value, error) {
	if left, ok := asSimpleVector(lv); ok {
		if right, ok := asSimpleVector(rv); ok {
			if len(left) != len(right) {
				return Value{}, fmt.Errorf("vector dot product requires equal-length vectors")
			}
			sum := 0.0
			for i, x := range left {
				sum += x * right[i]
			}
			return Num(sum), nil
		}
	}
	if left, ok := asMatrix(lv); ok {
		if right, ok := asMatrix(rv); ok {
			if len(left[0]) != len(right) {
				return Value{}, fmt.Errorf("matrix product dimension mismatch")
			}
			return matrixToValue(multiplyMatrices(left, right)), nil
		}
		if right, ok := asSimpleVector(rv); ok {
			if len(left[0]) != len(right) {
				return Value{}, fmt.Errorf("matrix/vector product dimension mismatch")
			}
			return vectorToValue(multiplyMatrixVector(left, right)), nil
		}
	}
	if left, ok := asSimpleVector(lv); ok {
		if right, ok := asMatrix(rv); ok {
			if len(left) != len(right) {
				return Value{}, fmt.Errorf("vector/matrix product dimension mismatch")
			}
			return vectorToValue(multiplyVectorMatrix(left, right)), nil
		}
	}
	return Value{}, fmt.Errorf("operator * on lists requires vectors or matrices of numbers")
}

func asSimpleVector(v Value) ([]float64, bool) {
	if v.Kind != ValList {
		return nil, false
	}
	out := make([]float64, len(v.List))
	for i, elem := range v.List {
		if elem.Kind != ValNum {
			return nil, false
		}
		out[i] = elem.Num
	}
	return out, true
}

func asMatrix(v Value) ([][]float64, bool) {
	if v.Kind != ValList || len(v.List) == 0 {
		return nil, false
	}
	out := make([][]float64, len(v.List))
	rowLen := -1
	for i, rowVal := range v.List {
		row, ok := asSimpleVector(rowVal)
		if !ok {
			return nil, false
		}
		if rowLen == -1 {
			rowLen = len(row)
		} else if len(row) != rowLen {
			return nil, false
		}
		out[i] = row
	}
	return out, true
}

func multiplyMatrices(left, right [][]float64) [][]float64 {
	out := make([][]float64, len(left))
	for i := range left {
		row := make([]float64, len(right[0]))
		for j := range row {
			sum := 0.0
			for k, x := range left[i] {
				sum += x * right[k][j]
			}
			row[j] = sum
		}
		out[i] = row
	}
	return out
}

func multiplyMatrixVector(left [][]float64, right []float64) []float64 {
	out := make([]float64, len(left))
	for i := range left {
		sum := 0.0
		for k, x := range left[i] {
			sum += x * right[k]
		}
		out[i] = sum
	}
	return out
}

func multiplyVectorMatrix(left []float64, right [][]float64) []float64 {
	out := make([]float64, len(right[0]))
	for j := range out {
		sum := 0.0
		for k, x := range left {
			sum += x * right[k][j]
		}
		out[j] = sum
	}
	return out
}

func vectorToValue(v []float64) Value {
	out := make([]Value, len(v))
	for i, x := range v {
		out[i] = Num(x)
	}
	return List(out)
}

func matrixToValue(m [][]float64) Value {
	out := make([]Value, len(m))
	for i, row := range m {
		out[i] = vectorToValue(row)
	}
	return List(out)
}

func arithmeticOpString(op TokenKind) string {
	switch op {
	case TokPlus:
		return "+"
	case TokMinus:
		return "-"
	case TokStar:
		return "*"
	case TokSlash:
		return "/"
	case TokPercent:
		return "%"
	case TokCaret:
		return "^"
	default:
		return "?"
	}
}
