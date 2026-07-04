package proxy

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ProxyConfig 儲存代理伺服器的全域設定
type ProxyConfig struct {
	CertPath    string `json:"certPath"`
	KeyPath     string `json:"keyPath"`
	NavTitle    string `json:"navTitle"`
	NavSubtitle string `json:"navSubtitle"`
	ThemeColor  string `json:"themeColor"`
}

// getConfigPath 取得全域設定檔的絕對路徑
func getConfigPath() string {
	exeDir := getExeDir()
	if exeDir == "" {
		return "proxy_config.json"
	}
	return filepath.Join(exeDir, "proxy_config.json")
}

// LoadConfig 從檔案載入設定，若檔案不存在則回傳預設設定
func LoadConfig() *ProxyConfig {
	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		// 檔案不存在或無法讀取，回傳空白設定
		return &ProxyConfig{}
	}

	var cfg ProxyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return &ProxyConfig{}
	}
	return &cfg
}

// Save 將設定儲存至檔案
func (c *ProxyConfig) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(getConfigPath(), data, 0644)
}
