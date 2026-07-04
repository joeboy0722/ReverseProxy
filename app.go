package main

import (
	"context"

	"reverse-proxy/backend/proxy"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx        context.Context
	manager    *proxy.Manager
	engine     *proxy.Engine
	logManager *proxy.LogManager
}

// NewApp 建立一個新的 App 實例
func NewApp() *App {
	logMgr := proxy.NewLogManager(200)
	mgr := proxy.NewManager()
	engine := proxy.NewEngine(mgr, logMgr)

	// 載入持久化的自訂憑證
	cfg := proxy.LoadConfig()
	if cfg.CertPath != "" && cfg.KeyPath != "" {
		_ = engine.ReloadTLSConfig(cfg.CertPath, cfg.KeyPath)
	}

	return &App{
		manager:    mgr,
		engine:     engine,
		logManager: logMgr,
	}
}

// startup 當 App 啟動時呼叫，儲存 context
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.logManager.SetContext(ctx)
}

// StartServer 啟動或重新啟動反向代理伺服器
func (a *App) StartServer(bindAddr string, port int, useTLS bool) error {
	return a.engine.Start(bindAddr, port, useTLS)
}

// StopServer 停止反向代理伺服器
func (a *App) StopServer() error {
	return a.engine.Stop()
}

// ServerStatus 取回當前伺服器狀態
type ServerStatus struct {
	IsRunning bool   `json:"isRunning"`
	BindAddr  string `json:"bindAddr"`
	Port      int    `json:"port"`
}

// GetServerStatus 取得當前反向代理運行狀態
func (a *App) GetServerStatus() ServerStatus {
	isRunning, bindAddr, port := a.engine.Status()
	return ServerStatus{
		IsRunning: isRunning,
		BindAddr:  bindAddr,
		Port:      port,
	}
}

// GetRules 取得所有轉發規則
func (a *App) GetRules() []proxy.RouteRule {
	return a.manager.GetRules()
}

// AddRule 新增一筆轉發規則
func (a *App) AddRule(source string, routeType string, target string) (string, error) {
	return a.manager.AddRule(source, proxy.RouteType(routeType), target)
}

// DeleteRule 刪除一筆轉發規則
func (a *App) DeleteRule(id string) error {
	return a.manager.DeleteRule(id)
}

// ToggleRule 啟用/暫停指定轉發規則
func (a *App) ToggleRule(id string, active bool) error {
	return a.manager.ToggleRule(id, active)
}

// UpdateRuleHeaders 更新指定規則的 Headers
func (a *App) UpdateRuleHeaders(id string, headers map[string]string) error {
	return a.manager.UpdateRuleHeaders(id, headers)
}

// UpdateRuleConfig 更新指定規則的代理設定
func (a *App) UpdateRuleConfig(id string, keepPrefix bool, injectBase bool, redirectSlash bool, healthCheckEnabled bool, healthCheckPath string, showInIndex bool, title string) error {
	return a.manager.UpdateRuleConfig(id, keepPrefix, injectBase, redirectSlash, healthCheckEnabled, healthCheckPath, showInIndex, title)
}

// GetLogs 取得記憶體中的所有即時日誌
func (a *App) GetLogs() []*proxy.RequestLog {
	return a.logManager.GetLogs()
}

// ClearLogs 清空所有即時日誌
func (a *App) ClearLogs() {
	a.logManager.ClearLogs()
}

// SetCustomCert 設定並啟用自訂 TLS 憑證
func (a *App) SetCustomCert(certPath, keyPath string) error {
	err := a.engine.ReloadTLSConfig(certPath, keyPath)
	if err != nil {
		return err
	}

	cfg := proxy.LoadConfig()
	cfg.CertPath = certPath
	cfg.KeyPath = keyPath
	return cfg.Save()
}

// CustomCert 儲存憑證回傳格式
type CustomCert struct {
	CertPath string `json:"certPath"`
	KeyPath  string `json:"keyPath"`
}

// GetCustomCert 取得目前儲存的憑證設定
func (a *App) GetCustomCert() CustomCert {
	cfg := proxy.LoadConfig()
	return CustomCert{
		CertPath: cfg.CertPath,
		KeyPath:  cfg.KeyPath,
	}
}

// SelectDirectory 提供 UI 選取本地目錄
func (a *App) SelectDirectory() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "選擇靜態網頁目錄",
	})
	return dir, err
}

// SelectFile 提供 UI 選取憑證/金鑰檔案
func (a *App) SelectFile(filterName, extensions string) (string, error) {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "選擇憑證或私鑰檔案",
		Filters: []runtime.FileFilter{
			{
				DisplayName: filterName,
				Pattern:     extensions,
			},
		},
	})
	return file, err
}

// NavConfig 導覽首頁傳輸用結構
type NavConfig struct {
	NavTitle    string `json:"navTitle"`
	NavSubtitle string `json:"navSubtitle"`
	ThemeColor  string `json:"themeColor"`
}

// GetNavConfig 取得當前首頁導覽全域設定，若無則賦予預設值
func (a *App) GetNavConfig() NavConfig {
	cfg := proxy.LoadConfig()
	title := cfg.NavTitle
	if title == "" {
		title = "反向代理服務導航首頁"
	}
	subtitle := cfg.NavSubtitle
	if subtitle == "" {
		subtitle = "歡迎使用自訂反向代理伺服器。以下為您已啟用且公開的轉發服務，點擊即可快速跳轉。"
	}
	color := cfg.ThemeColor
	if color == "" {
		color = "#6366f1"
	}
	return NavConfig{
		NavTitle:    title,
		NavSubtitle: subtitle,
		ThemeColor:  color,
	}
}

// SaveNavConfig 儲存首頁導覽全域設定
func (a *App) SaveNavConfig(navTitle, navSubtitle, themeColor string) error {
	cfg := proxy.LoadConfig()
	cfg.NavTitle = navTitle
	cfg.NavSubtitle = navSubtitle
	cfg.ThemeColor = themeColor
	return cfg.Save()
}

// RegisterLogListener 供 CLI 註冊日誌監聽
func (a *App) RegisterLogListener(fn func(*proxy.RequestLog)) {
	a.logManager.RegisterListener(fn)
}

// UnregisterLogListener 供 CLI 註銷日誌監聽
func (a *App) UnregisterLogListener() {
	a.logManager.UnregisterListener()
}
