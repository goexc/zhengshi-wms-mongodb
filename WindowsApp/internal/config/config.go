package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"zhengshi-wms-windowsapp/internal/localfile"
)

const DefaultAPIBaseURL = "https://wmsx.api.goexc.cn:1443"
const APIBaseURLEnv = "ZHENGSHI_WMS_API_BASE_URL"
const DefaultImageBaseURL = "https://wms.file.goexc.cn/images/"
const ImageBaseURLEnv = "ZHENGSHI_WMS_IMAGE_BASE_URL"

type Config struct {
	APIBaseURL   string         `json:"api_base_url"`
	RememberUser bool           `json:"remember_user"`
	KeepLoggedIn bool           `json:"keep_logged_in"`
	Mobile       string         `json:"mobile,omitempty"`
	Workspace    WorkspaceState `json:"workspace,omitempty"`
}

// TableLayoutState contains presentation-only preferences for one table. The
// keys are stable data-member names, never business values returned by the API.
type TableLayoutState struct {
	ColumnOrder  []string       `json:"column_order,omitempty"`
	ColumnWidths map[string]int `json:"column_widths,omitempty"`
}

// WorkspaceState only stores navigation, window/table/splitter layout,
// table-density and fixed-enum filter preferences. Raw text/IDs, loaded
// business rows and drafts are excluded.
type WorkspaceState struct {
	LayoutVersion      int                         `json:"layout_version,omitempty"`
	TableLayouts       map[string]TableLayoutState `json:"table_layouts,omitempty"`
	SplitterLayouts    map[string][]int            `json:"splitter_layouts,omitempty"`
	APIBaseURL         string                      `json:"api_base_url,omitempty"`
	Mobile             string                      `json:"mobile,omitempty"`
	OpenPages          []string                    `json:"open_pages,omitempty"`
	CurrentPage        string                      `json:"current_page,omitempty"`
	MaterialPageSize   int                         `json:"material_page_size,omitempty"`
	InventoryPageSize  int                         `json:"inventory_page_size,omitempty"`
	InboundPageSize    int                         `json:"inbound_page_size,omitempty"`
	OutboundPageSize   int                         `json:"outbound_page_size,omitempty"`
	InventoryModeIndex int                         `json:"inventory_mode_index,omitempty"`
	InventoryTypeIndex int                         `json:"inventory_type_index,omitempty"`
	InboundStatusIndex int                         `json:"inbound_status_index,omitempty"`
	InboundTypeIndex   int                         `json:"inbound_type_index,omitempty"`
	OutboundStageIndex int                         `json:"outbound_stage_index,omitempty"`
	OutboundTypeIndex  int                         `json:"outbound_type_index,omitempty"`
	SideMenuCollapsed  bool                        `json:"side_menu_collapsed,omitempty"`
	WindowBoundsSet    bool                        `json:"window_bounds_set,omitempty"`
	WindowX            int                         `json:"window_x,omitempty"`
	WindowY            int                         `json:"window_y,omitempty"`
	WindowWidth        int                         `json:"window_width,omitempty"`
	WindowHeight       int                         `json:"window_height,omitempty"`
}

func Default() Config {
	return Config{APIBaseURL: DefaultAPIBaseURL, KeepLoggedIn: true}
}

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir = filepath.Join(dir, "ZhengshiWMS")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "windowsapp.json"), nil
}

func Load() Config {
	name, err := path()
	if err != nil {
		return withEnvironment(Default())
	}
	return withEnvironment(loadAt(name))
}

func loadAt(name string) Config {
	data, _, err := localfile.ReadWithBackup(name, 0o600, validateConfigData)
	if err != nil {
		return Default()
	}
	cfg, err := decodeConfig(data)
	if err != nil {
		return Default()
	}
	return cfg
}

func withEnvironment(cfg Config) Config {
	if value := os.Getenv(APIBaseURLEnv); value != "" {
		cfg.APIBaseURL = value
	}
	return cfg
}

func ImageBaseURL() string {
	if value := os.Getenv(ImageBaseURLEnv); value != "" {
		return value
	}
	return DefaultImageBaseURL
}

func Save(cfg Config) error {
	name, err := path()
	if err != nil {
		return err
	}
	return saveAt(name, cfg)
}

func saveAt(name string, cfg Config) error {
	if cfg.APIBaseURL == "" {
		return errors.New("API 地址不能为空")
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return localfile.WriteAtomicWithBackup(name, data, 0o600, validateConfigData)
}

func validateConfigData(data []byte) error {
	_, err := decodeConfig(data)
	return err
}

func decodeConfig(data []byte) (Config, error) {
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.APIBaseURL == "" {
		return Config{}, errors.New("API 地址为空")
	}
	return cfg, nil
}
