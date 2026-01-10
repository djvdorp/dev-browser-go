// === roleUtils ===
const validRoles = ["alert","alertdialog","application","article","banner","blockquote","button","caption","cell","checkbox","code","columnheader","combobox","complementary","contentinfo","definition","deletion","dialog","directory","document","emphasis","feed","figure","form","generic","grid","gridcell","group","heading","img","insertion","link","list","listbox","listitem","log","main","mark","marquee","math","meter","menu","menubar","menuitem","menuitemcheckbox","menuitemradio","navigation","none","note","option","paragraph","presentation","progressbar","radio","radiogroup","region","row","rowgroup","rowheader","scrollbar","search","searchbox","separator","slider","spinbutton","status","strong","subscript","superscript","switch","tab","table","tablist","tabpanel","term","textbox","time","timer","toolbar","tooltip","tree","treegrid","treeitem"];

let cacheAccessibleName;
let cacheIsHidden;
let cachePointerEvents;
let ariaCachesCounter = 0;

function beginAriaCaches() {
  beginDOMCaches();
  ++ariaCachesCounter;
  cacheAccessibleName = cacheAccessibleName || new Map();
  cacheIsHidden = cacheIsHidden || new Map();
  cachePointerEvents = cachePointerEvents || new Map();
}

function endAriaCaches() {
  if (!--ariaCachesCounter) {
    cacheAccessibleName = undefined;
    cacheIsHidden = undefined;
    cachePointerEvents = undefined;
  }
  endDOMCaches();
}

function hasExplicitAccessibleName(e) {
  return e.hasAttribute("aria-label") || e.hasAttribute("aria-labelledby");
}

const kAncestorPreventingLandmark = "article:not([role]), aside:not([role]), main:not([role]), nav:not([role]), section:not([role]), [role=article], [role=complementary], [role=main], [role=navigation], [role=region]";

const kGlobalAriaAttributes = [
  ["aria-atomic", undefined],["aria-busy", undefined],["aria-controls", undefined],["aria-current", undefined],
  ["aria-describedby", undefined],["aria-details", undefined],["aria-dropeffect", undefined],["aria-flowto", undefined],
  ["aria-grabbed", undefined],["aria-hidden", undefined],["aria-keyshortcuts", undefined],
  ["aria-label", ["caption","code","deletion","emphasis","generic","insertion","paragraph","presentation","strong","subscript","superscript"]],
  ["aria-labelledby", ["caption","code","deletion","emphasis","generic","insertion","paragraph","presentation","strong","subscript","superscript"]],
  ["aria-live", undefined],["aria-owns", undefined],["aria-relevant", undefined],["aria-roledescription", ["generic"]]
];

function hasGlobalAriaAttribute(element, forRole) {
  return kGlobalAriaAttributes.some(([attr, prohibited]) => !prohibited?.includes(forRole || "") && element.hasAttribute(attr));
}

function hasTabIndex(element) {
  return !Number.isNaN(Number(String(element.getAttribute("tabindex"))));
}

function isFocusable(element) {
  return !isNativelyDisabled(element) && (isNativelyFocusable(element) || hasTabIndex(element));
}

function isNativelyFocusable(element) {
  const tagName = elementSafeTagName(element);
  if (["BUTTON","DETAILS","SELECT","TEXTAREA"].includes(tagName)) return true;
  if (tagName === "A" || tagName === "AREA") return element.hasAttribute("href");
  if (tagName === "INPUT") return !element.hidden;
  return false;
}

function isNativelyDisabled(element) {
  const isNativeFormControl = ["BUTTON","INPUT","SELECT","TEXTAREA","OPTION","OPTGROUP"].includes(elementSafeTagName(element));
  return isNativeFormControl && (element.hasAttribute("disabled") || belongsToDisabledFieldSet(element));
}

function belongsToDisabledFieldSet(element) {
  const fieldSetElement = element?.closest("FIELDSET[DISABLED]");
  if (!fieldSetElement) return false;
  const legendElement = fieldSetElement.querySelector(":scope > LEGEND");
  return !legendElement || !legendElement.contains(element);
}

const inputTypeToRole = {button:"button",checkbox:"checkbox",image:"button",number:"spinbutton",radio:"radio",range:"slider",reset:"button",submit:"button"};

function getIdRefs(element, ref) {
  if (!ref) return [];
  const root = enclosingShadowRootOrDocument(element);
  if (!root) return [];
  try {
    const ids = ref.split(" ").filter(id => !!id);
    const result = [];
    for (const id of ids) {
      const firstElement = root.querySelector("#" + CSS.escape(id));
      if (firstElement && !result.includes(firstElement)) result.push(firstElement);
    }
    return result;
  } catch { return []; }
}

const kImplicitRoleByTagName = {
  A: e => e.hasAttribute("href") ? "link" : null,
  AREA: e => e.hasAttribute("href") ? "link" : null,
  ARTICLE: () => "article", ASIDE: () => "complementary", BLOCKQUOTE: () => "blockquote", BUTTON: () => "button",
  CAPTION: () => "caption", CODE: () => "code", DATALIST: () => "listbox", DD: () => "definition",
  DEL: () => "deletion", DETAILS: () => "group", DFN: () => "term", DIALOG: () => "dialog", DT: () => "term",
  EM: () => "emphasis", FIELDSET: () => "group", FIGURE: () => "figure",
  FOOTER: e => closestCrossShadow(e, kAncestorPreventingLandmark) ? null : "contentinfo",
  FORM: e => hasExplicitAccessibleName(e) ? "form" : null,
  H1: () => "heading", H2: () => "heading", H3: () => "heading", H4: () => "heading", H5: () => "heading", H6: () => "heading",
  HEADER: e => closestCrossShadow(e, kAncestorPreventingLandmark) ? null : "banner",
  HR: () => "separator", HTML: () => "document",
  IMG: e => e.getAttribute("alt") === "" && !e.getAttribute("title") && !hasGlobalAriaAttribute(e) && !hasTabIndex(e) ? "presentation" : "img",
  INPUT: e => {
    const type = e.type.toLowerCase();
    if (type === "search") return e.hasAttribute("list") ? "combobox" : "searchbox";
    if (["email","tel","text","url",""].includes(type)) {
      const list = getIdRefs(e, e.getAttribute("list"))[0];
      return list && elementSafeTagName(list) === "DATALIST" ? "combobox" : "textbox";
    }
    if (type === "hidden") return null;
    if (type === "file") return "button";
    return inputTypeToRole[type] || "textbox";
  },
  INS: () => "insertion", LI: () => "listitem", MAIN: () => "main", MARK: () => "mark", MATH: () => "math",
  MENU: () => "list", METER: () => "meter", NAV: () => "navigation", OL: () => "list", OPTGROUP: () => "group",
  OPTION: () => "option", OUTPUT: () => "status", P: () => "paragraph", PROGRESS: () => "progressbar",
  SEARCH: () => "search", SECTION: e => hasExplicitAccessibleName(e) ? "region" : null,
  SELECT: e => e.hasAttribute("multiple") || e.size > 1 ? "listbox" : "combobox",
  STRONG: () => "strong", SUB: () => "subscript", SUP: () => "superscript", SVG: () => "img",
  TABLE: () => "table", TBODY: () => "rowgroup",
  TD: e => { const table = closestCrossShadow(e, "table"); const role = table ? getExplicitAriaRole(table) : ""; return role === "grid" || role === "treegrid" ? "gridcell" : "cell"; },
  TEXTAREA: () => "textbox", TFOOT: () => "rowgroup",
  TH: e => { const scope = e.getAttribute("scope"); if (scope === "col" || scope === "colgroup") return "columnheader"; if (scope === "row" || scope === "rowgroup") return "rowheader"; return "columnheader"; },
  THEAD: () => "rowgroup", TIME: () => "time", TR: () => "row", UL: () => "list"
};

function getExplicitAriaRole(element) {
  const roles = (element.getAttribute("role") || "").split(" ").map(role => role.trim());
  return roles.find(role => validRoles.includes(role)) || null;
}

function getImplicitAriaRole(element) {
  const fn = kImplicitRoleByTagName[elementSafeTagName(element)];
  return fn ? fn(element) : null;
}

function hasPresentationConflictResolution(element, role) {
  return hasGlobalAriaAttribute(element, role) || isFocusable(element);
}

function getAriaRole(element) {
  const explicitRole = getExplicitAriaRole(element);
  if (!explicitRole) return getImplicitAriaRole(element);
  if (explicitRole === "none" || explicitRole === "presentation") {
    const implicitRole = getImplicitAriaRole(element);
    if (hasPresentationConflictResolution(element, implicitRole)) return implicitRole;
  }
  return explicitRole;
}

function getAriaBoolean(attr) {
  return attr === null ? undefined : attr.toLowerCase() === "true";
}

function isElementIgnoredForAria(element) {
  return ["STYLE","SCRIPT","NOSCRIPT","TEMPLATE"].includes(elementSafeTagName(element));
}

function isElementHiddenForAria(element) {
  if (isElementIgnoredForAria(element)) return true;
  const style = getElementComputedStyle(element);
  const isSlot = element.nodeName === "SLOT";
  if (style?.display === "contents" && !isSlot) {
    for (let child = element.firstChild; child; child = child.nextSibling) {
      if (child.nodeType === 1 && !isElementHiddenForAria(child)) return false;
      if (child.nodeType === 3 && isVisibleTextNode(child)) return false;
    }
    return true;
  }
  const isOptionInsideSelect = element.nodeName === "OPTION" && !!element.closest("select");
  if (!isOptionInsideSelect && !isSlot && !isElementStyleVisibilityVisible(element, style)) return true;
  return belongsToDisplayNoneOrAriaHiddenOrNonSlotted(element);
}

function belongsToDisplayNoneOrAriaHiddenOrNonSlotted(element) {
  let hidden = cacheIsHidden?.get(element);
  if (hidden === undefined) {
    hidden = false;
    if (element.parentElement && element.parentElement.shadowRoot && !element.assignedSlot) hidden = true;
    if (!hidden) {
      const style = getElementComputedStyle(element);
      hidden = !style || style.display === "none" || getAriaBoolean(element.getAttribute("aria-hidden")) === true;
    }
    if (!hidden) {
      const parent = parentElementOrShadowHost(element);
      if (parent) hidden = belongsToDisplayNoneOrAriaHiddenOrNonSlotted(parent);
    }
    cacheIsHidden?.set(element, hidden);
  }
  return hidden;
}

function getAriaLabelledByElements(element) {
  const ref = element.getAttribute("aria-labelledby");
  if (ref === null) return null;
  const refs = getIdRefs(element, ref);
  return refs.length ? refs : null;
}

function getElementAccessibleName(element, includeHidden) {
  let accessibleName = cacheAccessibleName?.get(element);
  if (accessibleName === undefined) {
    accessibleName = "";
    const elementProhibitsNaming = ["caption","code","definition","deletion","emphasis","generic","insertion","mark","paragraph","presentation","strong","subscript","suggestion","superscript","term","time"].includes(getAriaRole(element) || "");
    if (!elementProhibitsNaming) {
      accessibleName = normalizeWhiteSpace(getTextAlternativeInternal(element, { includeHidden, visitedElements: new Set(), embeddedInTargetElement: "self" }));
    }
    cacheAccessibleName?.set(element, accessibleName);
  }
  return accessibleName;
}

function getTextAlternativeInternal(element, options) {
  if (options.visitedElements.has(element)) return "";
  const childOptions = { ...options, embeddedInTargetElement: options.embeddedInTargetElement === "self" ? "descendant" : options.embeddedInTargetElement };

  if (!options.includeHidden) {
    const isEmbeddedInHiddenReferenceTraversal = !!options.embeddedInLabelledBy?.hidden || !!options.embeddedInLabel?.hidden;
    if (isElementIgnoredForAria(element) || (!isEmbeddedInHiddenReferenceTraversal && isElementHiddenForAria(element))) {
      options.visitedElements.add(element);
      return "";
    }
  }

  const labelledBy = getAriaLabelledByElements(element);
  if (!options.embeddedInLabelledBy) {
    const accessibleName = (labelledBy || []).map(ref => getTextAlternativeInternal(ref, { ...options, embeddedInLabelledBy: { element: ref, hidden: isElementHiddenForAria(ref) }, embeddedInTargetElement: undefined, embeddedInLabel: undefined })).join(" ");
    if (accessibleName) return accessibleName;
  }

  const role = getAriaRole(element) || "";
  const tagName = elementSafeTagName(element);

  const ariaLabel = element.getAttribute("aria-label") || "";
  if (ariaLabel.trim()) { options.visitedElements.add(element); return ariaLabel; }

  if (!["presentation","none"].includes(role)) {
    if (tagName === "INPUT" && ["button","submit","reset"].includes(element.type)) {
      options.visitedElements.add(element);
      const value = element.value || "";
