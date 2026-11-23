// @vitest-environment jsdom
import { shallowMount, flushPromises } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import AdminUsersView from './AdminUsersView.vue';
import { useAuthStore } from '../store/auth';
import api from '../api';

const messages = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), warning: vi.fn() }));
vi.mock('naive-ui', async () => ({ ...await vi.importActual('naive-ui'), useMessage: () => messages }));
vi.mock('vue-router', () => ({ RouterLink: { name: 'RouterLink', props: ['to'], template: '<a><slot /></a>' } }));
vi.mock('../api', async () => ({ default: { get: vi.fn(), post: vi.fn() }, getSanitizedErrorMessage: (await vi.importActual('../utils/errorMapping')).extractAndSanitizeError }));

const rootUser = { id: 1, username: 'root', display_name: 'Root', email: 'root@example.test', role: 6, is_fgf: false, created_at: '2026-11-01T00:00:00Z' };
const localUser = { id: 2, username: 'alice', display_name: 'Alice', email: 'alice@example.test', role: 1, is_fgf: false, created_at: '2026-11-02T00:00:00Z' };
const fgfUser = { id: 3, username: 'fgf', display_name: 'FGF Bob', email: 'bob@example.test', role: 3, is_fgf: true, created_at: '2026-11-03T00:00:00Z' };

let wrapper;
let pinia;
beforeEach(() => {
    vi.clearAllMocks();
    pinia = createPinia();
    setActivePinia(pinia);
    api.get.mockResolvedValue({ data: [rootUser, localUser, fgfUser] });
    api.post.mockResolvedValue({ message: 'ok' });
});
afterEach(() => wrapper?.unmount());

const mountAs = async user => {
    const auth = useAuthStore();
    auth.user = user;
    auth.fetchUser = vi.fn().mockResolvedValue(user);
    wrapper = shallowMount(AdminUsersView, { global: { plugins: [pinia], renderStubDefaultSlot: true } });
    await flushPromises();
    return wrapper.vm.$.setupState;
};
const asRoot = () => mountAs({ userId: '1', username: 'root', role: 6 });

it('shows no permission and does not call the admin API for non-Root', async () => {
    await mountAs({ userId: '2', username: 'alice', role: 1 });
    expect(wrapper.text()).toContain('沒有權限查看此頁面');
    expect(api.get).not.toHaveBeenCalled();
    expect(wrapper.findComponent({ name: 'DataTable' }).exists()).toBe(false);
});

it('lists users with role names and FGF flag for Root', async () => {
    const state = await asRoot();
    expect(api.get).toHaveBeenCalledWith('/op/v1/users', { silent: true });
    const table = wrapper.findComponent({ name: 'DataTable' });
    expect(table.props('data')).toHaveLength(3);
    const titles = state.columns.map(c => c.title);
    expect(titles).toEqual(['ID', '帳號', '顯示名稱', 'Email', '角色', 'FGF', '建立時間', '操作']);
    const roleColumn = state.columns.find(c => c.key === 'role');
    expect([rootUser, localUser, fgfUser].map(u => roleColumn.render(u))).toEqual(['Root', 'Member', 'External Member']);
});

it('creates users with assignable roles only and validates like the backend', async () => {
    const state = await asRoot();
    expect(state.assignableRoles.map(r => r.label)).toEqual(['Guest', 'Member', 'VIP', 'Root']);
    state.openCreate();
    state.createForm = { username: 'guest1', display_name: 'Guest', email: 'not-an-email', password: '12345678', role: 0 };
    expect(state.createError).toBe('請輸入有效的 email');
    state.createForm.email = 'g@example.test';
    state.createForm.password = '1234567';
    expect(state.createError).toBe('密碼至少 8 碼');
    await state.submitCreate();
    expect(api.post).not.toHaveBeenCalled();
    state.createForm.password = '12345678';
    expect(state.createError).toBe('');
    await state.submitCreate();
    expect(api.post).toHaveBeenCalledWith('/op/v1/register',
        { username: 'guest1', display_name: 'Guest', email: 'g@example.test', password: '12345678', role: 0 }, { silent: true });
    expect(state.showCreate).toBe(false);
    expect(api.get).toHaveBeenCalledTimes(2);
});

it('shows the backend error when creation fails', async () => {
    const state = await asRoot();
    state.createForm = { username: 'x', display_name: 'X', email: 'x@example.test', password: '12345678', role: 1 };
    api.post.mockRejectedValue({ response: { status: 400, data: { message: 'Invalid role' } } });
    await state.submitCreate();
    expect(messages.error).toHaveBeenCalledWith('建立失敗: Invalid role');
});

it('edits only changed fields and refreshes the list', async () => {
    const state = await asRoot();
    state.openEdit(localUser);
    state.editForm.email = 'alice2@example.test';
    state.editForm.role = 2;
    await state.submitEdit();
    expect(api.post).toHaveBeenCalledWith('/op/v1/edit', { user_id: 2, new_email: 'alice2@example.test', new_role: 2 }, { silent: true });
    expect(api.get).toHaveBeenCalledTimes(2);
});

it('locks role and display name for FGF accounts', async () => {
    const state = await asRoot();
    state.openEdit(fgfUser);
    await flushPromises();
    expect(wrapper.text()).toContain('角色與顯示名稱由 FGF 同步');
    state.editForm.display_name = 'Changed';
    state.editForm.role = 1;
    state.editForm.username = 'fgf2';
    expect(state.editChanges).toEqual({ new_username: 'fgf2' });
});

it('asks again before Root changes its own role', async () => {
    const state = await asRoot();
    state.openEdit(rootUser);
    state.editForm.role = 1;
    await state.submitEdit();
    expect(api.post).not.toHaveBeenCalled();
    expect(state.confirmSelfRole).toBe(true);
    await state.sendEdit();
    expect(api.post).toHaveBeenCalledWith('/op/v1/edit', { user_id: 1, new_role: 1 }, { silent: true });
    expect(useAuthStore().fetchUser).toHaveBeenCalled();
});

it('resets passwords for local accounts only and requires matching 8+ chars', async () => {
    const state = await asRoot();
    const actions = state.columns.find(c => c.key === 'actions');
    const buttonsFor = user => actions.render(user).children.default().map(b => b.children.default());
    expect(buttonsFor(localUser)).toEqual(['編輯', '重設密碼']);
    expect(buttonsFor(fgfUser)).toEqual(['編輯']);

    state.openReset(localUser);
    state.resetForm = { password: '12345678', confirm: '12345679' };
    expect(state.resetError).toBe('兩次輸入的密碼不一致');
    await state.submitReset();
    expect(api.post).not.toHaveBeenCalled();
    state.resetForm.confirm = '12345678';
    await state.submitReset();
    expect(api.post).toHaveBeenCalledWith('/op/v1/reset', { user_id: 2, new_password: '12345678' }, { silent: true });
    expect(state.showReset).toBe(false);
});
