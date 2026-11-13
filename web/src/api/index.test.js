// @vitest-environment jsdom
import { beforeEach, expect, it, vi } from 'vitest';
const messages = vi.hoisted(() => ({ error: vi.fn(), warning: vi.fn() }));
vi.mock('naive-ui', () => ({ darkTheme: 'dark', createDiscreteApi: vi.fn(() => ({ message: messages })) }));
import { createDiscreteApi } from 'naive-ui';
import api, { getSanitizedErrorMessage } from './index';

beforeEach(() => { messages.error.mockClear(); messages.warning.mockClear(); });
const fail = async (url, config = {}) => {
    const error = { response: { status: 500, data: { error: { message: 'panic: failed' } } }, config: { url, ...config } };
    await expect(api.interceptors.response.handlers[0].rejected(error)).rejects.toBe(error);
    return error;
};
it('uses the dark theme', () => {
    expect(createDiscreteApi).toHaveBeenCalledWith(['message'], { configProviderProps: { theme: 'dark' } });
});
it('keeps status and log failures silent', async () => {
    await fail('/api/v1/server/status/test');
    await fail('/api/v1/server/log/test');
    expect(messages.error).not.toHaveBeenCalled();
});
it('lets callers own notifications while preserving default notifications', async () => {
    const error = await fail('/api/v1/server/start/test', { silent: true });
    expect(messages.error).not.toHaveBeenCalled();
    expect(getSanitizedErrorMessage(error)).toBe('伺服器發生嚴重錯誤，請聯絡管理員');
    await fail('/api/v1/server/start/test');
    expect(messages.error).toHaveBeenCalledTimes(1);
});

it.each([
    ['PERM_SERVER_LIMIT', '已達伺服器數量上限'],
    ['PERM_ONLINE_LIMIT', '已達同時運行伺服器數量上限'],
    ['PERM_MEMORY_LIMIT', '記憶體設定超過單台上限，請調小記憶體'],
    ['PERM_MOD_LIMIT', '已達模組數量上限'],
    ['PERM_BACKUP_DENIED', '沒有建立備份的權限'],
    ['PERM_ROLE_UNDEFINED', '角色未定義，無法執行此操作'],
    ['PERM_EDIT_ARGS', '只有 Root 可以修改 JVM 參數'],
])('maps %s consistently for normal and silent requests', async (code, expected) => {
    for (const silent of [false, true]) {
        const error = {
            response: { status: 403, data: { error: 'permission denied', code } },
            config: { url: '/api/v1/server/start/test', silent },
        };
        await expect(api.interceptors.response.handlers[0].rejected(error)).rejects.toBe(error);
        if (!silent) expect(error.sanitizedMessage).toBe(expected);
        expect(getSanitizedErrorMessage(error)).toBe(expected);
    }
});
