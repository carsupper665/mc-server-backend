/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { ref } from 'vue';

// Mock vue's onBeforeUnmount
vi.mock('vue', async () => {
    const actual = await vi.importActual('vue');
    return {
        ...actual,
        onBeforeUnmount: vi.fn(),
    };
});

// 由於 useSmartPolling 依賴 useRateLimitGuard，需要 mock
vi.mock('./useRateLimitGuard', () => ({
    isPaused: ref(false),
}));

describe('useSmartPolling', () => {
    let useSmartPolling;

    beforeEach(async () => {
        vi.useFakeTimers();
        const module = await import('./useSmartPolling');
        useSmartPolling = module.useSmartPolling;
    });

    afterEach(() => {
        vi.useRealTimers();
        vi.clearAllMocks();
    });

    it('should start in idle mode', () => {
        const fetchFn = vi.fn().mockResolvedValue({});
        const polling = useSmartPolling(fetchFn);

        expect(polling.isPolling.value).toBe(false);
        expect(polling.isActive.value).toBe(false);
    });

    it('should start polling and call fetchFn immediately', async () => {
        const fetchFn = vi.fn().mockResolvedValue({});
        const polling = useSmartPolling(fetchFn);

        polling.startPolling();

        expect(polling.isPolling.value).toBe(true);
        expect(fetchFn).toHaveBeenCalledTimes(1);
    });

    it('should call fetchFn at idle interval', async () => {
        const fetchFn = vi.fn().mockResolvedValue({});
        const polling = useSmartPolling(fetchFn, { idleInterval: 1000 });

        polling.startPolling();
        expect(fetchFn).toHaveBeenCalledTimes(1);

        await vi.advanceTimersByTimeAsync(1000);
        expect(fetchFn).toHaveBeenCalledTimes(2);

        await vi.advanceTimersByTimeAsync(1000);
        expect(fetchFn).toHaveBeenCalledTimes(3);
    });

    it('should switch to active mode with faster interval', async () => {
        const fetchFn = vi.fn().mockResolvedValue({});
        const polling = useSmartPolling(fetchFn, {
            idleInterval: 10000,
            activeInterval: 1000,
        });

        polling.startPolling();
        expect(fetchFn).toHaveBeenCalledTimes(1);

        // 進入活躍模式
        polling.enterActiveMode();
        expect(polling.isActive.value).toBe(true);

        // 活躍模式間隔為 1 秒
        await vi.advanceTimersByTimeAsync(1000);
        expect(fetchFn).toHaveBeenCalledTimes(2);
    });

    it('should automatically exit active mode after duration', async () => {
        const fetchFn = vi.fn().mockResolvedValue({});
        const polling = useSmartPolling(fetchFn, {
            idleInterval: 10000,
            activeInterval: 1000,
            activeDuration: 5000,
        });

        polling.startPolling();
        polling.enterActiveMode();
        expect(polling.isActive.value).toBe(true);

        // 5 秒後應該自動退出活躍模式
        await vi.advanceTimersByTimeAsync(5000);
        expect(polling.isActive.value).toBe(false);
    });

    it('should stop polling completely when stopPolling is called', () => {
        const fetchFn = vi.fn().mockResolvedValue({});
        const polling = useSmartPolling(fetchFn);

        polling.startPolling();
        expect(polling.isPolling.value).toBe(true);

        polling.stopPolling();
        expect(polling.isPolling.value).toBe(false);
        expect(polling.isActive.value).toBe(false);
    });

    it('should call pollNow immediately', async () => {
        const fetchFn = vi.fn().mockResolvedValue({ status: 'Running' });
        const polling = useSmartPolling(fetchFn);

        const result = await polling.pollNow();

        expect(fetchFn).toHaveBeenCalledTimes(1);
        expect(result).toEqual({ status: 'Running' });
    });
});
