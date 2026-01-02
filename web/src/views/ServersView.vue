<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { 
  NSpace, NButton, NCard, NDataTable, NTag, NIcon, NText,
  NModal, NForm, NFormItem, NSelect, NInput, useMessage 
} from 'naive-ui';
import { PlusOutlined, ReloadOutlined } from '@vicons/antd';
import api from '../api';
import { useRouter } from 'vue-router';
import { useVersionCacheStore } from '../store/versionCache';
import MinecraftLoader from '../components/MinecraftLoader.vue';

const message = useMessage();
const router = useRouter();
const versionCache = useVersionCacheStore();

const servers = ref([]);
const loading = ref(false);
const showCreateModal = ref(false);
const creating = ref(false);
const isDeploying = ref(false);
const deployMessage = ref('');

const createForm = ref({
  server_type: 'Vanilla',
  server_ver: '',
  display_name: ''
});

const serverTypes = [
  { label: 'Vanilla', value: 'Vanilla' },
  { label: 'Fabric', value: 'Fabric' }
];

// 從快取取得版本列表，根據選擇的伺服器類型動態切換
const versions = computed(() => {
  return versionCache.getVersionsByType(createForm.value.server_type);
});

// 當伺服器類型改變時，清除已選版本
watch(() => createForm.value.server_type, () => {
  createForm.value.server_ver = '';
});

const fetchServers = async () => {
  loading.value = true;
  try {
    const res = await api.get('/user/myservers');
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
        let type = 'Unknown';
        if (srv.server_id.startsWith('mcsfv')) type = 'Fabric';
        else if (srv.server_id.startsWith('mcsvv')) type = 'Vanilla';
        
        // 2. Fetch Status
        let status = 'Unknown';
        try {
            // Use POST as per backend controller definition
            const statusRes = await api.post(`/mc-api/a/status/${srv.server_id}`);
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

const fetchVersions = async () => {
  try {
    let res = await api.get('/mc-api/vinfo');
    
    // Defensive check: if response is wrapped in { versions: ... } or { data: ... }
    if (res.versions) res = res.versions;
    if (res.data) res = res.data;

    if (res && typeof res === 'object') {
        // Filter out keys that don't look like version numbers (simple heuristic)
        // detailed list keys are usually valid, but let's be safe against metadata keys
        const keys = Object.keys(res).filter(k => k.match(/^[\d\w\.-]+$/));
        
        // If keys include typical versions, assume it's good
        if (keys.length > 0) {
             versions.value = keys.map(v => ({ label: v, value: v }));
        } else {
             console.warn('No version keys found in response', res);
        }
    }
  } catch (err) {
    console.error('Failed to fetch versions', err);
  }
};

const handleCreate = async () => {
  if (!createForm.value.display_name) {
    message.warning('請輸入伺服器名稱');
    return;
  }
  if (!createForm.value.server_ver) {
    message.warning('請選擇伺服器版本');
    return;
  }

  // 防止重複點擊
  if (isDeploying.value) return;
  
  creating.value = true;
  isDeploying.value = true;
  deployMessage.value = '正在初始化部署環境...';
  
  try {
    // 模擬不同階段的提示訊息
    setTimeout(() => {
      if (isDeploying.value) deployMessage.value = '正在下載伺服器核心，請稍候...';
    }, 2000);
    setTimeout(() => {
      if (isDeploying.value) deployMessage.value = '正在配置伺服器檔案...';
    }, 5000);
    
    const res = await api.post('/mc-api/a/create', createForm.value);
    message.success('伺服器建立成功！');
    showCreateModal.value = false;
    
    // 重置表單
    createForm.value = { server_type: 'Vanilla', server_ver: '', display_name: '' };
    
    fetchServers();
  } catch (err) {
    message.error('建立失敗: ' + (err.response?.data?.error || '未知錯誤'));
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
      return h(NTag, { type: row.type === 'Fabric' ? 'info' : row.type === 'Vanilla' ? 'success' : 'default', size: 'small' }, { default: () => row.type });
    }
  },
  {
    title: '狀態',
    key: 'status',
    render(row) {
      const isRunning = row.status?.toLowerCase?.() === 'running';
      return h(NTag, {
        type: isRunning ? 'success' : 'error',
        size: 'small'
      }, { default: () => row.status });
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
          onClick: () => router.push(`/servers/${row.server_id}`)
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
            <n-text depth="3">MANAGE YOUR MINECRAFT ENVIRONMENTS</n-text>
          </div>
          <n-space>
            <n-button ghost @click="fetchServers" :loading="loading">
              <template #icon><n-icon><ReloadOutlined /></n-icon></template>
              REFRESH
            </n-button>
            <n-button type="primary" @click="showCreateModal = true">
              <template #icon><n-icon><PlusOutlined /></n-icon></template>
              DEPLOY NEW SERVER
            </n-button>
          </n-space>
        </n-space>
      </div>

      <n-card class="table-card" :bordered="false" content-style="padding: 0;">
        <div class="table-container">
          <n-data-table
            :columns="columns"
            :data="servers"
            :loading="loading"
            :bordered="false"
          />
        </div>
      </n-card>
    </n-space>

    <n-modal v-model:show="showCreateModal" preset="card" class="create-modal" title="DEPLOY NEW INSTANCE">
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
        </n-form>
      </template>
      
      <template #footer>
        <n-space justify="end">
          <n-button @click="showCreateModal = false" :disabled="isDeploying">CANCEL</n-button>
          <n-button 
            type="primary" 
            :loading="creating" 
            :disabled="isDeploying"
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

.create-modal {
  width: 90%;
  max-width: 500px;
}

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
