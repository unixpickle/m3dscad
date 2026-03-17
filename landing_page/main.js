import { MeshRenderer, parseBinarySTL } from "./renderer.js";

const pageBody = document.body;
const updateScrolledState = () => {
  pageBody.classList.toggle("home-scrolled", window.scrollY > 8);
};
updateScrolledState();
window.addEventListener("scroll", updateScrolledState, { passive: true });

const cards = Array.from(document.querySelectorAll(".example-card[data-example-id]"));

cards.forEach((card) => {
  const id = card.getAttribute("data-example-id");
  if (!id) return;
  const statusEl = card.querySelector(".viewer-status");
  const canvasEl = card.querySelector(".example-canvas");
  const resetBtn = card.querySelector(".reset-view");
  if (!statusEl || !canvasEl || !resetBtn) return;

  fetch(`./assets/examples/${id}.stl`)
    .then((r) => {
      if (!r.ok) throw new Error(`Could not fetch ${id}.stl`);
      return r.arrayBuffer();
    })
    .then((stlBytes) => {
      const renderer = new MeshRenderer(canvasEl);
      const mesh = parseBinarySTL(stlBytes);
      renderer.setMesh(mesh.positions, mesh.normals, mesh.bounds);
      resetBtn.addEventListener("click", () => renderer.resetView());
      statusEl.textContent = "";
    })
    .catch((err) => {
      statusEl.textContent = err.message;
    });
});
