import type { AxisElements, Bounds, CameraState, Vec3 } from "../types";
import { updateAxisIndicator } from "./axis_indicator";
import { setupInteraction } from "./interaction";
import { buildMatrices, normalize3, resizeCanvas } from "./math";
import { createProgram, fragmentShader, vertexShader } from "./webgl";

export type DragMode = "pan" | "rotate" | null;

interface MeshRendererOptions {
  axisElements?: AxisElements;
  onFatalError?: (message: string) => void;
}

interface MeshBuffers {
  position: WebGLBuffer;
  normal: WebGLBuffer;
}

interface MeshAttribs {
  position: number;
  normal: number;
}

interface MeshUniforms {
  model: WebGLUniformLocation;
  view: WebGLUniformLocation;
  proj: WebGLUniformLocation;
  lightDir: WebGLUniformLocation;
  color: WebGLUniformLocation;
}

interface RenderMeshData {
  positions: Float32Array;
  normals: Float32Array;
}

function requiredBuffer(gl: WebGLRenderingContext, name: string): WebGLBuffer {
  const buffer = gl.createBuffer();
  if (!buffer) {
    throw new Error(`Failed to create ${name} buffer.`);
  }
  return buffer;
}

function requiredUniform(
  gl: WebGLRenderingContext,
  program: WebGLProgram,
  name: string,
): WebGLUniformLocation {
  const location = gl.getUniformLocation(program, name);
  if (!location) {
    throw new Error(`Missing uniform ${name}.`);
  }
  return location;
}

export class MeshRenderer {
  canvas: HTMLCanvasElement;
  axisElements: AxisElements | undefined;
  gl: WebGLRenderingContext | null;
  program: WebGLProgram | null;
  buffers: MeshBuffers | null;
  attribs: MeshAttribs | null;
  uniforms: MeshUniforms | null;
  vertexCount: number;
  meshData: RenderMeshData | null;
  bounds: Bounds | null;
  contextLost: boolean;
  camera: CameraState;
  dragging: boolean;
  dragMode: DragMode;
  lastPos: [number, number];
  fitPending: boolean;
  frameHandle: number | null;
  resizeObserver: ResizeObserver | null;

  constructor(canvas: HTMLCanvasElement, options: MeshRendererOptions = {}) {
    this.canvas = canvas;
    this.axisElements = options.axisElements;
    this.gl = canvas.getContext("webgl");
    this.program = null;
    this.buffers = null;
    this.attribs = null;
    this.uniforms = null;
    this.vertexCount = 0;
    this.meshData = null;
    this.bounds = null;
    this.contextLost = false;
    this.camera = {
      theta: 0.6,
      phi: 0.9,
      radius: 6,
      target: [0, 0, 0],
    };
    this.dragging = false;
    this.dragMode = null;
    this.lastPos = [0, 0];
    this.fitPending = true;
    this.frameHandle = null;
    this.resizeObserver = null;

    if (!this.gl) {
      this.contextLost = true;
      options.onFatalError?.("WebGL not available.");
      return;
    }

    this.initializeContextResources();
    setupInteraction(this, canvas);
    this.setupContextRecovery();
    this.requestRender();
    if (typeof ResizeObserver === "function") {
      this.resizeObserver = new ResizeObserver(() => this.requestRender());
      this.resizeObserver.observe(this.canvas);
    }
    window.addEventListener("resize", () => this.requestRender(), {
      passive: true,
    });
  }

  setMesh(
    positions: Float32Array,
    normals: Float32Array,
    bounds: Bounds | null,
  ): void {
    const hadMesh = this.vertexCount > 0;
    this.meshData = { positions, normals };
    this.uploadMesh();
    this.bounds = bounds;
    this.fitPending = !hadMesh;
    this.requestRender();
  }

  fitView(): void {
    if (!this.bounds) {
      return;
    }
    const min = this.bounds.min;
    const max = this.bounds.max;
    const center: Vec3 = [
      (min[0] + max[0]) / 2,
      (min[1] + max[1]) / 2,
      (min[2] + max[2]) / 2,
    ];
    const dx = max[0] - min[0];
    const dy = max[1] - min[1];
    const dz = max[2] - min[2];
    const radius = Math.max(Math.hypot(dx, dy, dz) * 0.75, 1.5);
    this.camera.target = center;
    this.camera.radius = radius * 2.2;
    this.fitPending = false;
  }

  resetView(): void {
    this.camera.theta = 0.6;
    this.camera.phi = 0.9;
    this.fitPending = true;
    this.requestRender();
  }

  requestRender(): void {
    if (this.contextLost || this.frameHandle != null) {
      return;
    }
    this.frameHandle = requestAnimationFrame(() => {
      this.frameHandle = null;
      this.render();
    });
  }

  render(): void {
    const gl = this.gl;
    if (
      !gl ||
      this.contextLost ||
      !this.program ||
      !this.buffers ||
      (typeof gl.isContextLost === "function" && gl.isContextLost())
    ) {
      return;
    }
    const { attribs, buffers, uniforms } = this;
    if (!attribs || !buffers || !uniforms) {
      return;
    }
    resizeCanvas(this.canvas);
    if (this.fitPending) {
      this.fitView();
    }
    gl.viewport(0, 0, this.canvas.width, this.canvas.height);
    gl.clearColor(0.05, 0.06, 0.09, 1);
    gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
    gl.enable(gl.DEPTH_TEST);
    gl.useProgram(this.program);
    updateAxisIndicator(this.camera, this.axisElements);

    if (this.vertexCount === 0) {
      return;
    }

    const { model, view, proj, eye } = buildMatrices(
      this.camera,
      this.canvas,
      this.bounds,
    );
    gl.uniformMatrix4fv(uniforms.model, false, model);
    gl.uniformMatrix4fv(uniforms.view, false, view);
    gl.uniformMatrix4fv(uniforms.proj, false, proj);
    const lightDir = normalize3([
      eye[0] - this.camera.target[0],
      eye[1] - this.camera.target[1],
      eye[2] - this.camera.target[2],
    ]);
    gl.uniform3f(uniforms.lightDir, lightDir[0], lightDir[1], lightDir[2]);
    gl.uniform3f(uniforms.color, 0.67, 0.75, 0.95);

    gl.bindBuffer(gl.ARRAY_BUFFER, buffers.position);
    gl.enableVertexAttribArray(attribs.position);
    gl.vertexAttribPointer(attribs.position, 3, gl.FLOAT, false, 0, 0);
    gl.bindBuffer(gl.ARRAY_BUFFER, buffers.normal);
    gl.enableVertexAttribArray(attribs.normal);
    gl.vertexAttribPointer(attribs.normal, 3, gl.FLOAT, false, 0, 0);
    gl.drawArrays(gl.TRIANGLES, 0, this.vertexCount);
  }

  initializeContextResources(): void {
    const gl = this.gl;
    if (!gl) {
      return;
    }
    this.program = createProgram(gl, vertexShader, fragmentShader);
    this.buffers = {
      position: requiredBuffer(gl, "position"),
      normal: requiredBuffer(gl, "normal"),
    };
    this.attribs = {
      position: gl.getAttribLocation(this.program, "a_position"),
      normal: gl.getAttribLocation(this.program, "a_normal"),
    };
    this.uniforms = {
      model: requiredUniform(gl, this.program, "u_model"),
      view: requiredUniform(gl, this.program, "u_view"),
      proj: requiredUniform(gl, this.program, "u_proj"),
      lightDir: requiredUniform(gl, this.program, "u_lightDir"),
      color: requiredUniform(gl, this.program, "u_color"),
    };
  }

  uploadMesh(): void {
    if (this.contextLost || !this.meshData || !this.buffers || !this.gl) {
      this.vertexCount = this.meshData ? this.meshData.positions.length / 3 : 0;
      return;
    }
    const { positions, normals } = this.meshData;
    this.vertexCount = positions.length / 3;
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.buffers.position);
    this.gl.bufferData(this.gl.ARRAY_BUFFER, positions, this.gl.STATIC_DRAW);
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.buffers.normal);
    this.gl.bufferData(this.gl.ARRAY_BUFFER, normals, this.gl.STATIC_DRAW);
  }

  setupContextRecovery(): void {
    this.canvas.addEventListener("webglcontextlost", (event) => {
      event.preventDefault();
      this.contextLost = true;
      this.program = null;
      this.buffers = null;
      this.attribs = null;
      this.uniforms = null;
      if (this.frameHandle != null) {
        cancelAnimationFrame(this.frameHandle);
        this.frameHandle = null;
      }
    });
    this.canvas.addEventListener("webglcontextrestored", () => {
      this.contextLost = false;
      this.initializeContextResources();
      this.uploadMesh();
      this.requestRender();
    });
  }
}
