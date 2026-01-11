package devbrowser

import (
	_ "embed"
	"strings"
)

//go:embed snapshot_assets/base_snapshot.js
var baseSnapshotJS string

//go:embed vendor/dev_browser_snapshot/dom_utils.js
var ariaDomUtils string

//go:embed vendor/dev_browser_snapshot/yaml_utils.js
var ariaYamlUtils string

//go:embed vendor/dev_browser_snapshot/role_utils.part1.js
var ariaRoleUtilsPart1 string

//go:embed vendor/dev_browser_snapshot/role_utils.part2.js
var ariaRoleUtilsPart2 string

//go:embed vendor/dev_browser_snapshot/aria_snapshot.js
var ariaSnapshotJS string

func baseScript() string {
	return baseSnapshotJS
}

func ariaScript() string {
	vendorCode := strings.Join([]string{
		ariaDomUtils,
		ariaYamlUtils,
		ariaRoleUtilsPart1,
		ariaRoleUtilsPart2,
		ariaSnapshotJS,
	}, "\n")

	var sb strings.Builder
	sb.WriteString("(() => {\n")
	sb.WriteString("  if (globalThis.__devBrowser_getAISnapshotAria) return;\n\n")
	sb.WriteString(vendorCode)
	sb.WriteString("\n\n  function __devBrowser_ariaSnapshot(userOpts) {\n")
	sb.WriteString("    const opts = userOpts || {};\n")
	sb.WriteString("    const format = String(opts.format || \"list\").toLowerCase();\n")
	sb.WriteString("    const maxItems = typeof opts.maxItems === \"number\" && opts.maxItems > 0 ? opts.maxItems : 80;\n")
	sb.WriteString("    const maxChars = typeof opts.maxChars === \"number\" && opts.maxChars > 0 ? opts.maxChars : 8000;\n")
	sb.WriteString("    const interactiveOnly = opts.interactiveOnly !== false;\n")
	sb.WriteString("    const includeHeadings = opts.includeHeadings !== false;\n\n")
	sb.WriteString("    const snap = generateAriaTree(document.body);\n")
	sb.WriteString("    const nodesToWalk = snap.root && snap.root.role === \"fragment\" ? (snap.root.children || []) : [snap.root];\n\n")
	sb.WriteString("    function truncate(text) {\n")
	sb.WriteString("      if (typeof text !== \"string\") return \"\";\n")
	sb.WriteString("      if (text.length <= maxChars) return text;\n")
	sb.WriteString("      const suffix = `\\n- [...] truncated (max_chars=${maxChars})`;\n")
	sb.WriteString("      return text.slice(0, Math.max(0, maxChars - suffix.length)) + suffix;\n")
	sb.WriteString("    }\n\n")
	sb.WriteString("    const refsObject = {};\n")
	sb.WriteString("    try {\n")
	sb.WriteString("      for (const [ref, element] of snap.elements) refsObject[ref] = element;\n")
	sb.WriteString("    } catch {\n")
	sb.WriteString("    }\n")
	sb.WriteString("    globalThis.__devBrowserRefs = refsObject;\n\n")
	sb.WriteString("    const items = [];\n")
	sb.WriteString("    let currentHeading = null;\n")
	sb.WriteString("    const stack = [];\n")
	sb.WriteString("    for (let i = nodesToWalk.length - 1; i >= 0; i--) stack.push(nodesToWalk[i]);\n")
	sb.WriteString("    while (stack.length) {\n")
	sb.WriteString("      const node = stack.pop();\n")
	sb.WriteString("      if (!node || typeof node === \"string\") continue;\n\n")
	sb.WriteString("      if (includeHeadings && node.role === \"heading\" && node.name) currentHeading = node.name;\n\n")
	sb.WriteString("      if (!interactiveOnly || node.ref) {\n")
	sb.WriteString("        if (node.ref) {\n")
	sb.WriteString("          items.push({\n")
	sb.WriteString("            ref: node.ref,\n")
	sb.WriteString("            role: node.role,\n")
	sb.WriteString("            name: node.name || null,\n")
	sb.WriteString("            heading: currentHeading || null,\n")
	sb.WriteString("            disabled: !!node.disabled,\n")
	sb.WriteString("            checked: node.checked ?? null,\n")
	sb.WriteString("            expanded: !!node.expanded,\n")
	sb.WriteString("            selected: !!node.selected,\n")
	sb.WriteString("            pressed: node.pressed ?? null,\n")
	sb.WriteString("            active: !!node.active,\n")
	sb.WriteString("            cursorPointer: !!(node.box && node.box.cursor === \"pointer\")\n")
	sb.WriteString("          });\n")
	sb.WriteString("          if (items.length >= maxItems) break;\n")
	sb.WriteString("        }\n")
	sb.WriteString("      }\n\n")
	sb.WriteString("      const children = node.children || [];\n")
	sb.WriteString("      for (let i = children.length - 1; i >= 0; i--) stack.push(children[i]);\n")
	sb.WriteString("    }\n\n")
	sb.WriteString("    const truncated = items.length >= maxItems;\n")
	sb.WriteString("    const listYaml = globalThis.__devBrowser_buildYaml(items, { maxItems, maxChars, truncated });\n\n")
	sb.WriteString("    if (format === \"list\") return { yaml: listYaml, items };\n")
	sb.WriteString("    if (format === \"tree\") {\n")
	sb.WriteString("      if (!interactiveOnly) return { yaml: truncate(renderAriaTree(snap)), items };\n\n")
	sb.WriteString("      function prune(node) {\n")
	sb.WriteString("        if (!node || typeof node === \"string\") return null;\n")
	sb.WriteString("        const kids = node.children || [];\n")
	sb.WriteString("        const next = [];\n")
	sb.WriteString("        for (const child of kids) {\n")
	sb.WriteString("          const pruned = prune(child);\n")
	sb.WriteString("          if (pruned) next.push(pruned);\n")
	sb.WriteString("        }\n")
	sb.WriteString("        node.children = next;\n")
	sb.WriteString("        const keep = !!node.ref || next.length > 0 || (includeHeadings && node.role === \"heading\" && node.name);\n")
	sb.WriteString("        return keep ? node : null;\n")
	sb.WriteString("      }\n")
	sb.WriteString("      const prunedRoot = prune(snap.root);\n")
	sb.WriteString("      const pruned = { ...snap, root: prunedRoot || { role: \"fragment\", name: \"\", children: [] } };\n")
	sb.WriteString("      return { yaml: truncate(renderAriaTree(pruned)), items };\n")
	sb.WriteString("    }\n")
	sb.WriteString("    throw new Error(`Unknown snapshot format: ${format}`);\n")
	sb.WriteString("  }\n\n")
	sb.WriteString("  globalThis.__devBrowser_getAISnapshotAria = __devBrowser_ariaSnapshot;\n")
	sb.WriteString("})();\n")
	return sb.String()
}
