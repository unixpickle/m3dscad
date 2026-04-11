/* global Go */

let ready = false;
let wasmReadyPromise = null;
const goExitedError = "Go program has already exited";

function resetWasmState() {
  ready = false;
  wasmReadyPromise = null;
  self.m3dscadCompile = undefined;
}

function postCompileError(id, error) {
  postMessage({
    type: "result",
    id,
    ok: false,
    error,
  });
}

function initWasm() {
  if (wasmReadyPromise) {
    return wasmReadyPromise;
  }
  wasmReadyPromise = new Promise((resolve, reject) => {
    try {
      importScripts("./wasm_exec.js");
    } catch (err) {
      reject(new Error("Missing wasm_exec.js. Run ./webui/build_wasm.sh first."));
      return;
    }
    const go = new Go();
    const wasmUrl = "./m3dscad.wasm";
    const load = WebAssembly.instantiateStreaming
      ? WebAssembly.instantiateStreaming(fetch(wasmUrl), go.importObject)
      : fetch(wasmUrl)
          .then((resp) => resp.arrayBuffer())
          .then((bytes) => WebAssembly.instantiate(bytes, go.importObject));
    load
      .then((result) => {
        go.run(result.instance)
          .catch(() => {})
          .finally(() => {
            resetWasmState();
          });
        ready = true;
        postMessage({ type: "ready" });
        resolve();
      })
      .catch((err) => {
        reject(err);
      });
  });
  return wasmReadyPromise;
}

onmessage = (event) => {
  const msg = event.data;
  if (!msg) {
    return;
  }
  if (msg.type === "init") {
    initWasm().catch((err) => {
      postMessage({
        type: "init_error",
        error: err.message || String(err),
      });
    });
    return;
  }
  if (msg.type !== "compile") {
    return;
  }
  initWasm()
    .then(() => {
      if (!ready || typeof self.m3dscadCompile !== "function") {
        postCompileError(msg.id, goExitedError);
        return;
      }
      let result;
      try {
        result = self.m3dscadCompile(msg.code, msg.gridSize);
      } catch (err) {
        postCompileError(msg.id, (err && err.message) || String(err));
        return;
      }
      if (!result || typeof result !== "object") {
        resetWasmState();
        postCompileError(msg.id, goExitedError);
        return;
      }
      result.type = "result";
      result.id = msg.id;
      postMessage(result);
    })
    .catch((err) => {
      postCompileError(msg.id, err.message || String(err));
    });
};
