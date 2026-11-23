import { createPinia, setActivePinia } from 'pinia';
import { beforeEach, expect, it, vi } from 'vitest';
import api from '../api';
import { defaultLoaderBuild, loaderBuildLabel, normalizeVersionList, useVersionCacheStore } from './versionCache';

vi.mock('../api', () => ({ default: { get: vi.fn() } }));

beforeEach(() => {
    vi.clearAllMocks();
    setActivePinia(createPinia());
});

it('reads the /vinfo array format and the legacy map format', () => {
    expect(normalizeVersionList({ versions: [{ id: '1.21.4', type: 'release' }, { id: '1.21.3', type: 'release' }] })).toEqual(['1.21.4', '1.21.3']);
    expect(normalizeVersionList({ versions: ['1.21', '1.20.6'] })).toEqual(['1.21', '1.20.6']);
    expect(normalizeVersionList({ versions: { '1.20.1': 'https://x' } })).toEqual(['1.20.1']);
    expect(normalizeVersionList(null)).toEqual([]);
});

it('prefetches vanilla, fabric, forge and neoforge versions', async () => {
    api.get.mockImplementation(async url => ({
        '/api/v1/public/vinfo': { versions: [{ id: '1.21.4', type: 'release' }] },
        '/api/v1/public/finfo': { versions: ['1.21.4'] },
        '/api/v1/public/versions/forge': { versions: [{ id: '1.20.1' }] },
        '/api/v1/public/versions/neoforge': { versions: [{ id: '1.21.1' }] },
    }[url]));
    const store = useVersionCacheStore();
    await store.prefetchVersions();
    expect(store.getVersionsByType('Vanilla')).toEqual([{ label: '1.21.4', value: '1.21.4' }]);
    expect(store.getVersionsByType('Fabric')).toEqual([{ label: '1.21.4', value: '1.21.4' }]);
    expect(store.getVersionsByType('Forge')).toEqual([{ label: '1.20.1', value: '1.20.1' }]);
    expect(store.getVersionsByType('NeoForge')).toEqual([{ label: '1.21.1', value: '1.21.1' }]);
});

it('caches loader builds per type and MC version', async () => {
    api.get.mockResolvedValue({ versions: [{ id: '0.16.9', stable: true, latest: true }] });
    const store = useVersionCacheStore();
    expect(await store.fetchLoaderBuilds('Fabric', '1.21.4')).toEqual([{ id: '0.16.9', stable: true, latest: true }]);
    await store.fetchLoaderBuilds('Fabric', '1.21.4');
    expect(api.get).toHaveBeenCalledTimes(1);
    expect(api.get).toHaveBeenCalledWith('/api/v1/public/versions/fabric', { params: { mc: '1.21.4' }, silent: true });
    expect(await store.fetchLoaderBuilds('Vanilla', '1.21.4')).toEqual([]);
});

it('defaults to the Forge recommended build and the latest stable otherwise', () => {
    const builds = [
        { id: '47.4.10', stable: true, latest: true },
        { id: '47.3.0', stable: true, recommended: true },
    ];
    expect(defaultLoaderBuild('Forge', builds)).toBe('47.3.0');
    expect(defaultLoaderBuild('NeoForge', builds)).toBe('47.4.10');
    expect(defaultLoaderBuild('Fabric', [{ id: '0.17.0-beta', stable: false }, { id: '0.16.9', stable: true, latest: true }])).toBe('0.16.9');
    expect(defaultLoaderBuild('Fabric', [])).toBe('');
    expect(loaderBuildLabel({ id: '47.3.0', stable: true, recommended: true })).toBe('47.3.0（推薦）');
    expect(loaderBuildLabel({ id: '0.17.0', stable: false })).toBe('0.17.0（不穩定）');
});
