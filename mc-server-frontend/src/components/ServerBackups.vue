<script setup>
import { ref, onMounted } from 'vue';
import { NCard, NButton, NIcon, NText, NSpace, NModal, NInput, useMessage } from 'naive-ui';
import { SaveOutlined } from '@vicons/antd';
import api, { getSanitizedErrorMessage } from '../api';
import { useActivityLogStore } from '../store/activityLog';

const props = defineProps({
  serverId: {
    type: String,
    required: true,
  },
  serverName: {
    type: String,
    default: '',
  },
});

const emit = defineEmits(['recovery-started']);

const message = useMessage();
const activityLog = useActivityLogStore();

// Backups state
const backups = ref([]);
const loading = ref(false);

// Recover modal state
const showRecoverModal = ref(false);
const recoverConfirmText = ref('');
const recoveringBackup = ref(null);
const isRecovering = ref(false);

const fetchBackups = async () => {
  loading.value = true;
  try {
    const res = await api.post(`/mc-api/a/ls-backup/${props.serverId}`);
    // api interceptor returns response.data directly
    backups.value = res && res.backups ? res.backups.map(b => ({ name: b })) : [];
  } catch (err) {
    backups.value = [];
  } finally {
    loading.value = false;
  }
};

const handleBackup = async () => {
  try {
    await api.post(`/mc-api/a/backup/${props.serverId}`);
    message.success('備份任務已啟動');
    activityLog.logBackup(props.serverId, props.serverName);
    fetchBackups();
  } catch (err) {
    message.error('備份失敗');
  }
};

const openRecoverModal = backup => {
  recoveringBackup.value = backup;
  recoverConfirmText.value = '';
  showRecoverModal.value = true;
};

const confirmRecover = async () => {
  if (recoverConfirmText.value !== 'CONFIRM') {
    message.warning('請輸入 CONFIRM 以確認還原操作');
    return;
  }

  isRecovering.value = true;
  try {
    await api.post(`/mc-api/a/recover`, {
      server_id: props.serverId,
      file_name: recoveringBackup.value.name,
    });
    message.success('還原成功！伺服器將重新啟動。');
    activityLog.logRecover(props.serverId, props.serverName, recoveringBackup.value.name);
    showRecoverModal.value = false;
    emit('recovery-started');
  } catch (err) {
    const errorMsg = getSanitizedErrorMessage(err);
    message.error('還原失敗: ' + errorMsg);
    activityLog.logError('還原備份', props.serverId, errorMsg);
  } finally {
    isRecovering.value = false;
  }
};

onMounted(() => {
  fetchBackups();
});

defineExpose({ fetchBackups });
</script>

<template>
  <div class="server-backups-container">
    <n-card title="BACKUPS" class="backup-card">
      <template #header-extra>
        <n-button size="tiny" tertiary @click="handleBackup">
          <template #icon
            ><n-icon><SaveOutlined /></n-icon
          ></template>
          NEW
        </n-button>
      </template>

      <div class="backup-list">
        <div v-if="loading" class="empty-backups">載入中...</div>
        <div v-else-if="backups.length === 0" class="empty-backups">No backups found</div>
        <div v-for="b in backups" :key="b.name" class="backup-item">
          <n-text class="backup-name">{{ b.name }}</n-text>
          <n-button size="tiny" ghost type="warning" @click="openRecoverModal(b)"> 還原 </n-button>
        </div>
      </div>
    </n-card>

    <!-- 還原確認 Modal -->
    <n-modal
      v-model:show="showRecoverModal"
      preset="card"
      title="☠️ 還原備份 - 危險操作"
      class="recover-modal"
      :closable="!isRecovering"
      :mask-closable="!isRecovering"
    >
      <div class="recover-warning">
        <n-text type="warning"> 此操作將會覆蓋現有的伺服器檔案，且無法復原。 </n-text>
        <n-text depth="3" style="display: block; margin-top: 8px">
          即將還原的備份: <strong>{{ recoveringBackup?.name }}</strong>
        </n-text>
      </div>

      <n-input
        v-model:value="recoverConfirmText"
        placeholder="請輸入 CONFIRM 以確認"
        class="confirm-input"
        :disabled="isRecovering"
        @keyup.enter="confirmRecover"
      />

      <template #footer>
        <n-space justify="end">
          <n-button :disabled="isRecovering" @click="showRecoverModal = false"> 取消 </n-button>
          <n-button
            type="error"
            :loading="isRecovering"
            :disabled="recoverConfirmText !== 'CONFIRM'"
            @click="confirmRecover"
          >
            確認還原
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.backup-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
}

.backup-list {
  max-height: 200px;
  overflow-y: auto;
}

.backup-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.backup-item:last-child {
  border-bottom: none;
}

.backup-name {
  font-family: 'Fira Code', monospace;
  font-size: 12px;
}

.empty-backups {
  color: #666;
  font-size: 12px;
  text-align: center;
  padding: 20px 0;
}

.recover-modal {
  width: 90%;
  max-width: 450px;
}

.recover-warning {
  margin-bottom: 16px;
  padding: 12px;
  background: rgba(255, 165, 0, 0.1);
  border-radius: 4px;
  border: 1px solid rgba(255, 165, 0, 0.3);
}

.confirm-input {
  margin-top: 12px;
}
</style>
