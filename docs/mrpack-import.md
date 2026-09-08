# mrpack 匯入

目前沿用既有 Vanilla / Fabric 建服能力；其他 loader 會在建立 session 前明確拒絕。本次只實作匯入與中斷紀錄，沒有恢復、更新或匯出整合包。

## API

所有路由沿用 JWT 與 X-Device-ID 驗證。前端於伺服器列表點選「匯入 .mrpack」。本地讀 ZIP 索引、選取模組與覆寫檔、編輯文字設定，計算修改後大小，再經置中確認框提交。

| 方法與路由 | 行為 |
| --- | --- |
| GET /api/v1/server/modpack/config | max_size（bytes）、blacklist；預覽使用同一組後端規則 |
| POST /api/v1/server/modpack/import | 唯一匯入入口；body 是修改後的 ZIP bytes，Content-Type: application/x-modrinth-modpack+zip |
| GET /api/v1/server/mod/job/:ins_ses_id | 擁有者查詢完整 session；記憶體已清除時從 DB 讀取 |
| GET /api/v1/server/mod/subscribe/:ins_ses_id | 沿用 progress SSE；最近事件供 UI，完整紀錄由 job 查詢 |

POST 建立成功回 HTTP 200，JSON 含 ins_ses_id、server_id。這表示背景工作建立成功，不代表安裝完成。上傳超限回 413，格式錯誤回 400。不使用 multipart，也沒有另一個後端預覽/提交入口。

## 設定

.env 加入 MRPACK_MAX_SIZE_MB=150，單位為 MiB（150 × 1024 × 1024 bytes）。限制上傳壓縮檔與 ZIP 解壓內容各自總量，不計下載清單指向的外部檔案。前端顯示下載量、各覆寫檔大小及修改後壓縮/解壓大小。預覽只讀目錄及索引，因此大於後端限制的原包仍可先刪選。

MRPACK_BLACKLIST 為逗號分隔的實例相對路徑、尾端 / 的目錄前綴或 glob（建議使用 *、?）。未設定時使用 service/mrpack.go 的預設值；明確設為空字串則停用黑名單。預設排除 options.txt、optionsof.txt、servers.dat、servers.dat_old、resourcepacks/、shaderpacks/、screenshots/、config/*-client.*、config/iris.properties、config/sodium-options.json。

MRPACK_DOWNLOAD_HOSTS 可選，為逗號分隔的精確 HTTPS 主機名稱。預設包含 Modrinth CDN、GitHub/raw GitHub、GitLab、GitHub release-assets/objects。重新導向目的地也套用相同規則。

## 邊界與紀錄

解析器驗證索引、雜湊格式、HTTPS 來源與安裝路徑，排除 env.server=unsupported 和黑名單，再按 overrides → server-overrides 排序。client-overrides 不套用；缺省 env 視為可安裝。optional 預設選取，前端可取消任何模組。黑名單不是完整的客戶端模組辨識系統：缺少正確環境宣告的模組仍需使用者選擇。

下載保留包內指定檔案，依序嘗試 downloads，核對大小、SHA-1、SHA-512，失敗不留下最終檔案。Modrinth hash lookup 只補充既有 Mod / ModVersion / ServerMod 表，不重新選版本或開啟自動更新。外部或無法取得 metadata 的檔案仍安裝，原因寫入 session；覆寫內的 JAR 亦嘗試以雜湊補充資料。

ModpackSession 沿用 InstallJobSnapshot 與 InstallEvent，新增 owner、ins_ses_id 和完整 details。伺服器表新增 install_session_id、install_status、install_details（JSON text），由既有 AutoMigrate 建欄，不新增 table。建立時及完成時保存，完成後保留記憶體一小時。沿用現有 event loop 每分鐘掃描，先存 DB 成功再移除；保存失敗保留 session 重試。

正常關閉及自動更新的 shutdown 路徑會拒絕新 session、取消下載、等工作結束並保存 interrupted。強制終止程序或斷電無法執行 hook；已保存的初始 session 仍可查詢，但不會自動恢復或補判中斷。安裝未完成不能啟動，進行中不能刪除實例。失敗保留檔案與紀錄供除錯。

## 驗證

Go 測試涵蓋提供的 test/mrcpack/test_mod_pack.mrpack、環境/黑名單、覆寫順序、路徑、hash/大小、mock 完整安裝、DB 模組紀錄、owner 隔離、SSE、過期保存、shutdown 取消與 interrupted。前端測試涵蓋實包、刪選、文字修改、重打包和大小一致性；瀏覽器使用真實元件與真實 ZIP、mock API 驗證二次確認、ZIP 上傳及既有進度元件。

沒有將測試包部署到使用者的實際 Minecraft 伺服器，也未保證該包內所有模組可在 dedicated server 啟動。

目前針對新增功能的 Go 測試、go build ./...、go vet 與前端 mrpack 測試、production build 已通過。全庫既有測試仍有獨立問題：test/add_mod_route_test.go 使用舊 SetAPIRouter 簽名而無法編譯；前端 errorMapping 與 useRateLimitGuard 各有一項既有斷言失敗。Windows 環境未啟用 CGO 且無 C 編譯器，因此 -race 未執行成功。這些不等同於本次功能驗證通過，也沒有宣稱全庫測試全綠。
