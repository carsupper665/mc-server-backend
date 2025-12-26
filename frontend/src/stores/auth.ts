import { defineStore } from 'pinia';
import { authApi } from '../api/minecraft';

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null as any,
    token: localStorage.getItem('token') || null,
  }),
  getters: {
    isAuthenticated: (state) => !!state.token,
  },
  actions: {
    setToken(token: string) {
      this.token = token;
      localStorage.setItem('token', token);
    },
    async logout() {
      try {
        await authApi.logout();
      } finally {
        this.token = null;
        this.user = null;
        localStorage.removeItem('token');
      }
    },
  },
});
