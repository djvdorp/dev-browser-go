// === domUtils ===
let cacheStyle;
let cachesCounter = 0;

function beginDOMCaches() {
  ++cachesCounter;
  cacheStyle = cacheStyle || new Map();
}

function endDOMCaches() {
  if (!--cachesCounter) {
    cacheStyle = undefined;
  }
}

function getElementComputedStyle(element, pseudo) {
  const cache = cacheStyle;
  const cacheKey = pseudo ? undefined : element;
  if (cache && cacheKey && cache.has(cacheKey)) return cache.get(cacheKey);
  const style = element.ownerDocument && element.ownerDocument.defaultView
    ? element.ownerDocument.defaultView.getComputedStyle(element, pseudo)
    : undefined;
  if (cache && cacheKey) cache.set(cacheKey, style);
  return style;
}

function parentElementOrShadowHost(element) {
  if (element.parentElement) return element.parentElement;
  if (!element.parentNode) return;
  if (element.parentNode.nodeType === 11 && element.parentNode.host)
    return element.parentNode.host;
}

function enclosingShadowRootOrDocument(element) {
  let node = element;
  while (node.parentNode) node = node.parentNode;
  if (node.nodeType === 11 || node.nodeType === 9)
    return node;
}

function closestCrossShadow(element, css, scope) {
  while (element) {
    const closest = element.closest(css);
    if (scope && closest !== scope && closest?.contains(scope)) return;
    if (closest) return closest;
    element = enclosingShadowHost(element);
  }
}

function enclosingShadowHost(element) {
  while (element.parentElement) element = element.parentElement;
  return parentElementOrShadowHost(element);
}

function isElementStyleVisibilityVisible(element, style) {
  style = style || getElementComputedStyle(element);
  if (!style) return true;
  if (style.visibility !== "visible") return false;
  const detailsOrSummary = element.closest("details,summary");
  if (detailsOrSummary !== element && detailsOrSummary?.nodeName === "DETAILS" && !detailsOrSummary.open)
    return false;
  return true;
}

function computeBox(element) {
  const style = getElementComputedStyle(element);
  if (!style) return { visible: true, inline: false };
  const cursor = style.cursor;
  if (style.display === "contents") {
    for (let child = element.firstChild; child; child = child.nextSibling) {
      if (child.nodeType === 1 && isElementVisible(child))
        return { visible: true, inline: false, cursor };
      if (child.nodeType === 3 && isVisibleTextNode(child))
        return { visible: true, inline: true, cursor };
    }
    return { visible: false, inline: false, cursor };
  }
  if (!isElementStyleVisibilityVisible(element, style))
    return { cursor, visible: false, inline: false };
  const rect = element.getBoundingClientRect();
  return { rect, cursor, visible: rect.width > 0 && rect.height > 0, inline: style.display === "inline" };
}

function isElementVisible(element) {
  return computeBox(element).visible;
}

function isVisibleTextNode(node) {
  const range = node.ownerDocument.createRange();
  range.selectNode(node);
  const rect = range.getBoundingClientRect();
  return rect.width > 0 && rect.height > 0;
}

function elementSafeTagName(element) {
  const tagName = element.tagName;
  if (typeof tagName === "string") return tagName.toUpperCase();
  if (element instanceof HTMLFormElement) return "FORM";
  return element.tagName.toUpperCase();
}

function normalizeWhiteSpace(text) {
  return text.split("\u00A0").map(chunk =>
    chunk.replace(/\r\n/g, "\n").replace(/[\u200b\u00ad]/g, "").replace(/\s\s*/g, " ")
  ).join("\u00A0").trim();
}
