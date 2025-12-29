# Go 後端

一個使用 Go 語言編寫的輕量級後端服務，僅供娛樂和實驗使用。

## 概述

一個簡單的後端應用程式，具有以下功能（註：所有功能仍在開發中，尚未完全實現。）：
- 登入認證
- 身份管理
- 管理
- 遊戲伺服器控制
- 雲端儲存
- 我稍後再想想。;D

## 開始使用

1. 複製儲存庫  
   ```bash
   git clone https://github.com/carsupper665/web-server-backend.git
   cd web-server-backend
   ```

2. 在專案根目錄建立 `.env` 檔案（詳細說明見下方）。

3. 安裝依賴並執行：  
   ```bash
   go mod download
   go run main.go
   ```

4. 伺服器將在您於 `.env` 中指定的端口啟動（預設 `8080`）。

---

## `.env` 配置

在專案根目錄建立一個名為 `.env` 的檔案，包含以下變數：

```dotenv
# 前端應用程式的 URL（例如 http://localhost:3000）
FRONTEND_BASE_URL=

# 後端監聽的端口
PORT=8080

# 啟用除錯日誌（true 或 false）
DEBUG=false

# 會話和加密密鑰
SESSION_SECRET=your-secret-key
CRYPTO_SECRET=your-crypto-secret

# 路徑和快取
SQLITE_PATH=path/to/db.sqlite
MEMORY_CACHE_ENABLED=false

# 計時設定（以秒為單位）
SYNC_FREQUENCY=60
BATCH_UPDATE_INTERVAL=300
RELAY_TIMEOUT=30

# 資料庫連接池設定：
#   SQL_MAX_IDLE_CONNS: 連接池中保留的最大空閒連接數（預設: 100）
SQL_MAX_IDLE_CONNS=100

#   SQL_MAX_OPEN_CONNS: 同時打開的資料庫連接最大數量（預設: 1000）
SQL_MAX_OPEN_CONNS=1000

#   SQL_MAX_LIFETIME: 連接在被關閉前可以重用的最大時間（以秒為單位，預設: 60）
SQL_MAX_LIFETIME=60

# 在啟動時自動建立根/管理員用戶（true 或 false）
CREATE_ROOT_USER=true
```

### 變數說明

- **FRONTEND_BASE_URL**  
  前端應用程式託管的基礎 URL（例如 `http://localhost:3000`）。用於配置 CORS 和生成連結。

- **PORT**  
  Go HTTP 伺服器監聽的 TCP 端口（預設: `3000`）。

- **DEBUG**  
  設置為 `true` 時，啟用詳細日誌和除錯端點。生產環境請使用 `false`。

- **SESSION_SECRET**  
  用於簽名和驗證會話 Cookie 的密鑰。請保持此密鑰安全且隨機。

- **CRYPTO_SECRET**  
  用於加密和解密敏感資料的密鑰。

- **SQLITE_PATH**  
  SQLite 資料庫的檔案路徑。範例：`./data/db.sqlite`。

- **MEMORY_CACHE_ENABLED**  
  啟用記憶體快取（`true` 或 `false`）。

- **SYNC_FREQUENCY**  
  背景同步任務應多久執行一次（以秒為單位）。

- **BATCH_UPDATE_INTERVAL**  
  批次更新操作之間的間隔（以秒為單位）。

- **RELAY_TIMEOUT**  
  中繼操作在放棄前的逾時時間（以秒為單位）。

- **SQL_MAX_IDLE_CONNS**  
  資料庫連接池將保持打開的最大空閒（未使用）連接數。  
  預設：`100`。

- **SQL_MAX_OPEN_CONNS**  
  資料庫同時打開的最大連接總數。  
  預設：`1000`。

- **SQL_MAX_LIFETIME**  
  連接在被關閉和替換之前可以重用的最大時間（以秒為單位）。  
  預設：`60`。

- **CREATE_ROOT_USER**  
  如果為 `true`，應用程式將在啟動時（當不存在根用戶時）自動建立預設的根（管理員）用戶。  
  設置為 `false` 以停用自動用戶建立。

---
## 參考

- 本專案受到 [QuantumNous/new-api](https://github.com/QuantumNous/new-api) 的啟發
