package scad

import (
	"fmt"
	"strings"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleMetaball(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind {
	case ShapeSDF2D:
		return shapeMetaball2D(model2d.SDFToMetaball(childUnion.SDF2)), nil
	case ShapeSDF3D:
		return shapeMetaball3D(model3d.SDFToMetaball(childUnion.SDF3)), nil
	default:
		return ShapeRep{}, fmt.Errorf("metaball(): requires an SDF")
	}
}

func handleWeightMetaball(e *env, st *CallStmt, children []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "weight", Pos: 0, Required: true},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	weight, err := argNum(args, "weight")
	if err != nil {
		return ShapeRep{}, err
	}
	if len(children) != 1 {
		return ShapeRep{}, fmt.Errorf("weight_metaball(): requires exactly 1 child")
	}
	return weightMetaball(children[0], weight)
}

func handleMetaballSolid(e *env, st *CallStmt, children []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "threshold", Pos: 0, Required: true},
		{Name: "falloff", Pos: 1, Default: String("quartic")},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	threshold, err := argNum(args, "threshold")
	if err != nil {
		return ShapeRep{}, err
	}
	falloffName, err := argString(args, "falloff")
	if err != nil {
		return ShapeRep{}, err
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return ShapeRep{}, err
	}
	switch kind {
	case ShapeMetaball2D:
		f, err := metaballFalloff2D(falloffName)
		if err != nil {
			return ShapeRep{}, err
		}
		var coll *Metaball2D
		for _, child := range children {
			coll = coll.Join(child.MB2)
		}
		return shapeSolid2D(model2d.WeightedMetaballSolid(f, threshold, coll.Balls, coll.Weights)), nil
	case ShapeMetaball3D:
		f, err := metaballFalloff3D(falloffName)
		if err != nil {
			return ShapeRep{}, err
		}
		var coll *Metaball3D
		for _, child := range children {
			coll = coll.Join(child.MB3)
		}
		return shapeSolid3D(model3d.WeightedMetaballSolid(f, threshold, coll.Balls, coll.Weights)), nil
	default:
		return ShapeRep{}, fmt.Errorf("metaball_solid(): requires metaball children")
	}
}

func weightMetaball(shape ShapeRep, weight float64) (ShapeRep, error) {
	switch shape.Kind {
	case ShapeMetaball2D:
		return ShapeRep{
			Kind: ShapeMetaball2D,
			MB2:  shape.MB2.Scale(weight),
		}, nil
	case ShapeMetaball3D:
		return ShapeRep{
			Kind: ShapeMetaball3D,
			MB3:  shape.MB3.Scale(weight),
		}, nil
	default:
		return ShapeRep{}, fmt.Errorf("weight_metaball(): requires a metaball")
	}
}

func metaballFalloff2D(name string) (model2d.MetaballFalloffFunc, error) {
	switch strings.ToLower(name) {
	case "linear":
		return model2d.LinearMetaballFalloffFunc, nil
	case "quadratic":
		return model2d.QuadraticMetaballFalloffFunc, nil
	case "cubic":
		return model2d.CubicMetaballFalloffFunc, nil
	case "quartic":
		return model2d.QuarticMetaballFalloffFunc, nil
	case "quintic":
		return model2d.QuinticMetaballFalloffFunc, nil
	case "exponential":
		return model2d.ExponentialMetaballFalloffFunc, nil
	case "gaussian":
		return model2d.GaussianMetaballFalloffFunc, nil
	default:
		return nil, fmt.Errorf("metaball_solid(): unknown falloff %q", name)
	}
}

func metaballFalloff3D(name string) (model3d.MetaballFalloffFunc, error) {
	switch strings.ToLower(name) {
	case "linear":
		return model3d.LinearMetaballFalloffFunc, nil
	case "quadratic":
		return model3d.QuadraticMetaballFalloffFunc, nil
	case "cubic":
		return model3d.CubicMetaballFalloffFunc, nil
	case "quartic":
		return model3d.QuarticMetaballFalloffFunc, nil
	case "quintic":
		return model3d.QuinticMetaballFalloffFunc, nil
	case "exponential":
		return model3d.ExponentialMetaballFalloffFunc, nil
	case "gaussian":
		return model3d.GaussianMetaballFalloffFunc, nil
	default:
		return nil, fmt.Errorf("metaball_solid(): unknown falloff %q", name)
	}
}
