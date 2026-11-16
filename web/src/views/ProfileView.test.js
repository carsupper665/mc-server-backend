// @vitest-environment jsdom
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, expect, it } from 'vitest';
import ProfileView from './ProfileView.vue';
import { useAuthStore } from '../store/auth';

const gb = 1024 * 1024 * 1024;
let wrapper;
let pinia;
beforeEach(() => {
    pinia = createPinia();
    setActivePinia(pinia);
});
afterEach(() => wrapper?.unmount());

const rows = () => Object.fromEntries(wrapper.findAll('tr').map(tr => {
    const cells = tr.findAll('th, td');
    return [cells[0].text(), cells[1].text()];
}));

it('shows account info, role name and limits with memory in GB', () => {
    useAuthStore().user = {
        username: 'vipuser', display_name: 'Vip User', role: 2,
        permissions: {
            unlimited: false, server_max_mem: 8 * gb, max_server_count: 10, max_online_count: 2,
            max_mods_count: 256, max_back_up_count: 5, can_back_up_server: true, can_edit_server_args: false,
        },
    };
    wrapper = mount(ProfileView, { global: { plugins: [pinia] } });
    expect(rows()).toMatchObject({
        '顯示名稱': 'Vip User',
        '帳號': 'vipuser',
        '角色': 'VIP',
        '單台記憶體上限': '8 GB',
        '伺服器數量上限': '10',
        '同時運行上限': '2',
        '模組數量上限': '256',
        '備份數量上限': '5',
        '建立備份': '可以',
        '修改 JVM 參數': '不可以',
    });
});

it('shows 不限 for every Root limit', () => {
    useAuthStore().user = {
        username: 'root', display_name: '', role: 6,
        permissions: {
            unlimited: true, server_max_mem: 0, max_server_count: 0, max_online_count: 0,
            max_mods_count: 0, max_back_up_count: 0, can_back_up_server: true, can_edit_server_args: true,
        },
    };
    wrapper = mount(ProfileView, { global: { plugins: [pinia] } });
    expect(rows()).toMatchObject({
        '顯示名稱': 'root',
        '角色': 'Root',
        '單台記憶體上限': '不限',
        '伺服器數量上限': '不限',
        '同時運行上限': '不限',
        '模組數量上限': '不限',
        '備份數量上限': '不限',
        '建立備份': '可以',
        '修改 JVM 參數': '可以',
    });
});

it('shows dashes when permissions were not loaded', () => {
    useAuthStore().user = { username: 'member', role: 1 };
    wrapper = mount(ProfileView, { global: { plugins: [pinia] } });
    expect(rows()).toMatchObject({ '角色': 'Member', '單台記憶體上限': '—', '建立備份': '—' });
});
