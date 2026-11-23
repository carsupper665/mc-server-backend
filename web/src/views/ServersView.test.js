// @vitest-environment jsdom
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import ServersView from './ServersView.vue';
import { useAuthStore } from '../store/auth';
import { useVersionCacheStore } from '../store/versionCache';
import { useModInstallStore } from '../store/modInstall';
import api from '../api';

vi.mock('naive-ui', async () => ({ ...await vi.importActual('naive-ui'), useMessage: () => ({ error: vi.fn(), success: vi.fn(), warning: vi.fn() }) }));
vi.mock('vue-router', () => ({ useRouter: () => ({ push: vi.fn() }) }));
vi.mock('../api', () => ({ default: { get: vi.fn(), post: vi.fn() }, getSanitizedErrorMessage: vi.fn() }));
vi.mock('../composables/useMobileLayout', () => ({ useMobileLayout: () => false }));

let wrapper;
let pinia;
beforeEach(() => {
    vi.clearAllMocks();
    pinia = createPinia();
    setActivePinia(pinia);
    useVersionCacheStore().vanillaVersions = [{ label: '1.21', value: '1.21' }];
    // Install progress streams never connect in tests.
    vi.stubGlobal('fetch', vi.fn(() => new Promise(() => {})));
});
afterEach(() => wrapper?.unmount());

it.each([
    { count: 4, unlimited: false, disabled: false, limit: '5' },
    { count: 5, unlimited: false, disabled: true, limit: '5' },
    { count: 6, unlimited: false, disabled: true, limit: '5' },
    { count: 50, unlimited: true, disabled: false, limit: '不限' },
])('shows $count / $limit and disables creation: $disabled', async ({ count, unlimited, disabled, limit }) => {
    useAuthStore().user = { permissions: { max_server_count: unlimited ? 0 : 5, can_create_server: true, unlimited } };
    api.get.mockResolvedValue(Array.from({ length: count }, (_, i) => ({ server_id: `server${i}`, install_status: 'installing' })));
    wrapper = shallowMount(ServersView, { global: { plugins: [pinia], renderStubDefaultSlot: true } });
    await flushPromises();
    expect(wrapper.text()).toContain(`伺服器數 ${count} / ${limit}`);
    const deploy = wrapper.findAllComponents({ name: 'Button' }).find(button => button.text().includes('DEPLOY NEW SERVER'));
    expect(deploy.props('disabled')).toBe(disabled);
    if (!unlimited && count === 5) {
        api.get.mockResolvedValue([]);
        await wrapper.findAllComponents({ name: 'Button' }).find(button => button.text().includes('REFRESH')).vm.$emit('click');
        await flushPromises();
        expect(wrapper.text()).toContain('伺服器數 0 / 5');
        expect(deploy.props('disabled')).toBe(false);
    }
});

it('shows the loader select only for Fabric, defaults to the latest stable and sends fabric_loader', async () => {
    useAuthStore().user = { permissions: { unlimited: true } };
    useVersionCacheStore().fabricVersions = [{ label: '1.21.4', value: '1.21.4' }];
    api.get.mockImplementation(async (url) => {
        if (url === '/api/v1/public/versions/fabric') {
            return { versions: [{ id: '0.17.0-beta.1', stable: false }, { id: '0.16.9', stable: true, latest: true }] };
        }
        return [];
    });
    api.post.mockResolvedValue({ server_id: 'mcsfv-1.21.4-1234-OID-1', ins_ses_id: 'create-1' });
    wrapper = shallowMount(ServersView, { global: { plugins: [pinia], renderStubDefaultSlot: true } });
    await flushPromises();
    const state = wrapper.vm.$.setupState;
    state.showCreateModal = true;
    await flushPromises();
    expect(state.needsLoader).toBe(false);
    const loaderItem = () => wrapper.findAllComponents({ name: 'FormItem' }).find(item => item.props('label') === 'FABRIC VERSION');
    expect(loaderItem()).toBeUndefined();

    state.createForm.server_type = 'Fabric';
    await flushPromises();
    state.createForm.server_ver = '1.21.4';
    await flushPromises();
    expect(state.needsLoader).toBe(true);
    expect(loaderItem()).toBeDefined();
    expect(state.createForm.loader_version).toBe('0.16.9');
    expect(state.loaderOptions.map(o => o.value)).toEqual(['0.17.0-beta.1', '0.16.9']);

    state.createForm.display_name = 'fab';
    await state.handleCreate();
    expect(api.post).toHaveBeenCalledWith('/api/v1/server/create',
        { server_type: 'Fabric', server_ver: '1.21.4', display_name: 'fab', fabric_loader: '0.16.9' },
        { silent: true });
    const job = useModInstallStore().jobs.find(j => j.jobId === 'create-1');
    expect(job).toMatchObject({ serverId: 'mcsfv-1.21.4-1234-OID-1', serverName: 'fab' });
    expect(state.showCreateModal).toBe(false);
});

it('sends loader_version for Forge and labels install states', async () => {
    useAuthStore().user = { permissions: { unlimited: true } };
    useVersionCacheStore().forgeVersions = [{ label: '1.20.1', value: '1.20.1' }];
    api.get.mockImplementation(async (url) => {
        if (url === '/api/v1/public/versions/forge') {
            return { versions: [{ id: '47.4.10', stable: true, latest: true }, { id: '47.3.0', stable: true, recommended: true }] };
        }
        if (url === '/user/myservers') {
            return [
                { server_id: 'mcsgv-1.20.1-0001-OID-1', install_status: 'running', ins_ses_id: 'create-a', display_name: 'a' },
                { server_id: 'mcsnv-1.21.1-0002-OID-1', install_status: 'failed', ins_ses_id: 'create-b', display_name: 'b' },
                { server_id: 'mcsvv-1.21.4-0003-OID-1', install_status: 'completed', ins_ses_id: 'create-c', display_name: 'c' },
            ];
        }
        // 安裝完成、正在執行的伺服器：執行狀態也叫 'running'，不能被當成安裝中
        if (url === '/api/v1/server/status/mcsvv-1.21.4-0003-OID-1') return { status: 'running' };
        return {};
    });
    api.post.mockResolvedValue({ server_id: 'x', ins_ses_id: 'create-x' });
    wrapper = shallowMount(ServersView, { global: { plugins: [pinia], renderStubDefaultSlot: true } });
    await flushPromises();
    const state = wrapper.vm.$.setupState;
    expect(state.servers.map(s => [s.type, state.statusLabel(s.status)])).toEqual([['Forge', '安裝中'], ['NeoForge', '安裝失敗'], ['Vanilla', 'running']]);
    expect(state.statusTagType(state.servers[2].status)).toBe('success');
    // Installing servers keep showing progress after a reload.
    expect(useModInstallStore().jobs.map(j => j.jobId)).toEqual(['create-a']);

    state.createForm.server_type = 'Forge';
    await flushPromises();
    state.createForm.server_ver = '1.20.1';
    await flushPromises();
    expect(state.createForm.loader_version).toBe('47.3.0');
    state.createForm.display_name = 'forge';
    await state.handleCreate();
    expect(api.post).toHaveBeenCalledWith('/api/v1/server/create',
        { server_type: 'Forge', server_ver: '1.20.1', display_name: 'forge', loader_version: '47.3.0' },
        { silent: true });
});
