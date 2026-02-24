package scad

import (
	"fmt"
	"math"
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
			return evalStmt(e, st.Then)
		}
		if st.Else != nil {
			return evalStmt(e, st.Else)
		}
		return nil, nil

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
		// Strings mostly for future extensions.
		return Value{Kind: ValList, List: []Value{}}, nil
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
			vals = append(vals, v)
		}
		return List(vals), nil
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

func evalFuncCall(e *env, c Call) (Value, error) {
	// Built-in numeric functions (minimal set).
	if len(c.Args) == 0 {
		return Value{}, fmt.Errorf("%v: function %s needs args", c.P, c.Name)
	}
	arg0, err := evalExpr(e, c.Args[0].Expr)
	if err != nil {
		return Value{}, err
	}
	x, err := arg0.AsNum(c.P)
	if err != nil {
		return Value{}, err
	}
	switch c.Name {
	case "sin":
		return Num(math.Sin(x)), nil
	case "cos":
		return Num(math.Cos(x)), nil
	case "sqrt":
		return Num(math.Sqrt(x)), nil
	case "abs":
		return Num(math.Abs(x)), nil
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
