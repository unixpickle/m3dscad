package scad

import "fmt"

// newCacheModule is called each time a declaration is evaluated, so the result
// belongs to that scope instance rather than to the parsed declaration.
func newCacheModule(captured []*scope, body *BlockStmt) moduleDef {
	var result *ShapeRep
	var ready, evaluating bool
	return moduleDef{Native: func(e *env, st *CallStmt) (*ShapeRep, error) {
		if len(st.Call.Args) != 0 {
			return nil, fmt.Errorf("cache %s() does not take arguments", st.Call.Name)
		}
		if len(st.Children) != 0 {
			return nil, fmt.Errorf("cache %s() does not take children", st.Call.Name)
		}
		if ready {
			return result, nil
		}
		if evaluating {
			return nil, fmt.Errorf("recursive evaluation of cache %s()", st.Call.Name)
		}
		evaluating = true
		defer func() { evaluating = false }()
		// A cache has no call-time inputs. Preserve the enclosing scope's
		// children bindings instead of installing new module bindings.
		err := e.withCapturedScopes(captured, func() error {
			var err error
			result, err = evalStmtsAsOne(e, body.Stmts)
			return err
		})
		if err != nil {
			return nil, err
		}
		// An empty result is cached too, so side effects still run only once.
		ready = true
		return result, nil
	}}
}
