<script setup>
import { ref, computed, watch } from 'vue';
import {
  NModal,
  NButton,
  NText,
  NSpace,
  NAvatar,
  NTag,
  NSelect,
  NSpin,
  NDivider,
  NGrid,
  NGi
} from 'naive-ui';
import { getProjectVersions, formatDownloads } from '../api/modrinth';

const props = defineProps({
  show: {
    type: Boolean,
    default: false
  },
  mod: {
    type: Object,
    default: null
  },
  servers: {
    type: Array,
    default: () => []
  },
  loadingServers: {
    type: Boolean,
    default: false
  },
  installing: {
    type: Boolean,
    default: false
  }
});

const emit = defineEmits(['update:show', 'install']);

const show = computed({
  get: () => props.show,
  set: (val) => emit('update:show', val)
});

const modalStyle = computed(() => ({
  width: 'min(65vw, 620px)',
  aspectRatio: '4 / 3',
  maxHeight: '80vh'
}));

const selectedServerId = ref('');
const selectedVersionId = ref('');
const versionOptions = ref([]);
const loadingVersions = ref(false);
const compatibility = ref({ ok: false, reason: '' });

const loaderKeys = ['fabric', 'forge', 'neoforge', 'quilt'];

const modId = computed(() => props.mod?.project_id || props.mod?.id || props.mod?.slug || '');

const modLoaders = computed(() => {
  const categories = props.mod?.display_categories || props.mod?.categories || [];
  return categories.map(c => String(c).toLowerCase()).filter(c => loaderKeys.includes(c));
});

const latestVersion = computed(() => {
  const versions = props.mod?.versions || [];
  return versions.length > 0 ? versions[versions.length - 1] : null;
});

const categoryTags = computed(() => {
  const categories = props.mod?.display_categories || props.mod?.categories || [];
  return categories.filter(c => !loaderKeys.includes(String(c).toLowerCase())).slice(0, 3);
});

const downloads = computed(() => formatDownloads(props.mod?.downloads || 0));
const follows = computed(() => formatDownloads(props.mod?.follows || 0));

const selectedServer = computed(() => {
  return props.servers.find(s => s.server_id === selectedServerId.value);
});

const serverLoaderRaw = computed(() => {
  return selectedServer.value?.mod_loader || selectedServer.value?.modLoader || selectedServer.value?.loader || '';
});

const serverVersion = computed(() => {
  return selectedServer.value?.mc_version || selectedServer.value?.server_ver || selectedServer.value?.mcVersion || '';
});

const normalizeLoader = (value) => {
  const v = String(value || '').toLowerCase();
  if (v.includes('neo')) return 'neoforge';
  if (v.includes('forge')) return 'forge';
  if (v.includes('quilt')) return 'quilt';
  if (v.includes('fabric')) return 'fabric';
  if (v.includes('vanilla')) return 'vanilla';
  return v;
};

const serverLoader = computed(() => normalizeLoader(serverLoaderRaw.value));

const serverOptions = computed(() => {
  return props.servers.map(srv => {
    const name = srv.display_name || srv.server_id;
    const loader = srv.mod_loader || srv.modLoader || 'Unknown';
    const version = srv.mc_version || srv.server_ver || '';
    return {
      label: `${name} (${loader}${version ? ` · ${version}` : ''})`,
      value: srv.server_id
    };
  });
});

const precheckCompatibility = () => {
  if (!selectedServer.value) {
    return { ok: false, reason: '請先選擇伺服器' };
  }
  if (serverLoader.value === 'vanilla') {
    return { ok: false, reason: 'Vanilla 伺服器不支援 Mod' };
  }
  if (!serverLoader.value || serverLoader.value === 'unknown') {
    return { ok: false, reason: '伺服器平台未知' };
  }
  if (modLoaders.value.length === 0) {
    return { ok: false, reason: '模組未標記支援平台' };
  }
  if (!modLoaders.value.includes(serverLoader.value)) {
    return { ok: false, reason: `此模組不支援 ${serverLoaderRaw.value || serverLoader.value}` };
  }
  if (!serverVersion.value) {
    return { ok: false, reason: '伺服器版本未知' };
  }
  const modVersions = props.mod?.versions || [];
  if (modVersions.length > 0 && !modVersions.includes(serverVersion.value)) {
    return { ok: false, reason: `此模組未標記支援 ${serverVersion.value}` };
  }
  return { ok: true, reason: '' };
};

const refreshVersions = async () => {
  versionOptions.value = [];
  selectedVersionId.value = '';
  if (!props.mod || !selectedServer.value) {
    compatibility.value = { ok: false, reason: '請先選擇伺服器' };
    return;
  }

  const precheck = precheckCompatibility();
  if (!precheck.ok) {
    compatibility.value = precheck;
    return;
  }

  loadingVersions.value = true;
  try {
    const versions = await getProjectVersions(modId.value, {
      loaders: serverLoader.value ? [serverLoader.value] : [],
      game_versions: serverVersion.value ? [serverVersion.value] : []
    });

    const opts = (versions || []).map(v => ({
      label: v.version_number || v.name || v.id,
      value: v.id
    }));

    versionOptions.value = opts;
    if (opts.length === 0) {
      compatibility.value = { ok: false, reason: '找不到相容版本' };
      return;
    }

    compatibility.value = { ok: true, reason: '相容' };
  } catch (err) {
    compatibility.value = { ok: false, reason: '版本載入失敗' };
  } finally {
    loadingVersions.value = false;
  }
};

const handleInstall = () => {
  if (!props.mod || !selectedServer.value) return;
  if (!compatibility.value.ok) return;
  emit('install', {
    mod: props.mod,
    server: selectedServer.value,
    versionId: selectedVersionId.value
  });
};

watch(() => props.show, (val) => {
  if (val) {
    selectedServerId.value = '';
    selectedVersionId.value = '';
    versionOptions.value = [];
    compatibility.value = { ok: false, reason: '請先選擇伺服器' };
  }
});

watch([() => props.mod, selectedServerId], () => {
  if (props.show) {
    refreshVersions();
  }
});
</script>

<template>
  <n-modal
    v-model:show="show"
    preset="card"
    class="install-modal"
    title="INSTALL MOD"
    :closable="!installing"
    :mask-closable="!installing"
    :style="modalStyle"
  >
    <div v-if="!mod" class="empty-state">
      <n-text depth="3">尚未選擇模組</n-text>
    </div>

    <template v-else>
      <div class="modal-header">
        <div class="mod-summary">
          <n-avatar
            :src="mod.icon_url"
            :size="56"
            :fallback-src="'https://cdn.modrinth.com/placeholder.svg'"
            class="mod-icon"
            object-fit="cover"
          />
          <div class="mod-header-info">
            <n-text class="mod-title">{{ mod.title }}</n-text>
            <n-text depth="3" class="mod-author">by {{ mod.author }}</n-text>
          </div>
        </div>
        <n-button
          type="primary"
          :loading="installing"
          :disabled="!compatibility.ok"
          @click="handleInstall"
        >
          INSTALL
        </n-button>
      </div>

      <n-divider />

      <n-grid cols="1 m:2" :x-gap="16" :y-gap="12" responsive="screen">
        <n-gi>
          <div class="info-block">
            <n-text class="block-title">MOD INFO</n-text>
            <n-text depth="3" class="mod-description">{{ mod.description }}</n-text>
            <div class="stats-row">
              <n-tag size="small" type="success" round>{{ downloads }} downloads</n-tag>
              <n-tag size="small" type="default" round>{{ follows }} follows</n-tag>
              <n-tag v-if="latestVersion" size="small" type="warning" round>{{ latestVersion }}</n-tag>
            </div>
            <div class="tags-row">
              <n-tag v-for="loader in modLoaders" :key="loader" size="small" type="info" round>
                {{ loader }}
              </n-tag>
              <n-tag v-for="cat in categoryTags" :key="cat" size="small" type="default" round>
                {{ cat }}
              </n-tag>
            </div>
          </div>
        </n-gi>
        <n-gi>
          <div class="info-block">
            <n-text class="block-title">TARGET SERVER</n-text>
            <n-spin :show="loadingServers">
              <n-select
                v-model:value="selectedServerId"
                :options="serverOptions"
                placeholder="選擇要安裝的伺服器"
                clearable
              />
            </n-spin>

            <div class="server-meta" v-if="selectedServer">
              <n-space :size="8" align="center" wrap>
                <n-tag size="small" type="info" round>{{ serverLoaderRaw || 'Unknown' }}</n-tag>
                <n-tag size="small" type="default" round>{{ serverVersion || 'Unknown version' }}</n-tag>
              </n-space>
            </div>

            <div class="version-select">
              <n-text depth="3" class="label">Mod 版本</n-text>
              <n-spin :show="loadingVersions">
                <n-select
                  v-model:value="selectedVersionId"
                  :options="[{ label: 'Auto (由後端決定)', value: '' }, ...versionOptions]"
                  placeholder="選擇特定版本（可選）"
                  :disabled="!compatibility.ok"
                  clearable
                />
              </n-spin>
            </div>

            <div class="compatibility" :class="compatibility.ok ? 'ok' : 'error'">
              <n-text :type="compatibility.ok ? 'success' : 'error'">
                {{ compatibility.ok ? '相容' : compatibility.reason }}
              </n-text>
            </div>
          </div>
        </n-gi>
      </n-grid>
    </template>
  </n-modal>
</template>

<style scoped>
.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.mod-summary {
  display: flex;
  align-items: center;
  gap: 12px;
}

.mod-icon {
  border-radius: 12px;
  background-color: #2a2a30;
}

.mod-header-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.mod-title {
  font-size: 16px;
  font-weight: 600;
  font-family: 'Fira Code', monospace;
}

.mod-author {
  font-size: 12px;
}

.info-block {
  background: #1a1a20;
  border: 1px solid #333;
  border-radius: 10px;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 10px;
  height: 100%;
}

.block-title {
  font-size: 12px;
  color: #888;
  letter-spacing: 1px;
}

.mod-description {
  font-size: 13px;
  line-height: 1.5;
}

.stats-row,
.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.server-meta {
  margin-top: 6px;
}

.version-select {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.label {
  font-size: 12px;
}

.compatibility {
  padding: 8px 10px;
  border-radius: 6px;
  border: 1px solid rgba(24, 160, 88, 0.3);
  background: rgba(24, 160, 88, 0.08);
}

.compatibility.error {
  border-color: rgba(220, 38, 38, 0.4);
  background: rgba(220, 38, 38, 0.08);
}

.empty-state {
  padding: 24px 0;
  text-align: center;
}

.install-modal :deep(.n-card) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.install-modal :deep(.n-card__content) {
  flex: 1;
  overflow-y: auto;
}

@media (max-width: 768px) {
  .install-modal :deep(.n-card) {
    max-height: 90vh;
  }
}
</style>
