import "../style.css";
import { createEditor, loadInitialSource } from "./editor";
import { buildBinarySTL } from "./export/stl";
import { MeshRenderer } from "./renderer/mesh_renderer";
import type { AxisElements } from "./types";
import { setupMobileToggle, setupResizer } from "./ui/layout";
import { createOverlayController } from "./ui/overlay";
import { isWebGPUSupported } from "./webgpu/meshing";
import {
  COMPILATION_CANCELED_ERROR,
  GO_EXITED_ERROR,
  createWorkerClient,
} from "./worker_client";

const WEBGPU_STORAGE_KEY = "m3dscad_use_webgpu";

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
const resetViewBtn = requireElement<HTMLButtonElement>("resetView");
const downloadBtn = requireElement<HTMLButtonElement>("download");
const gridEl = requireElement<HTMLInputElement>("grid");
const useWebGPUEl = requireElement<HTMLInputElement>("useWebGPU");
const canvas = requireElement<HTMLCanvasElement>("preview");
const resizer = requireElement("resizer");
const appEl = requireElement("app");
const toggleCodeBtn = requireElement<HTMLButtonElement>("toggleCode");
const togglePreviewBtn = requireElement<HTMLButtonElement>("togglePreview");

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
  initialSource: loadInitialSource(),
  onSave: () => {
    void compile();
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

async function compile(): Promise<void> {
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
  } catch (err: unknown) {
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
