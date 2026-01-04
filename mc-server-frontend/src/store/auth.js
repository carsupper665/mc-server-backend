import { defineStore } from 'pinia';
import api from '../api';

/**
 * 認證 Store
 * 
 * @security 安全性注意事項：
 * - Token 透過 httpOnly cookie 由後端管理（建議後端設定 SameSite=Strict）
 * - 前端不直接存取 token，僅保存 username 用於 UI 顯示
 * - 敏感操作使用 withCredentials: true 確保 cookie 自動附帶
 * - 建議後端實作 CSRF token 驗證以增強安全性
 */

export const useAuthStore = defineStore('auth', {
    state: () => ({
        user: null,
        loading: false,
    }),
    getters: {
        isLoggedIn: (state) => !!state.user,
        isAdmin: (state) => state.user?.role === 100, // RoleRootUser
    },
    actions: {
        async login(username, password) {
            this.loading = true;
            try {
                const res = await api.post('/Authentication/login', { username, password });

                // If backend returns 202, verification is needed
                if (res.message && res.message.includes('verification code sent')) {
                    return res;
                }

                // Login successful (or already logged in)
                localStorage.setItem('username', username);
                await this.fetchUser();
                return res;
            } finally {
                this.loading = false;
            }
        },
        async verifyCode(code) {
            this.loading = true;
            try {
                const res = await api.post('/Authentication/verify', { code });
                await this.fetchUser();
                return res;
            } finally {
                this.loading = false;
            }
        },
        async logout() {
            try {
                await api.post('/logout');
            } finally {
                this.user = null;
                localStorage.removeItem('username');
                window.location.href = '/login';
            }
        },
        async fetchUser() {
            try {
                // Use /user/myservers just to validate the session
                await api.get('/user/myservers');

                // Since we can't get profile from backend, rely on localStorage or default
                const storedName = localStorage.getItem('username') || 'Operator';
                this.user = {
                    username: storedName,
                    role: 1 // Assume normal user, we can't know for sure without API
                };
            } catch (err) {
                this.user = null;
                localStorage.removeItem('username');
                throw err;
            }
        }
    }
});
