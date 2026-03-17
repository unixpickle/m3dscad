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
    vec3 color = u_color * (0.2 + diff * 0.8) + vec3(0.12, 0.15, 0.24) * rim;
    gl_FragColor = vec4(color, 1.0);
  }
`;

export class MeshRenderer {
  constructor(canvas) {
    const gl = canvas.getContext("webgl");
    if (!gl) {
      throw new Error("WebGL not available");
    }
    this.canvas = canvas;
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

  setMesh(positions, normals, bounds) {
    const gl = this.gl;
    this.vertexCount = positions.length / 3;
    gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.position);
    gl.bufferData(gl.ARRAY_BUFFER, positions, gl.STATIC_DRAW);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.normal);
    gl.bufferData(gl.ARRAY_BUFFER, normals, gl.STATIC_DRAW);
    this.bounds = bounds;
    this.fitPending = true;
  }

  fitView() {
    if (!this.bounds) return;
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
    this.camera.radius = radius * 1.45;
    this.fitPending = false;
  }

  resetView() {
    this.camera.theta = 0.6;
    this.camera.phi = 0.9;
    this.fitPending = true;
  }

  render() {
    const gl = this.gl;
    resizeCanvas(this.canvas);
    if (this.fitPending) {
      this.fitView();
    }

    gl.viewport(0, 0, this.canvas.width, this.canvas.height);
    gl.clearColor(0.05, 0.06, 0.09, 1);
    gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);
    gl.enable(gl.DEPTH_TEST);
    gl.useProgram(this.program);

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
      gl.uniform3f(this.uniforms.color, 0.72, 0.8, 0.95);

      gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.position);
      gl.enableVertexAttribArray(this.attribs.position);
      gl.vertexAttribPointer(this.attribs.position, 3, gl.FLOAT, false, 0, 0);
      gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers.normal);
      gl.enableVertexAttribArray(this.attribs.normal);
      gl.vertexAttribPointer(this.attribs.normal, 3, gl.FLOAT, false, 0, 0);

      gl.drawArrays(gl.TRIANGLES, 0, this.vertexCount);
    }

    requestAnimationFrame(() => this.render());
  }
}

export function parseBinarySTL(arrayBuffer) {
  if (arrayBuffer.byteLength < 84) {
    throw new Error("STL too small");
  }
  const view = new DataView(arrayBuffer);
  const triangleCount = view.getUint32(80, true);
  const expectedSize = 84 + triangleCount * 50;
  if (expectedSize > arrayBuffer.byteLength) {
    throw new Error("Corrupt STL triangle count");
  }

  const positions = new Float32Array(triangleCount * 9);
  const normals = new Float32Array(triangleCount * 9);

  let pMin = [Infinity, Infinity, Infinity];
  let pMax = [-Infinity, -Infinity, -Infinity];
  let offset = 84;

  for (let i = 0; i < triangleCount; i++) {
    const nx = view.getFloat32(offset, true);
    const ny = view.getFloat32(offset + 4, true);
    const nz = view.getFloat32(offset + 8, true);
    offset += 12;

    const base = i * 9;
    for (let j = 0; j < 9; j += 3) {
      const x = view.getFloat32(offset, true);
      const y = view.getFloat32(offset + 4, true);
      const z = view.getFloat32(offset + 8, true);
      positions[base + j] = x;
      positions[base + j + 1] = y;
      positions[base + j + 2] = z;
      pMin[0] = Math.min(pMin[0], x);
      pMin[1] = Math.min(pMin[1], y);
      pMin[2] = Math.min(pMin[2], z);
      pMax[0] = Math.max(pMax[0], x);
      pMax[1] = Math.max(pMax[1], y);
      pMax[2] = Math.max(pMax[2], z);
      offset += 12;
    }

    const [fnx, fny, fnz] = (nx === 0 && ny === 0 && nz === 0)
      ? triangleNormal(
        positions[base], positions[base + 1], positions[base + 2],
        positions[base + 3], positions[base + 4], positions[base + 5],
        positions[base + 6], positions[base + 7], positions[base + 8],
      )
      : normalize3([nx, ny, nz]);

    for (let j = 0; j < 9; j += 3) {
      normals[base + j] = fnx;
      normals[base + j + 1] = fny;
      normals[base + j + 2] = fnz;
    }

    offset += 2; // attribute byte count
  }

  return {
    positions,
    normals,
    bounds: {
      min: pMin,
      max: pMax,
    },
  };
}

function triangleNormal(ax, ay, az, bx, by, bz, cx, cy, cz) {
  const ux = bx - ax;
  const uy = by - ay;
  const uz = bz - az;
  const vx = cx - ax;
  const vy = cy - ay;
  const vz = cz - az;
  return normalize3([
    uy * vz - uz * vy,
    uz * vx - ux * vz,
    ux * vy - uy * vx,
  ]);
}

function setupInteraction(renderer, canvas) {
  canvas.style.touchAction = "none";

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
    camera.target[0] + camera.radius * Math.cos(camera.theta) * Math.sin(camera.phi),
    camera.target[1] + camera.radius * Math.sin(camera.theta) * Math.sin(camera.phi),
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
    f / aspect, 0, 0, 0,
    0, f, 0, 0,
    0, 0, (far + near) * nf, -1,
    0, 0, 2 * far * near * nf, 0,
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
    xx, yx, zx, 0,
    xy, yy, zy, 0,
    xz, yz, zz, 0,
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
