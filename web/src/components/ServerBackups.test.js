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

const mountBackups = async (user, backups, props = {}) => {
    api.get.mockResolvedValue({ backups });
    const pinia = createPinia();
    useAuthStore(pinia).user = user;
    const wrapper = shallowMount(ServerBackups, {
        props: { serverId: 'test', ...props },
        global: {
            plugins: [pinia],
            renderStubDefaultSlot: true,
            stubs: { Card: { template: '<div><slot name="header" /><slot name="header-extra" /><slot /></div>' } },
        },
    });
    await flushPromises();
    return wrapper;
};
const vip = { permissions: { can_back_up_server: true, max_back_up_count: 3 } };
const threeBackups = ['20261114_120000', '20261113_120000', '20261115_120000'];

it.each([
    ['allowed', vip, ['20261114_120000'], '1 / 3'],
    ['Root', { permissions: { unlimited: true, can_back_up_server: true } }, ['a', 'b'], '2 / 不限'],
    ['denied', { permissions: { can_back_up_server: false, max_back_up_count: 0 } }, [], null],
])('shows backup count against the limit: %s', async (_, user, backups, text) => {
    const wrapper = await mountBackups(user, backups);
    const count = wrapper.find('.backup-count');
    expect(count.exists()).toBe(text !== null);
    if (text) expect(count.text()).toBe(text);
    wrapper.unmount();
});

it('backs up directly below the limit', async () => {
    const wrapper = await mountBackups(vip, ['20261114_120000']);
    api.post.mockClear();
    await wrapper.vm.$.setupState.handleBackup();
    expect(wrapper.vm.$.setupState.showBackupConfirmModal).toBe(false);
    expect(api.post).toHaveBeenCalledWith('/api/v1/server/backup/test', {}, { timeout: 600000, silent: true });
    wrapper.unmount();
});

it('confirms before rotating the oldest backup and does nothing when cancelled', async () => {
    const wrapper = await mountBackups(vip, threeBackups);
    const state = wrapper.vm.$.setupState;
    api.post.mockClear();
    await state.handleBackup();
    expect(state.showBackupConfirmModal).toBe(true);
    expect(state.backupsToRotate).toEqual(['20261113_120000']);
    expect(wrapper.text()).toContain('已達上限，建立新備份會刪除最舊的備份 20261113_120000');
    state.showBackupConfirmModal = false;
    expect(api.post).not.toHaveBeenCalled();

    await state.handleBackup();
    await state.confirmBackup();
    expect(api.post).toHaveBeenCalledTimes(1);
    wrapper.unmount();
});

it('Root never gets the rotation warning', async () => {
    const wrapper = await mountBackups({ permissions: { unlimited: true, can_back_up_server: true } }, threeBackups);
    await wrapper.vm.$.setupState.handleBackup();
    expect(wrapper.vm.$.setupState.showBackupConfirmModal).toBe(false);
    wrapper.unmount();
});

it('shows a single confirmation when running and at the limit', async () => {
    const stop = vi.fn(done => done());
    const start = vi.fn(done => done());
    const wrapper = await mountBackups(vip, threeBackups, { isServerRunning: true, onRequestStop: stop, onRequestStart: start });
    const state = wrapper.vm.$.setupState;
    await state.handleBackup();
    expect(state.showBackupConfirmModal).toBe(true);
    expect(wrapper.text()).toContain('20261113_120000');
    expect(wrapper.text()).toContain('伺服器目前正在運行中');
    await wrapper.setProps({ isServerRunning: false });
    await wrapper.setProps({ isServerRunning: true });
    // confirmBackup runs stop → backup → start without another dialog
    const flow = state.confirmBackup();
    await wrapper.setProps({ isServerRunning: false });
    await flow;
    expect(stop).toHaveBeenCalledTimes(1);
    expect(start).toHaveBeenCalledTimes(1);
    expect(state.showBackupConfirmModal).toBe(false);
    wrapper.unmount();
});
