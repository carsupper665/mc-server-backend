/**
 * 錯誤訊息淨化對照表
 * 將後端技術性或不雅錯誤訊息轉換為友善提示語
 */

// 精確匹配對照表
const exactErrorMappings = {
    'record not found': '找不到該項目',
    'not found': '找不到該項目',
    'unauthorized': '登入已過期，請重新登入',
    'forbidden': '您沒有權限執行此操作',
    'internal server error': '伺服器發生錯誤，請稍後再試',
    'bad request': '請求格式有誤，請檢查輸入內容',
    'server already running': '伺服器已在運行中',
    'server not running': '伺服器目前未運行',
    'invalid token': '驗證已失效，請重新登入',
    'token expired': '登入已過期，請重新登入',
    'permission denied': '權限不足',
    'connection refused': '無法連線至伺服器',
    'timeout': '請求逾時，請稍後再試',
    'network error': '網路連線錯誤',
};

// 包含匹配對照表 (用於模糊匹配)
const partialErrorMappings = [
    { pattern: /傻逼|傻b|sb|fuck|shit|damn/i, message: '發生未知的伺服器錯誤' },
    { pattern: /sql|database|query|syntax/i, message: '資料處理發生錯誤' },
    { pattern: /panic|fatal|crash/i, message: '伺服器發生嚴重錯誤，請聯絡管理員' },
    { pattern: /nil pointer|null pointer|undefined/i, message: '伺服器內部錯誤' },
    { pattern: /file not found|no such file/i, message: '找不到指定的檔案' },
    { pattern: /permission|access denied/i, message: '權限不足，無法執行操作' },
    { pattern: /already exists/i, message: '該項目已存在' },
    { pattern: /invalid.*format/i, message: '格式不正確' },
    { pattern: /too many requests/i, message: '請求過於頻繁，請稍後再試' },
];

/**
 * 淨化錯誤訊息
 * @param {string} errorMessage - 原始錯誤訊息
 * @returns {string} - 友善的錯誤訊息
 */
export function sanitizeErrorMessage(errorMessage) {
    if (!errorMessage || typeof errorMessage !== 'string') {
        return '發生未知錯誤';
    }

    const lowerMessage = errorMessage.toLowerCase().trim();

    // 1. 先嘗試精確匹配
    if (exactErrorMappings[lowerMessage]) {
        return exactErrorMappings[lowerMessage];
    }

    // 2. 嘗試部分匹配 (精確對照表的 key)
    for (const [pattern, friendly] of Object.entries(exactErrorMappings)) {
        if (lowerMessage.includes(pattern)) {
            return friendly;
        }
    }

    // 3. 嘗試正則表達式模糊匹配
    for (const { pattern, message } of partialErrorMappings) {
        if (pattern.test(errorMessage)) {
            return message;
        }
    }

    // 4. 如果訊息看起來是技術性的 (包含特殊字元或很長)，返回通用訊息
    if (
        errorMessage.includes('::') ||
        errorMessage.includes('0x') ||
        errorMessage.includes('\\') ||
        errorMessage.length > 100
    ) {
        return '伺服器發生錯誤，請稍後再試';
    }

    // 5. 如果訊息看起來正常，直接返回 (可能已經是友善訊息)
    return errorMessage;
}

/**
 * 從 API 錯誤響應中提取並淨化錯誤訊息
 * @param {Error|Object} error - Axios 錯誤物件或一般錯誤
 * @returns {string} - 友善的錯誤訊息
 */
export function extractAndSanitizeError(error) {
    let rawMessage = '';

    if (error?.response?.data?.error) {
        rawMessage = error.response.data.error;
    } else if (error?.response?.data?.message) {
        rawMessage = error.response.data.message;
    } else if (error?.message) {
        rawMessage = error.message;
    } else if (typeof error === 'string') {
        rawMessage = error;
    }

    return sanitizeErrorMessage(rawMessage);
}
