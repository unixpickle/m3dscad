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

## Test the UI

From `webui/`, run `npx playwright install chromium` once, then `npm test`.
The browser tests use the real editor, IndexedDB, history, and renderer with a
stubbed compiler worker, so they do not require building WASM.

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

- `Format` reformats the current file using two-space indentation and one modifier
  per line. A trailing comma expands a list onto separate lines. Formatting is a
  single Undo step; syntax errors leave the editor unchanged.
- `Command+S` (or `Ctrl+S`) compiles the code and updates the preview.
- `Files` opens the browser-backed project filesystem. Projects start in
  `main.scad`, and can contain uploaded files, folders, and additional `.scad`
  files for `use <path>` statements. Uploaded PNG, JPEG, and GIF files can be
  loaded with `image()`, and uploaded TTF/OTF fonts with TrueType outlines can
  be selected with the `font` parameter of `text()`, `text_mesh()`, and
  `text_sdf()`. Asset paths are relative to the SCAD file containing the call.
  File contents are stored per path in IndexedDB; the WASM worker reads
  dependencies on demand instead of loading a complete project snapshot.
  `Download .tar` exports the current folder and its descendants, with archive
  paths relative to that folder.
  The URL hash tracks the current file or folder (for example,
  `#/parts/bracket.scad` or `#/parts/`). Browser Back/Forward and reload restore
  that location, and file/folder links can be opened in separate tabs.
  Loading a different file starts a fresh undo history and clears the preview.
  Uploading an existing filename asks whether to replace it; Cancel skips that
  file and continues with the remaining uploads.
- Mesh grid size controls the maximum dual contour cell side length (default 128).
- Drag to orbit.
- Scroll to zoom.
