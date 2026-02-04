<script setup>
import { ref, onMounted } from 'vue';
import { NSpace, NText, useMessage } from 'naive-ui';
import api from '../api';
import ModBrowser from '../components/ModBrowser.vue';
import ModInstallModal from '../components/ModInstallModal.vue';
import { useModInstallStore } from '../store/modInstall';

const message = useMessage();
const modInstallStore = useModInstallStore();

const servers = ref([]);
const loadingServers = ref(false);

const showInstallModal = ref(false);
const selectedMod = ref(null);
const installing = ref(false);

const placeholderIcon = 'https://cdn.modrinth.com/placeholder.svg';

const fetchServers = async () => {
  loadingServers.value = true;
  try {
    const res = await api.get('/user/myservers');
    if (Array.isArray(res)) {
      servers.value = res;
    } else {
      servers.value = res.data || [];
    }
  } catch (err) {
    servers.value = [];
    message.error('無法取得伺服器列表');
  } finally {
    loadingServers.value = false;
  }
};

const handleAddMod = (mod) => {
  selectedMod.value = mod;
  showInstallModal.value = true;
  if (servers.value.length === 0 && !loadingServers.value) {
    fetchServers();
  }
};

const handleInstall = async ({ mod, server, versionId }) => {
  if (!mod || !server) return;
  installing.value = true;
  try {
    const payload = {
      mod_id: mod.project_id,
      version_id: versionId || '',
      auto_update: true
    };
    const res = await api.post(`/api/v1/server/mod/add/${server.server_id}?async=true`, payload);
    const jobId = res.job_id;
    if (!jobId) {
      message.error('無法取得安裝工作 ID');
      return;
    }

    const job = {
      jobId,
      serverId: server.server_id,
      serverName: server.display_name || server.server_id,
      modId: mod.project_id,
      modTitle: mod.title || mod.project_id,
      modIcon: mod.icon_url || placeholderIcon,
      stage: res.status || 'queued',
      status: 'running',
      percent: 0,
      message: ''
    };

    modInstallStore.addJob(job);
    message.success('已加入安裝佇列');
    showInstallModal.value = false;
  } catch (err) {
    message.error('安裝失敗: ' + (err.response?.data?.error || '未知錯誤'));
  } finally {
    installing.value = false;
  }
};

onMounted(() => {
  fetchServers();
});
</script>

<template>
  <div class="mods-view fade-in-up">
    <n-space vertical :size="24">
      <div class="page-header">
        <h2 class="title">MOD BROWSER</h2>
        <n-text depth="3">EXPLORE AND DISCOVER MINECRAFT MODS FROM MODRINTH</n-text>
      </div>
      
      <ModBrowser @add-mod="handleAddMod" />
    </n-space>

    <ModInstallModal
      v-model:show="showInstallModal"
      :mod="selectedMod"
      :servers="servers"
      :loading-servers="loadingServers"
      :installing="installing"
      @install="handleInstall"
    />

  </div>
</template>

<style scoped>
.mods-view {
  max-width: 1400px;
  margin: 0 auto;
}

.title {
  margin: 0;
  font-size: 24px;
  font-family: 'Fira Code', monospace;
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
