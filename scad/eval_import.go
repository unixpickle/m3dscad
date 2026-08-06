package scad

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/unixpickle/model3d/model3d"
)

func handleImport(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "file", Aliases: []string{"filename"}, Pos: 0, Required: true},
		{Name: "convexity", Pos: 1, Default: Num(10)},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	filenameArg, err := argString(args, "file")
	if err != nil {
		return ShapeRep{}, err
	}
	if _, err := argNum(args, "convexity"); err != nil {
		return ShapeRep{}, err
	}
	if e.hooks.FS == nil {
		return ShapeRep{}, fmt.Errorf("cannot import %q: Hooks.FS is not configured", filenameArg)
	}

	filename := strings.ReplaceAll(strings.TrimSpace(filenameArg), "\\", "/")
	base := "."
	if st.P.Filename != "" {
		base = path.Dir(st.P.Filename)
	}
	filename = path.Clean(path.Join(base, filename))
	if !fs.ValidPath(filename) {
		return ShapeRep{}, fmt.Errorf("invalid import path %q", filenameArg)
	}
	if !strings.EqualFold(path.Ext(filename), ".stl") {
		return ShapeRep{}, fmt.Errorf("import(): unsupported file type %q (expected .stl)", path.Ext(filename))
	}

	file, err := e.hooks.FS.Open(filename)
	if err != nil {
		return ShapeRep{}, fmt.Errorf("open imported file %q: %w", filename, err)
	}
	defer file.Close()
	triangles, err := model3d.ReadSTL(file)
	if err != nil {
		return ShapeRep{}, fmt.Errorf("decode imported STL %q: %w", filename, err)
	}
	return shapeMesh3D(model3d.NewMeshTriangles(triangles)), nil
}
