import type {
  CompileSuccess,
  InitRequest,
  MeshData,
  WorkerInboundMessage,
  WorkerRequest,
} from "./types";

export const GO_EXITED_ERROR = "Go program has already exited";
export const WORKER_RESTARTED_ERROR = "WASM worker restarted";
export const COMPILATION_CANCELED_ERROR = "Compilation canceled.";

interface InitOptions {
  silent?: boolean;
}

interface ReadyInfo {
  silent: boolean;
}

interface WorkerClientOptions {
  onReady?: (info: ReadyInfo) => void;
  onEcho?: (text: string) => void;
  onLog?: (text: string) => void;
  onInitError?: (message: string) => void;
  onWorkerError?: (message: string) => void;
}

interface WorkerClient {
  init(options?: InitOptions): Promise<void>;
  ensureReady(options?: InitOptions): Promise<void>;
  compile(
    code: string,
    gridSize: number,
    useWebGPU?: boolean,
  ): Promise<MeshData | null>;
  cancel(): boolean;
  isReady(): boolean;
  isBusy(): boolean;
}

type PromiseResolver<T> = (value: T | PromiseLike<T>) => void;
type PromiseRejector = (reason?: unknown) => void;

function toMeshData(msg: CompileSuccess): MeshData {
  return {
    positions: new Float32Array(msg.positions),
    normals: new Float32Array(msg.normals),
    bounds: msg.bounds,
  };
}

function initMessage(): InitRequest {
  return { type: "init" };
}

function compileMessage(
  id: number,
  code: string,
  gridSize: number,
  useWebGPU: boolean,
): WorkerRequest {
  return {
    type: "compile",
    id,
    code,
    gridSize,
    useWebGPU,
  };
}

export function createWorkerClient({
  onReady,
  onEcho,
  onLog,
  onInitError,
  onWorkerError,
}: WorkerClientOptions = {}): WorkerClient {
  let worker: Worker | null = null;
  let workerReady = false;
  let workerInitPromise: Promise<void> = Promise.resolve();
  let resolveWorkerInit: PromiseResolver<void> | null = null;
  let rejectWorkerInit: PromiseRejector | null = null;
  let requestId = 0;
  let pendingRequestId: number | null = null;
  let compilePreparing = false;
  let resolveCompile: PromiseResolver<MeshData> | null = null;
  let rejectCompile: PromiseRejector | null = null;

  function clearWorkerInitHandlers(): void {
    resolveWorkerInit = null;
    rejectWorkerInit = null;
  }

  function clearPendingCompile(): void {
    pendingRequestId = null;
    resolveCompile = null;
    rejectCompile = null;
  }

  function init(options: InitOptions = {}): Promise<void> {
    const silent = Boolean(options.silent);
    workerReady = false;
    if (rejectWorkerInit) {
      rejectWorkerInit(new Error(WORKER_RESTARTED_ERROR));
      clearWorkerInitHandlers();
    }
    if (pendingRequestId != null) {
      const rejectCurrent = rejectCompile;
      clearPendingCompile();
      rejectCurrent?.(new Error(WORKER_RESTARTED_ERROR));
    }
    worker?.terminate();

    worker = new Worker(new URL("../worker.ts", import.meta.url));
    const currentWorker = worker;
    workerInitPromise = new Promise<void>((resolve, reject) => {
      resolveWorkerInit = resolve;
      rejectWorkerInit = reject;
    });
    worker.onmessage = (event: MessageEvent<WorkerInboundMessage>) => {
      if (worker !== currentWorker) {
        return;
      }
      const msg = event.data;
      if (!msg) {
        return;
      }
      if (msg.type === "ready") {
        workerReady = true;
        resolveWorkerInit?.();
        clearWorkerInitHandlers();
        onReady?.({ silent });
        return;
      }
      if (msg.type === "echo") {
        onEcho?.(msg.message || "");
        return;
      }
      if (msg.type === "log") {
        onLog?.(msg.message || "");
        return;
      }
      if (msg.type === "init_error") {
        workerReady = false;
        const errText = msg.error || "WASM initialization failed.";
        rejectWorkerInit?.(new Error(errText));
        clearWorkerInitHandlers();
        if (pendingRequestId != null) {
          const rejectCurrent = rejectCompile;
          clearPendingCompile();
          rejectCurrent?.(new Error(errText));
        }
        onInitError?.(errText);
        return;
      }
      if (pendingRequestId == null || msg.id !== pendingRequestId) {
        return;
      }
      const resolveCurrent = resolveCompile;
      const rejectCurrent = rejectCompile;
      clearPendingCompile();
      if (!msg.ok) {
        rejectCurrent?.(new Error(msg.error || "Unknown error."));
        return;
      }
      resolveCurrent?.(toMeshData(msg));
    };
    worker.onerror = (event: ErrorEvent) => {
      if (worker !== currentWorker) {
        return;
      }
      workerReady = false;
      const errText = `Worker error: ${event.message}`;
      rejectWorkerInit?.(new Error(errText));
      clearWorkerInitHandlers();
      if (pendingRequestId != null) {
        const rejectCurrent = rejectCompile;
        clearPendingCompile();
        rejectCurrent?.(new Error(errText));
      }
      onWorkerError?.(errText);
    };
    worker.postMessage(initMessage());
    return workerInitPromise;
  }

  async function compile(
    code: string,
    gridSize: number,
    useWebGPU = false,
  ): Promise<MeshData | null> {
    if (pendingRequestId != null || compilePreparing) {
      return null;
    }
    compilePreparing = true;
    try {
      if (!worker) {
        void init({ silent: true });
      }
      if (!workerReady) {
        await workerInitPromise;
      }
      if (!workerReady || !worker) {
        throw new Error("WASM initialization failed.");
      }
      pendingRequestId = ++requestId;
      const resultPromise = new Promise<MeshData>((resolve, reject) => {
        resolveCompile = resolve;
        rejectCompile = reject;
      });
      worker.postMessage(
        compileMessage(pendingRequestId, code, gridSize, useWebGPU),
      );
      return await resultPromise;
    } catch (err) {
      if (err instanceof Error && err.message === GO_EXITED_ERROR) {
        void init({ silent: true });
      }
      throw err;
    } finally {
      compilePreparing = false;
    }
  }

  function ensureReady(options: InitOptions = {}): Promise<void> {
    if (workerReady) {
      return Promise.resolve();
    }
    if (!worker) {
      return init(options);
    }
    return workerInitPromise;
  }

  function cancel(): boolean {
    if (pendingRequestId == null) {
      return false;
    }
    const rejectCurrent = rejectCompile;
    clearPendingCompile();
    rejectCurrent?.(new Error(COMPILATION_CANCELED_ERROR));
    void init({ silent: true });
    return true;
  }

  return {
    init,
    ensureReady,
    compile,
    cancel,
    isReady() {
      return workerReady;
    },
    isBusy() {
      return compilePreparing || pendingRequestId != null;
    },
  };
}
