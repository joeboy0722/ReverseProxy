# Custom Reverse Proxy & Developer Server / 自訂反向代理與前端開發伺服器

Bilingual project documentation for a modern, lightweight reverse proxy server built using **Wails v2 (Go + Vue 3)**.

本專案是一個基於 **Wails v2 (Go + Vue 3)** 開發的現代化輕量級反向代理伺服器與靜態網頁伺服器，專為開發人員解決本地跨域、靜態網站託管與多服務整合需求。

<p align="center">
  <img src="image.png" alt="反向代理伺服器主介面" width="750">
</p>

---

## 🚀 Features / 功能特點

### 🌐 Routing & Proxying / 路由與代理轉發
* **Domain Routing (Host-based) / 網域名稱路由**: Route requests by hostname (e.g., `banana.local` -> `127.0.0.1:8080`).
  * 根據網域名稱進行轉發，輕鬆配置本機虛擬主機（e.g., `banana.local` -> `127.0.0.1:8080`）。
* **Path Routing / 路徑前綴路由**: Route requests by URL path prefix (e.g., `/api` -> `127.0.0.1:9000/api`) with auto-strip option.
  * 根據路徑前綴將流量轉發至不同後端服務（如 `/api` -> `127.0.0.1:9000`），支援自動修正 Cookie 路徑與 Location 轉址標頭。
* **Static Directory Serving / 靜態目錄託管**: Serve local folders as static websites (e.g., hosting your SPA app built with Vue/React).
  * 直接將本機目錄作為網頁伺服器執行，並支援 **SPA Fallback (Single Page Application)** 路由機制（自動將未找到的 HTML 請求導向 `index.html`）。

### 🛠️ Developer Tools / 開發輔助功能
* **Auto-CORS Injection / 自動跨域注入**: Automatically injection of CORS headers (`Access-Control-Allow-Origin: *`) to solve frontend local development issues.
  * 自動攔截並答覆 `OPTIONS` 預檢請求，並在響應中注入 CORS 標頭，徹底解決開發時前端請求後端發生的跨域問題。
* **WebSocket & SSE Support / 支援 WebSocket 與串流**: Transparently proxy WebSocket connections and Server-Sent Events (SSE).
  * 完美支援 WebSocket 雙向連線協議（自動進行連線升級）與 Server-Sent Events 伺服器發送事件流。
* **Live Traffic Monitor / 即時請求日誌**: Capture, inspect, and analyze incoming HTTP requests (headers, body up to 8KB, latency, status codes) streaming directly to the UI.
  * 即時錄製所有代理流量，前端介面可直接檢視詳細的 HTTP 請求頭、請求體、響應頭、響應體（大於 8KB 或二進位串流將自動截斷/跳過）以及連線延遲 (Latency)。
* **Health Check Monitor / 後端健康檢查**: Automatically check the status of backend servers and static folder existence every 30 seconds.
  * 每 30 秒自動在背景對所有啟用的代理路徑與目錄進行健康檢查，並將連線健康狀態即時回饋至 UI 介面。
* **Dynamic SSL/TLS Certificate Reloading / 動態 HTTPS 憑證重載**: Configure custom PEM certificates or fall back to an auto-generated self-signed certificate dynamically.
  * 支援設定並隨時重載自訂的 SSL/TLS 憑證；若無設定憑證，伺服器在啟用 HTTPS 時會自動動態產生自簽憑證（Self-Signed Certificate）以支援本地安全加密通訊。
---

## 🖥️ Command Line Interface (CLI) / 命令列互動與指令模式

本專案除了 GUI 視窗介面外，也支援強大的 CLI 互動與指令模式，適用於無頭伺服器（Headless Server）、後台部署與腳本自動化管理。

### How to Start CLI / 如何啟動 CLI 模式
在終端機中，使用 `-cli` 或 `-c` 旗標啟動：
```bash
# 啟動並進入互動控制台
ReverseProxy.exe -cli
# 或簡寫
ReverseProxy.exe -c
```

### Rapid Commands / 快速快捷指令
您可以在啟動時直接附帶指令來快速執行任務：
* **`ReverseProxy.exe -cli start [addr] [port] [useTLS]`**
  * 快速啟動伺服器並**自動進入**互動式 CLI 控制台進行手動管理。
  * 例如：`ReverseProxy.exe -cli start 127.0.0.1 9090`
* **`ReverseProxy.exe -cli list` (或 `ls`)**
  * 列出當前所有路由轉發規則，並在**執行完畢後自動退出程式**（適合指令行腳本整合）。
* **`ReverseProxy.exe -cli status`**
  * 查詢當前代理伺服器的運行狀態並自動退出。
* **`ReverseProxy.exe -cli logs`**
  * 列出記憶體中最新的 20 筆請求日誌並自動退出。
* **`ReverseProxy.exe -cli clear-logs`**
  * 清除記憶體日誌並自動退出。
* **`ReverseProxy.exe -cli show-cert`**
  * 顯示當前的 SSL 憑證路徑設定並自動退出。
* **`ReverseProxy.exe -cli show-nav`**
  * 顯示當前導航首頁自訂設定並自動退出。

> [!IMPORTANT]
> **Windows 打包特別說明 / Build Note for Windows Support**:
> 
> Wails 預設打包時會隱藏 Windows 的主控台視窗。若要支援 CLI 模式，您在 Windows 下打包時**必須加上 `-windowsconsole` 參數**：
> ```bash
> wails build -windowsconsole
> ```
> 如此編譯出來的程式在雙擊啟動 GUI 時仍會自動隱藏 Console，但以 `-cli` 指令啟動時則能完美在終端機中正常顯示與輸入。

---

## 📂 Project Structure / 專案結構

```
ReverseProxy/
├── backend/                  # Go Backend modules / Go 後端程式碼
│   └── proxy/                # Core Proxy logic / 代理伺服器核心
│       ├── config.go         # Configuration manager (proxy_config.json) / 全域憑證設定
│       ├── engine.go         # Core HTTP/HTTPS proxy engine / 反向代理引擎與日誌記錄
│       ├── log_manager.go    # In-memory traffic ring buffer / 記憶體流量日誌管理
│       └── manager.go        # Route rules registry & Health checks / 路由清單與健康檢查
├── frontend/                 # Vue 3 Frontend UI / Vue 3 前端介面
│   ├── src/                  # Components, styles, API interfaces / 組件與頁面邏輯
│   ├── wailsjs/              # Auto-generated JS bindings / Wails 自動生成的 Go-JS 綁定
│   └── package.json          # Node dependencies / 前端套件定義
├── build/                    # App icons & installer assets / 軟體安裝包與圖示資源
├── wails.json                # Wails configuration / Wails 設定檔
├── main.go                   # Go application entrypoint / Go 程式啟動進入點
└── app.go                    # Application lifecycle & Bridge APIs / 視窗生命週期與 API 橋接
```

---

## 🛠️ How to Run / 如何在本機開發與打包

### Prerequisites / 事前準備
1. **Install Go / 安裝 Go**: Make sure Go (v1.20+) is installed on your computer.
2. **Install Node.js / 安裝 Node.js**: Ensure Node.js and npm are installed for building the frontend.
3. **Install Wails CLI / 安裝 Wails 工具**: Install Wails command line tool manually:
   ```bash
   go install github.com/wailsapp/wails/v2/cmd/wails@latest
   ```

### 1. Run in Live Development Mode / 啟動開發環境
This will launch a desktop window with hot-reloading for both Go code and frontend code.
執行以下指令會啟動帶有即時熱重載（Hot Reload）的桌面視窗開發環境：
```bash
wails dev
```

### 2. Build for Production / 打包生產版本
Compile and package the application into a standalone native application (e.g., `.exe` for Windows).
編譯並將程式打包為各平台的獨立原生安裝檔（如 Windows 下的單一執行檔）：
```bash
# 預設打包（Windows 下會隱藏主控台視窗）
wails build

# Windows 下若需要支援 CLI 模式，必須加上 -windowsconsole 參數：
wails build -windowsconsole
```

---

## 📜 License / 授權條款

This project is licensed under the **MIT License**. For more details, see the [LICENSE](file:///d:/ReverseProxy/LICENSE) file.

本專案採用 **MIT 授權條款** 開源，詳情請參閱 [LICENSE](file:///d:/ReverseProxy/LICENSE) 檔案。
