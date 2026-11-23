<script setup>
import { ref, computed, onMounted } from 'vue';
import { NCard, NButton, NIcon, NText, NSpace, NModal, NInput, useMessage } from 'naive-ui';
import { SaveOutlined } from '@vicons/antd';
import api, { getSanitizedErrorMessage } from '../api';
import { useActivityLogStore } from '../store/activityLog';
import { useAuthStore } from '../store/auth';
import { UNLIMITED_LABEL } from '../utils/roles';

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
const auth = useAuthStore();

// Backups state
const backups = ref([]);
const loading = ref(false);

const canBackup = computed(() => !!auth.user?.permissions?.can_back_up_server);
const unlimited = computed(() => !!auth.user?.permissions?.unlimited);
const maxBackups = computed(() => auth.user?.permissions?.max_back_up_count);
const backupLimitText = computed(() => (unlimited.value ? UNLIMITED_LABEL : maxBackups.value ?? '—'));

// 滾動備份：達上限時後端會刪掉最舊的備份（名稱是時間戳，排序後最前面最舊）
const backupsToRotate = computed(() => {
  const max = Number(maxBackups.value);
  if (unlimited.value || !Number.isFinite(max) || max <= 0 || backups.value.length < max) return [];
  const names = backups.value.map(b => b.name).sort();
  return names.slice(0, names.length - max + 1);
});

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
    const res = await api.get(`/api/v1/server/list/backup/${props.serverId}`, { silent: true });
    // api interceptor returns response.data directly
    backups.value = res && res.backups ? res.backups.map(b => ({ name: b })) : [];
  } catch (err) {
    backups.value = [];
    message.error('無法載入備份列表: ' + getSanitizedErrorMessage(err));
  } finally {
    loading.value = false;
  }
};

// 運行中或已達上限時先確認（兩種提醒合併在同一個對話框）
const handleBackup = async () => {
  if (props.isServerRunning || backupsToRotate.value.length > 0) {
    showBackupConfirmModal.value = true;
    return;
  }

  // 伺服器已停止且未達上限，直接執行備份
  await executeBackup();
};

const confirmBackup = async () => {
  if (props.isServerRunning) {
    await executeFullBackupFlow();
    return;
  }
  showBackupConfirmModal.value = false;
  isBackingUp.value = true;
  backupProgress.value = '正在建立備份...';
  try {
    await executeBackup();
  } finally {
    isBackingUp.value = false;
    backupProgress.value = '';
  }
};

// 執行備份（伺服器必須已停止）
const executeBackup = async () => {
  try {
    await api.post(`/api/v1/server/backup/${props.serverId}`, {}, { timeout: 600000, silent: true });
    message.success('備份完成！');
    activityLog.logBackup(props.serverId, props.serverName);
    fetchBackups();
    return true;
  } catch (err) {
    const errorMsg = getSanitizedErrorMessage(err);
    message.error(`備份失敗: ${errorMsg}`);
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
    await new Promise((resolve, reject) => emit('request-stop', error => error ? reject(error) : resolve()));
    
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
      return;
    }
    
    // 步驟 3: 重啟伺服器
    backupProgress.value = '正在重新啟動伺服器...';
    message.info('備份完成，正在重新啟動伺服器...');
    await new Promise((resolve, reject) => emit('request-start', error => error ? reject(error) : resolve()));
    
    message.success('備份流程完成！伺服器正在重新啟動。');
    emit('backup-started');
    
  } catch (err) {
    message.error('備份流程發生錯誤: ' + getSanitizedErrorMessage(err));
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
    await api.post(`/api/v1/server/recover`, {
      server_id: props.serverId,
      file_name: recoveringBackup.value.name,
    }, { timeout: 600000, silent: true });
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
    <n-card class="backup-card">
      <template #header>
        <span>BACKUPS</span>
        <n-text v-if="canBackup" depth="3" class="backup-count">{{ backups.length }} / {{ backupLimitText }}</n-text>
      </template>
      <template #header-extra>
        <n-button 
          v-if="auth.user?.permissions?.can_back_up_server"
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
          <n-button v-if="auth.user?.permissions?.can_back_up_server" size="tiny" ghost type="warning" @click="openRecoverModal(b)"> 還原 </n-button>
        </div>
      </div>
    </n-card>

    <!-- 還原確認 Modal -->
    <n-modal
      v-model:show="showRecoverModal"
      preset="card"
      title="☠️ 還原備份 - 危險操作"
      :style="{ width: 'min(450px, 94vw)' }"
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

    <!-- 備份確認 Modal (伺服器運行中／已達備份上限) -->
    <n-modal
      v-model:show="showBackupConfirmModal"
      preset="card"
      :title="isServerRunning ? '⚠️ 伺服器運行中' : '⚠️ 已達備份上限'"
      :style="{ width: 'min(450px, 94vw)' }"
      :closable="true"
      :mask-closable="true"
    >
      <div v-if="backupsToRotate.length" class="rotate-warning">
        <n-text type="warning">
          已達上限，建立新備份會刪除最舊的備份 <strong>{{ backupsToRotate.join('、') }}</strong>
        </n-text>
      </div>
      <div v-if="isServerRunning" class="backup-confirm-content">
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
          <n-button type="primary" class="confirm-backup-btn" @click="confirmBackup">
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

.backup-count {
  margin-left: 8px;
  font-size: 12px;
}

.rotate-warning {
  margin-bottom: 12px;
  overflow-wrap: anywhere;
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
