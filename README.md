# m3dscad

This is a 3D CAD language inspired by OpenSCAD, built on top of [model3d](https://github.com/unixpickle/model3d).

Visit [m3dscad.com](https://m3dscad.com) to see examples, read documentation, and try the web app.

# Structure

The [scad](scad/) directory contains the core interpreter implementation. This does not include rendering code, but does include the language parser, geometry construction logic, and all of the built-in functions.

The [webui](webui/) directory is the browser application, including a harness to run the interpreter through WASM.

The [landing_page](landing_page/) directory contains the homepage of [m3dscad.com](https://m3dscad.com), including code for rendering examples in the browser.

# Tests

Most of the tests should run as is. Some tests compare against OpenSCAD, which require you to generate the reference STL files beforehand with the following command:

```
./scripts/compile_openscad_testdata.sh
```
