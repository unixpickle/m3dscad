package scad

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"
)

type moduleDef struct {
	Params   []Param
	Body     *BlockStmt
	Captured []*scope
}

type funcDef struct {
	Params   []Param
	Body     Expr
	Captured []*scope
}

type scope struct {
	vars map[string]Value
	mods map[string]moduleDef
	fncs map[string]funcDef
}

type env struct {
	scopes []*scope
	echo   EchoHandler
}

type EchoHandler func(msg string)

func defaultEchoHandler(msg string) {
	log.Println(msg)
}

func newEnv(echo EchoHandler) *env {
	if echo == nil {
		echo = defaultEchoHandler
	}
	root := newScope()
	root.vars["PI"] = Num(math.Pi)
	return &env{
		scopes: []*scope{root},
		echo:   echo,
	}
}

func newScope() *scope {
	return &scope{
		vars: map[string]Value{},
		mods: map[string]moduleDef{},
		fncs: map[string]funcDef{},
	}
}

func (e *env) push() { e.scopes = append(e.scopes, newScope()) }
func (e *env) pop()  { e.scopes = e.scopes[:len(e.scopes)-1] }

func (e *env) currentScope() *scope {
	return e.scopes[len(e.scopes)-1]
}

func (e *env) set(name string, v Value) error {
	cur := e.currentScope()
	if _, ok := cur.vars[name]; ok {
		return fmt.Errorf("cannot redeclare variable %q in current scope", name)
	}
	cur.vars[name] = v
	return nil
}

func (e *env) defineFunc(name string, f funcDef) error {
	cur := e.currentScope()
	if _, ok := cur.fncs[name]; ok {
		return fmt.Errorf("cannot redeclare function %q in current scope", name)
	}
	cur.fncs[name] = f
	return nil
}

func (e *env) defineModule(name string, m moduleDef) error {
	cur := e.currentScope()
	if _, ok := cur.mods[name]; ok {
		return fmt.Errorf("cannot redeclare module %q in current scope", name)
	}
	cur.mods[name] = m
	return nil
}

func (e *env) get(name string) (Value, bool) {
	for i := len(e.scopes) - 1; i >= 0; i-- {
		if v, ok := e.scopes[i].vars[name]; ok {
			return v, true
		}
	}
	return Value{}, false
}

func (e *env) getFunc(name string) (funcDef, bool) {
	for i := len(e.scopes) - 1; i >= 0; i-- {
		if f, ok := e.scopes[i].fncs[name]; ok {
			return f, true
		}
	}
	return funcDef{}, false
}

func (e *env) getModule(name string) (moduleDef, bool) {
	for i := len(e.scopes) - 1; i >= 0; i-- {
		if m, ok := e.scopes[i].mods[name]; ok {
			return m, true
		}
	}
	return moduleDef{}, false
}

func (e *env) captureScopes() []*scope {
	out := make([]*scope, len(e.scopes))
	copy(out, e.scopes)
	return out
}

func (e *env) withCapturedScopes(scopes []*scope, fn func() error) error {
	orig := e.scopes
	e.scopes = append(append([]*scope{}, scopes...), newScope())
	defer func() {
		e.scopes = orig
	}()
	return fn()
}

func Parse(src string) (*Program, error) {
	p, err := NewParser(src)
	if err != nil {
		return nil, err
	}
	return p.ParseProgram()
}

func Eval(p *Program) (ShapeRep, error) {
	return EvalWithEcho(p, nil)
}

func EvalWithEcho(p *Program, echo EchoHandler) (ShapeRep, error) {
	e := newEnv(echo)
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
	// OpenSCAD-like ordering:
	// 1) definitions, 2) assignments, 3) geometry/control statements.
	for _, s := range ss {
		switch st := s.(type) {
		case *ModuleDefStmt, *FuncDefStmt:
			if _, err := evalStmt(e, st); err != nil {
				return nil, err
			}
		default:
		}
	}
	for _, s := range ss {
		if st, ok := s.(*AssignStmt); ok {
			if _, err := evalStmt(e, st); err != nil {
				return nil, err
			}
		}
	}
	var out []ShapeRep
	for _, s := range ss {
		switch s.(type) {
		case *ModuleDefStmt, *FuncDefStmt, *AssignStmt:
			continue
		}
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

func evalStmt(e *env, s Stmt) (shape *ShapeRep, err error) {
	defer func() {
		if err != nil {
			err = WithPos(err, s.pos())
		}
	}()

	switch st := s.(type) {
	case *AssignStmt:
		v, err := evalExpr(e, st.Expr)
		if err != nil {
			return nil, err
		}
		if err := e.set(st.Name, v); err != nil {
			return nil, err
		}
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
		b, err := condV.AsBool()
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
		err := e.defineModule(st.Name, moduleDef{
			Params:   st.Params,
			Body:     st.Body,
			Captured: e.captureScopes(),
		})
		if err != nil {
			return nil, err
		}
		return nil, nil

	case *FuncDefStmt:
		err := e.defineFunc(st.Name, funcDef{
			Params:   st.Params,
			Body:     st.Body,
			Captured: e.captureScopes(),
		})
		if err != nil {
			return nil, err
		}
		return nil, nil

	case *CallStmt:
		return evalCallStmt(e, st)

	default:
		return nil, fmt.Errorf("unknown stmt type")
	}
}

func evalCallStmt(e *env, st *CallStmt) (*ShapeRep, error) {
	name := st.Call.Name

	if name == "echo" {
		if len(st.Children) > 0 {
			return nil, fmt.Errorf("%s() does not take children", name)
		}
		args, err := evalEchoArgs(e, st.Call.Args)
		if err != nil {
			return nil, err
		}
		e.echo(strings.Join(args, ", "))
		return nil, nil
	}
	if name == "assert" {
		if len(st.Children) > 0 {
			return nil, fmt.Errorf("%s() does not take children", name)
		}
		failed, msg, err := evalAssertCall(e, st.Call)
		if err != nil {
			return nil, err
		}
		if failed {
			return nil, fmt.Errorf("%s", msg)
		}
		return nil, nil
	}

	if handler, ok := builtinHandlers[name]; ok {
		if len(st.Children) == 0 && handler.RequireChildren {
			return nil, fmt.Errorf("%s() requires children", name)
		}
		if len(st.Children) > 0 && !handler.AllowChildren {
			return nil, fmt.Errorf("%s() does not take children", name)
		}
		var children []ShapeRep
		var childUnion *ShapeRep
		if len(st.Children) > 0 {
			e.push()
			err := func() error {
				var err error
				children, err = evalStmts(e, st.Children)
				if err != nil {
					return err
				}
				if handler.NeedsChildUnion {
					u, err := unionAll(children)
					if err != nil {
						return err
					}
					childUnion = &u
				}
				return nil
			}()
			e.pop()
			if err != nil {
				return nil, err
			}
		}
		res, err := handler.Eval(e, st, children, childUnion)
		if err != nil {
			return nil, err
		}
		return &res, nil
	}

	// User-defined module call (solids)
	if md, ok := e.getModule(name); ok {
		if len(st.Children) > 0 {
			return nil, fmt.Errorf("module %s does not support children", name)
		}
		caller := &env{
			scopes: append([]*scope{}, e.scopes...),
			echo:   e.echo,
		}
		var out *ShapeRep
		err := e.withCapturedScopes(md.Captured, func() error {
			if err := bindParams(e, caller, md.Params, st.Call.Args); err != nil {
				return err
			}
			solids, err := evalStmts(e, md.Body.Stmts)
			if err != nil {
				return err
			}
			u, err := unionAll(solids)
			if err != nil {
				return err
			}
			out = &u
			return nil
		})
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	return nil, fmt.Errorf("unknown module/primitive %q", name)
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
			return Value{}, PosErrorf(x.pos(), "undefined variable %q", x.Name)
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
				elems, err := v.IterableElems()
				if err != nil {
					return Value{}, err
				}
				vals = append(vals, elems...)
				continue
			}
			if v.Kind == ValEach {
				elems, err := v.IterableElems()
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
		start, err := startV.AsNum()
		if err != nil {
			return Value{}, err
		}
		end, err := endV.AsNum()
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
			step, err = stepV.AsNum()
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
		idxNum, err := idxV.AsNum()
		if err != nil {
			return Value{}, err
		}
		idx := int(idxNum)
		if float64(idx) != idxNum {
			return Value{}, PosErrorf(x.Index.pos(), "index must be an integer")
		}
		return base.ElemAt(idx)
	case *DotExpr:
		base, err := evalExpr(e, x.X)
		if err != nil {
			return Value{}, err
		}
		idx := -1
		switch x.Name {
		case "x":
			idx = 0
		case "y":
			idx = 1
		case "z":
			idx = 2
		default:
			return Value{}, PosErrorf(x.P, "unknown vector accessor %q", x.Name)
		}
		return base.ElemAt(idx)
	case *FuncLitExpr:
		return FuncValue(FuncClosure{
			Params:   x.Params,
			Body:     x.Body,
			Captured: e.captureScopes(),
		}), nil
	case *InvokeExpr:
		fnV, err := evalExpr(e, x.Fn)
		if err != nil {
			return Value{}, err
		}
		if fnV.Kind != ValFunc || fnV.Func == nil {
			return Value{}, PosErrorf(x.P, "expression is not callable")
		}
		return evalClosureCall(e, fnV.Func, x.Args)
	case *ForExpr:
		var out []Value
		err := evalForBindsExpr(e, x.Binds, 0, func() error {
			v, err := evalExpr(e, x.Body)
			if err != nil {
				return err
			}
			if v.Kind == ValEach {
				elems, err := v.IterableElems()
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
			if err := e.set(b.Name, v); err != nil {
				return Value{}, err
			}
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
			b, err := v.AsBool()
			if err != nil {
				return Value{}, err
			}
			return Bool(!b), nil
		case TokMinus, TokPlus:
			return evalUnaryArithmetic(x.Op, v)
		default:
			return Value{}, PosErrorf(x.pos(), "unknown unary op")
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
			v, err := evalBinaryArithmetic(x.Op, lv, rv)
			if err != nil {
				return Value{}, WithPos(err, x.pos())
			}
			return v, nil
		case TokEq:
			return Bool(lv.Equal(rv)), nil
		case TokNeq:
			return Bool(!lv.Equal(rv)), nil
		case TokLt, TokLte, TokGt, TokGte:
			ord, ok := lv.CompareOrder(rv)
			if !ok {
				return Bool(false), nil
			}
			switch x.Op {
			case TokLt:
				return Bool(ord < 0), nil
			case TokLte:
				return Bool(ord <= 0), nil
			case TokGt:
				return Bool(ord > 0), nil
			case TokGte:
				return Bool(ord >= 0), nil
			}
		case TokAnd, TokOr:
			a, err := lv.AsBool()
			if err != nil {
				return Value{}, err
			}
			b, err := rv.AsBool()
			if err != nil {
				return Value{}, err
			}
			if x.Op == TokAnd {
				return Bool(a && b), nil
			}
			return Bool(a || b), nil
		default:
			return Value{}, PosErrorf(x.pos(), "unknown binary op")
		}
	case *TernaryExpr:
		cv, err := evalExpr(e, x.Cond)
		if err != nil {
			return Value{}, err
		}
		b, err := cv.AsBool()
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
		return Value{}, PosErrorf(ex.pos(), "unknown expr type")
	}
	return Value{}, PosErrorf(ex.pos(), "unreachable")
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
	elems, err := iterVal.IterableElems()
	if err != nil {
		return err
	}
	for _, val := range elems {
		e.push()
		if err := e.set(b.Name, val); err != nil {
			e.pop()
			return err
		}
		err := evalForBindsExpr(e, binds, idx+1, fn)
		e.pop()
		if err != nil {
			return err
		}
	}
	return nil
}

func evalFuncCall(e *env, c Call) (Value, error) {
	if c.Name == "assert" {
		return Value{}, PosErrorf(c.P, "assert() must be used as a statement")
	}
	if v, ok := e.get(c.Name); ok {
		if v.Kind != ValFunc || v.Func == nil {
			return Value{}, PosErrorf(c.P, "%q is not callable", c.Name)
		}
		return evalClosureCall(e, v.Func, c.Args)
	}

	switch c.Name {
	case "echo":
		args, err := evalEchoArgs(e, c.Args)
		if err != nil {
			return Value{}, err
		}
		e.echo(strings.Join(args, ", "))
		return Value{}, nil
	case "len":
		if len(c.Args) != 1 {
			return Value{}, PosErrorf(c.P, "len() takes exactly 1 argument")
		}
		arg0, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		n, err := arg0.Len()
		if err != nil {
			return Value{}, err
		}
		return Num(float64(n)), nil
	case "concat":
		if len(c.Args) == 0 {
			return Value{}, PosErrorf(c.P, "concat() needs at least 1 argument")
		}
		var out []Value
		for _, a := range c.Args {
			v, err := evalExpr(e, a.Expr)
			if err != nil {
				return Value{}, err
			}
			elems, err := v.IterableElems()
			if err != nil {
				return Value{}, err
			}
			out = append(out, elems...)
		}
		return List(out), nil
	case "str":
		var sb strings.Builder
		for _, a := range c.Args {
			v, err := evalExpr(e, a.Expr)
			if err != nil {
				return Value{}, err
			}
			sb.WriteString(strValueString(v))
		}
		return String(sb.String()), nil
	case "is_list":
		arg0, err := evalUnaryFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Bool(arg0.Kind == ValList), nil
	case "is_num":
		arg0, err := evalUnaryFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Bool(arg0.Kind == ValNum && !math.IsNaN(arg0.Num)), nil
	case "is_bool":
		arg0, err := evalUnaryFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Bool(arg0.Kind == ValBool), nil
	case "is_string":
		arg0, err := evalUnaryFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Bool(arg0.Kind == ValString), nil
	case "is_function":
		arg0, err := evalUnaryFuncArg(e, c)
		if err != nil {
			return Value{}, err
		}
		return Bool(arg0.Kind == ValFunc && arg0.Func != nil), nil
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
			return Value{}, PosErrorf(c.P, "norm() needs exactly 1 argument")
		}
		v, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		xs, err := iterableAsNums(v)
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
			return Value{}, PosErrorf(c.P, "cross() needs exactly 2 arguments")
		}
		aV, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		bV, err := evalExpr(e, c.Args[1].Expr)
		if err != nil {
			return Value{}, err
		}
		a, err := iterableAsNums(aV)
		if err != nil {
			return Value{}, err
		}
		b, err := iterableAsNums(bV)
		if err != nil {
			return Value{}, err
		}
		if len(a) != len(b) {
			return Value{}, PosErrorf(c.P, "cross() vectors must have matching dimensions")
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
		return Value{}, PosErrorf(c.P, "cross() only supports 2D or 3D vectors")
	case "rands":
		if len(c.Args) != 3 && len(c.Args) != 4 {
			return Value{}, PosErrorf(c.P, "rands() needs 3 or 4 arguments")
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
		minX, err := minV.AsNum()
		if err != nil {
			return Value{}, err
		}
		maxX, err := maxV.AsNum()
		if err != nil {
			return Value{}, err
		}
		countF, err := countV.AsNum()
		if err != nil {
			return Value{}, err
		}
		count := int(countF)
		if float64(count) != countF || count < 0 {
			return Value{}, PosErrorf(c.P, "rands() count must be a non-negative integer")
		}
		var rng *rand.Rand
		if len(c.Args) == 4 {
			seedV, err := evalExpr(e, c.Args[3].Expr)
			if err != nil {
				return Value{}, err
			}
			seedF, err := seedV.AsNum()
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
			return Value{}, PosErrorf(c.P, "lookup() needs exactly 2 arguments")
		}
		keyV, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return Value{}, err
		}
		tableV, err := evalExpr(e, c.Args[1].Expr)
		if err != nil {
			return Value{}, err
		}
		key, err := keyV.AsNum()
		if err != nil {
			return Value{}, err
		}
		if tableV.Kind != ValList || len(tableV.List) == 0 {
			return Value{}, PosErrorf(c.P, "lookup() table must be a non-empty list of [key,value] pairs")
		}
		type kv struct {
			K float64
			V float64
		}
		pairs := make([]kv, 0, len(tableV.List))
		for _, p := range tableV.List {
			if p.Kind != ValList || len(p.List) != 2 {
				return Value{}, PosErrorf(c.P, "lookup() table entries must be [key, value]")
			}
			k, err := p.List[0].AsNum()
			if err != nil {
				return Value{}, err
			}
			v, err := p.List[1].AsNum()
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
	if fd, ok := e.getFunc(c.Name); ok {
		return evalClosureCall(e, &FuncClosure{
			Params:   fd.Params,
			Body:     fd.Body,
			Captured: fd.Captured,
		}, c.Args)
	}

	return Value{}, PosErrorf(c.P, "unknown function %q", c.Name)
}

func evalClosureCall(e *env, fn *FuncClosure, args []Arg) (Value, error) {
	if fn == nil {
		return Value{}, fmt.Errorf("invalid function value")
	}
	caller := &env{
		scopes: append([]*scope{}, e.scopes...),
		echo:   e.echo,
	}
	callEnv := &env{
		scopes: append(append([]*scope{}, fn.Captured...), newScope()),
		echo:   e.echo,
	}
	if err := bindParams(callEnv, caller, fn.Params, args); err != nil {
		return Value{}, err
	}
	return evalExpr(callEnv, fn.Body)
}

func evalClosureCallValues(e *env, fn *FuncClosure, args []Value) (Value, error) {
	if fn == nil {
		return Value{}, fmt.Errorf("invalid function value")
	}
	callEnv := &env{
		scopes: append(append([]*scope{}, fn.Captured...), newScope()),
		echo:   e.echo,
	}
	if err := bindParamsValues(callEnv, fn.Params, args); err != nil {
		return Value{}, err
	}
	return evalExpr(callEnv, fn.Body)
}

func evalEchoArgs(e *env, args []Arg) ([]string, error) {
	out := make([]string, 0, len(args))
	for _, a := range args {
		v, err := evalExpr(e, a.Expr)
		if err != nil {
			return nil, err
		}
		elem := valueString(v)
		if a.Name != "" {
			elem = a.Name + " = " + elem
		}
		out = append(out, elem)
	}
	return out, nil
}

func evalAssertCall(e *env, c Call) (bool, string, error) {
	condArg, msgArg, err := parseAssertArgs(c)
	if err != nil {
		return false, "", err
	}
	condValue, err := evalExpr(e, condArg.Expr)
	if err != nil {
		return false, "", err
	}
	if condValue.Kind != ValBool {
		return false, "", PosErrorf(condArg.P, "expected bool")
	}
	if condValue.Bool {
		return false, "", nil
	}
	msg := "assertion failed"
	if msgArg != nil {
		msgValue, err := evalExpr(e, msgArg.Expr)
		if err != nil {
			return false, "", err
		}
		if msgValue.Kind != ValString {
			return false, "", PosErrorf(msgArg.P, "expected string")
		}
		msg = "assertion failed: " + msgValue.Str
	}
	return true, msg, nil
}

func parseAssertArgs(c Call) (Arg, *Arg, error) {
	if len(c.Args) == 0 {
		return Arg{}, nil, PosErrorf(c.P, "assert() takes 1 or 2 arguments")
	}

	var condArg *Arg
	var msgArg *Arg
	seenNamed := false
	positionalCount := 0
	for _, rawArg := range c.Args {
		arg := rawArg
		if arg.Name != "" {
			seenNamed = true
			switch arg.Name {
			case "condition":
				if condArg != nil {
					return Arg{}, nil, PosErrorf(arg.P, `assert(): duplicate argument "condition"`)
				}
				condArg = &arg
			case "message":
				if msgArg != nil {
					return Arg{}, nil, PosErrorf(arg.P, `assert(): duplicate argument "message"`)
				}
				msgArg = &arg
			default:
				return Arg{}, nil, PosErrorf(arg.P, `assert(): unknown argument %q`, arg.Name)
			}
			continue
		}
		if seenNamed {
			return Arg{}, nil, PosErrorf(arg.P, "assert(): positional args cannot follow named args")
		}
		switch positionalCount {
		case 0:
			condArg = &arg
		case 1:
			msgArg = &arg
		default:
			return Arg{}, nil, PosErrorf(arg.P, "assert() takes 1 or 2 arguments")
		}
		positionalCount++
	}
	if condArg == nil {
		return Arg{}, nil, PosErrorf(c.P, `missing parameter "condition"`)
	}
	return *condArg, msgArg, nil
}

func valueString(v Value) string {
	switch v.Kind {
	case ValNull:
		return "undef"
	case ValNum:
		return strconv.FormatFloat(v.Num, 'g', -1, 64)
	case ValBool:
		if v.Bool {
			return "true"
		}
		return "false"
	case ValString:
		return strconv.Quote(v.Str)
	case ValRange:
		return "[" + strconv.FormatFloat(v.Rng.Start, 'g', -1, 64) +
			" : " + strconv.FormatFloat(v.Rng.Step, 'g', -1, 64) +
			" : " + strconv.FormatFloat(v.Rng.End, 'g', -1, 64) + "]"
	case ValEach:
		if v.Each == nil {
			return "each(undef)"
		}
		return "each(" + valueString(*v.Each) + ")"
	case ValList:
		parts := make([]string, 0, len(v.List))
		for _, elem := range v.List {
			parts = append(parts, valueString(elem))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case ValFunc:
		if v.Func == nil {
			return "function() undef"
		}
		paramStrs := make([]string, 0, len(v.Func.Params))
		for _, p := range v.Func.Params {
			if p.Default == nil {
				paramStrs = append(paramStrs, p.Name)
			} else {
				paramStrs = append(paramStrs, p.Name+"="+formatExpr(p.Default))
			}
		}
		return "function(" + strings.Join(paramStrs, ", ") + ") " + formatExpr(v.Func.Body)
	default:
		return "undef"
	}
}

func strValueString(v Value) string {
	if v.Kind == ValString {
		return v.Str
	}
	return valueString(v)
}

func formatExpr(ex Expr) string {
	switch x := ex.(type) {
	case *NumberLit:
		return strconv.FormatFloat(x.V, 'g', -1, 64)
	case *BoolLit:
		if x.V {
			return "true"
		}
		return "false"
	case *StringLit:
		return strconv.Quote(x.V)
	case *VarExpr:
		return x.Name
	case *ArrayLit:
		parts := make([]string, 0, len(x.Elems))
		for _, elem := range x.Elems {
			parts = append(parts, formatExpr(elem))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case *RangeLit:
		if x.Step == nil {
			return "[" + formatExpr(x.Start) + " : " + formatExpr(x.End) + "]"
		}
		return "[" + formatExpr(x.Start) + " : " + formatExpr(x.Step) + " : " + formatExpr(x.End) + "]"
	case *UnaryExpr:
		op := "?"
		switch x.Op {
		case TokNot:
			op = "!"
		case TokMinus:
			op = "-"
		case TokPlus:
			op = "+"
		}
		return "(" + op + formatExpr(x.X) + ")"
	case *BinaryExpr:
		op := "?"
		switch x.Op {
		case TokPlus:
			op = "+"
		case TokMinus:
			op = "-"
		case TokStar:
			op = "*"
		case TokSlash:
			op = "/"
		case TokPercent:
			op = "%"
		case TokCaret:
			op = "^"
		case TokEq:
			op = "=="
		case TokNeq:
			op = "!="
		case TokLt:
			op = "<"
		case TokLte:
			op = "<="
		case TokGt:
			op = ">"
		case TokGte:
			op = ">="
		case TokAnd:
			op = "&&"
		case TokOr:
			op = "||"
		}
		return "(" + formatExpr(x.L) + " " + op + " " + formatExpr(x.R) + ")"
	case *TernaryExpr:
		return "(" + formatExpr(x.Cond) + " ? " + formatExpr(x.Then) + " : " + formatExpr(x.Else) + ")"
	case *CallExpr:
		return x.Call.Name + "(" + formatArgs(x.Call.Args) + ")"
	case *InvokeExpr:
		return formatExpr(x.Fn) + "(" + formatArgs(x.Args) + ")"
	case *IndexExpr:
		return formatExpr(x.X) + "[" + formatExpr(x.Index) + "]"
	case *DotExpr:
		return formatExpr(x.X) + "." + x.Name
	case *ForExpr:
		return "for(...) " + formatExpr(x.Body)
	case *LetExpr:
		return "let(...) " + formatExpr(x.Body)
	case *EachExpr:
		return "each " + formatExpr(x.X)
	case *FuncLitExpr:
		paramStrs := make([]string, 0, len(x.Params))
		for _, p := range x.Params {
			if p.Default == nil {
				paramStrs = append(paramStrs, p.Name)
			} else {
				paramStrs = append(paramStrs, p.Name+"="+formatExpr(p.Default))
			}
		}
		return "function(" + strings.Join(paramStrs, ", ") + ") " + formatExpr(x.Body)
	default:
		return "?"
	}
}

func formatArgs(args []Arg) string {
	parts := make([]string, 0, len(args))
	for _, a := range args {
		if a.Name == "" {
			parts = append(parts, formatExpr(a.Expr))
		} else {
			parts = append(parts, a.Name+"="+formatExpr(a.Expr))
		}
	}
	return strings.Join(parts, ", ")
}

func iterableAsNums(v Value) ([]float64, error) {
	elems, err := v.IterableElems()
	if err != nil {
		return nil, err
	}
	out := make([]float64, 0, len(elems))
	for _, x := range elems {
		n, err := x.AsNum()
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func evalMinMaxArgs(e *env, c Call) ([]float64, error) {
	if len(c.Args) == 0 {
		return nil, PosErrorf(c.P, "function %s needs at least 1 argument", c.Name)
	}
	if len(c.Args) == 1 {
		v, err := evalExpr(e, c.Args[0].Expr)
		if err != nil {
			return nil, err
		}
		if v.Kind == ValList || v.Kind == ValRange || v.Kind == ValEach {
			xs, err := iterableAsNums(v)
			if err != nil {
				return nil, err
			}
			if len(xs) == 0 {
				return nil, PosErrorf(c.P, "function %s needs a non-empty vector/range", c.Name)
			}
			return xs, nil
		}
		x, err := v.AsNum()
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
		x, err := v.AsNum()
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, nil
}

func evalUnaryNumericFuncArg(e *env, c Call) (float64, error) {
	arg0, err := evalUnaryFuncArg(e, c)
	if err != nil {
		return 0, err
	}
	return arg0.AsNum()
}

func evalUnaryFuncArg(e *env, c Call) (Value, error) {
	if len(c.Args) != 1 {
		return Value{}, PosErrorf(c.P, "function %s needs exactly 1 argument", c.Name)
	}
	arg0, err := evalExpr(e, c.Args[0].Expr)
	if err != nil {
		return Value{}, err
	}
	return arg0, nil
}

func evalBinaryNumericFuncArgs(e *env, c Call) (float64, float64, error) {
	if len(c.Args) != 2 {
		return 0, 0, PosErrorf(c.P, "function %s needs exactly 2 arguments", c.Name)
	}
	arg0, err := evalExpr(e, c.Args[0].Expr)
	if err != nil {
		return 0, 0, err
	}
	arg1, err := evalExpr(e, c.Args[1].Expr)
	if err != nil {
		return 0, 0, err
	}
	x, err := arg0.AsNum()
	if err != nil {
		return 0, 0, err
	}
	y, err := arg1.AsNum()
	if err != nil {
		return 0, 0, err
	}
	return x, y, nil
}

func bindParams(bindEnv, evalEnv *env, params []Param, args []Arg) error {
	paramNames := make(map[string]struct{}, len(params))
	for _, p := range params {
		paramNames[p.Name] = struct{}{}
	}

	// Evaluate defaults first.
	values := make(map[string]Value, len(params))
	for _, p := range params {
		if p.Default != nil {
			v, err := evalExpr(bindEnv, p.Default)
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
				return PosErrorf(a.P, "too many positional args")
			}
			v, err := evalExpr(evalEnv, a.Expr)
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
			if _, ok := paramNames[a.Name]; !ok {
				return PosErrorf(a.P, "unknown named argument %q", a.Name)
			}
			v, err := evalExpr(evalEnv, a.Expr)
			if err != nil {
				return err
			}
			values[a.Name] = v
		}
	}
	// Check required
	for _, p := range params {
		if _, ok := values[p.Name]; !ok {
			return fmt.Errorf("missing parameter %q (declared at %s)", p.Name, p.P)
		}
	}
	for k, v := range values {
		if err := bindEnv.set(k, v); err != nil {
			return err
		}
	}
	return nil
}

func bindParamsValues(bindEnv *env, params []Param, args []Value) error {
	values := make(map[string]Value, len(params))
	for _, p := range params {
		if p.Default != nil {
			v, err := evalExpr(bindEnv, p.Default)
			if err != nil {
				return err
			}
			values[p.Name] = v
		}
	}
	if len(args) > len(params) {
		return fmt.Errorf("too many positional args")
	}
	for i, v := range args {
		values[params[i].Name] = v
	}
	for _, p := range params {
		if _, ok := values[p.Name]; !ok {
			return PosErrorf(p.P, "missing parameter %q", p.Name)
		}
	}
	for k, v := range values {
		if err := bindEnv.set(k, v); err != nil {
			return err
		}
	}
	return nil
}
