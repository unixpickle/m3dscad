package scad

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestUseImportsDirectDefinitions(t *testing.T) {
	files := fstest.MapFS{
		"lib/shapes.scad": {Data: []byte(`
use <private.scad>
radius = 2;
function library_x() = private_x();
module library_shape() {
    translate([private_x(), 0, 0]) sphere(radius);
}
translate([100, 0, 0]) cube(10);
`)},
		"lib/private.scad": {Data: []byte(`
function private_x() = 3;
module private_shape() { cube(1); }
`)},
	}
	prog, err := ParseFile("main.scad", `
use <lib/shapes.scad>
translate([library_x(), 0, 0]) library_shape();
`)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	solid, err := shapeToSolid3D(shape)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := solid.Min().X, 4.0; got != want {
		t.Fatalf("min X: got %v want %v", got, want)
	}
	if got, want := solid.Max().X, 8.0; got != want {
		t.Fatalf("max X: got %v want %v", got, want)
	}
}

func TestUseDoesNotReexportNestedDefinitions(t *testing.T) {
	files := fstest.MapFS{
		"outer.scad": {Data: []byte(`
use <inner.scad>
module outer() { inner(); }
`)},
		"inner.scad": {Data: []byte(`module inner() { cube(1); }`)},
	}
	prog, err := ParseFile("main.scad", `
use <outer.scad>
inner();
`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Eval(prog, Hooks{FS: files})
	if err == nil || !strings.Contains(err.Error(), `unknown module/primitive "inner"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	prog, err = ParseFile("main.scad", `
use <outer.scad>
outer();
`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Eval(prog, Hooks{FS: files}); err != nil {
		t.Fatalf("nested definition should remain visible inside outer file: %v", err)
	}
}

func TestUsedFileCannotSeeCallerDefinitions(t *testing.T) {
	files := fstest.MapFS{
		"library.scad": {Data: []byte(`module library_shape() { caller_shape(); }`)},
	}
	prog, err := ParseFile("main.scad", `
module caller_shape() { cube(1); }
use <library.scad>
library_shape();
`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Eval(prog, Hooks{FS: files})
	if err == nil || !strings.Contains(err.Error(), `unknown module/primitive "caller_shape"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUseDoesNotImportVariablesOrGeometry(t *testing.T) {
	files := fstest.MapFS{
		"library.scad": {Data: []byte(`
secret = 4;
module library_shape() { sphere(secret); }
translate([100, 0, 0]) cube(10);
`)},
	}
	prog, err := ParseFile("main.scad", `
use <library.scad>
cube(secret);
`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Eval(prog, Hooks{FS: files})
	if err == nil || !strings.Contains(err.Error(), `undefined variable "secret"`) {
		t.Fatalf("unexpected error: %v", err)
	}

	prog, err = ParseFile("main.scad", `
use <library.scad>
library_shape();
`)
	if err != nil {
		t.Fatal(err)
	}
	shape, err := Eval(prog, Hooks{FS: files})
	if err != nil {
		t.Fatal(err)
	}
	solid, err := shapeToSolid3D(shape)
	if err != nil {
		t.Fatal(err)
	}
	if solid.Max().X != 4 {
		t.Fatalf("used file's top-level geometry was executed: max X=%v", solid.Max().X)
	}
}

func TestUseErrorsContainBothFilenames(t *testing.T) {
	files := fstest.MapFS{
		"broken.scad": {Data: []byte(`module broken( {`)},
	}
	prog, err := ParseFile("main.scad", `use <broken.scad>`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Eval(prog, Hooks{FS: files})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); !strings.Contains(got, "main.scad:1:1") || !strings.Contains(got, "broken.scad:1:") {
		t.Fatalf("error does not contain both filenames: %q", got)
	}
}

func TestUseOnlyAllowedAtTopLevel(t *testing.T) {
	_, err := ParseFile("main.scad", `module bad() { use <other.scad> }`)
	if err == nil || !strings.Contains(err.Error(), "use is only allowed at the top level") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUseRequiresFSHook(t *testing.T) {
	prog, err := ParseFile("main.scad", `use <library.scad>`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Eval(prog, Hooks{})
	if err == nil || !strings.Contains(err.Error(), "Hooks.FS is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUseRejectsCycles(t *testing.T) {
	files := fstest.MapFS{
		"a.scad": {Data: []byte(`use <b.scad>`)},
		"b.scad": {Data: []byte(`use <a.scad>`)},
	}
	prog, err := ParseFile("a.scad", string(files["a.scad"].Data))
	if err != nil {
		t.Fatal(err)
	}
	_, err = Eval(prog, Hooks{FS: files})
	if err == nil || !strings.Contains(err.Error(), "circular use: a.scad -> b.scad -> a.scad") {
		t.Fatalf("unexpected error: %v", err)
	}
}
