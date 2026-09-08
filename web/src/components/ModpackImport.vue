<script setup>
import { ref, shallowRef, computed, watch } from 'vue';
import { NButton, NModal, NSpace, NInput, NText, NAlert, NSpin, useMessage } from 'naive-ui';
import api from '../api';
import ModpackFileBrowser from './ModpackFileBrowser.vue';
import { useModInstallStore } from '../store/modInstall';
import { readMRPack, buildMRPack, formatPackSize, overrideText } from '../utils/mrpack';

const emit = defineEmits(['submitted']);
const message = useMessage();
const installs = useModInstallStore();
const show = ref(false);
const confirming = ref(false);
const loading = ref(false);
const submitting = ref(false);
const error = ref('');
const config = ref(null);
const pack = shallowRef(null);
const prepared = shallowRef(null);
const sourceSize = ref(0);
const name = ref('');
const summary = ref('');
const files = ref([]);
const overrides = ref([]);
const edits = ref({});
const editing = ref(null);
const editText = ref('');
const downloadSize = computed(() => pack.value?.files.filter(file => files.value.includes(file.path)).reduce((sum, file) => sum + file.fileSize, 0) || 0);
const expandedOverrides = computed(() => pack.value?.overrides.filter(file => overrides.value.includes(file.path)).reduce((sum, file) => sum + overrideSize(file), 0) || 0);
const overrideSize = file => Object.hasOwn(edits.value, file.path) ? new TextEncoder().encode(edits.value[file.path]).length : file.size;
const browserEntries = computed(() => !pack.value ? [] : [
  ...pack.value.files.map(file => ({ ...file, id: `download:${file.path}`, size: file.fileSize, source: '下載檔案' })),
  ...pack.value.overrides.map(file => ({ ...file, id: file.path, path: file.path.slice(file.path.indexOf('/') + 1), size: overrideSize(file), source: file.path.startsWith('server-overrides/') ? '伺服器覆寫' : '共用覆寫' })),
]);
const selectedEntries = computed({
  get: () => [...files.value.map(path => `download:${path}`), ...overrides.value],
  set: ids => {
    files.value = ids.filter(id => id.startsWith('download:')).map(id => id.slice('download:'.length));
    overrides.value = ids.filter(id => !id.startsWith('download:'));
  },
});
function editEntry(entry) { edit(pack.value.overrides.find(file => file.path === entry.id)); }
watch([name, summary, files, overrides, edits], () => { prepared.value = null; confirming.value = false; }, { deep: true });

async function open() {
  show.value = true;
  loading.value = true;
  error.value = '';
  try { config.value = await api.get('/api/v1/server/modpack/config'); }
  catch (err) { config.value = null; error.value = err.response?.data?.error || err.message; }
  finally { loading.value = false; }
}

async function selectFile(event) {
  const file = event.target.files[0];
  if (!file) return;
  loading.value = true;
  pack.value = null;
  prepared.value = null;
  error.value = '';
  try {
    sourceSize.value = file.size;
    pack.value = await readMRPack(new Uint8Array(await file.arrayBuffer()), config.value);
    name.value = pack.value.index.name;
    summary.value = pack.value.index.summary || '';
    files.value = pack.value.files.filter(file => !file.reason).map(file => file.path);
    overrides.value = pack.value.overrides.filter(file => !file.reason).map(file => file.path);
    edits.value = {};
  } catch (err) { error.value = err.message; }
  finally { loading.value = false; event.target.value = ''; }
}

async function edit(file) {
  try {
    editText.value = edits.value[file.path] ?? await overrideText(pack.value, file);
    editing.value = file;
  } catch (err) { error.value = err.message; }
}

async function prepare() {
  loading.value = true;
  error.value = '';
  try {
    if (expandedOverrides.value > config.value.max_size) {
      error.value = '選取的覆寫內容超過後端解壓限制，請先取消較大的檔案。';
      return;
    }
    prepared.value = await buildMRPack(pack.value, { name: name.value.trim(), summary: summary.value, files: files.value, overrides: overrides.value, edits: edits.value });
    if (prepared.value.blob.size > config.value.max_size || prepared.value.expandedSize > config.value.max_size) {
      error.value = '修改後的壓縮檔或解壓內容超過後端限制，請取消較大的覆寫檔後重試。';
      return;
    }
    confirming.value = true;
  } catch (err) { error.value = err.message; }
  finally { loading.value = false; }
}

async function submit() {
  if (submitting.value) return;
  submitting.value = true;
  try {
    const result = await api.post('/api/v1/server/modpack/import', prepared.value.blob, { timeout: 0, headers: { 'Content-Type': 'application/x-modrinth-modpack+zip' } });
    installs.addJob({ jobId: result.ins_ses_id, serverId: result.server_id, serverName: name.value, modTitle: name.value, status: 'queued', stage: 'queued', percent: 0 });
    confirming.value = false;
    show.value = false;
    pack.value = null;
    prepared.value = null;
    emit('submitted');
    message.success('已建立整合包安裝工作');
  } catch (err) { error.value = err.response?.data?.error || err.message; confirming.value = false; }
  finally { submitting.value = false; }
}
</script>

<template>
  <n-button @click="open">匯入 .mrpack</n-button>
  <n-modal v-model:show="show" preset="card" title="匯入伺服器整合包" :style="{ width: 'min(900px, 94vw)' }" :mask-closable="!loading" :closable="!loading && !submitting">
    <n-spin :show="loading">
      <n-space vertical :size="16">
        <n-text v-if="config" depth="3">上傳與解壓內容各限 {{ formatPackSize(config.max_size) }}。預覽在本機處理，可先移除檔案再上傳。</n-text>
        <input aria-label="選擇 mrpack" type="file" accept=".mrpack" :disabled="!config || loading" @change="selectFile" />
        <n-alert v-if="error" type="error">{{ error }}</n-alert>
        <template v-if="pack">
          <n-space vertical>
            <label>伺服器名稱<n-input v-model:value="name" placeholder="伺服器名稱" /></label>
            <label>描述<n-input v-model:value="summary" type="textarea" :autosize="{ minRows: 1, maxRows: 3 }" /></label>
            <n-text>{{ pack.index.versionId }} · {{ Object.entries(pack.index.dependencies).map(([key, value]) => `${key} ${value}`).join(' · ') }}</n-text>
            <n-text depth="3">原始檔 {{ formatPackSize(sourceSize) }} · 選取下載 {{ formatPackSize(downloadSize) }} · 選取覆寫 {{ formatPackSize(expandedOverrides) }}</n-text>
          </n-space>
          <n-text strong>選擇安裝內容 · {{ files.length }} 個下載檔案 · {{ overrides.length }} 個覆寫檔案</n-text>
          <ModpackFileBrowser v-model:selected="selectedEntries" :entries="browserEntries" @edit="editEntry" />
          <n-text depth="3">共用覆寫先套用，伺服器覆寫後套用。文字設定可編輯（上限 1 MiB）。</n-text>
          <n-alert v-if="prepared" :type="error ? 'warning' : 'info'">修改後上傳 {{ formatPackSize(prepared.blob.size) }} · 解壓 {{ formatPackSize(prepared.expandedSize) }}</n-alert>
          <n-button type="primary" :disabled="!name.trim() || loading" @click="prepare">檢查大小並提交</n-button>
        </template>
      </n-space>
    </n-spin>
  </n-modal>
  <n-modal :show="!!editing" preset="card" title="編輯覆寫設定" :style="{ width: 'min(900px, 94vw)' }" @update:show="value => { if (!value) editing = null; }">
    <template v-if="editing">
      <n-text>{{ editing.path }}</n-text>
      <n-input v-model:value="editText" type="textarea" :autosize="{ minRows: 12, maxRows: 22 }" />
      <n-button type="primary" @click="edits[editing.path] = editText; editing = null">套用修改</n-button>
    </template>
  </n-modal>
  <n-modal v-model:show="confirming" preset="dialog" title="確認建立伺服器？" positive-text="確認匯入" negative-text="返回修改" :loading="submitting" :closable="!submitting" :mask-closable="!submitting" :close-on-esc="!submitting" :negative-button-props="{ disabled: submitting }" @positive-click="submit" @negative-click="confirming = false">
    將建立「{{ name }}」，安裝 {{ files.length }} 個下載檔案、套用 {{ overrides.length }} 個覆寫檔。
    <div v-if="prepared">上傳 {{ formatPackSize(prepared.blob.size) }}。確認後開始背景安裝。</div>
  </n-modal>
</template>

