package scad

import "fmt"

type ArgSpec struct {
	Name     string
	Aliases  []string
	Pos      int
	Default  Value
	Required bool
}

func bindArgs(e *env, c Call, specs []ArgSpec) (map[string]Value, error) {
	named := make(map[string]Value, len(c.Args))
	positional := make([]Value, 0, len(c.Args))
	for _, a := range c.Args {
		v, err := evalExpr(e, a.Expr)
		if err != nil {
			return nil, err
		}
		if a.Name != "" {
			named[a.Name] = v
		} else {
			positional = append(positional, v)
		}
	}

	out := make(map[string]Value, len(specs))
	for _, spec := range specs {
		v, ok := named[spec.Name]
		if !ok {
			for _, alias := range spec.Aliases {
				if val, found := named[alias]; found {
					v = val
					ok = true
					break
				}
			}
		}
		if !ok && spec.Pos >= 0 && spec.Pos < len(positional) {
			v = positional[spec.Pos]
			ok = true
		}
		if !ok {
			if spec.Required {
				return nil, fmt.Errorf("%v: missing parameter %q", c.P, spec.Name)
			}
			v = spec.Default
		}
		out[spec.Name] = v
	}
	return out, nil
}

func argNum(args map[string]Value, name string, pos Pos) (float64, error) {
	v, ok := args[name]
	if !ok {
		return 0, fmt.Errorf("%v: missing parameter %q", pos, name)
	}
	return v.AsNum(pos)
}

func argBool(args map[string]Value, name string, pos Pos) (bool, error) {
	v, ok := args[name]
	if !ok {
		return false, fmt.Errorf("%v: missing parameter %q", pos, name)
	}
	return v.AsBool(pos)
}

func argString(args map[string]Value, name string, pos Pos) (string, error) {
	v, ok := args[name]
	if !ok {
		return "", fmt.Errorf("%v: missing parameter %q", pos, name)
	}
	return v.AsString(pos)
}

func argVec3(args map[string]Value, name string, pos Pos) ([3]float64, error) {
	v, ok := args[name]
	if !ok {
		return [3]float64{}, fmt.Errorf("%v: missing parameter %q", pos, name)
	}
	return v.AsVec3(pos)
}

func argVec2(args map[string]Value, name string, pos Pos) ([2]float64, error) {
	v, ok := args[name]
	if !ok {
		return [2]float64{}, fmt.Errorf("%v: missing parameter %q", pos, name)
	}
	return v.AsVec2(pos)
}
