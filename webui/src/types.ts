export interface Bounds {
  min: [number, number, number] | number[];
  max: [number, number, number] | number[];
}

export type Vec3 = [number, number, number];
export type Mat4 = Float32Array<ArrayBuffer>;

export interface MeshData {
  positions: Float32Array;
  normals: Float32Array;
  bounds: Bounds | null;
}

export interface CameraState {
  theta: number;
  phi: number;
  radius: number;
  target: Vec3;
}

export interface AxisElements {
  axisLineX: SVGLineElement;
  axisLineY: SVGLineElement;
  axisLineZ: SVGLineElement;
  axisLabelX: SVGTextElement;
  axisLabelY: SVGTextElement;
  axisLabelZ: SVGTextElement;
}

export interface OverlayOptions {
  idle?: boolean;
  cancelable?: boolean;
}

export interface CompileRequest {
  type: "compile";
  id: number;
  code: string;
  gridSize: number;
  useWebGPU: boolean;
}

export interface InitRequest {
  type: "init";
}

export type WorkerRequest = CompileRequest | InitRequest;

export interface CompileSuccess {
  type: "result";
  id: number;
  ok: true;
  positions: ArrayLike<number>;
  normals: ArrayLike<number>;
  bounds: Bounds | null;
}

export interface CompileError {
  type: "result";
  id: number;
  ok: false;
  error: string;
}

export interface ReadyMessage {
  type: "ready";
}

export interface InitErrorMessage {
  type: "init_error";
  error: string;
}

export interface EchoMessage {
  type: "echo";
  message: string;
}

export interface LogMessage {
  type: "log";
  message: string;
}

export interface WebGPUKernelBinding {
  name: string;
  kind: "uniform" | "storage";
  wgslType: string;
  wgslDefs?: string;
  source: BufferSource;
}

export interface WebGPUKernelSpec {
  dimension: 2 | 3;
  wgsl: string;
  bindings: WebGPUKernelBinding[];
}

export type WebGPUMeshMethod =
  | "marching_squares"
  | "marching_cubes"
  | "dual_contour";

export interface WebGPUMeshRequestPayload {
  method: WebGPUMeshMethod;
  kernel: WebGPUKernelSpec;
  min: number[];
  max: number[];
  delta: number;
  subdiv?: number;
  repair?: boolean;
  clip?: boolean;
}

export interface WebGPUMeshSuccessMessage {
  type: "webgpu_mesh_result";
  id: number;
  ok: true;
  meshType: "mesh2d" | "mesh3d";
  positions: Float32Array;
  indices: Uint32Array;
}

export interface WebGPUMeshErrorMessage {
  type: "webgpu_mesh_result";
  id: number;
  ok: false;
  error: string;
}

export type WebGPUMeshResultMessage =
  | WebGPUMeshSuccessMessage
  | WebGPUMeshErrorMessage;

export type WorkerResponse =
  | CompileSuccess
  | CompileError
  | ReadyMessage
  | InitErrorMessage
  | EchoMessage
  | LogMessage;

export type WorkerInboundMessage = WorkerResponse;

export type WorkerOutboundMessage = WorkerRequest;
