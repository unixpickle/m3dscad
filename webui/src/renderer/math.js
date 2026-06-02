export const CAMERA_FOV_RAD = (60 * Math.PI) / 180;

export function resizeCanvas(canvas) {
  const ratio = window.devicePixelRatio || 1;
  const width = Math.floor(canvas.clientWidth * ratio);
  const height = Math.floor(canvas.clientHeight * ratio);
  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width;
    canvas.height = height;
  }
}

export function cameraEye(camera) {
  return [
    camera.target[0] +
      camera.radius * Math.cos(camera.theta) * Math.sin(camera.phi),
    camera.target[1] +
      camera.radius * Math.sin(camera.theta) * Math.sin(camera.phi),
    camera.target[2] + camera.radius * Math.cos(camera.phi),
  ];
}

export function buildMatrices(camera, canvas, bounds) {
  const aspect = canvas.width / Math.max(canvas.height, 1);
  const eye = cameraEye(camera);
  const { near, far } = clipPlanesForBounds(eye, bounds);
  const proj = mat4Perspective(CAMERA_FOV_RAD, aspect, near, far);
  const view = mat4LookAt(eye, camera.target, [0, 0, 1]);
  const model = mat4Identity();
  return { model, view, proj, eye };
}

export function clipPlanesForBounds(eye, bounds) {
  if (!bounds) {
    return { near: 0.01, far: 1000 };
  }
  const center = [
    (bounds.min[0] + bounds.max[0]) / 2,
    (bounds.min[1] + bounds.max[1]) / 2,
    (bounds.min[2] + bounds.max[2]) / 2,
  ];
  const dx = bounds.max[0] - bounds.min[0];
  const dy = bounds.max[1] - bounds.min[1];
  const dz = bounds.max[2] - bounds.min[2];
  const sceneRadius = Math.max(Math.hypot(dx, dy, dz) * 0.5, 1);
  const distanceToCenter = Math.hypot(
    eye[0] - center[0],
    eye[1] - center[1],
    eye[2] - center[2],
  );
  const farthestPoint = distanceToCenter + sceneRadius;
  const far = Math.max(farthestPoint * 1.5, 1);
  const surfaceDistance = Math.max(distanceToCenter - sceneRadius, 0);
  const near = Math.max(
    0.01,
    Math.min(Math.max(surfaceDistance * 0.5, 0.01), far / 1000),
  );
  return { near, far };
}

export function mat4Identity() {
  return new Float32Array([1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]);
}

export function mat4Perspective(fov, aspect, near, far) {
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

export function mat4LookAt(eye, target, up) {
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

export function normalize3(v) {
  const len = Math.hypot(v[0], v[1], v[2]) || 1;
  return [v[0] / len, v[1] / len, v[2] / len];
}

export function getCameraBasis(camera) {
  const eye = cameraEye(camera);
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

export function panCamera(camera, dx, dy, canvas) {
  const viewportHeight = Math.max(canvas.clientHeight, 1);
  const worldPerPixel =
    (2 * camera.radius * Math.tan(CAMERA_FOV_RAD / 2)) / viewportHeight;
  const { right, up } = getCameraBasis(camera);
  const rightDelta = -dx * worldPerPixel;
  const upDelta = dy * worldPerPixel;
  camera.target = [
    camera.target[0] + right[0] * rightDelta + up[0] * upDelta,
    camera.target[1] + right[1] * rightDelta + up[1] * upDelta,
    camera.target[2] + right[2] * rightDelta + up[2] * upDelta,
  ];
}

export function dot3(a, b) {
  return a[0] * b[0] + a[1] * b[1] + a[2] * b[2];
}

export function cross3(a, b) {
  return [
    a[1] * b[2] - a[2] * b[1],
    a[2] * b[0] - a[0] * b[2],
    a[0] * b[1] - a[1] * b[0],
  ];
}
