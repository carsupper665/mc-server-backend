// @vitest-environment jsdom
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import DashboardView from './DashboardView.vue';
import { useAuthStore } from '../store/auth';
import { useVersionCacheStore } from '../store/versionCache';
import api from '../api';

vi.mock('naive-ui', async () => ({ ...await vi.importActual('naive-ui'), useMessage: () => ({ error: vi.fn() }) }));
vi.mock('../api', () => ({ default: { get: vi.fn(), post: vi.fn() } }));
vi.mock('../components/MetricsChart.vue', () => ({ default: { template: '<div />' } }));

let wrapper;
let pinia;
beforeEach(() => {
    vi.clearAllMocks();
    pinia = createPinia();
    setActivePinia(pinia);
    useVersionCacheStore().prefetchVersions = vi.fn();
    api.get.mockImplementation(async url => {
        if (url === '/user/myservers') return [{ server_id: 'a' }, { server_id: 'b' }, { server_id: 'c' }];
        return { status: url.endsWith('/c') ? 'stopped' : 'running' };
    });
});
afterEach(() => wrapper?.unmount());

const statistic = label => wrapper.findAllComponents({ name: 'Statistic' }).find(s => s.props('label') === label);
const mountDashboard = async user => {
    useAuthStore().user = user;
    wrapper = shallowMount(DashboardView, { global: { plugins: [pinia], renderStubDefaultSlot: true, stubs: { Statistic: { name: 'Statistic', props: ['label', 'value'], template: '<div>{{ value }}<slot name="suffix" /></div>' } } } });
    await flushPromises();
};

it.each([
    [0, 'Guest'], [1, 'Member'], [2, 'VIP'], [3, 'External Member'], [6, 'Root'],
])('shows role %s as %s in SECURITY LEVEL', async (role, name) => {
    await mountDashboard({ username: 'u', role, permissions: { unlimited: role === 6, max_online_count: 1 } });
    expect(statistic('SECURITY LEVEL').props('value')).toBe(name);
});

it('shows running servers against max_online_count', async () => {
    await mountDashboard({ username: 'vip', role: 2, permissions: { unlimited: false, max_online_count: 2 } });
    expect(statistic('ACTIVE SERVERS').props('value')).toBe(2);
    expect(statistic('ACTIVE SERVERS').text().replace(/\s+/g, ' ')).toContain('2 / 2');
});

it('shows 不限 as the running limit for Root', async () => {
    await mountDashboard({ username: 'root', role: 6, permissions: { unlimited: true, max_online_count: 0 } });
    expect(statistic('ACTIVE SERVERS').text().replace(/\s+/g, ' ')).toContain('2 / 不限');
});
