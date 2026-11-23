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

it('shows the Users menu only to Root', () => {
    const pinia = createPinia();
    setActivePinia(pinia);
    const store = useAuthStore();
    store.user = { username: 'member', role: 1 };
    const wrapper = shallowMount(MainLayout, { global: { plugins: [pinia], renderStubDefaultSlot: true, stubs: ['router-view'] } });
    const keys = () => wrapper.vm.$.setupState.menuOptions.map(o => o.key);
    expect(keys()).not.toContain('AdminUsers');
    store.user = { username: 'root', role: 6 };
    expect(keys()).toContain('AdminUsers');
    wrapper.vm.$.setupState.menuOptions.find(o => o.key === 'AdminUsers').onClick();
    expect(push).toHaveBeenCalledWith({ name: 'AdminUsers' });
    wrapper.unmount();
});
