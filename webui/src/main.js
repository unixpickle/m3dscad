import "../style.css";
import { createEditor, loadInitialSource } from "./editor.js";
import { buildBinarySTL } from "./export/stl.js";
import { MeshRenderer } from "./renderer/mesh-renderer.js";
import { setupMobileToggle, setupResizer } from "./ui/layout.js";
import { createOverlayController } from "./ui/overlay.js";
import { isWebGPUSupported } from "./webgpu/meshing";
import {
  COMPILATION_CANCELED_ERROR,
  GO_EXITED_ERROR,
  createWorkerClient,
} from "./worker-client";

const WEBGPU_STORAGE_KEY = "m3dscad_use_webgpu";

function requireElement(id) {
  const element = document.getElementById(id);
  if (!element) {
    throw new Error(`Missing required element #${id}`);
  }
  return element;
}

const codeHostEl = requireElement("code");
const statusEl = requireElement("status");
const overlayCard = requireElement("overlayCard");
const overlayText = requireElement("overlayText");
const spinnerEl = requireElement("spinner");
const cancelBtn = requireElement("cancel");
const compileBtn = requireElement("compile");
const resetViewBtn = requireElement("resetView");
const downloadBtn = requireElement("download");
const gridEl = requireElement("grid");
const useWebGPUEl = requireElement("useWebGPU");
const canvas = requireElement("preview");
const resizer = requireElement("resizer");
const appEl = requireElement("app");
const toggleCodeBtn = requireElement("toggleCode");
const togglePreviewBtn = requireElement("togglePreview");

const axisElements = {
  axisLineX: requireElement("axisLineX"),
  axisLineY: requireElement("axisLineY"),
  axisLineZ: requireElement("axisLineZ"),
  axisLabelX: requireElement("axisLabelX"),
  axisLabelY: requireElement("axisLabelY"),
  axisLabelZ: requireElement("axisLabelZ"),
};

const overlay = createOverlayController({
  overlayCard,
  overlayText,
  spinnerEl,
  cancelBtn,
});

const webgpuSupported = isWebGPUSupported();
const storedWebGPUSetting = window.localStorage.getItem(WEBGPU_STORAGE_KEY);
useWebGPUEl.checked = webgpuSupported && storedWebGPUSetting === "true";
useWebGPUEl.disabled = !webgpuSupported;
if (!webgpuSupported) {
  useWebGPUEl.title = "WebGPU is unavailable in this browser.";
  useWebGPUEl.closest("label")?.setAttribute("aria-disabled", "true");
}
useWebGPUEl.addEventListener("change", () => {
  window.localStorage.setItem(WEBGPU_STORAGE_KEY, String(useWebGPUEl.checked));
});

function clearMeshResult() {
  lastMesh = null;
  downloadBtn.disabled = true;
}

function handleWorkerError(errText) {
  statusEl.textContent = errText;
  overlay.set(errText, { idle: true });
  clearMeshResult();
}

const editor = createEditor({
  parent: codeHostEl,
  initialSource: loadInitialSource(),
  onSave: () => {
    void compile();
  },
});

let lastMesh = null;

const renderer = new MeshRenderer(canvas, {
  axisElements,
  onFatalError: (message) => {
    statusEl.textContent = message;
    overlay.set(message, { idle: true });
  },
});

const workerClient = createWorkerClient({
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

async function compile() {
  if (workerClient.isBusy()) {
    return;
  }
  try {
    if (!workerClient.isReady()) {
      statusEl.textContent = "Loading WASM...";
      overlay.set(statusEl.textContent, { cancelable: false });
      await workerClient.ensureReady();
    }
    const gridSize = Number(gridEl.value || "128");
    statusEl.textContent = "Compiling...";
    overlay.set("Compiling...", { cancelable: true });
    const result = await workerClient.compile(
      editor.getSource(),
      gridSize,
      useWebGPUEl.checked,
    );
    if (!result) {
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
  } catch (err) {
    if (err?.message === COMPILATION_CANCELED_ERROR) {
      return;
    }
    if (err?.message === GO_EXITED_ERROR) {
      statusEl.textContent = "WASM runtime exited. Reinitializing...";
      overlay.set(statusEl.textContent, { idle: true });
      return;
    }
    const errText = err?.message || "WASM initialization failed.";
    handleWorkerError(errText);
  }
}

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

void workerClient.init();
