<script setup>
import { computed, ref } from 'vue';
import { NCheckbox, NInput } from 'naive-ui';
import { FolderOpenOutlined, FileOutlined, RightOutlined, RollbackOutlined, UpOutlined, DownOutlined } from '@vicons/antd';
import { formatPackSize } from '../utils/mrpack';

const props = defineProps({
  entries: { type: Array, required: true },
  selected: { type: Array, required: true },
});
const emit = defineEmits(['update:selected', 'edit']);
const directory = ref('');
const search = ref('');
const sort = ref('name');
const descending = ref(false);
const selectedIds = computed(() => new Set(props.selected));

// Group only the current level; each folder retains its leaves for size and selection.
const rows = computed(() => {
  const folders = new Map();
  const files = [];
  for (const entry of props.entries) {
    if (!entry.path.startsWith(directory.value)) continue;
    const relative = entry.path.slice(directory.value.length);
    const slash = relative.indexOf('/');
    if (slash < 0) {
      files.push({ key: entry.id, name: relative, path: entry.path, size: entry.size, entries: [entry], file: entry });
    } else {
      const name = relative.slice(0, slash);
      if (!folders.has(name)) folders.set(name, { key: `folder:${name}`, name, path: `${directory.value}${name}/`, directory: true, size: 0, entries: [] });
      const folder = folders.get(name);
      folder.size += entry.size;
      folder.entries.push(entry);
    }
  }
  return [...folders.values(), ...files]
    .filter(row => row.name.toLowerCase().includes(search.value.toLowerCase()))
    .sort((a, b) => {
      const order = sort.value === 'size' ? a.size - b.size : a.name.localeCompare(b.name, 'zh-Hant', { numeric: true });
      return Number(!!b.directory) - Number(!!a.directory) || (descending.value ? -order : order);
    });
});
const visibleEntries = computed(() => rows.value.flatMap(row => row.entries));

function selection(entries) {
  const eligible = entries.filter(entry => !entry.reason);
  const count = eligible.filter(entry => selectedIds.value.has(entry.id)).length;
  return { checked: eligible.length > 0 && count === eligible.length, indeterminate: count > 0 && count < eligible.length, disabled: eligible.length === 0 };
}
function select(entries, checked) {
  const ids = new Set(props.selected);
  for (const entry of entries) {
    if (!entry.reason) { if (checked) ids.add(entry.id); else ids.delete(entry.id); }
  }
  emit('update:selected', [...ids]);
}
function navigate(path) { directory.value = path; search.value = ''; }
function parent() {
  const current = directory.value.slice(0, -1);
  navigate(current.slice(0, current.lastIndexOf('/') + 1));
}
function sortBy(key) {
  descending.value = sort.value === key ? !descending.value : false;
  sort.value = key;
}
</script>

<template>
  <div class="pack-browser">
    <div class="browser-toolbar">
      <div class="browser-location">
        <button type="button" @click="navigate('')">所有檔案</button>
        <span v-if="directory">/ {{ directory.slice(0, -1) }}</span>
      </div>
      <n-input v-model:value="search" size="small" placeholder="搜尋此層名稱" aria-label="搜尋此層名稱" clearable />
    </div>
    <div class="file-table-scroll">
      <table class="file-table" aria-label="整合包檔案選擇">
        <thead>
          <tr>
            <th class="check-cell">
              <n-checkbox v-bind="selection(visibleEntries)" aria-label="選取目前列表的所有可用檔案" @update:checked="checked => select(visibleEntries, checked)" />
            </th>
            <th><button class="sort-button" type="button" @click="sortBy('name')">名稱<component :is="descending ? DownOutlined : UpOutlined" v-if="sort === 'name'" class="sort-icon" /></button></th>
            <th class="size-cell"><button class="sort-button" type="button" @click="sortBy('size')">大小<component :is="descending ? DownOutlined : UpOutlined" v-if="sort === 'size'" class="sort-icon" /></button></th>
            <th class="action-cell"><span class="sr-only">操作</span></th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="directory" class="folder-row" @click="parent">
            <td></td>
            <td colspan="3"><button type="button" class="entry-name" @click.stop="parent"><RollbackOutlined class="file-icon" />父資料夾</button></td>
          </tr>
          <tr v-for="row in rows" :key="row.key" :class="{ 'folder-row': row.directory, unavailable: selection(row.entries).disabled }" @click="row.directory && navigate(row.path)">
            <td class="check-cell" @click.stop>
              <n-checkbox v-bind="selection(row.entries)" :aria-label="`選取 ${row.name}`" @update:checked="checked => select(row.entries, checked)" />
            </td>
            <td class="name-cell">
              <button v-if="row.directory" type="button" class="entry-name" :aria-label="`開啟資料夾 ${row.name}`" @click.stop="navigate(row.path)"><FolderOpenOutlined class="file-icon" /><span>{{ row.name }}</span></button>
              <div v-else class="file-name" :title="row.file.reason || row.path"><FileOutlined class="file-icon" /><span>{{ row.name }}<small>{{ row.file.reason || row.file.source }}</small></span></div>
            </td>
            <td class="size-cell" :title="`${row.size.toLocaleString()} bytes${row.directory ? '（包含子資料夾）' : ''}`">{{ formatPackSize(row.size) }}</td>
            <td class="action-cell">
              <RightOutlined v-if="row.directory" class="file-icon" />
              <button v-else-if="row.file.editable && !row.file.reason" type="button" class="edit-button" :aria-label="`編輯 ${row.name}`" @click.stop="emit('edit', row.file)">編輯</button>
            </td>
          </tr>
          <tr v-if="!rows.length"><td colspan="4" class="empty-row">{{ search ? '沒有符合的檔案' : '沒有檔案' }}</td></tr>
        </tbody>
      </table>
    </div>
    <div class="browser-caption">資料夾大小包含所有子檔案；勾選資料夾會選取其中可用的檔案。</div>
  </div>
</template>

<style scoped>
.browser-toolbar { display: flex; align-items: center; gap: 16px; margin-bottom: 10px; }
.browser-location { display: flex; align-items: center; gap: 6px; min-width: 0; flex: 1; color: #a2abb5; overflow-wrap: anywhere; }
.browser-location button, .entry-name, .sort-button, .edit-button { background: none; border: 0; color: inherit; font: inherit; cursor: pointer; padding: 0; }
.browser-location button { flex-shrink: 0; }
.browser-toolbar :deep(.n-input) { width: 200px; }
.file-table-scroll { border: 1px solid #24262a; border-radius: 14px; max-height: 480px; overflow: auto; background: #09090b; }
.file-table { width: 100%; border-spacing: 0; table-layout: fixed; color: #b5bec8; font-size: 14px; }
.file-table th { background: #111114; position: sticky; top: 0; z-index: 1; color: #99a4af; text-align: left; height: 38px; }
.file-table td { height: 38px; border-top: 1px solid #202226; }
.file-table th, .file-table td { padding: 0 8px; }
.file-table .check-cell { width: 22px; padding-right: 4px; }
.file-table .size-cell { width: 106px; text-align: right; white-space: nowrap; font-variant-numeric: tabular-nums; }
.file-table .action-cell { width: 40px; text-align: center; }
.sort-button, .entry-name, .file-name { display: flex; align-items: center; gap: 8px; }
.sort-button { width: 100%; }
.size-cell .sort-button { justify-content: flex-end; }
.entry-name { width: 100%; height: 38px; text-align: left; }
.entry-name span, .file-name > span { min-width: 0; overflow-wrap: anywhere; }
.file-name { padding: 5px 0; }
.file-name small { display: block; color: #77838f; font-size: 11px; }
.file-icon { width: 16px; height: 16px; flex-shrink: 0; }
.sort-icon { width: 11px; height: 11px; }
.folder-row { cursor: pointer; }
.file-table tbody tr:hover { background: #14171a; }
.unavailable { color: #747c86; }
.edit-button { font-size: 12px; color: #8de6b8; }
.browser-caption { margin-top: 8px; font-size: 12px; color: #88939e; }
.file-table .empty-row { height: 76px; text-align: center; color: #88939e; }
button:focus-visible { outline: 2px solid #36ad6a; outline-offset: 3px; }
.sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip-path: inset(50%); }
@media (max-width: 600px) {
  .browser-toolbar { align-items: stretch; flex-direction: column; gap: 8px; }
  .browser-toolbar :deep(.n-input) { width: 100%; }
  .file-table .size-cell { width: 82px; }
  .file-table .action-cell { width: 30px; }
}
</style>
