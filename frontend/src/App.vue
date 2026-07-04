<script setup>
import { ref, onMounted, computed } from 'vue'
import { 
  GetServerStatus, StartServer, StopServer, 
  GetRules, AddRule, ToggleRule, DeleteRule,
  GetLogs, ClearLogs, GetCustomCert, SetCustomCert,
  SelectDirectory, SelectFile, UpdateRuleHeaders, UpdateRuleConfig,
  GetNavConfig, SaveNavConfig
} from '../wailsjs/go/main/App'
import { EventsOn } from '../wailsjs/runtime/runtime'

// 伺服器狀態
const serverRunning = ref(false)
const bindAddr = ref('0.0.0.0')
const port = ref(8080)
const useTLS = ref(false)
const statusMessage = ref('')

// 規則狀態
const rules = ref([])
const newSource = ref('')
const newRouteType = ref('path') // 'host', 'path', or 'static'
const newTarget = ref('')

// 自訂憑證狀態
const customCertPath = ref('')
const customKeyPath = ref('')
const showCertPanel = ref(false)

// 自訂首頁導覽狀態
const showNavPanel = ref(false)
const navTitle = ref('')
const navSubtitle = ref('')
const themeColor = ref('#6366f1')

// 日誌狀態
const logs = ref([])
const activeLogTab = ref('all') // 'all', 'error', 'static'
const selectedLog = ref(null) // 用於日誌詳情 Modal
const showLogModal = ref(false)

// Headers 彈窗狀態
const editingRule = ref(null)
const editingHeaders = ref([]) // [{ key: '', value: '' }]
const showHeadersModal = ref(false)

// 初始化
onMounted(async () => {
  await fetchStatus()
  await fetchRules()
  await fetchCertConfig()
  await fetchNavConfig()
  await fetchLogs()
  
  // 訂閱即時日誌事件
  EventsOn('log:new', (log) => {
    logs.value.unshift(log)
    if (logs.value.length > 200) {
      logs.value.pop()
    }
  })

  EventsOn('log:cleared', () => {
    logs.value = []
  })
  
  // 定期刷新規則列表以獲取健康狀態
  setInterval(async () => {
    await fetchRules()
  }, 5000)
})

// 取回資料
async function fetchStatus() {
  try {
    const status = await GetServerStatus()
    serverRunning.value = status.isRunning
    if (status.isRunning) {
      bindAddr.value = status.bindAddr
      port.value = status.port
      statusMessage.value = `Server running on ${useTLS.value ? 'https' : 'http'}://${status.bindAddr}:${status.port}`
    } else {
      statusMessage.value = 'Server is currently stopped'
    }
  } catch (err) {
    console.error(err)
  }
}

async function fetchCertConfig() {
  try {
    const cert = await GetCustomCert()
    customCertPath.value = cert.certPath || ''
    customKeyPath.value = cert.keyPath || ''
  } catch (err) {
    console.error(err)
  }
}

async function fetchNavConfig() {
  try {
    const nav = await GetNavConfig()
    navTitle.value = nav.navTitle || ''
    navSubtitle.value = nav.navSubtitle || ''
    themeColor.value = nav.themeColor || '#6366f1'
  } catch (err) {
    console.error(err)
  }
}

async function handleApplyNavConfig() {
  try {
    await SaveNavConfig(navTitle.value.trim(), navSubtitle.value.trim(), themeColor.value.trim())
    alert("Homepage Navigation config saved successfully!")
  } catch (err) {
    alert("Failed to save homepage config: " + err)
  }
}

async function fetchRules() {
  try {
    rules.value = await GetRules() || []
  } catch (err) {
    console.error(err)
  }
}

async function fetchLogs() {
  try {
    logs.value = await GetLogs() || []
  } catch (err) {
    console.error(err)
  }
}

// 伺服器控制
async function handleStartServer() {
  try {
    await StartServer(bindAddr.value, Number(port.value), useTLS.value)
    await fetchStatus()
  } catch (err) {
    alert("Failed to start server: " + err)
  }
}

async function handleStopServer() {
  try {
    await StopServer()
    await fetchStatus()
  } catch (err) {
    alert("Failed to stop server: " + err)
  }
}

// 規則管理
async function handleAddRule() {
  if (!newSource.value || !newTarget.value) {
    alert("Please fill in both Source and Target")
    return
  }
  
  let t = newTarget.value.trim()
  // 只有非 static 類型才自動補上 http
  if (newRouteType.value !== 'static' && !t.startsWith('http') && !t.startsWith('https')) {
      t = 'http://' + t
  }

  try {
    await AddRule(newSource.value.trim(), newRouteType.value, t)
    newSource.value = ''
    newTarget.value = ''
    await fetchRules()
  } catch (err) {
    alert("Failed to add rule: " + err)
  }
}

async function handleToggleRule(id, currentStatus) {
  try {
    await ToggleRule(id, !currentStatus)
    await fetchRules()
  } catch (err) {
    alert("Failed to toggle rule: " + err)
    await fetchRules()
  }
}

async function handleDeleteRule(id) {
  if (!confirm("Are you sure you want to delete this rule?")) return
  try {
    await DeleteRule(id)
    await fetchRules()
  } catch (err) {
    alert("Failed to delete rule: " + err)
  }
}

// 目錄與檔案選取器
async function handleBrowseDirectory() {
  try {
    const dir = await SelectDirectory()
    if (dir) {
      newTarget.value = dir
    }
  } catch (err) {
    console.error(err)
  }
}

async function handleBrowseCert() {
  try {
    const file = await SelectFile("SSL Certificate (*.crt, *.pem)", "*.crt;*.pem;*.*")
    if (file) {
      customCertPath.value = file
    }
  } catch (err) {
    console.error(err)
  }
}

async function handleBrowseKey() {
  try {
    const file = await SelectFile("Private Key (*.key, *.pem)", "*.key;*.pem;*.*")
    if (file) {
      customKeyPath.value = file
    }
  } catch (err) {
    console.error(err)
  }
}

// 儲存憑證
async function handleApplyCert() {
  try {
    await SetCustomCert(customCertPath.value.trim(), customKeyPath.value.trim())
    alert("TLS Certificate config saved and applied successfully!")
  } catch (err) {
    alert("Failed to apply certificate: " + err)
  }
}

async function handleClearCert() {
  try {
    await SetCustomCert("", "")
    customCertPath.value = ""
    customKeyPath.value = ""
    alert("Custom TLS Certificate removed. Fallback to self-signed cert.")
  } catch (err) {
    alert("Failed to clear certificate: " + err)
  }
}

// Headers Modal 編輯 (升級為規則設定)
function openHeadersModal(rule) {
  editingRule.value = { ...rule } // 使用淺拷貝以避免即時修改清單資料
  const hdrs = []
  if (rule.headers) {
    for (const [k, v] of Object.entries(rule.headers)) {
      hdrs.push({ key: k, value: v })
    }
  }
  if (hdrs.length === 0) {
    hdrs.push({ key: '', value: '' })
  }
  editingHeaders.value = hdrs
  showHeadersModal.value = true
}

// 新增一行 Header Row
function addHeaderRow() {
  editingHeaders.value.push({ key: '', value: '' })
}

// 刪除一行 Header Row
function removeHeaderRow(index) {
  editingHeaders.value.splice(index, 1)
  if (editingHeaders.value.length === 0) {
    editingHeaders.value.push({ key: '', value: '' })
  }
}

async function handleSaveHeaders() {
  const headersMap = {}
  for (const h of editingHeaders.value) {
    const k = h.key.trim()
    const v = h.value.trim()
    if (k) {
      headersMap[k] = v
    }
  }
  try {
    await UpdateRuleHeaders(editingRule.value.id, headersMap)
    // 更新代理設定（包含首頁跳轉與健康檢查設定）
    await UpdateRuleConfig(
      editingRule.value.id,
      !!editingRule.value.keepPrefix,
      !!editingRule.value.injectBase,
      !!editingRule.value.redirectSlash,
      editingRule.value.healthCheckEnabled !== false,
      editingRule.value.healthCheckPath || '',
      !!editingRule.value.showInIndex,
      editingRule.value.title || ''
    )
    showHeadersModal.value = false
    await fetchRules()
  } catch (err) {
    alert("Failed to save settings: " + err)
  }
}

// 日誌處理
async function handleClearLogs() {
  try {
    await ClearLogs()
  } catch (err) {
    console.error(err)
  }
}

function showLogDetail(log) {
  selectedLog.value = log
  showLogModal.value = true
}

const filteredLogs = computed(() => {
  if (activeLogTab.value === 'error') {
    return logs.value.filter(l => l.statusCode >= 400)
  }
  if (activeLogTab.value === 'static') {
    return logs.value.filter(l => l.reqBody === '[Static Route Request]')
  }
  return logs.value
})

function formatTime(timestamp) {
  if (!timestamp) return ""
  const d = new Date(timestamp)
  return d.toTimeString().split(' ')[0] + '.' + String(d.getMilliseconds()).padStart(3, '0')
}

</script>

<template>
  <main class="min-h-screen text-slate-200 p-8 flex flex-col gap-8 max-w-6xl mx-auto">
    <!-- Header -->
    <header class="flex justify-between items-center border-b border-slate-700 pb-4">
      <div>
        <h1 class="text-4xl font-bold bg-gradient-to-r from-blue-400 via-teal-400 to-indigo-400 bg-clip-text text-transparent">
          Dynamic Reverse Proxy
        </h1>
        <p class="text-slate-400 mt-2">Manage routing rules, static directory servers, custom SSL and view real-time traffic.</p>
      </div>
      <div class="flex gap-3">
        <button @click="showCertPanel = !showCertPanel; if(showCertPanel) showNavPanel = false" 
                class="px-4 py-2 rounded-lg border border-slate-600 hover:border-slate-400 text-sm font-semibold transition-colors flex items-center gap-2"
                :class="{'bg-slate-700': showCertPanel}">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
          </svg>
          SSL Config
        </button>
        <button @click="showNavPanel = !showNavPanel; if(showNavPanel) showCertPanel = false" 
                class="px-4 py-2 rounded-lg border border-slate-600 hover:border-slate-400 text-sm font-semibold transition-colors flex items-center gap-2"
                :class="{'bg-slate-700': showNavPanel}">
          <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
          Homepage Config
        </button>
      </div>
    </header>

    <!-- SSL Config Panel -->
    <transition name="slide">
      <section v-if="showCertPanel" class="bg-slate-800 rounded-xl p-6 shadow-xl border border-slate-700 flex flex-col gap-4">
        <h2 class="text-xl font-semibold text-blue-300">Custom SSL Certificate Config</h2>
        <p class="text-sm text-slate-400">Specify custom SSL credentials to overwrite Wails automatic self-signed configuration. Changes apply instantly without server restart.</p>
        
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="flex flex-col gap-2">
            <label class="text-xs uppercase font-semibold text-slate-400">Certificate File (.crt / .pem)</label>
            <div class="flex gap-2">
              <input v-model="customCertPath" type="text" placeholder="Not specified (Using self-signed certificate)" class="flex-1 bg-slate-900 border border-slate-700 rounded-md px-3 py-2 text-sm outline-none" />
              <button @click="handleBrowseCert" class="px-3 bg-slate-700 hover:bg-slate-600 text-sm rounded-md transition-colors whitespace-nowrap">Browse</button>
            </div>
          </div>
          <div class="flex flex-col gap-2">
            <label class="text-xs uppercase font-semibold text-slate-400">Private Key File (.key / .pem)</label>
            <div class="flex gap-2">
              <input v-model="customKeyPath" type="text" placeholder="Not specified (Using self-signed certificate)" class="flex-1 bg-slate-900 border border-slate-700 rounded-md px-3 py-2 text-sm outline-none" />
              <button @click="handleBrowseKey" class="px-3 bg-slate-700 hover:bg-slate-600 text-sm rounded-md transition-colors whitespace-nowrap">Browse</button>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-2">
          <button @click="handleClearCert" class="px-4 py-2 border border-rose-500/50 hover:bg-rose-500/10 text-rose-400 font-semibold text-sm rounded-md transition-colors">Uninstall Cert</button>
          <button @click="handleApplyCert" class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white font-semibold text-sm rounded-md transition-colors">Apply Cert Config</button>
        </div>
      </section>
    </transition>

    <!-- Homepage Config Panel -->
    <transition name="slide">
      <section v-if="showNavPanel" class="bg-slate-800 rounded-xl p-6 shadow-xl border border-slate-700 flex flex-col gap-4">
        <h2 class="text-xl font-semibold text-blue-300">Homepage Navigation Config</h2>
        <p class="text-sm text-slate-400">自訂代理伺服器根路徑導覽首頁的標題、副標題與主題色彩。</p>
        
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div class="flex flex-col gap-2 md:col-span-1">
            <label class="text-xs uppercase font-semibold text-slate-400">網頁標題 (Title)</label>
            <input v-model="navTitle" type="text" placeholder="預設：反向代理服務導航首頁" class="bg-slate-900 border border-slate-700 rounded-md px-3 py-2 text-sm outline-none text-slate-200 focus:border-blue-500" />
          </div>
          <div class="flex flex-col gap-2 md:col-span-1">
            <label class="text-xs uppercase font-semibold text-slate-400">主題色彩 (Theme Color)</label>
            <div class="flex gap-2 items-center">
              <input v-model="themeColor" type="color" class="h-9 w-12 bg-slate-900 border border-slate-700 rounded-md cursor-pointer" />
              <input v-model="themeColor" type="text" placeholder="#6366f1" class="flex-1 bg-slate-900 border border-slate-700 rounded-md px-3 py-2 text-sm outline-none text-slate-200 uppercase font-mono focus:border-blue-500" />
            </div>
          </div>
          <div class="flex flex-col gap-2 md:col-span-3">
            <label class="text-xs uppercase font-semibold text-slate-400">網頁副標題 (Subtitle)</label>
            <input v-model="navSubtitle" type="text" placeholder="輸入網頁頂部顯示的副標題..." class="w-full bg-slate-900 border border-slate-700 rounded-md px-3 py-2 text-sm outline-none text-slate-200 focus:border-blue-500" />
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-2">
          <button @click="handleApplyNavConfig" class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white font-semibold text-sm rounded-md transition-colors">Apply Homepage Config</button>
        </div>
      </section>
    </transition>

    <!-- Server Control -->
    <section class="bg-slate-800 rounded-xl p-6 shadow-lg border border-slate-700 relative overflow-hidden transition-all duration-300"
             :class="{'ring-2 ring-emerald-500/50 bg-emerald-900/10': serverRunning}">
             
      <div class="flex flex-col md:flex-row justify-between items-start md:items-center gap-4">
        <div>
          <h2 class="text-xl font-semibold mb-2 flex items-center gap-2">
            <span class="w-3 h-3 rounded-full" :class="serverRunning ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500'"></span>
            Server Status
          </h2>
          <p class="text-sm text-slate-400">{{ statusMessage }}</p>
        </div>
        
        <div class="flex items-center gap-3 bg-slate-900/50 p-2 border border-slate-700 rounded-lg">
          <select v-model="bindAddr" :disabled="serverRunning" class="bg-slate-800 border-none rounded px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none">
            <option value="0.0.0.0">0.0.0.0 (All Interfaces)</option>
            <option value="127.0.0.1">127.0.0.1 (Local Only)</option>
          </select>
          <span class="text-slate-500">:</span>
          <input type="number" v-model="port" :disabled="serverRunning" class="w-20 bg-slate-800 border-none rounded px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none text-center" />
          
          <label v-if="!serverRunning" class="flex items-center gap-2 px-2 cursor-pointer group">
            <input type="checkbox" v-model="useTLS" class="rounded border-slate-700 bg-slate-800 text-blue-600 focus:ring-blue-500" />
            <span class="text-xs font-semibold text-slate-400 group-hover:text-slate-200 transition-colors">HTTPS</span>
          </label>

          <button v-if="!serverRunning" @click="handleStartServer" class="ml-2 bg-blue-600 hover:bg-blue-500 text-white font-semibold py-2 px-4 rounded-md transition-colors whitespace-nowrap">
            Start Server
          </button>
          <button v-else @click="handleStopServer" class="ml-2 bg-rose-600 hover:bg-rose-500 text-white font-semibold py-2 px-4 rounded-md transition-colors whitespace-nowrap">
            Stop Server
          </button>
        </div>
      </div>
    </section>

    <!-- Add Rule Form -->
    <section class="bg-slate-800 rounded-xl p-6 shadow-lg border border-slate-700">
      <h2 class="text-xl font-semibold mb-4">Add Route Rule</h2>
      
      <form @submit.prevent="handleAddRule" class="flex flex-col md:flex-row gap-4 items-end">
        <div class="flex-1 w-full">
          <label class="block text-xs text-slate-400 mb-1 uppercase tracking-wider font-semibold">Match Type</label>
          <select v-model="newRouteType" class="w-full bg-slate-900 border border-slate-700 rounded-md px-4 py-2.5 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition-colors">
            <option value="path">Path Prefix (e.g. /api)</option>
            <option value="host">Host Domain (e.g. app.local)</option>
            <option value="static">Static Directory (Local Hosting)</option>
          </select>
        </div>
        
        <div class="flex-1 w-full relative">
          <label class="block text-xs text-slate-400 mb-1 uppercase tracking-wider font-semibold">Source Identifier</label>
          <input v-model="newSource" type="text" :placeholder="newRouteType === 'static' ? '/static' : (newRouteType === 'path' ? '/apple' : 'example.local')" class="w-full bg-slate-900 border border-slate-700 rounded-md px-4 py-2.5 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition-colors" />
        </div>

        <div class="flex-none text-slate-500 pb-3 hidden md:block">
          <svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
        </div>

        <div class="flex-1 w-full relative">
          <label class="block text-xs text-slate-400 mb-1 uppercase tracking-wider font-semibold">Target</label>
          <div class="flex gap-2">
            <input v-model="newTarget" type="text" :placeholder="newRouteType === 'static' ? 'D:\\my-project\\dist' : 'http://127.0.0.1:8600'" class="flex-1 bg-slate-900 border border-slate-700 rounded-md px-4 py-2.5 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 outline-none transition-colors" />
            <button v-if="newRouteType === 'static'" type="button" @click="handleBrowseDirectory" class="px-3 bg-slate-700 hover:bg-slate-600 rounded-md transition-colors text-sm font-semibold whitespace-nowrap">Browse</button>
          </div>
        </div>

        <button type="submit" class="w-full md:w-auto bg-indigo-600 hover:bg-indigo-500 text-white font-semibold py-2.5 px-6 rounded-md transition-colors shadow-lg shadow-indigo-900/20 whitespace-nowrap h-[42px]">
          Add Rule
        </button>
      </form>
    </section>

    <!-- Rules List -->
    <section class="relative">
      <h2 class="text-xl font-semibold mb-4">Active Rules</h2>
      
      <div v-if="rules.length === 0" class="flex flex-col items-center justify-center p-12 bg-slate-800/50 rounded-xl border border-slate-700/50 border-dashed">
        <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 text-slate-500 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
        </svg>
        <p class="text-slate-400">No routing rules specified yet.</p>
        <p class="text-slate-500 text-sm mt-1">Add a rule above to start routing traffic.</p>
      </div>

      <div v-else class="grid grid-cols-1 gap-4">
        <div v-for="rule in rules" :key="rule.id" 
             class="bg-slate-800 rounded-lg p-5 border flex flex-col md:flex-row items-start md:items-center justify-between gap-4 transition-all duration-200 hover:border-slate-500"
             :class="rule.active ? 'border-slate-700' : 'border-slate-700/50 opacity-60'">
          
          <div class="flex-1 min-w-0 flex items-center gap-4">
             <div class="px-3 py-1 bg-slate-900 rounded-md font-mono text-xs uppercase border border-slate-700 shrink-0"
                  :class="{'text-emerald-400 border-emerald-800/50 bg-emerald-950/20': rule.type === 'static', 'text-slate-400': rule.type !== 'static'}">
               {{ rule.type }}
             </div>
             
             <div class="min-w-0 truncate font-semibold text-lg text-blue-100">
                {{ rule.source }}
             </div>
             
             <div class="text-slate-500 shrink-0">
               <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12h14"/><path d="m12 5 7 7-7 7"/></svg>
             </div>
             
             <div class="min-w-0 truncate font-mono text-emerald-400 bg-emerald-900/20 px-3 py-1 rounded-md border border-emerald-800/50 flex items-center gap-2">
                <span class="w-2 h-2 rounded-full" :class="rule.healthy ? 'bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.6)]' : 'bg-rose-500 shadow-[0_0_8px_rgba(244,63,94,0.6)]'"></span>
                {{ rule.target }}
             </div>
          </div>

          <div class="flex items-center gap-3 shrink-0">
             <!-- Settings BTN -->
             <button @click="openHeadersModal(rule)" class="px-3 py-1 border border-slate-600 hover:border-slate-400 text-xs rounded transition-colors flex items-center gap-1 font-semibold text-slate-300 hover:text-white">
               <svg xmlns="http://www.w3.org/2000/svg" class="h-3 w-3" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                 <path stroke-linecap="round" stroke-linejoin="round" d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                 <path stroke-linecap="round" stroke-linejoin="round" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
               </svg>
               Settings
             </button>

             <!-- Toggle switch -->
             <label class="relative inline-flex items-center cursor-pointer">
              <input type="checkbox" :checked="rule.active" @click.prevent="handleToggleRule(rule.id, rule.active)" class="sr-only peer">
              <div class="w-11 h-6 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
              <span class="ml-3 text-sm font-medium" :class="rule.active ? 'text-blue-300' : 'text-slate-500'">
                {{ rule.active ? 'Active' : 'Paused' }}
              </span>
            </label>

            <!-- Delete btn -->
            <button @click="handleDeleteRule(rule.id)" class="text-slate-400 hover:text-rose-400 transition-colors p-2 hover:bg-slate-700/50 rounded-full">
               <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/><line x1="10" x2="10" y1="11" y2="17"/><line x1="14" x2="14" y1="11" y2="17"/></svg>
            </button>
          </div>
          
        </div>
      </div>
    </section>

    <!-- Real-time Traffic Logs -->
    <section class="bg-slate-800 rounded-xl p-6 shadow-lg border border-slate-700 flex flex-col gap-4">
      <div class="flex justify-between items-center border-b border-slate-700 pb-3">
        <div class="flex items-center gap-3">
          <h2 class="text-xl font-semibold text-teal-300 flex items-center gap-2">
            <span class="relative flex h-3 w-3">
              <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-teal-400 opacity-75"></span>
              <span class="relative inline-flex rounded-full h-3 w-3 bg-teal-500"></span>
            </span>
            Real-time HTTP Logs
          </h2>
          <!-- Tab Filters -->
          <div class="flex bg-slate-900/60 p-1 rounded-lg border border-slate-700 text-xs font-semibold">
            <button @click="activeLogTab = 'all'" :class="{'bg-slate-700 text-white': activeLogTab === 'all'}" class="px-3 py-1 rounded transition-colors text-slate-400 hover:text-slate-200">All</button>
            <button @click="activeLogTab = 'error'" :class="{'bg-slate-700 text-rose-400': activeLogTab === 'error'}" class="px-3 py-1 rounded transition-colors text-slate-400 hover:text-slate-200">Errors</button>
            <button @click="activeLogTab = 'static'" :class="{'bg-slate-700 text-teal-400': activeLogTab === 'static'}" class="px-3 py-1 rounded transition-colors text-slate-400 hover:text-slate-200">Static</button>
          </div>
        </div>
        <button @click="handleClearLogs" class="text-xs text-rose-400 hover:text-rose-300 font-semibold px-3 py-1.5 border border-rose-500/30 rounded-md transition-colors hover:bg-rose-500/10">Clear Logs</button>
      </div>

      <!-- Logs List -->
      <div class="overflow-y-auto max-h-[300px] border border-slate-900 rounded-lg bg-slate-900/30">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-slate-900/80 text-xs font-semibold text-slate-400 uppercase tracking-wider border-b border-slate-800">
              <th class="p-3">Time</th>
              <th class="p-3">Method</th>
              <th class="p-3">Source Route</th>
              <th class="p-3">Requested Path</th>
              <th class="p-3">Status</th>
              <th class="p-3 text-right">Latency</th>
            </tr>
          </thead>
          <tbody class="text-sm font-mono text-slate-300">
            <tr v-if="filteredLogs.length === 0">
              <td colspan="6" class="p-8 text-center text-slate-500 font-sans">No HTTP traffic recorded yet.</td>
            </tr>
            <tr v-for="log in filteredLogs" :key="log.id" @click="showLogDetail(log)" class="hover:bg-slate-700/30 border-b border-slate-800/50 cursor-pointer transition-colors group">
              <td class="p-3 text-xs text-slate-500">{{ formatTime(log.timestamp) }}</td>
              <td class="p-3 text-xs">
                <span class="px-2 py-0.5 rounded text-[10px] font-bold" 
                      :class="{'bg-blue-500/20 text-blue-400': log.method === 'GET',
                               'bg-emerald-500/20 text-emerald-400': log.method === 'POST',
                               'bg-amber-500/20 text-amber-400': log.method === 'PUT' || log.method === 'PATCH',
                               'bg-rose-500/20 text-rose-400': log.method === 'DELETE'}">
                  {{ log.method }}
                </span>
              </td>
              <td class="p-3 text-xs text-indigo-300 max-w-[150px] truncate">{{ log.ruleSource }}</td>
              <td class="p-3 max-w-[250px] truncate text-slate-400 group-hover:text-slate-200 transition-colors">{{ log.path }}</td>
              <td class="p-3 text-xs">
                <span class="font-bold" 
                      :class="{'text-emerald-400': log.statusCode < 300,
                               'text-blue-300': log.statusCode >= 300 && log.statusCode < 400,
                               'text-rose-400': log.statusCode >= 400}">
                  {{ log.statusCode }}
                </span>
              </td>
              <td class="p-3 text-right text-xs" :class="log.latencyMs > 500 ? 'text-amber-400' : 'text-slate-500'">{{ log.latencyMs }}ms</td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Settings Modal -->
    <div v-if="showHeadersModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex justify-center items-center p-4 z-50 animate-fade-in">
      <div class="bg-slate-800 rounded-xl border border-slate-700 w-full max-w-lg shadow-2xl overflow-hidden flex flex-col max-h-[90vh]">
        <div class="p-5 border-b border-slate-700 flex justify-between items-center bg-slate-900/40 shrink-0">
          <div>
            <h3 class="text-lg font-bold text-blue-300">
              {{ editingRule?.type === 'path' ? 'Proxy Path Settings' : (editingRule?.type === 'host' ? 'Proxy Host Settings' : 'Proxy Rule Settings') }}
            </h3>
            <p class="text-xs text-slate-400 mt-1">
              {{ editingRule?.type === 'path' ? 'Configure options and headers for path-based proxy:' : 'Configure headers for host-based proxy:' }}
              <span class="font-mono text-indigo-300">{{ editingRule?.source }}</span>
            </p>
          </div>
          <button @click="showHeadersModal = false" class="text-slate-400 hover:text-white transition-colors">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>

        <div class="overflow-y-auto flex-1">
          <!-- Host Mode Tips -->
          <div v-if="editingRule && editingRule.type === 'host'" class="p-5 border-b border-slate-700/50 bg-slate-900/10 text-slate-400 text-xs">
            <div class="flex items-start gap-2">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4 text-blue-400 shrink-0 mt-0.5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <div>
                <span class="block font-semibold text-slate-300 mb-0.5">主機代理模式 (Host Proxy Mode)</span>
                主機類型轉發會完整代理對應 Host 的所有流量，不需且不支援進行路徑前綴移除、HTML Base 標籤注入或尾部斜線修正等網頁重寫設定。
              </div>
            </div>
          </div>

          <!-- Homepage Link Options -->
          <div v-if="editingRule" class="p-5 border-b border-slate-700/50 flex flex-col gap-4 bg-slate-900/20">
            <h4 class="text-xs font-bold text-blue-400 uppercase tracking-wider">Homepage Navigation Settings (首頁導覽跳轉設定)</h4>
            
            <div class="flex flex-col gap-2">
              <span class="block text-sm font-semibold text-slate-200">顯示標題名稱 (Display Title)</span>
              <span class="block text-xs text-slate-400">在根目錄導覽網頁上顯示的自訂標題。若留空，將預設顯示來源路徑/網域。</span>
              <input v-model="editingRule.title" type="text" placeholder="例如：我的內部系統" class="w-full bg-slate-900 border border-slate-700 rounded-md px-3 py-1.5 text-sm outline-none focus:border-blue-500 text-slate-300" />
            </div>

            <div class="flex items-center justify-between">
              <div class="pr-2">
                <span class="block text-sm font-semibold text-slate-200">顯示在首頁 (Show on Homepage)</span>
                <span class="block text-xs text-slate-400">是否要將此跳轉連結公開顯示於反向代理的根目錄首頁。</span>
              </div>
              <label class="relative inline-flex items-center cursor-pointer shrink-0 ml-4">
                <input type="checkbox" v-model="editingRule.showInIndex" class="sr-only peer">
                <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>
          </div>

          <!-- Web Options -->
          <div v-if="editingRule && editingRule.type === 'path'" class="p-5 border-b border-slate-700/50 flex flex-col gap-4 bg-slate-900/10">
            <h4 class="text-xs font-bold text-indigo-400 uppercase tracking-wider">Proxy Web Options (網頁代理設定)</h4>
            
            <div class="flex items-center justify-between">
              <div class="pr-2">
                <span class="block text-sm font-semibold text-slate-200">保留來源前綴 (Keep Prefix)</span>
                <span class="block text-xs text-slate-400">若關閉，/web/main 會轉發為後端的 /main；若開啟，則轉發為 /web/main。</span>
              </div>
              <label class="relative inline-flex items-center cursor-pointer shrink-0 ml-4">
                <input type="checkbox" v-model="editingRule.keepPrefix" class="sr-only peer">
                <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>

            <div class="flex items-center justify-between">
              <div class="pr-2">
                <span class="block text-sm font-semibold text-slate-200">注入 Base 標籤 (Inject HTML Base Tag)</span>
                <span class="block text-xs text-slate-400">當後端回傳 HTML 時，自動在 &lt;head&gt; 中注入 &lt;base&gt; 標籤，修正相對資源路徑。</span>
              </div>
              <label class="relative inline-flex items-center cursor-pointer shrink-0 ml-4">
                <input type="checkbox" v-model="editingRule.injectBase" class="sr-only peer">
                <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>

            <div class="flex items-center justify-between">
              <div class="pr-2">
                <span class="block text-sm font-semibold text-slate-200">重導向尾部斜線 (Redirect Slash)</span>
                <span class="block text-xs text-slate-400">當要求無尾斜線的 /web 時，自動重導向到 /web/（API 建議關閉，網頁建議開啟）。</span>
              </div>
              <label class="relative inline-flex items-center cursor-pointer shrink-0 ml-4">
                <input type="checkbox" v-model="editingRule.redirectSlash" class="sr-only peer">
                <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>
          </div>

          <!-- Health Check Options -->
          <div v-if="editingRule && editingRule.type !== 'static'" class="p-5 border-b border-slate-700/50 flex flex-col gap-4 bg-slate-900/5">
            <h4 class="text-xs font-bold text-teal-400 uppercase tracking-wider">Health Check Settings (健康檢查設定)</h4>
            
            <div class="flex items-center justify-between">
              <div class="pr-2">
                <span class="block text-sm font-semibold text-slate-200">啟用健康檢查 (Enable Health Check)</span>
                <span class="block text-xs text-slate-400">是否定期檢測此後端伺服器的存活狀態，若關閉將預設為健康。</span>
              </div>
              <label class="relative inline-flex items-center cursor-pointer shrink-0 ml-4">
                <input type="checkbox" v-model="editingRule.healthCheckEnabled" class="sr-only peer">
                <div class="w-9 h-5 bg-slate-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"></div>
              </label>
            </div>

            <div v-if="editingRule.healthCheckEnabled !== false" class="flex flex-col gap-2">
              <span class="block text-sm font-semibold text-slate-200">自訂健康檢查路徑 (Custom Ping Path)</span>
              <span class="block text-xs text-slate-400">自訂發送檢測請求的子路徑。若為空，則直接 ping 目標主機 (例如 <code>/healthz</code> 或 <code>/api/ping</code>)。</span>
              <input v-model="editingRule.healthCheckPath" type="text" placeholder="e.g. /healthz (預設為空，即直接 ping 目標主機)" class="w-full bg-slate-900 border border-slate-700 rounded-md px-3 py-1.5 text-sm outline-none focus:border-blue-500 font-mono text-slate-300" />
            </div>
          </div>

          <!-- Headers List -->
          <div class="p-5 flex flex-col gap-3">
            <h4 class="text-xs font-bold text-indigo-400 uppercase tracking-wider">Inject Custom Headers (自訂注入標頭)</h4>
            <div v-for="(h, idx) in editingHeaders" :key="idx" class="flex gap-2 items-center">
              <input v-model="h.key" type="text" placeholder="Key (e.g. Authorization)" class="flex-1 bg-slate-900 border border-slate-700 rounded-md px-3 py-1.5 text-sm outline-none focus:border-blue-500 font-mono" />
              <span class="text-slate-600">:</span>
              <input v-model="h.value" type="text" placeholder="Value" class="flex-1 bg-slate-900 border border-slate-700 rounded-md px-3 py-1.5 text-sm outline-none focus:border-blue-500 font-mono" />
              <button @click="removeHeaderRow(idx)" class="text-slate-500 hover:text-rose-400 p-1.5 hover:bg-slate-700/50 rounded transition-all">
                <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" /></svg>
              </button>
            </div>
            <button @click="addHeaderRow" class="mt-2 text-xs text-blue-400 hover:text-blue-300 font-semibold flex items-center gap-1">
              <svg xmlns="http://www.w3.org/2000/svg" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" /></svg>
              Add Header Rule
            </button>
          </div>
        </div>

        <div class="p-5 border-t border-slate-700 flex justify-end gap-3 bg-slate-900/20 shrink-0">
          <button @click="showHeadersModal = false" class="px-4 py-2 border border-slate-600 hover:border-slate-500 text-sm font-semibold rounded-md transition-colors">Cancel</button>
          <button @click="handleSaveHeaders" class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-sm font-semibold rounded-md transition-colors">Save Changes</button>
        </div>
      </div>
    </div>

    <!-- Logs Details Inspector Modal -->
    <div v-if="showLogModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex justify-center items-center p-4 z-50 animate-fade-in">
      <div class="bg-slate-800 rounded-xl border border-slate-700 w-full max-w-3xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]">
        <div class="p-5 border-b border-slate-700 flex justify-between items-center bg-slate-900/40 shrink-0">
          <div>
            <h3 class="text-lg font-bold text-teal-300 flex items-center gap-2">
              <span class="px-2 py-0.5 rounded text-xs font-bold bg-slate-900 border border-slate-700" 
                    :class="{'text-blue-400': selectedLog?.method === 'GET', 'text-emerald-400': selectedLog?.method === 'POST'}">
                {{ selectedLog?.method }}
              </span>
              {{ selectedLog?.path }}
            </h3>
            <p class="text-xs text-slate-400 mt-1">Routed to: <span class="font-mono text-indigo-300">{{ selectedLog?.targetURL }}</span></p>
          </div>
          <button @click="showLogModal = false" class="text-slate-400 hover:text-white transition-colors">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>

        <div class="p-6 overflow-y-auto flex-1 flex flex-col gap-6">
          <!-- Metadata -->
          <div class="grid grid-cols-2 md:grid-cols-4 gap-4 bg-slate-900/40 p-4 border border-slate-700/50 rounded-lg text-sm font-mono">
            <div>
              <span class="block text-xs text-slate-500 uppercase font-semibold font-sans mb-1">Status Code</span>
              <span :class="{'text-emerald-400': selectedLog?.statusCode < 300, 'text-rose-400': selectedLog?.statusCode >= 400}">{{ selectedLog?.statusCode }}</span>
            </div>
            <div>
              <span class="block text-xs text-slate-500 uppercase font-semibold font-sans mb-1">Latency</span>
              <span class="text-indigo-300">{{ selectedLog?.latencyMs }}ms</span>
            </div>
            <div>
              <span class="block text-xs text-slate-500 uppercase font-semibold font-sans mb-1">Matched Rule</span>
              <span class="text-blue-300">{{ selectedLog?.ruleSource }}</span>
            </div>
            <div>
              <span class="block text-xs text-slate-500 uppercase font-semibold font-sans mb-1">Timestamp</span>
              <span class="text-slate-400 font-sans text-xs">{{ new Date(selectedLog?.timestamp).toLocaleString() }}</span>
            </div>
          </div>

          <!-- Request Section -->
          <div>
            <h4 class="text-sm font-bold text-slate-400 border-l-2 border-blue-500 pl-2 mb-3">REQUEST</h4>
            <div class="flex flex-col gap-3">
              <!-- Headers -->
              <details class="bg-slate-900/60 p-3 rounded-lg border border-slate-800 text-xs">
                <summary class="cursor-pointer font-sans font-semibold text-slate-400 hover:text-slate-200">Headers ({{ Object.keys(selectedLog?.reqHeaders || {}).length }})</summary>
                <div class="grid grid-cols-1 gap-1.5 mt-3 font-mono text-slate-300 break-all select-all">
                  <div v-for="(val, key) in selectedLog?.reqHeaders" :key="key" class="flex flex-col md:flex-row md:gap-2">
                    <span class="text-indigo-400 shrink-0">{{ key }}:</span>
                    <span class="text-slate-300">{{ val }}</span>
                  </div>
                </div>
              </details>
              <!-- Body -->
              <div class="bg-slate-900/60 p-3 rounded-lg border border-slate-800">
                <div class="flex justify-between items-center mb-2">
                  <span class="text-xs font-semibold text-slate-400">Body</span>
                  <span v-if="selectedLog?.reqBodyTrunc" class="text-[10px] bg-rose-500/20 text-rose-400 font-semibold px-2 py-0.5 rounded">Truncated (max 8KB)</span>
                </div>
                <pre class="text-xs font-mono bg-slate-950 p-3 rounded border border-slate-900 text-slate-300 overflow-x-auto whitespace-pre-wrap max-h-[150px] break-all select-all">{{ selectedLog?.reqBody || '[Empty]' }}</pre>
              </div>
            </div>
          </div>

          <!-- Response Section -->
          <div>
            <h4 class="text-sm font-bold text-slate-400 border-l-2 border-teal-500 pl-2 mb-3">RESPONSE</h4>
            <div class="flex flex-col gap-3">
              <!-- Headers -->
              <details class="bg-slate-900/60 p-3 rounded-lg border border-slate-800 text-xs">
                <summary class="cursor-pointer font-sans font-semibold text-slate-400 hover:text-slate-200">Headers ({{ Object.keys(selectedLog?.respHeaders || {}).length }})</summary>
                <div class="grid grid-cols-1 gap-1.5 mt-3 font-mono text-slate-300 break-all select-all">
                  <div v-for="(val, key) in selectedLog?.respHeaders" :key="key" class="flex flex-col md:flex-row md:gap-2">
                    <span class="text-teal-400 shrink-0">{{ key }}:</span>
                    <span class="text-slate-300">{{ val }}</span>
                  </div>
                </div>
              </details>
              <!-- Body -->
              <div class="bg-slate-900/60 p-3 rounded-lg border border-slate-800">
                <div class="flex justify-between items-center mb-2">
                  <span class="text-xs font-semibold text-slate-400">Body</span>
                  <span v-if="selectedLog?.respBodyTrunc" class="text-[10px] bg-rose-500/20 text-rose-400 font-semibold px-2 py-0.5 rounded">Truncated (max 8KB)</span>
                </div>
                <pre class="text-xs font-mono bg-slate-950 p-3 rounded border border-slate-900 text-slate-300 overflow-x-auto whitespace-pre-wrap max-h-[150px] break-all select-all">{{ selectedLog?.respBody || '[Empty]' }}</pre>
              </div>
            </div>
          </div>
        </div>

        <div class="p-5 border-t border-slate-700 flex justify-end bg-slate-900/20 shrink-0">
          <button @click="showLogModal = false" class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-sm font-semibold rounded-md transition-colors">Close</button>
        </div>
      </div>
    </div>
  </main>
</template>

<style>
/* 簡單轉場與全域動畫 */
.slide-enter-active, .slide-leave-active {
  transition: all 0.3s ease-out;
}
.slide-enter-from, .slide-leave-to {
  transform: translateY(-10px);
  opacity: 0;
}

@keyframes fadeIn {
  from { opacity: 0; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1); }
}

.animate-fade-in {
  animation: fadeIn 0.2s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}
</style>

<style scoped>
/* Scoped styles kept minimal due to tailwind use */
</style>
