# 遊戲伺服器管理平台

一個輕量級、自託管的遊戲伺服器管理平台，使用 Go + Vue 3 開發，專為私人社群和朋友設計。

## 專案概述

本專案是一個全棧應用程式，包含後端 API 服務和前端管理介面，主要用於管理和控制多個 Minecraft 伺服器實例。專案採用前後端分離的架構，後端使用 Go 語言開發，前端使用 Vue 3 框架。

## 專案結構

```
mcserver/
├── mc-server-backend/          # 後端服務（Go）
│   ├── common/                 # 共用工具、常數、配置載入器
│   ├── controller/             # HTTP 請求處理器
│   ├── middleware/             # Gin 中介軟體（驗證、CORS、日誌、限流）
│   ├── model/                  # 資料庫模型（SQLite/GORM）
│   ├── router/                 # API 路由定義
│   ├── service/                # 業務邏輯（伺服器程序管理）
│   ├── data/                   # 資料庫檔案存放目錄
│   ├── logs/                   # 伺服器日誌
│   ├── minecraft_servers/      # Minecraft 伺服器實例目錄
│   ├── mc-server-frontend/     # 嵌入用前端（可選，目前未啟用）
│   ├── main.go                 # 進入點
│   ├── go.mod                  # Go 依賴管理
│   └── .env                    # 環境變數配置（需自行建立）
│
└── mc-server-frontend/         # 前端應用（Vue 3）
    ├── src/
    │   ├── api/                # API 呼叫封裝
    │   ├── components/         # Vue 元件
    │   ├── composables/        # Vue Composables（可重用邏輯）
    │   ├── layout/             # 佈局元件
    │   ├── router/             # 路由配置
    │   ├── store/              # Pinia 狀態管理
    │   ├── utils/              # 工具函數
    │   ├── views/              # 頁面視圖
    │   ├── App.vue             # 根元件
    │   └── main.js             # 進入點
    ├── public/                 # 靜態資源
    ├── package.json            # Node.js 依賴管理
    ├── vite.config.js          # Vite 配置
    └── index.html              # HTML 模板
```

## 功能特色

### 遊戲伺服器管理

- **Minecraft 伺服器管理**：
  - 建立、啟動、停止和配置 Minecraft（原版/Fabric）伺服器
  - 即時控制台存取與指令執行
  - 伺服器資源監控（CPU/RAM）
  - 可配置的 `server.properties` 編輯器
  - 備份管理（世界資料）
  - 多版本支援（Minecraft 1.10+）
- *實驗性支援 Among Us 伺服器*

### 使用者系統

- 私人存取控制（無公開註冊）
- 基於角色的權限（Root/Admin/User）
- 透過 JWT 和 Cookies 進行安全驗證
- 新裝置的多因素驗證（電子郵件驗證）
- 登入嘗試限制和速率限制

### 前端 UI/UX 特色

- **終端控制台增強**：
  - CRT 掃描線效果，增加復古終端感
  - 虛擬游標（方塊游標）
  - 背景光暈效果
  - Tab 自動補全指令
  - 點擊自動聚焦
  - 支援 `/clear` 指令清空控制台
  - 智慧滾動機制（Sticky Scroll）
  - 系統提示優化（`admin@node-01:~$` 樣式）
- **現代化介面**：
  - 響應式設計
  - 即時資料更新
  - 資源監控圖表
  - 直觀的操作介面

### 架構特性

- **前後端分離**：前端和後端可獨立部署和開發
- **輕量級**：使用 SQLite 進行資料儲存，無需重型資料庫
- **自包含**：基於環境變數的配置
- **可擴展**：模組化設計，易於添加新功能

## 技術堆疊

### 後端（mc-server-backend）

- **語言**：Go 1.24+
- **框架**：Gin
- **資料庫**：SQLite（GORM）
- **認證**：JWT、Gin Sessions
- **WebSocket**：Gorilla WebSocket（用於即時控制台）
- **其他**：CORS 支援、速率限制、日誌系統

### 前端（mc-server-frontend）

- **框架**：Vue 3（Composition API）
- **構建工具**：Vite
- **UI 庫**：Naive UI
- **狀態管理**：Pinia
- **路由**：Vue Router
- **HTTP 客戶端**：Axios
- **終端模擬**：xterm.js
- **圖表**：Chart.js

## 開始使用

### 前置需求

- **Go 1.24+**（後端開發）
- **Node.js 20.19.0+ 或 22.12.0+**（前端開發）
- **Java**（建議 JDK 17/21）已安裝並可在系統 PATH 中使用（用於執行 Minecraft 伺服器）
- **npm** 或 **yarn**（前端依賴管理）

### 快速安裝

1. **複製儲存庫**
   ```bash
   git clone <repository-url>
   cd mcserver
   ```

2. **後端設定**

   ```bash
   cd mc-server-backend
   
   # 安裝 Go 依賴
   go mod download
   
   # 建立環境變數檔案
   cp .env.example .env  # 如果有的話，或直接建立 .env 檔案
   # 編輯 .env 檔案，配置必要的環境變數（見下方配置說明）
   
   # 執行後端服務
   go run main.go
   # 或建置後執行
   go build -o server.exe main.go  # Windows
   go build -o server main.go      # Linux/Mac
   ./server.exe  # 或 ./server
   ```

   後端服務將在 `.env` 中指定的端口啟動（預設 8080）。

3. **前端設定**

   ```bash
   cd mc-server-frontend
   
   # 安裝依賴
   npm install
   
   # 開發模式執行
   npm run dev
   
   # 或建置生產版本
   npm run build
   ```

   開發模式下，前端通常執行在 `http://localhost:5173`（Vite 預設端口）。

### 開發模式（推薦）

開發時，建議分別執行前端和後端，以獲得熱重載功能：

1. **啟動後端服務**（在一個終端視窗）
   ```bash
   cd mc-server-backend
   go run main.go
   ```

2. **啟動前端開發伺服器**（在另一個終端視窗）
   ```bash
   cd mc-server-frontend
   npm run dev
   ```

3. **存取應用**
   - 前端開發伺服器：`http://localhost:5173`
   - 後端 API：`http://localhost:8080`
   - 前端會自動將 API 請求代理到後端（需要在 `vite.config.js` 中配置）

**開發模式的優勢**：
- ✅ 熱重載：修改程式碼立即看到效果
- ✅ 更快的編譯速度
- ✅ 更好的錯誤提示
- ✅ 支援 Vue DevTools 除錯

### 生產部署

#### 方式一：分離部署（目前使用的方式）

1. **建置前端**
   ```bash
   cd mc-server-frontend
   npm run build
   ```
   建置後的檔案將在 `dist/` 目錄中。

2. **部署前端**
   - 將 `dist/` 目錄部署到靜態檔案伺服器（如 Nginx、Apache 等）
   - 或使用 CDN 服務

3. **部署後端**
   - 建置後端二進位檔案
   - 配置 `.env` 檔案
   - 執行後端服務
   - 配置反向代理（如 Nginx）將前端路由請求轉發到前端，API 請求轉發到後端

4. **配置 CORS**
   - 在後端的 `.env` 中設定 `FRONTEND_BASE_URL` 為前端實際部署的 URL

#### 方式二：單一二進位部署（計劃中）

未來計劃將前端靜態檔案嵌入到 Go 二進位檔案中，實現單一二進位部署。目前相關程式碼已被註釋，待實現。

## 環境配置

在 `mc-server-backend` 目錄下建立一個名為 `.env` 的檔案：

```dotenv
# 伺服器配置
PORT=8080
DEBUG=false

# 前端應用 URL（用於 CORS 配置）
FRONTEND_BASE_URL=http://localhost:5173

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

### 環境變數說明

- **PORT**：後端 HTTP 伺服器監聽端口（預設: 8080）
- **DEBUG**：啟用詳細日誌和除錯端點（生產環境請設置為 false）
- **FRONTEND_BASE_URL**：前端應用的 URL，用於配置 CORS 和生成連結
- **SESSION_SECRET**：用於簽名和驗證會話 Cookie 的密鑰，請使用安全且隨機的值
- **CRYPTO_SECRET**：用於加密和解密敏感資料的密鑰
- **SQLITE_PATH**：SQLite 資料庫檔案路徑
- **CREATE_ROOT_USER**：如果為 true，在啟動時如果沒有根用戶，將自動建立預設根（管理員）用戶
- **ROOT_USER_NAME**、**ROOT_USER_EMAIL**、**ROOT_USER_PASSWORD**：初始管理員帳號資訊
- **MINECRAFT_SERVER_PATH**：Minecraft 伺服器實例存放目錄
- **SMTP_***：電子郵件服務器配置（用於發送 MFA 驗證碼）

## 開發說明

### 後端開發

- 後端使用 Go 標準專案結構
- API 路由定義在 `mc-server-backend/router/` 目錄
- 業務邏輯在 `mc-server-backend/service/` 目錄
- 資料模型在 `mc-server-backend/model/` 目錄
- 中介軟體在 `mc-server-backend/middleware/` 目錄
- HTTP 請求處理器在 `mc-server-backend/controller/` 目錄

### 前端開發

- 使用 Vue 3 Composition API
- 元件化開發，元件位於 `mc-server-frontend/src/components/`
- 頁面視圖位於 `mc-server-frontend/src/views/`
- API 呼叫統一在 `mc-server-frontend/src/api/` 管理
- 狀態管理使用 Pinia（`mc-server-frontend/src/store/`）
- 可重用邏輯使用 Composables（`mc-server-frontend/src/composables/`）

### API 文件

後端 API 文檔請參考 `mc-server-backend/README.md` 或查看 `mc-server-backend/router/` 目錄中的路由定義。

## 功能特性詳情

### 已實現功能

- ✅ 用戶認證系統（登入、登出、會話管理）
- ✅ Minecraft 伺服器建立和管理
- ✅ 伺服器啟動/停止控制
- ✅ 即時日誌查看（WebSocket）
- ✅ 伺服器備份和回滾
- ✅ 伺服器配置管理（server.properties 編輯器）
- ✅ 資源監控（CPU/RAM 使用率）
- ✅ 終端控制台增強功能
- ✅ 角色權限管理

### 開發中功能

- 🚧 雲端儲存整合
- 🚧 進階權限管理
- 🚧 伺服器效能監控（詳細指標）
- 🚧 多用戶協作功能
- 🚧 前端嵌入到後端二進位檔案（單一二進位部署）

## 授權

此專案為個人使用和實驗而建立。

## 參考

本專案受到 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 的啟發。

## 貢獻

歡迎提交 Issue 和 Pull Request！
