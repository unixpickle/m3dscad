#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GOROOT="$(go env GOROOT)"
PUBLIC_DIR="${ROOT}/webui/public"

WASM_EXEC=""
if [[ -f "${GOROOT}/misc/wasm/wasm_exec.js" ]]; then
  WASM_EXEC="${GOROOT}/misc/wasm/wasm_exec.js"
elif [[ -f "${GOROOT}/lib/wasm/wasm_exec.js" ]]; then
  WASM_EXEC="${GOROOT}/lib/wasm/wasm_exec.js"
else
  echo "could not find wasm_exec.js under ${GOROOT}" >&2
  exit 1
fi

mkdir -p "${PUBLIC_DIR}"
cp "${WASM_EXEC}" "${PUBLIC_DIR}/wasm_exec.js"
(cd "${ROOT}" && GOOS=js GOARCH=wasm go build -o "${PUBLIC_DIR}/m3dscad.wasm" ./webui/wasm)

echo "Built webui/public/m3dscad.wasm"
