package scad

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/unixpickle/model3d/model3d"
)

const (
	openscadTestMaxGridSide = 128
	openscadTestMCIters     = 8
	openscadTestSamples     = 1000
)

func TestOpenSCADMeshParity(t *testing.T) {
	scadDir := filepath.Join("..", "testdata", "openscad_scad")
	stlDir := filepath.Join("..", "testdata", "openscad_stl")
	entries, err := os.ReadDir(scadDir)
	if err != nil {
		t.Fatalf("read scad dir: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".scad") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".scad")
		t.Run(name, func(t *testing.T) {
			scadPath := filepath.Join(scadDir, entry.Name())
			stlPath := filepath.Join(stlDir, name+".stl")
			srcBytes, err := os.ReadFile(scadPath)
			if os.IsNotExist(err) {
				t.Skipf("missing OpenSCAD STL: %s (run scripts/compile_openscad_testdata.sh)", stlPath)
			}
			if err != nil {
				t.Fatalf("read scad: %v", err)
			}
			src := stripLeadingOpenSCADAssignments(string(srcBytes))
			solid := mustEvalSolid(t, src)

			delta := marchingDelta(solid, openscadTestMaxGridSide)
			if delta <= 0 {
				t.Fatalf("invalid marching delta: %v", delta)
			}

			expectedMesh, err := loadSTLMesh(stlPath)
			if err != nil {
				t.Fatalf("load OpenSCAD STL: %v", err)
			}
			actualMesh := model3d.MarchingCubesSearch(solid, delta, openscadTestMCIters)

			threshold := math.Max(3*delta, 0.02)
			rng := rand.New(rand.NewSource(int64(len(name)) * 101))

			compareMeshes(t, "openscad_vs_m3dscad", expectedMesh, actualMesh, threshold, rng)
			compareMeshes(t, "m3dscad_vs_openscad", actualMesh, expectedMesh, threshold, rng)
		})
	}
}

func stripLeadingOpenSCADAssignments(src string) string {
	lines := strings.Split(src, "\n")
	if len(lines) == 0 {
		return src
	}
	first := strings.TrimSpace(lines[0])
	if strings.HasPrefix(first, "$") && strings.Contains(first, "=") && strings.HasSuffix(first, ";") {
		return strings.Join(lines[1:], "\n")
	}
	return src
}

func loadSTLMesh(path string) (*model3d.Mesh, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	tris, err := model3d.ReadSTL(f)
	if err != nil {
		return nil, err
	}
	return model3d.NewMeshTriangles(tris), nil
}

func marchingDelta(solid model3d.Solid, maxSide int) float64 {
	min := solid.Min()
	max := solid.Max()
	size := max.Sub(min)
	maxDim := size.Abs().MaxCoord()
	if maxDim == 0 {
		return 0
	}
	return maxDim / float64(maxSide)
}

func compareMeshes(t *testing.T, label string, source, target *model3d.Mesh, threshold float64, rng *rand.Rand) {
	t.Helper()
	sdf := model3d.MeshToSDF(target)
	sampler, err := newTriangleSampler(source)
	if err != nil {
		t.Fatalf("sampler: %v", err)
	}

	maxDist := 0.0
	for i := 0; i < openscadTestSamples; i++ {
		p := sampler.Sample(rng)
		dist := math.Abs(sdf.SDF(p))
		if dist > maxDist {
			maxDist = dist
		}
		if dist > threshold {
			source.SaveGroupedSTL("/Users/alex/Desktop/out1.stl")
			target.SaveGroupedSTL("/Users/alex/Desktop/out2.stl")
			t.Fatalf("%s: surface distance %.6f exceeds threshold %.6f", label, dist, threshold)

		}
	}
}

type triangleSampler struct {
	tris     []*model3d.Triangle
	cumAreas []float64
	total    float64
}

func newTriangleSampler(mesh *model3d.Mesh) (*triangleSampler, error) {
	rawTris := mesh.TriangleSlice()
	if len(rawTris) == 0 {
		return nil, fmt.Errorf("mesh has no triangles")
	}
	tris := make([]*model3d.Triangle, 0, len(rawTris))
	cum := make([]float64, 0, len(rawTris))
	total := 0.0
	for _, t := range rawTris {
		area := t.Area()
		if area <= 0 {
			continue
		}
		total += area
		tris = append(tris, t)
		cum = append(cum, total)
	}
	if total == 0 {
		return nil, fmt.Errorf("mesh has zero area")
	}
	return &triangleSampler{tris: tris, cumAreas: cum, total: total}, nil
}

func (s *triangleSampler) Sample(rng *rand.Rand) model3d.Coord3D {
	r := rng.Float64() * s.total
	idx := sort.SearchFloat64s(s.cumAreas, r)
	if idx >= len(s.tris) {
		idx = len(s.tris) - 1
	}
	tri := s.tris[idx]
	return tri.AtBarycentric(randomBarycentric(rng))
}

func randomBarycentric(rng *rand.Rand) [3]float64 {
	r1 := rng.Float64()
	r2 := rng.Float64()
	sqrtR1 := math.Sqrt(r1)
	return [3]float64{
		1 - sqrtR1,
		sqrtR1 * (1 - r2),
		sqrtR1 * r2,
	}
}
