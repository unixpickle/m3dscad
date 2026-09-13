package scad

import "fmt"

// newChildrenModule captures the call site's scope, independently of the
// enclosing module's lexical scope. Children are evaluated only when selected.
func newChildrenModule(caller *env, stmts []Stmt) (moduleDef, int) {
	var setup, children []Stmt
	for _, st := range stmts {
		switch st.(type) {
		case *UseStmt, *ModuleDefStmt, *FuncDefStmt, *AssignStmt:
			setup = append(setup, st)
		default:
			children = append(children, st)
		}
	}
	return moduleDef{Native: func(e *env, st *CallStmt) (*ShapeRep, error) {
		if len(st.Children) > 0 {
			return nil, fmt.Errorf("children() does not take children")
		}
		args, err := bindArgsDetailed(e, st.Call, []ArgSpec{
			{Name: "index", Pos: 0},
		})
		if err != nil {
			return nil, err
		}
		selected := children
		if args.Provided["index"] {
			n, err := argNum(args.Values, "index")
			if err != nil {
				return nil, err
			}
			i := int(n)
			if float64(i) != n {
				return nil, fmt.Errorf("children(): index must be an integer")
			}
			if i < 0 || i >= len(children) {
				return nil, fmt.Errorf("children(): index out of range")
			}
			selected = children[i : i+1]
		}
		// Give each evaluation its own child-block scope so declarations do not
		// leak into the caller or collide on repeated children() calls.
		childEnv := caller.Clone()
		childEnv.push()
		ss := append(append([]Stmt{}, setup...), selected...)
		return evalStmtsAsOne(childEnv, ss)
	}}, len(children)
}
