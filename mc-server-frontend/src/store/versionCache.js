import { defineStore } from 'pinia';
import api from '../api';

/**
 * 版本快取 Store
 * 
 * 在使用者進入 Dashboard 時背景預載版本資訊，
 * 確保開啟「新增伺服器」視窗時版本選單秒開。
 */
export const useVersionCacheStore = defineStore('versionCache', {
    state: () => ({
        vanillaVersions: [],
        fabricVersions: [],
        lastFetched: null,
        isFetching: false,
        fetchError: null,
        // 快取有效期 5 分鐘
        cacheDuration: 5 * 60 * 1000
    }),

    getters: {
        /**
         * 檢查快取是否仍然有效
         */
        isCacheValid: (state) => {
            if (!state.lastFetched) return false;
            return Date.now() - state.lastFetched < state.cacheDuration;
        },

        /**
         * 取得所有版本（合併）
         */
        allVersions: (state) => {
            return {
                vanilla: state.vanillaVersions,
                fabric: state.fabricVersions
            };
        },

        /**
         * 判斷是否已載入
         */
        isLoaded: (state) => {
            return state.vanillaVersions.length > 0 || state.fabricVersions.length > 0;
        }
    },

    actions: {
        /**
         * 背景預載所有版本資訊
         * 這個函式會在 Dashboard 掛載時被呼叫
         */
        async prefetchVersions() {
            // 如果快取仍然有效，跳過
            if (this.isCacheValid) {
                console.log('[VersionCache] 快取仍有效，跳過預載');
                return;
            }

            // 避免重複請求
            if (this.isFetching) {
                console.log('[VersionCache] 正在載入中，跳過');
                return;
            }

            this.isFetching = true;
            this.fetchError = null;

            try {
                // 並行請求 Vanilla 和 Fabric 版本
                const [vanillaRes, fabricRes] = await Promise.allSettled([
                    api.get('/mc-api/vinfo'),
                    api.get('/mc-api/finfo')
                ]);

                // 處理 Vanilla 版本
                if (vanillaRes.status === 'fulfilled') {
                    let data = vanillaRes.value;
                    if (data.versions) data = data.versions;
                    if (data.data) data = data.data;

                    if (data && typeof data === 'object') {
                        const keys = Object.keys(data).filter(k => k.match(/^[\d\w\.-]+$/));
                        this.vanillaVersions = keys.map(v => ({ label: v, value: v }));
                        console.log(`[VersionCache] Vanilla 版本載入完成: ${keys.length} 個`);
                    }
                } else {
                    console.warn('[VersionCache] Vanilla 版本載入失敗:', vanillaRes.reason);
                }

                // 處理 Fabric 版本
                if (fabricRes.status === 'fulfilled') {
                    let data = fabricRes.value;
                    if (data.versions) data = data.versions;
                    if (data.data) data = data.data;

                    if (data && typeof data === 'object') {
                        const keys = Object.keys(data).filter(k => k.match(/^[\d\w\.-]+$/));
                        this.fabricVersions = keys.map(v => ({ label: v, value: v }));
                        console.log(`[VersionCache] Fabric 版本載入完成: ${keys.length} 個`);
                    }
                } else {
                    console.warn('[VersionCache] Fabric 版本載入失敗:', fabricRes.reason);
                }

                this.lastFetched = Date.now();

            } catch (err) {
                console.error('[VersionCache] 預載失敗:', err);
                this.fetchError = err.message;
            } finally {
                this.isFetching = false;
            }
        },

        /**
         * 根據伺服器類型取得版本列表
         * @param {'Vanilla' | 'Fabric'} type 
         */
        getVersionsByType(type) {
            if (type === 'Fabric') {
                return this.fabricVersions;
            }
            return this.vanillaVersions;
        },

        /**
         * 強制重新載入（清除快取）
         */
        async forceRefresh() {
            this.lastFetched = null;
            await this.prefetchVersions();
        }
    }
});
