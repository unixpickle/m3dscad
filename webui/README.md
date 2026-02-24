# m3dscad WebUI

This is a standalone front-end that runs the m3dscad compiler in WebAssembly and renders a 3D preview in pure WebGL.

## Build the WASM

From the repo root:

```bash
./webui/build_wasm.sh
```

This writes:
- `webui/m3dscad.wasm`
- `webui/wasm_exec.js`

## Run the UI

Serve the `webui/` directory with a static file server, for example:

```bash
python3 -m http.server 8080 --directory webui
```

Then open `http://localhost:8080`.

## Controls

- `Command+S` (or `Ctrl+S`) compiles the code and updates the preview.
- Mesh grid size controls the maximum marching cubes side length (default 128).
- Drag to orbit.
- Scroll to zoom.
