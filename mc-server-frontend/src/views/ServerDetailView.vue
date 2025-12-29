<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { 
  NSpace, NButton, NCard, NGrid, NGridItem, NTag, NTabs, NTabPane, 
  NInput, NText, NDivider, useMessage, NPopconfirm, NDataTable, NIcon,
  NAutoComplete, NModal
} from 'naive-ui';
import { 
  CaretRightOutlined, 
  PoweroffOutlined, 
  CodeOutlined, 
  SaveOutlined, 
  HistoryOutlined,
  SendOutlined,
  ClearOutlined
} from '@vicons/antd';
import api, { getSanitizedErrorMessage } from '../api';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { useSmartPolling } from '../composables/useSmartPolling';
import PropertiesEditor from '../components/PropertiesEditor.vue';
import MacroPanel from '../components/MacroPanel.vue';
import { useActivityLogStore } from '../store/activityLog';

const props = defineProps({
  id: {
    type: String,
    required: true
  }
});

const message = useMessage();
const activityLog = useActivityLogStore();

// 初始化活動紀錄 Store
activityLog.init();

// LocalStorage key 用於持久化指令歷史
const COMMAND_HISTORY_KEY = `mc_cmd_history_`;

// 從 server ID 解析伺服器類型
const parseServerType = (id) => {
  if (id.startsWith('mcsfv')) return 'Fabric';
  if (id.startsWith('mcsvv')) return 'Vanilla';
  return 'Unknown';
};

const server = ref({
  id: props.id,
  display_name: props.id, // 使用 ID 作為預設名稱
  status: 'Checking...',
  type: parseServerType(props.id)
});

// 使用 computed 處理狀態比對（大小寫不敏感）
const isRunning = computed(() => {
  const status = server.value.status?.toLowerCase?.() || '';
  return status === 'running';
});

const command = ref('');
const commandHistory = ref([]);
const historyIndex = ref(-1);
const showSuggestions = ref(false);
const terminalRef = ref(null);
let term = null;
let fitAddon = null;

// 自動捲動開關
const autoScroll = ref(true);

// 樂觀更新狀態
const isOptimisticLoading = ref(false);
const optimisticAction = ref(null); // 'starting' | 'stopping' | null

// 整合智慧輪詢 - 稍後在 fetchServerDetail 定義後初始化
let smartPolling = null;

// 載入持久化的指令歷史
const loadCommandHistory = () => {
  try {
    const stored = localStorage.getItem(COMMAND_HISTORY_KEY + props.id);
    if (stored) {
      commandHistory.value = JSON.parse(stored);
    }
  } catch (err) {
    console.error('Failed to load command history:', err);
  }
};

// 儲存指令歷史到 LocalStorage
const saveCommandHistory = () => {
  try {
    localStorage.setItem(
      COMMAND_HISTORY_KEY + props.id,
      JSON.stringify(commandHistory.value.slice(0, 50))
    );
  } catch (err) {
    console.error('Failed to save command history:', err);
  }
};

const initTerminal = () => {
  term = new Terminal({
    theme: {
      background: '#0c0c0e',
      foreground: '#d4d4d4',
      cursor: '#18a058'
    },
    fontFamily: 'Fira Code, monospace',
    fontSize: 13,
    convertEol: true,
    cursorBlink: true,
    disableStdin: true
  });
  
  fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.open(terminalRef.value);
  fitAddon.fit();
  
  term.writeln('\x1b[1;32m[ SYSTEM ] Terminal initialized. Waiting for logs...\x1b[0m');
};

const fetchServerDetail = async () => {
  try {
    const res = await api.post(`/mc-api/a/status/${props.id}`);
    console.log('[DEBUG] Status API response:', res, typeof res);
    
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
      console.log('[DEBUG] Server status set to:', status);
    } else {
      server.value.status = 'Unknown';
    }
  } catch (err) {
    server.value.status = 'Error';
    console.error('Failed to fetch server status:', err);
  }
};

const fetchLogs = async () => {
  try {
    const res = await api.get(`/server-api/a/log/${props.id}`);
    // API interceptor 已返回 response.data
    const logData = typeof res === 'string' ? res : (res?.data || res?.log || res);
    if (logData && typeof logData === 'string') {
      term.clear();
      term.write(logData);
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

// Minecraft 常用指令列表 (用於 Tab 補全)
const mcCommands = [
  '/help', '/list', '/say', '/tell', '/msg', '/me',
  '/gamemode', '/tp', '/teleport', '/give', '/clear',
  '/time', '/weather', '/difficulty', '/seed',
  '/ban', '/pardon', '/kick', '/op', '/deop', '/whitelist',
  '/stop', '/save-all', '/save-on', '/save-off', '/reload',
  '/setblock', '/fill', '/clone', '/summon', '/kill',
  '/effect', '/enchant', '/experience', '/xp',
  '/gamerule', '/worldborder', '/spawnpoint', '/setworldspawn',
  '/scoreboard', '/team', '/title', '/bossbar',
  '/execute', '/function', '/schedule', '/data', '/datapack'
];

// 過濾後的指令列表 (用於下拉建議)
const filteredCommands = computed(() => {
  if (!command.value || !command.value.startsWith('/')) {
    showSuggestions.value = false;
    return [];
  }
  const input = command.value.toLowerCase();
  const matches = mcCommands.filter(cmd => cmd.startsWith(input));
  showSuggestions.value = matches.length > 0 && command.value !== matches[0];
  return matches.slice(0, 8); // 最多顯示 8 個建議
});

// 選擇建議指令
const selectCommand = (cmd) => {
  command.value = cmd + ' ';
  showSuggestions.value = false;
};

// Tab 補全處理
const handleTabComplete = () => {
  if (!command.value) return;
  
  const input = command.value.toLowerCase();
  const matches = mcCommands.filter(cmd => cmd.startsWith(input));
  
  if (matches.length === 1) {
    command.value = matches[0] + ' ';
    showSuggestions.value = false;
  } else if (matches.length > 1) {
    term.writeln(`\x1b[1;33m[ TAB ] ${matches.join(', ')}\x1b[0m`);
  }
};

const sendCommand = async () => {
  if (!command.value) return;
  try {
    // 儲存指令歷史
    commandHistory.value.unshift(command.value);
    if (commandHistory.value.length > 50) {
      commandHistory.value.pop();
    }
    historyIndex.value = -1;
    // 持久化到 LocalStorage
    saveCommandHistory();
    
    const cmd = command.value;
    command.value = '';
    
    // 使用語法高亮顯示發送的指令
    const highlightedCmd = highlightCommand(cmd);
    term.writeln(`\x1b[1;36m❯\x1b[0m ${highlightedCmd}`);
    
    await api.post(`/mc-api/a/cmd/${props.id}`, { command: cmd });
    
    // 發送指令後短暫延遲再獲取 logs，讓伺服器有時間回應
    setTimeout(() => {
      fetchLogs();
    }, 500);
    
  } catch (err) {
    term.writeln(`\x1b[1;31m[ ERROR ] 指令發送失敗\x1b[0m`);
  }
};

// 指令語法高亮
const highlightCommand = (cmd) => {
  // 斜線指令標綠色
  if (cmd.startsWith('/')) {
    const parts = cmd.split(' ');
    const cmdName = `\x1b[1;32m${parts[0]}\x1b[0m`; // 綠色指令名
    const args = parts.slice(1).map(arg => {
      // 數字標黃色
      if (/^\d+$/.test(arg)) return `\x1b[1;33m${arg}\x1b[0m`;
      // @選擇器標紫色
      if (arg.startsWith('@')) return `\x1b[1;35m${arg}\x1b[0m`;
      // 其他參數標白色
      return `\x1b[1;37m${arg}\x1b[0m`;
    });
    return [cmdName, ...args].join(' ');
  }
  // 普通聊天訊息標白色
  return `\x1b[0;37m${cmd}\x1b[0m`;
};

// 處理指令輸入框的鍵盤事件 (上/下箭頭切換歷史, Tab 補全)
const handleCommandKeydown = (e) => {
  if (e.key === 'Tab') {
    e.preventDefault();
    handleTabComplete();
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    if (historyIndex.value < commandHistory.value.length - 1) {
      historyIndex.value++;
      command.value = commandHistory.value[historyIndex.value];
    }
  } else if (e.key === 'ArrowDown') {
    e.preventDefault();
    if (historyIndex.value > 0) {
      historyIndex.value--;
      command.value = commandHistory.value[historyIndex.value];
    } else if (historyIndex.value === 0) {
      historyIndex.value = -1;
      command.value = '';
    }
  }
};

const backups = ref([]);
const fetchBackups = async () => {
  try {
    const res = await api.post(`/mc-api/a/ls-backup/${props.id}`);
    backups.value = (res.data && res.data.backups) ? res.data.backups.map(b => ({ name: b })) : [];
  } catch (err) {
    // Backend returns 500 if backup folder doesn't exist
    backups.value = [];
  }
};

const handleBackup = async () => {
  try {
    await api.post(`/mc-api/a/backup/${props.id}`);
    message.success('備份任務已啟動');
    activityLog.logBackup(props.id, server.value.display_name);
    fetchBackups();
  } catch (err) {
    message.error('備份失敗');
  }
};

// 還原確認 Modal
const showRecoverModal = ref(false);
const recoverConfirmText = ref('');
const recoveringBackup = ref(null);
const isRecovering = ref(false);

// 開啟還原確認 Modal
const openRecoverModal = (backup) => {
  recoveringBackup.value = backup;
  recoverConfirmText.value = '';
  showRecoverModal.value = true;
};

// 確認還原
const confirmRecover = async () => {
  if (recoverConfirmText.value !== 'CONFIRM') {
    message.warning('請輸入 CONFIRM 以確認還原操作');
    return;
  }
  
  isRecovering.value = true;
  try {
    await api.post(`/mc-api/a/recover/${props.id}`, {
      backup_name: recoveringBackup.value.name
    });
    message.success('還原成功！伺服器將重新啟動。');
    activityLog.logRecover(props.id, server.value.display_name, recoveringBackup.value.name);
    showRecoverModal.value = false;
    smartPolling?.enterActiveMode();
  } catch (err) {
    const errorMsg = getSanitizedErrorMessage(err);
    message.error('還原失敗: ' + errorMsg);
    activityLog.logError('還原備份', props.id, errorMsg);
  } finally {
    isRecovering.value = false;
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
    // API 回傳純文字字串
    propertiesContent.value = typeof res === 'string' ? res : (res.data || res.content || '');
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
      content: propertiesContent.value
    });
    message.success('設定已儲存');
  } catch (err) {
    message.error('儲存失敗: ' + (err.response?.data?.error || '未知錯誤'));
  } finally {
    propertiesSaving.value = false;
  }
};

// 當切換到 Properties Tab 時載入設定
const handleTabChange = (tabName) => {
  activeTab.value = tabName;
  if (tabName === 'properties' && !propertiesContent.value) {
    fetchProperties();
  }
};

// 清除終端機畫面
const clearTerminal = () => {
  if (term) {
    term.clear();
    term.writeln('\x1b[1;33m[ SYSTEM ] Terminal cleared.\x1b[0m');
  }
};

// 狀態變更回調 - 用於提前退出活躍模式並重置樂觀狀態
const onStatusChange = (newStatus, oldStatus) => {
  console.log(`[SmartPolling] 狀態變更: ${oldStatus} -> ${newStatus}`);
  isOptimisticLoading.value = false;
  optimisticAction.value = null;
};

onMounted(() => {
  initTerminal();
  fetchBackups();
  
  // 載入持久化的指令歷史
  loadCommandHistory();
  
  // 初始化智慧輪詢
  smartPolling = useSmartPolling(fetchServerDetail, {
    idleInterval: 12000,  // 閒置模式: 12 秒
    activeInterval: 2000, // 活躍模式: 2 秒
    activeDuration: 30000, // 活躍持續: 30 秒
    onStatusChange
  });
  
  // 開始輪詢
  smartPolling.startPolling();
  
  // Logs 仍使用固定間隔
  const logInterval = setInterval(fetchLogs, 5000);
  
  window.addEventListener('resize', () => fitAddon?.fit());
  
  // 儲存 cleanup 引用
  onBeforeUnmount(() => {
    smartPolling?.stopPolling();
    clearInterval(logInterval);
    term?.dispose();
    window.removeEventListener('resize', () => fitAddon?.fit());
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
            <n-button 
              type="primary" 
              ghost 
              :disabled="isRunning"
              @click="handleStart"
            >
              <template #icon><n-icon><CaretRightOutlined /></n-icon></template>
              START
            </n-button>
            <n-popconfirm @positive-click="handleStop" title="確定要停止伺服器嗎？">
              <template #trigger>
                <n-button 
                  type="error" 
                  ghost 
                  :disabled="!isRunning"
                >
                  <template #icon><n-icon><PoweroffOutlined /></n-icon></template>
                  STOP
                </n-button>
              </template>
            </n-popconfirm>
          </n-space>
        </n-space>
      </div>

      <n-grid cols="1 m:4" responsive="screen" :x-gap="20" :y-gap="20">
        <n-grid-item span="1 m:3">
          <n-card class="terminal-card fade-in-up" :bordered="false">
            <template #header>
              <n-space align="center" justify="space-between" class="terminal-header">
                <n-space align="center" :size="12">
                  <div class="status-indicator" :class="{ 'status-online': isRunning }">
                    <div class="status-dot"></div>
                  </div>
                  <n-icon color="#18a058" size="18"><CodeOutlined /></n-icon>
                  <n-text strong class="console-title">LIVE CONSOLE</n-text>
                  <n-tag 
                    :type="isRunning ? 'success' : 'error'" 
                    size="small"
                    class="status-tag"
                  >
                    {{ isRunning ? 'CONNECTED' : 'OFFLINE' }}
                  </n-tag>
                </n-space>
                <n-button size="tiny" quaternary @click="clearTerminal" title="Clear Console">
                  <template #icon><n-icon><ClearOutlined /></n-icon></template>
                </n-button>
              </n-space>
            </template>
            
            <div class="terminal-wrapper" :class="{ 'terminal-glow': isRunning }">
              <div ref="terminalRef" class="terminal-container"></div>
            </div>
            
            <div class="command-input-area">
              <div class="input-wrapper">
                <n-input 
                  v-model:value="command" 
                  placeholder="輸入指令... (Tab 補全, ↑↓ 歷史)" 
                  @keyup.enter="sendCommand"
                  @keydown="handleCommandKeydown"
                  :disabled="!isRunning"
                  class="command-input"
                >
                  <template #prefix>
                    <n-text :depth="isRunning ? 1 : 3" class="prompt-symbol">❯</n-text>
                  </template>
                </n-input>
                <!-- 指令建議下拉選單 -->
                <div v-if="showSuggestions && filteredCommands.length > 0" class="suggestions-dropdown">
                  <div 
                    v-for="cmd in filteredCommands" 
                    :key="cmd" 
                    class="suggestion-item"
                    @click="selectCommand(cmd)"
                  >
                    <span class="cmd-name">{{ cmd }}</span>
                  </div>
                </div>
              </div>
              <n-button 
                type="primary" 
                :disabled="!isRunning || !command"
                @click="sendCommand"
                class="send-btn"
              >
                <template #icon><n-icon><SendOutlined /></n-icon></template>
              </n-button>
            </div>
          </n-card>
        </n-grid-item>
        
        <n-grid-item span="1 m:1">
          <n-space vertical :size="20">
            <n-card title="SYSTEM INFO" class="info-card fade-in-up" style="animation-delay: 0.1s;">
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
            
            <n-card title="BACKUPS" class="backup-card fade-in-up" style="animation-delay: 0.2s;">
              <template #header-extra>
                <n-button size="tiny" tertiary @click="handleBackup">
                  <template #icon><n-icon><SaveOutlined /></n-icon></template>
                  NEW
                </n-button>
              </template>
              
              <div class="backup-list">
                <div v-if="backups.length === 0" class="empty-backups">
                  No backups found
                </div>
                <div v-for="b in backups" :key="b.name" class="backup-item">
                  <n-text class="backup-name">{{ b.name }}</n-text>
                  <!-- 移除下載按鈕，後端不支援 -->
                  <n-button 
                    size="tiny" 
                    ghost 
                    type="warning"
                    @click="openRecoverModal(b)"
                  >
                    還原
                  </n-button>
                </div>
              </div>
            </n-card>
            
            <!-- 巨集面板 -->
            <MacroPanel 
              :server-id="props.id"
              :terminal="term"
              :disabled="!isRunning"
              class="fade-in-up"
              style="animation-delay: 0.3s;"
            />
          </n-space>
        </n-grid-item>
      </n-grid>
      
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
          <n-text type="warning">
            此操作將會覆蓋現有的伺服器檔案，且無法復原。
          </n-text>
          <n-text depth="3" style="display: block; margin-top: 8px;">
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
            <n-button @click="showRecoverModal = false" :disabled="isRecovering">
              取消
            </n-button>
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
      
      <!-- 設定檔 Tab -->
      <n-card class="properties-section fade-in-up" style="animation-delay: 0.3s;" v-if="activeTab === 'properties'">
        <template #header>
          <n-space justify="space-between" align="center">
            <n-text strong>SERVER PROPERTIES</n-text>
            <n-button 
              type="primary" 
              size="small"
              :loading="propertiesSaving"
              @click="saveProperties"
            >
              <template #icon><n-icon><SaveOutlined /></n-icon></template>
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

.terminal-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
  overflow: hidden;
}

.terminal-header {
  width: 100%;
}

.console-title {
  font-family: 'Fira Code', monospace;
  letter-spacing: 1px;
}

.status-tag {
  font-family: 'Fira Code', monospace;
  font-size: 10px;
  letter-spacing: 1px;
}

/* 狀態指示燈 */
.status-indicator {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background-color: #444;
  position: relative;
}

.status-indicator .status-dot {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background-color: #ef4444;
  box-shadow: 0 0 6px rgba(239, 68, 68, 0.5);
}

.status-indicator.status-online .status-dot {
  background-color: #18a058;
  box-shadow: 0 0 8px rgba(24, 160, 88, 0.6);
  animation: pulse 2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.7;
    transform: scale(0.9);
  }
}

.terminal-wrapper {
  background-color: #0c0c0e;
  padding: 12px;
  border-radius: 6px;
  margin-bottom: 12px;
  border: 1px solid #222;
  transition: all 0.3s ease;
}

.terminal-wrapper.terminal-glow {
  border-color: rgba(24, 160, 88, 0.3);
  box-shadow: 0 0 20px rgba(24, 160, 88, 0.1), inset 0 0 30px rgba(24, 160, 88, 0.02);
}

.terminal-container {
  height: 450px;
}

/* 命令輸入區 */
.command-input-area {
  display: flex;
  gap: 8px;
  align-items: stretch;
}

.command-input {
  flex: 1;
}

.command-input :deep(.n-input) {
  background-color: rgba(0, 0, 0, 0.4);
  border-radius: 4px;
  font-family: 'Fira Code', monospace;
}

.command-input :deep(.n-input__input-el) {
  font-family: 'Fira Code', monospace;
}

.prompt-symbol {
  font-family: 'Fira Code', monospace;
  font-weight: bold;
  margin-right: 4px;
}

/* 輸入框包裝器 (用於下拉選單定位) */
.input-wrapper {
  flex: 1;
  position: relative;
}

/* 指令建議下拉選單 */
.suggestions-dropdown {
  position: absolute;
  bottom: 100%;
  left: 0;
  right: 0;
  background: #1a1a20;
  border: 1px solid #333;
  border-radius: 4px;
  margin-bottom: 4px;
  max-height: 200px;
  overflow-y: auto;
  z-index: 100;
  box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.3);
}

.suggestion-item {
  padding: 8px 12px;
  cursor: pointer;
  font-family: 'Fira Code', monospace;
  font-size: 13px;
  color: #d4d4d4;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  transition: all 0.15s ease;
}

.suggestion-item:last-child {
  border-bottom: none;
}

.suggestion-item:hover {
  background: rgba(24, 160, 88, 0.15);
  color: #18a058;
}

.suggestion-item .cmd-name {
  color: #18a058;
  font-weight: 500;
}

.send-btn {
  padding: 0 16px;
}

@media (max-width: 768px) {
  .terminal-container {
    height: 300px;
  }
  
  .send-btn {
    padding: 0 12px;
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

.info-card, .backup-card {
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

.backup-list {
  max-height: 300px;
  overflow-y: auto;
}

.backup-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.backup-name {
  font-size: 12px;
  font-family: 'Fira Code', monospace;
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.empty-backups {
  text-align: center;
  padding: 20px;
  color: #666;
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

/* 還原確認 Modal */
.recover-modal {
  max-width: 400px;
}

.recover-warning {
  background: rgba(255, 171, 0, 0.1);
  border: 1px solid rgba(255, 171, 0, 0.3);
  border-radius: 4px;
  padding: 16px;
  margin-bottom: 16px;
}

.confirm-input {
  margin-top: 8px;
}

.confirm-input :deep(.n-input__input-el) {
  font-family: 'Fira Code', monospace;
  text-align: center;
  letter-spacing: 2px;
}
</style>
