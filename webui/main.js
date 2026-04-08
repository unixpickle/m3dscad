import { basicSetup } from "codemirror";
import { EditorState } from "@codemirror/state";
import { EditorView, keymap } from "@codemirror/view";
import { indentWithTab } from "@codemirror/commands";

const codeHostEl = document.getElementById("code");
const statusEl = document.getElementById("status");
const overlayEl = document.getElementById("overlay");
const overlayCard = document.getElementById("overlayCard");
const overlayText = document.getElementById("overlayText");
const spinnerEl = document.getElementById("spinner");
const cancelBtn = document.getElementById("cancel");
const compileBtn = document.getElementById("compile");
const resetViewBtn = document.getElementById("resetView");
const downloadBtn = document.getElementById("download");
const gridEl = document.getElementById("grid");
const canvas = document.getElementById("preview");
const axisLineX = document.getElementById("axisLineX");
const axisLineY = document.getElementById("axisLineY");
const axisLineZ = document.getElementById("axisLineZ");
const axisLabelX = document.getElementById("axisLabelX");
const axisLabelY = document.getElementById("axisLabelY");
const axisLabelZ = document.getElementById("axisLabelZ");
const resizer = document.getElementById("resizer");
const appEl = document.getElementById("app");
const toggleCodeBtn = document.getElementById("toggleCode");
const togglePreviewBtn = document.getElementById("togglePreview");

const defaultSource = `// Example
difference() {
  cube(2, center=true);
  translate([1, 1, 1]) sphere(1);
}
`;

const storedSource = window.localStorage.getItem("m3dscad_source");
const initialSource = storedSource && storedSource.trim().length > 0 ? storedSource : defaultSource;

let editorView = null;
createEditor(initialSource);

const vertexShader = `
  attribute vec3 a_position;
  attribute vec3 a_normal;
  uniform mat4 u_model;
  uniform mat4 u_view;
  uniform mat4 u_proj;
  varying vec3 v_normal;
  varying vec3 v_pos;
  void main() {
    vec4 world = u_model * vec4(a_position, 1.0);
    v_pos = world.xyz;
    v_normal = mat3(u_model) * a_normal;
    gl_Position = u_proj * u_view * world;
  }
`;

const fragmentShader = `
  precision mediump float;
  uniform vec3 u_lightDir;
  uniform vec3 u_color;
  varying vec3 v_normal;
  varying vec3 v_pos;
  void main() {
    vec3 normal = normalize(v_normal);
    float diff = max(dot(normal, normalize(u_lightDir)), 0.0);
    float rim = pow(1.0 - max(dot(normal, vec3(0.0, 0.0, 1.0)), 0.0), 2.0);
    vec3 color = u_color * (0.2 + diff * 0.8) + vec3(0.15, 0.2, 0.3) * rim;
    gl_FragColor = vec4(color, 1.0);
  }
`;

const CAMERA_FOV_RAD = (60 * Math.PI) / 180;

let renderer = null;
let worker = null;
let workerReady = false;
let requestId = 0;
let pendingRequest = null;
let lastMesh = null;
const goExitedError = "Go program has already exited";

function createEditor(source) {
  editorView = new EditorView({
    state: EditorState.create({
      doc: source,
      extensions: [
        basicSetup,
        keymap.of([
          {
            key: "Mod-s",
            run: () => {
              compile();
              return true;
            },
          },
          indentWithTab,
        ]),
        EditorView.lineWrapping,
        EditorView.updateListener.of((update) => {
          if (!update.docChanged) return;
          window.localStorage.setItem("m3dscad_source", getSource());
        }),
      ],
    }),
    parent: codeHostEl,
  });
}

function getSource() {
  return editorView ? editorView.state.doc.toString() : "";
}

function initWorker(options = {}) {
  const silent = Boolean(options.silent);
  if (!silent) {
    setOverlay("Loading WASM...");
  }
  workerReady = false;
  if (worker) {
    worker.terminate();
  }
  worker = new Worker("./worker.js");
  worker.onmessage = (event) => {
    const msg = event.data;
    if (!msg || !msg.type) return;
    if (msg.type === "ready") {
      workerReady = true;
      setOverlay("", true);
      if (!silent) {
        statusEl.textContent = "WASM ready. Press Command+S to compile.";
      }
      return;
    }
    if (msg.type === "echo") {
      const text = msg.message || "";
      statusEl.textContent = `echo: ${text}`;
      if (typeof window.alert === "function") {
        window.alert(text);
      }
      return;
    }
    if (msg.type === "init_error") {
      workerReady = false;
      const errText = msg.error || "WASM initialization failed.";
      statusEl.textContent = errText;
      setOverlay(statusEl.textContent, true);
      lastMesh = null;
      downloadBtn.disabled = true;
      return;
    }
    if (msg.type === "result") {
      if (!pendingRequest || msg.id !== pendingRequest) {
        return;
      }
      pendingRequest = null;
      if (!msg.ok) {
        const errText = msg.error || "Unknown error.";
        if (errText.includes(goExitedError)) {
          statusEl.textContent = "WASM runtime exited. Reinitializing...";
          setOverlay(statusEl.textContent, true);
          initWorker({ silent: true });
          return;
        }
        statusEl.textContent = errText;
        setOverlay(statusEl.textContent, true);
        lastMesh = null;
        downloadBtn.disabled = true;
        return;
      }
      const positions = new Float32Array(msg.positions);
      const normals = new Float32Array(msg.normals);
      renderer.setMesh(positions, normals, msg.bounds);
      lastMesh = { positions, normals };
      downloadBtn.disabled = positions.length === 0;
      statusEl.textContent = `Triangles: ${positions.length / 9}`;
      setOverlay("", true);
      return;
    }
  };
  worker.onerror = (event) => {
    workerReady = false;
    statusEl.textContent = `Worker error: ${event.message}`;
    setOverlay(statusEl.textContent, true);
  };
  worker.postMessage({ type: "init" });
}

function compile() {
  if (pendingRequest) {
    return;
  }
  if (!workerReady) {
    statusEl.textContent = "WASM not ready.";
    setOverlay(statusEl.textContent, true);
    return;
  }
  const gridSize = Number(gridEl.value || "128");
  statusEl.textContent = "Compiling...";
  setOverlay("Compiling...");
  pendingRequest = ++requestId;
  worker.postMessage({
    type: "compile",
    id: pendingRequest,
    code: getSource(),
    gridSize,
  });
}

compileBtn.addEventListener("click", compile);
document.addEventListener("keydown", (event) => {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "s") {
    event.preventDefault();
    compile();
  }
});

setupResizer();
setupMobileToggle();

cancelBtn.addEventListener("click", () => {
  if (!pendingRequest) return;
  statusEl.textContent = "Compilation canceled.";
  setOverlay(statusEl.textContent, true);
  pendingRequest = null;
  initWorker({ silent: true });
});

downloadBtn.addEventListener("click", () => {
  if (!lastMesh || lastMesh.positions.length === 0) return;
  const stl = buildBinarySTL(lastMesh.positions, lastMesh.normals);
  const blob = new Blob([stl], { type: "application/sla" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = "model.stl";
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
});

resetViewBtn.addEventListener("click", () => {
  if (!renderer) return;
  renderer.resetView();
});

function MeshRenderer(canvas) {
  const gl = canvas.getContext("webgl");
  if (!gl) {
    overlayEl.textContent = "WebGL not available.";
    return;
  }
  this.canvas = canvas;
  this.gl = gl;
  this.program = null;
  this.buffers = null;
  this.attribs = null;
  this.uniforms = null;
  this.vertexCount = 0;
  this.meshData = null;
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

  this.initializeContextResources();
  setupInteraction(this, canvas);
  this.setupContextRecovery();
  this.requestRender();
  if (typeof ResizeObserver === "function") {
    this.resizeObserver = new ResizeObserver(() => this.requestRender());
    this.resizeObserver.observe(this.canvas);
  }
  window.addEventListener("resize", () => this.requestRender(), { passive: true });
}

MeshRenderer.prototype.setMesh = function (positions, normals, bounds) {
  const hadMesh = this.vertexCount > 0;
  this.meshData = { positions, normals };
  this.uploadMesh();
  this.bounds = bounds;
  // Preserve user camera settings across recompiles after the first successful mesh.
  this.fitPending = !hadMesh;
  this.requestRender();
};

MeshRenderer.prototype.fitView = function () {
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
};

MeshRenderer.prototype.resetView = function () {
  this.camera.theta = 0.6;
  this.camera.phi = 0.9;
  this.fitPending = true;
  this.requestRender();
};

MeshRenderer.prototype.requestRender = function () {
  if (this.contextLost) {
    return;
  }
  if (this.frameHandle != null) {
    return;
  }
  this.frameHandle = requestAnimationFrame(() => {
    this.frameHandle = null;
    this.render();
  });
};

MeshRenderer.prototype.render = function () {
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
  updateAxisIndicator(this.camera);

  if (this.vertexCount > 0) {
    const { model, view, proj, eye } = buildMatrices(this.camera, this.canvas);
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
};

MeshRenderer.prototype.initializeContextResources = function () {
  const gl = this.gl;
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
};

MeshRenderer.prototype.uploadMesh = function () {
  if (this.contextLost || !this.meshData || !this.buffers) {
    this.vertexCount = this.meshData ? this.meshData.positions.length / 3 : 0;
    return;
  }
  const gl = this.gl;
  const { positions, normals } = this.meshData;
  this.vertexCount = positions.length / 3;
  gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.position);
  gl.bufferData(gl.ARRAY_BUFFER, positions, gl.STATIC_DRAW);
  gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.normal);
  gl.bufferData(gl.ARRAY_BUFFER, normals, gl.STATIC_DRAW);
};

MeshRenderer.prototype.setupContextRecovery = function () {
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
};

function setupInteraction(renderer, canvas) {
  canvas.style.touchAction = "none";
  canvas.addEventListener("contextmenu", (event) => {
    event.preventDefault();
  });
  canvas.addEventListener("mousedown", (event) => {
    if (event.button !== 0 && event.button !== 2) {
      return;
    }
    renderer.dragging = true;
    renderer.dragMode = event.button === 2 ? "pan" : "rotate";
    renderer.lastPos = [event.clientX, event.clientY];
  });
  window.addEventListener("mouseup", () => {
    renderer.dragging = false;
    renderer.dragMode = null;
  });
  window.addEventListener("mousemove", (event) => {
    if (!renderer.dragging) return;
    const dx = event.clientX - renderer.lastPos[0];
    const dy = event.clientY - renderer.lastPos[1];
    renderer.lastPos = [event.clientX, event.clientY];
    if (renderer.dragMode === "pan") {
      panCamera(renderer.camera, dx, dy, canvas);
    } else {
      renderer.camera.theta -= dx * 0.005;
      renderer.camera.phi -= dy * 0.005;
      renderer.camera.phi = Math.min(Math.max(renderer.camera.phi, 0.1), Math.PI - 0.1);
    }
    renderer.requestRender();
  });
  canvas.addEventListener("wheel", (event) => {
    event.preventDefault();
    renderer.camera.radius *= 1 + event.deltaY * 0.001;
    renderer.camera.radius = Math.max(renderer.camera.radius, 0.4);
    renderer.requestRender();
  });

  const pointers = new Map();
  let lastPinchDist = null;

  const updatePinch = () => {
    if (pointers.size !== 2) {
      lastPinchDist = null;
      return;
    }
    const pts = Array.from(pointers.values());
    const dx = pts[0].x - pts[1].x;
    const dy = pts[0].y - pts[1].y;
    const dist = Math.hypot(dx, dy);
    if (lastPinchDist != null && lastPinchDist > 0) {
      const scale = lastPinchDist / dist;
      renderer.camera.radius *= scale;
      renderer.camera.radius = Math.max(renderer.camera.radius, 0.4);
      renderer.requestRender();
    }
    lastPinchDist = dist;
  };

  canvas.addEventListener("pointerdown", (event) => {
    pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });
    if (event.pointerType === "mouse") {
      if (event.button !== 0 && event.button !== 2) {
        return;
      }
      renderer.dragging = true;
      renderer.dragMode = event.button === 2 ? "pan" : "rotate";
      renderer.lastPos = [event.clientX, event.clientY];
    } else if (pointers.size === 1) {
      renderer.dragging = true;
      renderer.dragMode = "rotate";
      renderer.lastPos = [event.clientX, event.clientY];
    } else if (pointers.size === 2) {
      renderer.dragging = false;
      renderer.dragMode = null;
      lastPinchDist = null;
    }
    canvas.setPointerCapture(event.pointerId);
  });

  canvas.addEventListener("pointermove", (event) => {
    if (!pointers.has(event.pointerId)) return;
    pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });
    if (pointers.size === 1 && renderer.dragging) {
      const dx = event.clientX - renderer.lastPos[0];
      const dy = event.clientY - renderer.lastPos[1];
      renderer.lastPos = [event.clientX, event.clientY];
      if (renderer.dragMode === "pan") {
        panCamera(renderer.camera, dx, dy, canvas);
      } else {
        renderer.camera.theta -= dx * 0.005;
        renderer.camera.phi -= dy * 0.005;
        renderer.camera.phi = Math.min(Math.max(renderer.camera.phi, 0.1), Math.PI - 0.1);
      }
      renderer.requestRender();
    } else if (pointers.size === 2) {
      updatePinch();
    }
  });

  const onPointerEnd = (event) => {
    pointers.delete(event.pointerId);
    if (pointers.size === 1) {
      const remaining = Array.from(pointers.values())[0];
      renderer.dragging = true;
      renderer.dragMode = "rotate";
      renderer.lastPos = [remaining.x, remaining.y];
      lastPinchDist = null;
    } else if (pointers.size === 0) {
      renderer.dragging = false;
      renderer.dragMode = null;
      lastPinchDist = null;
    }
    if (canvas.hasPointerCapture(event.pointerId)) {
      canvas.releasePointerCapture(event.pointerId);
    }
  };

  canvas.addEventListener("pointerup", onPointerEnd);
  canvas.addEventListener("pointercancel", onPointerEnd);
}

function resizeCanvas(canvas) {
  const ratio = window.devicePixelRatio || 1;
  const width = Math.floor(canvas.clientWidth * ratio);
  const height = Math.floor(canvas.clientHeight * ratio);
  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width;
    canvas.height = height;
  }
}

function buildMatrices(camera, canvas) {
  const aspect = canvas.width / Math.max(canvas.height, 1);
  const proj = mat4Perspective(CAMERA_FOV_RAD, aspect, 0.01, 1000);
  const eye = [
    camera.target[0] +
      camera.radius * Math.cos(camera.theta) * Math.sin(camera.phi),
    camera.target[1] +
      camera.radius * Math.sin(camera.theta) * Math.sin(camera.phi),
    camera.target[2] + camera.radius * Math.cos(camera.phi),
  ];
  const view = mat4LookAt(eye, camera.target, [0, 0, 1]);
  const model = mat4Identity();
  return { model, view, proj, eye };
}

function mat4Identity() {
  return new Float32Array([1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]);
}

function mat4Perspective(fov, aspect, near, far) {
  const f = 1 / Math.tan(fov / 2);
  const nf = 1 / (near - far);
  return new Float32Array([
    f / aspect,
    0,
    0,
    0,
    0,
    f,
    0,
    0,
    0,
    0,
    (far + near) * nf,
    -1,
    0,
    0,
    2 * far * near * nf,
    0,
  ]);
}

function mat4LookAt(eye, target, up) {
  const z0 = eye[0] - target[0];
  const z1 = eye[1] - target[1];
  const z2 = eye[2] - target[2];
  const zLen = Math.hypot(z0, z1, z2) || 1;
  const zx = z0 / zLen;
  const zy = z1 / zLen;
  const zz = z2 / zLen;

  const x0 = up[1] * zz - up[2] * zy;
  const x1 = up[2] * zx - up[0] * zz;
  const x2 = up[0] * zy - up[1] * zx;
  const xLen = Math.hypot(x0, x1, x2) || 1;
  const xx = x0 / xLen;
  const xy = x1 / xLen;
  const xz = x2 / xLen;

  const yx = zy * xz - zz * xy;
  const yy = zz * xx - zx * xz;
  const yz = zx * xy - zy * xx;

  return new Float32Array([
    xx,
    yx,
    zx,
    0,
    xy,
    yy,
    zy,
    0,
    xz,
    yz,
    zz,
    0,
    -(xx * eye[0] + xy * eye[1] + xz * eye[2]),
    -(yx * eye[0] + yy * eye[1] + yz * eye[2]),
    -(zx * eye[0] + zy * eye[1] + zz * eye[2]),
    1,
  ]);
}

function normalize3(v) {
  const len = Math.hypot(v[0], v[1], v[2]) || 1;
  return [v[0] / len, v[1] / len, v[2] / len];
}

function getCameraBasis(camera) {
  const eye = [
    camera.target[0] + camera.radius * Math.cos(camera.theta) * Math.sin(camera.phi),
    camera.target[1] + camera.radius * Math.sin(camera.theta) * Math.sin(camera.phi),
    camera.target[2] + camera.radius * Math.cos(camera.phi),
  ];
  const forward = normalize3([
    camera.target[0] - eye[0],
    camera.target[1] - eye[1],
    camera.target[2] - eye[2],
  ]);
  let right = cross3(forward, [0, 0, 1]);
  if (Math.hypot(right[0], right[1], right[2]) < 1e-8) {
    right = [1, 0, 0];
  } else {
    right = normalize3(right);
  }
  const up = normalize3(cross3(right, forward));
  return { right, up };
}

function panCamera(camera, dx, dy, canvas) {
  const viewportHeight = Math.max(canvas.clientHeight, 1);
  const worldPerPixel = (2 * camera.radius * Math.tan(CAMERA_FOV_RAD / 2)) / viewportHeight;
  const { right, up } = getCameraBasis(camera);
  const rightDelta = -dx * worldPerPixel;
  const upDelta = dy * worldPerPixel;
  camera.target = [
    camera.target[0] + right[0] * rightDelta + up[0] * upDelta,
    camera.target[1] + right[1] * rightDelta + up[1] * upDelta,
    camera.target[2] + right[2] * rightDelta + up[2] * upDelta,
  ];
}

function dot3(a, b) {
  return a[0] * b[0] + a[1] * b[1] + a[2] * b[2];
}

function cross3(a, b) {
  return [
    a[1] * b[2] - a[2] * b[1],
    a[2] * b[0] - a[0] * b[2],
    a[0] * b[1] - a[1] * b[0],
  ];
}

function setAxis(lineEl, labelEl, sx, sy, depth) {
  if (!lineEl || !labelEl) return;
  const cx = 32;
  const cy = 32;
  const extent = 16;
  const x2 = cx + sx * extent;
  const y2 = cy - sy * extent;
  const labelDist = 3.5;
  const lx = cx + sx * (extent + labelDist);
  const ly = cy - sy * (extent + labelDist);
  const opacity = 0.35 + 0.65 * (depth + 1) * 0.5;
  lineEl.setAttribute("x1", `${cx}`);
  lineEl.setAttribute("y1", `${cy}`);
  lineEl.setAttribute("x2", x2.toFixed(2));
  lineEl.setAttribute("y2", y2.toFixed(2));
  lineEl.style.opacity = opacity.toFixed(2);
  labelEl.setAttribute("x", lx.toFixed(2));
  labelEl.setAttribute("y", ly.toFixed(2));
  labelEl.style.opacity = opacity.toFixed(2);
}

function updateAxisIndicator(camera) {
  if (!axisLineX || !axisLineY || !axisLineZ || !axisLabelX || !axisLabelY || !axisLabelZ) {
    return;
  }
  const eye = [
    camera.target[0] + camera.radius * Math.cos(camera.theta) * Math.sin(camera.phi),
    camera.target[1] + camera.radius * Math.sin(camera.theta) * Math.sin(camera.phi),
    camera.target[2] + camera.radius * Math.cos(camera.phi),
  ];
  const forward = normalize3([
    camera.target[0] - eye[0],
    camera.target[1] - eye[1],
    camera.target[2] - eye[2],
  ]);
  const worldUp = [0, 0, 1];
  const right = normalize3(cross3(forward, worldUp));
  const up = normalize3(cross3(right, forward));
  setAxis(axisLineX, axisLabelX, dot3([1, 0, 0], right), dot3([1, 0, 0], up), dot3([1, 0, 0], forward));
  setAxis(axisLineY, axisLabelY, dot3([0, 1, 0], right), dot3([0, 1, 0], up), dot3([0, 1, 0], forward));
  setAxis(axisLineZ, axisLabelZ, dot3([0, 0, 1], right), dot3([0, 0, 1], up), dot3([0, 0, 1], forward));
}

function createShader(gl, type, source) {
  const shader = gl.createShader(type);
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    const err = gl.getShaderInfoLog(shader);
    gl.deleteShader(shader);
    throw new Error(err);
  }
  return shader;
}

function createProgram(gl, vsSource, fsSource) {
  const program = gl.createProgram();
  const vs = createShader(gl, gl.VERTEX_SHADER, vsSource);
  const fs = createShader(gl, gl.FRAGMENT_SHADER, fsSource);
  gl.attachShader(program, vs);
  gl.attachShader(program, fs);
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    const err = gl.getProgramInfoLog(program);
    gl.deleteProgram(program);
    throw new Error(err);
  }
  return program;
}

renderer = new MeshRenderer(canvas);
initWorker();

function setOverlay(text, idle) {
  const show = text && text.trim().length > 0;
  overlayCard.style.display = show ? "flex" : "none";
  overlayText.textContent = text || "";
  const isIdle = Boolean(idle);
  spinnerEl.style.display = isIdle ? "none" : "block";
  cancelBtn.style.display = isIdle ? "none" : "block";
}

function setupMobileToggle() {
  if (!appEl || !toggleCodeBtn || !togglePreviewBtn) return;
  const showCode = () => {
    appEl.classList.add("show-code");
    appEl.classList.remove("show-preview");
    toggleCodeBtn.classList.add("active");
    togglePreviewBtn.classList.remove("active");
  };
  const showPreview = () => {
    appEl.classList.add("show-preview");
    appEl.classList.remove("show-code");
    togglePreviewBtn.classList.add("active");
    toggleCodeBtn.classList.remove("active");
  };
  toggleCodeBtn.addEventListener("click", showCode);
  togglePreviewBtn.addEventListener("click", showPreview);
  showCode();
}

function buildBinarySTL(positions, normals) {
  const triCount = Math.floor(positions.length / 9);
  const buffer = new ArrayBuffer(84 + triCount * 50);
  const view = new DataView(buffer);
  let offset = 80;
  view.setUint32(offset, triCount, true);
  offset += 4;
  for (let i = 0; i < triCount; i++) {
    const nIdx = i * 9;
    const pIdx = i * 9;
    const n0 = normals[nIdx] ?? 0;
    const n1 = normals[nIdx + 1] ?? 0;
    const n2 = normals[nIdx + 2] ?? 0;
    view.setFloat32(offset, n0, true);
    view.setFloat32(offset + 4, n1, true);
    view.setFloat32(offset + 8, n2, true);
    offset += 12;
    for (let v = 0; v < 9; v++) {
      view.setFloat32(offset, positions[pIdx + v] ?? 0, true);
      offset += 4;
    }
    view.setUint16(offset, 0, true);
    offset += 2;
  }
  return buffer;
}

function setupResizer() {
  if (!resizer) return;
  const app = document.querySelector(".app");
  if (!app) return;
  const minLeft = 260;
  const minRight = 320;
  const mobileBreakpoint = 900;
  let dragging = false;
  let preferredLeft = null;

  const clampLeft = (left, width) =>
    Math.min(Math.max(left, minLeft), Math.max(minLeft, width - minRight));

  const applyLayout = (left) => {
    if (window.innerWidth <= mobileBreakpoint) {
      app.style.gridTemplateColumns = "";
      if (renderer) {
        renderer.requestRender();
      }
      return;
    }
    const rect = app.getBoundingClientRect();
    const clampedLeft = clampLeft(left, rect.width);
    preferredLeft = clampedLeft;
    const right = rect.width - clampedLeft;
    app.style.gridTemplateColumns = `${clampedLeft}px 8px ${right}px`;
    if (renderer) {
      renderer.requestRender();
    }
  };

  const onMove = (event) => {
    if (!dragging) return;
    const rect = app.getBoundingClientRect();
    applyLayout(event.clientX - rect.left);
  };

  resizer.addEventListener("mousedown", (event) => {
    dragging = true;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
    onMove(event);
  });

  window.addEventListener("mousemove", onMove);
  window.addEventListener("mouseup", () => {
    if (!dragging) return;
    dragging = false;
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  });
  window.addEventListener("resize", () => {
    if (preferredLeft == null) return;
    applyLayout(preferredLeft);
  });
}
