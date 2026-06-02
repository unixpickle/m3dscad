import type {
  WebGPUKernelBinding,
  WebGPUMeshRequestPayload,
  WebGPUMeshResultMessage,
} from "../types";
import { dualContourWebGPU } from "../webgpu_meshes/dual_contouring";
import { marchingCubesWebGPU } from "../webgpu_meshes/marching_cubes";
import { marchingSquaresWebGPU } from "../webgpu_meshes/marching_squares";

interface GPUAdapterLike {
  requestDevice(): Promise<GPUDeviceLike>;
}

interface GPUDeviceLike {
  lost: Promise<unknown>;
}

interface NavigatorGPULike {
  requestAdapter(): Promise<GPUAdapterLike | null>;
}

let devicePromise: Promise<GPUDeviceLike> | null = null;

function getNavigatorGPU(): NavigatorGPULike | undefined {
  if (typeof navigator === "undefined") {
    return undefined;
  }
  const gpu = (navigator as Navigator & { gpu?: unknown }).gpu;
  if (!gpu || typeof gpu !== "object" || !("requestAdapter" in gpu)) {
    return undefined;
  }
  return gpu as NavigatorGPULike;
}

export function isWebGPUSupported(): boolean {
  return typeof getNavigatorGPU() !== "undefined";
}

async function getWebGPUDevice(): Promise<GPUDeviceLike> {
  if (!devicePromise) {
    const gpu = getNavigatorGPU();
    if (!gpu) {
      throw new Error("WebGPU is not available in this browser.");
    }
    devicePromise = gpu.requestAdapter().then(async (adapter) => {
      if (!adapter) {
        throw new Error("No WebGPU adapter is available.");
      }
      const device = await adapter.requestDevice();
      void device.lost.then(() => {
        devicePromise = null;
      });
      return device;
    });
  }
  return devicePromise;
}

function convertBindings(bindings: WebGPUKernelBinding[]) {
  return bindings.map((binding) => ({
    ...binding,
    wgslDefs: binding.wgslDefs ?? "",
  }));
}

export async function handleWebGPUMeshRequest(
  request: WebGPUMeshRequestPayload,
): Promise<WebGPUMeshResultMessage> {
  const device = await getWebGPUDevice();
  const solidBindings = convertBindings(request.kernel.bindings);

  switch (request.method) {
    case "marching_squares": {
      const result = await marchingSquaresWebGPU({
        device,
        solidWGSL: request.kernel.wgsl,
        solidBindings,
        min: [request.min[0], request.min[1]],
        max: [request.max[0], request.max[1]],
        delta: request.delta,
        bisectionSteps: request.subdiv,
      });
      return {
        type: "webgpu_mesh_result",
        id: 0,
        ok: true,
        meshType: "mesh2d",
        positions: result.mesh.positions,
        indices: result.mesh.indices,
      };
    }
    case "marching_cubes": {
      const result = await marchingCubesWebGPU({
        device,
        solidWGSL: request.kernel.wgsl,
        solidBindings,
        min: [request.min[0], request.min[1], request.min[2]],
        max: [request.max[0], request.max[1], request.max[2]],
        delta: request.delta,
        bisectionSteps: request.subdiv,
      });
      const mesh = result.mesh.compact();
      return {
        type: "webgpu_mesh_result",
        id: 0,
        ok: true,
        meshType: "mesh3d",
        positions: mesh.positions,
        indices: mesh.indices,
      };
    }
    case "dual_contour": {
      const result = await dualContourWebGPU({
        device,
        solidWGSL: request.kernel.wgsl,
        solidBindings,
        min: [request.min[0], request.min[1], request.min[2]],
        max: [request.max[0], request.max[1], request.max[2]],
        delta: request.delta,
        repair: request.repair,
        clip: request.clip,
      });
      return {
        type: "webgpu_mesh_result",
        id: 0,
        ok: true,
        meshType: "mesh3d",
        positions: result.repaired.positions,
        indices: result.repaired.indices,
      };
    }
    default:
      throw new Error(`Unsupported WebGPU meshing method: ${request.method}`);
  }
}
