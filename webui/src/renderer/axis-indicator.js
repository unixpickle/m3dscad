import { cameraEye, cross3, dot3, normalize3 } from "./math.js";

function setAxis(lineEl, labelEl, sx, sy, depth) {
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

export function updateAxisIndicator(camera, axisElements) {
  const {
    axisLineX,
    axisLineY,
    axisLineZ,
    axisLabelX,
    axisLabelY,
    axisLabelZ,
  } = axisElements;
  const eye = cameraEye(camera);
  const forward = normalize3([
    camera.target[0] - eye[0],
    camera.target[1] - eye[1],
    camera.target[2] - eye[2],
  ]);
  const worldUp = [0, 0, 1];
  const right = normalize3(cross3(forward, worldUp));
  const up = normalize3(cross3(right, forward));
  setAxis(
    axisLineX,
    axisLabelX,
    dot3([1, 0, 0], right),
    dot3([1, 0, 0], up),
    dot3([1, 0, 0], forward),
  );
  setAxis(
    axisLineY,
    axisLabelY,
    dot3([0, 1, 0], right),
    dot3([0, 1, 0], up),
    dot3([0, 1, 0], forward),
  );
  setAxis(
    axisLineZ,
    axisLabelZ,
    dot3([0, 0, 1], right),
    dot3([0, 0, 1], up),
    dot3([0, 0, 1], forward),
  );
}
