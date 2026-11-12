// @vitest-environment jsdom
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import ServersView from './ServersView.vue';
import { useAuthStore } from '../store/auth';
import { useVersionCacheStore } from '../store/versionCache';
import api from '../api';

vi.mock('naive-ui', async () => ({ ...await vi.importActual('naive-ui'), useMessage: () => ({ error: vi.fn() }) }));
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
