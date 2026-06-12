package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/model3d/model3d"
	shapekernel "github.com/unixpickle/webgpu-meshes/shapekernel"
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

func (s ShapeKind) Dimension() int {
	switch s {
	case ShapeSolid2D, ShapeMesh2D, ShapeSDF2D, ShapeMetaball2D, ShapeHull2D:
		return 2
	case ShapeSolid3D, ShapeMesh3D, ShapeSDF3D, ShapeMetaball3D:
		return 3
	default:
		panic("unknown ShapeKind")
	}
}

type WeightedMetaballs[T any] struct {
	Balls   []T
	Weights []float64
	Kernels []*shapekernel.ShapeKernel
}

func (m *WeightedMetaballs[T]) Map(fn func(T, *shapekernel.ShapeKernel) (T, *shapekernel.ShapeKernel)) *WeightedMetaballs[T] {
	if m == nil {
		return &WeightedMetaballs[T]{}
	}
	out := &WeightedMetaballs[T]{
		Balls:   make([]T, len(m.Balls)),
		Weights: append([]float64{}, m.Weights...),
		Kernels: append([]*shapekernel.ShapeKernel{}, m.Kernels...),
	}
	for i, mb := range m.Balls {
		out.Balls[i], out.Kernels[i] = fn(mb, out.Kernels[i])
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
		Kernels: append([]*shapekernel.ShapeKernel{}, m.Kernels...),
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
		out.Kernels = append(out.Kernels, m.Kernels...)
	}
	if other != nil {
		out.Balls = append(out.Balls, other.Balls...)
		out.Weights = append(out.Weights, other.Weights...)
		out.Kernels = append(out.Kernels, other.Kernels...)
	}
	return out
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

func (h *Hull2D) Solid(n shapekernel.Numerics) ShapeRep {
	if h == nil || len(h.Circles) == 0 {
		min, max := h.bounds()
		emptySolid := model2d.CheckedFuncSolid(min, max, func(model2d.Coord) bool { return false })
		return shapeSolid2D(emptySolid, asPtr(shapekernel.Empty(n, shapekernel.Solid2D)))
	}
	if h.MaxRadius() > 0 {
		arcHull := h.ArcHull()
		return shapeSolid2D(arcHull, asPtr(shapekernel.ArcHullSolid(n, arcHull)))
	}
	mesh := h.CenterMesh()
	if mesh.NumSegments() > 0 {
		return shapeSolid2D(mesh.Solid(), meshSolidKernel2D(n, mesh))
	}
	min, max := h.bounds()
	center := h.Circles[0].Center
	pointSolid := model2d.CheckedFuncSolid(min, max, func(c model2d.Coord) bool { return c == center })
	return shapeSolid2D(pointSolid, asPtr(shapekernel.ArcHullSolid(n, h.ArcHull())))
}

func (h *Hull2D) SDF(n shapekernel.Numerics) ShapeRep {
	if h == nil || len(h.Circles) == 0 {
		min, max := h.bounds()
		emptySDF := model2d.FuncSDF(min, max, func(model2d.Coord) float64 { return -1 })
		return shapeSDF2D(emptySDF, asPtr(shapekernel.Empty(n, shapekernel.SDF2D)))
	}
	if h.MaxRadius() > 0 {
		arcHull := h.ArcHull()
		return shapeSDF2D(arcHull, asPtr(shapekernel.ArcHullSDF(n, arcHull)))
	}
	mesh := h.CenterMesh()
	if mesh.NumSegments() > 0 {
		return shapeSDF2D(model2d.MeshToSDF(mesh), meshSDFKernel2D(n, mesh))
	}
	min, max := h.bounds()
	center := h.Circles[0].Center
	pointSDF := model2d.FuncSDF(min, max, func(c model2d.Coord) float64 {
		if c == center {
			return 0
		}
		return -c.Dist(center)
	})
	return shapeSDF2D(pointSDF, asPtr(shapekernel.ArcHullSDF(n, h.ArcHull())))
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

	Kernel *shapekernel.ShapeKernel
}

func shapeSolid2D(s model2d.Solid, k *shapekernel.ShapeKernel) ShapeRep {
	return ShapeRep{Kind: ShapeSolid2D, S2: s, Kernel: k}
}

func shapeSolid3D(s model3d.Solid, k *shapekernel.ShapeKernel) ShapeRep {
	return ShapeRep{Kind: ShapeSolid3D, S3: s, Kernel: k}
}

func shapeMesh2D(m *model2d.Mesh) ShapeRep {
	return ShapeRep{Kind: ShapeMesh2D, M2: m}
}

func shapeMesh3D(m *model3d.Mesh) ShapeRep {
	return ShapeRep{Kind: ShapeMesh3D, M3: m}
}

func shapeSDF2D(s model2d.SDF, k *shapekernel.ShapeKernel) ShapeRep {
	return ShapeRep{Kind: ShapeSDF2D, SDF2: s, Kernel: k}
}

func shapeSDF3D(s model3d.SDF, k *shapekernel.ShapeKernel) ShapeRep {
	return ShapeRep{Kind: ShapeSDF3D, SDF3: s, Kernel: k}
}

func shapeHull2D(h *Hull2D) ShapeRep {
	return ShapeRep{Kind: ShapeHull2D, H2: h}
}

func shapeMetaball2D(m model2d.Metaball, k *shapekernel.ShapeKernel) ShapeRep {
	return shapeMultiMetaball2D(&Metaball2D{
		Balls:   []model2d.Metaball{m},
		Weights: []float64{1},
		Kernels: []*shapekernel.ShapeKernel{k},
	})
}

func shapeMultiMetaball2D(m *Metaball2D) ShapeRep {
	return ShapeRep{
		Kind: ShapeMetaball2D,
		MB2:  m,
	}
}

func shapeMetaball3D(m model3d.Metaball, k *shapekernel.ShapeKernel) ShapeRep {
	return shapeMultiMetaball3D(&Metaball3D{
		Balls:   []model3d.Metaball{m},
		Weights: []float64{1},
		Kernels: []*shapekernel.ShapeKernel{k},
	})
}

func shapeMultiMetaball3D(m *Metaball3D) ShapeRep {
	return ShapeRep{
		Kind: ShapeMetaball3D,
		MB3:  m,
	}
}

func SDFToSolid(n shapekernel.Numerics, sdf ShapeRep) ShapeRep {
	switch sdf.Kind {
	case ShapeSDF2D:
		var k *shapekernel.ShapeKernel
		if sdf.Kernel != nil {
			k = asPtr(shapekernel.SDFToSolid(n, *sdf.Kernel))
		}
		return shapeSolid2D(model2d.SDFToSolid(sdf.SDF2, 0), k)
	case ShapeSDF3D:
		var k *shapekernel.ShapeKernel
		if sdf.Kernel != nil {
			k = asPtr(shapekernel.SDFToSolid(n, *sdf.Kernel))
		}
		return shapeSolid3D(model3d.SDFToSolid(sdf.SDF3, 0), k)
	default:
		panic("expected SDF argument")
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
		return shapeSolid2D(
			childUnion.M2.Solid(),
			asPtr(shapekernel.Mesh2DSolid(e.hooks.Numerics, childUnion.M2)),
		), nil
	case ShapeMesh3D:
		return shapeSolid3D(
			childUnion.M3.Solid(),
			asPtr(shapekernel.Mesh3DSolid(e.hooks.Numerics, childUnion.M3)),
		), nil
	case ShapeSDF2D, ShapeSDF3D:
		return SDFToSolid(e.hooks.Numerics, *childUnion), nil
	default:
		return ShapeRep{}, fmt.Errorf("solid(): unsupported shape kind")
	}
}

func unionAll(n shapekernel.Numerics, children []ShapeRep) (ShapeRep, error) {
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
		solids := make(model3d.JoinedSolid, len(children))
		for i, ch := range children {
			solids[i] = ch.S3
		}
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.UnionSolids(n, kernels))
		}
		return shapeSolid3D(solids, k), nil
	case ShapeSolid2D:
		solids := make(model2d.JoinedSolid, len(children))
		for i, ch := range children {
			solids[i] = ch.S2
		}
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.UnionSolids(n, kernels))
		}
		return shapeSolid2D(solids, k), nil
	case ShapeSDF3D:
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.UnionSDFs(n, kernels))
		}
		return shapeSDF3D(sdfUnion3D(children), k), nil
	case ShapeSDF2D:
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.UnionSDFs(n, kernels))
		}
		return shapeSDF2D(sdfUnion2D(children), k), nil
	case ShapeMetaball2D:
		var out *Metaball2D
		for _, ch := range children {
			out = out.Join(ch.MB2)
		}
		return shapeMultiMetaball2D(out), nil
	case ShapeMetaball3D:
		var out *Metaball3D
		for _, ch := range children {
			out = out.Join(ch.MB3)
		}
		return shapeMultiMetaball3D(out), nil
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

func intersectAll(n shapekernel.Numerics, children []ShapeRep) (ShapeRep, error) {
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
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.IntersectSolids(n, kernels))
		}
		return shapeSolid3D(model3d.IntersectedSolid(solids), k), nil
	case ShapeSolid2D:
		solids := make([]model2d.Solid, 0, len(children))
		for _, ch := range children {
			solids = append(solids, ch.S2)
		}
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.IntersectSolids(n, kernels))
		}
		return shapeSolid2D(model2d.IntersectedSolid(solids), k), nil
	case ShapeSDF3D:
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.IntersectSDFs(n, kernels))
		}
		return shapeSDF3D(sdfIntersect3D(children), k), nil
	case ShapeSDF2D:
		kernels, useKernel := concatKernels(children)
		var k *shapekernel.ShapeKernel
		if useKernel {
			k = asPtr(shapekernel.IntersectSDFs(n, kernels))
		}
		return shapeSDF2D(sdfIntersect2D(children), k), nil
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
	return model2d.JoinSDFs(sdfs)
}

func sdfUnion3D(children []ShapeRep) model3d.SDF {
	sdfs := make([]model3d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF3)
	}
	return model3d.JoinSDFs(sdfs)
}

func sdfIntersect2D(children []ShapeRep) model2d.SDF {
	sdfs := make([]model2d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF2)
	}
	return model2d.IntersectSDFs(sdfs)
}

func sdfIntersect3D(children []ShapeRep) model3d.SDF {
	sdfs := make([]model3d.SDF, 0, len(children))
	for _, ch := range children {
		sdfs = append(sdfs, ch.SDF3)
	}
	return model3d.IntersectSDFs(sdfs)
}
