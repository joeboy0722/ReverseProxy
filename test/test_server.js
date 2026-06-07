const http = require('http');

// 讓 Server 1 變成一個帶有「一鍵測試面板」的網頁
const server1 = http.createServer((req, res) => {

    // --- API 端點: 測試大量請求用的延遲回覆 (確保代理可承受併發) ---
    if (req.url === '/api/ping') {
        // 刻意延遲 50ms 模擬真實後端處理
        setTimeout(() => {
            res.writeHead(200, { 'Content-Type': 'application/json' });
            res.end(JSON.stringify({ status: "ok", time: Date.now() }));
        }, 50);
        return;
    }

    // --- 大檔案回傳 ---
    if (req.url === '/api/heavy') {
        const bigString = "0123456789".repeat(500000); // 5MB
        res.writeHead(200, { 'Content-Type': 'text/plain' });
        res.end(bigString);
        return;
    }

    // --- HTML 面板 ---
    res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
    res.end(`
        <!DOCTYPE html>
        <html>
        <head>
            <title>代理測試儀表板</title>
            <style>
                body { background: #1a1a2e; color: #fff; font-family: system-ui; text-align: center; padding: 2rem; }
                .card { background: #16213e; padding: 2rem; border-radius: 12px; margin: 2rem auto; max-width: 600px; box-shadow: 0 10px 30px rgba(0,0,0,0.5); }
                button { background: #e94560; color: white; border: none; padding: 12px 24px; font-size: 1.1rem; border-radius: 6px; cursor: pointer; transition: 0.2s; font-weight:bold; }
                button:hover { background: #ff5773; }
                button:disabled { background: #555; cursor: not-allowed; }
                .stats { display: flex; justify-content: space-around; margin-top: 20px; }
                .stat-box { background: rgba(0,0,0,0.3); padding: 15px; border-radius: 8px; width: 30%; }
                .number { font-size: 2rem; font-weight: bold; color: #4ecca3; margin: 10px 0; }
                #log { background: #0f3460; padding: 10px; border-radius: 8px; max-height: 200px; overflow-y: auto; text-align: left; font-family: monospace; font-size: 0.9rem; margin-top: 20px; }
            </style>
        </head>
        <body>
            <h1>🚀 反向代理效能測試面板</h1>
            <p>請確認您的網址列目前是透過 Reverse Proxy 進入 (如: localhost:8080/apple)</p>
            
            <div class="card">
                <h2>併發壓力測試 (發送 500 個請求)</h2>
                <button id="startBtn" onclick="startTest()">開始打擊測試</button>
                
                <div class="stats">
                    <div class="stat-box">成功數<div class="number" id="s-success">0</div></div>
                    <div class="stat-box">失敗數<div class="number text-red" id="s-error" style="color:#ff5773">0</div></div>
                    <div class="stat-box">耗時<div class="number" id="s-time">0s</div></div>
                </div>

                <div id="log">等待測試開始...</div>
            </div>

            <script>
                async function startTest() {
                    const btn = document.getElementById('startBtn');
                    const logEl = document.getElementById('log');
                    btn.disabled = true;
                    btn.innerText = "測試進行中...";
                    
                    let success = 0;
                    let error = 0;
                    const totalReqs = 500;
                    const startTime = Date.now();
                    
                    // 以這個網頁目前的 URL 為基底去打 API
                    const basePath = window.location.pathname.replace(/\\/$/, "");
                    
                    logEl.innerHTML = "開始發送 500 個併發請求到 " + basePath + "/api/ping ...<br/>";
                    
                    const promises = [];
                    for(let i=0; i<totalReqs; i++) {
                        promises.push(
                            fetch(basePath + '/api/ping')
                            .then(res => { if(res.ok) success++; else error++; })
                            .catch(e => { error++; })
                            .finally(() => {
                                document.getElementById('s-success').innerText = success;
                                document.getElementById('s-error').innerText = error;
                            })
                        );
                    }
                    
                    await Promise.all(promises);
                    const cost = ((Date.now() - startTime) / 1000).toFixed(2);
                    document.getElementById('s-time').innerText = cost + 's';
                    
                    logEl.innerHTML += "✅ 測試完成！總耗時: " + cost + " 秒。<br/>";
                    logEl.innerHTML += "👉 速度參考: 若耗時在 3 秒以內代表代理引擎效能極佳。<br/>";
                    
                    btn.disabled = false;
                    btn.innerText = "重新測試";
                }
            </script>
        </body>
        </html>
    `);
});

server1.listen(3001, () => {
    console.log('面板與測試 API 運行於 Port 3001');
});
