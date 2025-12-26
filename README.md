# 伺服器控制器後端

一個輕量級、自託管的遊戲伺服器管理平台，使用 Go 語言開發，專為私人社群和朋友設計。

## 功能特色

-   **遊戲伺服器管理**：
    -   建立、啟動、停止和配置 **Minecraft**（原版/Fabric）伺服器。
    -   即時控制台存取與指令執行。
    -   伺服器資源監控（CPU/RAM）。
    -   可配置的 `server.properties` 編輯器。
    -   備份管理（世界資料）。
    -   *實驗性支援 Among Us 伺服器。*
-   **使用者系統**：
    -   私人存取控制（無公開註冊）。
    -   基於角色的權限（Root/Admin/User）。
    -   透過 JWT 和 Cookies 進行安全驗證。
    -   新裝置的多因素驗證（電子郵件驗證）。
-   **架構**：
    -   **單一二進位部署**：前端（Vue 3）嵌入到 Go 二進位檔案中。
    -   **輕量級**：使用 SQLite 進行資料儲存，無需重型資料庫。
    -   **自包含**：基於環境變數的配置。
-   **終端控制台 UI/UX 升級亮點**：
    -   **視覺與氛圍強化**：
        -   **CRT 掃描線效果**：加入細微的水平掃描線疊加層，增加復古終端感。
        -   **虛擬游標**：輸入框現在有一個跟隨文字閃爍的「方塊游標（Caret）」，而不僅僅是標準的垂直線。
        -   **背景光暈**：添加了暗綠色的背景微光（Glow Effect）。
    -   **進階操作功能**：
        -   **Tab 自動補全**：現在輸入 `/` 觸發建議後，按 `Tab` 鍵會自動補全第一個建議指令。
        -   **點擊自動聚焦**：點擊終端機內部的任何位置都會自動將焦點放回輸入框。
        -   **清空控制台**：支援 `/clear` 指令或點擊右上角的垃圾桶圖示來本地端清空目前的顯示內容。
    -   **智慧滾動機制（Sticky Scroll）**：
        -   **新活動提示**：當您向上捲動查看歷史訊息時，新訊息進入不會再粗魯地將畫面彈回底部。
        -   **快速返回**：向上捲動時會出現「New Activity（新活動）」按鈕，點擊即可快速跳回最下方。
    -   **系統提示優化**：
        -   更新了 Prompt 樣式為 `admin@node-01:~$`。
        -   強化了指令輸出的視覺反饋，輸入的指令會以綠色底線高亮顯示。

## 開始使用

### 前置需求

-   Go 1.24+
-   Node.js 18+（用於建置前端）
-   Java（建議 JDK 17/21）已安裝並可在系統 PATH 中使用（用於執行 Minecraft 伺服器）。

### 安裝

1.  **複製儲存庫**
    ```bash
    git clone https://github.com/carsupper665/mc-server-backend.git
    cd mc-server-backend
    ```

2.  **建置前端與後端**
    此命令將建置 Vue 前端並將其嵌入到 Go 二進位檔案中。
    ```bash
    # Windows
    cd frontend && npm install && npm run build
    cd ..
    go build -o server.exe main.go

    # Linux/Mac
    cd frontend && npm install && npm run build
    cd ..
    go build -o server main.go
    ```

3.  **配置**
    在根目錄建立一個 `.env` 檔案（請參閱下方配置說明）。

4.  **執行**
    ```bash
    # Windows
    ./server.exe
    
    # Linux/Mac
    ./server
    ```
    在 `http://localhost:8080` 存取儀表板。

    **注意**：前端靜態檔案已經嵌入到 Go 二進位檔案中，執行伺服器後會自動提供前端服務。無需單獨啟動前端伺服器。

### 開發模式（前端熱重載）

如果您正在開發前端程式碼，可以使用開發模式以獲得熱重載功能：

1. **啟動後端伺服器**（在一個終端視窗）
   ```bash
   go run main.go
   # 或
   ./server.exe
   ```

2. **啟動前端開發伺服器**（在另一個終端視窗）
   ```bash
   cd frontend
   npm install  # 如果還沒安裝依賴
   npm run dev
   ```

3. **存取前端**
   - 前端開發伺服器通常執行在 `http://localhost:5173`（Vite 預設埠）
   - 前端會自動將 API 請求代理到 `http://localhost:8080`（後端）
   - 修改前端程式碼後會自動重新整理，無需重新建置

**開發模式的優勢**：
- ✅ 熱重載：修改程式碼立即看到效果
- ✅ 更快的編譯速度
- ✅ 更好的錯誤提示
- ✅ 支援 Vue DevTools 除錯

### 單獨測試前端靜態檔案（開發除錯用）

如果您想單獨測試編譯後的前端靜態檔案（不透過 Go 後端），可以使用以下方法：

1. **使用 Vite Preview（推薦）**
   ```bash
   cd frontend
   npm run preview
   ```
   這會在 `http://localhost:4173` 啟動一個預覽伺服器。

2. **使用 Python HTTP 伺服器**
   ```bash
   # Python 3
   cd frontend/dist
   python -m http.server 8080
   
   # Python 2
   python -m SimpleHTTPServer 8080
   ```

3. **使用 Node.js serve**
   ```bash
   npx serve frontend/dist -p 8080
   ```

**注意**：單獨執行靜態檔案時，API 請求會失敗（因為後端未執行）。僅用於測試前端介面，完整功能需要執行 Go 伺服器。

---

## `.env` 配置

在專案根目錄建立一個名為 `.env` 的檔案：

```dotenv
# 伺服器配置
PORT=8080
DEBUG=false

# 安全性（請更改這些！）
SESSION_SECRET=change-this-to-a-random-string
CRYPTO_SECRET=change-this-to-another-random-string

# 資料庫
SQLITE_PATH=./data/db.sqlite
SQL_MAX_IDLE_CONNS=100
SQL_MAX_OPEN_CONNS=1000
SQL_MAX_LIFETIME=60

# 初始管理員設定
CREATE_ROOT_USER=true
ROOT_USER_NAME=admin
ROOT_USER_EMAIL=admin@example.com
ROOT_USER_PASSWORD=securepassword

# 遊戲伺服器設定
MINECRAFT_SERVER_PATH=./minecraft_servers
LATEST_FABRIC_INSTALLER_VERSION=1.1.0

# 電子郵件（選用，用於 MFA 驗證碼）
SMTP_SERVER=smtp.example.com
SMTP_PORT=587
SMTP_ACCOUNT=noreply@example.com
SMTP_TOKEN=your-smtp-password
SMTP_FROM=noreply@example.com
SMTP_SSL_ENABLED=false

# 系統調校
GLOBAL_API_RATE_LIMIT=60
GLOBAL_API_RATE_LIMIT_DURATION=60
UA_FILTER=true
```

## 專案結構

```
├── common/         # 共用工具、常數、配置載入器
├── controller/     # HTTP 請求處理器
├── frontend/       # Vue 3 + Vite 前端原始碼
├── middleware/     # Gin 中介軟體（驗證、CORS、日誌）
├── model/          # 資料庫模型（SQLite/GORM）
├── router/         # API 路由與靜態檔案服務
├── service/        # 業務邏輯（伺服器程序管理）
└── main.go         # 進入點
```

## 技術堆疊

-   **後端**：Go（Gin Framework）、GORM（SQLite）、Gorilla WebSocket
-   **前端**：Vue 3、TypeScript、Vite、Tailwind CSS、Pinia
-   **基礎設施**：Embed（Go 1.16+）、Viper（配置）

## 授權

此專案為個人使用和實驗而建立。
