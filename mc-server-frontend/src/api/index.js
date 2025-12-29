import axios from 'axios';
import { createDiscreteApi } from 'naive-ui';
import { sanitizeErrorMessage } from '../utils/errorMapping';
import { triggerBackoff, resetBackoff } from '../composables/useRateLimitGuard';

// 建立獨立的 message API (可在非 Vue 元件中使用)
const { message } = createDiscreteApi(['message'], {
    configProviderProps: {
        theme: undefined // 使用預設暗色主題
    }
});

// 友善錯誤訊息對照表 (HTTP Status Code)
const friendlyErrorMessages = {
    400: '請求格式有誤，請檢查輸入內容',
    401: '登入已過期，請重新登入',
    403: '您沒有權限執行此操作',
    404: '找不到請求的資源',
    408: '請求逾時，請檢查網路連線',
    429: '請求過於頻繁，系統正在自動退避重試...',
    500: '伺服器稍稍打了個盹 💤 請稍後再試',
    502: '伺服器正在重新啟動中...',
    503: '服務暫時無法使用，請稍後再試',
    504: '閘道逾時，請稍後再試'
};

// 需要靜默處理的錯誤 (不顯示 Toast)
const silentErrors = [
    '/mc-api/a/status/', // 狀態檢查錯誤不需要提示
    '/server-api/a/log/' // Log 獲取錯誤不需要提示
];

const api = axios.create({
    baseURL: '',
    timeout: 30000, // 增加 timeout 以容納慢速 API
    withCredentials: true,
});

// Response interceptor
api.interceptors.response.use(
    (response) => {
        // 請求成功，重置退避計數
        resetBackoff();
        return response.data;
    },
    (error) => {
        const status = error.response?.status;
        const url = error.config?.url || '';

        // 檢查是否為靜默錯誤
        const isSilent = silentErrors.some(pattern => url.includes(pattern));

        // 429 Rate Limit - 觸發指數退避
        if (status === 429) {
            triggerBackoff();
            if (!isSilent) {
                const friendlyMsg = friendlyErrorMessages[429];
                message.warning(friendlyMsg);
            }
        } else if (status === 401) {
            // 未授權：清除本地狀態並重導向
            localStorage.removeItem('username');

            if (!window.location.pathname.startsWith('/login')) {
                message.warning('登入已過期，正在跳轉...');
                setTimeout(() => {
                    window.location.href = '/login';
                }, 1000);
            }
        } else if (status >= 500 && !isSilent) {
            // 伺服器錯誤：顯示友善訊息
            const friendlyMsg = friendlyErrorMessages[status] || '發生未知錯誤，請稍後再試';
            message.error(friendlyMsg);
        } else if (status >= 400 && status < 500 && !isSilent) {
            // 客戶端錯誤：嘗試淨化後端回傳的錯誤訊息
            const rawError = error.response?.data?.error || error.response?.data?.message || '';
            if (rawError) {
                const sanitized = sanitizeErrorMessage(rawError);
                // 不在這裡顯示，讓元件自己處理，但儲存淨化後的訊息
                error.sanitizedMessage = sanitized;
            }
        } else if (error.code === 'ECONNABORTED') {
            // 請求逾時
            if (!isSilent) {
                message.warning('請求逾時，請檢查網路連線');
            }
        } else if (error.code === 'ERR_NETWORK' || error.code === 'ECONNREFUSED') {
            // 網路錯誤
            console.error('Backend connection refused. Please ensure the backend server is running.');
            if (!isSilent) {
                message.error('無法連線至伺服器，請確認後端服務已啟動');
            }
        }

        return Promise.reject(error);
    }
);

/**
 * 取得淨化後的錯誤訊息
 * @param {Error} error - Axios 錯誤物件
 * @returns {string} - 友善的錯誤訊息
 */
export function getSanitizedErrorMessage(error) {
    // 優先使用已淨化的訊息
    if (error.sanitizedMessage) {
        return error.sanitizedMessage;
    }

    // 嘗試淨化後端回傳的訊息
    const rawError = error.response?.data?.error || error.response?.data?.message || error.message || '';
    return sanitizeErrorMessage(rawError);
}

export default api;

