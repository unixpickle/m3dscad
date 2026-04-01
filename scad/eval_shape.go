package scad

import (
	"fmt"
	"math"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
)

type ShapeKind int

const (
	ShapeSolid2D ShapeKind = iota
	ShapeSolid3D
	ShapeMesh2D
	ShapeMesh3D
	ShapeSDF2D
	ShapeSDF3D
	ShapeMetaball2D
	ShapeMetaball3D
	ShapeHull2D
)

type WeightedMetaballs[T any] struct {
	Balls   []T
	Weights []float64
}

type Metaball2D = WeightedMetaballs[model2d.Metaball]
type Metaball3D = WeightedMetaballs[model3d.Metaball]

type Hull2D struct {
	Circles []*model2d.Circle
}

func (h *Hull2D) ArcHull() *model2d.ArcHull {
	if h == nil {
		return model2d.NewArcHull(nil)
	}
	return model2d.NewArcHull(h.Circles)
}

func (h *Hull2D) MaxRadius() float64 {
	if h == nil {
		return 0
	}
	maxRadius := 0.0
	for _, c := range h.Circles {
		if c.Radius > maxRadius {
			maxRadius = c.Radius
		}
	}
	return maxRadius
}

func (h *Hull2D) CenterMesh() *model2d.Mesh {
	if h == nil {
		return model2d.NewMesh()
	}
	centers := make([]model2d.Coord, 0, len(h.Circles))
	for _, c := range h.Circles {
		centers = append(centers, c.Center)
	}
	return model2d.ConvexHullMesh(centers)
}

func (h *Hull2D) bounds() (model2d.Coord, model2d.Coord) {
	if h == nil || len(h.Circles) == 0 {
		return model2d.Coord{}, model2d.Coord{}
	}
	min := h.Circles[0].Center
	max := h.Circles[0].Center
	for _, c := range h.Circles[1:] {
		min = min.Min(c.Center)
		max = max.Max(c.Center)
	}
	return min, max
}

func (h *Hull2D) Solid() model2d.Solid {
	if h.MaxRadius() > 0 {
		return h.ArcHull()
	}
	mesh := h.CenterMesh()
	if mesh.NumSegments() > 0 {
		return mesh.Solid()
	}
	min, max := h.bounds()
	if h == nil || len(h.Circles) == 0 {
		return model2d.CheckedFuncSolid(min, max, func(model2d.Coord) bool { return false })
	}
	center := h.Circles[0].Center
	return model2d.CheckedFuncSolid(min, max, func(c model2d.Coord) bool { return c == center })
}

func (h *Hull2D) SDF() model2d.SDF {
	if h.MaxRadius() > 0 {
		return h.ArcHull()
	}
	mesh := h.CenterMesh()
	if mesh.NumSegments() > 0 {
		return model2d.MeshToSDF(mesh)
	}
	min, max := h.bounds()
	if h == nil || len(h.Circles) == 0 {
		return model2d.FuncSDF(min, max, func(model2d.Coord) float64 { return -1 })
	}
	center := h.Circles[0].Center
	return model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
		if c == center {
			return 0
		}
		return -c.Dist(center)
	})
}

func (h *Hull2D) Map(fn func(*model2d.Circle) *model2d.Circle) *Hull2D {
	if h == nil {
		return &Hull2D{}
	}
	out := &Hull2D{Circles: make([]*model2d.Circle, len(h.Circles))}
	for i, c := range h.Circles {
		out.Circles[i] = fn(c)
	}
	return out
}

func (h *Hull2D) Join(other *Hull2D) *Hull2D {
	out := &Hull2D{}
	if h != nil {
		out.Circles = append(out.Circles, h.Circles...)
	}
	if other != nil {
		out.Circles = append(out.Circles, other.Circles...)
	}
	return out
}

func (m *WeightedMetaballs[T]) Map(fn func(T) T) *WeightedMetaballs[T] {
	if m == nil {
		return &WeightedMetaballs[T]{}
	}
	out := &WeightedMetaballs[T]{
		Balls:   make([]T, len(m.Balls)),
		Weights: append([]float64{}, m.Weights...),
	}
	for i, mb := range m.Balls {
		out.Balls[i] = fn(mb)
	}
	return out
}

func (m *WeightedMetaballs[T]) Scale(weight float64) *WeightedMetaballs[T] {
	if m == nil {
		return &WeightedMetaballs[T]{}
	}
	out := &WeightedMetaballs[T]{
		Balls:   append([]T{}, m.Balls...),
		Weights: make([]float64, len(m.Weights)),
	}
	for i, w := range m.Weights {
		out.Weights[i] = w * weight
	}
	return out
}

func (m *WeightedMetaballs[T]) Join(other *WeightedMetaballs[T]) *WeightedMetaballs[T] {
	out := &WeightedMetaballs[T]{}
	if m != nil {
		out.Balls = append(out.Balls, m.Balls...)
		out.Weights = append(out.Weights, m.Weights...)
	}
	if other != nil {
		out.Balls = append(out.Balls, other.Balls...)
		out.Weights = append(out.Weights, other.Weights...)
	}
	return out
}

type ShapeRep struct {
	Kind ShapeKind
	S2   model2d.Solid
	S3   model3d.Solid
	M2   *model2d.Mesh
	M3   *model3d.Mesh
	SDF2 model2d.SDF
	SDF3 model3d.SDF
	MB2  *Metaball2D
	MB3  *Metaball3D
	H2   *Hull2D
}

func shapeSolid2D(s model2d.Solid) ShapeRep { return ShapeRep{Kind: ShapeSolid2D, S2: s} }
func shapeSolid3D(s model3d.Solid) ShapeRep { return ShapeRep{Kind: ShapeSolid3D, S3: s} }
func shapeMesh2D(m *model2d.Mesh) ShapeRep  { return ShapeRep{Kind: ShapeMesh2D, M2: m} }
func shapeMesh3D(m *model3d.Mesh) ShapeRep  { return ShapeRep{Kind: ShapeMesh3D, M3: m} }
func shapeSDF2D(s model2d.SDF) ShapeRep     { return ShapeRep{Kind: ShapeSDF2D, SDF2: s} }
func shapeSDF3D(s model3d.SDF) ShapeRep     { return ShapeRep{Kind: ShapeSDF3D, SDF3: s} }
func shapeHull2D(h *Hull2D) ShapeRep        { return ShapeRep{Kind: ShapeHull2D, H2: h} }
func shapeMetaball2D(m model2d.Metaball) ShapeRep {
	return ShapeRep{
		Kind: ShapeMetaball2D,
		MB2: &Metaball2D{
			Balls:   []model2d.Metaball{m},
			Weights: []float64{1},
		},
	}
}
func shapeMetaball3D(m model3d.Metaball) ShapeRep {
	return ShapeRep{
		Kind: ShapeMetaball3D,
		MB3: &Metaball3D{
			Balls:   []model3d.Metaball{m},
			Weights: []float64{1},
		},
	}
}

func handleSolid(e *env, st *CallStmt, _ []ShapeRep, childUnion *ShapeRep) (ShapeRep, error) {
	if _, err := bindArgs(e, st.Call, []ArgSpec{}); err != nil {
		return ShapeRep{}, err
	}
	switch childUnion.Kind {
	case ShapeSolid2D, ShapeSolid3D:
		return *childUnion, nil
	case ShapeMesh2D:
		return shapeSolid2D(childUnion.M2.Solid()), nil
	case ShapeMesh3D:
		return shapeSolid3D(childUnion.M3.Solid()), nil
	case ShapeSDF2D:
		return shapeSolid2D(model2d.SDFToSolid(childUnion.SDF2, 0)), nil
	case ShapeSDF3D:
		return shapeSolid3D(model3d.SDFToSolid(childUnion.SDF3, 0)), nil
	default:
		return ShapeRep{}, fmt.Errorf("solid(): unsupported shape kind")
	}
}

func unionAll(children []ShapeRep) (ShapeRep, error) {
	if len(children) == 0 {
		return ShapeRep{}, fmt.Errorf("no shapes produced")
	}
	if len(children) == 1 {
		return children[0], nil
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return ShapeRep{}, err
	}
	switch kind {
	case ShapeSolid3D:
		solids := make([]model3d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S3)
		}
		return shapeSolid3D(model3d.JoinedSolid(solids)), nil
	case ShapeSolid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S2)
		}
		return shapeSolid2D(model2d.JoinedSolid(solids)), nil
	case ShapeSDF3D:
		return shapeSDF3D(sdfUnion3D(children)), nil
	case ShapeSDF2D:
		return shapeSDF2D(sdfUnion2D(children)), nil
	case ShapeMetaball2D:
		var out *Metaball2D
		for _, ch := range children {
			out = out.Join(ch.MB2)
		}
		return ShapeRep{Kind: ShapeMetaball2D, MB2: out}, nil
	case ShapeMetaball3D:
		var out *Metaball3D
		for _, ch := range children {
			out = out.Join(ch.MB3)
		}
		return ShapeRep{Kind: ShapeMetaball3D, MB3: out}, nil
	case ShapeHull2D:
		var out *Hull2D
		for _, ch := range children {
			out = out.Join(ch.H2)
		}
		return shapeHull2D(out), nil
	case ShapeMesh3D:
		out := model3d.NewMesh()
		for _, ch := range children {
			out.AddMesh(ch.M3)
		}
		return shapeMesh3D(out), nil
	case ShapeMesh2D:
		out := model2d.NewMesh()
		for _, ch := range children {
			out.AddMesh(ch.M2)
		}
		return shapeMesh2D(out), nil
	default:
		return ShapeRep{}, fmt.Errorf("unknown shape kind")
	}
}

func intersectAll(children []ShapeRep) (ShapeRep, error) {
	if len(children) == 0 {
		return ShapeRep{}, fmt.Errorf("no shapes produced")
	}
	kind, err := ensureSameKind(children)
	if err != nil {
		return ShapeRep{}, err
	}
	switch kind {
	case ShapeSolid3D:
		solids := make([]model3d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S3)
		}
		return shapeSolid3D(model3d.IntersectedSolid(solids)), nil
	case ShapeSolid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S2)
		}
		return shapeSolid2D(model2d.IntersectedSolid(solids)), nil
	case ShapeSDF3D:
		return shapeSDF3D(sdfIntersect3D(children)), nil
	case ShapeSDF2D:
		return shapeSDF2D(sdfIntersect2D(children)), nil
	case ShapeMesh2D, ShapeMesh3D:
		return ShapeRep{}, fmt.Errorf("intersection() not supported for meshes")
	default:
		return ShapeRep{}, fmt.Errorf("intersection(): unknown shape kind")
	}
}

func ensureSameKind(children []ShapeRep) (ShapeKind, error) {
	if len(children) == 0 {
		return ShapeSolid3D, fmt.Errorf("no shapes produced")
	}
	kind := children[0].Kind
	for _, ch := range children[1:] {
		if ch.Kind != kind {
			return kind, fmt.Errorf("mixed shape kinds")
		}
	}
	return kind, nil
}

func sdfUnion2D(children []ShapeRep) model2d.SDF {
	sdfs := make([]model2d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF2)
	}
	min, max := model2d.BoundsUnion(sdfs)
	return model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
		val := sdfs[0].SDF(c)
		for _, s := range sdfs[1:] {
			val = math.Max(val, s.SDF(c))
		}
		return val
	})
}

func sdfUnion3D(children []ShapeRep) model3d.SDF {
	sdfs := make([]model3d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF3)
	}
	min, max := model3d.BoundsUnion(sdfs)
	return model3d.FuncSDF(min, max, func(c model3d.Coord3D) float64 {
		val := sdfs[0].SDF(c)
		for _, s := range sdfs[1:] {
			val = math.Max(val, s.SDF(c))
		}
		return val
	})
}

func sdfIntersect2D(children []ShapeRep) model2d.SDF {
	sdfs := make([]model2d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF2)
	}
	min, max := boundsIntersect2D(sdfs)
	return model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
		val := sdfs[0].SDF(c)
		for _, s := range sdfs[1:] {
			val = math.Min(val, s.SDF(c))
		}
		return val
	})
}

func sdfIntersect3D(children []ShapeRep) model3d.SDF {
	sdfs := make([]model3d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF3)
	}
	min, max := boundsIntersect3D(sdfs)
	return model3d.FuncSDF(min, max, func(c model3d.Coord3D) float64 {
		val := sdfs[0].SDF(c)
		for _, s := range sdfs[1:] {
			val = math.Min(val, s.SDF(c))
		}
		return val
	})
}

func sdfSubtract2D(a, b ShapeRep) model2d.SDF {
	min := a.SDF2.Min()
	max := a.SDF2.Max()
	return model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
		return math.Min(a.SDF2.SDF(c), -b.SDF2.SDF(c))
	})
}

func sdfSubtract3D(a, b ShapeRep) model3d.SDF {
	min := a.SDF3.Min()
	max := a.SDF3.Max()
	return model3d.FuncSDF(min, max, func(c model3d.Coord3D) float64 {
		return math.Min(a.SDF3.SDF(c), -b.SDF3.SDF(c))
	})
}

func boundsIntersect2D[B model2d.Bounder](bounds []B) (min, max model2d.Coord) {
	min = bounds[0].Min()
	max = bounds[0].Max()
	for _, b := range bounds[1:] {
		min = min.Max(b.Min())
		max = max.Min(b.Max())
	}
	return min, max
}

func boundsIntersect3D[B model3d.Bounder](bounds []B) (min, max model3d.Coord3D) {
	min = bounds[0].Min()
	max = bounds[0].Max()
	for _, b := range bounds[1:] {
		min = min.Max(b.Min())
		max = max.Min(b.Max())
	}
	return min, max
}
