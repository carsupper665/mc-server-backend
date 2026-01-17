# 📊 資料庫設計文檔

## 主表：`mods`

| 欄位名 | 資料型別 | 約束 | 說明 | 範例 |
|--------|---------|------|------|------|
| `id` | INTEGER | PRIMARY KEY, AUTOINCREMENT | 自動遞增的主鍵 | `1`, `2`, `3` |
| `server_id` | TEXT | NOT NULL | 伺服器唯一識別碼 | `"server1"`, `"creative_server"` |
| `mod_id` | TEXT | NOT NULL | MOD 的 Modrinth slug 或 ID | `"sodium"`, `"fabric-api"` |
| `version_id` | TEXT | NOT NULL | Modrinth 版本 ID (UUID) | `"abc123def456"` |
| `version_number` | TEXT | NOT NULL | 人類可讀的版本號 | `"0.5.3"`, `"1.2.0"` |
| `filename` | TEXT | NOT NULL | 下載的檔案名稱 | `"sodium-fabric-0.5.3.jar"` |
| `auto_update` | BOOLEAN | NOT NULL, DEFAULT 0 | 是否自動更新 | `true`, `false` |
| `created_at` | DATETIME | NOT NULL | 記錄創建時間 | `"2026-01-06 10:30:00"` |
| `updated_at` | DATETIME | NOT NULL | 記錄更新時間 | `"2026-01-06 15:45:00"` |

## 約束條件

```sql
-- 唯一約束：同一伺服器不能有重複的 MOD
UNIQUE(server_id, mod_id)
```

## 索引

```sql
-- 提升按伺服器查詢的效能
CREATE INDEX idx_mods_server_id ON mods(server_id);
```

---

## 🔗 資料關係圖

```
┌─────────────────────────────────────────────────┐
│                    mods                         │
├─────────────────────────────────────────────────┤
│ id (PK)              │ INTEGER                  │
│ server_id            │ TEXT        ◄───┐        │
│ mod_id               │ TEXT            │        │
│ version_id           │ TEXT            │        │
│ version_number       │ TEXT            │        │
│ filename             │ TEXT            │        │
│ auto_update          │ BOOLEAN         │        │
│ created_at           │ DATETIME        │        │
│ updated_at           │ DATETIME        │        │
└─────────────────────────────────────────────────┘
                                          │
                                          │
                        ┌─────────────────┴─────────────────┐
                        │  同一個 server_id 可以有多個 MOD  │
                        │  但每個 mod_id 只能出現一次       │
                        └───────────────────────────────────┘
```

---

## 📝 資料範例

### 範例 1：單一伺服器的多個 MOD

| id | server_id | mod_id | version_id | version_number | filename | auto_update | created_at | updated_at |
|----|-----------|--------|------------|----------------|----------|-------------|------------|------------|
| 1 | server1 | sodium | abc123 | 0.5.3 | sodium-fabric-0.5.3.jar | true | 2026-01-05 10:00:00 | 2026-01-05 10:00:00 |
| 2 | server1 | fabric-api | def456 | 0.92.0 | fabric-api-0.92.0.jar | true | 2026-01-05 10:05:00 | 2026-01-05 10:05:00 |
| 3 | server1 | lithium | ghi789 | 0.11.2 | lithium-fabric-0.11.2.jar | false | 2026-01-05 10:10:00 | 2026-01-05 10:10:00 |
| 4 | server2 | sodium | abc123 | 0.5.3 | sodium-fabric-0.5.3.jar | true | 2026-01-06 09:00:00 | 2026-01-06 09:00:00 |

### 範例 2：MOD 更新後的變化

**更新前：**

| id | server_id | mod_id | version_id | version_number | updated_at |
|----|-----------|--------|------------|----------------|------------|
| 1 | server1 | sodium | abc123 | 0.5.3 | 2026-01-05 10:00:00 |

**更新後：**

| id | server_id | mod_id | version_id | version_number | updated_at |
|----|-----------|--------|------------|----------------|------------|
| 1 | server1 | sodium | xyz999 | 0.5.5 | 2026-01-06 14:30:00 |

---

## 🔄 關係說明

### 1. 一對多關係

```
Server (1) ──────── (N) Mods

一個伺服器 ─► 可以有多個 MOD
一個 MOD ─► 只能屬於一個伺服器
```

**查詢範例：**

```sql
-- 獲取 server1 的所有 MOD
SELECT * FROM mods WHERE server_id = 'server1';
```

### 2. 唯一性約束

```
(server_id, mod_id) 組合必須唯一

✅ 允許：
   server1 + sodium
   server1 + fabric-api
   server2 + sodium

❌ 不允許：
   server1 + sodium (已存在)
   server1 + sodium (重複)
```

---

## 📊 常見查詢模式

### 1. 按伺服器查詢所有 MOD

```sql
SELECT 
    mod_id,
    version_number,
    filename,
    auto_update,
    created_at
FROM mods
WHERE server_id = ?
ORDER BY created_at DESC;
```

**結果：**

| mod_id | version_number | filename | auto_update | created_at |
|--------|----------------|----------|-------------|------------|
| lithium | 0.11.2 | lithium-fabric-0.11.2.jar | false | 2026-01-05 10:10:00 |
| fabric-api | 0.92.0 | fabric-api-0.92.0.jar | true | 2026-01-05 10:05:00 |
| sodium | 0.5.3 | sodium-fabric-0.5.3.jar | true | 2026-01-05 10:00:00 |

### 2. 查詢需要自動更新的 MOD

```sql
SELECT 
    server_id,
    mod_id,
    version_number,
    filename
FROM mods
WHERE auto_update = true
ORDER BY server_id, mod_id;
```

**結果：**

| server_id | mod_id | version_number | filename |
|-----------|--------|----------------|----------|
| server1 | fabric-api | 0.92.0 | fabric-api-0.92.0.jar |
| server1 | sodium | 0.5.3 | sodium-fabric-0.5.3.jar |
| server2 | sodium | 0.5.3 | sodium-fabric-0.5.3.jar |

### 3. 檢查 MOD 是否已安裝

```sql
SELECT COUNT(*) as count
FROM mods
WHERE server_id = ? AND mod_id = ?;
```

**結果：**

| count |
|-------|
| 1 (已安裝) 或 0 (未安裝) |

---

## 🔍 實際使用情境

### 情境 1：安裝新 MOD

```go
// 第一次安裝 sodium
AddModToServer("server1", "sodium", "abc123", "0.5.3", "sodium-fabric-0.5.3.jar", true)
```

**資料庫變化：**

```diff
+ | 1 | server1 | sodium | abc123 | 0.5.3 | sodium-fabric-0.5.3.jar | true | 2026-01-06 10:00:00 | 2026-01-06 10:00:00 |
```

### 情境 2：更新現有 MOD

```go
// 更新 sodium 到新版本
UpdateMod("server1", "sodium", "xyz999", "0.5.5", "sodium-fabric-0.5.5.jar", true)
```

**資料庫變化：**

```diff
- | 1 | server1 | sodium | abc123 | 0.5.3 | sodium-fabric-0.5.3.jar | true | 2026-01-06 10:00:00 | 2026-01-06 10:00:00 |
+ | 1 | server1 | sodium | xyz999 | 0.5.5 | sodium-fabric-0.5.5.jar | true | 2026-01-06 10:00:00 | 2026-01-06 14:30:00 |
```

### 情境 3：刪除 MOD

```go
DeleteMod("server1", "sodium")
```

**資料庫變化：**

```diff
- | 1 | server1 | sodium | xyz999 | 0.5.5 | sodium-fabric-0.5.5.jar | true | 2026-01-06 10:00:00 | 2026-01-06 14:30:00 |
```

---

## 🎯 設計優點

| 優點 | 說明 |
|------|------|
| ✅ **防止重複** | `UNIQUE(server_id, mod_id)` 確保同一伺服器不會重複安裝同一個 MOD |
| ✅ **易於查詢** | 索引 `idx_mods_server_id` 加速按伺服器查詢 |
| ✅ **時間追蹤** | `created_at` 和 `updated_at` 記錄安裝和更新時間 |
| ✅ **自動更新支援** | `auto_update` 欄位方便實作定時更新功能 |
| ✅ **靈活性** | 支援多個伺服器，每個伺服器可以有不同的 MOD 組合 |
| ✅ **可維護性** | 結構簡單清晰，易於理解和維護 |

---

## 📦 完整建表語句

```sql
CREATE TABLE IF NOT EXISTS mods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id TEXT NOT NULL,
    mod_id TEXT NOT NULL,
    version_id TEXT NOT NULL,
    version_number TEXT NOT NULL,
    filename TEXT NOT NULL,
    auto_update BOOLEAN NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE(server_id, mod_id)
);

CREATE INDEX IF NOT EXISTS idx_mods_server_id ON mods(server_id);
```

---

## 🚀 使用範例

### 初始化資料庫

```go
import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
    db, err := sql.Open("sqlite3", "./minecraft.db")
    if err != nil {
        return nil, err
    }
    
    modModel := NewModModel(db)
    if err := modModel.CreateModsTable(); err != nil {
        return nil, err
    }
    
    return db, nil
}
```

### 常用操作

```go
// 1. 新增 MOD
err := modModel.AddModToServer("server1", "sodium", "abc123", "0.5.3", "sodium-fabric-0.5.3.jar", true)

// 2. 更新 MOD
err := modModel.UpdateMod("server1", "sodium", "xyz999", "0.5.5", "sodium-fabric-0.5.5.jar", true)

// 3. 查詢單個 MOD
mod, err := modModel.GetMod("server1", "sodium")

// 4. 查詢伺服器所有 MOD
mods, err := modModel.GetServerMods("server1")

// 5. 查詢需要自動更新的 MOD
autoUpdateMods, err := modModel.GetAutoUpdateMods()

// 6. 刪除 MOD
err := modModel.DeleteMod("server1", "sodium")
```

這個設計能夠滿足 Minecraft 伺服器 MOD 管理的所有基本需求！🎉