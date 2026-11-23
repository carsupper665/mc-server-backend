import { defineStore } from 'pinia';
import api from '../api';

/**
 * 版本快取 Store
 *
 * 在使用者進入 Dashboard 時背景預載版本資訊，
 * 確保開啟「新增伺服器」視窗時版本選單秒開。
 *
 * 後端：
 * - /api/v1/public/vinfo → { versions: [{ id, type, stable }] }（只有 release）
 * - /api/v1/public/finfo → { versions: ['1.21', ...] }
 * - /api/v1/public/versions/:loader → { versions: [{ id, ... }] }
 * - /api/v1/public/versions/:loader?mc=x → { versions: [{ id, stable, latest, recommended }] }
 */

const toOptions = list => list.map(v => ({ label: v, value: v }));

// 接受 { versions: [...] }、{ data: [...] } 或直接陣列；元素可以是字串或 { id }
export const normalizeVersionList = payload => {
  let data = payload;
  if (data && !Array.isArray(data) && data.versions) data = data.versions;
  if (data && !Array.isArray(data) && data.data) data = data.data;
  if (Array.isArray(data)) {
    return data.map(v => (typeof v === 'string' ? v : v?.id)).filter(Boolean);
  }
  if (data && typeof data === 'object') {
    // 舊格式 { "<ver>": "<url>" }
    return Object.keys(data).filter(k => k.match(/^[\d\w.-]+$/));
  }
  return [];
};

export const LOADER_TYPES = ['Fabric', 'Forge', 'NeoForge'];

export const useVersionCacheStore = defineStore('versionCache', {
  state: () => ({
    vanillaVersions: [],
    fabricVersions: [],
    forgeVersions: [],
    neoforgeVersions: [],
    // `${type}:${mc}` → [{ id, stable, latest, recommended }]
    loaderBuilds: {},
    lastFetched: null,
    isFetching: false,
    fetchError: null,
    // 快取有效期 5 分鐘
    cacheDuration: 5 * 60 * 1000,
  }),

  getters: {
    /**
     * 檢查快取是否仍然有效
     */
    isCacheValid: state => {
      if (!state.lastFetched) return false;
      return Date.now() - state.lastFetched < state.cacheDuration;
    },

    /**
     * 取得所有版本（合併）
     */
    allVersions: state => {
      return {
        vanilla: state.vanillaVersions,
        fabric: state.fabricVersions,
        forge: state.forgeVersions,
        neoforge: state.neoforgeVersions,
      };
    },

    /**
     * 判斷是否已載入
     */
    isLoaded: state => {
      return state.vanillaVersions.length > 0 || state.fabricVersions.length > 0;
    },
  },

  actions: {
    /**
     * 背景預載所有版本資訊
     * 這個函式會在 Dashboard 掛載時被呼叫
     */
    async prefetchVersions() {
      // 如果快取仍然有效，跳過
      if (this.isCacheValid) {
        return;
      }

      // 避免重複請求
      if (this.isFetching) {
        return;
      }

      this.isFetching = true;
      this.fetchError = null;

      try {
        const sources = [
          ['vanillaVersions', '/api/v1/public/vinfo'],
          ['fabricVersions', '/api/v1/public/finfo'],
          ['forgeVersions', '/api/v1/public/versions/forge'],
          ['neoforgeVersions', '/api/v1/public/versions/neoforge'],
        ];
        const results = await Promise.allSettled(sources.map(([, url]) => api.get(url, { silent: true })));
        results.forEach((res, i) => {
          const [key, url] = sources[i];
          if (res.status === 'fulfilled') {
            this[key] = toOptions(normalizeVersionList(res.value));
          } else {
            console.warn(`[VersionCache] ${url} 載入失敗:`, res.reason);
          }
        });

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
     * @param {'Vanilla' | 'Fabric' | 'Forge' | 'NeoForge'} type
     */
    getVersionsByType(type) {
      switch (type) {
        case 'Fabric': return this.fabricVersions;
        case 'Forge': return this.forgeVersions;
        case 'NeoForge': return this.neoforgeVersions;
        default: return this.vanillaVersions;
      }
    },

    /**
     * 取得某個 MC 版本可用的 loader 版本（Fabric／Forge／NeoForge），結果會快取
     */
    async fetchLoaderBuilds(type, mc) {
      if (!LOADER_TYPES.includes(type) || !mc) return [];
      const key = `${type}:${mc}`;
      if (this.loaderBuilds[key]) return this.loaderBuilds[key];
      const res = await api.get(`/api/v1/public/versions/${type.toLowerCase()}`, { params: { mc }, silent: true });
      const builds = Array.isArray(res?.versions) ? res.versions : [];
      this.loaderBuilds = { ...this.loaderBuilds, [key]: builds };
      return builds;
    },

    /**
     * 強制重新載入（清除快取）
     */
    async forceRefresh() {
      this.lastFetched = null;
      this.loaderBuilds = {};
      await this.prefetchVersions();
    },
  },
});

/**
 * 預設 loader 版本：Forge 用推薦版，其他用最新穩定版
 */
export const defaultLoaderBuild = (type, builds) => {
  if (!builds?.length) return '';
  const pick = (type === 'Forge' && builds.find(b => b.recommended))
    || builds.find(b => b.latest)
    || builds.find(b => b.stable)
    || builds[0];
  return pick.id;
};

export const loaderBuildLabel = build => {
  const tags = [];
  if (build.recommended) tags.push('推薦');
  if (build.latest) tags.push('最新');
  if (!build.stable) tags.push('不穩定');
  return tags.length ? `${build.id}（${tags.join('、')}）` : build.id;
};
