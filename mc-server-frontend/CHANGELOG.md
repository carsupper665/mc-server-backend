# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-04

### Added
- **Mod Browser** - 整合 Modrinth API 的模組瀏覽功能
  - 搜尋、過濾、分頁
  - 載入器過濾 (Fabric/Forge/NeoForge/Quilt)
  - 遊戲版本過濾
- **ESLint + Prettier** - 程式碼品質工具配置
- **Vitest** - 單元測試框架
  - `useSmartPolling` 測試
  - `useRateLimitGuard` 測試
  - `errorMapping` 測試
- **配置管理** - `src/config/constants.js` 集中管理常數
- **專案文件** - README.md、CHANGELOG.md

### Fixed
- Fabric 伺服器版本選項顯示錯誤
- ServerDetail 頁面導航失效問題
- ServerConsole 缺少 Terminal import

### Changed
- 更新 package.json 版本至 1.0.0
- 更新 MainLayout.vue 的 activeKey 計算邏輯

### Removed
- 模板檔案 (HelloWorld.vue, TheWelcome.vue, WelcomeItem.vue, icons/)

## [0.1.0] - 2025-12-XX

### Added
- 初始專案結構
- Dashboard 儀表板
- Servers 伺服器列表與管理
- ServerDetail 伺服器詳情與控制台
- Login 登入頁面
- Among Us 頁面
