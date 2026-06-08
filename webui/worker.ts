/// <reference lib="webworker" />

import type {
  WebGPUMeshRequestPayload,
  WorkerRequest,
  WorkerResponse,
} from "./src/types";
import { handleWebGPUMeshRequest } from "./src/webgpu/meshing";

const goExitedError = "Go program has already exited";

interface GoRuntime {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void> | void;
}

declare const Go: {
  new (): GoRuntime;
};

interface CompileOptions {
  useWebGPU: boolean;
}

interface WorkerGlobalWithRuntime extends DedicatedWorkerGlobalScope {
  m3dscadCompile?: (
    code: string,
    gridSize: number,
    options: CompileOptions,
  ) => Promise<unknown> | unknown;
  m3dscadWebGPUMesh?: (request: WebGPUMeshRequestPayload) => Promise<unknown>;
}

const workerScope = self as WorkerGlobalWithRuntime;

let ready = false;
let wasmReadyPromise: Promise<void> | null = null;

function resetWasmState(): void {
  ready = false;
  wasmReadyPromise = null;
  workerScope.m3dscadCompile = undefined;
}

function postWorkerMessage(message: WorkerResponse): void {
  workerScope.postMessage(message);
}

function postCompileError(id: number, error: string): void {
  postWorkerMessage({
    type: "result",
    id,
    ok: false,
    error,
  });
}

function installWebGPUBridge(): void {
  workerScope.m3dscadWebGPUMesh = (request) => handleWebGPUMeshRequest(request);
}

function instantiateWasm(
  go: GoRuntime,
  wasmUrl: string,
): Promise<WebAssembly.WebAssemblyInstantiatedSource> {
  return fetch(wasmUrl).then((resp) => {
    if (!resp.ok) {
      throw new Error(
        `Failed to fetch ${wasmUrl}: ${resp.status} ${resp.statusText}`,
      );
    }
    const contentType = resp.headers.get("content-type") || "";
    if (
      WebAssembly.instantiateStreaming &&
      contentType.toLowerCase().includes("application/wasm")
    ) {
      return WebAssembly.instantiateStreaming(
        Promise.resolve(resp),
        go.importObject,
      );
    }
    return resp
      .arrayBuffer()
      .then((bytes) => WebAssembly.instantiate(bytes, go.importObject));
  });
}

function initWasm(): Promise<void> {
  if (wasmReadyPromise) {
    return wasmReadyPromise;
  }
  wasmReadyPromise = new Promise<void>((resolve, reject) => {
    const distRootUrl = new URL("../", workerScope.location.href);
    const wasmExecUrl = new URL("wasm_exec.js", distRootUrl);
    const wasmUrl = new URL("m3dscad.wasm", distRootUrl);
    try {
      workerScope.importScripts(wasmExecUrl.href);
    } catch {
      reject(
        new Error("Missing wasm_exec.js. Run ./webui/build_wasm.sh first."),
      );
      return;
    }
    const go = new Go();
    installWebGPUBridge();
    instantiateWasm(go, wasmUrl.href)
      .then((result) => {
        Promise.resolve(go.run(result.instance))
          .catch(() => {})
          .finally(() => {
            resetWasmState();
          });
        ready = true;
        postWorkerMessage({ type: "ready" });
        resolve();
      })
      .catch((err: unknown) => {
        reject(err instanceof Error ? err : new Error(String(err)));
      });
  });
  return wasmReadyPromise;
}

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function compileResultMessage(
  id: number,
  result: unknown,
): WorkerResponse | null {
  if (!isObject(result)) {
    return null;
  }
  return {
    ...result,
    type: "result",
    id,
  } as WorkerResponse;
}

workerScope.addEventListener(
  "message",
  (event: MessageEvent<WorkerRequest>) => {
    const msg = event.data;
    if (!msg) {
      return;
    }
    if (msg.type === "init") {
      void initWasm().catch((err: unknown) => {
        postWorkerMessage({
          type: "init_error",
          error: err instanceof Error ? err.message : String(err),
        });
      });
      return;
    }
    if (msg.type !== "compile") {
      return;
    }
    void initWasm()
      .then(() => {
        if (!ready || typeof workerScope.m3dscadCompile !== "function") {
          postCompileError(msg.id, goExitedError);
          return;
        }
        return Promise.resolve(
          workerScope.m3dscadCompile(msg.code, msg.gridSize, {
            useWebGPU: Boolean(msg.useWebGPU),
          }),
        )
          .then((result) => {
            const response = compileResultMessage(msg.id, result);
            if (!response) {
              resetWasmState();
              postCompileError(msg.id, goExitedError);
              return;
            }
            postWorkerMessage(response);
          })
          .catch((err: unknown) => {
            postCompileError(
              msg.id,
              err instanceof Error ? err.message : String(err),
            );
          });
      })
      .catch((err: unknown) => {
        postCompileError(
          msg.id,
          err instanceof Error ? err.message : String(err),
        );
      });
  },
);
