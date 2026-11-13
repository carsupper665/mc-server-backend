// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import api from '../api';
import { useAuthStore } from './auth';
import { getAccessToken, setAccessToken } from '../utils/authStorage';
vi.mock('../api', () => ({ default: { get: vi.fn() } }));

const token = `e30.${btoa(JSON.stringify({ username: 'tester', exp: 4102444800 }))}.sig`;
describe('fetchUser', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
        localStorage.clear();
        vi.clearAllMocks();
        setAccessToken(token);
    });
    it.each([undefined, 429, 502])('preserves the JWT session after transient failure %s', async status => {
        api.get.mockRejectedValue({ response: status ? { status } : undefined });
        const store = useAuthStore();
        expect((await store.fetchUser()).username).toBe('tester');
        expect(store.isLoggedIn).toBe(true);
        expect(getAccessToken()).toBe(token);
    });
    it('clears unauthorized sessions', async () => {
        api.get.mockRejectedValue({ response: { status: 401 } });
        const store = useAuthStore();
        await expect(store.fetchUser()).rejects.toBeDefined();
        expect(getAccessToken()).toBeFalsy();
    });
    it('clears expired tokens', async () => {
        setAccessToken(`e30.${btoa(JSON.stringify({ exp: 1 }))}.sig`);
        await expect(useAuthStore().fetchUser()).rejects.toThrow();
        expect(getAccessToken()).toBeFalsy();
    });
    it('loads permissions from the profile even when the token has a display name', async () => {
        setAccessToken(`e30.${btoa(JSON.stringify({ username: 'tester', display_name: 'Tester', exp: 4102444800 }))}.sig`);
        api.get.mockResolvedValue({ username: 'tester', display_name: 'Tester', role: 2,
            permissions: { max_server_count: 10, can_create_server: true, can_back_up_server: true, unlimited: false } });
        const store = useAuthStore();
        await store.fetchUser();
        expect(api.get).toHaveBeenCalledWith('/user/me', expect.any(Object));
        expect(store.user.permissions.can_back_up_server).toBe(true);
        expect(store.serverLimit).toBe(10);
        expect(store.canCreateServer(9)).toBe(true);
        expect(store.canCreateServer(10)).toBe(false);
        expect(store.canCreateServer(11)).toBe(false);
    });
    it('denies creation when permissions are missing or creation is forbidden', () => {
        const store = useAuthStore();
        expect(store.canCreateServer(0)).toBe(false);
        store.user = { permissions: { max_server_count: 5, can_create_server: false, unlimited: false } };
        expect(store.canCreateServer(0)).toBe(false);
    });

});
