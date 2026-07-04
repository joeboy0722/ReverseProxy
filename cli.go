package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"reverse-proxy/backend/proxy"
)

// runCLI 啟動命令列介面（支援互動模式與直接指令執行）
func runCLI(app *App, initArgs []string) {
	if len(initArgs) > 0 {
		// 執行初始指令
		cmd := strings.ToLower(initArgs[0])
		switch cmd {
		case "start":
			handleStart(app, initArgs[1:])
			// 不 return，繼續流向互動模式

		case "help":
			printHelp()
			return

		case "status":
			printStatus(app)
			return

		case "list", "ls":
			handleList(app)
			return

		case "add":
			handleSecuredAdd(app, initArgs[1:])
			return

		case "delete", "rm":
			handleDelete(app, initArgs[1:])
			return

		case "toggle":
			handleToggle(app, initArgs[1:])
			return

		case "config":
			handleConfig(app, initArgs[1:])
			return

		case "logs":
			handleLogs(app)
			return

		case "clear-logs":
			app.ClearLogs()
			fmt.Println("記憶體中的請求日誌已全部清除。")
			return

		case "cert":
			handleCert(app, initArgs[1:])
			return

		case "show-cert":
			handleShowCert(app)
			return

		case "nav-config":
			handleNavConfig(app, initArgs[1:])
			return

		case "show-nav":
			handleShowNav(app)
			return

		case "exit", "quit":
			fmt.Println("正在停止伺服器並退出程式...")
			_ = app.StopServer()
			return

		default:
			fmt.Printf("未知的快速指令: %s，或該指令不支援非互動式模式下直接運行。\n", initArgs[0])
			printHelp()
			return
		}
		fmt.Println("----------------------------------------------------------")
	}

	// 互動模式的開頭提示
	fmt.Println("==========================================================")
	fmt.Println("   反向代理伺服器 (Reverse Proxy) - 終端機控制台互動模式   ")
	fmt.Println("==========================================================")
	fmt.Println("當前伺服器狀態:")
	printStatus(app)
	fmt.Println("----------------------------------------------------------")
	fmt.Println("請輸入指令控制系統。輸入 'help' 取得指令說明列表。")
	fmt.Println("輸入 'exit' 或 'quit' 可以停止伺服器並關閉程式。")
	fmt.Println("==========================================================")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("proxy> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		args := parseCommandLine(line)
		if len(args) == 0 {
			continue
		}

		cmd := strings.ToLower(args[0])
		switch cmd {
		case "exit", "quit":
			fmt.Println("正在停止伺服器並退出程式...")
			_ = app.StopServer()
			return

		case "help":
			printHelp()

		case "status":
			printStatus(app)

		case "start":
			handleStart(app, args[1:])

		case "stop":
			handleStop(app)

		case "list", "ls":
			handleList(app)

		case "add":
			handleSecuredAdd(app, args[1:])

		case "delete", "rm":
			handleDelete(app, args[1:])

		case "toggle":
			handleToggle(app, args[1:])

		case "config":
			handleConfig(app, args[1:])

		case "logs":
			handleLogs(app)

		case "clear-logs":
			app.ClearLogs()
			fmt.Println("記憶體中的請求日誌已全部清除。")

		case "cert":
			handleCert(app, args[1:])

		case "show-cert":
			handleShowCert(app)

		case "nav-config":
			handleNavConfig(app, args[1:])

		case "show-nav":
			handleShowNav(app)

		case "tail":
			handleTail(app, scanner)

		default:
			fmt.Printf("未知指令: %s。輸入 'help' 查看可用指令。\n", cmd)
		}
	}
}

// parseCommandLine 解析含雙引號參數的命令列引數
func parseCommandLine(line string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for i := 0; i < len(line); i++ {
		r := line[i]
		if r == '"' {
			inQuotes = !inQuotes
			continue
		}
		if r == ' ' && !inQuotes {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteByte(r)
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func printHelp() {
	fmt.Println("\n可用指令列表：")
	fmt.Println("  help                                              - 顯示指令說明列表")
	fmt.Println("  status                                            - 查詢伺服器運行狀態")
	fmt.Println("  start [addr] [port] [useTLS]                      - 啟動伺服器 (預設 0.0.0.0 8080 false)")
	fmt.Println("  stop                                              - 停止伺服器")
	fmt.Println("  list / ls                                         - 列出所有路由轉發規則")
	fmt.Println("  add <source> <type> <target>                      - 新增規則 (type: host, path, static)")
	fmt.Println("  delete / rm <id>                                  - 刪除指定規則")
	fmt.Println("  toggle <id> <true/false>                          - 啟用或暫停指定規則")
	fmt.Println("  config <id> <keepPrefix:t/f> <injectBase:t/f> \\")
	fmt.Println("         <redirectSlash:t/f> <showInIndex:t/f> \\")
	fmt.Println("         \"[title]\"                                    - 修改指定規則的詳細設定")
	fmt.Println("  logs                                              - 列出最新 20 筆請求日誌")
	fmt.Println("  clear-logs                                        - 清除記憶體日誌")
	fmt.Println("  tail                                              - 進入即時日誌流 (按 Enter 退出監控)")
	fmt.Println("  cert <certPath> <keyPath>                         - 設定 SSL 憑證與金鑰路徑")
	fmt.Println("  show-cert                                         - 顯示當前憑證設定")
	fmt.Println("  nav-config \"[title]\" \"[subtitle]\" <themeColor>    - 設定導航首頁標題、副標題與主題色")
	fmt.Println("  show-nav                                          - 顯示當前導航首頁設定")
	fmt.Println("  exit / quit                                       - 退出 CLI 模式")
	fmt.Println()
}

func printStatus(app *App) {
	status := app.GetServerStatus()
	if status.IsRunning {
		fmt.Printf("● 狀態: 運行中 (Running)\n")
		fmt.Printf("  綁定位址: %s\n", status.BindAddr)
		fmt.Printf("  通訊埠: %d\n", status.Port)
	} else {
		fmt.Println("○ 狀態: 已停止 (Stopped)")
	}
}

func handleStart(app *App, args []string) {
	addr := "0.0.0.0"
	port := 8080
	useTLS := false

	if len(args) > 0 {
		addr = args[0]
	}
	if len(args) > 1 {
		p, err := strconv.Atoi(args[1])
		if err == nil {
			port = p
		}
	}
	if len(args) > 2 {
		useTLS = strings.ToLower(args[2]) == "true"
	}

	fmt.Printf("正在啟動伺服器於 %s:%d (HTTPS: %t)...\n", addr, port, useTLS)
	err := app.StartServer(addr, port, useTLS)
	if err != nil {
		fmt.Printf("啟動失敗: %v\n", err)
	} else {
		fmt.Println("伺服器啟動成功！")
	}
}

func handleStop(app *App) {
	fmt.Println("正在停止伺服器...")
	err := app.StopServer()
	if err != nil {
		fmt.Printf("停止失敗: %v\n", err)
	} else {
		fmt.Println("伺服器已停止。")
	}
}

func handleList(app *App) {
	rules := app.GetRules()
	if len(rules) == 0 {
		fmt.Println("目前沒有設定任何轉發規則。")
		return
	}

	fmt.Println("\n+--------------------------------------+---------------------+---------+--------------------------------+--------+---------+----------------------+")
	fmt.Println("| ID                                   | Source              | Type    | Target                         | Active | Healthy | ShowTitle            |")
	fmt.Println("+--------------------------------------+---------------------+---------+--------------------------------+--------+---------+----------------------+")
	for _, r := range rules {
		// 截斷過長欄位以利排版
		title := r.Title
		if len(title) > 20 {
			title = title[:17] + "..."
		}
		target := r.Target
		if len(target) > 30 {
			target = target[:27] + "..."
		}
		source := r.Source
		if len(source) > 19 {
			source = source[:16] + "..."
		}
		fmt.Printf("| %-36s | %-19s | %-7s | %-30s | %-6t | %-7t | %-20s |\n",
			r.ID, source, r.Type, target, r.Active, r.Healthy, title)
	}
	fmt.Println("+--------------------------------------+---------------------+---------+--------------------------------+--------+---------+----------------------+")
	fmt.Println()
}

func handleSecuredAdd(app *App, args []string) {
	if len(args) < 3 {
		fmt.Println("錯誤：指令參數不足。用法：add <source> <type> <target>")
		return
	}
	source := args[0]
	t := args[1]
	target := args[2]

	id, err := app.AddRule(source, t, target)
	if err != nil {
		fmt.Printf("新增規則失敗: %v\n", err)
	} else {
		fmt.Printf("規則新增成功！ID: %s\n", id)
	}
}

func handleDelete(app *App, args []string) {
	if len(args) < 1 {
		fmt.Println("錯誤：請指定要刪除的規則 ID。用法：delete <id>")
		return
	}
	id := args[0]
	err := app.DeleteRule(id)
	if err != nil {
		fmt.Printf("刪除規則失敗: %v\n", err)
	} else {
		fmt.Println("規則已成功刪除。")
	}
}

func handleToggle(app *App, args []string) {
	if len(args) < 2 {
		fmt.Println("錯誤：請指定規則 ID 與啟用狀態。用法：toggle <id> <true/false>")
		return
	}
	id := args[0]
	active := strings.ToLower(args[1]) == "true"

	err := app.ToggleRule(id, active)
	if err != nil {
		fmt.Printf("切換狀態失敗: %v\n", err)
	} else {
		fmt.Printf("規則已成功 %s。\n", map[bool]string{true: "啟用", false: "停用"}[active])
	}
}

func handleConfig(app *App, args []string) {
	if len(args) < 5 {
		fmt.Println("錯誤：指令參數不足。用法：config <id> <keepPrefix:t/f> <injectBase:t/f> <redirectSlash:t/f> <showInIndex:t/f> \"[title]\"")
		return
	}
	id := args[0]
	keepPrefix := strings.ToLower(args[1]) == "t" || strings.ToLower(args[1]) == "true"
	injectBase := strings.ToLower(args[2]) == "t" || strings.ToLower(args[2]) == "true"
	redirectSlash := strings.ToLower(args[3]) == "t" || strings.ToLower(args[3]) == "true"
	showInIndex := strings.ToLower(args[4]) == "t" || strings.ToLower(args[4]) == "true"
	
	title := ""
	if len(args) > 5 {
		title = args[5]
	}

	err := app.UpdateRuleConfig(id, keepPrefix, injectBase, redirectSlash, true, "", showInIndex, title)
	if err != nil {
		fmt.Printf("更新規則代理設定失敗: %v\n", err)
	} else {
		fmt.Println("規則代理設定更新成功！")
	}
}

func handleLogs(app *App) {
	logs := app.GetLogs()
	if len(logs) == 0 {
		fmt.Println("目前無請求日誌記錄。")
		return
	}

	limit := 20
	if len(logs) < limit {
		limit = len(logs)
	}

	fmt.Println("\n最新 20 筆代理請求日誌:")
	for i := 0; i < limit; i++ {
		l := logs[i]
		fmt.Printf("[%s] %d | %s | %s -> %s (%dms)\n",
			l.Timestamp.Format("15:04:05.000"),
			l.StatusCode,
			l.Method,
			l.Path,
			l.TargetURL,
			l.LatencyMs,
		)
	}
	fmt.Println()
}

func handleCert(app *App, args []string) {
	if len(args) < 2 {
		fmt.Println("錯誤：請指定憑證與金鑰路徑。用法：cert <certPath> <keyPath>")
		return
	}
	certPath := args[0]
	keyPath := args[1]

	err := app.SetCustomCert(certPath, keyPath)
	if err != nil {
		fmt.Printf("設定憑證失敗: %v\n", err)
	} else {
		fmt.Println("自訂 SSL 憑證設定成功並已啟用。")
	}
}

func handleShowCert(app *App) {
	cert := app.GetCustomCert()
	if cert.CertPath == "" && cert.KeyPath == "" {
		fmt.Println("當前未使用自訂憑證 (使用 Wails 系統預設自簽憑證)。")
	} else {
		fmt.Printf("憑證檔案路徑: %s\n", cert.CertPath)
		fmt.Printf("金鑰檔案路徑: %s\n", cert.KeyPath)
	}
}

func handleNavConfig(app *App, args []string) {
	if len(args) < 3 {
		fmt.Println("錯誤：參數不足。用法：nav-config \"[title]\" \"[subtitle]\" <themeColor>")
		return
	}
	title := args[0]
	subtitle := args[1]
	color := args[2]

	err := app.SaveNavConfig(title, subtitle, color)
	if err != nil {
		fmt.Printf("儲存導航設定失敗: %v\n", err)
	} else {
		fmt.Println("導航首頁自訂設定已成功儲存。")
	}
}

func handleShowNav(app *App) {
	nav := app.GetNavConfig()
	fmt.Printf("導覽首頁標題: %s\n", nav.NavTitle)
	fmt.Printf("導覽首頁副標標題: %s\n", nav.NavSubtitle)
	fmt.Printf("主題配色代碼: %s\n", nav.ThemeColor)
}

func handleTail(app *App, scanner *bufio.Scanner) {
	fmt.Println("=== 進入即時日誌流追蹤模式 (輸入 Enter 鍵可退出追蹤) ===")
	
	// 註冊日誌監聽器
	app.RegisterLogListener(func(l *proxy.RequestLog) {
		fmt.Printf("[%s] %d | %s | %s -> %s (%dms)\n",
			l.Timestamp.Format("15:04:05.000"),
			l.StatusCode,
			l.Method,
			l.Path,
			l.TargetURL,
			l.LatencyMs,
		)
	})

	// 等待使用者按 Enter 退出
	scanner.Scan()

	// 註銷監聽器
	app.UnregisterLogListener()
	fmt.Println("=== 已退出即時日誌流追蹤 ===")
}
