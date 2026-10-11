# m3dscad

This is a 3D CAD language inspired by OpenSCAD, built on top of [model3d](https://github.com/unixpickle/model3d).

Visit [m3dscad.com](https://m3dscad.com) to see examples, read documentation, and try the web app.

# Structure

The [scad](scad/) directory contains the core interpreter implementation. This does not include rendering code, but does include the language parser, geometry construction logic, and all of the built-in functions.

The [webui](webui/) directory is the browser application, including a harness to run the interpreter through WASM.

The [landing_page](landing_page/) directory contains the homepage of [m3dscad.com](https://m3dscad.com), including code for rendering examples in the browser.

# Formatting

Install the standalone formatter from this checkout with:

```sh
go install ./cmd/m3dscadfmt
```

From a checkout, use `go run ./cmd/m3dscadfmt` with the same arguments:

```sh
m3dscadfmt model.scad             # print formatted source
m3dscadfmt -w model.scad          # update a file in place
m3dscadfmt -l *.scad              # list files needing formatting
m3dscadfmt < model.scad           # read standard input
```

The web editor's **Format** button uses the same formatter and supports Undo.
Formatting validates syntax without evaluating code or loading imported files.
Syntax errors leave the source unchanged.

The formatter uses two spaces per indentation level. Each modifier goes on its
own line: the first starts a chain, and the remaining modifiers and final object
align one level deeper. Loops and conditionals introduce their own nesting.
Trailing commas request one item per line in arrays, calls, bindings, and
declaration parameters. Module, named function, and anonymous function parameter
lists all accept trailing commas. Arithmetic spacing highlights precedence, as
in `a*b + c` and `[a+3 : b]`. Comments, literal spellings, and explicit parentheses
are preserved; blank lines between statements are limited to one.

# Tests

Most of the tests should run as is. Some tests compare against OpenSCAD, which require you to generate the reference STL files beforehand with the following command:

```
./scripts/compile_openscad_testdata.sh
```
