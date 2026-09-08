// @vitest-environment node
import { readFileSync } from 'node:fs';
import { describe, it, expect } from 'vitest';
import { zipSync, unzipSync, strToU8, strFromU8 } from 'fflate';
import { readMRPack, buildMRPack, blockedPackPath, overrideText } from './mrpack';

const config = { max_size: 150 * 1024 * 1024, blacklist: ['options.txt', 'config/*-client.*'] };
const index = { formatVersion: 1, game: 'minecraft', name: 'Pack', versionId: '1', dependencies: { minecraft: '1.21.6', 'fabric-loader': '0.18.1' }, files: [
  { path: 'mods/server.jar', fileSize: 3, downloads: [], env: { server: 'required' } },
  { path: 'mods/optional.jar', fileSize: 5, downloads: [], env: { server: 'optional' } },
  { path: 'mods/client.jar', fileSize: 7, downloads: [], env: { server: 'unsupported' } },
] };
const fixture = () => zipSync({
  'modrinth.index.json': strToU8(JSON.stringify(index)),
  'overrides/config/a.txt': strToU8('common'),
  'server-overrides/config/a.txt': strToU8('server'),
  'overrides/options.txt': strToU8('client'),
  'client-overrides/a.txt': strToU8('client-only'),
});

describe('mrpack local editing', () => {
  it('reads the real pack without uploading it', async () => {
    const pack = await readMRPack(new Uint8Array(readFileSync('../test/mrcpack/test_mod_pack.mrpack')), config);
    expect(pack.index.name).toBe('EZRedstone');
    expect(pack.files).toHaveLength(92);
    expect(pack.files.filter(file => file.env?.server === 'unsupported').every(file => file.reason)).toBe(true);
    expect(pack.overrides.some(file => file.size > 1024 * 1024)).toBe(true);
  });
  it('repackages only selected server files and edited overrides with exact sizes', async () => {
    const pack = await readMRPack(fixture(), config);
    expect(await overrideText(pack, pack.overrides.find(file => file.path === 'overrides/config/a.txt'))).toBe('common');
    const result = await buildMRPack(pack, { name: 'Edited', summary: 'Summary', files: ['mods/server.jar', 'mods/client.jar'], overrides: ['server-overrides/config/a.txt', 'overrides/options.txt', 'client-overrides/a.txt'], edits: { 'server-overrides/config/a.txt': 'edited 中文' } });
    const bytes = new Uint8Array(await result.blob.arrayBuffer());
    const output = unzipSync(bytes);
    const manifest = JSON.parse(strFromU8(output['modrinth.index.json']));
    expect(manifest.name).toBe('Edited');
    expect(manifest.summary).toBe('Summary');
    expect(manifest.files.map(file => file.path)).toEqual(['mods/server.jar']);
    expect(Object.keys(output).sort()).toEqual(['modrinth.index.json', 'server-overrides/config/a.txt']);
    expect(strFromU8(output['server-overrides/config/a.txt'])).toBe('edited 中文');
    expect(result.blob.size).toBe(bytes.length);
    expect(result.expandedSize).toBe(Object.values(output).reduce((sum, data) => sum + data.length, 0));
  });
  it('can drop overrides to pass a smaller backend size limit', async () => {
    const pack = await readMRPack(fixture(), config);
    const common = { name: 'Pack', summary: '', files: [], edits: {} };
    const full = await buildMRPack(pack, { ...common, overrides: ['overrides/config/a.txt', 'server-overrides/config/a.txt'] });
    const reduced = await buildMRPack(pack, { ...common, overrides: [] });
    expect(reduced.blob.size).toBeLessThan(full.blob.size);
    expect(reduced.expandedSize).toBeLessThan(full.expandedSize);
  });
  it('matches configurable client paths and rejects invalid packs', async () => {
    expect(blockedPackPath('config/test-client.toml', config.blacklist)).toBe(true);
    expect(blockedPackPath('config/deeper/test-client.toml', config.blacklist)).toBe(false);
    expect(blockedPackPath('resourcepacks/a.zip', ['resourcepacks/'])).toBe(true);
    await expect(readMRPack(zipSync({ 'other.txt': strToU8('x') }), config)).rejects.toThrow('缺少');
    const preview = await readMRPack(fixture(), { ...config, max_size: 10 });
    expect(preview.overrides.length).toBe(4); // oversized originals can still be trimmed locally
  });
});
