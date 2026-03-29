package scad

import "fmt"

type ArgSpec struct {
	Name     string
	Aliases  []string
	Pos      int
	Default  Value
	Required bool
}

type boundArgs struct {
	Values        map[string]Value
	Provided      map[string]bool
	NamedProvided map[string]bool
}

func bindArgs(e *env, c Call, specs []ArgSpec) (map[string]Value, error) {
	res, err := bindArgsDetailed(e, c, specs)
	if err != nil {
		return nil, err
	}
	return res.Values, nil
}

func bindArgsDetailed(e *env, c Call, specs []ArgSpec) (*boundArgs, error) {
	nameToCanonical := make(map[string]string, len(specs))
	maxPos := -1
	for _, spec := range specs {
		if _, exists := nameToCanonical[spec.Name]; exists {
			return nil, fmt.Errorf("%s(): duplicate arg spec %q", c.Name, spec.Name)
		}
		nameToCanonical[spec.Name] = spec.Name
		for _, alias := range spec.Aliases {
			if _, exists := nameToCanonical[alias]; exists {
				return nil, fmt.Errorf("%s(): duplicate arg spec alias %q", c.Name, alias)
			}
			nameToCanonical[alias] = spec.Name
		}
		if spec.Pos > maxPos {
			maxPos = spec.Pos
		}
	}

	named := make(map[string]Value, len(c.Args))
	positional := make([]Value, 0, len(c.Args))
	namedProvided := make(map[string]bool, len(specs))
	seenNamed := false
	for _, a := range c.Args {
		v, err := evalExpr(e, a.Expr)
		if err != nil {
			return nil, err
		}
		if a.Name != "" {
			seenNamed = true
			canonical, ok := nameToCanonical[a.Name]
			if !ok {
				return nil, fmt.Errorf("%s(): unknown argument %q", c.Name, a.Name)
			}
			if _, exists := named[canonical]; exists {
				return nil, fmt.Errorf("%s(): duplicate argument %q", c.Name, a.Name)
			}
			named[canonical] = v
			namedProvided[canonical] = true
			continue
		}
		if seenNamed {
			return nil, fmt.Errorf("%s(): positional args cannot follow named args", c.Name)
		}
		positional = append(positional, v)
	}
	if maxPos >= 0 {
		if len(positional) > maxPos+1 {
			return nil, fmt.Errorf("%s(): too many positional args", c.Name)
		}
	} else if len(positional) > 0 {
		return nil, fmt.Errorf("%s(): too many positional args", c.Name)
	}

	out := make(map[string]Value, len(specs))
	provided := make(map[string]bool, len(specs))
	for _, spec := range specs {
		v, ok := named[spec.Name]
		if !ok && spec.Pos >= 0 && spec.Pos < len(positional) {
			v = positional[spec.Pos]
			ok = true
		}
		if !ok {
			if spec.Required {
				return nil, fmt.Errorf("missing parameter %q", spec.Name)
			}
			v = spec.Default
		}
		out[spec.Name] = v
		provided[spec.Name] = ok
	}
	return &boundArgs{
		Values:        out,
		Provided:      provided,
		NamedProvided: namedProvided,
	}, nil
}

func argNum(args map[string]Value, name string) (float64, error) {
	v, ok := args[name]
	if !ok {
		return 0, fmt.Errorf("missing parameter %q", name)
	}
	return v.AsNum()
}

func argBool(args map[string]Value, name string) (bool, error) {
	v, ok := args[name]
	if !ok {
		return false, fmt.Errorf("missing parameter %q", name)
	}
	return v.AsBool()
}

func argString(args map[string]Value, name string) (string, error) {
	v, ok := args[name]
	if !ok {
		return "", fmt.Errorf("missing parameter %q", name)
	}
	return v.AsString()
}

func argVec3(args map[string]Value, name string) ([3]float64, error) {
	v, ok := args[name]
	if !ok {
		return [3]float64{}, fmt.Errorf("missing parameter %q", name)
	}
	return v.AsVec3()
}

func argVec2(args map[string]Value, name string) ([2]float64, error) {
	v, ok := args[name]
	if !ok {
		return [2]float64{}, fmt.Errorf("missing parameter %q", name)
	}
	return v.AsVec2()
}
