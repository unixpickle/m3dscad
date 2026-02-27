#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GOROOT="$(go env GOROOT)"

WASM_EXEC=""
if [[ -f "${GOROOT}/misc/wasm/wasm_exec.js" ]]; then
  WASM_EXEC="${GOROOT}/misc/wasm/wasm_exec.js"
elif [[ -f "${GOROOT}/lib/wasm/wasm_exec.js" ]]; then
  WASM_EXEC="${GOROOT}/lib/wasm/wasm_exec.js"
else
  echo "could not find wasm_exec.js under ${GOROOT}" >&2
  exit 1
fi

cp "${WASM_EXEC}" "${ROOT}/webui/wasm_exec.js"
(cd "${ROOT}" && GOOS=js GOARCH=wasm go build -o "${ROOT}/webui/m3dscad.wasm" ./webui/wasm)

echo "Built webui/m3dscad.wasm"
