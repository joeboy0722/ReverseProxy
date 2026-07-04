package proxy

import (
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RequestLog 儲存單一 HTTP 請求的代理日誌
type RequestLog struct {
	ID            string            `json:"id"`
	Timestamp     time.Time         `json:"timestamp"`
	Method        string            `json:"method"`
	Path          string            `json:"path"`
	RuleSource    string            `json:"ruleSource"`
	TargetURL     string            `json:"targetURL"`
	StatusCode    int               `json:"statusCode"`
	LatencyMs     int64             `json:"latencyMs"`
	ReqHeaders    map[string]string `json:"reqHeaders"`
	ReqBody       string            `json:"reqBody"`       // 最大限制 8KB，超出則截斷
	RespHeaders   map[string]string `json:"respHeaders"`
	RespBody      string            `json:"respBody"`      // 最大限制 8KB，超出則截斷
	ReqBodyTrunc  bool              `json:"reqBodyTrunc"`  // 請求體是否截斷
	RespBodyTrunc bool              `json:"respBodyTrunc"` // 響應體是否截斷
}

// LogManager 負責管理記憶體中的環形緩衝區日誌，並推送到前端
type LogManager struct {
	mu      sync.Mutex
	logs    []*RequestLog
	maxSize int
	ctx     context.Context
	onLog   func(*RequestLog)
}

// NewLogManager 建立一個新的 LogManager 實例
func NewLogManager(maxSize int) *LogManager {
	return &LogManager{
		logs:    make([]*RequestLog, 0, maxSize),
		maxSize: maxSize,
	}
}

// RegisterListener 註冊日誌監聽器以供 CLI 即時顯示
func (lm *LogManager) RegisterListener(fn func(*RequestLog)) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.onLog = fn
}

// UnregisterListener 註銷日誌監聽器
func (lm *LogManager) UnregisterListener() {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.onLog = nil
}

// SetContext 設定 Wails Context 以便 EventsEmit
func (lm *LogManager) SetContext(ctx context.Context) {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	lm.ctx = ctx
}

// AddLog 加入一筆新的日誌。若超過 maxSize 則拋棄最舊的一筆，並透過 Wails 事件發送給前端
func (lm *LogManager) AddLog(log *RequestLog) {
	lm.mu.Lock()
	// 如果日誌數量超出限制，淘汰最舊的
	if len(lm.logs) >= lm.maxSize {
		lm.logs = lm.logs[1:]
	}
	lm.logs = append(lm.logs, log)
	ctx := lm.ctx
	onLog := lm.onLog
	lm.mu.Unlock()

	// 發送即時事件給前端，不佔用鎖定時間
	if ctx != nil {
		runtime.EventsEmit(ctx, "log:new", log)
	}

	// 若有註冊 CLI 監聽器，則呼叫它以即時輸出
	if onLog != nil {
		onLog(log)
	}
}

// GetLogs 取得目前儲存的所有日誌（以時間降序排序，即最新的在前）
func (lm *LogManager) GetLogs() []*RequestLog {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	n := len(lm.logs)
	result := make([]*RequestLog, n)
	// 反轉陣列，讓最新產生的日誌在最前面
	for i, log := range lm.logs {
		result[n-1-i] = log
	}
	return result
}

// ClearLogs 清除所有記憶體中的日誌
func (lm *LogManager) ClearLogs() {
	lm.mu.Lock()
	lm.logs = make([]*RequestLog, 0, lm.maxSize)
	ctx := lm.ctx
	lm.mu.Unlock()

	if ctx != nil {
		runtime.EventsEmit(ctx, "log:cleared")
	}
}
