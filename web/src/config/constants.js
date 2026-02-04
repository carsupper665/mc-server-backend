/**
 * 應用程式配置常數
 * 集中管理所有硬編碼的設定值
 */

// API 端點
export const API_CONFIG = {
    BASE_URL: '',
    TIMEOUT: 30000,
    MODRINTH_BASE_URL: 'https://api.modrinth.com/v2',
    MODRINTH_TIMEOUT: 15000,
};

// 輪詢設定
export const POLLING_CONFIG = {
    IDLE_INTERVAL: 12000,        // 閒置模式：12 秒
    ACTIVE_INTERVAL: 2000,       // 活躍模式：2 秒
    ACTIVE_DURATION: 30000,      // 活躍持續：30 秒
    LOG_INTERVAL: 5000,          // 日誌輪詢：5 秒
    METRICS_INTERVAL: 10000,     // 指標更新：10 秒
};

// Rate Limit 設定
export const RATE_LIMIT_CONFIG = {
    BACKOFF_DELAYS: [2000, 4000, 8000, 16000, 30000],
};

// 快取設定
export const CACHE_CONFIG = {
    VERSION_CACHE_DURATION: 5 * 60 * 1000, // 5 分鐘
    METRICS_HISTORY_MAX: 30,
};

// 終端機設定
export const TERMINAL_CONFIG = {
    FONT_SIZE: 14,
    FONT_FAMILY: 'Fira Code, monospace',
    ROWS: 20,
    THEME: {
        background: '#1a1a20',
        foreground: '#d4d4d4',
        cursor: '#aeafad',
        selectionBackground: '#264f78',
        black: '#000000',
        red: '#cd3131',
        green: '#0dbc79',
        yellow: '#e5e510',
        blue: '#2472c8',
        magenta: '#bc3fbc',
        cyan: '#11a8cd',
        white: '#e5e5e5',
    },
};

// 分頁設定
export const PAGINATION_CONFIG = {
    DEFAULT_PAGE_SIZE: 20,
    MAX_PAGE_SIZE: 100,
};

// LocalStorage Keys
export const STORAGE_KEYS = {
    USERNAME: 'username',
    COMMAND_HISTORY_PREFIX: 'mc_cmd_history_',
};

// 伺服器類型前綴
export const SERVER_TYPE_PREFIX = {
    FABRIC: 'mcsfv',
    VANILLA: 'mcsvv',
};

export default {
    API_CONFIG,
    POLLING_CONFIG,
    RATE_LIMIT_CONFIG,
    CACHE_CONFIG,
    TERMINAL_CONFIG,
    PAGINATION_CONFIG,
    STORAGE_KEYS,
    SERVER_TYPE_PREFIX,
};
