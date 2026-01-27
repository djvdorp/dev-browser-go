(() => {
  if (globalThis.__devBrowser_styleCapture) return;

  const DEFAULT_PROPERTIES = [
    "display",
    "position",
    "top",
    "right",
    "bottom",
    "left",
    "width",
    "height",
    "min-width",
    "max-width",
    "min-height",
    "max-height",
    "margin",
    "margin-top",
    "margin-right",
    "margin-bottom",
    "margin-left",
    "padding",
    "padding-top",
    "padding-right",
    "padding-bottom",
    "padding-left",
    "box-sizing",
    "border",
    "border-top",
    "border-right",
    "border-bottom",
    "border-left",
    "border-radius",
    "background",
    "background-color",
    "background-image",
    "background-position",
    "background-repeat",
    "background-size",
    "color",
    "font-family",
    "font-size",
    "font-weight",
    "font-style",
    "line-height",
    "letter-spacing",
    "text-align",
    "text-transform",
    "text-decoration",
    "white-space",
    "overflow",
    "overflow-x",
    "overflow-y",
    "opacity",
    "box-shadow",
    "transform",
    "transform-origin",
    "z-index",
    "flex",
    "flex-direction",
    "flex-wrap",
    "justify-content",
    "align-items",
    "align-content",
    "gap",
    "row-gap",
    "column-gap",
    "grid",
    "grid-template-columns",
    "grid-template-rows",
    "grid-area",
    "cursor",
  ];

  function normalizeProperties(input) {
    if (Array.isArray(input)) {
      const cleaned = input
        .map((p) => String(p || "").trim())
        .filter(Boolean);
      if (cleaned.length) return cleaned;
    }
    if (typeof input === "string") {
      const cleaned = input
        .split(",")
        .map((p) => p.trim())
        .filter(Boolean);
      if (cleaned.length) return cleaned;
    }
    return DEFAULT_PROPERTIES.slice();
  }

  function buildStyleText(style, properties, includeAll) {
    if (!style) return "";
    const parts = [];
    if (includeAll) {
      for (let i = 0; i < style.length; i++) {
        const prop = style[i];
        const value = style.getPropertyValue(prop);
        if (!value) continue;
        parts.push(`${prop}:${value.trim()};`);
      }
      return parts.join("");
    }
    for (const prop of properties) {
      const value = style.getPropertyValue(prop);
      if (!value) continue;
      parts.push(`${prop}:${value.trim()};`);
    }
    return parts.join("");
  }

  function styleCapture(opts) {
    const options = opts || {};
    const selector = String(options.selector || "");
    const maxNodes =
      typeof options.maxNodes === "number" && options.maxNodes > 0
        ? options.maxNodes
        : 1500;
    const includeAll = options.includeAll === true;
    const mode = String(options.mode || "inline").toLowerCase();
    if (mode !== "inline" && mode !== "bundle") {
      throw new Error(`Unknown style capture mode: ${mode}`);
    }
    const strip = options.strip !== false;
    const properties = normalizeProperties(options.properties);

    const root = selector
      ? document.querySelector(selector)
      : document.documentElement;
    if (!root) {
      throw new Error(`Selector not found: ${selector}`);
    }

    const cloneRoot = root.cloneNode(true);
    const originalNodes = [root, ...root.querySelectorAll("*")];
    const cloneNodes = [cloneRoot, ...cloneRoot.querySelectorAll("*")];
    const total = Math.min(originalNodes.length, cloneNodes.length);
    const limit = Math.min(total, maxNodes);
    const cssParts = [];

    for (let i = 0; i < limit; i++) {
      const original = originalNodes[i];
      const clone = cloneNodes[i];
      if (!(original instanceof Element) || !(clone instanceof Element)) {
        continue;
      }
      const style = getComputedStyle(original);
      const styleText = buildStyleText(style, properties, includeAll);
      if (!styleText) continue;
      if (mode === "bundle") {
        const id = String(i + 1);
        clone.setAttribute("data-devbrowser-style", id);
        cssParts.push(
          `[data-devbrowser-style="${id}"]{${styleText}}`
        );
      } else {
        clone.setAttribute("style", styleText);
      }
    }

    if (strip) {
      const stripNodes = cloneRoot.querySelectorAll(
        "script, style, link[rel='stylesheet']"
      );
      for (const node of stripNodes) {
        node.remove();
      }
    }

    const truncated = total > limit;
    const cssText = mode === "bundle" ? cssParts.join("\n") : "";

    let html = "";
    const isHtmlRoot =
      cloneRoot.tagName && cloneRoot.tagName.toLowerCase() === "html";
    if (isHtmlRoot) {
      if (mode === "bundle" && cssText) {
        const head = cloneRoot.querySelector("head");
        if (head) {
          const styleEl = cloneRoot.ownerDocument.createElement("style");
          styleEl.setAttribute("data-devbrowser", "computed");
          styleEl.textContent = cssText;
          head.appendChild(styleEl);
        }
      }
      html = "<!doctype html>\n" + cloneRoot.outerHTML;
    } else {
      const doc = document.implementation.createHTMLDocument(
        document.title || ""
      );
      if (mode === "bundle" && cssText) {
        const styleEl = doc.createElement("style");
        styleEl.setAttribute("data-devbrowser", "computed");
        styleEl.textContent = cssText;
        doc.head.appendChild(styleEl);
      }
      doc.body.appendChild(doc.importNode(cloneRoot, true));
      html = "<!doctype html>\n" + doc.documentElement.outerHTML;
    }

    return {
      html,
      css: cssText,
      nodeCount: limit,
      truncated,
      mode,
      selector: selector || null,
      properties,
    };
  }

  globalThis.__devBrowser_styleCapture = styleCapture;
})();
