const codeEl = document.getElementById("code");
const statusEl = document.getElementById("status");
const overlayEl = document.getElementById("overlay");
const overlayCard = document.getElementById("overlayCard");
const overlayText = document.getElementById("overlayText");
const spinnerEl = document.getElementById("spinner");
const cancelBtn = document.getElementById("cancel");
const compileBtn = document.getElementById("compile");
const gridEl = document.getElementById("grid");
const canvas = document.getElementById("preview");

const defaultSource = `// Example
difference() {
  cube(2, center=true);
  translate([1, 1, 1]) sphere(1);
}
`;

const storedSource = window.localStorage.getItem("m3dscad_source");
codeEl.value = storedSource && storedSource.trim().length > 0 ? storedSource : defaultSource;

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

let renderer = null;
let worker = null;
let workerReady = false;
let requestId = 0;
let pendingRequest = null;

function initWorker() {
  setOverlay("Loading WASM...");
  workerReady = false;
  if (worker) {
    worker.terminate();
  }
  worker = new Worker("./worker.js");
  worker.postMessage({ type: "init" });
  worker.onmessage = (event) => {
    const msg = event.data;
    if (!msg || !msg.type) return;
    if (msg.type === "ready") {
      workerReady = true;
      setOverlay("", true);
      statusEl.textContent = "WASM ready. Press Command+S to compile.";
      return;
    }
    if (msg.type === "result") {
      if (!pendingRequest || msg.id !== pendingRequest) {
        return;
      }
      pendingRequest = null;
      if (!msg.ok) {
        statusEl.textContent = msg.error || "Unknown error.";
        setOverlay(statusEl.textContent, true);
        return;
      }
      const positions = new Float32Array(msg.positions);
      const normals = new Float32Array(msg.normals);
      renderer.setMesh(positions, normals, msg.bounds);
      statusEl.textContent = `Triangles: ${positions.length / 9}`;
      setOverlay("", true);
    }
  };
  worker.onerror = (event) => {
    workerReady = false;
    statusEl.textContent = `Worker error: ${event.message}`;
    setOverlay(statusEl.textContent, true);
  };
}

function compile() {
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
    code: codeEl.value,
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

codeEl.addEventListener("keydown", (event) => {
  if (event.key === "Tab") {
    event.preventDefault();
    const start = codeEl.selectionStart;
    const end = codeEl.selectionEnd;
    const value = codeEl.value;
    const insert = "  ";
    codeEl.value = value.slice(0, start) + insert + value.slice(end);
    codeEl.selectionStart = start + insert.length;
    codeEl.selectionEnd = start + insert.length;
  }
});

codeEl.addEventListener("input", () => {
  window.localStorage.setItem("m3dscad_source", codeEl.value);
});

cancelBtn.addEventListener("click", () => {
  if (!pendingRequest) return;
  statusEl.textContent = "Compilation canceled.";
  setOverlay(statusEl.textContent, true);
  pendingRequest = null;
  initWorker();
});

function MeshRenderer(canvas) {
  const gl = canvas.getContext("webgl");
  if (!gl) {
    overlayEl.textContent = "WebGL not available.";
    return;
  }
  this.gl = gl;
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
  this.vertexCount = 0;
  this.camera = {
    theta: 0.6,
    phi: 0.9,
    radius: 6,
    target: [0, 0, 0],
  };
  this.dragging = false;
  this.lastPos = [0, 0];
  this.fitPending = true;

  setupInteraction(this, canvas);
  requestAnimationFrame(() => this.render());
}

MeshRenderer.prototype.setMesh = function (positions, normals, bounds) {
  const gl = this.gl;
  this.vertexCount = positions.length / 3;
  gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.position);
  gl.bufferData(gl.ARRAY_BUFFER, positions, gl.STATIC_DRAW);
  gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.normal);
  gl.bufferData(gl.ARRAY_BUFFER, normals, gl.STATIC_DRAW);
  this.bounds = bounds;
  this.fitPending = true;
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

MeshRenderer.prototype.render = function () {
  const gl = this.gl;
  if (!gl) {
    return;
  }
  resizeCanvas(canvas);
  if (this.fitPending) {
    this.fitView();
  }
  gl.viewport(0, 0, canvas.width, canvas.height);
  gl.clearColor(0.05, 0.06, 0.09, 1);
  gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
  gl.enable(gl.DEPTH_TEST);
  gl.useProgram(this.program);

  if (this.vertexCount > 0) {
    const { model, view, proj, eye } = buildMatrices(this.camera, canvas);
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
  requestAnimationFrame(() => this.render());
};

function setupInteraction(renderer, canvas) {
  canvas.addEventListener("mousedown", (event) => {
    renderer.dragging = true;
    renderer.lastPos = [event.clientX, event.clientY];
  });
  window.addEventListener("mouseup", () => {
    renderer.dragging = false;
  });
  window.addEventListener("mousemove", (event) => {
    if (!renderer.dragging) return;
    const dx = event.clientX - renderer.lastPos[0];
    const dy = event.clientY - renderer.lastPos[1];
    renderer.lastPos = [event.clientX, event.clientY];
    renderer.camera.theta -= dx * 0.005;
    renderer.camera.phi -= dy * 0.005;
    renderer.camera.phi = Math.min(Math.max(renderer.camera.phi, 0.1), Math.PI - 0.1);
  });
  canvas.addEventListener("wheel", (event) => {
    event.preventDefault();
    renderer.camera.radius *= 1 + event.deltaY * 0.001;
    renderer.camera.radius = Math.max(renderer.camera.radius, 0.4);
  });
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
  const proj = mat4Perspective((60 * Math.PI) / 180, aspect, 0.01, 1000);
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
