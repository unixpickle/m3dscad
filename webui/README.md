# m3dscad WebUI

This is a standalone front-end that runs the m3dscad compiler in WebAssembly and renders a 3D preview in pure WebGL.

## Install Dependencies

From the repo root:

```bash
cd webui
npm install
```

## Build the JS Bundle

From the repo root:

```bash
./webui/build_js.sh
```

This builds the app (including CodeMirror) into:
- `webui/dist/`

## Run the Dev Server

From `webui/`:

```bash
npm run dev
```

Then open the local URL printed by Vite.

## Build the WASM

From the repo root:

```bash
./webui/build_wasm.sh
```

This writes:
- `webui/public/m3dscad.wasm`
- `webui/public/wasm_exec.js`

Or run both build steps:

```bash
cd webui && npm install && cd ..
./webui/build_js.sh
./webui/build_wasm.sh
```

## Run the UI

Serve the `webui/` directory with a static file server, for example:

```bash
python3 -m http.server 8080 --directory webui/dist
```

Then open `http://localhost:8080`.

## Controls

- `Command+S` (or `Ctrl+S`) compiles the code and updates the preview.
- Mesh grid size controls the maximum dual contour cell side length (default 128).
- Drag to orbit.
- Scroll to zoom.
