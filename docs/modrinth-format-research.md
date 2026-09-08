# Modrinth 匯入缺口評估

查核日期：2026-09-08。範圍為此專案伺服器端 .mrpack 匯入，並非 Modrinth App 所有功能。本次僅靜態程式碼評估，沒有修改產品程式或執行整合包測試。

## 官方規格

來源：[Modrinth 官方格式規格](https://support.modrinth.com/en/articles/8802351-modrinth-modpack-format-mrpack)。

容器為 ZIP，副檔名 .mrpack，MIME 為 application/x-modrinth-modpack+zip，根目錄需有 UTF-8 modrinth.index.json。中繼資料包括 formatVersion（目前 1）、game（minecraft）、versionId、name、可選 summary；files 的項目包括 path、hashes（sha1 與 sha512）、可選 env、downloads、fileSize。

環境分 client/server，各有 required、optional、unsupported；官方建議提供 optional 選擇介面。官方此頁未明確寫出省略 env 的預設語意，實作前應以官方原始碼或相容性測試確認。下載使用合法編碼 HTTPS URL，需處理重新導向，實作者應定義允許網域。依賴 ID 包括 minecraft、forge、neoforge、fabric-loader、quilt-loader，且需考慮新增 ID。

覆寫順序為 overrides，再由目標端的 server-overrides 或 client-overrides 覆蓋。路徑必須限制在實例內。伺服器端應辨識並略過不適用的 client 檔案，並不需要安裝客戶端。

## 程式碼核對

| 項目 | 現況與證據 |
| --- | --- |
| 匯入入口 | 全庫搜尋無 mrpack 或 modrinth.index parser；router/api.go 無整合包上傳、預覽或套用 API |
| 可重用基礎 | service/modInstallQueue.go 已有單模組依賴解析、伺服器佇列、job 和事件進度 |
| 載入器 | service/minecraftServer.go:209 只有 Vanilla/Fabric；Forge、NeoForge、Quilt 未實作建立流程 |
| 啟動方式 | service/serverManager.go:81 固定 -jar server.jar，需要 loader 專用啟動方案 |
| 下載 | service/serverMods.go:446 固定 mods/、單 URL，有暫存檔 rename，沒有 SHA-1/SHA-512 內容校驗 |
| 雜湊 | service/serverMods.go:413 只從中繼資料取雜湊字串，不等於核對下載內容 |
| 檔案追蹤 | model/serverMods.go 以 Modrinth project/version 為中心，需增加整合包與外部檔案、覆寫檔、選項紀錄 |
| 失敗回復 | executeInstallPlan 逐個 AddMod，遇錯返回，沒有整包套用回復 |
| 前端 | web/src/api/modrinth.js 搜尋固定 project_type:mod；loader 搜尋選項不代表後端支援建立 |

## 建議與工作量

新增專用 mrpack 檔案安裝計畫，保留清單指定的內容，避免以最新模組查詢或 API 依賴解析重新決定包內版本。mrpack 的檔案 env 與 API 的模組 dependencies 是不同概念。

先完成上傳、解析、預覽、optional 選擇、ZIP 與路徑驗證、多下載來源、雜湊校驗、暫存安裝、覆寫順序，接上 Vanilla/Fabric 和現有進度介面；之後補 Forge、NeoForge、Quilt 的精確版本安裝、Java 需求與啟動適配。第一階段只能稱為部分 loader 支援。

替代 URL 重試、下載逾時、解壓限制、重新導向目的地檢查、整包失敗回復是建議的相容性與可靠性設計，不全是官方強制欄位。未知依賴或格式不能靜默忽略後宣告成功。匯入既有伺服器需另定義衝突與回復策略；整合包更新、匯出不是本次格式匯入的必要範圍。

驗收應涵蓋環境三種值及缺省、optional 選擇、覆寫衝突、client 排除、非 Modrinth 檔案、替代下載、校驗失敗、非法路徑、未知依賴、中途失敗，並實際啟動各 loader 的代表性整合包。

結論：完整伺服器端支援屬中大型功能，已有管理基礎，但匯入核心與三種 loader 都尚缺。此評估沒有實際包測試，不提供支援百分比或確定工期。
