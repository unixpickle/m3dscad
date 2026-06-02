export const vertexShader = `
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

export const fragmentShader = `
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

export function createShader(gl, type, source) {
  const shader = gl.createShader(type);
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    const err = gl.getShaderInfoLog(shader) || "Shader compilation failed";
    gl.deleteShader(shader);
    throw new Error(err);
  }
  return shader;
}

export function createProgram(gl, vsSource, fsSource) {
  const program = gl.createProgram();
  const vs = createShader(gl, gl.VERTEX_SHADER, vsSource);
  const fs = createShader(gl, gl.FRAGMENT_SHADER, fsSource);
  gl.attachShader(program, vs);
  gl.attachShader(program, fs);
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    const err = gl.getProgramInfoLog(program) || "Program linking failed";
    gl.deleteProgram(program);
    throw new Error(err);
  }
  return program;
}
