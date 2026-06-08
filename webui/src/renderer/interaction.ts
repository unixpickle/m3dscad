import { panCamera } from "./math";
import type { MeshRenderer } from "./mesh_renderer";

interface PointerPoint {
  x: number;
  y: number;
}

function clampCameraPhi(renderer: MeshRenderer): void {
  renderer.camera.phi = Math.min(
    Math.max(renderer.camera.phi, 0.1),
    Math.PI - 0.1,
  );
}

export function setupInteraction(
  renderer: MeshRenderer,
  canvas: HTMLCanvasElement,
): void {
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
    if (!renderer.dragging) {
      return;
    }
    const dx = event.clientX - renderer.lastPos[0];
    const dy = event.clientY - renderer.lastPos[1];
    renderer.lastPos = [event.clientX, event.clientY];
    if (renderer.dragMode === "pan") {
      panCamera(renderer.camera, dx, dy, canvas);
    } else {
      renderer.camera.theta -= dx * 0.005;
      renderer.camera.phi -= dy * 0.005;
      clampCameraPhi(renderer);
    }
    renderer.requestRender();
  });
  canvas.addEventListener("wheel", (event) => {
    event.preventDefault();
    renderer.camera.radius *= 1 + event.deltaY * 0.001;
    renderer.camera.radius = Math.max(renderer.camera.radius, 0.4);
    renderer.requestRender();
  });

  const pointers = new Map<number, PointerPoint>();
  let lastPinchDist: number | null = null;
  let lastPinchCenter: PointerPoint | null = null;

  const updatePinch = (): void => {
    if (pointers.size !== 2) {
      lastPinchDist = null;
      lastPinchCenter = null;
      return;
    }
    const pts = Array.from(pointers.values());
    const dx = pts[0].x - pts[1].x;
    const dy = pts[0].y - pts[1].y;
    const dist = Math.hypot(dx, dy);
    const center = {
      x: (pts[0].x + pts[1].x) / 2,
      y: (pts[0].y + pts[1].y) / 2,
    };
    if (lastPinchCenter != null) {
      panCamera(
        renderer.camera,
        center.x - lastPinchCenter.x,
        center.y - lastPinchCenter.y,
        canvas,
      );
    }
    if (lastPinchDist != null && lastPinchDist > 0 && dist > 0) {
      const scale = lastPinchDist / dist;
      renderer.camera.radius *= scale;
      renderer.camera.radius = Math.max(renderer.camera.radius, 0.4);
    }
    lastPinchDist = dist;
    lastPinchCenter = center;
    renderer.requestRender();
  };

  canvas.addEventListener("pointerdown", (event) => {
    pointers.set(event.pointerId, { x: event.clientX, y: event.clientY });
    if (event.pointerType === "mouse") {
      if (event.button !== 0 && event.button !== 2) {
        pointers.delete(event.pointerId);
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
      lastPinchCenter = null;
    }
    canvas.setPointerCapture(event.pointerId);
  });

  canvas.addEventListener("pointermove", (event) => {
    if (!pointers.has(event.pointerId)) {
      return;
    }
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
        clampCameraPhi(renderer);
      }
      renderer.requestRender();
    } else if (pointers.size === 2) {
      updatePinch();
    }
  });

  const onPointerEnd = (event: PointerEvent): void => {
    pointers.delete(event.pointerId);
    if (pointers.size === 1) {
      const remaining = Array.from(pointers.values())[0];
      renderer.dragging = true;
      renderer.dragMode = "rotate";
      renderer.lastPos = [remaining.x, remaining.y];
      lastPinchDist = null;
      lastPinchCenter = null;
    } else if (pointers.size === 0) {
      renderer.dragging = false;
      renderer.dragMode = null;
      lastPinchDist = null;
      lastPinchCenter = null;
    }
    if (canvas.hasPointerCapture(event.pointerId)) {
      canvas.releasePointerCapture(event.pointerId);
    }
  };

  canvas.addEventListener("pointerup", onPointerEnd);
  canvas.addEventListener("pointercancel", onPointerEnd);
}
