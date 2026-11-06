// @vitest-environment jsdom
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import ServerBackups from './ServerBackups.vue';
import api from '../api';

const messages = vi.hoisted(() => ({ info: vi.fn(), error: vi.fn(), success: vi.fn() }));
vi.mock('naive-ui', async () => ({ ...await vi.importActual('naive-ui'), useMessage: () => messages }));
vi.mock('../api', async () => ({ default: { post: vi.fn() }, getSanitizedErrorMessage: (await vi.importActual('../utils/errorMapping')).extractAndSanitizeError }));
beforeEach(() => { vi.clearAllMocks(); api.post.mockResolvedValue({ backups: [] }); });
afterEach(() => { vi.useRealTimers(); });

it('aborts immediately on stop failure without creating a backup or claiming restart', async () => {
    const wrapper = shallowMount(ServerBackups, { props: {
        serverId: 'test', isServerRunning: true,
        onRequestStop: done => done(new Error('stop failed')),
    }, global: { plugins: [createPinia()] } });
    await flushPromises();
    api.post.mockClear();
    await wrapper.vm.$.setupState.executeFullBackupFlow();
    expect(api.post).not.toHaveBeenCalled();
    expect(messages.error).toHaveBeenCalledWith('備份流程發生錯誤: stop failed');
    expect(messages.success).not.toHaveBeenCalled();
    wrapper.unmount();
});

it('reports restart failure without announcing a completed restart', async () => {
    const wrapper = shallowMount(ServerBackups, { props: {
        serverId: 'test', isServerRunning: false,
        onRequestStop: done => done(),
        onRequestStart: done => done(new Error('start failed')),
    }, global: { plugins: [createPinia()] } });
    await flushPromises();
    await wrapper.vm.$.setupState.executeFullBackupFlow();
    expect(messages.error).toHaveBeenCalledWith('備份流程發生錯誤: start failed');
    expect(messages.success).not.toHaveBeenCalledWith('備份流程完成！伺服器正在重新啟動。');
    expect(api.post).toHaveBeenCalledWith('/api/v1/server/backup/test', {}, { timeout: 600000, silent: true });
    wrapper.unmount();
});
