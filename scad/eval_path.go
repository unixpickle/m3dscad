package scad

import (
	"fmt"

	"github.com/unixpickle/model3d/model2d"
	"github.com/unixpickle/path2d"
)

func handlePath(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parsePathMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSolid2D(mesh.Solid()), nil
}

func handlePathSDF(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parsePathMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeSDF2D(model2d.MeshToSDF(mesh)), nil
}

func handlePathMesh(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	mesh, err := parsePathMesh(e, st)
	if err != nil {
		return ShapeRep{}, err
	}
	return shapeMesh2D(mesh), nil
}

func parsePathMesh(e *env, st *CallStmt) (*model2d.Mesh, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "path", Pos: 0, Required: true},
		{Name: "segments", Pos: 1, Default: Num(1000)},
	})
	if err != nil {
		return nil, err
	}
	path, err := argString(args, "path", st.pos())
	if err != nil {
		return nil, err
	}
	segments, err := argNum(args, "segments", st.pos())
	if err != nil {
		return nil, err
	}
	if float64(int(segments)) != segments {
		return nil, fmt.Errorf("path(): segments must be an integer")
	}
	if segments < 1 {
		return nil, fmt.Errorf("path(): segments must be >= 1")
	}
	curve, err := path2d.ParseSVGPath(path)
	if err != nil {
		return nil, fmt.Errorf("path(): %w", err)
	}
	mesh := model2d.CurveMesh(curve, int(segments))
	if mesh == nil || mesh.NumSegments() == 0 {
		return nil, fmt.Errorf("path(): no segments produced")
	}
	return mesh, nil
}
