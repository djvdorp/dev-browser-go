// === ariaSnapshot ===
let lastRef = 0;

function generateAriaTree(rootElement) {
  const options = { visibility: "ariaOrVisible", refs: "interactable", refPrefix: "", includeGenericRole: true, renderActive: true, renderCursorPointer: true };
  const visited = new Set();
  const snapshot = {
    root: { role: "fragment", name: "", children: [], element: rootElement, props: {}, box: computeBox(rootElement), receivesPointerEvents: true },
    elements: new Map(),
    refs: new Map(),
    iframeRefs: []
  };

  const visit = (ariaNode, node, parentElementVisible) => {
    if (visited.has(node)) return;
    visited.add(node);
    if (node.nodeType === Node.TEXT_NODE && node.nodeValue) {
      if (!parentElementVisible) return;
      const text = node.nodeValue;
      if (ariaNode.role !== "textbox" && text) ariaNode.children.push(node.nodeValue || "");
      return;
    }
    if (node.nodeType !== Node.ELEMENT_NODE) return;
    const element = node;
    const isElementVisibleForAria = !isElementHiddenForAria(element);
    let visible = isElementVisibleForAria;
    if (options.visibility === "ariaOrVisible") visible = isElementVisibleForAria || isElementVisible(element);
    if (options.visibility === "ariaAndVisible") visible = isElementVisibleForAria && isElementVisible(element);
    if (options.visibility === "aria" && !visible) return;
    const ariaChildren = [];
    if (element.hasAttribute("aria-owns")) {
      const ids = element.getAttribute("aria-owns").split(/\s+/);
      for (const id of ids) {
        const ownedElement = rootElement.ownerDocument.getElementById(id);
        if (ownedElement) ariaChildren.push(ownedElement);
      }
    }
    const childAriaNode = visible ? toAriaNode(element, options) : null;
    if (childAriaNode) {
      if (childAriaNode.ref) {
        snapshot.elements.set(childAriaNode.ref, element);
        snapshot.refs.set(element, childAriaNode.ref);
        if (childAriaNode.role === "iframe") snapshot.iframeRefs.push(childAriaNode.ref);
      }
      ariaNode.children.push(childAriaNode);
    }
    processElement(childAriaNode || ariaNode, element, ariaChildren, visible);
  };

  function processElement(ariaNode, element, ariaChildren, parentElementVisible) {
    const display = getElementComputedStyle(element)?.display || "inline";
    const treatAsBlock = display !== "inline" || element.nodeName === "BR" ? " " : "";
    if (treatAsBlock) ariaNode.children.push(treatAsBlock);
    ariaNode.children.push(getCSSContent(element, "::before") || "");
    const assignedNodes = element.nodeName === "SLOT" ? element.assignedNodes() : [];
    if (assignedNodes.length) {
      for (const child of assignedNodes) visit(ariaNode, child, parentElementVisible);
    } else {
      for (let child = element.firstChild; child; child = child.nextSibling) {
        if (!child.assignedSlot) visit(ariaNode, child, parentElementVisible);
      }
      if (element.shadowRoot) {
        for (let child = element.shadowRoot.firstChild; child; child = child.nextSibling) visit(ariaNode, child, parentElementVisible);
      }
    }
    for (const child of ariaChildren) visit(ariaNode, child, parentElementVisible);
    ariaNode.children.push(getCSSContent(element, "::after") || "");
    if (treatAsBlock) ariaNode.children.push(treatAsBlock);
    if (ariaNode.children.length === 1 && ariaNode.name === ariaNode.children[0]) ariaNode.children = [];
    if (ariaNode.role === "link" && element.hasAttribute("href")) ariaNode.props["url"] = element.getAttribute("href");
    if (ariaNode.role === "textbox" && element.hasAttribute("placeholder") && element.getAttribute("placeholder") !== ariaNode.name) ariaNode.props["placeholder"] = element.getAttribute("placeholder");
  }

  beginAriaCaches();
  try { visit(snapshot.root, rootElement, true); }
  finally { endAriaCaches(); }
  normalizeStringChildren(snapshot.root);
  normalizeGenericRoles(snapshot.root);
  return snapshot;
}

function computeAriaRef(ariaNode, options) {
  if (options.refs === "none") return;
  if (options.refs === "interactable" && (!ariaNode.box.visible || !ariaNode.receivesPointerEvents)) return;
  let ariaRef = ariaNode.element._ariaRef;
  if (!ariaRef || ariaRef.role !== ariaNode.role || ariaRef.name !== ariaNode.name) {
    ariaRef = { role: ariaNode.role, name: ariaNode.name, ref: (options.refPrefix || "") + "e" + (++lastRef) };
    ariaNode.element._ariaRef = ariaRef;
  }
  ariaNode.ref = ariaRef.ref;
}

function toAriaNode(element, options) {
  const active = element.ownerDocument.activeElement === element;
  if (element.nodeName === "IFRAME") {
    const ariaNode = { role: "iframe", name: "", children: [], props: {}, element, box: computeBox(element), receivesPointerEvents: true, active };
    computeAriaRef(ariaNode, options);
    return ariaNode;
  }
  const defaultRole = options.includeGenericRole ? "generic" : null;
  const role = getAriaRole(element) || defaultRole;
  if (!role || role === "presentation" || role === "none") return null;
  const name = normalizeWhiteSpace(getElementAccessibleName(element, false) || "");
  const receivesPointerEventsValue = receivesPointerEvents(element);
  const box = computeBox(element);
  if (role === "generic" && box.inline && element.childNodes.length === 1 && element.childNodes[0].nodeType === Node.TEXT_NODE) return null;
  const result = { role, name, children: [], props: {}, element, box, receivesPointerEvents: receivesPointerEventsValue, active };
  computeAriaRef(result, options);
  if (kAriaCheckedRoles.includes(role)) result.checked = getAriaChecked(element);
  if (kAriaDisabledRoles.includes(role)) result.disabled = getAriaDisabled(element);
  if (kAriaExpandedRoles.includes(role)) result.expanded = getAriaExpanded(element);
  if (kAriaLevelRoles.includes(role)) result.level = getAriaLevel(element);
  if (kAriaPressedRoles.includes(role)) result.pressed = getAriaPressed(element);
  if (kAriaSelectedRoles.includes(role)) result.selected = getAriaSelected(element);
  if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement) {
    if (element.type !== "checkbox" && element.type !== "radio" && element.type !== "file") result.children = [element.value];
  }
  return result;
}

function normalizeGenericRoles(node) {
  const normalizeChildren = (node) => {
    const result = [];
    for (const child of node.children || []) {
      if (typeof child === "string") { result.push(child); continue; }
      const normalized = normalizeChildren(child);
      result.push(...normalized);
    }
    const removeSelf = node.role === "generic" && !node.name && result.length <= 1 && result.every(c => typeof c !== "string" && !!c.ref);
    if (removeSelf) return result;
    node.children = result;
    return [node];
  };
  normalizeChildren(node);
}

function normalizeStringChildren(rootA11yNode) {
  const flushChildren = (buffer, normalizedChildren) => {
    if (!buffer.length) return;
    const text = normalizeWhiteSpace(buffer.join(""));
    if (text) normalizedChildren.push(text);
    buffer.length = 0;
  };
  const visit = (ariaNode) => {
    const normalizedChildren = [];
    const buffer = [];
    for (const child of ariaNode.children || []) {
      if (typeof child === "string") { buffer.push(child); }
      else { flushChildren(buffer, normalizedChildren); visit(child); normalizedChildren.push(child); }
    }
    flushChildren(buffer, normalizedChildren);
    ariaNode.children = normalizedChildren.length ? normalizedChildren : [];
    if (ariaNode.children.length === 1 && ariaNode.children[0] === ariaNode.name) ariaNode.children = [];
  };
  visit(rootA11yNode);
}

function hasPointerCursor(ariaNode) { return ariaNode.box.cursor === "pointer"; }

function renderAriaTree(ariaSnapshot) {
  const options = { visibility: "ariaOrVisible", refs: "interactable", refPrefix: "", includeGenericRole: true, renderActive: true, renderCursorPointer: true };
  const lines = [];
  let nodesToRender = ariaSnapshot.root.role === "fragment" ? ariaSnapshot.root.children : [ariaSnapshot.root];

  const visitText = (text, indent) => {
    const escaped = yamlEscapeValueIfNeeded(text);
    if (escaped) lines.push(indent + "- text: " + escaped);
  };

  const createKey = (ariaNode, renderCursorPointer) => {
    let key = ariaNode.role;
    if (ariaNode.name && ariaNode.name.length <= 900) {
      const name = ariaNode.name;
      if (name) {
        const stringifiedName = name.startsWith("/") && name.endsWith("/") ? name : JSON.stringify(name);
        key += " " + stringifiedName;
      }
    }
    if (ariaNode.checked === "mixed") key += " [checked=mixed]";
    if (ariaNode.checked === true) key += " [checked]";
    if (ariaNode.disabled) key += " [disabled]";
    if (ariaNode.expanded) key += " [expanded]";
    if (ariaNode.active && options.renderActive) key += " [active]";
    if (ariaNode.level) key += " [level=" + ariaNode.level + "]";
    if (ariaNode.pressed === "mixed") key += " [pressed=mixed]";
    if (ariaNode.pressed === true) key += " [pressed]";
    if (ariaNode.selected === true) key += " [selected]";
    if (ariaNode.ref) {
      key += " [ref=" + ariaNode.ref + "]";
      if (renderCursorPointer && hasPointerCursor(ariaNode)) key += " [cursor=pointer]";
    }
    return key;
  };

  const getSingleInlinedTextChild = (ariaNode) => {
    return ariaNode?.children.length === 1 && typeof ariaNode.children[0] === "string" && !Object.keys(ariaNode.props).length ? ariaNode.children[0] : undefined;
  };

  const visit = (ariaNode, indent, renderCursorPointer) => {
    const escapedKey = indent + "- " + yamlEscapeKeyIfNeeded(createKey(ariaNode, renderCursorPointer));
    const singleInlinedTextChild = getSingleInlinedTextChild(ariaNode);
    if (!ariaNode.children.length && !Object.keys(ariaNode.props).length) {
      lines.push(escapedKey);
    } else if (singleInlinedTextChild !== undefined) {
      lines.push(escapedKey + ": " + yamlEscapeValueIfNeeded(singleInlinedTextChild));
    } else {
      lines.push(escapedKey + ":");
      for (const [name, value] of Object.entries(ariaNode.props)) lines.push(indent + "  - /" + name + ": " + yamlEscapeValueIfNeeded(value));
      const childIndent = indent + "  ";
      const inCursorPointer = !!ariaNode.ref && renderCursorPointer && hasPointerCursor(ariaNode);
      for (const child of ariaNode.children) {
        if (typeof child === "string") visitText(child, childIndent);
        else visit(child, childIndent, renderCursorPointer && !inCursorPointer);
      }
    }
  };

  for (const nodeToRender of nodesToRender) {
    if (typeof nodeToRender === "string") visitText(nodeToRender, "");
    else visit(nodeToRender, "", !!options.renderCursorPointer);
  }
  return lines.join("\n");
}

function getAISnapshot() {
  const snapshot = generateAriaTree(document.body);
  const refsObject = {};
  for (const [ref, element] of snapshot.elements) refsObject[ref] = element;
  window.__devBrowserRefs = refsObject;
  return renderAriaTree(snapshot);
}

function selectSnapshotRef(ref) {
  const refs = window.__devBrowserRefs;
  if (!refs) throw new Error("No snapshot refs found. Call getAISnapshot first.");
  const element = refs[ref];
  if (!element) throw new Error('Ref "' + ref + '" not found. Available refs: ' + Object.keys(refs).join(", "));
  return element;
}
