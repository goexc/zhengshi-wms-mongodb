package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const DefaultAPIBaseURL = "https://wmsx.api.goexc.cn:1443"
const APIBaseURLEnv = "ZHENGSHI_WMS_API_BASE_URL"
const DefaultImageBaseURL = "https://wms.file.goexc.cn/images/"
const ImageBaseURLEnv = "ZHENGSHI_WMS_IMAGE_BASE_URL"

type Config struct {
	APIBaseURL   string `json:"api_base_url"`
	RememberUser bool   `json:"remember_user"`
	KeepLoggedIn bool   `json:"keep_logged_in"`
	Mobile       string `json:"mobile,omitempty"`
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
	cfg := Default()
	name, err := path()
	if err != nil {
		return withEnvironment(cfg)
	}
	data, err := os.ReadFile(name)
	if err != nil {
		return withEnvironment(cfg)
	}
	if json.Unmarshal(data, &cfg) != nil || cfg.APIBaseURL == "" {
		return withEnvironment(Default())
	}
	return withEnvironment(cfg)
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
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(name, data, 0o600)
}
