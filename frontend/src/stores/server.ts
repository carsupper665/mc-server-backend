import { defineStore } from 'pinia';
import { minecraftApi } from '../api/minecraft';

export const useServerStore = defineStore('server', {
  state: () => ({
    servers: [] as any[],
    currentServerStatus: {} as Record<string, any>,
    loading: false,
  }),
  actions: {
    async fetchMyServers() {
      this.loading = true;
      try {
        const res = await minecraftApi.getMyServers();
        this.servers = res.data || [];
      } catch (error) {
        console.error('Failed to fetch servers:', error);
        this.servers = []; // Ensure empty array on error
      } finally {
        this.loading = false;
      }
    },
    async updateServerStatus(id: string) {
      try {
        const res = await minecraftApi.getStatus(id);
        this.currentServerStatus[id] = res.data;
      } catch (err) {
        console.error('Failed to get status', err);
      }
    },
  },
});
