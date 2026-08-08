package scad

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"path"
	"strings"

	"github.com/unixpickle/model3d/model2d"
)

func handleImage(e *env, st *CallStmt, _ []ShapeRep, _ *ShapeRep) (ShapeRep, error) {
	args, err := bindArgs(e, st.Call, []ArgSpec{
		{Name: "file", Aliases: []string{"filename"}, Pos: 0, Required: true},
	})
	if err != nil {
		return ShapeRep{}, err
	}
	filenameArg, err := argString(args, "file")
	if err != nil {
		return ShapeRep{}, err
	}
	if e.hooks.FS == nil {
		return ShapeRep{}, fmt.Errorf("cannot load image %q: Hooks.FS is not configured", filenameArg)
	}

	filename := strings.ReplaceAll(strings.TrimSpace(filenameArg), "\\", "/")
	base := "."
	if st.P.Filename != "" {
		base = path.Dir(st.P.Filename)
	}
	filename = path.Clean(path.Join(base, filename))
	if !fs.ValidPath(filename) {
		return ShapeRep{}, fmt.Errorf("invalid image path %q", filenameArg)
	}

	file, err := e.hooks.FS.Open(filename)
	if err != nil {
		return ShapeRep{}, fmt.Errorf("open image %q: %w", filename, err)
	}
	defer file.Close()
	// ReadBitmap uses os.Open internally. Decode from Hooks.FS and call the
	// same nil-ColorBitFunc constructor so this also works in WebAssembly.
	decoded, _, err := image.Decode(file)
	if err != nil {
		return ShapeRep{}, fmt.Errorf("decode image %q: %w", filename, err)
	}
	return shapeMesh2D(model2d.NewBitmapImage(decoded, nil).Mesh()), nil
}
