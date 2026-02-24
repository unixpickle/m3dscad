/* global Go */

let ready = false;
let wasmReadyPromise = null;

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
        go.run(result.instance);
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
        type: "result",
        id: msg.id,
        ok: false,
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
        postMessage({
          type: "result",
          id: msg.id,
          ok: false,
          error: "WASM not ready.",
        });
        return;
      }
      const result = self.m3dscadCompile(msg.code, msg.gridSize);
      result.type = "result";
      result.id = msg.id;
      postMessage(result);
    })
    .catch((err) => {
      postMessage({
        type: "result",
        id: msg.id,
        ok: false,
        error: err.message || String(err),
      });
    });
};
