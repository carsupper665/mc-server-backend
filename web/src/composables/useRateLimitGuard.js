import { ref } from 'vue';

/**
 * Rate Limit Guard - 全域速率限制守衛
 * 
 * 功能:
 * - 監控 HTTP 429 回應並觸發指數退避
 * - 廣播暫停狀態給所有輪詢系統
 * - 成功回應後自動恢復並重置退避計數
 */

// 全域狀態 (Singleton pattern)
const isPaused = ref(false);
const backoffLevel = ref(0);
const nextRetryTime = ref(null);

// 退避延遲序列 (毫秒)
const BACKOFF_DELAYS = [2000, 4000, 8000, 16000, 30000];

// 正在進行的退避超時
let backoffTimeout = null;

/**
 * 觸發退避機制
 * 當偵測到 429 回應時呼叫
 */
function triggerBackoff() {
    if (backoffTimeout) {
        clearTimeout(backoffTimeout);
    }

    isPaused.value = true;
    const delay = BACKOFF_DELAYS[Math.min(backoffLevel.value, BACKOFF_DELAYS.length - 1)];
    nextRetryTime.value = Date.now() + delay;

    console.log(`[RateLimitGuard] 429 detected, backing off for ${delay / 1000}s (level ${backoffLevel.value})`);

    backoffTimeout = setTimeout(() => {
        isPaused.value = false;
        nextRetryTime.value = null;
        console.log('[RateLimitGuard] Backoff ended, resuming requests');
    }, delay);

    backoffLevel.value++;
}

/**
 * 重置退避計數
 * 當請求成功 (非 429) 時呼叫
 */
function resetBackoff() {
    if (backoffLevel.value > 0) {
        console.log('[RateLimitGuard] Request successful, resetting backoff level');
        backoffLevel.value = 0;
    }
}

/**
 * 取得剩餘等待秒數
 */
function getRemainingSeconds() {
    if (!nextRetryTime.value) return 0;
    return Math.max(0, Math.ceil((nextRetryTime.value - Date.now()) / 1000));
}

/**
 * 強制立即解除暫停 (用於使用者手動重試)
 */
function forceResume() {
    if (backoffTimeout) {
        clearTimeout(backoffTimeout);
        backoffTimeout = null;
    }
    isPaused.value = false;
    nextRetryTime.value = null;
    backoffLevel.value = 0;
    console.log('[RateLimitGuard] Force resumed by user');
}

/**
 * Composable hook
 */
export function useRateLimitGuard() {
    return {
        isPaused,
        backoffLevel,
        triggerBackoff,
        resetBackoff,
        getRemainingSeconds,
        forceResume
    };
}

// 匯出單例函式供 Axios interceptor 使用
export { isPaused, triggerBackoff, resetBackoff };
