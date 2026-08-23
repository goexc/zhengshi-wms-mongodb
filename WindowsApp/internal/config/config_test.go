package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvironmentOverridesAPIBaseURL(t *testing.T) {
	t.Setenv(APIBaseURLEnv, "https://test.example.invalid")
	cfg := withEnvironment(Default())
	if cfg.APIBaseURL != "https://test.example.invalid" {
		t.Fatalf("APIBaseURL = %q", cfg.APIBaseURL)
	}
}

func TestDefaultKeepsLoginEnabled(t *testing.T) {
	if !Default().KeepLoggedIn {
		t.Fatal("keep login should default to enabled")
	}
}

func TestImageBaseURLCanBeOverridden(t *testing.T) {
	t.Setenv(ImageBaseURLEnv, "https://files.example.invalid/materials/")
	if got := ImageBaseURL(); got != "https://files.example.invalid/materials/" {
		t.Fatalf("ImageBaseURL = %q", got)
	}
}

func TestWorkspaceStateContainsOnlyNavigationLayoutAndPageDensity(t *testing.T) {
	state := WorkspaceState{
		OpenPages: []string{"material"}, CurrentPage: "material", MaterialPageSize: 50,
		SideMenuCollapsed: true, WindowBoundsSet: true, WindowWidth: 1440, WindowHeight: 840,
		LayoutVersion: 1,
		TableLayouts: map[string]TableLayoutState{
			"material.results": {ColumnOrder: []string{"Name"}, ColumnWidths: map[string]int{"Name": 180}},
		},
		SplitterLayouts: map[string][]int{"main.workspace": {176, 900}},
	}
	if len(state.OpenPages) != 1 || state.CurrentPage != "material" || state.MaterialPageSize != 50 ||
		!state.SideMenuCollapsed || !state.WindowBoundsSet || state.WindowWidth != 1440 || state.WindowHeight != 840 ||
		state.LayoutVersion != 1 || state.TableLayouts["material.results"].ColumnWidths["Name"] != 180 || state.SplitterLayouts["main.workspace"][0] != 176 {
		t.Fatalf("state = %#v", state)
	}
}

func TestConfigSaveIsAtomicAndCorruptPrimaryRecoversBackup(t *testing.T) {
	name := filepath.Join(t.TempDir(), "windowsapp.json")
	first := Default()
	first.Mobile = "first"
	if err := saveAt(name, first); err != nil {
		t.Fatal(err)
	}
	second := Default()
	second.Mobile = "second"
	if err := saveAt(name, second); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	got := loadAt(name)
	if got.Mobile != "first" {
		t.Fatalf("recovered mobile = %q", got.Mobile)
	}
	restored, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateConfigData(restored); err != nil {
		t.Fatalf("primary was not restored: %v", err)
	}
}
