(() => {
  if (globalThis.__devBrowser_getHarnessState) return;

  const state = {
    startedAt: Date.now(),
    errors: [],
    overlays: [],
    lastOverlayText: null,
  };

  function nowMs() { return Date.now(); }

  function pushError(entry) {
    try {
      entry.time_ms = entry.time_ms || nowMs();
      state.errors.push(entry);
      if (state.errors.length > 200) state.errors = state.errors.slice(state.errors.length - 200);
    } catch {}
  }

  // window.onerror
  globalThis.addEventListener('error', (event) => {
    try {
      const err = event && event.error;
      pushError({
        type: 'error',
        message: String(event && event.message || (err && err.message) || ''),
        source: String(event && event.filename || ''),
        line: Number(event && event.lineno || 0) || 0,
        column: Number(event && event.colno || 0) || 0,
        stack: err && err.stack ? String(err.stack) : null,
      });
    } catch {}
  });

  // unhandledrejection
  globalThis.addEventListener('unhandledrejection', (event) => {
    try {
      const reason = event && event.reason;
      pushError({
        type: 'unhandledrejection',
        message: String(reason && reason.message ? reason.message : reason),
        stack: reason && reason.stack ? String(reason.stack) : null,
      });
    } catch {}
  });

  // Vite dev overlay detection (best-effort)
  function readViteOverlay() {
    const el = globalThis.document && globalThis.document.querySelector
      ? globalThis.document.querySelector('vite-error-overlay')
      : null;
    if (!el) return null;

    // Vite overlay content is in shadow root.
    let text = '';
    try {
      const root = el.shadowRoot;
      if (root) {
        const pre = root.querySelector('pre');
        text = pre ? (pre.innerText || pre.textContent || '') : (root.innerText || root.textContent || '');
      } else {
        text = el.innerText || el.textContent || '';
      }
    } catch {}

    text = String(text || '').trim();
    if (!text) return { detected: true, text: null };
    if (text.length > 4000) text = text.slice(0, 4000);
    return { detected: true, text };
  }

  function pollOverlay() {
    try {
      const ov = readViteOverlay();
      if (ov && ov.detected) {
        const text = ov.text || '';
        // Record overlay when detected, even if text extraction failed
        if (text !== state.lastOverlayText) {
          state.lastOverlayText = text;
          state.overlays.push({ type: 'vite', time_ms: nowMs(), text: text || null });
          if (state.overlays.length > 50) state.overlays = state.overlays.slice(state.overlays.length - 50);
        }
      }
    } catch {}
  }

  // Polling is cheap; headless-friendly.
  setInterval(pollOverlay, 500);
  pollOverlay();

  globalThis.__devBrowser_getHarnessState = () => ({
    startedAt: state.startedAt,
    errors: state.errors.slice(),
    overlays: state.overlays.slice(),
  });
})();
