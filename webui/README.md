# m3dscad WebUI

This is a standalone front-end that runs the m3dscad compiler in WebAssembly and renders a 3D preview in pure WebGL.

## Build the JS Bundle

From the repo root:

```bash
./webui/build_js.sh
```

This bundles the app (including CodeMirror) into:
- `webui/main.bundle.js`

## Build the WASM

From the repo root:

```bash
./webui/build_wasm.sh
```

This writes:
- `webui/m3dscad.wasm`
- `webui/wasm_exec.js`

Or run both build steps:

```bash
./webui/build_js.sh
./webui/build_wasm.sh
```

## Run the UI

Serve the `webui/` directory with a static file server, for example:

```bash
python3 -m http.server 8080 --directory webui
```

Then open `http://localhost:8080`.

## Controls

- `Command+S` (or `Ctrl+S`) compiles the code and updates the preview.
- Mesh grid size controls the maximum dual contour cell side length (default 128).
- Drag to orbit.
- Scroll to zoom.
