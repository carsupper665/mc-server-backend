<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick, unref } from 'vue';
import {
  NSpace,
  NButton,
  NCard,
  NGrid,
  NGridItem,
  NTag,
  NInput,
  NText,
  useMessage,
  NPopconfirm,
  NIcon,
} from 'naive-ui';
import {
  CaretRightOutlined,
  PoweroffOutlined,
  SaveOutlined,
} from '@vicons/antd';
import api, { getSanitizedErrorMessage } from '../api';
import { useSmartPolling } from '../composables/useSmartPolling';
import PropertiesEditor from '../components/PropertiesEditor.vue';
import MacroPanel from '../components/MacroPanel.vue';
import ServerBackups from '../components/ServerBackups.vue';
import ServerConsole from '../components/ServerConsole.vue';
import { useActivityLogStore } from '../store/activityLog';

// Macro 狀態
const macroStatus = ref(null);
const handleMacroStatus = status => {
  macroStatus.value = status;
};

const props = defineProps({
  id: {
    type: String,
    required: true,
  },
});

const message = useMessage();
const activityLog = useActivityLogStore();

// 初始化活動紀錄 Store
activityLog.init();

// LocalStorage key 用於持久化指令歷史
const COMMAND_HISTORY_KEY = `mc_cmd_history_`;

// 從 server ID 解析伺服器類型
const parseServerType = id => {
  if (id.startsWith('mcsfv')) return 'Fabric';
  if (id.startsWith('mcsvv')) return 'Vanilla';
  return 'Unknown';
};

const server = ref({
  id: props.id,
  display_name: props.id, // 使用 ID 作為預設名稱
  status: 'Checking...',
  type: parseServerType(props.id),
});

// 使用 computed 處理狀態比對（大小寫不敏感）
const isRunning = computed(() => {
  const status = server.value.status?.toLowerCase?.() || '';
  return status === 'running';
});

const command = ref('');
const consoleRef = ref(null);
// 透過 computed 取得 console 元件中的 terminal 實例 (需要解包 ref)，供 MacroPanel 使用
const termInstance = computed(() => unref(consoleRef.value?.term));

// 樂觀更新狀態
const isOptimisticLoading = ref(false);
const optimisticAction = ref(null); // 'starting' | 'stopping' | null

// 整合智慧輪詢 - 稍後在 fetchServerDetail 定義後初始化
let smartPolling = null;

const fetchServerDetail = async () => {
  try {
    const res = await api.post(`/mc-api/a/status/${props.id}`);


    // API interceptor 已返回 response.data，所以 res 就是實際資料
    let status = null;

    if (typeof res === 'string') {
      // 直接回傳字串 "Running" 或 "Stopped"
      status = res;
    } else if (res && typeof res === 'object') {
      // 物件格式 { status: "Running" } 或 { data: "Running" }
      status = res.status || res.data || res.message || null;
    }

    if (status) {
      server.value.status = status;

    } else {
      server.value.status = 'Unknown';
    }
  } catch (err) {
    server.value.status = 'Error';
    console.error('Failed to fetch server status:', err);
  }
};

const fetchLogs = async () => {
  if (!isRunning.value) return; // 伺服器未運行時不獲取日誌
  try {
    const res = await api.get(`/server-api/a/log/${props.id}`);
    // API interceptor 已返回 response.data
<<<<<<< HEAD
    const logData = res.logs
    if (logData && typeof logData === 'string' && consoleRef.value) {
      consoleRef.value.term?.value?.clear?.();
=======
    const logData =
      typeof res === 'string'
        ? res
        : typeof res === 'object' && res
          ? res.logs || res.log || res.data || ''
          : '';

    if (logData && typeof logData === 'string' && consoleRef.value) {
      consoleRef.value.term?.value?.clear?.(); // replace console content
>>>>>>> 6b16455ec9382b83ef1210bfcc64bdb1ab160a2b
      consoleRef.value.writeLog(logData);
    }
  } catch (err) {
    // 靜默處理錯誤，避免頻繁提示
    console.log('Log fetch skipped:', err.message);
  }
};

const handleStart = async () => {
  // 樂觀更新：立即鎖定按鈕並顯示轉場狀態
  isOptimisticLoading.value = true;
  optimisticAction.value = 'starting';
  server.value.status = 'Starting...';

  try {
    await api.post(`/mc-api/a/start/${props.id}`);
    message.success('正在啟動伺服器...');
    // 記錄活動
    activityLog.logServerStart(props.id, server.value.display_name);
    // 切換至活躍輪詢模式
    smartPolling?.enterActiveMode();
  } catch (err) {
    // 回滾樂觀更新
    isOptimisticLoading.value = false;
    optimisticAction.value = null;

    if (err.response?.data?.error === 'server already running') {
      server.value.status = 'Running';
      return;
    }
    const errorMsg = getSanitizedErrorMessage(err);
    message.error('啟動失敗: ' + errorMsg);
    activityLog.logError('啟動伺服器', props.id, errorMsg);
    smartPolling?.pollNow();
  }
};

const handleStop = async () => {
  // 樂觀更新：立即鎖定按鈕並顯示轉場狀態
  isOptimisticLoading.value = true;
  optimisticAction.value = 'stopping';
  server.value.status = 'Stopping...';

  try {
    await api.post(`/mc-api/a/stop/${props.id}`);
    message.success('正在停止伺服器...');
    // 記錄活動
    activityLog.logServerStop(props.id, server.value.display_name);
    // 切換至活躍輪詢模式
    smartPolling?.enterActiveMode();
  } catch (err) {
    // 回滾樂觀更新
    isOptimisticLoading.value = false;
    optimisticAction.value = null;
    const errorMsg = getSanitizedErrorMessage(err);
    message.error('停止失敗: ' + errorMsg);
    activityLog.logError('停止伺服器', props.id, errorMsg);
    smartPolling?.pollNow();
  }
};





// Properties 設定檔管理
const activeTab = ref('console');
const propertiesContent = ref('');
const propertiesLoading = ref(false);
const propertiesSaving = ref(false);

const fetchProperties = async () => {
  propertiesLoading.value = true;
  try {
    const res = await api.post(`/mc-api/a/property/${props.id}`);
    // Backend returns { message: "...", property: "..." }
    // API interceptor returns response.data directly
    propertiesContent.value = typeof res === 'string' ? res : res.property || '';
  } catch (err) {
    console.error('Failed to load properties:', err);
    message.error('無法載入伺服器設定');
  } finally {
    propertiesLoading.value = false;
  }
};

const saveProperties = async () => {
  propertiesSaving.value = true;
  try {
    await api.post(`/mc-api/a/property/${props.id}`, {
      content: propertiesContent.value,
    });
    message.success('設定已儲存');
  } catch (err) {
    message.error('儲存失敗: ' + (err.response?.data?.error || '未知錯誤'));
  } finally {
    propertiesSaving.value = false;
  }
};

// 當切換到 Properties Tab 時載入設定
const handleTabChange = tabName => {
  activeTab.value = tabName;
  if (tabName === 'properties' && !propertiesContent.value) {
    fetchProperties();
  }
};



// 狀態變更回調 - 用於提前退出活躍模式並重置樂觀狀態
const onStatusChange = (newStatus, oldStatus) => {
  console.log(`[SmartPolling] 狀態變更: ${oldStatus} -> ${newStatus}`);
  isOptimisticLoading.value = false;
  optimisticAction.value = null;
};

// 當指令發送後延遲刷新日誌 (解決模板中 setTimeout 無法識別的問題)
const handleCommandSent = () => {
  setTimeout(() => fetchLogs(), 500);
};

// 備份流程相關函數 - 被 ServerBackups 元件呼叫
const handleBackupStop = async () => {
  try {
    await api.post(`/mc-api/a/stop/${props.id}`);
    smartPolling?.enterActiveMode();
  } catch (err) {
    console.error('Backup stop error:', err);
  }
};

const handleBackupStart = async () => {
  try {
    await api.post(`/mc-api/a/start/${props.id}`);
    smartPolling?.enterActiveMode();
  } catch (err) {
    console.error('Backup start error:', err);
  }
};

onMounted(() => {
  // 載入持久化的指令歷史 - 移至 ServerConsole
  // loadCommandHistory();

  // 初始化智慧輪詢
  smartPolling = useSmartPolling(fetchServerDetail, {
    idleInterval: 12000, // 閒置模式: 12 秒
    activeInterval: 2000, // 活躍模式: 2 秒
    activeDuration: 30000, // 活躍持續: 30 秒
    onStatusChange,
  });

  // 開始輪詢
  smartPolling.startPolling();

  // Logs 仍使用固定間隔
  const logInterval = setInterval(fetchLogs, 5000);

  // 儲存 cleanup 引用
  onBeforeUnmount(() => {
    smartPolling?.stopPolling();
    clearInterval(logInterval);
  });
});
</script>

<template>
  <div class="server-detail-view">
    <n-space vertical :size="20">
      <div class="server-header">
        <n-space justify="space-between" align="center">
          <n-space align="baseline">
            <h2 class="title">{{ server.display_name }}</h2>
            <n-text depth="3" class="id-text">{{ server.id }}</n-text>
          </n-space>

          <n-space>
            <n-button type="primary" ghost :disabled="isRunning" @click="handleStart">
              <template #icon
                ><n-icon><CaretRightOutlined /></n-icon
              ></template>
              START
            </n-button>
            <n-popconfirm title="確定要停止伺服器嗎？" @positive-click="handleStop">
              <template #trigger>
                <n-button type="error" ghost :disabled="!isRunning">
                  <template #icon
                    ><n-icon><PoweroffOutlined /></n-icon
                  ></template>
                  STOP
                </n-button>
              </template>
            </n-popconfirm>
          </n-space>
        </n-space>
      </div>

      <n-grid cols="1 m:4" responsive="screen" :x-gap="20" :y-gap="20">
        <n-grid-item span="1 m:3">
          <ServerConsole
            ref="consoleRef"
            :server-id="props.id"
            :is-running="isRunning"
            :macro-status="macroStatus"
            @command-sent="handleCommandSent"
          />
        </n-grid-item>

        <n-grid-item span="1 m:1">
          <n-space vertical :size="20">
            <n-card title="SYSTEM INFO" class="info-card fade-in-up" style="animation-delay: 0.1s">
              <n-space vertical>
                <div class="info-item">
                  <n-text depth="3">STATUS</n-text>
                  <n-tag :type="isRunning ? 'success' : 'error'" size="small">
                    {{ server.status }}
                  </n-tag>
                </div>
                <div class="info-item">
                  <n-text depth="3">PLATFORM</n-text>
                  <n-text strong>{{ server.type }}</n-text>
                </div>
              </n-space>
            </n-card>



            <!-- 巨集面板 -->
            <MacroPanel
              :server-id="props.id"
              :terminal="termInstance"
              :disabled="!isRunning"
              class="fade-in-up"
              style="animation-delay: 0.3s"
              @update:status="handleMacroStatus"
            />
          </n-space>
        </n-grid-item>
      </n-grid>

      <!-- 還原確認 Modal -->


      <!-- Backups Tab -->
      <ServerBackups
        v-if="activeTab === 'backups'"
        :server-id="props.id"
        :server-name="server.display_name"
        :is-server-running="isRunning"
        class="fade-in-up"
        @recovery-started="smartPolling?.enterActiveMode()"
        @request-stop="handleBackupStop"
        @request-start="handleBackupStart"
        @backup-started="smartPolling?.enterActiveMode()"
      />

      <!-- 設定檔 Tab -->
      <n-card
        v-if="activeTab === 'properties'"
        class="properties-section fade-in-up"
        style="animation-delay: 0.3s"
      >
        <template #header>
          <n-space justify="space-between" align="center">
            <n-text strong>SERVER PROPERTIES</n-text>
            <n-button
              type="primary"
              size="small"
              :loading="propertiesSaving"
              @click="saveProperties"
            >
              <template #icon
                ><n-icon><SaveOutlined /></n-icon
              ></template>
              SAVE CHANGES
            </n-button>
          </n-space>
        </template>

        <template v-if="propertiesLoading">
          <div class="loading-placeholder">
            <n-text depth="3">正在載入設定檔...</n-text>
          </div>
        </template>
        <template v-else>
          <PropertiesEditor v-model="propertiesContent" />
        </template>
      </n-card>

      <!-- Tab 切換 -->
      <div class="tab-switcher">
        <n-button
          :type="activeTab === 'console' ? 'primary' : 'default'"
          ghost
          @click="handleTabChange('console')"
        >
          CONSOLE
        </n-button>
        <n-button
          :type="activeTab === 'backups' ? 'primary' : 'default'"
          ghost
          @click="handleTabChange('backups')"
        >
          BACKUPS
        </n-button>
        <n-button
          :type="activeTab === 'properties' ? 'primary' : 'default'"
          ghost
          @click="handleTabChange('properties')"
        >
          PROPERTIES
        </n-button>
      </div>
    </n-space>
  </div>
</template>

<style scoped>
.title {
  margin: 0;
  font-size: 24px;
}

.id-text {
  font-family: 'Fira Code', monospace;
  font-size: 12px;
}

@media (max-width: 768px) {
  .server-detail-view {
    padding: 8px 4px;
  }

  .title {
    font-size: 18px;
  }

  .info-card,
  .backup-card {
    margin-bottom: 12px;
  }

  .tab-switcher {
    flex-wrap: wrap;
    gap: 6px;
  }

  .tab-switcher .n-button {
    flex: 1;
    min-width: 100px;
  }
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

.info-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
}

.info-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

/* Properties 區塊 */
.properties-section {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
  margin-top: 20px;
}

.loading-placeholder {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}

/* Tab 切換 */
.tab-switcher {
  display: flex;
  gap: 8px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #333;
}
</style>
