import { defineStore } from 'pinia';
import api from '../api';
import {
    buildUserFromToken,
    clearAuthSession,
    getAccessToken,
    getTokenPayload,
    isTokenExpired,
    setAccessToken,
    setStoredUsername,
} from '../utils/authStorage';

const emailPattern = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const isEmail = (value) => emailPattern.test(String(value || '').trim());

const parseCallbackPayload = (query = {}) => {
    let code = String(query.code || '').trim();
    let id = String(query.id || '').trim();

    // Backward-compatible parser for malformed redirect:
    // /login/callback?code=<authCode>?=id<id>
    if (!id && code) {
        const markers = ['?=id', '?id=', '&id='];
        for (const marker of markers) {
            const idx = code.indexOf(marker);
            if (idx !== -1) {
                id = code.slice(idx + marker.length).trim();
                code = code.slice(0, idx).trim();
                break;
            }
        }
    }

    return { code, id };
};

/**
 * 認證 Store
 * 
 * @security 安全性注意事項：
 * - Access token 由前端保存並透過 Authorization: Bearer 傳送
 * - 所有受保護 API 一律附帶 X-Device-ID 與 C-MPMC-WEB-Header
 * - 登入後以 JWT payload 重建 user context，避免再依賴 cookie session
 */

export const useAuthStore = defineStore('auth', {
    state: () => ({
        user: null,
        loading: false,
    }),
    getters: {
        isLoggedIn: (state) => !!state.user,
        isAdmin: (state) => state.user?.role === 6, // RoleRootUser
    },
    actions: {
        clearAuthState() {
            this.user = null;
            clearAuthSession();
        },
        async login(username, password) {
            this.loading = true;
            try {
                const payload = isEmail(username)
                    ? { email: username, password }
                    : { username, password };
                const res = await api.withMeta.post('/Authentication/login', payload, {
                    skipAuthToken: true
                });

                if (res.status === 202) {
                    setStoredUsername(username);
                    return {
                        status: res.status,
                        data: res.data,
                        requiresVerification: true
                    };
                }

                if (res.status < 200 || res.status >= 300) {
                    throw new Error(`Unexpected login status: ${res.status}`);
                }

                const token = res.data?.token;
                if (!token) {
                    throw new Error('伺服器沒有回傳 access token');
                }

                setStoredUsername(username);
                setAccessToken(token);
                await this.fetchUser({ skipRemoteCheck: true });
                return {
                    status: res.status,
                    data: res.data,
                    requiresVerification: false
                };
            } finally {
                this.loading = false;
            }
        },
        async verifyCode(code) {
            this.loading = true;
            try {
                const res = await api.withMeta.post('/Authentication/verify', { code }, {
                    skipAuthToken: true
                });
                if (res.status < 200 || res.status >= 300) {
                    throw new Error(`Unexpected verify status: ${res.status}`);
                }
                await this.fetchUser();
                return {
                    status: res.status,
                    data: res.data
                };
            } finally {
                this.loading = false;
            }
        },
        async exchangeCallbackToken(query) {
            this.loading = true;
            try {
                const { code, id } = parseCallbackPayload(query);
                if (!code || !id) {
                    throw new Error('登入連結無效或已損壞');
                }

                const res = await api.withMeta.get('/Authentication/challenge', {
                    params: { code, id },
                    skipAuthRedirect: true,
                    silent: true,
                    skipAuthToken: true
                });
                if (res.status < 200 || res.status >= 300) {
                    throw new Error(`Unexpected callback status: ${res.status}`);
                }

                const token = res.data?.token;
                if (!token) {
                    throw new Error('伺服器沒有回傳 access token');
                }

                setAccessToken(token);
                await this.fetchUser({ skipRemoteCheck: true });
                return {
                    status: res.status,
                    data: res.data
                };
            } finally {
                this.loading = false;
            }
        },
        async logout() {
            try {
                await api.post('/logout', null, {
                    skipAuthRedirect: true,
                    silent: true
                });
            } finally {
                this.clearAuthState();
                window.location.href = '/login';
            }
        },
        async fetchUser(options = {}) {
            const token = getAccessToken();
            const payload = getTokenPayload(token);

            if (!token || !payload || isTokenExpired(payload)) {
                this.clearAuthState();
                throw new Error('Access token missing or expired');
            }

            try {
                if (!options.skipRemoteCheck) {
                    await api.get('/user/myservers', {
                        silent: true,
                        skipAuthRedirect: options.skipAuthRedirect === true
                    });
                }

                const user = buildUserFromToken(token);
                if (!user) {
                    throw new Error('Invalid access token payload');
                }

                this.user = user;
                return user;
            } catch (err) {
                this.clearAuthState();
                throw err;
            }
        }
    }
});
