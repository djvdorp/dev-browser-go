package devbrowser

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/playwright-community/playwright-go"
)

type DomSnapshot struct {
	Engine string                   `json:"engine"`
	Format string                   `json:"format"`
	Items  []map[string]interface{} `json:"items"`
}

type DomDiff struct {
	Added        []map[string]interface{} `json:"added"`
	Removed      []map[string]interface{} `json:"removed"`
	Changed      []DomChange              `json:"changed"`
	AddedCount   int                      `json:"added_count"`
	RemovedCount int                      `json:"removed_count"`
	ChangedCount int                      `json:"changed_count"`
}

type DomChange struct {
	Key    string                 `json:"key"`
	Before map[string]interface{} `json:"before"`
	After  map[string]interface{} `json:"after"`
	Fields []string               `json:"fields"`
}

func CaptureDomSnapshot(page playwright.Page, engine string, maxItems int) (*DomSnapshot, error) {
	if strings.TrimSpace(engine) == "" {
		engine = "simple"
	}
	if maxItems <= 0 {
		maxItems = 200
	}
	// Use the list format because it is stable and already used for ref selection.
	snap, err := GetSnapshot(page, SnapshotOptions{
		Engine:          engine,
		Format:          "list",
		InteractiveOnly: false,
		IncludeHeadings: true,
		MaxItems:        maxItems,
		MaxChars:        8000,
	})
	if err != nil {
		return nil, err
	}
	return &DomSnapshot{Engine: engine, Format: "list", Items: snap.Items}, nil
}

func WriteDomSnapshot(path string, snap *DomSnapshot) error {
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func ReadDomSnapshot(path string) (*DomSnapshot, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snap DomSnapshot
	if err := json.Unmarshal(b, &snap); err != nil {
		return nil, err
	}
	if snap.Items == nil {
		snap.Items = []map[string]interface{}{}
	}
	if strings.TrimSpace(snap.Engine) == "" {
		snap.Engine = "simple"
	}
	if strings.TrimSpace(snap.Format) == "" {
		snap.Format = "list"
	}
	return &snap, nil
}

func DiffDomSnapshots(before, after *DomSnapshot) DomDiff {
	bItems := []map[string]interface{}{}
	aItems := []map[string]interface{}{}
	if before != nil {
		bItems = before.Items
	}
	if after != nil {
		aItems = after.Items
	}

	beforeMap := make(map[string]map[string]interface{}, len(bItems))
	for _, it := range bItems {
		beforeMap[domKey(it)] = it
	}
	afterMap := make(map[string]map[string]interface{}, len(aItems))
	for _, it := range aItems {
		afterMap[domKey(it)] = it
	}

	diff := DomDiff{}

	for k, a := range afterMap {
		b, ok := beforeMap[k]
		if !ok {
			diff.Added = append(diff.Added, a)
			continue
		}
		fields := compareDomFields(b, a)
		if len(fields) > 0 {
			sort.Strings(fields)
			diff.Changed = append(diff.Changed, DomChange{Key: k, Before: b, After: a, Fields: fields})
		}
	}
	for k, b := range beforeMap {
		if _, ok := afterMap[k]; !ok {
			diff.Removed = append(diff.Removed, b)
		}
	}

	// Deterministic output ordering.
	sort.Slice(diff.Added, func(i, j int) bool { return domKey(diff.Added[i]) < domKey(diff.Added[j]) })
	sort.Slice(diff.Removed, func(i, j int) bool { return domKey(diff.Removed[i]) < domKey(diff.Removed[j]) })
	sort.Slice(diff.Changed, func(i, j int) bool { return diff.Changed[i].Key < diff.Changed[j].Key })

	diff.AddedCount = len(diff.Added)
	diff.RemovedCount = len(diff.Removed)
	diff.ChangedCount = len(diff.Changed)

	return diff
}

func domKey(item map[string]interface{}) string {
	role, _ := item["role"].(string)
	name, _ := item["name"].(string)
	heading, _ := item["heading"].(string)
	role = strings.TrimSpace(role)
	name = strings.TrimSpace(name)
	heading = strings.TrimSpace(heading)
	return fmt.Sprintf("%s|%s|%s", role, name, heading)
}

func compareDomFields(before, after map[string]interface{}) []string {
	// Keep this intentionally narrow + stable: compare semantic fields, not volatile refs.
	fields := []string{"role", "name", "heading", "disabled", "checked", "expanded", "selected", "pressed", "active"}
	changed := make([]string, 0, 4)
	for _, f := range fields {
		b := fmt.Sprintf("%v", before[f])
		a := fmt.Sprintf("%v", after[f])
		if b != a {
			changed = append(changed, f)
		}
	}
	return changed
}
