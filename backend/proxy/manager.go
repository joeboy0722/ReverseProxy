package proxy

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RouteType string

const (
	RouteTypeHost   RouteType = "host"
	RouteTypePath   RouteType = "path"
	RouteTypeStatic RouteType = "static" // 靜態目錄路由
)

// RouteRule 定義一條轉換規則
type RouteRule struct {
	ID             string            `json:"id"`
	Source         string            `json:"source"`
	Type           RouteType         `json:"type"` // "host"、"path" 或是 "static"
	Target         string            `json:"target"`
	Active         bool              `json:"active"`
	Headers        map[string]string `json:"headers"` // 自訂 Headers 注入
	Healthy        bool              `json:"healthy"` // 執行時健康狀態，傳遞給前端用
	KeepPrefix     bool              `json:"keepPrefix"` // 轉發時是否保留來源前綴
	InjectBase     bool              `json:"injectBase"` // 是否在 HTML 中注入 <base> 標籤
	RedirectSlash  bool              `json:"redirectSlash"` // 是否自動將缺少尾部斜線的請求重導向至帶斜線路徑
	HealthCheckEnabled *bool             `json:"healthCheckEnabled"` // 是否啟用健康檢查 (預設為 true)
	HealthCheckPath    string            `json:"healthCheckPath"`    // 自訂健康檢查路徑 (例如 /healthz)
	CreatedAt      time.Time         `json:"createdAt"`
}

// Manager 負責管理所有的路由規則
type Manager struct {
	mu    sync.RWMutex
	rules map[string]*RouteRule

	hostRules map[string]*RouteRule // O(1) 尋找 Host 規則
	pathRules map[string]*RouteRule // O(N) 尋找 Path 前綴匹配規則
	filename  string

	stopHealthCheck chan struct{}
	healthClient    *http.Client // 共用的健康檢查 HTTP Client，避免每次建立新連線
}

func NewManager() *Manager {
	// 建立共用的 Health Check HTTP Client：
	// - 設定 InsecureSkipVerify 讓自簽憑證的 HTTPS 後端也能被偵測
	// - 固定 Timeout 3 秒，避免某個慢後端拖垮整個健康檢查週期
	healthTransport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	healthClient := &http.Client{
		Transport: healthTransport,
		Timeout:   3 * time.Second,
	}

	m := &Manager{
		rules:           make(map[string]*RouteRule),
		hostRules:       make(map[string]*RouteRule),
		pathRules:       make(map[string]*RouteRule),
		filename:        "routes.json",
		stopHealthCheck: make(chan struct{}),
		healthClient:    healthClient,
	}
	m.loadRules()
	go m.StartHealthCheck() // 啟動背景檢查
	return m
}

// StartHealthCheck 每 30 秒檢查一次後端健康狀況
func (m *Manager) StartHealthCheck() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 啟動時先檢查一次
	m.checkAll()

	for {
		select {
		case <-ticker.C:
			m.checkAll()
		case <-m.stopHealthCheck:
			return
		}
	}
}

func (m *Manager) checkAll() {
	m.mu.RLock()
	// 建立副本以免長時間鎖定
	rulesCopy := make([]*RouteRule, 0, len(m.rules))
	for _, r := range m.rules {
		rulesCopy = append(rulesCopy, r)
	}
	m.mu.RUnlock()

	for _, r := range rulesCopy {
		if !r.Active {
			continue
		}
		
		isHealthy := m.pingRule(r)
		
		m.mu.Lock()
		if rule, ok := m.rules[r.ID]; ok {
			rule.Healthy = isHealthy
		}
		m.mu.Unlock()
	}
}

// ensureURLScheme 確保 Target 具備 http:// 或 https:// 前綴
func ensureURLScheme(target string) string {
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		return "http://" + target
	}
	return target
}

func (m *Manager) pingRule(r *RouteRule) bool {
	// 如果關閉了健康檢查，預設直接判定為健康
	if r.HealthCheckEnabled != nil && !*r.HealthCheckEnabled {
		return true
	}

	if r.Type == RouteTypeStatic {
		// 靜態目錄檢查：確認本地目錄是否存在
		info, err := os.Stat(r.Target)
		if err != nil {
			return false
		}
		return info.IsDir()
	}

	// 網路連線檢查：使用共用 Client 發送 HTTP GET 請求
	targetURL := ensureURLScheme(r.Target)
	if r.HealthCheckPath != "" {
		path := r.HealthCheckPath
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		targetURL = strings.TrimSuffix(targetURL, "/") + path
	}

	resp, err := m.healthClient.Get(targetURL)
	if err != nil {
		// 連線失敗或超時
		return false
	}
	defer resp.Body.Close()

	// 只要能建立連線並取得回應，就代表伺服器活著且有連通
	return true
}

func (m *Manager) saveRules() {
	// 假設呼叫時已經處於 mutex 鎖定狀態下
	data, err := json.MarshalIndent(m.rules, "", "  ")
	if err == nil {
		// 確保放置在執行檔目錄附近或明確路徑 (此處為啟動目錄)
		os.WriteFile(m.filename, data, 0644)
	}
}

func (m *Manager) loadRules() {
	data, err := os.ReadFile(m.filename)
	if err != nil {
		return // 檔案不存在等原因，忽略
	}

	var loadedRules map[string]*RouteRule
	if err := json.Unmarshal(data, &loadedRules); err == nil {
		m.rules = loadedRules
		// 重建快速搜尋樹，並初始化相容舊資料的預設值
		for _, r := range m.rules {
			if r.HealthCheckEnabled == nil {
				defaultVal := true
				r.HealthCheckEnabled = &defaultVal
			}
			if r.Active {
				if r.Type == RouteTypeHost {
					m.hostRules[r.Source] = r
				} else {
					m.pathRules[r.Source] = r
				}
			}
		}
	}
}

// AddRule 新增一筆代理規則
func (m *Manager) AddRule(source string, routeType RouteType, target string) (string, error) {
	if source == "" || target == "" {
		return "", errors.New("來源或目標不能為空")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 移除新增時的名稱衝突檢查，允許建立多筆相同來源的規則
	// 但如果新增時是 Active (預設值)，我們應該確保目前沒有其他相同且 Active 的規則
	for _, r := range m.rules {
		if r.Source == source && r.Type == routeType && r.Active {
			return "", errors.New("該來源名稱目前已有啟用的規則，請先停用它再新增")
		}
	}

	id := uuid.New().String()
	rule := &RouteRule{
		ID:             id,
		Source:         source,
		Type:           routeType,
		Target:         target,
		Active:             true, // 預設新增時就啟用
		Headers:            make(map[string]string),
		KeepPrefix:         false, // 預設移除前綴，維持原有行為
		InjectBase:         true,  // 預設注入 Base，這樣代理網頁時就更容易成功
		RedirectSlash:      false, // 預設關閉自動重導向斜線，避免影響 API 測試
		HealthCheckEnabled: func() *bool { b := true; return &b }(),
		HealthCheckPath:    "",
		CreatedAt:          time.Now(),
	}

	m.rules[id] = rule
	if routeType == RouteTypeHost {
		m.hostRules[source] = rule
	} else {
		m.pathRules[source] = rule
	}

	m.saveRules()

	return id, nil
}

// UpdateRule 更新一筆代理規則 (例如修改 Target)
func (m *Manager) UpdateRule(id string, target string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, exists := m.rules[id]
	if !exists {
		return errors.New("找不到指定的規則")
	}

	rule.Target = target
	return nil
}

// ToggleRule 切換規則的啟用與停用狀態
func (m *Manager) ToggleRule(id string, active bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, exists := m.rules[id]
	if !exists {
		return errors.New("找不到指定的規則")
	}

	// 如果是要啟用 (Toggle to True)，檢查是否有其他同來源且已啟用的規則
	if active {
		for _, r := range m.rules {
			if r.ID != id && r.Source == rule.Source && r.Type == rule.Type && r.Active {
				return errors.New("相同的來源目前已有啟用的規則，請先停用其他的")
			}
		}
	}

	rule.Active = active

	// 同步更新搜尋用 map 的指標 (因為可能有多個同名，搜尋時應該永遠指向 active 的那個)
	// 當變成 false 時，理論上下一次 Match 不會使用到它，但還是從搜尋 map 中移除比較乾淨
	if active {
		if rule.Type == RouteTypeHost {
			m.hostRules[rule.Source] = rule
		} else {
			m.pathRules[rule.Source] = rule
		}
	} else {
		if rule.Type == RouteTypeHost {
			// 只有當 map 裡的真的是目前這個 rule 才刪除 (避免剛好誤刪其他切換的同名規則)
			if m.hostRules[rule.Source] == rule {
				delete(m.hostRules, rule.Source)
			}
		} else {
			if m.pathRules[rule.Source] == rule {
				delete(m.pathRules, rule.Source)
			}
		}
	}

	m.saveRules()

	return nil
}

// DeleteRule 刪除一筆規則
func (m *Manager) DeleteRule(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, exists := m.rules[id]
	if !exists {
		return errors.New("找不到指定的規則")
	}

	delete(m.rules, id)

	// 從快速搜尋樹中移除 (僅當該樹中存的是自己時才移除，以免誤刪到同名稱但被啟用中的另一條規則)
	if rule.Type == RouteTypeHost {
		if m.hostRules[rule.Source] == rule {
			delete(m.hostRules, rule.Source)
		}
	} else {
		if m.pathRules[rule.Source] == rule {
			delete(m.pathRules, rule.Source)
		}
	}

	m.saveRules()

	return nil
}

// GetRules 取得全部的規則列表 (回傳副本避免被外部意外修改，並經過排序以維持介面顯示順序穩定)
func (m *Manager) GetRules() []RouteRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []RouteRule
	for _, r := range m.rules {
		result = append(result, *r)
	}

	// 進行排序：
	// 1. 啟用中 (Active = true) 的規則排在最上方
	// 2. 狀態相同時，依據 CreatedAt 升序排列 (最早建立的在最前面)，確保清單不會隨機亂跳
	sort.Slice(result, func(i, j int) bool {
		if result[i].Active != result[j].Active {
			return result[i].Active && !result[j].Active
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})

	return result
}

// Match 嘗試比對進來的請求應該被導向哪個目標
func (m *Manager) Match(host string, path string) (*RouteRule, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. 優先比對 Host，例如 banana.local (如果 Host 包含 Port，例如 banana.local:8080，我們先切掉 Port)
	hostWithoutPort := strings.Split(host, ":")[0]
	if rule, ok := m.hostRules[hostWithoutPort]; ok && rule.Active {
		return rule, true
	}

	// 2. 接著比對 Path 首碼，例如 /apple。尋找最長的前綴匹配
	var bestMatch *RouteRule
	longestMatchLen := 0

	for prefix, rule := range m.pathRules {
		if !rule.Active {
			continue
		}

		// 若 path == prefix 或者 path 以 prefix/ 開頭
		if strings.HasPrefix(path, prefix) {
			// 避免 /app 錯誤匹配到 /apple 這種情況。最好的做法是拆分 "/"
			// 簡單處理: 完全一樣，或是下一個字元是 "/"
			isMatch := false
			if path == prefix {
				isMatch = true
			} else if len(path) > len(prefix) && path[len(prefix)] == '/' {
				isMatch = true
			} else if strings.HasSuffix(prefix, "/") { // prefix 本身就是 /app/ 這樣的話
				isMatch = true
			}

			if isMatch && len(prefix) > longestMatchLen {
				longestMatchLen = len(prefix)
				bestMatch = rule
			}
		}
	}

	if bestMatch != nil {
		return bestMatch, true
	}

	return nil, false
}

// UpdateRuleHeaders 更新指定規則的 Headers
func (m *Manager) UpdateRuleHeaders(id string, headers map[string]string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, exists := m.rules[id]
	if !exists {
		return errors.New("找不到指定的規則")
	}

	rule.Headers = headers
	m.saveRules()
	return nil
}

// UpdateRuleConfig 更新指定規則的代理設定
func (m *Manager) UpdateRuleConfig(id string, keepPrefix bool, injectBase bool, redirectSlash bool, healthCheckEnabled bool, healthCheckPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rule, exists := m.rules[id]
	if !exists {
		return errors.New("找不到指定的規則")
	}

	rule.KeepPrefix = keepPrefix
	rule.InjectBase = injectBase
	rule.RedirectSlash = redirectSlash
	rule.HealthCheckEnabled = &healthCheckEnabled
	rule.HealthCheckPath = healthCheckPath
	m.saveRules()
	return nil
}
