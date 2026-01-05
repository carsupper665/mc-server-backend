/**
 * @vitest-environment jsdom
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { useRateLimitGuard } from './useRateLimitGuard';

describe('useRateLimitGuard', () => {
    let guard;

    beforeEach(() => {
        vi.useFakeTimers();
        guard = useRateLimitGuard();
        // 確保每次測試開始時狀態是乾淨的
        guard.forceResume();
    });

    afterEach(() => {
        guard.forceResume();
        vi.useRealTimers();
        vi.clearAllMocks();
    });

    it('should start with isPaused = false', () => {
        expect(guard.isPaused.value).toBe(false);
    });

    it('should set isPaused to true when triggerBackoff is called', () => {
        guard.triggerBackoff();
        expect(guard.isPaused.value).toBe(true);
    });

    it('should automatically resume after backoff delay', async () => {
        guard.triggerBackoff();
        expect(guard.isPaused.value).toBe(true);

        // 第一次退避延遲是 2000ms
        await vi.advanceTimersByTimeAsync(2000);
        expect(guard.isPaused.value).toBe(false);
    });

    it('should reset backoff level when resetBackoff is called', () => {
        guard.triggerBackoff();
        expect(guard.backoffLevel.value).toBeGreaterThan(0);

        guard.resetBackoff();
        expect(guard.backoffLevel.value).toBe(0);
    });

    it('should return remaining seconds', () => {
        guard.triggerBackoff();
        const remaining = guard.getRemainingSeconds();
        expect(remaining).toBeGreaterThan(0);
        expect(remaining).toBeLessThanOrEqual(2);
    });
});
