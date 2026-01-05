<template>
  <div class="server-console">
    <!-- Terminal 容器 -->
    <div ref="terminalContainer" class="terminal-container"></div>

    <!-- 狀態列與輸入框 -->
    <div class="console-footer">
      <div class="status-bar">
        <n-space justify="space-between" align="center">
          <n-badge dot :type="isConnected ? 'success' : 'error'">
            <n-text class="status-text">
              {{ isConnected ? 'CONNECTED' : 'DISCONNECTED' }}
            </n-text>
          </n-badge>
          <n-text depth="3" class="console-label">MINECRAFT CONSOLE</n-text>
        </n-space>
      </div>

      <div class="input-area">
        <n-input-group>
          <n-input-group-label class="prompt-label">/</n-input-group-label>
          <n-input
            v-model:value="inputCommand"
            type="text"
            placeholder="Type a command..."
            :disabled="!isConnected || processing"
            @keydown="handleKeydown"
            class="console-input"
          />
          <n-button 
            type="primary" 
            ghost 
            :disabled="!isConnected || processing || !inputCommand.trim()"
            @click="sendCommand"
          >
            SEND
          </n-button>
        </n-input-group>
      </div>
      
      <!-- 常用指令快捷鍵 (精簡版) -->
      <div class="quick-commands">
        <n-space size="small" align="center">
          <n-button size="tiny" type="primary" secondary @click="showCommandModal = true">
            Command Library
          </n-button>
          <div style="width: 1px; height: 16px; background: #444; margin: 0 4px;"></div>
          <!-- Keep only high frequency commands -->
          <n-button size="tiny" secondary @click="setInput('time set day')">Day</n-button>
          <n-button size="tiny" secondary @click="setInput('weather clear')">Clear Weather</n-button>
          <n-button size="tiny" secondary @click="setInput('list')">List</n-button>
          <n-button size="tiny" secondary @click="setInput('stop')">Stop</n-button>
          <n-button size="tiny" secondary @click="handleClearTerminal">Clear Console</n-button>
        </n-space>
      </div>
    </div>

    <!-- Command Library Modal -->
    <n-modal
      v-model:show="showCommandModal"
      preset="card"
      title="Command Library"
      style="width: 600px; max-width: 90vw;"
      :bordered="false"
    >
      <template #header-extra>
        <n-button size="small" secondary type="primary" tag="a" href="https://www.digminecraft.com/game_commands/index.php" target="_blank">
          🌐 Online Wiki
        </n-button>
      </template>

      <n-tabs type="line" animated>
        <n-tab-pane name="server" tab="Server">
          <n-space vertical>
            <n-text depth="3">Management</n-text>
            <n-space>
              <n-button size="small" @click="setInputAndClose('list')">List Players</n-button>
              <n-button size="small" @click="setInputAndClose('stop')">Stop Server</n-button>
              <n-button size="small" @click="setInputAndClose('save-all')">Save All</n-button>
              <n-button size="small" @click="setInputAndClose('save-off')">Disable Auto-Save</n-button>
              <n-button size="small" @click="setInputAndClose('save-on')">Enable Auto-Save</n-button>
            </n-space>
            <n-text depth="3">Whitelist</n-text>
            <n-space>
              <n-button size="small" @click="setInputAndClose('whitelist on')">Whitelist On</n-button>
              <n-button size="small" @click="setInputAndClose('whitelist off')">Whitelist Off</n-button>
              <n-button size="small" @click="setInputAndClose('whitelist list')">List Whitelist</n-button>
              <n-button size="small" @click="setInput('whitelist add ')">Add User...</n-button>
              <n-button size="small" @click="setInput('whitelist remove ')">Remove User...</n-button>
            </n-space>
          </n-space>
        </n-tab-pane>
        
        <n-tab-pane name="player" tab="Player">
          <n-space vertical>
            <n-text depth="3">Moderation</n-text>
            <n-space>
              <n-button size="small" @click="setInput('kick ')">Kick...</n-button>
              <n-button size="small" @click="setInput('ban ')">Ban...</n-button>
              <n-button size="small" @click="setInput('pardon ')">Unban (Pardon)...</n-button>
              <n-button size="small" @click="setInput('op ')">OP...</n-button>
              <n-button size="small" @click="setInput('deop ')">De-OP...</n-button>
            </n-space>
            <n-text depth="3">Gamemode</n-text>
            <n-space>
              <n-button size="small" @click="setInput('gamemode survival ')">Survival...</n-button>
              <n-button size="small" @click="setInput('gamemode creative ')">Creative...</n-button>
              <n-button size="small" @click="setInput('gamemode spectator ')">Spectator...</n-button>
              <n-button size="small" @click="setInput('gamemode adventure ')">Adventure...</n-button>
            </n-space>
            <n-text depth="3">Other</n-text>
            <n-space>
              <n-button size="small" @click="setInput('xp add ')">Give XP...</n-button>
              <n-button size="small" @click="setInput('tp ')">Teleport...</n-button>
              <n-button size="small" @click="setInput('clear ')">Clear Inventory...</n-button>
            </n-space>
          </n-space>
        </n-tab-pane>

        <n-tab-pane name="world" tab="World">
          <n-space vertical>
            <n-text depth="3">Time</n-text>
            <n-space>
              <n-button size="small" @click="setInputAndClose('time set day')">Day</n-button>
              <n-button size="small" @click="setInputAndClose('time set night')">Night</n-button>
              <n-button size="small" @click="setInputAndClose('time set noon')">Noon</n-button>
              <n-button size="small" @click="setInputAndClose('time set midnight')">Midnight</n-button>
              <n-button size="small" @click="setInput('time add ')">Add Time...</n-button>
            </n-space>
            <n-text depth="3">Weather</n-text>
            <n-space>
              <n-button size="small" @click="setInputAndClose('weather clear')">Clear</n-button>
              <n-button size="small" @click="setInputAndClose('weather rain')">Rain</n-button>
              <n-button size="small" @click="setInputAndClose('weather thunder')">Thunder</n-button>
            </n-space>
             <n-text depth="3">Difficulty</n-text>
            <n-space>
              <n-button size="small" @click="setInputAndClose('difficulty peaceful')">Peaceful</n-button>
              <n-button size="small" @click="setInputAndClose('difficulty easy')">Easy</n-button>
              <n-button size="small" @click="setInputAndClose('difficulty normal')">Normal</n-button>
              <n-button size="small" @click="setInputAndClose('difficulty hard')">Hard</n-button>
            </n-space>
          </n-space>
        </n-tab-pane>

        <n-tab-pane name="gamerules" tab="GameRules">
          <n-space vertical>
            <n-text depth="3">Common Rules</n-text>
            <n-space>
              <n-button size="small" @click="setInputAndClose('gamerule keepInventory true')">KeepInventory: True</n-button>
              <n-button size="small" @click="setInputAndClose('gamerule keepInventory false')">KeepInventory: False</n-button>
              <n-button size="small" @click="setInputAndClose('gamerule doMobSpawning true')">MobSpawning: True</n-button>
              <n-button size="small" @click="setInputAndClose('gamerule doMobSpawning false')">MobSpawning: False</n-button>
              <n-button size="small" @click="setInputAndClose('gamerule doDaylightCycle true')">DaylightCycle: True</n-button>
              <n-button size="small" @click="setInputAndClose('gamerule doDaylightCycle false')">DaylightCycle: False</n-button>
            </n-space>
             <n-text depth="3">Chat</n-text>
            <n-space>
               <n-button size="small" @click="setInput('say ')">Broadcast (Say)...</n-button>
               <n-button size="small" @click="setInput('title @a title ')">Show Title...</n-button>
            </n-space>
          </n-space>
        </n-tab-pane>
      </n-tabs>
    </n-modal>
  </div>
</template>

<script setup>
// ... (previous imports)
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue';
import { 
  NInput, NButton, NSpace, NText, NBadge, NInputGroup, 
  NInputGroupLabel, useMessage, NModal, NTabs, NTabPane 
} from 'naive-ui';
<<<<<<< HEAD
=======
// ... (rest of code)
>>>>>>> 6b16455ec9382b83ef1210bfcc64bdb1ab160a2b
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import api from '../api';

const props = defineProps({
  serverId: {
    type: String,
    required: true
  },
  isRunning: {
    type: Boolean,
    default: false
  },
  macroStatus: {
    type: String,
    default: null
  }
});

const emit = defineEmits(['command-sent']);
const message = useMessage();

// Terminal 實例
const terminalContainer = ref(null);
const term = ref(null);
const fitAddon = ref(null);

// 狀態
const inputCommand = ref('');
const isConnected = ref(false); // 這裡的連接狀態主要反映伺服器是否運行
const processing = ref(false);
const showCommandModal = ref(false);

// 指令歷史
const commandHistory = ref([]);
const historyIndex = ref(-1);
const HISTORY_KEY = `mc_cmd_history_${props.serverId}`;

// 初始化 Terminal
const initTerminal = () => {
  if (!terminalContainer.value) return; // 防呆

  term.value = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Fira Code, monospace',
    theme: {
      background: '#1a1a20',
      foreground: '#d4d4d4',
      cursor: '#aeafad',
      selectionBackground: '#264f78',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79', // Minecraft Green
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5'
    },
    disableStdin: true, // 只讀，輸入通過 Input 元件
    convertEol: true,
    rows: 20
  });

  fitAddon.value = new FitAddon();
  term.value.loadAddon(fitAddon.value);
  
  term.value.open(terminalContainer.value);
  
  // 延遲執行 fit 以確保 DOM 渲染完成
  setTimeout(() => {
    if (fitAddon.value) fitAddon.value.fit();
  }, 100);

  // 歡迎訊息
  term.value.writeln('\x1b[1;32mWelcome to Minecraft Server Console\x1b[0m');
  term.value.writeln('Type \x1b[1;33m/help\x1b[0m for list of commands.');
  
  // 監聽視窗大小變化
  window.addEventListener('resize', handleResize);
};

const handleResize = () => {
  if (fitAddon.value && term.value && terminalContainer.value) {
    // 檢查是否隱藏
    if (terminalContainer.value.offsetParent !== null) {
      fitAddon.value.fit();
    }
  }
};

// 載入歷史紀錄
const loadHistory = () => {
  try {
    const saved = localStorage.getItem(HISTORY_KEY);
    if (saved) {
      commandHistory.value = JSON.parse(saved);
    }
  } catch (e) {
    console.error('Failed to load command history', e);
  }
};

// 儲存歷史紀錄
const saveHistory = () => {
  try {
    // 只保留最近 50 筆
    const toSave = commandHistory.value.slice(0, 50);
    localStorage.setItem(HISTORY_KEY, JSON.stringify(toSave));
  } catch (e) {
    console.error('Failed to save command history', e);
  }
};

// 導航歷史紀錄
const navigateHistory = (direction) => {
  if (commandHistory.value.length === 0) return;

  if (direction === -1) { // Up
    if (historyIndex.value < commandHistory.value.length - 1) {
      historyIndex.value++;
      inputCommand.value = commandHistory.value[historyIndex.value];
    }
  } else { // Down
    if (historyIndex.value > 0) {
      historyIndex.value--;
      inputCommand.value = commandHistory.value[historyIndex.value];
    } else if (historyIndex.value === 0) {
      historyIndex.value = -1;
      inputCommand.value = '';
    }
  }
};

// 寫入日誌 (由父元件呼叫)
const writeLog = (logData) => {
  if (!term.value) return;
  term.value.write(logData);
};

// 設定輸入框內容 (快捷鍵用)
const setInput = (cmd) => {
  inputCommand.value = cmd;
  // 自動聚焦輸入框
  // TODO: 需要 ref 到 input 元素才能 focus，這裡簡化處理
};

const setInputAndClose = (cmd) => {
  setInput(cmd);
  showCommandModal.value = false;
};

// 處理鍵盤事件 (合併 handler 以避免 Vue 警告)
const handleKeydown = (e) => {
  if (e.key === 'Enter') {
    e.preventDefault();
    sendCommand();
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    navigateHistory(-1);
  } else if (e.key === 'ArrowDown') {
    e.preventDefault();
    navigateHistory(1);
  }
};

// 清除終端機 (前端操作)
const handleClearTerminal = () => {
  if (term.value) {
    term.value.clear();
    term.value.writeln('\x1b[3mConsole cleared locally.\x1b[0m');
  }
};

// 發送指令
const sendCommand = async () => {
  const cmd = inputCommand.value.trim();
  if (!cmd) return;

  // 處理本地指令
  if (cmd === '/clear' || cmd === 'clear') {
    handleClearTerminal();
    inputCommand.value = '';
    return;
  }

  // 加入歷史紀錄 (避免重複且移至最前)
  const existingIndex = commandHistory.value.indexOf(cmd);
  if (existingIndex !== -1) {
    commandHistory.value.splice(existingIndex, 1);
  }
  commandHistory.value.unshift(cmd);
  historyIndex.value = -1;
  saveHistory();

  processing.value = true;
  // 在 Console 顯示發送的指令 (類似 echo)
  term.value.writeln(`\r\n> \x1b[1m${cmd}\x1b[0m`);

  try {
    await api.post(`/mc-api/a/cmd/${props.serverId}`, {
      command: cmd
    });
    // 成功不需特別提示，日誌會顯示結果
    emit('command-sent', cmd);
  } catch (err) {
    const errorMsg = err.response?.data?.error || err.message;
    term.value.writeln(`\x1b[1;31mError sending command: ${errorMsg}\x1b[0m`);
    message.error('指令發送失敗');
  } finally {
    inputCommand.value = '';
    processing.value = false;
  }
};

// 監聽與生命週期
watch(() => props.isRunning, (val) => {
  isConnected.value = val;
  if (!val && term.value) {
    term.value.writeln('\x1b[3mServer stopped. Console disconnected.\x1b[0m');
  } else if (val && term.value) {
    term.value.writeln('\x1b[3mServer running. Console connected.\x1b[0m');
  }
}, { immediate: true });

onMounted(() => {
  nextTick(() => {
    initTerminal();
    loadHistory();
  });
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', handleResize);
<<<<<<< HEAD
  // 正確清理順序：先清理 addon，再清理 terminal
  try {
    if (fitAddon.value) {
      fitAddon.value.dispose();
      fitAddon.value = null;
    }
  } catch (e) {
    // 忽略 addon dispose 錯誤
  }
  try {
    if (term.value) {
      term.value.dispose();
      term.value = null;
    }
  } catch (e) {
    // 忽略 terminal dispose 錯誤
=======
  if (term.value) {
    term.value.dispose();
>>>>>>> 6b16455ec9382b83ef1210bfcc64bdb1ab160a2b
  }
});

// 公開方法供父元件使用
defineExpose({
  term,
  writeLog
});
</script>

<style scoped>
.server-console {
  display: flex;
  flex-direction: column;
  height: 500px;
  background: #1a1a20;
  border: 1px solid #333;
  border-radius: 6px;
  overflow: hidden;
}

.terminal-container {
  flex: 1;
  padding: 8px;
  background: #000;
  overflow: hidden; 
  /* xterm container should manage scroll internally */
}

.console-footer {
  padding: 12px;
  background: #25252b;
  border-top: 1px solid #333;
}

.status-bar {
  margin-bottom: 8px;
}

.status-text {
  font-size: 11px;
  font-weight: bold;
  margin-left: 4px;
}

.console-label {
  font-family: 'Fira Code', monospace;
  font-size: 11px;
  letter-spacing: 1px;
}

.input-area {
  margin-bottom: 8px;
}

.prompt-label {
  background: #333;
  color: #fff;
  border: none;
  font-family: 'Fira Code', monospace;
}

.console-input {
  font-family: 'Fira Code', monospace;
}

.quick-commands {
  padding-top: 4px;
}
</style>
