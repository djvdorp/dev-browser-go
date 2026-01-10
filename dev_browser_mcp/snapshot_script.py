from __future__ import annotations

from pathlib import Path
from typing import Final

_VENDOR_DIR: Final[Path] = Path(__file__).with_name("vendor") / "dev_browser_snapshot"

_ARIA_SCRIPT_CACHE: str | None = None


def get_aria_script() -> str:
    global _ARIA_SCRIPT_CACHE
    if _ARIA_SCRIPT_CACHE is not None:
        return _ARIA_SCRIPT_CACHE

    vendor_parts = [
        "dom_utils.js",
        "yaml_utils.js",
        "role_utils.part1.js",
        "role_utils.part2.js",
        "aria_snapshot.js",
    ]
    vendor_code = "\n".join((_VENDOR_DIR / name).read_text(encoding="utf-8") for name in vendor_parts)

    _ARIA_SCRIPT_CACHE = (
        "(() => {\n"
        "  if (globalThis.__devBrowser_getAISnapshotAria) return;\n\n"
        f"{vendor_code}\n\n"
        "  function __devBrowser_ariaSnapshot(userOpts) {\n"
        "    const opts = userOpts || {};\n"
        "    const format = String(opts.format || \"list\").toLowerCase();\n"
        "    const maxItems = typeof opts.maxItems === \"number\" && opts.maxItems > 0 ? opts.maxItems : 80;\n"
        "    const maxChars = typeof opts.maxChars === \"number\" && opts.maxChars > 0 ? opts.maxChars : 8000;\n"
        "    const interactiveOnly = opts.interactiveOnly !== false;\n"
        "    const includeHeadings = opts.includeHeadings !== false;\n\n"
        "    // Vendor algorithm: generates an ARIA-like tree and stable eN refs.\n"
        "    const snap = generateAriaTree(document.body);\n"
        "    const nodesToWalk = snap.root && snap.root.role === \"fragment\" ? (snap.root.children || []) : [snap.root];\n\n"
        "    function truncate(text) {\n"
        "      if (typeof text !== \"string\") return \"\";\n"
        "      if (text.length <= maxChars) return text;\n"
        "      const suffix = `\\n- [...] truncated (max_chars=${maxChars})`;\n"
        "      return text.slice(0, Math.max(0, maxChars - suffix.length)) + suffix;\n"
        "    }\n\n"
        "    const refsObject = {};\n"
        "    try {\n"
        "      for (const [ref, element] of snap.elements) refsObject[ref] = element;\n"
        "    } catch {\n"
        "      // ignore\n"
        "    }\n"
        "    globalThis.__devBrowserRefs = refsObject;\n\n"
        "    const items = [];\n"
        "    let currentHeading = null;\n"
        "    const stack = [];\n"
        "    for (let i = nodesToWalk.length - 1; i >= 0; i--) stack.push(nodesToWalk[i]);\n"
        "    while (stack.length) {\n"
        "      const node = stack.pop();\n"
        "      if (!node || typeof node === \"string\") continue;\n\n"
        "      if (includeHeadings && node.role === \"heading\" && node.name) currentHeading = node.name;\n\n"
        "      if (!interactiveOnly || node.ref) {\n"
        "        if (node.ref) {\n"
        "          items.push({\n"
        "            ref: node.ref,\n"
        "            role: node.role,\n"
        "            name: node.name || null,\n"
        "            heading: currentHeading || null,\n"
        "            disabled: !!node.disabled,\n"
        "            checked: node.checked ?? null,\n"
        "            expanded: !!node.expanded,\n"
        "            selected: !!node.selected,\n"
        "            pressed: node.pressed ?? null,\n"
        "            active: !!node.active,\n"
        "            cursorPointer: !!(node.box && node.box.cursor === \"pointer\")\n"
        "          });\n"
        "          if (items.length >= maxItems) break;\n"
        "        }\n"
        "      }\n\n"
        "      const children = node.children || [];\n"
        "      for (let i = children.length - 1; i >= 0; i--) stack.push(children[i]);\n"
        "    }\n\n"
        "    const truncated = items.length >= maxItems;\n"
        "    const listYaml = globalThis.__devBrowser_buildYaml(items, { maxItems, maxChars, truncated });\n\n"
        "    if (format === \"list\") return { yaml: listYaml, items };\n"
        "    if (format === \"tree\") {\n"
        "      if (!interactiveOnly) return { yaml: truncate(renderAriaTree(snap)), items };\n\n"
        "      // Token-light tree: drop text nodes and empty non-ref leaves, keep ancestors of refs.\n"
        "      function prune(node) {\n"
        "        if (!node || typeof node === \"string\") return null;\n"
        "        const kids = node.children || [];\n"
        "        const next = [];\n"
        "        for (const child of kids) {\n"
        "          const pruned = prune(child);\n"
        "          if (pruned) next.push(pruned);\n"
        "        }\n"
        "        node.children = next;\n"
        "        const keep = !!node.ref || next.length > 0 || (includeHeadings && node.role === \"heading\" && node.name);\n"
        "        return keep ? node : null;\n"
      "      }\n"
        "      const prunedRoot = prune(snap.root);\n"
        "      const pruned = { ...snap, root: prunedRoot || { role: \"fragment\", name: \"\", children: [] } };\n"
        "      return { yaml: truncate(renderAriaTree(pruned)), items };\n"
        "    }\n"
        "    throw new Error(`Unknown snapshot format: ${format}`);\n"
        "  }\n\n"
        "  globalThis.__devBrowser_getAISnapshotAria = __devBrowser_ariaSnapshot;\n"
        "})();\n"
    )
    return _ARIA_SCRIPT_CACHE
