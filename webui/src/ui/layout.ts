interface RenderRequester {
  requestRender(): void;
}

interface MobileToggleOptions {
  appEl: HTMLElement;
  toggleCodeBtn: HTMLButtonElement;
  togglePreviewBtn: HTMLButtonElement;
}

interface ResizerOptions {
  resizer: HTMLElement;
  appEl: HTMLElement;
  getRenderer: () => RenderRequester | null;
  minLeft?: number;
  minRight?: number;
  mobileBreakpoint?: number;
}

export function setupMobileToggle({
  appEl,
  toggleCodeBtn,
  togglePreviewBtn,
}: MobileToggleOptions): void {
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

export function setupResizer({
  resizer,
  appEl,
  getRenderer,
  minLeft = 260,
  minRight = 320,
  mobileBreakpoint = 900,
}: ResizerOptions): void {
  let dragging = false;
  let preferredLeft: number | null = null;

  const requestRender = () => {
    const renderer = getRenderer();
    if (renderer) {
      renderer.requestRender();
    }
  };

  const clampLeft = (left: number, width: number): number =>
    Math.min(Math.max(left, minLeft), Math.max(minLeft, width - minRight));

  const applyLayout = (left: number): void => {
    if (window.innerWidth <= mobileBreakpoint) {
      appEl.style.gridTemplateColumns = "";
      requestRender();
      return;
    }
    const rect = appEl.getBoundingClientRect();
    const clampedLeft = clampLeft(left, rect.width);
    preferredLeft = clampedLeft;
    const right = rect.width - clampedLeft;
    appEl.style.gridTemplateColumns = `${clampedLeft}px 8px ${right}px`;
    requestRender();
  };

  const onMove = (event: MouseEvent): void => {
    if (!dragging) {
      return;
    }
    const rect = appEl.getBoundingClientRect();
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
    if (!dragging) {
      return;
    }
    dragging = false;
    document.body.style.cursor = "";
    document.body.style.userSelect = "";
  });
  window.addEventListener("resize", () => {
    if (preferredLeft == null) {
      return;
    }
    applyLayout(preferredLeft);
  });
}
