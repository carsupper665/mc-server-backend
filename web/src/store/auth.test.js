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
});
