// @vitest-environment jsdom
import { shallowMount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { expect, it, vi } from 'vitest';
import MainLayout from './MainLayout.vue';
import { useAuthStore } from '../store/auth';

const push = vi.hoisted(() => vi.fn());
vi.mock('vue-router', () => ({ useRouter: () => ({ push }), useRoute: () => ({ name: 'Profile', params: {} }) }));
vi.mock('../api', () => ({ default: { get: vi.fn(), post: vi.fn() } }));

it('opens the profile page from the user menu', async () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useAuthStore();
    store.user = { username: 'tester', role: 1 };
    store.logout = vi.fn();
    const wrapper = shallowMount(MainLayout, { global: { plugins: [pinia], renderStubDefaultSlot: true, stubs: ['router-view'] } });
    wrapper.findComponent({ name: 'Dropdown' }).vm.$emit('select', 'profile');
    expect(push).toHaveBeenCalledWith({ name: 'Profile' });
    expect(store.logout).not.toHaveBeenCalled();
    expect(wrapper.vm.$.setupState.breadcrumbs).toEqual([{ label: 'Profile', to: '/profile' }]);
    wrapper.unmount();
});
