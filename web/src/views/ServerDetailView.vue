<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick, unref } from 'vue';
import { RouterLink } from 'vue-router';
import {
  NSpace,
  NButton,
  NAvatar,
  NCard,
  NEmpty,
  NGrid,
  NGridItem,
  NTag,
  NInput,
  NSwitch,
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
import { useAuthStore } from '../store/auth';

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
const authStore = useAuthStore();
const isRoot = computed(() => authStore.isAdmin);

// 初始化活動紀錄 Store
activityLog.init();

// LocalStorage key 用於持久化指令歷史
const COMMAND_HISTORY_KEY = `mc_cmd_history_`;

const server = ref({
  status: 'Checking...',
});

const serverDetails = ref(null);
const serverInfoLoading = ref(true);
const serverUnavailable = ref(false);

const displayName = computed(() => serverDetails.value?.display_name || '');
const displayId = computed(() => serverDetails.value?.server_id || '');
const displayPlatform = computed(() => serverDetails.value?.mod_loader || '');

const formatMemBytes = bytes => {
  const n = Number(bytes);
  if (!Number.isFinite(n) || n <= 0) return '';
  const gb = 1024 * 1024 * 1024;
  const mb = 1024 * 1024;
  if (n % gb === 0) return `${n / gb}G`;
  if (n % mb === 0) return `${n / mb}M`;
  return String(n);
};

const memoryXmx = ref('');
const memoryXms = ref('');
const memorySaving = ref(false);
const jvmArgs = ref('');
const jvmArgsSaving = ref(false);

const syncMemoryInputs = detail => {
  memoryXmx.value = formatMemBytes(detail?.xmx);
  memoryXms.value = formatMemBytes(detail?.xms);
};

const syncJvmArgsInput = detail => {
  jvmArgs.value = detail?.args ?? '';
};

const saveMemory = async () => {
  const xmx = memoryXmx.value.trim();
  if (!xmx) {
    message.error('請填寫 Xmx');
    return;
  }
  memorySaving.value = true;
  try {
    const body = { xmx };
    const xms = memoryXms.value.trim();
    if (xms) body.xms = xms;
    await api.post(`/api/v1/server/memory/${props.id}`, body, { silent: true });
    message.success('記憶體設定已更新，下次啟動生效');
    await fetchServerInfo();
  } catch (err) {
    message.error('調整記憶體失敗: ' + getSanitizedErrorMessage(err));
  } finally {
    memorySaving.value = false;
  }
};

const saveJvmArgs = async () => {
  jvmArgsSaving.value = true;
  try {
    await api.post(`/api/v1/server/args/${props.id}`, { args: jvmArgs.value }, { silent: true });
    message.success('JVM 參數已更新，下次啟動生效');
    await fetchServerInfo();
  } catch (err) {
    message.error('調整 JVM 參數失敗: ' + getSanitizedErrorMessage(err));
  } finally {
    jvmArgsSaving.value = false;
  }
};

const fetchServerInfo = async () => {
  try {
    const res = await api.get(`/api/v1/server/details/${props.id}`, { silent: true });
    const detail = res?.server || res;
    if (!detail || !detail.server_id) {
      return;
    }
    serverUnavailable.value = false;
    serverDetails.value = detail;
    syncMemoryInputs(detail);
    syncJvmArgsInput(detail);
  } catch (err) {
    if ([403, 404].includes(err.response?.status)) {
      serverUnavailable.value = true;
      serverDetails.value = null;
      smartPolling?.stopPolling();
      return;
    }
    console.error('Failed to fetch server details:', err);
    message.error('無法載入伺服器資訊: ' + getSanitizedErrorMessage(err));
  } finally {
    serverInfoLoading.value = false;
  }
};

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
const actionInFlight = ref(false);
const actionsDisabled = computed(() => serverInfoLoading.value || serverUnavailable.value || actionInFlight.value || isOptimisticLoading.value || !['running', 'stopped'].includes(server.value.status));
const optimisticAction = ref(null); // 'starting' | 'stopping' | null
let optimisticTimer = null;
const clearOptimistic = () => {
  clearTimeout(optimisticTimer);
  isOptimisticLoading.value = false;
  optimisticAction.value = null;
};
// 伺服器可能一啟動就結束，狀態永遠不會變成預期值，逾時後要解除按鈕鎖定
const armOptimisticTimeout = () => {
  clearTimeout(optimisticTimer);
  optimisticTimer = setTimeout(clearOptimistic, 30000);
};

// 整合智慧輪詢 - 稍後在 fetchServerDetail 定義後初始化
let smartPolling = null;

const fetchServerDetail = async () => {
  if (serverUnavailable.value) return;
  try {
    const res = await api.get(`/api/v1/server/status/${props.id}`, { silent: true });


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
      if ((optimisticAction.value === 'starting' && status === 'running') ||
          (optimisticAction.value === 'stopping' && status === 'stopped')) {
        clearOptimistic();
      }

    } else {
      server.value.status = 'Unknown';
    }
    return status;
  } catch (err) {
    server.value.status = 'Error';
    console.error('Failed to fetch server status:', err);
  }
};

let previousLog = '';
let fetchingLogs = false;
const fetchLogs = async () => {
  if (serverUnavailable.value || document.hidden || fetchingLogs || !isRunning.value) return; // 伺服器未運行時不獲取日誌
  fetchingLogs = true;
  try {
    const res = await api.get(`/api/v1/server/log/${props.id}`, { silent: true });
    // API interceptor 已返回 response.data
    const logData = res.logs
    if (typeof logData === 'string' && consoleRef.value && logData !== previousLog) {
      if (logData.startsWith(previousLog)) {
        consoleRef.value.writeLog(logData.slice(previousLog.length));
      } else {
        consoleRef.value.term?.clear?.();
        consoleRef.value.writeLog(logData);
      }
      previousLog = logData;
    }
  } catch (err) {
    // 靜默處理錯誤，避免頻繁提示
    console.log('Log fetch skipped:', err.message);
  } finally {
    fetchingLogs = false;
  }
};

const handleStart = async () => {
  if (actionsDisabled.value) return;
  actionInFlight.value = true;
  // 樂觀更新：立即鎖定按鈕並顯示轉場狀態
  isOptimisticLoading.value = true;
  optimisticAction.value = 'starting';
  server.value.status = 'Starting...';

  try {
    await api.post(`/api/v1/server/start/${props.id}`, null, { silent: true });
    armOptimisticTimeout();
    message.success('正在啟動伺服器...');
    // 記錄活動
    activityLog.logServerStart(props.id, displayName.value);
    // 切換至活躍輪詢模式
    smartPolling?.enterActiveMode();
  } catch (err) {
    // 回滾樂觀更新
    isOptimisticLoading.value = false;
    optimisticAction.value = null;

    await fetchServerDetail();
    const errorMsg = getSanitizedErrorMessage(err);
    message.error('啟動失敗: ' + errorMsg);
    activityLog.logError('啟動伺服器', props.id, errorMsg);
  } finally {
    actionInFlight.value = false;
  }
};

const handleStop = async () => {
  if (actionsDisabled.value) return;
  actionInFlight.value = true;
  // 樂觀更新：立即鎖定按鈕並顯示轉場狀態
  isOptimisticLoading.value = true;
  optimisticAction.value = 'stopping';
  server.value.status = 'Stopping...';

  try {
    await api.post(`/api/v1/server/stop/${props.id}`, null, { silent: true });
    armOptimisticTimeout();
    message.success('正在停止伺服器...');
    // 記錄活動
    activityLog.logServerStop(props.id, displayName.value);
    // 切換至活躍輪詢模式
    smartPolling?.enterActiveMode();
  } catch (err) {
    // 回滾樂觀更新
    isOptimisticLoading.value = false;
    optimisticAction.value = null;
    const errorMsg = getSanitizedErrorMessage(err);
    message.error('停止失敗: ' + errorMsg);
    activityLog.logError('停止伺服器', props.id, errorMsg);
    await fetchServerDetail();
  } finally {
    actionInFlight.value = false;
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
    const res = await api.post(`/api/v1/server/property/${props.id}`, null, { silent: true });
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

const mods = ref([]);
const modsLoading = ref(false);
const modSearch = ref('');
const placeholderIcon = 'https://cdn.modrinth.com/placeholder.svg';
const updatingMods = ref({});

const filteredMods = computed(() => {
  const keyword = modSearch.value.trim().toLowerCase();
  if (!keyword) return mods.value;
  return mods.value.filter(mod => (mod.name || '').toLowerCase().includes(keyword));
});

const parseCategories = (raw) => {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw.filter(Boolean);
  if (typeof raw === 'string') {
    const trimmed = raw.trim();
    if (!trimmed) return [];
    try {
      const parsed = JSON.parse(trimmed);
      if (Array.isArray(parsed)) {
        return parsed.filter(Boolean);
      }
    } catch (err) {
      // ignore JSON parse errors, fallback to splitting string
    }
    return trimmed.split(',').map(item => item.trim()).filter(Boolean);
  }
  return [String(raw)];
};

const normalizeModList = (payload) => {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.data)) return payload.data;
  if (Array.isArray(payload?.message)) return payload.message;
  return [];
};

const fetchMods = async () => {
  modsLoading.value = true;
  try {
    const res = await api.get(`/api/v1/server/mod/list/${props.id}`, { silent: true });
    mods.value = normalizeModList(res);
  } catch (err) {
    mods.value = [];
    message.error('無法載入模組列表');
  } finally {
    modsLoading.value = false;
  }
};

const handleToggleMod = async (mod, nextValue) => {
  const prevValue = mod.enabled;
  mod.enabled = nextValue;
  try {
    await api.post(`/api/v1/server/mod/toggle/${props.id}`, null, { silent: true,
      params: { mod_id: mod.mod_id },
    });
    message.success(`${mod.name || mod.mod_id} 已${nextValue ? '啟用' : '停用'}`);
  } catch (err) {
    mod.enabled = prevValue;
    message.error('切換失敗: ' + getSanitizedErrorMessage(err));
  }
};

const handleDeleteMod = async (mod) => {
  try {
    await api.post(`/api/v1/server/mod/remove/${props.id}`, null, { silent: true,
      params: { mod_id: mod.mod_id },
    });
    mods.value = mods.value.filter(item => item.mod_id !== mod.mod_id);
    message.success('已刪除模組');
  } catch (err) {
    message.error('刪除失敗: ' + getSanitizedErrorMessage(err));
  }
};

const handleUpdateMod = async (mod) => {
  if (!mod?.mod_id || updatingMods.value[mod.mod_id]) return;
  updatingMods.value = { ...updatingMods.value, [mod.mod_id]: true };
  try {
    await api.post(`/api/v1/server/mod/update/${props.id}`, null, { silent: true,
      params: { mod_id: mod.mod_id },
      timeout: 600000,
    });
    await fetchMods();
    message.success('更新已送出');
  } catch (err) {
    message.error('更新失敗: ' + getSanitizedErrorMessage(err));
  } finally {
    const next = { ...updatingMods.value };
    delete next[mod.mod_id];
    updatingMods.value = next;
  }
};

const saveProperties = async () => {
  propertiesSaving.value = true;
  try {
    await api.post(`/api/v1/server/property/upload/${props.id}`, {
      texts: propertiesContent.value,
    }, { silent: true });
    message.success('設定已儲存');
  } catch (err) {
    message.error('儲存失敗: ' + getSanitizedErrorMessage(err));
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
  if (tabName === 'mods' && mods.value.length === 0) {
    fetchMods();
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
const handleBackupStop = async (done) => {
  try {
    await api.post(`/api/v1/server/stop/${props.id}`, null, { silent: true });
    smartPolling?.enterActiveMode();
  } catch (err) {
    done(err);
    return;
  }
  done();
};

const handleBackupStart = async (done) => {
  try {
    await api.post(`/api/v1/server/start/${props.id}`, null, { silent: true });
    smartPolling?.enterActiveMode();
  } catch (err) {
    done(err);
    return;
  }
  done();
};

onMounted(() => {
  // 載入持久化的指令歷史 - 移至 ServerConsole
  // loadCommandHistory();

  fetchServerInfo();

  // 初始化智慧輪詢
  smartPolling = useSmartPolling(fetchServerDetail, {
    idleInterval: 12000, // 閒置模式: 12 秒
    activeInterval: 2000, // 活躍模式: 2 秒
    activeDuration: 30000, // 活躍持續: 30 秒
    onStatusChange,
  });

  smartPolling.startPolling();

  // Logs 仍使用固定間隔
  const logInterval = setInterval(fetchLogs, 5000);

  // 儲存 cleanup 引用
  onBeforeUnmount(() => {
    smartPolling?.stopPolling();
    clearTimeout(optimisticTimer);
    clearInterval(logInterval);
  });
});
</script>

<template>
  <div class="server-detail-view">
    <n-text v-if="serverInfoLoading">正在載入伺服器資訊...</n-text>
    <n-space v-else-if="serverUnavailable" vertical :size="20" role="status">
      <n-text>找不到伺服器或沒有權限</n-text>
      <RouterLink to="/servers">
        <n-button>返回伺服器列表</n-button>
      </RouterLink>
    </n-space>
    <n-space v-else vertical :size="20">
      <div class="server-header">
        <n-space justify="space-between" align="center">
          <n-space align="baseline">
            <h2 class="title">{{ displayName }}</h2>
            <n-text depth="3" class="id-text">{{ displayId }}</n-text>
          </n-space>

          <n-space>
            <n-button type="primary" ghost :disabled="actionsDisabled || isRunning" @click="handleStart">
              <template #icon
                ><n-icon><CaretRightOutlined /></n-icon
              ></template>
              START
            </n-button>
            <n-popconfirm title="確定要停止伺服器嗎？" @positive-click="handleStop">
              <template #trigger>
                <n-button type="error" ghost :disabled="actionsDisabled || !isRunning">
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
                  <n-text strong>{{ displayPlatform }}</n-text>
                </div>
              </n-space>
            </n-card>

            <n-card title="MEMORY" class="info-card fade-in-up" style="animation-delay: 0.2s">
              <n-space vertical>
                <n-text depth="3">下次啟動生效</n-text>
                <div class="info-item">
                  <n-text depth="3">Xmx</n-text>
                  <n-input v-model:value="memoryXmx" size="small" placeholder="例如 2G" class="memory-input" />
                </div>
                <div class="info-item">
                  <n-text depth="3">Xms（可留空）</n-text>
                  <n-input v-model:value="memoryXms" size="small" placeholder="例如 1G" class="memory-input" />
                </div>
                <n-button type="primary" size="small" :loading="memorySaving" @click="saveMemory">
                  儲存記憶體
                </n-button>
              </n-space>
            </n-card>

            <n-card v-if="isRoot" title="JVM ARGS" class="info-card fade-in-up" style="animation-delay: 0.25s">
              <n-space vertical>
                <n-text depth="3">下次啟動生效</n-text>
                <n-input
                  v-model:value="jvmArgs"
                  type="textarea"
                  size="small"
                  placeholder="例如 -XX:+UseG1GC"
                  :autosize="{ minRows: 2, maxRows: 6 }"
                />
                <n-button type="primary" size="small" :loading="jvmArgsSaving" @click="saveJvmArgs">
                  儲存 JVM 參數
                </n-button>
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
          :type="activeTab === 'mods' ? 'primary' : 'default'"
          ghost
          @click="handleTabChange('mods')"
        >
          MODS
        </n-button>
        <n-button
          :type="activeTab === 'properties' ? 'primary' : 'default'"
          ghost
          @click="handleTabChange('properties')"
        >
          PROPERTIES
        </n-button>
      </div>

      <!-- Backups Tab -->
      <ServerBackups
        v-if="activeTab === 'backups'"
        :server-id="props.id"
        :server-name="displayName"
        :is-server-running="isRunning"
        class="fade-in-up"
        @recovery-started="smartPolling?.enterActiveMode()"
        @request-stop="handleBackupStop"
        @request-start="handleBackupStart"
        @backup-started="smartPolling?.enterActiveMode()"
      />

      <n-card
        v-if="activeTab === 'mods'"
        class="mods-section fade-in-up"
        style="animation-delay: 0.3s"
      >
        <template #header>
          <div class="mods-header">
            <div class="mods-title">
              <n-text strong>SERVER MODS</n-text>
              <n-text depth="3" class="mods-count">
                {{ filteredMods.length }} / {{ mods.length }}
              </n-text>
            </div>
            <n-input
              v-model:value="modSearch"
              size="small"
              clearable
              placeholder="搜尋模組名稱"
              class="mods-search"
            />
          </div>
        </template>

        <template v-if="modsLoading">
          <div class="loading-placeholder">
            <n-text depth="3">正在載入模組...</n-text>
          </div>
        </template>
        <template v-else>
          <n-empty v-if="filteredMods.length === 0" description="沒有可顯示的模組" />
          <div v-else class="mods-list">
            <div
              v-for="mod in filteredMods"
              :key="mod.mod_id"
              :class="['mod-row', { 'is-running': isRunning }]"
            >
              <n-avatar
                :src="mod.icon_url || placeholderIcon"
                :size="56"
                round
                class="mod-icon"
              />
              <div class="mod-main">
                <div class="mod-info">
                  <div class="mod-title-row">
                    <a
                      class="mod-link"
                      :href="`https://modrinth.com/mod/${mod.mod_id}`"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      <n-text strong>{{ mod.name }}</n-text>
                    </a>
                  </div>
                  <n-text depth="3" class="mod-summary">{{ mod.summary }}</n-text>
                  <div class="mod-categories">
                    <n-tag
                      v-for="cat in parseCategories(mod.categories)"
                      :key="`${mod.mod_id}-${cat}`"
                      size="small"
                      type="info"
                      class="category-tag"
                    >
                      {{ cat }}
                    </n-tag>
                  </div>
                </div>
                <div class="mod-version">
                  <n-text strong>{{ mod.version }}</n-text>
                  <n-text depth="3">{{ mod.game_versions }}</n-text>
                  <n-text v-if="updatingMods[mod.mod_id]" class="mod-updating" depth="3">
                    UPDATING...
                  </n-text>
                </div>
              </div>
              <div v-if="!isRunning" class="mod-actions">
                <n-switch
                  :value="mod.enabled"
                  :disabled="updatingMods[mod.mod_id]"
                  @update:value="value => handleToggleMod(mod, value)"
                />
                <n-popconfirm title="確定要刪除此模組嗎？" @positive-click="handleDeleteMod(mod)">
                  <template #trigger>
                    <n-button type="error" class="delete-btn" :disabled="updatingMods[mod.mod_id]">-</n-button>
                  </template>
                </n-popconfirm>
                <n-button
                  type="primary"
                  ghost
                  :loading="updatingMods[mod.mod_id]"
                  :disabled="updatingMods[mod.mod_id]"
                  @click="handleUpdateMod(mod)"
                >
                  UPDATE
                </n-button>
              </div>
            </div>
          </div>
        </template>
      </n-card>

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

.memory-input {
  width: 120px;
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

.mods-section {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
  margin-top: 20px;
}

.mods-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.mods-title {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.mods-count {
  font-size: 12px;
}

.mods-search {
  width: 240px;
}

.mods-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.mod-row {
  display: grid;
  grid-template-columns: 64px 1fr 220px;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border: 1px solid #333;
  border-radius: 10px;
  background-color: #16161c;
}

.mod-row.is-running {
  grid-template-columns: 56px 1fr;
}

.mod-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 12px;
}

.delete-btn {
  min-width: 36px;
  padding: 0 10px;
}

.mod-icon {
  background-color: #24242a;
}

.mod-main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 140px;
  align-items: start;
  gap: 12px;
  min-width: 0;
}

.mod-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.mod-info .n-text--strong {
  font-size: 16px;
}

.mod-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.mod-link {
  display: inline-flex;
  align-items: center;
  text-decoration: none;
  color: inherit;
}

.mod-link:hover .n-text {
  color: #7fd1ff !important;
}

.mod-link:focus-visible {
  outline: 1px solid #7fd1ff;
  border-radius: 4px;
}

.mod-summary {
  font-size: 13px;
  line-height: 1.4;
}

.mod-categories {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.category-tag {
  background-color: rgba(92, 196, 255, 0.15) !important;
  border-color: rgba(92, 196, 255, 0.4) !important;
}

.mod-version {
  display: flex;
  flex-direction: column;
  gap: 4px;
  text-align: right;
  min-width: 120px;
  margin-right: 15%;
}

.mod-version .n-text {
  font-size: 13px;
}

.mod-version .n-text--strong {
  font-size: 14px;
}

.mod-updating {
  color: #7fd1ff;
  font-size: 12px;
  letter-spacing: 0.5px;
}

/* Tab 切換 */
.tab-switcher {
  display: flex;
  gap: 8px;
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #333;
}

@media (max-width: 768px) {
  .mods-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
  }

  .mods-search {
    width: 100%;
  }

  .mod-row {
    grid-template-columns: 48px 1fr;
    align-items: start;
  }

  .mod-row.is-running {
    grid-template-columns: 48px 1fr;
  }

  .mod-actions {
    grid-column: 1 / -1;
    justify-content: flex-start;
  }

  .mod-version {
    text-align: left;
  }

  .mod-main {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 767px) {
  .tab-switcher { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; }
  .tab-switcher .n-button { min-height: 44px; width: 100%; padding: 0 4px; }
  .mod-actions { flex-wrap: wrap; }
  .id-text { overflow-wrap: anywhere; }
}
</style>
