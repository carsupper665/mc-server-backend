import axios from 'axios';

/**
 * Modrinth API 客戶端
 * 文檔: https://docs.modrinth.com/api/
 * 
 * 此 API 支援 CORS，可直接從前端呼叫
 */

const modrinthApi = axios.create({
    baseURL: 'https://api.modrinth.com/v2',
    timeout: 15000,
    // 注意: 瀏覽器不允許設定 User-Agent header，已移除
});

/**
 * 搜尋模組
 * @param {Object} options - 搜尋選項
 * @param {string} options.query - 搜尋關鍵字
 * @param {string[]} options.loaders - 載入器過濾 (fabric, forge, neoforge, quilt)
 * @param {string[]} options.versions - 遊戲版本過濾
 * @param {string} options.category - 類別過濾
 * @param {number} options.limit - 每頁數量 (預設 20, 最大 100)
 * @param {number} options.offset - 偏移量 (分頁用)
 * @param {string} options.index - 排序方式 (relevance, downloads, follows, newest, updated)
 * @returns {Promise<Object>} - { hits, offset, limit, total_hits }
 */
export async function searchMods({
    query = '',
    loaders = [],
    versions = [],
    category = '',
    limit = 20,
    offset = 0,
    index = 'relevance'
} = {}) {
    // 構建 facets 陣列
    const facets = [];

    // 必須是 mod 類型
    facets.push(['project_type:mod']);

    // 載入器過濾 (OR 邏輯)
    if (loaders.length > 0) {
        facets.push(loaders.map(l => `categories:${l}`));
    }

    // 版本過濾 (OR 邏輯)
    if (versions.length > 0) {
        facets.push(versions.map(v => `versions:${v}`));
    }

    // 類別過濾
    if (category) {
        facets.push([`categories:${category}`]);
    }

    const params = {
        limit,
        offset,
        index
    };

    if (query) {
        params.query = query;
    }

    if (facets.length > 0) {
        params.facets = JSON.stringify(facets);
    }

    const response = await modrinthApi.get('/search', { params });
    return response.data;
}

/**
 * 取得專案詳情
 * @param {string} idOrSlug - 專案 ID 或 slug
 * @returns {Promise<Object>} - 專案詳情
 */
export async function getProject(idOrSlug) {
    const response = await modrinthApi.get(`/project/${idOrSlug}`);
    return response.data;
}

/**
 * 取得專案版本列表
 * @param {string} idOrSlug - 專案 ID 或 slug
 * @param {Object} options - 過濾選項
 * @param {string[]} options.loaders - 載入器過濾
 * @param {string[]} options.game_versions - 遊戲版本過濾
 * @returns {Promise<Array>} - 版本列表
 */
export async function getProjectVersions(idOrSlug, { loaders = [], game_versions = [] } = {}) {
    const params = {};

    if (loaders.length > 0) {
        params.loaders = JSON.stringify(loaders);
    }

    if (game_versions.length > 0) {
        params.game_versions = JSON.stringify(game_versions);
    }

    const response = await modrinthApi.get(`/project/${idOrSlug}/version`, { params });
    return response.data;
}

/**
 * 取得多個專案資訊
 * @param {string[]} ids - 專案 ID 陣列
 * @returns {Promise<Array>} - 專案列表
 */
export async function getProjects(ids) {
    const response = await modrinthApi.get('/projects', {
        params: { ids: JSON.stringify(ids) }
    });
    return response.data;
}

/**
 * 取得熱門類別
 * @returns {Promise<Array>} - 類別列表
 */
export async function getCategories() {
    const response = await modrinthApi.get('/tag/category');
    return response.data;
}

/**
 * 取得可用的載入器
 * @returns {Promise<Array>} - 載入器列表
 */
export async function getLoaders() {
    const response = await modrinthApi.get('/tag/loader');
    return response.data;
}

/**
 * 取得遊戲版本列表
 * @returns {Promise<Array>} - 遊戲版本列表
 */
export async function getGameVersions() {
    const response = await modrinthApi.get('/tag/game_version');
    return response.data;
}

/**
 * 格式化下載數量
 * @param {number} num - 下載數量
 * @returns {string} - 格式化後的字串
 */
export function formatDownloads(num) {
    if (num >= 1000000) {
        return (num / 1000000).toFixed(1) + 'M';
    }
    if (num >= 1000) {
        return (num / 1000).toFixed(1) + 'K';
    }
    return num.toString();
}

export default {
    searchMods,
    getProject,
    getProjectVersions,
    getProjects,
    getCategories,
    getLoaders,
    getGameVersions,
    formatDownloads
};
