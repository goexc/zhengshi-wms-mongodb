package ui

import (
	"testing"

	"zhengshi-wms-windowsapp/internal/api"
)

func TestMaterialQuoteStatusMappings(t *testing.T) {
	tests := []struct {
		label string
		value string
	}{
		{"未报价", "unquoted"},
		{"报价中", "quoting"},
		{"已报价", "quoted"},
		{"已定价", "priced"},
	}
	for _, test := range tests {
		if got := materialQuoteStatusValue(test.label); got != test.value {
			t.Fatalf("status value for %q = %q, want %q", test.label, got, test.value)
		}
		if got := materialQuoteStatusLabel(test.value); got != test.label {
			t.Fatalf("status label for %q = %q, want %q", test.value, got, test.label)
		}
	}
}

func TestDefaultMaterialQuoteCostsMatchWebSections(t *testing.T) {
	items := defaultMaterialQuoteCostItems()
	if len(items) != 26 {
		t.Fatalf("default item count = %d, want 26", len(items))
	}
	enabled := 0
	for _, item := range items {
		if item.Enabled {
			enabled++
			if item.CategoryCode != "material" || item.Name != "原材料" {
				t.Fatalf("unexpected enabled default item: %#v", item)
			}
		}
	}
	if enabled != 1 {
		t.Fatalf("enabled default item count = %d, want 1", enabled)
	}
}

func TestNormalizeMaterialQuoteCostIndexes(t *testing.T) {
	items := []api.MaterialQuoteCostItem{
		{CategoryCode: "other", CategoryName: "其他成本", Name: "B", Index: 7},
		{CategoryCode: "material", CategoryName: "材料成本", Name: "A2", Index: 4},
		{CategoryCode: "material", CategoryName: "材料成本", Name: "A1", Index: 1},
	}
	normalizeMaterialQuoteCostIndexes(items)
	if items[0].Name != "A1" || items[0].Index != 1 || items[1].Name != "A2" || items[1].Index != 2 {
		t.Fatalf("material indexes were not normalized: %#v", items)
	}
	if items[2].Name != "B" || items[2].Index != 1 {
		t.Fatalf("other section index = %#v, want 1", items[2])
	}
}

func TestFlattenMaterialCategoryOptionsExcludesSubtree(t *testing.T) {
	categories := []api.MaterialCategory{{
		ID: "root", Name: "根",
		Children: []api.MaterialCategory{{ID: "child", Name: "子"}},
	}, {ID: "other", Name: "其他"}}
	excluded := map[string]bool{}
	collectMaterialCategoryIDs(categories[0], excluded)
	options := flattenMaterialCategoryOptions(categories, "", excluded)
	if len(options) != 1 || options[0].ID != "other" {
		t.Fatalf("options = %#v, want only other root", options)
	}
}
