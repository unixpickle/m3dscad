import { updateAxisIndicator } from "./axis-indicator.js";
import { setupInteraction } from "./interaction.js";
import { buildMatrices, normalize3, resizeCanvas } from "./math.js";
import { createProgram, fragmentShader, vertexShader } from "./webgl.js";

export class MeshRenderer {
  constructor(canvas, options = {}) {
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

  setMesh(positions, normals, bounds) {
    const hadMesh = this.vertexCount > 0;
    this.meshData = { positions, normals };
    this.uploadMesh();
    this.bounds = bounds;
    this.fitPending = !hadMesh;
    this.requestRender();
  }

  fitView() {
    if (!this.bounds) {
      return;
    }
    const min = this.bounds.min;
    const max = this.bounds.max;
    const center = [
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

  resetView() {
    this.camera.theta = 0.6;
    this.camera.phi = 0.9;
    this.fitPending = true;
    this.requestRender();
  }

  requestRender() {
    if (this.contextLost || this.frameHandle != null) {
      return;
    }
    this.frameHandle = requestAnimationFrame(() => {
      this.frameHandle = null;
      this.render();
    });
  }

  render() {
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
    gl.uniformMatrix4fv(this.uniforms.model, false, model);
    gl.uniformMatrix4fv(this.uniforms.view, false, view);
    gl.uniformMatrix4fv(this.uniforms.proj, false, proj);
    const lightDir = normalize3([
      eye[0] - this.camera.target[0],
      eye[1] - this.camera.target[1],
      eye[2] - this.camera.target[2],
    ]);
    gl.uniform3f(this.uniforms.lightDir, lightDir[0], lightDir[1], lightDir[2]);
    gl.uniform3f(this.uniforms.color, 0.67, 0.75, 0.95);

    gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.position);
    gl.enableVertexAttribArray(this.attribs.position);
    gl.vertexAttribPointer(this.attribs.position, 3, gl.FLOAT, false, 0, 0);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.normal);
    gl.enableVertexAttribArray(this.attribs.normal);
    gl.vertexAttribPointer(this.attribs.normal, 3, gl.FLOAT, false, 0, 0);
    gl.drawArrays(gl.TRIANGLES, 0, this.vertexCount);
  }

  initializeContextResources() {
    const gl = this.gl;
    if (!gl) {
      return;
    }
    this.program = createProgram(gl, vertexShader, fragmentShader);
    this.buffers = {
      position: gl.createBuffer(),
      normal: gl.createBuffer(),
    };
    this.attribs = {
      position: gl.getAttribLocation(this.program, "a_position"),
      normal: gl.getAttribLocation(this.program, "a_normal"),
    };
    this.uniforms = {
      model: gl.getUniformLocation(this.program, "u_model"),
      view: gl.getUniformLocation(this.program, "u_view"),
      proj: gl.getUniformLocation(this.program, "u_proj"),
      lightDir: gl.getUniformLocation(this.program, "u_lightDir"),
      color: gl.getUniformLocation(this.program, "u_color"),
    };
  }

  uploadMesh() {
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

  setupContextRecovery() {
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
