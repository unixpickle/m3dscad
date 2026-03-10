package scad

import (
	"fmt"
	"strings"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

func handleMetaball(_ *env, _ *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	switch childUnion.Kind {
	case ShapeSDF2D:
		return shapeMetaball2D(model2d.SDFToMetaball(childUnion.SDF2)), nil
	case ShapeSDF3D:
		return shapeMetaball3D(model3d.SDFToMetaball(childUnion.SDF3)), nil
	default:
		return ShapeRep{}, fmt.Errorf("metaball(): requires an SDF")
	}
}

func handleNegateMetaball(_ *env, _ *CallStmt, children []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	if len(children) != 1 {
		return ShapeRep{}, fmt.Errorf("negate_metaball(): requires exactly 1 child")
	}
	return negateMetaball(children[0])
}

func handleMetaballSolid(e *env, st *CallStmt, children []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "threshold", Pos: 0, Required: true},
		{Name: "falloff", Pos: 1, Default: String("quartic")},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	threshold, err := argNum(args, "threshold", st.pos())
	if err != nil {
		return ShapeRep{}, err
	}
	falloffName, err := argString(args, "falloff", st.pos())
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
		var pos, neg []model2d.Metaball
		for _, child := range children {
			if child.MB2.Sign {
				pos = append(pos, child.MB2.Metaball)
			} else {
				neg = append(neg, child.MB2.Metaball)
			}
		}
		return shapeSolid2D(model2d.SignedMetaballSolid(f, threshold, pos, neg)), nil
	case ShapeMetaball3D:
		f, err := metaballFalloff3D(falloffName)
		if err != nil {
			return ShapeRep{}, err
		}
		var pos, neg []model3d.Metaball
		for _, child := range children {
			if child.MB3.Sign {
				pos = append(pos, child.MB3.Metaball)
			} else {
				neg = append(neg, child.MB3.Metaball)
			}
		}
		return shapeSolid3D(model3d.SignedMetaballSolid(f, threshold, pos, neg)), nil
	default:
		return ShapeRep{}, fmt.Errorf("metaball_solid(): requires metaball children")
	}
}

func negateMetaball(shape ShapeRep) (ShapeRep, error) {
	switch shape.Kind {
	case ShapeMetaball2D:
		return ShapeRep{
			Kind: ShapeMetaball2D,
			MB2: &Metaball2D{
				Metaball: shape.MB2.Metaball,
				Sign:     !shape.MB2.Sign,
			},
		}, nil
	case ShapeMetaball3D:
		return ShapeRep{
			Kind: ShapeMetaball3D,
			MB3: &Metaball3D{
				Metaball: shape.MB3.Metaball,
				Sign:     !shape.MB3.Sign,
			},
		}, nil
	default:
		return ShapeRep{}, fmt.Errorf("negate_metaball(): requires a metaball")
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
