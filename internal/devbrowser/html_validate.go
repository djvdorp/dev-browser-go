package devbrowser

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/net/html"
)

type HTMLValidateFinding struct {
	RuleID   string `json:"ruleId"`
	Message  string `json:"message"`
	Location string `json:"location"`
	Snippet  string `json:"snippet,omitempty"`
}

// ValidateHTML performs a light-weight static markup validation on the provided HTML.
// This is intentionally NOT a full W3C validator.
func ValidateHTML(docHTML string) ([]HTMLValidateFinding, error) {
	root, err := html.Parse(strings.NewReader(docHTML))
	if err != nil {
		return nil, err
	}

	findings := []HTMLValidateFinding{}

	// 1) duplicate IDs.
	seen := map[string]bool{}
	idToFirstLoc := map[string]string{}
	walk(root, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		id := attr(n, "id")
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		loc := nodePath(n)
		if !seen[id] {
			seen[id] = true
			idToFirstLoc[id] = loc
			return
		}
		msg := fmt.Sprintf("duplicate id '%s' (first seen at %s)", id, idToFirstLoc[id])
		findings = append(findings, HTMLValidateFinding{RuleID: "duplicate-id", Message: msg, Location: loc, Snippet: snippet(n)})
	})

	// 2) missing alt on img.
	walk(root, func(n *html.Node) {
		if n.Type != html.ElementNode || strings.ToLower(n.Data) != "img" {
			return
		}
		if _, ok := attrOk(n, "alt"); !ok {
			findings = append(findings, HTMLValidateFinding{RuleID: "img-alt", Message: "img missing alt attribute", Location: nodePath(n), Snippet: snippet(n)})
		}
	})

	// 3) form controls missing accessible name (basic).
	walk(root, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		tag := strings.ToLower(n.Data)
		if tag != "input" && tag != "textarea" && tag != "select" && tag != "button" {
			return
		}
		// ignore input type=hidden
		if tag == "input" {
			typ := strings.ToLower(strings.TrimSpace(attr(n, "type")))
			if typ == "hidden" {
				return
			}
		}
		if hasAccessibleName(root, n) {
			return
		}
		findings = append(findings, HTMLValidateFinding{RuleID: "control-name", Message: "form control missing accessible name (label/aria-label/aria-labelledby/title)", Location: nodePath(n), Snippet: snippet(n)})
	})

	// 4) a few common invalid nesting rules.
	walk(root, func(n *html.Node) {
		if n.Type != html.ElementNode {
			return
		}
		tag := strings.ToLower(n.Data)
		switch tag {
		case "p":
			// Disallow block elements as direct/indirect descendants of <p>.
			walk(n, func(child *html.Node) {
				if child == n || child.Type != html.ElementNode {
					return
				}
				ct := strings.ToLower(child.Data)
				if ct == "div" || ct == "p" || ct == "h1" || ct == "h2" || ct == "h3" || ct == "h4" || ct == "h5" || ct == "h6" || ct == "ul" || ct == "ol" || ct == "table" {
					findings = append(findings, HTMLValidateFinding{RuleID: "nesting", Message: fmt.Sprintf("invalid nesting: <%s> inside <p>", ct), Location: nodePath(child), Snippet: snippet(child)})
				}
			})
		case "a":
			// Disallow nested anchors.
			walk(n, func(child *html.Node) {
				if child == n || child.Type != html.ElementNode {
					return
				}
				if strings.ToLower(child.Data) == "a" {
					findings = append(findings, HTMLValidateFinding{RuleID: "nesting", Message: "invalid nesting: <a> inside <a>", Location: nodePath(child), Snippet: snippet(child)})
				}
			})
		}
	})

	// Deterministic ordering.
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].RuleID == findings[j].RuleID {
			return findings[i].Location < findings[j].Location
		}
		return findings[i].RuleID < findings[j].RuleID
	})

	return findings, nil
}

func walk(n *html.Node, fn func(*html.Node)) {
	if n == nil {
		return
	}
	fn(n)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c, fn)
	}
}

func attr(n *html.Node, name string) string {
	v, _ := attrOk(n, name)
	return v
}

func attrOk(n *html.Node, name string) (string, bool) {
	if n == nil {
		return "", false
	}
	name = strings.ToLower(name)
	for _, a := range n.Attr {
		if strings.ToLower(a.Key) == name {
			return a.Val, true
		}
	}
	return "", false
}

func nodePath(n *html.Node) string {
	if n == nil {
		return ""
	}
	parts := []string{}
	for cur := n; cur != nil; cur = cur.Parent {
		if cur.Type != html.ElementNode {
			continue
		}
		seg := strings.ToLower(cur.Data)
		if id := strings.TrimSpace(attr(cur, "id")); id != "" {
			seg += "#" + id
		}
		parts = append(parts, seg)
		if strings.ToLower(cur.Data) == "html" {
			break
		}
	}
	// reverse
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ">")
}

func snippet(n *html.Node) string {
	if n == nil || n.Type != html.ElementNode {
		return ""
	}
	b := &strings.Builder{}
	_ = html.Render(b, n)
	out := strings.TrimSpace(b.String())
	out, _, _ = clampBody(out, 512)
	return out
}

func hasAccessibleName(root, n *html.Node) bool {
	if n == nil {
		return false
	}
	if v := strings.TrimSpace(attr(n, "aria-label")); v != "" {
		return true
	}
	if v := strings.TrimSpace(attr(n, "title")); v != "" {
		return true
	}
	if v := strings.TrimSpace(attr(n, "aria-labelledby")); v != "" {
		ids := strings.Fields(v)
		for _, id := range ids {
			if strings.TrimSpace(id) == "" {
				continue
			}
			if label := findByID(root, id); label != nil {
				if strings.TrimSpace(textContent(label)) != "" {
					return true
				}
			}
		}
	}
	// implicit/explicit <label for="id">
	if id := strings.TrimSpace(attr(n, "id")); id != "" {
		if lab := findLabelFor(root, id); lab != nil {
			if strings.TrimSpace(textContent(lab)) != "" {
				return true
			}
		}
	}
	// wrapped by <label>
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && strings.ToLower(p.Data) == "label" {
			if strings.TrimSpace(textContent(p)) != "" {
				return true
			}
		}
	}
	// Buttons can use their text content.
	if strings.ToLower(n.Data) == "button" {
		if strings.TrimSpace(textContent(n)) != "" {
			return true
		}
	}
	return false
}

func textContent(n *html.Node) string {
	if n == nil {
		return ""
	}
	b := &strings.Builder{}
	walk(n, func(c *html.Node) {
		if c.Type == html.TextNode {
			b.WriteString(c.Data)
			b.WriteString(" ")
		}
	})
	return strings.TrimSpace(b.String())
}

func findByID(root *html.Node, id string) *html.Node {
	var out *html.Node
	walk(root, func(n *html.Node) {
		if out != nil {
			return
		}
		if n.Type != html.ElementNode {
			return
		}
		if strings.TrimSpace(attr(n, "id")) == id {
			out = n
		}
	})
	return out
}

func findLabelFor(root *html.Node, targetID string) *html.Node {
	var out *html.Node
	walk(root, func(n *html.Node) {
		if out != nil {
			return
		}
		if n.Type != html.ElementNode || strings.ToLower(n.Data) != "label" {
			return
		}
		if strings.TrimSpace(attr(n, "for")) == targetID {
			out = n
		}
	})
	return out
}
