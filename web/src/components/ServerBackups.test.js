// @vitest-environment jsdom
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia } from 'pinia';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import ServerBackups from './ServerBackups.vue';
import api from '../api';
import { useAuthStore } from '../store/auth';

const messages = vi.hoisted(() => ({ info: vi.fn(), error: vi.fn(), success: vi.fn() }));
vi.mock('naive-ui', async () => ({ ...await vi.importActual('naive-ui'), useMessage: () => messages }));
vi.mock('../api', async () => ({ default: { get: vi.fn(), post: vi.fn() }, getSanitizedErrorMessage: (await vi.importActual('../utils/errorMapping')).extractAndSanitizeError }));
beforeEach(() => { vi.clearAllMocks(); api.get.mockResolvedValue({ backups: [] }); api.post.mockResolvedValue({}); });
afterEach(() => { vi.useRealTimers(); });

it.each([
    ['missing profile', null, false],
    ['missing permissions', {}, false],
    ['denied', { permissions: { can_back_up_server: false } }, false],
    ['allowed', { permissions: { can_back_up_server: true } }, true],
    ['Root', { permissions: { unlimited: true, can_back_up_server: true } }, true],
])('shows the backup button according to profile permissions: %s', async (_, user, allowed) => {
    const pinia = createPinia();
    useAuthStore(pinia).user = user;
    const wrapper = shallowMount(ServerBackups, {
        props: { serverId: 'test' },
        global: {
            plugins: [pinia],
            stubs: { Card: { template: '<div><slot name="header-extra" /><slot /></div>' } },
        },
    });
    await flushPromises();
    const button = wrapper.findAllComponents({ name: 'Button' }).find(button => button.attributes('title') === '建立新備份');
    expect(!!button).toBe(allowed);
    if (allowed) {
        button.vm.$emit('click');
        await flushPromises();
        expect(api.post).toHaveBeenCalledWith('/api/v1/server/backup/test', {}, { timeout: 600000, silent: true });
    }
    wrapper.unmount();
});

it.each([
    ['missing profile', null, false],
    ['missing permissions', {}, false],
    ['denied', { permissions: { can_back_up_server: false } }, false],
    ['allowed', { permissions: { can_back_up_server: true } }, true],
    ['Root', { permissions: { unlimited: true, can_back_up_server: true } }, true],
])('shows restore only when backup is allowed: %s', async (_, user, allowed) => {
    api.get.mockResolvedValue({ backups: ['20261114_120000'] });
    const pinia = createPinia();
    useAuthStore(pinia).user = user;
    const wrapper = shallowMount(ServerBackups, {
        props: { serverId: 'test' },
        global: {
            plugins: [pinia],
            stubs: { Card: { template: '<div><slot name="header-extra" /><slot /></div>' } },
        },
    });
    await flushPromises();
    const restores = wrapper.findAll('.backup-item').flatMap(item => item.findAllComponents({ name: 'Button' }).filter(button => button.attributes('type') === 'warning'));
    expect(restores).toHaveLength(allowed ? 1 : 0);
    wrapper.unmount();
});

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
