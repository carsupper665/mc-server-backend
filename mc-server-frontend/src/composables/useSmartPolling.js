import { ref, onBeforeUnmount } from 'vue';
import { isPaused as isRateLimitPaused } from './useRateLimitGuard';

/**
 * 智慧輪詢 Composable
 * 
 * 提供兩種模式：
 * - 閒置模式：每 10-15 秒輪詢一次
 * - 活躍模式：每 2 秒輪詢一次（啟動後 30 秒自動恢復閒置模式）
 * 
 * 整合 Rate Limit Guard:
 * - 當 isRateLimitPaused 為 true 時，跳過輪詢
 * 
 * @param {Function} fetchFn - 要執行的輪詢函式
 * @param {Object} options - 設定選項
 * @param {number} options.idleInterval - 閒置模式間隔 (ms)，預設 12000
 * @param {number} options.activeInterval - 活躍模式間隔 (ms)，預設 2000
 * @param {number} options.activeDuration - 活躍模式持續時間 (ms)，預設 30000
 * @param {Function} options.onStatusChange - 狀態變更回調 (可用於提前退出活躍模式)
 */
export function useSmartPolling(fetchFn, options = {}) {
    const idleInterval = options.idleInterval || 12000; // 12 秒
    const activeInterval = options.activeInterval || 2000; // 2 秒
    const activeDuration = options.activeDuration || 30000; // 30 秒

    let intervalId = null;
    let activeTimeoutId = null;
    let previousStatus = null;

    const isActive = ref(false);
    const isPolling = ref(false);

    /**
     * 執行一次輪詢並檢查狀態變更
     */
    const poll = async () => {
        // Rate Limit Guard: 暫停期間跳過輪詢
        if (isRateLimitPaused.value) {
            console.log('[SmartPolling] Skipping poll - rate limit backoff in progress');
            return;
        }

        try {
            const result = await fetchFn();

            // 如果提供了狀態變更回調，檢查是否需要提前退出活躍模式
            if (options.onStatusChange && previousStatus !== null) {
                const currentStatus = typeof result === 'string' ? result : result?.status;
                if (currentStatus && currentStatus !== previousStatus) {
                    options.onStatusChange(currentStatus, previousStatus);
                    // 狀態已變更，退出活躍模式
                    if (isActive.value) {
                        exitActiveMode();
                    }
                }
                previousStatus = currentStatus;
            }

            return result;
        } catch (error) {
            console.error('[SmartPolling] 輪詢失敗:', error);
            throw error;
        }
    };

    /**
     * 開始輪詢（閒置模式）
     */
    const startPolling = () => {
        if (isPolling.value) return;

        isPolling.value = true;
        isActive.value = false;

        // 立即執行一次
        poll();

        // 設定閒置間隔
        intervalId = setInterval(poll, idleInterval);
    };

    /**
     * 進入活躍模式（高頻輪詢）
     */
    const enterActiveMode = () => {
        if (!isPolling.value) {
            startPolling();
        }

        isActive.value = true;

        // 清除現有的間隔
        if (intervalId) {
            clearInterval(intervalId);
        }

        // 設定高頻輪詢
        intervalId = setInterval(poll, activeInterval);

        // 清除之前的超時
        if (activeTimeoutId) {
            clearTimeout(activeTimeoutId);
        }

        // 設定自動退出活躍模式
        activeTimeoutId = setTimeout(() => {
            exitActiveMode();
        }, activeDuration);
    };

    /**
     * 退出活躍模式，恢復閒置模式
     */
    const exitActiveMode = () => {
        if (!isActive.value) return;

        isActive.value = false;

        // 清除現有的間隔和超時
        if (intervalId) {
            clearInterval(intervalId);
        }
        if (activeTimeoutId) {
            clearTimeout(activeTimeoutId);
            activeTimeoutId = null;
        }

        // 恢復閒置間隔
        intervalId = setInterval(poll, idleInterval);
    };

    /**
     * 停止所有輪詢
     */
    const stopPolling = () => {
        isPolling.value = false;
        isActive.value = false;
        previousStatus = null;

        if (intervalId) {
            clearInterval(intervalId);
            intervalId = null;
        }
        if (activeTimeoutId) {
            clearTimeout(activeTimeoutId);
            activeTimeoutId = null;
        }
    };

    /**
     * 立即執行一次輪詢
     */
    const pollNow = () => {
        return poll();
    };

    // 組件卸載時自動清理
    onBeforeUnmount(() => {
        stopPolling();
    });

    return {
        isActive,
        isPolling,
        startPolling,
        stopPolling,
        enterActiveMode,
        exitActiveMode,
        pollNow
    };
}
