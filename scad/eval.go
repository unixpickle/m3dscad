package scad

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"
)

type moduleDef struct {
	Params []Param
	Body   *BlockStmt
}

type funcDef struct {
	Params []Param
	Body   Expr
}

type env struct {
	scopes []map[string]Value
	mods   map[string]moduleDef
	funcs  map[string]funcDef
}

func newEnv() *env {
	return &env{
		scopes: []map[string]Value{{}},
		mods:   map[string]moduleDef{},
		funcs:  map[string]funcDef{},
	}
}

func (e *env) push() { e.scopes = append(e.scopes, map[string]Value{}) }
func (e *env) pop()  { e.scopes = e.scopes[:len(e.scopes)-1] }

func (e *env) set(name string, v Value) { e.scopes[len(e.scopes)-1][name] = v }

func (e *env) get(name string) (Value, bool) {
	for i := len(e.scopes) - 1; i >= 0; i-- {
		if v, ok := e.scopes[i][name]; ok {
			return v, true
		}
	}
	return Value{}, false
}

func Parse(src string) (*Program, error) {
	p, err := NewParser(src)
	if err != nil {
		return nil, err
	}
	return p.ParseProgram()
}

func Eval(p *Program) (ShapeRep, error) {
	e := newEnv()
	solids, err := evalStmts(e, p.Stmts)
	if err != nil {
		return ShapeRep{}, err
	}
	merged, err := unionAll(solids)
	if err != nil {
		return ShapeRep{}, err
	}
	return merged, nil
}

func evalStmts(e *env, ss []Stmt) ([]ShapeRep, error) {
	var out []ShapeRep
	for _, s := range ss {
		got, err := evalStmt(e, s)
		if err != nil {
			return nil, err
		}
		if got != nil {
			out = append(out, *got)
		}
	}
	return out, nil
}

func evalStmt(e *env, s Stmt) (*ShapeRep, error) {
	switch st := s.(type) {
	case *AssignStmt:
		v, err := evalExpr(e, st.Expr)
		if err != nil {
			return nil, err
		}
		e.set(st.Name, v)
		return nil, nil

	case *BlockStmt:
		e.push()
		defer e.pop()
		return evalStmtsAsOne(e, st.Stmts)

	case *IfStmt:
		condV, err := evalExpr(e, st.Cond)
		if err != nil {
			return nil, err
		}
		b, err := condV.AsBool(st.Cond.pos())
		if err != nil {
			return nil, err
		}
		if b {
			e.push()
			defer e.pop()
			return evalStmt(e, st.Then)
		}
		if st.Else != nil {
			e.push()
			defer e.pop()
			return evalStmt(e, st.Else)
		}
		return nil, nil

	case *ForStmt:
		return evalForStmt(e, st)

	case *ModuleDefStmt:
		e.mods[st.Name] = moduleDef{Params: st.Params, Body: st.Body}
		return nil, nil

	case *FuncDefStmt:
		e.funcs[st.Name] = funcDef{Params: st.Params, Body: st.Body}
		return nil, nil

	case *CallStmt:
		return evalCallStmt(e, st)

	default:
		return nil, fmt.Errorf("%v: unknown stmt type", s.pos())
	}
}

func evalCallStmt(e *env, st *CallStmt) (*ShapeRep, error) {
	name := st.Call.Name

	if handler, ok := builtinHandlers[name]; ok {
		if len(st.Children) == 0 && handler.RequireChildren {
			return nil, fmt.Errorf("%v: %s() requires children", st.pos(), name)
		}
		if len(st.Children) > 0 && !handler.AllowChildren {
			return nil, fmt.Errorf("%v: %s() does not take children", st.pos(), name)
		}
		var children []ShapeRep
		var childUnion *ShapeRep
		if len(st.Children) > 0 {
			var err error
			children, err = evalStmts(e, st.Children)
			if err != nil {
				return nil, err
			}
			if handler.NeedsChildUnion {
				u, err := unionAll(children)
				if err != nil {
					return nil, err
				}
				childUnion = &u
			}
		}
		res, err := handler.Eval(e, st, children, childUnion)
		if err != nil {
			return nil, fmt.Errorf("%v: %w", st.pos(), err)
		}
		return &res, nil
	}

	// User-defined module call (solids)
	if md, ok := e.mods[name]; ok {
		if len(st.Children) > 0 {
			return nil, fmt.Errorf("%v: module %s does not support children in this MVP", st.pos(), name)
		}
		e.push()
		defer e.pop()

		if err := bindParams(e, md.Params, st.Call.Args); err != nil {
			return nil, err
		}
		solids, err := evalStmts(e, md.Body.Stmts)
		if err != nil {
			return nil, err
		}
		u, err := unionAll(solids)
		if err != nil {
			return nil, err
		}
		return &u, nil
	}

	return nil, fmt.Errorf("%v: unknown module/primitive %q", st.pos(), name)
}

func evalStmtsAsOne(e *env, ss []Stmt) (*ShapeRep, error) {
	solids, err := evalStmts(e, ss)
	if err != nil {
		return nil, err
	}
	if len(solids) == 0 {
		return nil, nil
	}
	u, err := unionAll(solids)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func evalExpr(e *env, ex Expr) (Value, error) {
	switch x := ex.(type) {
	case *NumberLit:
		return Num(x.V), nil
	case *BoolLit:
		return Bool(x.V), nil
	case *StringLit:
		return String(x.V), nil
	case *VarExpr:
		v, ok := e.get(x.Name)
		if !ok {
			return Value{}, fmt.Errorf("%v: undefined variable %q", x.pos(), x.Name)
		}
		return v, nil
	case *ArrayLit:
		var vals []Value
		for _, el := range x.Elems {
			v, err := evalExpr(e, el)
			if err != nil {
				return Value{}, err
			}
			if _, ok := el.(*ForExpr); ok {
				elems, err := v.IterableElems(el.pos())
				if err != nil {
					return Value{}, err
				}
				vals = append(vals, elems...)
				continue
			}
			if v.Kind == ValEach {
				elems, err := v.IterableElems(el.pos())
				if err != nil {
					return Value{}, err
				}
				vals = append(vals, elems...)
				continue
			}
			vals = append(vals, v)
		}
		return List(vals), nil
	case *RangeLit:
		startV, err := evalExpr(e, x.Start)
		if err != nil {
			return Value{}, err
		}
		endV, err := evalExpr(e, x.End)
		if err != nil {
			return Value{}, err
		}
		start, err := startV.AsNum(x.Start.pos())
		if err != nil {
			return Value{}, err
		}
		end, err := endV.AsNum(x.End.pos())
		if err != nil {
			return Value{}, err
		}
		step := 1.0
		if end < start {
			step = -1.0
		}
		if x.Step != nil {
			stepV, err := evalExpr(e, x.Step)
			if err != nil {
				return Value{}, err
			}
			step, err = stepV.AsNum(x.Step.pos())
			if err != nil {
				return Value{}, err
			}
		}
		return RangeValue(Range{
			Start: start,
			End:   end,
			Step:  step,
		}), nil
	case *IndexExpr:
		base, err := evalExpr(e, x.X)
		if err != nil {
			return Value{}, err
		}
		idxV, err := evalExpr(e, x.Index)
		if err != nil {
			return Value{}, err
		}
		idxNum, err := idxV.AsNum(x.Index.pos())
		if err != nil {
			return Value{}, err
		}
		idx := int(idxNum)
		if float64(idx) != idxNum {
			return Value{}, fmt.Errorf("%v: index must be an integer", x.Index.pos())
		}
		return base.ElemAt(idx, x.Index.pos())
	case *ForExpr:
		var out []Value
		err := evalForBindsExpr(e, x.Binds, 0, func() error {
			v, err := evalExpr(e, x.Body)
			if err != nil {
				return err
			}
			if v.Kind == ValEach {
				elems, err := v.IterableElems(x.Body.pos())
				if err != nil {
					return err
				}
				out = append(out, elems...)
				return nil
			}
			out = append(out, v)
			return nil
		})
		if err != nil {
			return Value{}, err
		}
		return List(out), nil
	case *LetExpr:
		e.push()
		defer e.pop()
		for _, b := range x.Binds {
			v, err := evalExpr(e, b.Expr)
			if err != nil {
				return Value{}, err
			}
			e.set(b.Name, v)
		}
		return evalExpr(e, x.Body)
	case *EachExpr:
		v, err := evalExpr(e, x.X)
		if err != nil {
			return Value{}, err
		}
		return EachValue(v), nil
	case *UnaryExpr:
		v, err := evalExpr(e, x.X)
		if err != nil {
			return Value{}, err
		}
		switch x.Op {
		case TokNot:
			b, err := v.AsBool(x.pos())
			if err != nil {
				return Value{}, err
			}
			return Bool(!b), nil
		case TokMinus:
			n, err := v.AsNum(x.pos())
			if err != nil {
				return Value{}, err
			}
			return Num(-n), nil
		case TokPlus:
			n, err := v.AsNum(x.pos())
			if err != nil {
				return Value{}, err
			}
			return Num(n), nil
		default:
			return Value{}, fmt.Errorf("%v: unknown unary op", x.pos())
		}
	case *BinaryExpr:
		lv, err := evalExpr(e, x.L)
		if err != nil {
			return Value{}, err
		}
		rv, err := evalExpr(e, x.R)
		if err != nil {
			return Value{}, err
		}

		// boolean ops short-circuit not implemented in MVP (easy to add).
		switch x.Op {
		case TokPlus, TokMinus, TokStar, TokSlash, TokPercent, TokCaret:
			a, err := lv.AsNum(x.pos())
			if err != nil {
				return Value{}, err
			}
			b, err := rv.AsNum(x.pos())
			if err != nil {
				return Value{}, err
			}
			switch x.Op {
			case TokPlus:
				return Num(a + b), nil
			case TokMinus:
				return Num(a - b), nil
			case TokStar:
				return Num(a * b), nil
			case TokSlash:
				return Num(a / b), nil
			case TokPercent:
				return Num(math.Mod(a, b)), nil
			case TokCaret:
				return Num(math.Pow(a, b)), nil
			}
		case TokEq, TokNeq, TokLt, TokLte, TokGt, TokGte:
			a, err := lv.AsNum(x.pos())
			if err != nil {
				return Value{}, err
			}
			b, err := rv.AsNum(x.pos())
			if err != nil {
				return Value{}, err
			}
			switch x.Op {
			case TokEq:
				return Bool(a == b), nil
			case TokNeq:
				return Bool(a != b), nil
			case TokLt:
				return Bool(a < b), nil
			case TokLte:
				return Bool(a <= b), nil
			case TokGt:
				return Bool(a > b), nil
			case TokGte:
				return Bool(a >= b), nil
			}
		case TokAnd, TokOr:
			a, err := lv.AsBool(x.pos())
			if err != nil {
				return Value{}, err
			}
			b, err := rv.AsBool(x.pos())
			if err != nil {
				return Value{}, err
			}
			if x.Op == TokAnd {
				return Bool(a && b), nil
			}
			return Bool(a || b), nil
		default:
			return Value{}, fmt.Errorf("%v: unknown binary op", x.pos())
		}
	case *TernaryExpr:
		cv, err := evalExpr(e, x.Cond)
		if err != nil {
			return Value{}, err
		}
		b, err := cv.AsBool(x.Cond.pos())
		if err != nil {
			return Value{}, err
		}
		if b {
			return evalExpr(e, x.Then)
		}
		return evalExpr(e, x.Else)
	case *CallExpr:
		return evalFuncCall(e, x.Call)
	default:
		return Value{}, fmt.Errorf("%v: unknown expr type", ex.pos())
	}
	return Value{}, fmt.Errorf("%v: unreachable", ex.pos())
}

func evalForStmt(e *env, st *ForStmt) (*ShapeRep, error) {
	var results []ShapeRep
	err := evalForBindsExpr(e, st.Binds, 0, func() error {
		res, err := evalStmt(e, st.Body)
		if err != nil {
			return err
		}
		if res != nil {
			results = append(results, *res)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	var merged ShapeRep
	if st.Intersection {
		merged, err = intersectAll(results)
	} else {
		merged, err = unionAll(results)
	}
	if err != nil {
		return nil, err
	}
	return &merged, nil
}

func evalForBindsExpr(e *env, binds []ForBind, idx int, fn func() error) error {
	if idx == len(binds) {
		return fn()
	}
	b := binds[idx]
	iterVal, err := evalExpr(e, b.Expr)
	if err != nil {
		return err
	}
	elems, err := iterVal.IterableElems(b.P)
	if err != nil {
		return err
	}
	for _, val := range elems {
		e.push()
		e.set(b.Name, val)
		err := evalForBindsExpr(e, binds, idx+1, fn)
		e.pop()
		if err != nil {
			return err
		}
	}
	return nil
}

func evalFuncCall(e *env, c Call) (Value, error) {
	switch c.Name {
	case "len":
		if len(c.Args) != 1 {
			return Value{}, fmt.Errorf("%v: len() takes exactly 1 argument", c.P)
		}
		arg0, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		n, err := arg0.Len(c.P)
		if err != nil {
			return Value{}, err
		}
		return Num(float64(n)), nil
	case "concat":
		if len(c.Args) == 0 {
			return Value{}, fmt.Errorf("%v: concat() needs at least 1 argument", c.P)
		}
		var out []Value
		for _, a := range c.Args {
			v, err := evalExpr(e, a.Expr)
			if err != nil {
				return Value{}, err
			}
			elems, err := v.IterableElems(c.P)
			if err != nil {
				return Value{}, err
			}
			out = append(out, elems...)
		}
		return List(out), nil
	case "sin":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Sin(x * math.Pi / 180)), nil
	case "cos":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Cos(x * math.Pi / 180)), nil
	case "tan":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Tan(x * math.Pi / 180)), nil
	case "asin":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Asin(x) * 180 / math.Pi), nil
	case "acos":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Acos(x) * 180 / math.Pi), nil
	case "atan":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Atan(x) * 180 / math.Pi), nil
	case "atan2":
		y, x, err := evalBinaryNumericFuncArgs(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Atan2(y, x) * 180 / math.Pi), nil
	case "sign":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		if x > 0 {
			return Num(1), nil
		}
		if x < 0 {
			return Num(-1), nil
		}
		return Num(0), nil
	case "floor":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Floor(x)), nil
	case "round":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Round(x)), nil
	case "ceil":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Ceil(x)), nil
	case "ln":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Log(x)), nil
	case "log":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Log10(x)), nil
	case "sqrt":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Sqrt(x)), nil
	case "exp":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Exp(x)), nil
	case "abs":
		x, err := evalUnaryNumericFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Abs(x)), nil
	case "pow":
		x, y, err := evalBinaryNumericFuncArgs(e, c)
		if err != nil {
			return Value{}, err
		}
		return Num(math.Pow(x, y)), nil
	case "min":
		xs, err := evalMinMaxArgs(e, c)
		if err != nil {
			return Value{}, err
		}
		m := xs[0]
		for _, x := range xs[1:] {
			m = math.Min(m, x)
		}
		return Num(m), nil
	case "max":
		xs, err := evalMinMaxArgs(e, c)
		if err != nil {
			return Value{}, err
		}
		m := xs[0]
		for _, x := range xs[1:] {
			m = math.Max(m, x)
		}
		return Num(m), nil
	case "norm":
		if len(c.Args) != 1 {
			return Value{}, fmt.Errorf("%v: norm() needs exactly 1 argument", c.P)
		}
		v, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		xs, err := iterableAsNums(v, c.P)
		if err != nil {
			return Value{}, err
		}
		sum := 0.0
		for _, x := range xs {
			sum += x * x
		}
		return Num(math.Sqrt(sum)), nil
	case "cross":
		if len(c.Args) != 2 {
			return Value{}, fmt.Errorf("%v: cross() needs exactly 2 arguments", c.P)
		}
		aV, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		bV, err := evalExpr(e, c.Args[1].Expr)
		if err != nil {
			return Value{}, err
		}
		a, err := iterableAsNums(aV, c.P)
		if err != nil {
			return Value{}, err
		}
		b, err := iterableAsNums(bV, c.P)
		if err != nil {
			return Value{}, err
		}
		if len(a) != len(b) {
			return Value{}, fmt.Errorf("%v: cross() vectors must have matching dimensions", c.P)
		}
		if len(a) == 2 {
			return Num(a[0]*b[1] - a[1]*b[0]), nil
		}
		if len(a) == 3 {
			return List([]Value{
				Num(a[1]*b[2] - a[2]*b[1]),
				Num(a[2]*b[0] - a[0]*b[2]),
				Num(a[0]*b[1] - a[1]*b[0]),
			}), nil
		}
		return Value{}, fmt.Errorf("%v: cross() only supports 2D or 3D vectors", c.P)
	case "rands":
		if len(c.Args) != 3 && len(c.Args) != 4 {
			return Value{}, fmt.Errorf("%v: rands() needs 3 or 4 arguments", c.P)
		}
		minV, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		maxV, err := evalExpr(e, c.Args[1].Expr)
		if err != nil {
			return Value{}, err
		}
		countV, err := evalExpr(e, c.Args[2].Expr)
		if err != nil {
			return Value{}, err
		}
		minX, err := minV.AsNum(c.P)
		if err != nil {
			return Value{}, err
		}
		maxX, err := maxV.AsNum(c.P)
		if err != nil {
			return Value{}, err
		}
		countF, err := countV.AsNum(c.P)
		if err != nil {
			return Value{}, err
		}
		count := int(countF)
		if float64(count) != countF || count < 0 {
			return Value{}, fmt.Errorf("%v: rands() count must be a non-negative integer", c.P)
		}
		var rng *rand.Rand
		if len(c.Args) == 4 {
			seedV, err := evalExpr(e, c.Args[3].Expr)
			if err != nil {
				return Value{}, err
			}
			seedF, err := seedV.AsNum(c.P)
			if err != nil {
				return Value{}, err
			}
			rng = rand.New(rand.NewSource(int64(seedF)))
		} else {
			rng = rand.New(rand.NewSource(time.Now().UnixNano()))
		}
		out := make([]Value, 0, count)
		span := maxX - minX
		for i := 0; i < count; i++ {
			out = append(out, Num(minX+rng.Float64()*span))
		}
		return List(out), nil
	case "lookup":
		if len(c.Args) != 2 {
			return Value{}, fmt.Errorf("%v: lookup() needs exactly 2 arguments", c.P)
		}
		keyV, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		tableV, err := evalExpr(e, c.Args[1].Expr)
		if err != nil {
			return Value{}, err
		}
		key, err := keyV.AsNum(c.P)
		if err != nil {
			return Value{}, err
		}
		if tableV.Kind != ValList || len(tableV.List) == 0 {
			return Value{}, fmt.Errorf("%v: lookup() table must be a non-empty list of [key,value] pairs", c.P)
		}
		type kv struct {
			K float64
			V float64
		}
		pairs := make([]kv, 0, len(tableV.List))
		for _, p := range tableV.List {
			if p.Kind != ValList || len(p.List) != 2 {
				return Value{}, fmt.Errorf("%v: lookup() table entries must be [key, value]", c.P)
			}
			k, err := p.List[0].AsNum(c.P)
			if err != nil {
				return Value{}, err
			}
			v, err := p.List[1].AsNum(c.P)
			if err != nil {
				return Value{}, err
			}
			pairs = append(pairs, kv{K: k, V: v})
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].K < pairs[j].K })
		if key <= pairs[0].K {
			return Num(pairs[0].V), nil
		}
		last := pairs[len(pairs)-1]
		if key >= last.K {
			return Num(last.V), nil
		}
		for i := 1; i < len(pairs); i++ {
			a := pairs[i-1]
			b := pairs[i]
			if key <= b.K {
				if b.K == a.K {
					return Num(a.V), nil
				}
				t := (key - a.K) / (b.K - a.K)
				return Num(a.V + t*(b.V-a.V)), nil
			}
		}
		return Num(last.V), nil
	}

	// User-defined functions.
	if fd, ok := e.funcs[c.Name]; ok {
		e.push()
		defer e.pop()
		if err := bindParams(e, fd.Params, c.Args); err != nil {
			return Value{}, err
		}
		return evalExpr(e, fd.Body)
	}

	return Value{}, fmt.Errorf("%v: unknown function %q", c.P, c.Name)
}

func iterableAsNums(v Value, pos Pos) ([]float64, error) {
	elems, err := v.IterableElems(pos)
	if err != nil {
		return nil, err
	}
	out := make([]float64, 0, len(elems))
	for _, x := range elems {
		n, err := x.AsNum(pos)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func evalMinMaxArgs(e *env, c Call) ([]float64, error) {
	if len(c.Args) == 0 {
		return nil, fmt.Errorf("%v: function %s needs at least 1 argument", c.P, c.Name)
	}
	if len(c.Args) == 1 {
		v, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return nil, err
		}
		if v.Kind == ValList || v.Kind == ValRange || v.Kind == ValEach {
			xs, err := iterableAsNums(v, c.P)
			if err != nil {
				return nil, err
			}
			if len(xs) == 0 {
				return nil, fmt.Errorf("%v: function %s needs a non-empty vector/range", c.P, c.Name)
			}
			return xs, nil
		}
		x, err := v.AsNum(c.P)
		if err != nil {
			return nil, err
		}
		return []float64{x}, nil
	}
	out := make([]float64, 0, len(c.Args))
	for _, a := range c.Args {
		v, err := evalExpr(e, a.Expr)
		if err != nil {
			return nil, err
		}
		x, err := v.AsNum(c.P)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}

func evalUnaryNumericFuncArg(e *env, c Call) (float64, error) {
	if len(c.Args) != 1 {
		return 0, fmt.Errorf("%v: function %s needs exactly 1 argument", c.P, c.Name)
	}
	arg0, err := evalExpr(e, c.Args[0].Expr)
	if err != nil {
		return 0, err
	}
	return arg0.AsNum(c.P)
}

func evalBinaryNumericFuncArgs(e *env, c Call) (float64, float64, error) {
	if len(c.Args) != 2 {
		return 0, 0, fmt.Errorf("%v: function %s needs exactly 2 arguments", c.P, c.Name)
	}
	arg0, err := evalExpr(e, c.Args[0].Expr)
	if err != nil {
		return 0, 0, err
	}
	arg1, err := evalExpr(e, c.Args[1].Expr)
	if err != nil {
		return 0, 0, err
	}
	x, err := arg0.AsNum(c.P)
	if err != nil {
		return 0, 0, err
	}
	y, err := arg1.AsNum(c.P)
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func bindParams(e *env, params []Param, args []Arg) error {
	// Evaluate defaults first.
	values := make(map[string]Value, len(params))
	for _, p := range params {
		if p.Default != nil {
			v, err := evalExpr(e, p.Default)
			if err != nil {
				return err
			}
			values[p.Name] = v
		}
	}
	// Positional fill.
	posi := 0
	for _, a := range args {
		if a.Name == "" {
			if posi >= len(params) {
				return fmt.Errorf("%v: too many positional args", a.P)
			}
			v, err := evalExpr(e, a.Expr)
			if err != nil {
				return err
			}
			values[params[posi].Name] = v
			posi++
		}
	}
	// Named fill.
	for _, a := range args {
		if a.Name != "" {
			v, err := evalExpr(e, a.Expr)
			if err != nil {
				return err
			}
			values[a.Name] = v
		}
	}
	// Check required
	for _, p := range params {
		if _, ok := values[p.Name]; !ok {
			return fmt.Errorf("%v: missing parameter %q", p.P, p.Name)
		}
	}
	for k, v := range values {
		e.set(k, v)
	}
	return nil
}
