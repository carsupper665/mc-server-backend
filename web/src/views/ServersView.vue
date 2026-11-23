<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { 
  NSpace, NButton, NCard, NDataTable, NTag, NIcon, NText,
  NModal, NForm, NFormItem, NSelect, NInput, NSpin, NEmpty, useMessage
} from 'naive-ui';
import { PlusOutlined, ReloadOutlined } from '@vicons/antd';
import api, { getSanitizedErrorMessage } from '../api';
import { useRouter } from 'vue-router';
import { useVersionCacheStore, defaultLoaderBuild, loaderBuildLabel, LOADER_TYPES } from '../store/versionCache';
import { useMobileLayout } from '../composables/useMobileLayout';
import MinecraftLoader from '../components/MinecraftLoader.vue';
import ModpackImport from '../components/ModpackImport.vue';
import { useModInstallStore } from '../store/modInstall';
import { useAuthStore } from '../store/auth';

const isMobile = useMobileLayout();
const message = useMessage();
const router = useRouter();
const versionCache = useVersionCacheStore();
const installs = useModInstallStore();
const auth = useAuthStore();
// 安裝（建立伺服器、整合包、mod）結束時重新整理列表
watch(() => installs.jobs.filter(job => ['completed', 'failed'].includes(job.status)).length, () => fetchServers());

const servers = ref([]);
const loading = ref(false);
const showCreateModal = ref(false);
const creating = ref(false);
const isDeploying = ref(false);
const deployMessage = ref('');

const emptyForm = () => ({
  server_type: 'Vanilla',
  server_ver: '',
  loader_version: '',
  display_name: ''
});
const createForm = ref(emptyForm());

const serverTypes = [
  { label: 'Vanilla', value: 'Vanilla' },
  { label: 'Fabric', value: 'Fabric' },
  { label: 'Forge', value: 'Forge' },
  { label: 'NeoForge', value: 'NeoForge' }
];

// Server ID 前綴 → 類型（舊資料沒有 mod_loader 時使用）
const ID_PREFIX_TYPES = { mcsvv: 'Vanilla', mcsfv: 'Fabric', mcsgv: 'Forge', mcsnv: 'NeoForge' };
const serverTypeOf = srv => ID_PREFIX_TYPES[String(srv.server_id || '').slice(0, 5)] || srv.mod_loader || 'Unknown';
const typeTagColor = type => ({ Fabric: 'info', Vanilla: 'success', Forge: 'warning', NeoForge: 'warning' }[type] || 'default');

// install_status（安裝工作）也有 'running'，會跟伺服器執行中的 'running' 撞名，
// 所以安裝狀態一律轉成 'installing'／'install_failed' 再放進 status。
const INSTALLING = ['queued', 'running'];
const installState = installStatus => (INSTALLING.includes(installStatus) ? 'installing' : 'install_failed');
const statusLabel = status => {
  if (status === 'installing') return '安裝中';
  if (status === 'install_failed') return '安裝失敗';
  return status;
};
const statusTagType = status => {
  if (status?.toLowerCase?.() === 'running') return 'success';
  if (status === 'installing') return 'warning';
  return 'error';
};

// 從快取取得版本列表，根據選擇的伺服器類型動態切換
const versions = computed(() => {
  return versionCache.getVersionsByType(createForm.value.server_type);
});

const needsLoader = computed(() => LOADER_TYPES.includes(createForm.value.server_type));
const loaderBuilds = ref([]);
const loaderLoading = ref(false);
const loaderOptions = computed(() => loaderBuilds.value.map(build => ({ label: loaderBuildLabel(build), value: build.id })));

// 當伺服器類型改變時，清除已選版本
watch(() => createForm.value.server_type, () => {
  createForm.value.server_ver = '';
  createForm.value.loader_version = '';
  loaderBuilds.value = [];
});

// 選好 MC 版本後載入 loader 版本，預設 Forge 推薦版、其他最新穩定版
watch(() => [createForm.value.server_type, createForm.value.server_ver], async ([type, ver]) => {
  createForm.value.loader_version = '';
  loaderBuilds.value = [];
  if (!LOADER_TYPES.includes(type) || !ver) return;
  loaderLoading.value = true;
  try {
    const builds = await versionCache.fetchLoaderBuilds(type, ver);
    if (createForm.value.server_type !== type || createForm.value.server_ver !== ver) return;
    loaderBuilds.value = builds;
    createForm.value.loader_version = defaultLoaderBuild(type, builds);
  } catch (err) {
    message.error('無法載入 loader 版本: ' + getSanitizedErrorMessage(err));
  } finally {
    loaderLoading.value = false;
  }
});

const fetchServers = async () => {
  loading.value = true;
  try {
    const res = await api.get('/user/myservers', { silent: true });
    let list = [];
    if (Array.isArray(res)) {
        list = res;
    } else {
        list = res.data || [];
    }
    
    // Enrich data: Parse type and fetch status for each server
    // This is the N+1 solution because list API lacks details
    const enrichedList = await Promise.all(list.map(async (srv) => {
        // 1. Parse Type from ID
        const type = serverTypeOf(srv);

        // 2. Fetch Status
        let status = 'Unknown';
        if (srv.install_status && srv.install_status !== 'completed') {
          // 重新整理頁面後也能繼續看到安裝進度
          if (INSTALLING.includes(srv.install_status)) {
            installs.trackServerInstall({ jobId: srv.ins_ses_id, serverId: srv.server_id, serverName: srv.display_name, status: 'running' });
          }
          return { ...srv, type, status: installState(srv.install_status) };
        }
        try {
            const statusRes = await api.get(`/api/v1/server/status/${srv.server_id}`, { silent: true });
            // Check for status field in response
            if (statusRes && statusRes.status) {
                status = statusRes.status;
            } else if (typeof statusRes === 'string') {
                status = statusRes;
            }
        } catch (e) {
            status = 'Error';
        }

        return { ...srv, type, status };
    }));

    servers.value = enrichedList;

  } catch (err) {
    message.error('無法獲取伺服器列表');
  } finally {
    loading.value = false;
  }
};

const handleCreate = async () => {
  if (loading.value || !auth.canCreateServer(servers.value.length)) return;
  if (!createForm.value.display_name) {
    message.warning('請輸入伺服器名稱');
    return;
  }
  if (!createForm.value.server_ver) {
    message.warning('請選擇伺服器版本');
    return;
  }
  if (needsLoader.value && !createForm.value.loader_version) {
    message.warning('請選擇 loader 版本');
    return;
  }

  // 防止重複點擊
  if (isDeploying.value) return;
  
  creating.value = true;
  isDeploying.value = true;
  deployMessage.value = '正在建立伺服器...';

  try {
    const form = createForm.value;
    const payload = { server_type: form.server_type, server_ver: form.server_ver, display_name: form.display_name };
    if (form.server_type === 'Fabric') payload.fabric_loader = form.loader_version;
    else if (needsLoader.value) payload.loader_version = form.loader_version;
    // 後端立即回 202，安裝在背景進行，進度顯示在安裝進度面板
    const res = await api.post(`/api/v1/server/create`, payload, { silent: true });
    installs.trackServerInstall({ jobId: res?.ins_ses_id, serverId: res?.server_id, serverName: form.display_name });
    message.success('已開始安裝伺服器，完成後即可啟動');
    showCreateModal.value = false;

    // 重置表單
    createForm.value = emptyForm();

    fetchServers();
  } catch (err) {
    message.error('建立失敗: ' + getSanitizedErrorMessage(err));
  } finally {
    creating.value = false;
    isDeploying.value = false;
    deployMessage.value = '';
  }
};

const columns = [
  {
    title: 'ID',
    key: 'server_id',
    width: 200,
    render(row) {
      return h('span', { style: 'font-family: Fira Code' }, row.server_id);
    }
  },
  {
    title: '名稱',
    key: 'display_name',
  },
  {
    title: '類型',
    key: 'type',
    render(row) {
      return h(NTag, { type: typeTagColor(row.type), size: 'small' }, { default: () => row.type });
    }
  },
  {
    title: '狀態',
    key: 'status',
    render(row) {
      return h(NTag, {
        type: statusTagType(row.status),
        size: 'small'
      }, { default: () => statusLabel(row.status) });
    }
  },
  {
    title: '操作',
    key: 'actions',
    render(row) {
      return h(
        NButton,
        {
          size: 'small',
          ghost: true,
          type: 'primary',
          onClick: () => router.push({
            name: 'ServerDetail',
            params: { id: row.server_id },
            query: { display_name: row.display_name}
          })
        },
        { default: () => '管理' }
      );
    }
  }
];

import { h } from 'vue';

onMounted(() => {
  fetchServers();
  // 如果快取尚未載入，主動預載 (防止使用者直接進入 Servers 頁而非從 Dashboard)
  if (!versionCache.isLoaded) {
    versionCache.prefetchVersions();
  }
});
</script>

<template>
  <div class="servers-view fade-in-up">
    <n-space vertical :size="24">
      <div class="page-header">
        <n-space justify="space-between" align="center">
          <div>
            <h2 class="title">SERVER INSTANCES</h2>
            <div>伺服器數 {{ servers.length }} / {{ auth.serverLimit }}</div>
            <n-text depth="3">MANAGE YOUR MINECRAFT ENVIRONMENTS</n-text>
          </div>
          <n-space>
            <n-button ghost @click="fetchServers" :loading="loading">
              <template #icon><n-icon><ReloadOutlined /></n-icon></template>
              REFRESH
            </n-button>
            <ModpackImport @submitted="fetchServers" />
            <n-button type="primary" :disabled="loading || !auth.canCreateServer(servers.length)" @click="showCreateModal = true">
              <template #icon><n-icon><PlusOutlined /></n-icon></template>
              DEPLOY NEW SERVER
            </n-button>
          </n-space>
        </n-space>
      </div>

      <n-card v-if="!isMobile" class="table-card" :bordered="false" content-style="padding: 0;">
        <div class="table-container">
          <n-data-table
            :columns="columns"
            :data="servers"
            :loading="loading"
            :bordered="false"
          />
        </div>
      </n-card>
      <n-spin v-else :show="loading">
        <div class="server-list" :aria-busy="loading">
          <n-empty v-if="!servers.length" :description="loading ? '載入中...' : '尚無伺服器'" />
          <n-card v-for="server in servers" :key="server.server_id" :bordered="false">
            <h3 class="server-name">{{ server.display_name }}</h3>
            <div class="server-id">{{ server.server_id }}</div>
            <n-space class="server-tags">
              <n-tag :type="typeTagColor(server.type)" size="small">{{ server.type }}</n-tag>
              <n-tag :type="statusTagType(server.status)" size="small">{{ statusLabel(server.status) }}</n-tag>
            </n-space>
            <n-button block ghost type="primary" @click="router.push({ name: 'ServerDetail', params: { id: server.server_id }, query: { display_name: server.display_name } })">管理</n-button>
          </n-card>
        </div>
      </n-spin>
    </n-space>

    <n-modal v-model:show="showCreateModal" preset="card" :style="{ width: 'min(500px, 94vw)' }" title="DEPLOY NEW INSTANCE">
      <!-- 部署中 Loading 動畫 -->
      <template v-if="isDeploying">
        <MinecraftLoader :message="deployMessage" />
      </template>
      
      <!-- 表單 -->
      <template v-else>
        <n-form :model="createForm" label-placement="top">
          <n-form-item label="DISPLAY NAME">
            <n-input v-model:value="createForm.display_name" placeholder="My Awesome Server" />
          </n-form-item>
          <n-form-item label="SERVER TYPE">
            <n-select v-model:value="createForm.server_type" :options="serverTypes" />
          </n-form-item>
          <n-form-item label="VERSION">
            <n-select 
              v-model:value="createForm.server_ver" 
              :options="versions" 
              :loading="versionCache.isFetching"
              filterable 
              placeholder="選擇版本..."
            />
          </n-form-item>
          <n-form-item v-if="needsLoader" :label="`${createForm.server_type.toUpperCase()} VERSION`">
            <n-select
              v-model:value="createForm.loader_version"
              :options="loaderOptions"
              :loading="loaderLoading"
              :disabled="!createForm.server_ver"
              filterable
              placeholder="先選擇 MC 版本"
            />
          </n-form-item>
        </n-form>
      </template>
      
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false" :disabled="isDeploying">CANCEL</n-button>
          <n-button 
            type="primary" 
            :loading="creating" 
            :disabled="isDeploying || loading || !auth.canCreateServer(servers.length)"
            @click="handleCreate"
          >
            INITIALIZE DEPLOYMENT
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.title {
  margin: 0;
  font-size: 24px;
  font-family: 'Fira Code', monospace;
}

.table-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
  overflow: hidden;
}

.table-container {
  overflow-x: auto;
}

.server-list { display: grid; gap: 12px; }
.server-name { margin: 0 0 8px; overflow-wrap: anywhere; }
.server-id { font: 12px 'Fira Code', monospace; overflow-wrap: anywhere; }
.server-tags { margin: 12px 0; }

:deep(.n-data-table) {
  background-color: transparent;
}

:deep(.n-data-table .n-data-table-td) {
  background-color: transparent;
}

.fade-in-up {
  opacity: 0;
  animation: fadeInUp 0.6s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

@keyframes fadeInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>
