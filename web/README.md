# MC Server Frontend

> Minecraft 伺服器管理面板的前端應用程式

[![Vue 3](https://img.shields.io/badge/Vue-3.5-4FC08D?logo=vue.js)](https://vuejs.org/)
[![Vite](https://img.shields.io/badge/Vite-7.3-646CFF?logo=vite)](https://vitejs.dev/)
[![Naive UI](https://img.shields.io/badge/NaiveUI-2.43-18A058)](https://naiveui.com/)

## ✨ 功能特色

- 🖥️ **伺服器管理** - 建立、啟動、停止 Minecraft 伺服器 (Vanilla/Fabric)
- 📟 **即時終端機** - xterm.js 整合的伺服器主控台與指令歷史
- 📦 **Mod Browser** - 整合 Modrinth API 瀏覽與搜尋模組
- 📊 **系統監控** - 即時 CPU、RAM 使用量圖表
- 🔐 **安全認證** - 雙因素驗證支援
- 📱 **響應式設計** - 支援桌面與行動裝置

## 🧩 Mod Browser 功能

透過整合 [Modrinth API](https://modrinth.com/)，提供完整的模組瀏覽體驗：

- **搜尋模組** - 關鍵字即時搜尋
- **載入器過濾** - Fabric / Forge / NeoForge / Quilt
- **版本過濾** - 依 Minecraft 版本篩選
- **排序選項** - 相關性、下載數、追蹤數、最新、最近更新
- **分頁瀏覽** - 流暢的分頁載入
- **一鍵跳轉** - 點擊卡片直接前往 Modrinth 下載頁

相關檔案：
- `src/api/modrinth.js` - Modrinth API 客戶端
- `src/components/ModCard.vue` - 模組卡片元件
- `src/components/ModBrowser.vue` - 搜尋與過濾主元件
- `src/views/ModsView.vue` - 模組頁面

## 🚀 快速開始

### 環境需求

- Node.js >= 20.19.0 或 >= 22.12.0
- npm 或 pnpm

### 安裝

```bash
# 安裝依賴
npm install

# 啟動開發伺服器
npm run dev

# 建置生產版本
npm run build
```

## 📁 專案結構

```
src/
├── api/          # API 客戶端 (axios, modrinth)
├── components/   # 可重用 Vue 元件
├── composables/  # Vue Composition API hooks
├── config/       # 配置常數
├── layout/       # 佈局元件
├── router/       # Vue Router 設定
├── store/        # Pinia 狀態管理
├── utils/        # 工具函式
└── views/        # 頁面視圖
```

## 🛠️ 技術棧

| 技術 | 用途 |
|------|------|
| Vue 3 | UI 框架 |
| Vite | 建置工具 |
| Pinia | 狀態管理 |
| Vue Router | 路由 |
| Naive UI | 元件庫 |
| xterm.js | 終端機模擬 |
| Chart.js | 圖表 |
| Axios | HTTP 客戶端 |
| Modrinth API | 模組資料來源 |

## 📝 開發指南

### 程式碼風格

```bash
# ESLint 檢查
npm run lint

# Prettier 格式化
npm run format
```

### 新增頁面

1. 在 `src/views/` 建立 `XxxView.vue`
2. 在 `src/router/NameIndex.js` 添加路由
3. 在 `src/layout/MainLayout.vue` 添加導航

## 🔒 安全注意事項

- Token 透過 httpOnly cookie 處理（建議後端設定）
- 所有 API 請求使用 withCredentials
- 錯誤訊息經過淨化處理，避免洩露技術細節

## 📄 授權

Private - 僅供內部使用
