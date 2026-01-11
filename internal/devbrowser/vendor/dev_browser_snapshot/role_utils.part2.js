      if (value.trim()) return value;
      if (element.type === "submit") return "Submit";
      if (element.type === "reset") return "Reset";
      return element.getAttribute("title") || "";
    }
    if (tagName === "INPUT" && element.type === "image") {
      options.visitedElements.add(element);
      const alt = element.getAttribute("alt") || "";
      if (alt.trim()) return alt;
      const title = element.getAttribute("title") || "";
      if (title.trim()) return title;
      return "Submit";
    }
    if (tagName === "IMG") {
      options.visitedElements.add(element);
      const alt = element.getAttribute("alt") || "";
      if (alt.trim()) return alt;
      return element.getAttribute("title") || "";
    }
    if (!labelledBy && ["BUTTON","INPUT","TEXTAREA","SELECT"].includes(tagName)) {
      const labels = element.labels;
      if (labels?.length) {
        options.visitedElements.add(element);
        return [...labels].map(label => getTextAlternativeInternal(label, { ...options, embeddedInLabel: { element: label, hidden: isElementHiddenForAria(label) }, embeddedInLabelledBy: undefined, embeddedInTargetElement: undefined })).filter(name => !!name).join(" ");
      }
    }
  }

  const allowsNameFromContent = ["button","cell","checkbox","columnheader","gridcell","heading","link","menuitem","menuitemcheckbox","menuitemradio","option","radio","row","rowheader","switch","tab","tooltip","treeitem"].includes(role);
  if (allowsNameFromContent || !!options.embeddedInLabelledBy || !!options.embeddedInLabel) {
    options.visitedElements.add(element);
    const accessibleName = innerAccumulatedElementText(element, childOptions);
    const maybeTrimmedAccessibleName = options.embeddedInTargetElement === "self" ? accessibleName.trim() : accessibleName;
    if (maybeTrimmedAccessibleName) return accessibleName;
  }

  if (!["presentation","none"].includes(role) || tagName === "IFRAME") {
    options.visitedElements.add(element);
    const title = element.getAttribute("title") || "";
    if (title.trim()) return title;
  }

  options.visitedElements.add(element);
  return "";
}

function innerAccumulatedElementText(element, options) {
  const tokens = [];
  const visit = (node, skipSlotted) => {
    if (skipSlotted && node.assignedSlot) return;
    if (node.nodeType === 1) {
      const display = getElementComputedStyle(node)?.display || "inline";
      let token = getTextAlternativeInternal(node, options);
      if (display !== "inline" || node.nodeName === "BR") token = " " + token + " ";
      tokens.push(token);
    } else if (node.nodeType === 3) {
      tokens.push(node.textContent || "");
    }
  };
  const assignedNodes = element.nodeName === "SLOT" ? element.assignedNodes() : [];
  if (assignedNodes.length) {
    for (const child of assignedNodes) visit(child, false);
  } else {
    for (let child = element.firstChild; child; child = child.nextSibling) visit(child, true);
    if (element.shadowRoot) {
      for (let child = element.shadowRoot.firstChild; child; child = child.nextSibling) visit(child, true);
    }
  }
  return tokens.join("");
}

const kAriaCheckedRoles = ["checkbox","menuitemcheckbox","option","radio","switch","menuitemradio","treeitem"];
function getAriaChecked(element) {
  const tagName = elementSafeTagName(element);
  if (tagName === "INPUT" && element.indeterminate) return "mixed";
  if (tagName === "INPUT" && ["checkbox","radio"].includes(element.type)) return element.checked;
  if (kAriaCheckedRoles.includes(getAriaRole(element) || "")) {
    const checked = element.getAttribute("aria-checked");
    if (checked === "true") return true;
    if (checked === "mixed") return "mixed";
    return false;
  }
  return false;
}

const kAriaDisabledRoles = ["application","button","composite","gridcell","group","input","link","menuitem","scrollbar","separator","tab","checkbox","columnheader","combobox","grid","listbox","menu","menubar","menuitemcheckbox","menuitemradio","option","radio","radiogroup","row","rowheader","searchbox","select","slider","spinbutton","switch","tablist","textbox","toolbar","tree","treegrid","treeitem"];
function getAriaDisabled(element) {
  return isNativelyDisabled(element) || hasExplicitAriaDisabled(element);
}
function hasExplicitAriaDisabled(element, isAncestor) {
  if (!element) return false;
  if (isAncestor || kAriaDisabledRoles.includes(getAriaRole(element) || "")) {
    const attribute = (element.getAttribute("aria-disabled") || "").toLowerCase();
    if (attribute === "true") return true;
    if (attribute === "false") return false;
    return hasExplicitAriaDisabled(parentElementOrShadowHost(element), true);
  }
  return false;
}

const kAriaExpandedRoles = ["application","button","checkbox","combobox","gridcell","link","listbox","menuitem","row","rowheader","tab","treeitem","columnheader","menuitemcheckbox","menuitemradio","switch"];
function getAriaExpanded(element) {
  if (elementSafeTagName(element) === "DETAILS") return element.open;
  if (kAriaExpandedRoles.includes(getAriaRole(element) || "")) {
    const expanded = element.getAttribute("aria-expanded");
    if (expanded === null) return undefined;
    if (expanded === "true") return true;
    return false;
  }
  return undefined;
}

const kAriaLevelRoles = ["heading","listitem","row","treeitem"];
function getAriaLevel(element) {
  const native = {H1:1,H2:2,H3:3,H4:4,H5:5,H6:6}[elementSafeTagName(element)];
  if (native) return native;
  if (kAriaLevelRoles.includes(getAriaRole(element) || "")) {
    const attr = element.getAttribute("aria-level");
    const value = attr === null ? Number.NaN : Number(attr);
    if (Number.isInteger(value) && value >= 1) return value;
  }
  return 0;
}

const kAriaPressedRoles = ["button"];
function getAriaPressed(element) {
  if (kAriaPressedRoles.includes(getAriaRole(element) || "")) {
    const pressed = element.getAttribute("aria-pressed");
    if (pressed === "true") return true;
    if (pressed === "mixed") return "mixed";
  }
  return false;
}

const kAriaSelectedRoles = ["gridcell","option","row","tab","rowheader","columnheader","treeitem"];
function getAriaSelected(element) {
  if (elementSafeTagName(element) === "OPTION") return element.selected;
  if (kAriaSelectedRoles.includes(getAriaRole(element) || "")) return getAriaBoolean(element.getAttribute("aria-selected")) === true;
  return false;
}

function receivesPointerEvents(element) {
  const cache = cachePointerEvents;
  let e = element;
  let result;
  const parents = [];
  for (; e; e = parentElementOrShadowHost(e)) {
    const cached = cache?.get(e);
    if (cached !== undefined) { result = cached; break; }
    parents.push(e);
    const style = getElementComputedStyle(e);
    if (!style) { result = true; break; }
    const value = style.pointerEvents;
    if (value) { result = value !== "none"; break; }
  }
  if (result === undefined) result = true;
  for (const parent of parents) cache?.set(parent, result);
  return result;
}

function getCSSContent(element, pseudo) {
  const style = getElementComputedStyle(element, pseudo);
  if (!style) return undefined;
  const contentValue = style.content;
  if (!contentValue || contentValue === "none" || contentValue === "normal") return undefined;
  if (style.display === "none" || style.visibility === "hidden") return undefined;
  const match = contentValue.match(/^"(.*)"$/);
  if (match) {
    const content = match[1].replace(/\\"/g, '"');
    if (pseudo) {
      const display = style.display || "inline";
      if (display !== "inline") return " " + content + " ";
    }
    return content;
  }
  return undefined;
}
