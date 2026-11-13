// @vitest-environment jsdom
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import ServerDetailView from './ServerDetailView.vue';
import { useAuthStore } from '../store/auth';
import api from '../api';
const messages = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock('naive-ui', async () => ({ ...await vi.importActual('naive-ui'), useMessage: () => messages }));
vi.mock('../api', async () => ({ default: { get: vi.fn(), post: vi.fn() }, getSanitizedErrorMessage: (await vi.importActual('../utils/errorMapping')).extractAndSanitizeError }));
vi.mock('../components/ServerConsole.vue', () => ({ default: { template: '<div />' } }));
vi.mock('../components/MacroPanel.vue', () => ({ default: { template: '<div />' } }));
let wrapper;
let pinia;
beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    Object.defineProperty(document, 'hidden', { configurable: true, value: false });
    pinia = createPinia();
    setActivePinia(pinia);
    api.get.mockImplementation(async url => url.includes('/status/') ? { status: 'running' } : {});
    wrapper = shallowMount(ServerDetailView, { props: { id: 'test' }, global: { plugins: [pinia] } });
});
afterEach(() => { wrapper.unmount(); vi.useRealTimers(); });
it('appends only new logs, resets replacements and skips hidden polls', async () => {
    await flushPromises();
    const state = wrapper.vm.$.setupState;
    const writeLog = vi.fn();
    const clear = vi.fn();
    state.consoleRef = { term: { clear }, writeLog };
    api.get.mockResolvedValue({ logs: 'first\r\n' });
    await state.fetchLogs();
    writeLog('[MACRO] local\r\n');
    await state.fetchLogs();
    expect(writeLog).toHaveBeenCalledTimes(2);
    api.get.mockResolvedValue({ logs: 'first\r\nsecond\r\n' });
    await state.fetchLogs();
    expect(writeLog).toHaveBeenLastCalledWith('second\r\n');
    expect(clear).not.toHaveBeenCalled();
    api.get.mockResolvedValue({ logs: 'replacement' });
    await state.fetchLogs();
    expect(clear).toHaveBeenCalledTimes(1);
    expect(writeLog).toHaveBeenLastCalledWith('replacement');
    api.get.mockClear();
    Object.defineProperty(document, 'hidden', { configurable: true, value: true });
    await state.fetchLogs();
    expect(api.get).not.toHaveBeenCalled();
});
it('unlocks the buttons when a started server never reports running', async () => {
    api.get.mockImplementation(async url => url.includes('/status/') ? { status: 'stopped' } : {});
    await flushPromises();
    const state = wrapper.vm.$.setupState;
    await state.fetchServerDetail();
    api.post.mockResolvedValue({});
    await state.handleStart();
    await state.fetchServerDetail();
    expect(state.actionsDisabled).toBe(true);
    await vi.advanceTimersByTimeAsync(30000);
    expect(state.actionsDisabled).toBe(false);
});
it('blocks duplicate actions and restores backend status on failed stop', async () => {
    await flushPromises();
    const state = wrapper.vm.$.setupState;
    let rejectStop;
    api.post.mockImplementation(() => new Promise((resolve, reject) => { rejectStop = reject; }));
    const stopping = state.handleStop();
    expect(state.actionsDisabled).toBe(true);
    await state.handleStart();
    await state.handleStop();
    expect(api.post).toHaveBeenCalledTimes(1);
    rejectStop(new Error('stop failed'));
    await stopping;
    expect(state.server.status).toBe('running');
    expect(state.actionsDisabled).toBe(false);
    expect(messages.error).toHaveBeenCalledTimes(1);
});

it('loads memory inputs and saves xmx without empty xms', async () => {
    const gb = 1024 * 1024 * 1024;
    api.get.mockImplementation(async url => {
        if (url.includes('/status/')) return { status: 'stopped' };
        if (url.includes('/details/')) {
            return { server: { server_id: 'test', display_name: 'n', mod_loader: 'Vanilla', xmx: 2 * gb, xms: 1 * gb } };
        }
        return {};
    });
    wrapper.unmount();
    pinia = createPinia();
    setActivePinia(pinia);
    wrapper = shallowMount(ServerDetailView, { props: { id: 'test' }, global: { plugins: [pinia] } });
    await flushPromises();

    const state = wrapper.vm.$.setupState;
    expect(state.memoryXmx).toBe('2G');
    expect(state.memoryXms).toBe('1G');

    state.memoryXms = '';
    api.post.mockResolvedValue({ message: 'ok' });
    await state.saveMemory();
    expect(api.post).toHaveBeenCalledWith(
        '/api/v1/server/memory/test',
        { xmx: '2G' },
        { silent: true },
    );
    expect(messages.success).toHaveBeenCalledWith(expect.stringContaining('下次啟動生效'));
});

it('shows JVM args editor only for Root and saves args', async () => {
    api.get.mockImplementation(async url => {
        if (url.includes('/status/')) return { status: 'stopped' };
        if (url.includes('/details/')) {
            return { server: { server_id: 'test', display_name: 'n', mod_loader: 'Vanilla', args: '-XX:+UseG1GC' } };
        }
        return {};
    });

    wrapper.unmount();
    pinia = createPinia();
    setActivePinia(pinia);
    useAuthStore().user = { role: 1, username: 'member' };
    wrapper = shallowMount(ServerDetailView, { props: { id: 'test' }, global: { plugins: [pinia] } });
    await flushPromises();
    expect(wrapper.vm.$.setupState.isRoot).toBe(false);

    wrapper.unmount();
    pinia = createPinia();
    setActivePinia(pinia);
    useAuthStore().user = { role: 6, username: 'root' };
    wrapper = shallowMount(ServerDetailView, { props: { id: 'test' }, global: { plugins: [pinia] } });
    await flushPromises();

    const state = wrapper.vm.$.setupState;
    expect(state.isRoot).toBe(true);
    expect(state.jvmArgs).toBe('-XX:+UseG1GC');

    state.jvmArgs = '-Xlog:gc';
    api.post.mockResolvedValue({ message: 'ok' });
    await state.saveJvmArgs();
    expect(api.post).toHaveBeenCalledWith(
        '/api/v1/server/args/test',
        { args: '-Xlog:gc' },
        { silent: true },
    );
    expect(messages.success).toHaveBeenCalledWith(expect.stringContaining('下次啟動生效'));
});


it.each([403, 404])('hides server controls and links to the server list on details %s', async status => {
    await flushPromises();
    wrapper.unmount();
    api.get.mockImplementation(async url => {
        if (url.includes('/details/')) throw { response: { status } };
        return { status: 'stopped' };
    });
    wrapper = shallowMount(ServerDetailView, {
        props: { id: 'root-server' },
        global: { plugins: [pinia], renderStubDefaultSlot: true },
    });
    await flushPromises();
    expect(wrapper.text()).toContain('找不到伺服器或沒有權限');
    expect(wrapper.findComponent({ name: 'RouterLink' }).props('to')).toBe('/servers');
    expect(wrapper.text()).toContain('返回伺服器列表');
    expect(wrapper.text()).not.toContain('START');
    expect(wrapper.text()).not.toContain('STOP');
    expect(wrapper.text()).not.toContain('MEMORY');
    expect(wrapper.findComponent({ name: 'ServerConsole' }).exists()).toBe(false);
    api.get.mockClear();
    await vi.advanceTimersByTimeAsync(15000);
    expect(api.get).not.toHaveBeenCalled();
    expect(messages.error).not.toHaveBeenCalled();
});


it('keeps controls hidden until server details finish loading', async () => {
    await flushPromises();
    wrapper.unmount();
    let resolveDetails;
    api.get.mockImplementation(async url => {
        if (url.includes('/details/')) return new Promise(resolve => { resolveDetails = resolve; });
        return { status: 'stopped' };
    });
    wrapper = shallowMount(ServerDetailView, {
        props: { id: 'test' },
        global: { plugins: [pinia], renderStubDefaultSlot: true },
    });
    await flushPromises();
    expect(wrapper.find('.server-header').exists()).toBe(false);
    resolveDetails({ server: { server_id: 'test' } });
    await flushPromises();
    expect(wrapper.find('.server-header').exists()).toBe(true);
    expect(wrapper.text()).toContain('START');
});
