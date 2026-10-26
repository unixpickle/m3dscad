import "../style.css";
import { DEFAULT_SOURCE, createEditor } from "./editor";
import { buildBinarySTL } from "./export/stl";
import { BrowserFileSystem, type FileSystemEntryKind } from "./filesystem";
import { hashPath, parentDirectory, pathHash } from "./navigation";
import { MeshRenderer } from "./renderer/mesh_renderer";
import type { AxisElements, MeshBackend } from "./types";
import { createFileBrowser } from "./ui/file_browser";
import { setupMobileToggle, setupResizer } from "./ui/layout";
import { createOverlayController } from "./ui/overlay";
import { isWebGPUSupported } from "./webgpu/meshing";
import {
  COMPILATION_CANCELED_ERROR,
  GO_EXITED_ERROR,
  createWorkerClient,
} from "./worker_client";

const MESH_BACKEND_STORAGE_KEY = "m3dscad_mesh_backend";
const LAST_OPEN_FILE_STORAGE_KEY = "m3dscad_last_open_file";

interface ExportMesh {
  positions: Float32Array;
  normals: Float32Array;
}

function requireElement<T extends HTMLElement = HTMLElement>(id: string): T {
  const element = document.getElementById(id);
  if (!element) {
    throw new Error(`Missing required element #${id}`);
  }
  return element as unknown as T;
}

function requireSVGElement<T extends SVGElement = SVGElement>(id: string): T {
  const element = document.getElementById(id);
  if (!element) {
    throw new Error(`Missing required element #${id}`);
  }
  return element as unknown as T;
}

const codeHostEl = requireElement("code");
const statusEl = requireElement("status");
const overlayCard = requireElement("overlayCard");
const overlayText = requireElement("overlayText");
const spinnerEl = requireElement("spinner");
const cancelBtn = requireElement<HTMLButtonElement>("cancel");
const compileBtn = requireElement<HTMLButtonElement>("compile");
const formatBtn = requireElement<HTMLButtonElement>("format");
const resetViewBtn = requireElement<HTMLButtonElement>("resetView");
const downloadBtn = requireElement<HTMLButtonElement>("download");
const gridEl = requireElement<HTMLInputElement>("grid");
const meshBackendEl = requireElement<HTMLSelectElement>("meshBackend");
const canvas = requireElement<HTMLCanvasElement>("preview");
const resizer = requireElement("resizer");
const appEl = requireElement("app");
const toggleCodeBtn = requireElement<HTMLButtonElement>("toggleCode");
const togglePreviewBtn = requireElement<HTMLButtonElement>("togglePreview");
const fileBrowserEl = requireElement("fileBrowser");
const editorFilenameEl = requireElement("editorFilename");
const closeFilesBtn = requireElement<HTMLButtonElement>("closeFiles");

const fileSystem = await BrowserFileSystem.open(DEFAULT_SOURCE);
let activeFilename: string | null = null;
let fileRevision = 0;
let navigationRevision = 0;
editorFilenameEl.textContent = "No file open";
closeFilesBtn.disabled = true;

const axisElements: AxisElements = {
  axisLineX: requireSVGElement<SVGLineElement>("axisLineX"),
  axisLineY: requireSVGElement<SVGLineElement>("axisLineY"),
  axisLineZ: requireSVGElement<SVGLineElement>("axisLineZ"),
  axisLabelX: requireSVGElement<SVGTextElement>("axisLabelX"),
  axisLabelY: requireSVGElement<SVGTextElement>("axisLabelY"),
  axisLabelZ: requireSVGElement<SVGTextElement>("axisLabelZ"),
};

const overlay = createOverlayController({
  overlayCard,
  overlayText,
  spinnerEl,
  cancelBtn,
});

const webgpuSupported = isWebGPUSupported();
const storedMeshBackend = window.localStorage.getItem(MESH_BACKEND_STORAGE_KEY);
meshBackendEl.value =
  webgpuSupported && isMeshBackend(storedMeshBackend)
    ? storedMeshBackend
    : "cpu";
for (const option of meshBackendEl.options) {
  if (option.value !== "cpu") {
    option.disabled = !webgpuSupported;
  }
}
if (!webgpuSupported) {
  meshBackendEl.title = "WebGPU is unavailable in this browser.";
}
meshBackendEl.addEventListener("change", () => {
  window.localStorage.setItem(MESH_BACKEND_STORAGE_KEY, meshBackendEl.value);
});

function isMeshBackend(value: string | null): value is MeshBackend {
  return value === "cpu" || value === "gpu_f32" || value === "gpu_fixed64";
}

function clearMeshResult(): void {
  lastMesh = null;
  downloadBtn.disabled = true;
}

function handleWorkerError(errText: string): void {
  statusEl.textContent = errText;
  overlay.set(errText, { idle: true });
  clearMeshResult();
}

const editor = createEditor({
  parent: codeHostEl,
  initialSource: "",
  onSave: () => {
    void compile();
  },
  onChange: (source) => {
    const filename = activeFilename;
    if (filename) {
      void fileSystem.writeTextFile(filename, source);
    }
  },
});

const fileBrowser = createFileBrowser({
  fileSystem,
  elements: {
    browser: fileBrowserEl,
    app: appEl,
    closeButton: closeFilesBtn,
    openButton: requireElement<HTMLButtonElement>("openFiles"),
    newFileButton: requireElement<HTMLButtonElement>("newFile"),
    newDirectoryButton: requireElement<HTMLButtonElement>("newDirectory"),
    uploadButton: requireElement<HTMLButtonElement>("uploadFiles"),
    downloadDirectoryButton:
      requireElement<HTMLButtonElement>("downloadDirectory"),
    uploadInput: requireElement<HTMLInputElement>("fileUploadInput"),
    breadcrumbs: requireElement("fileBreadcrumbs"),
    fileList: requireElement("fileList"),
    status: requireElement("fileBrowserStatus"),
  },
  onOpen: () => {
    void navigate(parentDirectory(activeFilename || ""), "directory");
  },
  onClose: () => {
    if (activeFilename) {
      void navigate(activeFilename, "file");
    }
  },
  onEditFile: (filename) => navigate(filename, "file"),
  onWriteFile: async (filename) => {
    if (activeFilename === filename) {
      const revision = fileRevision;
      const source = await fileSystem.readTextFile(filename);
      if (revision === fileRevision) {
        loadEditor(filename, source);
      }
    }
  },
  onRenameEntry: (oldPath, newPath, kind) => {
    if (!activeFilename) {
      return;
    }
    if (
      activeFilename !== oldPath &&
      !(kind === "directory" && activeFilename.startsWith(`${oldPath}/`))
    ) {
      return;
    }
    activeFilename = `${newPath}${activeFilename.slice(oldPath.length)}`;
    resetFileResult();
    window.localStorage.setItem(LAST_OPEN_FILE_STORAGE_KEY, activeFilename);
    editorFilenameEl.textContent = activeFilename;
  },
  onDeleteEntry: async (path, kind) => {
    if (!activeFilename) {
      return;
    }
    if (
      activeFilename !== path &&
      !(kind === "directory" && activeFilename.startsWith(`${path}/`))
    ) {
      return;
    }
    loadEditor(null, "");
  },
});

let lastMesh: ExportMesh | null = null;

const renderer = new MeshRenderer(canvas, {
  axisElements,
  onFatalError: (message) => {
    statusEl.textContent = message;
    overlay.set(message, { idle: true });
  },
});

const workerClient = createWorkerClient({
  readFile: (path) => fileSystem.readFile(path),
  onReady: ({ silent }) => {
    overlay.set("", { idle: true });
    if (!silent) {
      statusEl.textContent = "WASM ready. Press Command+S to compile.";
    }
  },
  onEcho: (text) => {
    statusEl.textContent = `echo: ${text}`;
    if (typeof window.alert === "function") {
      window.alert(text);
    }
  },
  onLog: (text) => {
    console.log(text);
  },
  onInitError: handleWorkerError,
  onWorkerError: handleWorkerError,
});

function resetFileResult(): void {
  fileRevision++;
  workerClient.cancel();
  clearMeshResult();
  renderer.clearMesh();
  statusEl.textContent = "";
  overlay.set("", { idle: true });
}

function loadEditor(filename: string | null, source: string): void {
  resetFileResult();
  activeFilename = filename;
  editorFilenameEl.textContent = filename || "No file open";
  closeFilesBtn.disabled = !filename;
  window.localStorage.setItem(LAST_OPEN_FILE_STORAGE_KEY, filename || "");
  editor.setSource(source);
}

async function navigate(
  path: string,
  kind: FileSystemEntryKind,
): Promise<void> {
  const hash = pathHash(path, kind);
  if (window.location.hash !== hash) {
    window.history.pushState(null, "", hash);
  }
  await loadLocation();
}

async function loadLocation(): Promise<void> {
  const revision = ++navigationRevision;
  const hash = window.location.hash;
  // The URL changes before hashchange is dispatched, so a revision alone
  // cannot detect a Back/Forward click while an IndexedDB read is finishing.
  const isCurrent = () =>
    revision === navigationRevision && window.location.hash === hash;
  let path = "";
  try {
    path = hashPath(hash);
    const entries = await fileSystem.list();
    if (!isCurrent()) {
      return;
    }
    const entry = entries.find((entry) => entry.path === path);
    if (path && !entry) {
      throw new Error(`Path not found: ${path}`);
    }
    if (entry?.kind === "file") {
      if (!path.toLowerCase().endsWith(".scad")) {
        throw new Error(`Only .scad files can be edited: ${path}`);
      }
      const source = await fileSystem.readTextFile(path);
      if (!isCurrent()) {
        return;
      }
      if (activeFilename !== path || editor.getSource() !== source) {
        loadEditor(path, source);
      }
      window.history.replaceState(null, "", pathHash(path, "file"));
      fileBrowser.hide();
      renderer.requestRender();
      editor.view.focus();
    } else {
      window.history.replaceState(null, "", pathHash(path, "directory"));
      await fileBrowser.show(path);
    }
  } catch (error) {
    const entries = await fileSystem.list();
    if (!isCurrent()) {
      return;
    }
    // Old history entries can outlive renamed/deleted files. Show the nearest
    // surviving folder with an explanation instead of opening an unrelated file.
    let directory = parentDirectory(path);
    while (
      directory &&
      !entries.some(
        (entry) => entry.path === directory && entry.kind === "directory",
      )
    ) {
      directory = parentDirectory(directory);
    }
    window.history.replaceState(null, "", pathHash(directory, "directory"));
    await fileBrowser.show(
      directory,
      error instanceof Error ? error.message : String(error),
    );
  }
}

async function compile(): Promise<void> {
  if (appEl.hidden || workerClient.isBusy()) {
    return;
  }
  const filename = activeFilename;
  if (!filename) {
    statusEl.textContent = "Open a .scad file before compiling.";
    overlay.set("", { idle: true });
    return;
  }
  const revision = fileRevision;
  const source = editor.getSource();
  try {
    if (!workerClient.isReady()) {
      statusEl.textContent = "Loading WASM...";
      overlay.set(statusEl.textContent, { cancelable: false });
      await workerClient.ensureReady();
    }
    if (revision !== fileRevision) {
      return;
    }
    const gridSize = Number(gridEl.value || "128");
    statusEl.textContent = "Compiling...";
    overlay.set("Compiling...", { cancelable: true });
    const result = await workerClient.compile(
      source,
      gridSize,
      isMeshBackend(meshBackendEl.value) ? meshBackendEl.value : "cpu",
      filename,
    );
    if (!result || revision !== fileRevision) {
      return;
    }
    renderer.setMesh(result.positions, result.normals, result.bounds);
    lastMesh = {
      positions: result.positions,
      normals: result.normals,
    };
    downloadBtn.disabled = result.positions.length === 0;
    statusEl.textContent = `Triangles: ${result.positions.length / 9}`;
    overlay.set("", { idle: true });
  } catch (err: unknown) {
    if (revision !== fileRevision) {
      return;
    }
    const message = err instanceof Error ? err.message : undefined;
    if (message === COMPILATION_CANCELED_ERROR) {
      return;
    }
    if (message === GO_EXITED_ERROR) {
      statusEl.textContent = "WASM runtime exited. Reinitializing...";
      overlay.set(statusEl.textContent, { idle: true });
      return;
    }
    const errText = message || "WASM initialization failed.";
    handleWorkerError(errText);
  }
}

formatBtn.addEventListener("click", async () => {
  const filename = activeFilename;
  if (!filename) {
    statusEl.textContent = "Open a .scad file before formatting.";
    return;
  }
  if (workerClient.isBusy()) {
    statusEl.textContent =
      "Wait for the current operation to finish before formatting.";
    return;
  }
  const document = editor.view.state.doc;
  const revision = fileRevision;
  formatBtn.disabled = true;
  statusEl.textContent = "Formatting...";
  try {
    const formatted = await workerClient.format(document.toString(), filename);
    if (revision !== fileRevision) {
      return;
    }
    if (activeFilename !== filename || editor.view.state.doc !== document) {
      statusEl.textContent =
        "Code changed while formatting. Press Format again.";
      return;
    }
    editor.applyFormattedSource(formatted);
    statusEl.textContent = "Code formatted.";
    editor.view.focus();
  } catch (err: unknown) {
    if (revision === fileRevision) {
      statusEl.textContent = err instanceof Error ? err.message : String(err);
    }
  } finally {
    formatBtn.disabled = false;
  }
});

compileBtn.addEventListener("click", () => {
  void compile();
});

document.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
    event.preventDefault();
    void compile();
  }
});

cancelBtn.addEventListener("click", () => {
  if (!workerClient.cancel()) {
    return;
  }
  statusEl.textContent = COMPILATION_CANCELED_ERROR;
  overlay.set(statusEl.textContent, { idle: true });
});

downloadBtn.addEventListener("click", () => {
  if (!lastMesh || lastMesh.positions.length === 0) {
    return;
  }
  const stl = buildBinarySTL(lastMesh.positions, lastMesh.normals);
  const blob = new Blob([stl], { type: "application/sla" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = "model.stl";
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
});

resetViewBtn.addEventListener("click", () => {
  renderer.resetView();
});

setupResizer({
  resizer,
  appEl,
  getRenderer: () => renderer,
});

setupMobileToggle({
  appEl,
  toggleCodeBtn,
  togglePreviewBtn,
});

window.addEventListener("hashchange", () => {
  void loadLocation();
});

// Only an unaddressed visit uses the legacy last-open preference. Once a tab
// has a hash, its own location takes precedence over other tabs' selections.
if (!window.location.hash) {
  const entries = await fileSystem.list();
  const preferred =
    window.localStorage.getItem(LAST_OPEN_FILE_STORAGE_KEY) ?? "main.scad";
  const filename = entries.find(
    (entry) =>
      entry.path === preferred &&
      entry.kind === "file" &&
      entry.path.toLowerCase().endsWith(".scad"),
  );
  window.history.replaceState(
    null,
    "",
    pathHash(filename?.path || "", filename ? "file" : "directory"),
  );
}
await loadLocation();
void workerClient.init();
