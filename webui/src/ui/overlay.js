export function createOverlayController({
  overlayCard,
  overlayText,
  spinnerEl,
  cancelBtn,
}) {
  return {
    set(text, options = {}) {
      const show = text && text.trim().length > 0;
      overlayCard.style.display = show ? "flex" : "none";
      overlayText.textContent = text || "";
      const isIdle = Boolean(options.idle);
      const isCancelable = show && !isIdle && Boolean(options.cancelable);
      spinnerEl.style.display = isIdle ? "none" : "block";
      cancelBtn.style.display = isCancelable ? "block" : "none";
    },
  };
}
