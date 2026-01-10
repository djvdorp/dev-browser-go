from __future__ import annotations

from typing import Final

_BASE_SCRIPT: Final[str] = r"""
(() => {
  if (globalThis.__devBrowser_getAISnapshot) return;

  const INTERACTIVE_ROLE = new Set([
    "button","checkbox","combobox","link","listbox","menuitem","option","radio",
    "searchbox","slider","spinbutton","switch","tab","textbox","treeitem"
  ]);

  function norm(text) {
    return (text || "").replace(/\s+/g, " ").trim();
  }

  function isHidden(el) {
    try {
      const style = el.ownerDocument && el.ownerDocument.defaultView
        ? el.ownerDocument.defaultView.getComputedStyle(el)
        : null;
      if (style && (style.display === "none" || style.visibility === "hidden")) return true;
      if (el.hasAttribute("hidden")) return true;
      const rect = el.getBoundingClientRect();
      if (!rect || rect.width <= 0 || rect.height <= 0) return true;
      return false;
    } catch {
      return false;
    }
  }

  function getRole(el) {
    const explicit = el.getAttribute && el.getAttribute("role");
    if (explicit) return explicit.toLowerCase();
    const tag = (el.tagName || "").toLowerCase();
    if (tag === "a" && el.getAttribute("href")) return "link";
    if (tag === "button") return "button";
    if (tag === "textarea") return "textbox";
    if (tag === "select") return "combobox";
    if (tag === "option") return "option";
    if (tag === "summary") return "button";
    if (tag === "input") {
      const t = (el.getAttribute("type") || "text").toLowerCase();
      if (t === "checkbox") return "checkbox";
      if (t === "radio") return "radio";
      if (t === "range") return "slider";
      if (t === "search") return "searchbox";
      if (t === "button" || t === "submit" || t === "reset") return "button";
      return "textbox";
    }
    if (el.isContentEditable) return "textbox";
    return "generic";
  }

  function getAriaLabelledBy(el) {
    const ids = (el.getAttribute("aria-labelledby") || "").trim();
    if (!ids) return "";
    const parts = ids.split(/\s+/).filter(Boolean);
    const texts = [];
    for (const id of parts) {
      const n = el.ownerDocument ? el.ownerDocument.getElementById(id) : null;
      if (n) {
        const t = norm(n.innerText || n.textContent || "");
        if (t) texts.push(t);
      }
    }
    return texts.join(" ").trim();
  }

  function getLabel(el) {
    const ariaLabel = norm(el.getAttribute && el.getAttribute("aria-label"));
    if (ariaLabel) return ariaLabel;

    const labelledBy = getAriaLabelledBy(el);
    if (labelledBy) return labelledBy;

    try {
      if (el.labels && el.labels.length) {
        const parts = [];
        for (const l of el.labels) {
          const t = norm(l.innerText || l.textContent || "");
          if (t) parts.push(t);
        }
        if (parts.length) return parts.join(" ").trim();
      }
    } catch {
      // ignore
    }

    const placeholder = norm(el.getAttribute && el.getAttribute("placeholder"));
    if (placeholder) return placeholder;

    const title = norm(el.getAttribute && el.getAttribute("title"));
    if (title) return title;

    const alt = norm(el.getAttribute && el.getAttribute("alt"));
    if (alt) return alt;

    const text = norm(el.innerText || el.textContent || "");
    return text;
  }

  function isInteractive(el, role) {
    if (!el || !role) return false;
    const tag = (el.tagName || "").toLowerCase();
    if (tag === "a" && el.getAttribute("href")) return true;
    if (tag === "button" || tag === "select" || tag === "textarea" || tag === "summary") return true;
    if (tag === "input") return true;
    if (el.isContentEditable) return true;
    if (INTERACTIVE_ROLE.has(role)) return true;
    return false;
  }

  function isHeading(el) {
    const tag = (el.tagName || "").toLowerCase();
    if (/^h[1-6]$/.test(tag)) return true;
    return (el.getAttribute && el.getAttribute("role") || "").toLowerCase() === "heading";
  }

  function ensureRef(el) {
    if (!globalThis.__devBrowserRefCounter) globalThis.__devBrowserRefCounter = 0;
    let ref = el.__devBrowserRef;
    if (!ref) {
      ref = "e" + String(++globalThis.__devBrowserRefCounter);
      el.__devBrowserRef = ref;
    }
    return ref;
  }

  function triStateAttr(value) {
    const v = (value || "").toLowerCase();
    if (v === "mixed") return "mixed";
    if (v === "true") return true;
    return null;
  }

  function getStates(el) {
    const ariaDisabled = (el.getAttribute && el.getAttribute("aria-disabled") || "").toLowerCase();
    const disabled = ariaDisabled === "true" || el.disabled === true || (el.hasAttribute && el.hasAttribute("disabled"));

    const ariaChecked = triStateAttr(el.getAttribute && el.getAttribute("aria-checked"));
    const ariaPressed = triStateAttr(el.getAttribute && el.getAttribute("aria-pressed"));

    let checked = ariaChecked;
    try {
      if (checked === null && (el instanceof HTMLInputElement) && (el.type === "checkbox" || el.type === "radio")) {
        checked = el.checked ? true : null;
      }
    } catch {
      // ignore
    }

    const expanded = ((el.getAttribute && el.getAttribute("aria-expanded")) || "").toLowerCase() === "true";
    const selected = ((el.getAttribute && el.getAttribute("aria-selected")) || "").toLowerCase() === "true";
    const pressed = ariaPressed;
    const active = !!(el.ownerDocument && el.ownerDocument.activeElement === el);

    return { disabled, checked, expanded, selected, pressed, active };
  }

  function walk(root, state, items, opts) {
    if (!root || items.length >= opts.maxItems) return;

    const children = root.children ? Array.from(root.children) : [];
    for (const el of children) {
      if (items.length >= opts.maxItems) return;
      const tag = (el.tagName || "").toLowerCase();
      if (tag === "script" || tag === "style" || tag === "noscript") continue;

      if (isHeading(el)) {
        const t = norm(el.innerText || el.textContent || "");
        if (t) state.heading = t;
      }

      const role = getRole(el);
      if ((!opts.interactiveOnly || isInteractive(el, role)) && !isHidden(el)) {
        const name = getLabel(el);
        const ref = ensureRef(el);
        const st = getStates(el);
        globalThis.__devBrowserRefs[ref] = el;
        items.push({
          ref,
          role,
          name: name || null,
          heading: state.heading || null,
          disabled: !!st.disabled,
          checked: st.checked,
          expanded: !!st.expanded,
          selected: !!st.selected,
          pressed: st.pressed,
          active: !!st.active
        });
      }

      if (el.shadowRoot) walk(el.shadowRoot, state, items, opts);
      walk(el, state, items, opts);
    }
  }

  function buildYaml(items, opts) {
    const lines = [];
    let currentHeading = null;
    for (const item of items) {
      if (item.heading && item.heading !== currentHeading) {
        lines.push(`- heading ${JSON.stringify(item.heading)}`);
        currentHeading = item.heading;
      }
      const indent = currentHeading ? "  " : "";
      const name = item.name ? ` name=${JSON.stringify(item.name)}` : "";
      let suffix = ` [ref=${item.ref}]`;
      if (item.disabled) suffix += " [disabled]";
      if (item.checked === "mixed") suffix += " [checked=mixed]";
      else if (item.checked === true) suffix += " [checked]";
      if (item.expanded) suffix += " [expanded]";
      if (item.selected) suffix += " [selected]";
      if (item.pressed === "mixed") suffix += " [pressed=mixed]";
      else if (item.pressed === true) suffix += " [pressed]";
      if (item.active) suffix += " [active]";
      if (item.cursorPointer) suffix += " [cursor=pointer]";
      lines.push(`${indent}- ${item.role}${name}${suffix}`);
    }
    if (opts.truncated) lines.push(`- [...] truncated (max_items=${opts.maxItems})`);
    let text = lines.join("\n");
    if (text.length > opts.maxChars) {
      text = text.slice(0, Math.max(0, opts.maxChars - 40)) + `\n- [...] truncated (max_chars=${opts.maxChars})`;
    }
    return text;
  }

  function simpleSnapshot(userOpts) {
    const opts = userOpts || {};
    const maxItems = typeof opts.maxItems === "number" && opts.maxItems > 0 ? opts.maxItems : 80;
    const maxChars = typeof opts.maxChars === "number" && opts.maxChars > 0 ? opts.maxChars : 8000;
    const interactiveOnly = opts.interactiveOnly !== false;

    globalThis.__devBrowserRefs = {};
    const items = [];
    const state = { heading: null };
    walk(document.documentElement, state, items, { maxItems, maxChars, interactiveOnly });
    const truncated = items.length >= maxItems;

    const yaml = buildYaml(items, { maxItems, maxChars, truncated });
    globalThis.__devBrowserLastSnapshot = { yaml, items };
    return globalThis.__devBrowserLastSnapshot;
  }

  function selectSnapshotRef(ref) {
    const refs = globalThis.__devBrowserRefs;
    if (!refs) throw new Error("No snapshot refs found. Call snapshot first.");
    const el = refs[ref];
    if (!el) throw new Error(`Ref "${ref}" not found. Call snapshot again.`);
    if (!el.isConnected) throw new Error(`Ref "${ref}" is stale (element detached). Call snapshot again.`);
    return el;
  }

  function clearRefOverlay() {
    const root = globalThis.__devBrowserRefOverlayRoot;
    if (root && root.parentNode) root.parentNode.removeChild(root);
    globalThis.__devBrowserRefOverlayRoot = null;
  }

  function drawRefOverlay(userOpts) {
    const opts = userOpts || {};
    const maxRefs = typeof opts.maxRefs === "number" && opts.maxRefs > 0 ? opts.maxRefs : 80;
    const snap = globalThis.__devBrowserLastSnapshot;
    if (!snap || !Array.isArray(snap.items)) throw new Error("No snapshot found. Call snapshot first.");

    clearRefOverlay();
    const root = document.createElement("div");
    root.style.position = "absolute";
    root.style.left = "0px";
    root.style.top = "0px";
    root.style.pointerEvents = "none";
    root.style.zIndex = "2147483647";
    root.style.fontFamily = "ui-monospace, Menlo, Monaco, Consolas, monospace";
    root.setAttribute("data-dev-browser-overlay", "refs");

    const scrollX = window.scrollX || 0;
    const scrollY = window.scrollY || 0;
    let count = 0;
    for (const item of snap.items) {
      if (!item || !item.ref) continue;
      if (count >= maxRefs) break;
      const el = globalThis.__devBrowserRefs ? globalThis.__devBrowserRefs[item.ref] : null;
      if (!el || !el.getBoundingClientRect) continue;
      const r = el.getBoundingClientRect();
      if (!r || r.width <= 0 || r.height <= 0) continue;
      const x = Math.max(0, r.left + scrollX);
      const y = Math.max(0, r.top + scrollY);
      const w = Math.max(1, r.width);
      const h = Math.max(1, r.height);

      const box = document.createElement("div");
      box.style.position = "absolute";
      box.style.left = x + "px";
      box.style.top = y + "px";
      box.style.width = w + "px";
      box.style.height = h + "px";
      box.style.border = "2px solid #ff3b30";
      box.style.boxSizing = "border-box";
      box.style.borderRadius = "2px";
      box.style.background = "rgba(255,59,48,0.04)";

      const label = document.createElement("div");
      label.textContent = String(item.ref);
      label.style.position = "absolute";
      label.style.left = "0px";
      label.style.top = "0px";
      label.style.padding = "1px 4px";
      label.style.fontSize = "12px";
      label.style.lineHeight = "14px";
      label.style.background = "#ff3b30";
      label.style.color = "#ffffff";
      label.style.borderBottomRightRadius = "2px";

      box.appendChild(label);
      root.appendChild(box);
      count++;
    }

    document.documentElement.appendChild(root);
    globalThis.__devBrowserRefOverlayRoot = root;
    return { ok: true, refs: count };
  }

  function getAISnapshot(userOpts) {
    const opts = userOpts || {};
    const engine = (opts.engine || "simple").toLowerCase();
    if (engine === "simple") return simpleSnapshot(opts);
    if (engine === "aria") {
      if (!globalThis.__devBrowser_getAISnapshotAria) throw new Error("ARIA snapshot engine not installed");
      const result = globalThis.__devBrowser_getAISnapshotAria(opts);
      globalThis.__devBrowserLastSnapshot = result;
      return result;
    }
    throw new Error(`Unknown snapshot engine: ${engine}`);
  }

  globalThis.__devBrowser_buildYaml = buildYaml;
  globalThis.__devBrowser_getAISnapshot = getAISnapshot;
  globalThis.__devBrowser_selectSnapshotRef = selectSnapshotRef;
  globalThis.__devBrowser_drawRefOverlay = drawRefOverlay;
  globalThis.__devBrowser_clearRefOverlay = clearRefOverlay;
})();
"""


def get_base_script() -> str:
    return _BASE_SCRIPT
