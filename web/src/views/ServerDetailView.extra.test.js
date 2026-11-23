// @vitest-environment jsdom
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import ServerDetailView from './ServerDetailView.vue';
import { useAuthStore } from '../store/auth';
import { useModInstallStore } from '../store/modInstall';
import api from '../api';

const messages = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warning: vi.fn() }));
const push = vi.hoisted(() => vi.fn());
vi.mock('naive-ui', async () => ({ ...await vi.importActual('naive-ui'), useMessage: () => messages }));
vi.mock('vue-router', () => ({ useRouter: () => ({ push }), RouterLink: { name: 'RouterLink', props: ['to'], template: '<a><slot /></a>' } }));
vi.mock('../api', async () => ({ default: { get: vi.fn(), post: vi.fn(), delete: vi.fn() }, getSanitizedErrorMessage: (await vi.importActual('../utils/errorMapping')).extractAndSanitizeError }));
vi.mock('../components/ServerConsole.vue', () => ({ default: { template: '<div />' } }));
vi.mock('../components/MacroPanel.vue', () => ({ default: { template: '<div />' } }));
vi.mock('../components/ModBrowser.vue', () => ({ default: { name: 'ModBrowser', template: '<div />' } }));
vi.mock('../components/ModInstallModal.vue', () => ({ default: { name: 'ModInstallModal', props: ['show', 'mod', 'servers', 'presetServerId', 'installing'], template: '<div />' } }));

const GB = 1024 * 1024 * 1024;
let wrapper;
let pinia;

beforeEach(() => {
    vi.useFakeTimers();
    vi.clearAllMocks();
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})));
    pinia = createPinia();
    setActivePinia(pinia);
});
afterEach(() => { wrapper?.unmount(); vi.useRealTimers(); });

const mountDetail = async ({ detail, status = 'stopped', user = { role: 1, permissions: { server_max_mem: 3 * GB, max_mods_count: 50 } }, extra = {} }) => {
    useAuthStore().user = user;
    api.get.mockImplementation(async url => {
        if (url.includes('/status/')) return { status };
        if (url.includes('/details/')) return { server: { server_id: 'test', display_name: 'My Server', ...detail } };
        for (const [key, value] of Object.entries(extra)) if (url.includes(key)) return value;
        return {};
    });
    wrapper = shallowMount(ServerDetailView, { props: { id: 'test' }, global: {
            plugins: [pinia],
            renderStubDefaultSlot: true,
            stubs: { Card: { props: ['title'], template: '<div><span>{{ title }}</span><slot name="header" /><slot /></div>' } },
        },
    });
    await flushPromises();
    return wrapper.vm.$.setupState;
};

it('shows the per-server memory limit, or unlimited for Root', async () => {
    let state = await mountDetail({ detail: { mod_loader: 'Vanilla' } });
    expect(state.memoryLimitText).toBe('3 GB');
    expect(wrapper.text()).toContain('單台上限 3 GB');
    wrapper.unmount();
    state = await mountDetail({ detail: { mod_loader: 'Vanilla' }, user: { role: 6, permissions: { unlimited: true } } });
    expect(wrapper.text()).toContain('單台上限 不限');
});

it('shows MC version for Vanilla and both versions for Fabric', async () => {
    let state = await mountDetail({ detail: { mod_loader: 'Vanilla', mc_version: '1.21.4', loader_version: '' } });
    expect(state.displayMcVersion).toBe('1.21.4');
    expect(state.displayLoaderVersion).toBe('');
    expect(wrapper.text()).toContain('MC VERSION');
    expect(wrapper.text()).not.toContain('LOADER VERSION');
    wrapper.unmount();
    state = await mountDetail({ detail: { mod_loader: 'Fabric', mc_version: '1.21.4', loader_version: '0.16.9' } });
    expect(wrapper.text()).toContain('1.21.4');
    expect(wrapper.text()).toContain('LOADER VERSION');
    expect(wrapper.text()).toContain('0.16.9');
});

it('explains the Root-only JVM card and hides it from others', async () => {
    await mountDetail({ detail: { mod_loader: 'Vanilla' }, user: { role: 6, permissions: { unlimited: true } } });
    expect(wrapper.text()).toContain('只有 Root 可以修改；參數錯誤可能導致伺服器無法啟動');
    wrapper.unmount();
    await mountDetail({ detail: { mod_loader: 'Vanilla' } });
    expect(wrapper.text()).not.toContain('JVM ARGS');
    expect(wrapper.text()).not.toContain('只有 Root 可以修改');
});

it('shows installed mods against the mod limit and the add button only for modded servers', async () => {
    const state = await mountDetail({
        detail: { mod_loader: 'Fabric' },
        extra: { '/mod/list/': [{ mod_id: 'a', name: 'A', enabled: true }, { mod_id: 'b', name: 'B', enabled: true }] },
    });
    state.handleTabChange('mods');
    await flushPromises();
    expect(wrapper.find('.mods-limit').text()).toBe('已安裝 2 / 50');
    expect(wrapper.find('.mods-count').text()).toBe('2 / 2');
    expect(wrapper.find('.add-mod-btn').exists()).toBe(true);
    wrapper.unmount();

    const root = await mountDetail({ detail: { mod_loader: 'Vanilla' }, user: { role: 6, permissions: { unlimited: true } } });
    root.handleTabChange('mods');
    await flushPromises();
    expect(wrapper.find('.mods-limit').text()).toBe('已安裝 0 / 不限');
    expect(wrapper.find('.add-mod-btn').exists()).toBe(false);
});

it('installs a mod into this server through the shared queue and reloads the list', async () => {
    const state = await mountDetail({ detail: { mod_loader: 'Fabric', mc_version: '1.21.4' } });
    const modal = wrapper.findComponent({ name: 'ModInstallModal' });
    expect(modal.props('presetServerId')).toBe('test');
    expect(modal.props('servers')).toEqual([expect.objectContaining({ server_id: 'test' })]);
    api.post.mockResolvedValue({ job_id: 'job-1', status: 'queued' });
    await state.handleModInstall({ mod: { project_id: 'sodium', title: 'Sodium' }, server: state.serverDetails, versionId: '' });
    expect(api.post).toHaveBeenCalledWith('/api/v1/server/mod/add/test?async=true', { mod_id: 'sodium', version_id: '', auto_update: true }, { silent: true });
    const installs = useModInstallStore();
    expect(installs.jobs[0]).toMatchObject({ jobId: 'job-1', serverId: 'test', modTitle: 'Sodium' });
    api.get.mockClear();
    installs.updateJob('job-1', { status: 'completed' });
    await flushPromises();
    expect(api.get).toHaveBeenCalledWith('/api/v1/server/mod/list/test', { silent: true });
});

it('deletes the server only after typing its name and returns to the list', async () => {
    const state = await mountDetail({ detail: { mod_loader: 'Vanilla' } });
    state.openDeleteModal();
    expect(state.showDeleteModal).toBe(true);
    state.deleteConfirmText = 'wrong';
    await state.confirmDeleteServer();
    expect(api.delete).not.toHaveBeenCalled();
    state.deleteConfirmText = 'My Server';
    api.delete.mockResolvedValue({});
    await state.confirmDeleteServer();
    expect(api.delete).toHaveBeenCalledWith('/api/v1/server/del/test', { silent: true });
    expect(push).toHaveBeenCalledWith({ name: 'Servers' });
});

it('disables deletion while running or installing', async () => {
    let state = await mountDetail({ detail: { mod_loader: 'Vanilla' }, status: 'running' });
    expect(state.deleteDisabledReason).toBe('請先停止伺服器再刪除');
    expect(wrapper.find('.delete-server-btn').attributes('disabled')).toBeDefined();
    state.openDeleteModal();
    expect(state.showDeleteModal).toBe(false);
    wrapper.unmount();
    state = await mountDetail({ detail: { mod_loader: 'Vanilla', install_status: 'running', ins_ses_id: 'create-1' } });
    expect(state.deleteDisabledReason).toBe('安裝中無法刪除');
    state.deleteConfirmText = 'My Server';
    await state.confirmDeleteServer();
    expect(api.delete).not.toHaveBeenCalled();
});

it('shows a backend delete error through errorMapping', async () => {
    const state = await mountDetail({ detail: { mod_loader: 'Vanilla' } });
    state.openDeleteModal();
    state.deleteConfirmText = 'My Server';
    api.delete.mockRejectedValue({ response: { status: 409, data: { error: '請先停止伺服器', code: 'SERVER_RUNNING' } } });
    await state.confirmDeleteServer();
    expect(messages.error).toHaveBeenCalledWith('刪除失敗: 請先停止伺服器');
    expect(push).not.toHaveBeenCalled();
});

it('shows install progress, then allows starting once completed', async () => {
    const state = await mountDetail({ detail: { mod_loader: 'NeoForge', install_status: 'running', ins_ses_id: 'create-1' } });
    expect(state.isInstalling).toBe(true);
    expect(state.actionsDisabled).toBe(true);
    expect(wrapper.findComponent({ name: 'Alert' }).props('title')).toBe('安裝中');
    const installs = useModInstallStore();
    expect(installs.jobs.map(j => j.jobId)).toEqual(['create-1']);
    installs.updateJob('create-1', { percent: 40, message: 'running neoforge installer' });
    await flushPromises();
    expect(wrapper.text()).toContain('進度 40%');

    api.get.mockImplementation(async url => {
        if (url.includes('/status/')) return { status: 'stopped' };
        if (url.includes('/details/')) return { server: { server_id: 'test', display_name: 'My Server', install_status: 'completed', ins_ses_id: 'create-1' } };
        return {};
    });
    installs.updateJob('create-1', { status: 'completed' });
    await flushPromises();
    expect(state.isInstalling).toBe(false);
    expect(state.actionsDisabled).toBe(false);
});

it('shows the install error and a delete button when installation failed', async () => {
    const state = await mountDetail({
        detail: { mod_loader: 'Vanilla', install_status: 'failed', ins_ses_id: 'create-2' },
        extra: { '/mod/job/create-2': { status: 'failed', error: 'sha1 mismatch' } },
    });
    expect(state.installFailed).toBe(true);
    expect(state.actionsDisabled).toBe(true);
    expect(wrapper.findComponent({ name: 'Alert' }).props('title')).toBe('安裝失敗');
    expect(wrapper.text()).toContain('sha1 mismatch');
    expect(wrapper.find('.install-delete-btn').exists()).toBe(true);
    expect(state.deleteDisabledReason).toBe('');
});
