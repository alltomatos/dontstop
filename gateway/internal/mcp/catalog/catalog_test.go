package catalog

import "testing"

func TestCatalogHasUniqueTools(t *testing.T) {
	if len(Tools) != 15 {
		t.Fatalf("expected 15 tools, got %d", len(Tools))
	}
	seen := map[string]bool{}
	for _, tool := range Tools {
		if tool.ID == "" || tool.Name == "" {
			t.Fatalf("tool with empty id/name: %+v", tool)
		}
		if seen[tool.ID] {
			t.Fatalf("duplicate tool id %q", tool.ID)
		}
		seen[tool.ID] = true
	}
	health, ok := Find("dontstop_health")
	if !ok || health.Scope != "" {
		t.Fatalf("health must be authenticated without extra scope: %+v", health)
	}
	if _, ok := Find("dontstop_combo_create"); !ok {
		t.Fatal("combo create tool missing")
	}
}
