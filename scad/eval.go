package scad

import (
	"fmt"
	"math"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

type SolidKind int

const (
	Solid2D SolidKind = iota
	Solid3D
)

type SolidValue struct {
	Kind   SolidKind
	Solid2 model2d.Solid
	Solid3 model3d.Solid
}

func solid2D(s model2d.Solid) SolidValue {
	return SolidValue{Kind: Solid2D, Solid2: s}
}

func solid3D(s model3d.Solid) SolidValue {
	return SolidValue{Kind: Solid3D, Solid3: s}
}

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

func Eval(p *Program) (model3d.Solid, error) {
	e := newEnv()
	solids, err := evalStmts(e, p.Stmts)
	if err != nil {
		return nil, err
	}
	merged, err := unionAll(solids)
	if err != nil {
		return nil, err
	}
	if merged.Kind != Solid3D {
		return nil, fmt.Errorf("top-level did not produce a 3D solid")
	}
	return merged.Solid3, nil
}

func evalStmts(e *env, ss []Stmt) ([]SolidValue, error) {
	var out []SolidValue
	for _, s := range ss {
		got, err := evalStmt(e, s)
		if err != nil {
			return nil, err
		}
		out = append(out, got...)
	}
	return out, nil
}

func evalStmt(e *env, s Stmt) ([]SolidValue, error) {
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
		return evalStmts(e, st.Stmts)

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

func evalCallStmt(e *env, st *CallStmt) ([]SolidValue, error) {
	name := st.Call.Name

	// If has children, evaluate to solids first (implicit union).
	var childSolid SolidValue
	if len(st.Children) > 0 {
		children, err := evalStmts(e, st.Children)
		if err != nil {
			return nil, err
		}
		u, err := unionAll(children)
		if err != nil {
			return nil, err
		}
		childSolid = u
	}

	// Built-in CSG/transform modules
	switch name {
	case "union":
		if len(st.Children) == 0 {
			return nil, fmt.Errorf("%v: union() requires children", st.pos())
		}
		children, err := evalStmts(e, st.Children)
		if err != nil {
			return nil, err
		}
		u, err := unionAll(children)
		if err != nil {
			return nil, err
		}
		return []SolidValue{u}, nil

	case "difference":
		if len(st.Children) == 0 {
			return nil, fmt.Errorf("%v: difference() requires children", st.pos())
		}
		children, err := evalStmts(e, st.Children)
		if err != nil {
			return nil, err
		}
		if len(children) == 0 {
			return nil, fmt.Errorf("%v: difference() had no solids", st.pos())
		}
		kind, err := ensureSameKind(children)
		if err != nil {
			return nil, fmt.Errorf("%v: difference(): %w", st.pos(), err)
		}
		if len(children) == 1 {
			return []SolidValue{children[0]}, nil
		}
		subUnion, err := unionAll(children[1:])
		if err != nil {
			return nil, err
		}
		switch kind {
		case Solid3D:
			return []SolidValue{solid3D(model3d.Subtract(children[0].Solid3, subUnion.Solid3))}, nil
		case Solid2D:
			return []SolidValue{solid2D(model2d.Subtract(children[0].Solid2, subUnion.Solid2))}, nil
		default:
			return nil, fmt.Errorf("%v: difference(): unknown solid kind", st.pos())
		}

	case "intersection":
		if len(st.Children) == 0 {
			return nil, fmt.Errorf("%v: intersection() requires children", st.pos())
		}
		children, err := evalStmts(e, st.Children)
		if err != nil {
			return nil, err
		}
		kind, err := ensureSameKind(children)
		if err != nil {
			return nil, fmt.Errorf("%v: intersection(): %w", st.pos(), err)
		}
		switch kind {
		case Solid3D:
			solids := make([]model3d.Solid, 0, len(children))
			for _, ch := range children {
				solids = append(solids, ch.Solid3)
			}
			return []SolidValue{solid3D(model3d.IntersectedSolid(solids))}, nil
		case Solid2D:
			solids := make([]model2d.Solid, 0, len(children))
			for _, ch := range children {
				solids = append(solids, ch.Solid2)
			}
			return []SolidValue{solid2D(model2d.IntersectedSolid(solids))}, nil
		default:
			return nil, fmt.Errorf("%v: intersection(): unknown solid kind", st.pos())
		}

	case "translate", "scale", "rotate":
		if len(st.Children) == 0 {
			return nil, fmt.Errorf("%v: %s() requires children", st.pos(), name)
		}
		arg0, err := evalArg0Vec3(e, st.Call)
		if err != nil {
			return nil, err
		}
		switch childSolid.Kind {
		case Solid3D:
			var xf model3d.Transform
			switch name {
			case "translate":
				xf = &model3d.Translate{Offset: model3d.XYZ(arg0[0], arg0[1], arg0[2])}
			case "scale":
				xf = &model3d.VecScale{Scale: model3d.XYZ(arg0[0], arg0[1], arg0[2])}
			case "rotate":
				// Apply X then Y then Z (and invert reverse order with negative angles).
				xf = model3d.JoinedTransform{
					model3d.Rotation(model3d.XYZ(1, 0, 0), arg0[0]*math.Pi/180),
					model3d.Rotation(model3d.XYZ(0, 1, 0), arg0[1]*math.Pi/180),
					model3d.Rotation(model3d.XYZ(0, 0, 1), arg0[2]*math.Pi/180),
				}
			}
			return []SolidValue{solid3D(model3d.TransformSolid(xf, childSolid.Solid3))}, nil
		case Solid2D:
			if arg0[2] != 0 && name != "rotate" {
				return nil, fmt.Errorf("%v: %s(): z component not supported for 2D solids", st.pos(), name)
			}
			switch name {
			case "translate":
				xf := &model2d.Translate{Offset: model2d.XY(arg0[0], arg0[1])}
				return []SolidValue{solid2D(model2d.TransformSolid(xf, childSolid.Solid2))}, nil
			case "scale":
				xf := &model2d.VecScale{Scale: model2d.XY(arg0[0], arg0[1])}
				return []SolidValue{solid2D(model2d.TransformSolid(xf, childSolid.Solid2))}, nil
			case "rotate":
				if arg0[0] != 0 || arg0[1] != 0 {
					return nil, fmt.Errorf("%v: rotate(): only Z rotation supported for 2D solids", st.pos())
				}
				xf := model2d.Rotation(arg0[2] * math.Pi / 180)
				return []SolidValue{solid2D(model2d.TransformSolid(xf, childSolid.Solid2))}, nil
			}
		}
		return nil, fmt.Errorf("%v: %s(): unsupported solid kind", st.pos(), name)

	case "linear_extrude":
		if len(st.Children) == 0 {
			return nil, fmt.Errorf("%v: linear_extrude() requires children", st.pos())
		}
		if childSolid.Kind != Solid2D {
			return nil, fmt.Errorf("%v: linear_extrude() requires 2D children", st.pos())
		}
		h, err := getNamedOrPosNumAny(e, st.Call, []string{"height", "h"}, 0, 1.0)
		if err != nil {
			return nil, err
		}
		center, err := getNamedOrPosBool(e, st.Call, "center", 1, false)
		if err != nil {
			return nil, err
		}
		return []SolidValue{solid3D(linearExtrude(childSolid.Solid2, h, center))}, nil
	}

	// Built-in primitives (no children)
	if len(st.Children) > 0 && (name == "cube" || name == "sphere" || name == "cylinder" || name == "square" || name == "circle") {
		return nil, fmt.Errorf("%v: %s() does not take children", st.pos(), name)
	}

	switch name {
	case "sphere":
		r, err := getNamedOrPosNum(e, st.Call, "r", 0, 1.0)
		if err != nil {
			return nil, err
		}
		return []SolidValue{solid3D(&model3d.Sphere{Radius: r})}, nil

	case "cube":
		sizeV, err := getNamedOrPosValue(e, st.Call, "size", 0, Num(1))
		if err != nil {
			return nil, err
		}
		vec, err := sizeV.AsVec3(st.pos())
		if err != nil {
			return nil, err
		}
		center, err := getNamedOrPosBool(e, st.Call, "center", 1, false)
		if err != nil {
			return nil, err
		}
		min := [3]float64{0, 0, 0}
		max := vec
		if center {
			min = [3]float64{-vec[0] / 2, -vec[1] / 2, -vec[2] / 2}
			max = [3]float64{vec[0] / 2, vec[1] / 2, vec[2] / 2}
		}
		return []SolidValue{solid3D(model3d.NewRect(
			model3d.XYZ(min[0], min[1], min[2]),
			model3d.XYZ(max[0], max[1], max[2]),
		))}, nil

	case "cylinder":
		h, err := getNamedOrPosNum(e, st.Call, "h", 0, 1.0)
		if err != nil {
			return nil, err
		}
		r, err := getNamedOrPosNum(e, st.Call, "r", 1, 1.0)
		if err != nil {
			return nil, err
		}
		center, err := getNamedOrPosBool(e, st.Call, "center", 2, false)
		if err != nil {
			return nil, err
		}
		z0 := 0.0
		z1 := h
		if center {
			z0 = -h / 2
			z1 = h / 2
		}
		return []SolidValue{solid3D(&model3d.Cylinder{
			P1:     model3d.XYZ(0, 0, z0),
			P2:     model3d.XYZ(0, 0, z1),
			Radius: r,
		})}, nil

	case "circle":
		r, err := getNamedOrPosNum(e, st.Call, "r", 0, 1.0)
		if err != nil {
			return nil, err
		}
		return []SolidValue{solid2D(&model2d.Circle{Radius: r})}, nil

	case "square":
		sizeV, err := getNamedOrPosValue(e, st.Call, "size", 0, Num(1))
		if err != nil {
			return nil, err
		}
		vec, err := sizeV.AsVec2(st.pos())
		if err != nil {
			return nil, err
		}
		center, err := getNamedOrPosBool(e, st.Call, "center", 1, false)
		if err != nil {
			return nil, err
		}
		min := [2]float64{0, 0}
		max := vec
		if center {
			min = [2]float64{-vec[0] / 2, -vec[1] / 2}
			max = [2]float64{vec[0] / 2, vec[1] / 2}
		}
		return []SolidValue{solid2D(model2d.NewRect(
			model2d.XY(min[0], min[1]),
			model2d.XY(max[0], max[1]),
		))}, nil
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
		return []SolidValue{u}, nil
	}

	return nil, fmt.Errorf("%v: unknown module/primitive %q", st.pos(), name)
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

func unionAll(children []SolidValue) (SolidValue, error) {
	if len(children) == 0 {
		return SolidValue{}, fmt.Errorf("no solids produced")
	}
	if len(children) == 1 {
		return children[0], nil
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return SolidValue{}, err
	}
	switch kind {
	case Solid3D:
		solids := make([]model3d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.Solid3)
		}
		return solid3D(model3d.JoinedSolid(solids)), nil
	case Solid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.Solid2)
		}
		return solid2D(model2d.JoinedSolid(solids)), nil
	default:
		return SolidValue{}, fmt.Errorf("unknown solid kind")
	}
}

func ensureSameKind(children []SolidValue) (SolidKind, error) {
	if len(children) == 0 {
		return Solid3D, fmt.Errorf("no solids produced")
	}
	kind := children[0].Kind
	for _, ch := range children[1:] {
		if ch.Kind != kind {
			return kind, fmt.Errorf("mixed 2D and 3D solids")
		}
	}
	return kind, nil
}

func linearExtrude(s model2d.Solid, height float64, center bool) model3d.Solid {
	if height < 0 {
		height = -height
	}
	z0 := 0.0
	z1 := height
	if center {
		z0 = -height / 2
		z1 = height / 2
	}
	min2 := s.Min()
	max2 := s.Max()
	min := model3d.XYZ(min2.X, min2.Y, z0)
	max := model3d.XYZ(max2.X, max2.Y, z1)
	return model3d.CheckedFuncSolid(min, max, func(c model3d.Coord3D) bool {
		if c.Z < z0 || c.Z > z1 {
			return false
		}
		return s.Contains(model2d.XY(c.X, c.Y))
	})
}

func evalArg0Vec3(e *env, c Call) ([3]float64, error) {
	v, err := getNamedOrPosValue(e, c, "v", 0, List([]Value{Num(0), Num(0), Num(0)}))
	if err != nil {
		return [3]float64{}, err
	}
	return v.AsVec3(c.P)
}

func getNamedOrPosNumAny(e *env, c Call, names []string, pos int, def float64) (float64, error) {
	for _, name := range names {
		for _, a := range c.Args {
			if a.Name == name {
				v, err := evalExpr(e, a.Expr)
				if err != nil {
					return 0, err
				}
				return v.AsNum(c.P)
			}
		}
	}
	return getNamedOrPosNum(e, c, names[0], pos, def)
}

func getNamedOrPosValue(e *env, c Call, name string, pos int, def Value) (Value, error) {
	// named
	for _, a := range c.Args {
		if a.Name == name {
			return evalExpr(e, a.Expr)
		}
	}
	// positional
	npos := 0
	for _, a := range c.Args {
		if a.Name == "" {
			if npos == pos {
				return evalExpr(e, a.Expr)
			}
			npos++
		}
	}
	return def, nil
}

func getNamedOrPosNum(e *env, c Call, name string, pos int, def float64) (float64, error) {
	v, err := getNamedOrPosValue(e, c, name, pos, Num(def))
	if err != nil {
		return 0, err
	}
	return v.AsNum(c.P)
}

func getNamedOrPosBool(e *env, c Call, name string, pos int, def bool) (bool, error) {
	v, err := getNamedOrPosValue(e, c, name, pos, Bool(def))
	if err != nil {
		return false, err
	}
	return v.AsBool(c.P)
}
