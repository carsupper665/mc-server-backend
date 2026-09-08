import { unzip, zip, strFromU8, strToU8 } from 'fflate';

const indexName = 'modrinth.index.json';
export const formatPackSize = bytes => bytes < 1024 ? `${bytes} B` : bytes < 1024 * 1024 ? `${(bytes / 1024).toFixed(2)} KiB` : `${(bytes / 1024 / 1024).toFixed(2)} MiB`;

export function blockedPackPath(name, blacklist) {
  return blacklist.some(value => {
    const pattern = value.trim();
    if (!pattern) return false;
    if (pattern.endsWith('/')) return name.startsWith(pattern);
    const regex = pattern.replace(/[.+^${}()|\\]/g, '\\$&').replace(/\*/g, '[^/]*').replace(/\?/g, '[^/]');
    return new RegExp(`^${regex}$`).test(name);
  });
}

const unzipEntries = (bytes, filter) => new Promise((resolve, reject) => {
  unzip(bytes, { filter }, (err, data) => err ? reject(err) : resolve(data));
});

export async function readMRPack(bytes, config) {
  const overrides = [];
  // Read the directory and manifest only; large overrides are inflated after selection.
  const entries = await unzipEntries(bytes, entry => {
    const name = entry.name;
    if (/^(overrides|server-overrides|client-overrides)\/.*[^/]$/.test(name)) {
      overrides.push({
        path: name, size: entry.originalSize,
        reason: name.startsWith('client-overrides/') ? '僅客戶端' : blockedPackPath(name.substring(name.indexOf('/') + 1), config.blacklist) ? '後端黑名單' : '',
        editable: /\.(json5?|toml|properties|txt|cfg|conf|ya?ml)$/i.test(name) && entry.originalSize <= 1024 * 1024,
      });
    }
    return name === indexName;
  });
  if (!entries[indexName]) throw new Error('缺少 modrinth.index.json');
  const index = JSON.parse(strFromU8(entries[indexName]));
  if (index.formatVersion !== 1 || index.game !== 'minecraft' || !Array.isArray(index.files) || !index.dependencies?.minecraft) {
    throw new Error('不支援的 mrpack 格式');
  }
  const files = index.files.map(file => ({
    ...file,
    reason: file.env?.server === 'unsupported' ? '僅客戶端' : blockedPackPath(file.path, config.blacklist) ? '後端黑名單' : '',
  }));
  return { index, files, overrides, bytes };
}

export async function buildMRPack(pack, { name, summary, files, overrides, edits }) {
  const index = { ...pack.index, name, summary, files: pack.files.filter(file => files.includes(file.path) && !file.reason).map(({ reason, ...file }) => file) };
  const selected = new Set(pack.overrides.filter(file => overrides.includes(file.path) && !file.reason).map(file => file.path));
  const entries = await unzipEntries(pack.bytes, entry => selected.has(entry.name) && !Object.hasOwn(edits, entry.name));
  entries[indexName] = strToU8(JSON.stringify(index, null, 2));
  for (const name of selected) {
    if (Object.hasOwn(edits, name)) entries[name] = strToU8(edits[name]);
  }
  const expandedSize = Object.values(entries).reduce((sum, data) => sum + data.length, 0);
  const data = await new Promise((resolve, reject) => zip(entries, { level: 6 }, (err, bytes) => err ? reject(err) : resolve(bytes)));
  return { blob: new Blob([data], { type: 'application/x-modrinth-modpack+zip' }), expandedSize };
}

export async function overrideText(pack, file) {
  const entries = await unzipEntries(pack.bytes, entry => entry.name === file.path);
  return strFromU8(entries[file.path]);
}
