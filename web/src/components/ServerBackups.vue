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
  isServerRunning: {
    type: Boolean,
    default: false,
  },
});

const emit = defineEmits(['recovery-started', 'request-stop', 'request-start', 'backup-started']);

const message = useMessage();
const activityLog = useActivityLogStore();

// Backups state
const backups = ref([]);
const loading = ref(false);

// Backup confirmation modal state (for when server is running)
const showBackupConfirmModal = ref(false);
const isBackingUp = ref(false);
const backupProgress = ref(''); // 顯示備份流程狀態

// Recover modal state
const showRecoverModal = ref(false);
const recoverConfirmText = ref('');
const recoveringBackup = ref(null);
const isRecovering = ref(false);

const fetchBackups = async () => {
  loading.value = true;
  try {
    // 傳送空物件以確保 Content-Type header 正確
    const res = await api.post(`/mc-api/a/ls-backup/${props.serverId}`, {});
    // api interceptor returns response.data directly
    backups.value = res && res.backups ? res.backups.map(b => ({ name: b })) : [];
  } catch (err) {
    backups.value = [];
  } finally {
    loading.value = false;
  }
};

const handleBackup = async () => {
  // 如果伺服器正在運行，顯示確認對話框
  if (props.isServerRunning) {
    showBackupConfirmModal.value = true;
    return;
  }
  
  // 伺服器已停止，直接執行備份
  await executeBackup();
};

// 執行備份（伺服器必須已停止）
const executeBackup = async () => {
  try {
    await api.post(`/mc-api/a/backup/${props.serverId}`, {});
    message.success('備份完成！');
    activityLog.logBackup(props.serverId, props.serverName);
    fetchBackups();
    return true;
  } catch (err) {
    const errorMsg = getSanitizedErrorMessage(err);
    if (errorMsg.toLowerCase().includes('running') || err.response?.status === 500) {
      message.error('備份失敗：伺服器運行中無法備份');
    } else {
      message.error(`備份失敗: ${errorMsg}`);
    }
    activityLog.logError('備份失敗', props.serverId, errorMsg);
    return false;
  }
};

// 休眠函數
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

// 執行完整備份流程：停止 → 備份 → 重啟
const executeFullBackupFlow = async () => {
  isBackingUp.value = true;
  showBackupConfirmModal.value = false;
  
  try {
    // 步驟 1: 停止伺服器
    backupProgress.value = '正在停止伺服器...';
    message.info('正在停止伺服器以進行備份...');
    emit('request-stop');
    
    // 等待伺服器停止 (輪詢檢查)
    let attempts = 0;
    const maxAttempts = 30; // 最多等待 30 秒
    while (props.isServerRunning && attempts < maxAttempts) {
      await sleep(1000);
      attempts++;
    }
    
    if (props.isServerRunning) {
      message.error('伺服器停止超時，請手動停止後再試');
      return;
    }
    
    // 步驟 2: 執行備份
    backupProgress.value = '正在建立備份...';
    const backupSuccess = await executeBackup();
    
    if (!backupSuccess) {
      message.error('備份失敗，伺服器將保持停止狀態');
      return;
    }
    
    // 步驟 3: 重啟伺服器
    backupProgress.value = '正在重新啟動伺服器...';
    message.info('備份完成，正在重新啟動伺服器...');
    emit('request-start');
    
    message.success('備份流程完成！伺服器正在重新啟動。');
    emit('backup-started');
    
  } catch (err) {
    message.error('備份流程發生錯誤: ' + err.message);
  } finally {
    isBackingUp.value = false;
    backupProgress.value = '';
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
        <n-button 
          size="tiny" 
          tertiary 
          :disabled="isBackingUp"
          :loading="isBackingUp"
          title="建立新備份"
          @click="handleBackup"
        >
          <template #icon
            ><n-icon><SaveOutlined /></n-icon
          ></template>
          NEW
        </n-button>
      </template>

      <!-- 備份進度顯示 -->
      <div v-if="isBackingUp" class="backup-progress">
        <n-text type="warning">{{ backupProgress }}</n-text>
      </div>

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

    <!-- 備份確認 Modal (伺服器運行中) -->
    <n-modal
      v-model:show="showBackupConfirmModal"
      preset="card"
      title="⚠️ 伺服器運行中"
      class="backup-confirm-modal"
      :closable="true"
      :mask-closable="true"
    >
      <div class="backup-confirm-content">
        <n-text>
          伺服器目前正在運行中。備份需要先停止伺服器。
        </n-text>
        <n-text depth="3" style="display: block; margin-top: 8px">
          系統將自動執行以下流程：
        </n-text>
        <ol class="backup-steps">
          <li>停止伺服器</li>
          <li>建立備份</li>
          <li>重新啟動伺服器</li>
        </ol>
        <n-text type="warning" style="display: block; margin-top: 8px; font-size: 12px">
          ⚠️ 備份過程中伺服器將暫時無法使用
        </n-text>
      </div>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showBackupConfirmModal = false">取消</n-button>
          <n-button type="primary" @click="executeFullBackupFlow">
            確認備份
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

/* 備份確認 Modal */
.backup-confirm-modal {
  width: 90%;
  max-width: 450px;
}

.backup-confirm-content {
  line-height: 1.6;
}

.backup-steps {
  margin: 12px 0;
  padding-left: 24px;
  color: #aaa;
}

.backup-steps li {
  margin: 4px 0;
}

/* 備份進度 */
.backup-progress {
  padding: 8px 12px;
  background: rgba(234, 179, 8, 0.1);
  border-radius: 4px;
  border: 1px solid rgba(234, 179, 8, 0.3);
  margin-bottom: 12px;
  text-align: center;
}
</style>
