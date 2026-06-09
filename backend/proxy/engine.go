package proxy

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Engine 負責真正啟動 Web Server 與轉發請求
type Engine struct {
	manager    *Manager
	logManager *LogManager

	mu     sync.Mutex
	server *http.Server

	isRunning   bool
	bindAddr    string
	port        int
	useTLS      bool
	transport   *http.Transport
	certPath    string
	keyPath     string
	currentCert *tls.Certificate
}

func NewEngine(manager *Manager, logManager *LogManager) *Engine {
	// 使用自訂的 Transport 加強對後端連線的生命週期管理
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConns = 100
	t.MaxIdleConnsPerHost = 100
	t.MaxConnsPerHost = 0
	t.IdleConnTimeout = 90 * time.Second
	
	return &Engine{
		manager:    manager,
		logManager: logManager,
		isRunning:  false,
		transport:  t,
	}
}

// Start 啟動或重新啟動反向代理伺服器
func (e *Engine) Start(bindAddr string, port int, useTLS bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// 如果已經在跑，先關掉
	if e.isRunning && e.server != nil {
		if err := e.server.Shutdown(context.Background()); err != nil {
			log.Printf("關閉原伺服器失敗: %v\n", err)
		}
	}

	addr := fmt.Sprintf("%s:%d", bindAddr, port)

	e.server = &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	e.bindAddr = bindAddr
	e.port = port
	e.useTLS = useTLS
	e.isRunning = true

	go func() {
		lc := net.ListenConfig{
			KeepAlive: 15 * time.Second,
		}
		ln, err := lc.Listen(context.Background(), "tcp", addr)
		if err != nil {
			log.Printf("監聽失敗: %v\n", err)
			e.mu.Lock()
			e.isRunning = false
			e.mu.Unlock()
			return
		}

		if useTLS {
			// 動態載入憑證的設定，若尚未載入自訂憑證則自動 Fallback 產生自簽憑證
			tlsConfig := &tls.Config{
				GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
					e.mu.Lock()
					defer e.mu.Unlock()

					if e.currentCert != nil {
						return e.currentCert, nil
					}

					// Fallback 到動態自簽憑證
					cert, err := generateSelfSignedCert()
					if err != nil {
						return nil, err
					}
					e.currentCert = &cert
					return e.currentCert, nil
				},
			}
			e.server.TLSConfig = tlsConfig

			log.Printf("反向代理伺服器啟動於 https://%s (支援憑證重載)", addr)
			if err := e.server.ServeTLS(ln, "", ""); err != nil && err != http.ErrServerClosed {
				log.Printf("代理伺服器例外關閉: %v\n", err)
			}
		} else {
			log.Printf("反向代理伺服器啟動於 http://%s", addr)
			if err := e.server.Serve(ln); err != nil && err != http.ErrServerClosed {
				log.Printf("代理伺服器例外關閉: %v\n", err)
			}
		}
		
		e.mu.Lock()
		e.isRunning = false
		e.mu.Unlock()
	}()

	return nil
}

// ReloadTLSConfig 動態載入/重載自訂的 TLS 憑證
func (e *Engine) ReloadTLSConfig(certPath, keyPath string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.certPath = certPath
	e.keyPath = keyPath

	if certPath != "" && keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return fmt.Errorf("載入憑證失敗: %v", err)
		}
		e.currentCert = &cert
		log.Printf("成功載入自訂憑證: %s", certPath)
	} else {
		// 卸載憑證，重設為 nil，下次會 Fallback 到自簽憑證
		e.currentCert = nil
		log.Println("已卸載自訂憑證，將還原使用自簽憑證")
	}
	return nil
}

// Stop 停止反向代理伺服器
func (e *Engine) Stop() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.isRunning || e.server == nil {
		return nil
	}

	err := e.server.Shutdown(context.Background())
	e.isRunning = false
	return err
}

// Status 回傳目前狀態
func (e *Engine) Status() (bool, string, int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.isRunning, e.bindAddr, e.port
}

// generateSelfSignedCert 動態產生一個簡單的自簽憑證
func generateSelfSignedCert() (tls.Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Custom Reverse Proxy"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("0.0.0.0")},
		DNSNames:              []string{"localhost"},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return tls.Certificate{}, err
	}
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})

	return tls.X509KeyPair(certPEM, privPEM)
}

// captureResponseWriter 用於攔截並記錄 HTTP 響應狀態碼與部分 Response Body
type captureResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
	truncated  bool
	bypass     bool
}

func (w *captureResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	if !w.bypass && w.body.Len() < 8192 {
		available := 8192 - w.body.Len()
		if len(b) > available {
			w.body.Write(b[:available])
			w.truncated = true
		} else {
			w.body.Write(b)
		}
	}
	return w.ResponseWriter.Write(b)
}

// 支援 Flush 介面以利 Streaming 與 SSE
func (w *captureResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// 支援 Hijack 介面以利 WebSocket
func (w *captureResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("webserver does not support hijacking")
}

// recordStaticLog 用於記錄靜態目錄代理的日誌
func (e *Engine) recordStaticLog(start time.Time, r *http.Request, rule *RouteRule, statusCode int, note string) {
	reqHeaders := make(map[string]string)
	for k, vv := range r.Header {
		reqHeaders[k] = strings.Join(vv, ", ")
	}
	logEntry := &RequestLog{
		ID:         uuid.New().String(),
		Timestamp:  start,
		Method:     r.Method,
		Path:       r.URL.Path,
		RuleSource: rule.Source,
		TargetURL:  rule.Target,
		StatusCode: statusCode,
		LatencyMs:  time.Since(start).Milliseconds(),
		ReqHeaders: reqHeaders,
		ReqBody:    "[Static Route Request]",
		RespHeaders: map[string]string{
			"Content-Type": "text/html; charset=utf-8",
		},
		RespBody:      note,
		ReqBodyTrunc:  false,
		RespBodyTrunc: false,
	}
	e.logManager.AddLog(logEntry)
}

// ServeHTTP 攔截所有進來的請求，並根據 Manager 的路由規則進行轉發
func (e *Engine) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.Host
	path := r.URL.Path
	start := time.Now()

	// 1. 去 Manager 尋找匹配的規則
	rule, found := e.manager.Match(host, path)

	// 2. 如果沒找到，或該規則目前被暫停，回傳 404
	if !found || !rule.Active {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("404 Not Found - Custom Reverse Proxy"))
		return
	}

	// 2.5 自動處理 CORS 預檢請求 (OPTIONS)
	if r.Method == "OPTIONS" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 包裝 ResponseWriter，以便在後面記錄日誌
	isWebSocket := strings.ToLower(r.Header.Get("Upgrade")) == "websocket" ||
		strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade")

	cWriter := &captureResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		bypass:         isWebSocket, // WebSocket 請求自動 Bypass Body 快取
	}

	// 3. 處理靜態目錄路由類型
	if rule.Type == RouteTypeStatic {
		// 取得相對路徑
		relPath := strings.TrimPrefix(r.URL.Path, rule.Source)
		if !strings.HasPrefix(relPath, "/") {
			relPath = "/" + relPath
		}
		// 本機目標實體路徑
		localFilePath := filepath.Join(rule.Target, relPath)

		// 檢查本地檔案是否存在
		info, err := os.Stat(localFilePath)

		// SPA Fallback 機制：如果是 HTML 請求，且本地檔案不存在或為目錄（但目錄下沒有 index.html）
		isHTMLRequest := strings.Contains(r.Header.Get("Accept"), "text/html")
		if (os.IsNotExist(err) || (err == nil && info.IsDir())) && isHTMLRequest {
			fallbackPath := filepath.Join(rule.Target, "index.html")
			if _, errIndex := os.Stat(fallbackPath); errIndex == nil {
				http.ServeFile(cWriter, r, fallbackPath)
				e.recordStaticLog(start, r, rule, http.StatusOK, "[SPA Fallback to index.html]")
				return
			}
		}

		if err != nil {
			cWriter.WriteHeader(http.StatusNotFound)
			cWriter.Write([]byte("404 Not Found - Local static file not found"))
			e.recordStaticLog(start, r, rule, http.StatusNotFound, "[Not Found]")
			return
		}

		// 使用 http.FileServer 服務該目錄
		fs := http.StripPrefix(rule.Source, http.FileServer(http.Dir(rule.Target)))
		fs.ServeHTTP(cWriter, r)
		e.recordStaticLog(start, r, rule, cWriter.statusCode, "[Served Local File]")
		return
	}

	// 4. 解析目標 URL
	targetURL, err := url.Parse(rule.Target)
	if err != nil {
		log.Printf("目標 URL 解析失敗: %v\n", err)
		cWriter.WriteHeader(http.StatusInternalServerError)
		cWriter.Write([]byte("500 Internal Server Error - Invalid Target URL"))
		return
	}

	// 5. 攔截並讀取 Request Body（完整讀取以確保後端轉發不截斷；Log 記錄只保留前 8KB，大檔案/WebSocket 自動跳過）
	var reqBodyStr string
	var reqBodyTrunc bool
	if !isWebSocket && r.ContentLength < 1024*1024 && r.Body != nil {
		// 完整讀取整個 Body，避免截斷轉發內容
		bodyBytes, errBytes := io.ReadAll(r.Body)
		if errBytes == nil {
			if len(bodyBytes) > 8192 {
				reqBodyStr = string(bodyBytes[:8192])
				reqBodyTrunc = true
			} else {
				reqBodyStr = string(bodyBytes)
			}
			// 將完整的 Body 重新包裝放回，確保後續轉發給後端時是完整的
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	} else if isWebSocket {
		reqBodyStr = "[WebSocket Connection Bypass]"
	} else if r.ContentLength >= 1024*1024 {
		reqBodyStr = "[Large Request Body Bypass]"
	}

	// 6. 建立反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = e.transport

	// 修改原請求設定
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		if rule.Type == RouteTypePath {
			req.URL.Path = strings.TrimPrefix(req.URL.Path, rule.Source)
			if !strings.HasPrefix(req.URL.Path, "/") {
				req.URL.Path = "/" + req.URL.Path
			}
			req.Header.Set("X-Forwarded-Prefix", rule.Source)
		}

		originalDirector(req)

		// 注入自訂 Headers
		for k, v := range rule.Headers {
			if k != "" {
				req.Header.Set(k, v)
			}
		}

		// 保留原始的 Host Header
		if isLocalTarget(targetURL.Host) {
			req.Host = host
		} else {
			req.Host = targetURL.Host
		}

		// 確保轉發標頭被正確寫入
		req.Header.Set("X-Forwarded-Host", host)
		if e.useTLS {
			req.Header.Set("X-Forwarded-Proto", "https")
		} else {
			req.Header.Set("X-Forwarded-Proto", "http")
		}

		if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			req.Header.Set("X-Real-IP", clientIP)
			if prior := req.Header.Get("X-Forwarded-For"); prior != "" {
				clientIP = prior + ", " + clientIP
			}
			req.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	// 修改回應內容
	proxy.ModifyResponse = func(resp *http.Response) error {
		// 動態決定是否 Bypass Response Body 讀取
		contentType := resp.Header.Get("Content-Type")
		contentLength := resp.ContentLength

		if strings.Contains(contentType, "text/event-stream") ||
			contentLength > 100*1024 ||
			(!strings.HasPrefix(contentType, "text/") &&
				!strings.Contains(contentType, "json") &&
				!strings.Contains(contentType, "xml") &&
				!strings.Contains(contentType, "javascript")) {
			cWriter.bypass = true
		}

		// 強制加上 CORS 標頭，解決前端開發常見的跨域問題
		resp.Header.Set("Access-Control-Allow-Origin", "*")
		resp.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		resp.Header.Set("Access-Control-Allow-Headers", "*")

		if rule.Type == RouteTypePath {
			// 修正 Location 標頭 (Redirect)
			location := resp.Header.Get("Location")
			if location != "" {
				prefix := strings.TrimSuffix(rule.Source, "/")
				if strings.HasPrefix(location, "/") && !strings.HasPrefix(location, prefix+"/") && location != prefix {
					resp.Header.Set("Location", prefix+location)
				}
			}

			// 修正 Set-Cookie 的 Path 屬性
			cookies := resp.Header.Values("Set-Cookie")
			if len(cookies) > 0 {
				resp.Header.Del("Set-Cookie")
				for _, cookie := range cookies {
					newCookie := strings.Replace(cookie, "Path=/", "Path="+rule.Source, 1)
					resp.Header.Add("Set-Cookie", newCookie)
				}
			}
		}
		return nil
	}

	// 錯誤處理：連線後端失敗時回傳 502
	proxy.ErrorHandler = func(writer http.ResponseWriter, req *http.Request, err error) {
		log.Printf("代理轉發失敗 [%s -> %s]: %v", req.URL.Path, rule.Target, err)
		// 使用 ErrorHandler 自身的 writer 參數（語意正確），而非閉包捕獲的 cWriter
		writer.WriteHeader(http.StatusBadGateway)
		writer.Write([]byte("502 Bad Gateway - Target server is unreachable"))
	}

	// 7. 執行轉發
	proxy.ServeHTTP(cWriter, r)

	// 8. 紀錄日誌
	latency := time.Since(start).Milliseconds()
	reqHeaders := make(map[string]string)
	for k, vv := range r.Header {
		reqHeaders[k] = strings.Join(vv, ", ")
	}
	respHeaders := make(map[string]string)
	for k, vv := range cWriter.Header() {
		respHeaders[k] = strings.Join(vv, ", ")
	}

	respBodyStr := ""
	respBodyTrunc := cWriter.truncated
	if cWriter.bypass {
		if isWebSocket {
			respBodyStr = "[WebSocket Upgraded]"
		} else {
			respBodyStr = "[Binary/Stream/Large File Bypass]"
		}
	} else {
		respBodyStr = cWriter.body.String()
	}

	logEntry := &RequestLog{
		ID:            uuid.New().String(),
		Timestamp:     start,
		Method:        r.Method,
		Path:          r.URL.Path,
		RuleSource:    rule.Source,
		TargetURL:     rule.Target,
		StatusCode:    cWriter.statusCode,
		LatencyMs:     latency,
		ReqHeaders:    reqHeaders,
		ReqBody:       reqBodyStr,
		RespHeaders:   respHeaders,
		RespBody:      respBodyStr,
		ReqBodyTrunc:  reqBodyTrunc,
		RespBodyTrunc: respBodyTrunc,
	}
	e.logManager.AddLog(logEntry)
}

func isLocalTarget(host string) bool {
	// 簡易判斷是否為本地目標
	return strings.Contains(host, "localhost") || strings.Contains(host, "127.0.0.1")
}
