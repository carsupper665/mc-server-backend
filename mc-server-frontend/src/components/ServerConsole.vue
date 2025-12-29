<script setup>
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue';
import {
  NSpace,
  NButton,
  NCard,
  NTag,
  NInput,
  NText,
  NIcon,
  useMessage,
} from 'naive-ui';
import {
  CodeOutlined,
  SendOutlined,
  ClearOutlined,
} from '@vicons/antd';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import api from '../api';

const props = defineProps({
  serverId: {
    type: String,
    required: true,
  },
  isRunning: {
    type: Boolean,
    default: false,
  },
  macroStatus: {
    type: Object,
    default: null,
  },
});

const message = useMessage();

// 終端機相關狀態
const terminalRef = ref(null);
let term = null;
let fitAddon = null;

// 指令相關狀態
const command = ref('');
const commandHistory = ref([]);
const historyIndex = ref(-1);
const showSuggestions = ref(false);
const filteredCommands = ref([]);
const selectedSuggestionIndex = ref(0);

const COMMAND_HISTORY_KEY = `mc_cmd_history_`;

// Minecraft 常用指令列表
const mcCommands = [
  '/help', '/list', '/say', '/tell', '/msg', '/me', '/gamemode', '/tp',
  '/teleport', '/give', '/clear', '/time', '/weather', '/difficulty',
  '/seed', '/ban', '/pardon', '/kick', '/op', '/deop', '/whitelist',
  '/stop', '/save-all', '/save-on', '/save-off', '/reload', '/setblock',
  '/fill', '/clone', '/summon', '/kill', '/effect', '/enchant',
  '/experience', '/xp', '/gamerule', '/worldborder', '/spawnpoint',
  '/setworldspawn', '/scoreboard', '/team', '/title', '/bossbar',
  '/execute', '/function', '/schedule', '/data', '/datapack',
];

// 初始化終端機
const initTerminal = () => {
  if (term) return; // 防止重複初始化

  term = new Terminal({
    theme: {
      background: '#0c0c0e',
      foreground: '#d4d4d4',
      cursor: '#18a058',
    },
    fontFamily: 'Fira Code, monospace',
    fontSize: 13,
    convertEol: true,
    cursorBlink: true,
    disableStdin: true,
  });

  fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.open(terminalRef.value);
  fitAddon.fit();

  term.writeln('\x1b[1;32m[ SYSTEM ] Terminal initialized. Waiting for logs...\x1b[0m');
};

// 公開方法：寫入日誌
const writeLog = (logData) => {
  if (term && logData) {
    term.clear();
    term.write(logData);
  }
};

// 公開方法：清除終端機
const clearTerminal = () => {
  if (term) {
    term.clear();
    term.writeln('\x1b[1;33m[ SYSTEM ] Terminal cleared.\x1b[0m');
  }
};

// 載入與儲存歷史紀錄
const loadCommandHistory = () => {
  try {
    const stored = localStorage.getItem(COMMAND_HISTORY_KEY + props.serverId);
    if (stored) {
      commandHistory.value = JSON.parse(stored);
    }
  } catch (err) {
    console.error('Failed to load command history:', err);
  }
};

const saveCommandHistory = () => {
  try {
    localStorage.setItem(
      COMMAND_HISTORY_KEY + props.serverId,
      JSON.stringify(commandHistory.value.slice(0, 50))
    );
  } catch (err) {
    console.error('Failed to save command history:', err);
  }
};

// 指令處理
const sendCommand = async () => {
  if (!command.value) return;
  try {
    // 儲存指令歷史
    commandHistory.value.unshift(command.value);
    if (commandHistory.value.length > 50) {
      commandHistory.value.pop();
    }
    historyIndex.value = -1;
    saveCommandHistory();

    const cmd = command.value;
    command.value = '';

    const highlightedCmd = highlightCommand(cmd);
    term.writeln(`\x1b[1;36m❯\x1b[0m ${highlightedCmd}`);

    await api.post(`/mc-api/a/cmd/${props.serverId}`, { command: cmd });

    // 觸發父元件刷新日誌? 
    // 不需要，父元件有定時輪詢。如果需要立即反饋，父元件可以監聽按鈕點擊? 
    // 為了保持簡單，我們這裡不做額外的 emit，依賴輪詢。
    // 或許可以 emit('command-sent') 來讓父元件立即 fetchLogs?
    // 但原代碼是在這裡 setTimeout fetchLogs。
    // 由於 fetchLogs 在父元件，我們可以 emit 一個事件。
    emit('command-sent');
    
  } catch (err) {
    term.writeln(`\x1b[1;31m[ ERROR ] 指令發送失敗\x1b[0m`);
    message.error('指令發送失敗');
  }
};

const highlightCommand = cmd => {
  if (cmd.startsWith('/')) {
    const parts = cmd.split(' ');
    const cmdName = `\x1b[1;32m${parts[0]}\x1b[0m`;
    const args = parts.slice(1).map(arg => {
      if (/^\d+$/.test(arg)) return `\x1b[1;33m${arg}\x1b[0m`;
      if (arg.startsWith('@')) return `\x1b[1;35m${arg}\x1b[0m`;
      return `\x1b[1;37m${arg}\x1b[0m`;
    });
    return [cmdName, ...args].join(' ');
  }
  return `\x1b[0;37m${cmd}\x1b[0m`;
};

// 監聽指令輸入以過濾建議
watch(command, newValue => {
  if (!newValue || newValue.trim() === '') {
    showSuggestions.value = false;
    filteredCommands.value = [];
    return;
  }

  if (newValue.startsWith('/')) {
    const input = newValue.toLowerCase();
    const matches = mcCommands.filter(cmd => cmd.startsWith(input));
    filteredCommands.value = matches;
    const exactMatch = matches.length === 1 && matches[0] === input;
    showSuggestions.value = matches.length > 0 && !exactMatch;
    selectedSuggestionIndex.value = 0;
  } else {
    showSuggestions.value = false;
  }
});

const selectCommand = cmd => {
  command.value = cmd + ' ';
  showSuggestions.value = false;
};

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

const handleCommandKeydown = e => {
  if (showSuggestions.value && filteredCommands.value.length > 0) {
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedSuggestionIndex.value =
        (selectedSuggestionIndex.value - 1 + filteredCommands.value.length) % filteredCommands.value.length;
      return;
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedSuggestionIndex.value =
        (selectedSuggestionIndex.value + 1) % filteredCommands.value.length;
      return;
    } else if (e.key === 'Tab' || e.key === 'Enter') {
      e.preventDefault();
      if (selectedSuggestionIndex.value >= 0 && selectedSuggestionIndex.value < filteredCommands.value.length) {
        selectCommand(filteredCommands.value[selectedSuggestionIndex.value]);
      }
      return;
    } else if (e.key === 'Escape') {
      e.preventDefault();
      showSuggestions.value = false;
      return;
    }
  }

  if (e.key === 'Tab') {
    e.preventDefault();
    handleTabComplete();
  } else if (e.key === 'Enter') {
    e.preventDefault();
    sendCommand();
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    if (historyIndex.value < commandHistory.value.length - 1) {
      historyIndex.value++;
      command.value = commandHistory.value[historyIndex.value];
      setTimeout(() => {
        const input = document.querySelector('.command-input input');
        if (input) input.selectionStart = input.selectionEnd = command.value.length;
      }, 0);
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

// 監聽 Macro 狀態變更，重新調整終端機尺寸
watch(
  () => props.macroStatus?.active,
  () => {
    nextTick(() => {
      fitAddon?.fit();
    });
  }
);

onMounted(() => {
  initTerminal();
  loadCommandHistory();
  window.addEventListener('resize', () => fitAddon?.fit());
});

onBeforeUnmount(() => {
  term?.dispose();
  window.removeEventListener('resize', () => fitAddon?.fit());
});

// Expose term for MacroPanel and methods for parent
defineExpose({
  term, // Exposing the term variable directly might be null initially if setup logic delays? No, onMounted sets it.
  // Actually, Vue 3 defineExpose with script setup exposes ref values.
  // We need to return the term instance. However, term is a let variable, not a ref.
  // We should make term a shallowRef or wrapper to expose it properly?
  // Or just a method to get it?
  getTerm: () => term,
  writeLog,
  clearTerminal
});

const emit = defineEmits(['command-sent']);
</script>

<template>
  <n-card class="terminal-card fade-in-up" :bordered="false">
    <template #header>
      <n-space align="center" justify="space-between" class="terminal-header">
        <n-space align="center" :size="12">
          <div class="status-indicator" :class="{ 'status-online': isRunning }">
            <div class="status-dot"></div>
          </div>
          <n-icon color="#18a058" size="18"><CodeOutlined /></n-icon>
          <n-text strong class="console-title">LIVE CONSOLE</n-text>
          <n-tag :type="isRunning ? 'success' : 'error'" size="small" class="status-tag">
            {{ isRunning ? 'CONNECTED' : 'OFFLINE' }}
          </n-tag>
        </n-space>
        <n-button size="tiny" quaternary title="Clear Console" @click="clearTerminal">
          <template #icon
            ><n-icon><ClearOutlined /></n-icon
          ></template>
        </n-button>
      </n-space>
    </template>

    <!-- Macro Status Bar -->
    <div v-if="macroStatus?.active" class="macro-status-bar fade-in-up">
      <n-icon :component="macroStatus.type === 'warning' ? 'span' : 'div'" size="16">
        ⏳
      </n-icon>
      <span class="macro-message">{{ macroStatus.message }}</span>
    </div>

    <div class="terminal-wrapper" :class="{ 'terminal-glow': isRunning }">
      <div ref="terminalRef" class="terminal-container"></div>
    </div>

    <div class="command-input-area">
      <div class="input-wrapper">
        <n-input
          v-model:value="command"
          placeholder="輸入指令... (Tab 補全, ↑↓ 歷史)"
          :disabled="!isRunning"
          class="command-input"
          @keydown="handleCommandKeydown"
        >
          <template #prefix>
            <n-text :depth="isRunning ? 1 : 3" class="prompt-symbol">❯</n-text>
          </template>
        </n-input>
        <!-- 指令建議下拉選單 -->
        <div
          v-if="showSuggestions && filteredCommands.length > 0"
          class="suggestions-dropdown"
        >
          <div
            v-for="(cmd, index) in filteredCommands"
            :key="cmd"
            class="suggestion-item"
            :class="{ active: index === selectedSuggestionIndex }"
            @click="selectCommand(cmd)"
            @mouseenter="selectedSuggestionIndex = index"
          >
            <span class="cmd-name">{{ cmd }}</span>
          </div>
        </div>
      </div>
      <n-button
        type="primary"
        :disabled="!isRunning || !command"
        class="send-btn"
        @click="sendCommand"
      >
        <template #icon
          ><n-icon><SendOutlined /></n-icon
        ></template>
      </n-button>
    </div>
  </n-card>
</template>

<style scoped>
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
  0%,
  100% {
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
  margin-bottom: 24px;
  border: 1px solid #222;
  transition: all 0.3s ease;
  position: relative;
}

.terminal-wrapper.terminal-glow {
  border-color: rgba(24, 160, 88, 0.3);
  box-shadow:
    0 0 20px rgba(24, 160, 88, 0.1),
    inset 0 0 30px rgba(24, 160, 88, 0.02);
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

/* 輸入框包裝器 */
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

.suggestion-item:hover,
.suggestion-item.active {
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

.macro-status-bar {
  background: rgba(234, 179, 8, 0.1);
  border: 1px solid rgba(234, 179, 8, 0.2);
  color: #eab308;
  padding: 8px 12px;
  border-radius: 4px;
  margin-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-family: 'Fira Code', monospace;
  font-size: 12px;
}

.macro-status-bar .macro-message {
  font-weight: 500;
}

@media (max-width: 768px) {
  .terminal-container {
    height: 250px;
    font-size: 11px !important;
  }
  
  .command-input-area {
    flex-direction: column;
    gap: 8px;
  }

  .input-wrapper {
    width: 100%;
  }

  .send-btn {
    width: 100%;
    padding: 8px 16px;
  }
}
</style>
