<script setup>
import { ref, onMounted } from 'vue';
import {
  NCard,
  NButton,
  NSpace,
  NText,
  NIcon,
  NInput,
  NModal,
  NEmpty,
  NPopconfirm,
  useMessage,
} from 'naive-ui';
import {
  PlusOutlined,
  PlayCircleOutlined,
  EditOutlined,
  DeleteOutlined,
  ThunderboltOutlined,
} from '@vicons/antd';
import api from '../api';
import { useActivityLogStore } from '../store/activityLog';

const props = defineProps({
  serverId: {
    type: String,
    required: true,
  },
  terminal: {
    type: Object,
    default: null,
  },
  disabled: {
    type: Boolean,
    default: false,
  },
});

const message = useMessage();
const activityLog = useActivityLogStore();
const STORAGE_KEY = 'mc_macros';

// 狀態
const macros = ref([]);
const showEditor = ref(false);
const editingMacro = ref(null);
const macroName = ref('');
const macroCommands = ref('');
const isExecuting = ref(false);
const executingMacroId = ref(null);

// 從 LocalStorage 載入巨集
const loadMacros = () => {
  try {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (stored) {
      macros.value = JSON.parse(stored);
    } else {
      // 預設範例巨集
      macros.value = [
        {
          id: 'default-restart',
          name: 'Restart Countdown',
          commands: `say Server will restart in 10s!
WAIT 5
say Server will restart in 5s!
WAIT 5
save-all
stop`,
        },
      ];
      saveMacros();
    }
  } catch (err) {
    console.error('Failed to load macros:', err);
    macros.value = [];
  }
};

// 儲存巨集到 LocalStorage
const saveMacros = () => {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(macros.value));
  } catch (err) {
    console.error('Failed to save macros:', err);
  }
};

// 開啟新增/編輯 Modal
const openEditor = (macro = null) => {
  if (macro) {
    editingMacro.value = macro;
    macroName.value = macro.name;
    macroCommands.value = macro.commands;
  } else {
    editingMacro.value = null;
    macroName.value = '';
    macroCommands.value = '';
  }
  showEditor.value = true;
};

// 儲存巨集
const saveMacro = () => {
  if (!macroName.value.trim()) {
    message.warning('請輸入巨集名稱');
    return;
  }
  if (!macroCommands.value.trim()) {
    message.warning('請輸入指令序列');
    return;
  }

  if (editingMacro.value) {
    // 編輯現有巨集
    const idx = macros.value.findIndex(m => m.id === editingMacro.value.id);
    if (idx !== -1) {
      macros.value[idx] = {
        ...macros.value[idx],
        name: macroName.value.trim(),
        commands: macroCommands.value,
      };
    }
  } else {
    // 新增巨集
    macros.value.push({
      id: `macro-${Date.now()}`,
      name: macroName.value.trim(),
      commands: macroCommands.value,
    });
  }

  saveMacros();
  showEditor.value = false;
  activityLog.logMacroSave(macroName.value.trim());
  message.success('巨集已儲存');
};

// 刪除巨集
const deleteMacro = macroId => {
  const macro = macros.value.find(m => m.id === macroId);
  macros.value = macros.value.filter(m => m.id !== macroId);
  saveMacros();
  if (macro) activityLog.logMacroDelete(macro.name);
  message.success('巨集已刪除');
};

// 休眠函式
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));

const emit = defineEmits(['update:status']);

// 執行巨集
const executeMacro = async macro => {
  if (isExecuting.value || props.disabled) return;

  isExecuting.value = true;
  executingMacroId.value = macro.id;

  const term = props.terminal;
  const lines = macro.commands.split('\n').filter(l => l.trim());

  term?.writeln(`\x1b[1;35m▶ [MACRO] 開始執行: ${macro.name}\x1b[0m`);
  emit('update:status', { active: true, message: `開始執行: ${macro.name}`, type: 'info' });

  try {
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i].trim();

      // 檢查 WAIT 指令
      if (line.toUpperCase().startsWith('WAIT ')) {
        const seconds = parseInt(line.substring(5));
        if (!isNaN(seconds) && seconds > 0) {
          term?.writeln(`\x1b[1;33m⏳ [MACRO] 等待 ${seconds} 秒...\x1b[0m`);
          emit('update:status', { active: true, message: `等待 ${seconds} 秒...`, type: 'warning' });
          await sleep(seconds * 1000);
        }
      } else if (line) {
        // 執行指令
        term?.writeln(`\x1b[1;36m▸ [MACRO] ${line}\x1b[0m`);
        emit('update:status', { active: true, message: `執行: ${line}`, type: 'info' });
        try {
          await api.post(`/mc-api/a/cmd/${props.serverId}`, { command: line });
        } catch (err) {
          term?.writeln(`\x1b[1;31m✗ [MACRO] 指令執行失敗: ${line}\x1b[0m`);
        }
        // 指令間隔
        await sleep(300);
      }
    }

    term?.writeln(`\x1b[1;32m✓ [MACRO] 執行完成: ${macro.name}\x1b[0m`);
    activityLog.logMacroExecute(macro.name, props.serverId, props.serverId);
    message.success('巨集執行完成');
  } catch (err) {
    term?.writeln(`\x1b[1;31m✗ [MACRO] 執行中斷\x1b[0m`);
    message.error('巨集執行中斷');
  } finally {
    isExecuting.value = false;
    executingMacroId.value = null;
    emit('update:status', null); // Clear status
  }
};

onMounted(() => {
  loadMacros();
});
</script>

<template>
  <n-card class="macro-panel" size="small">
    <template #header>
      <n-space align="center" :size="8">
        <n-icon color="#f59e0b" size="16"><ThunderboltOutlined /></n-icon>
        <n-text strong class="panel-title">MACROS</n-text>
      </n-space>
    </template>

    <template #header-extra>
      <n-button size="tiny" quaternary :disabled="isExecuting" @click="openEditor()">
        <template #icon
          ><n-icon><PlusOutlined /></n-icon
        ></template>
      </n-button>
    </template>

    <div class="macro-list">
      <n-empty v-if="macros.length === 0" description="尚無巨集" size="small" />

      <div
        v-for="macro in macros"
        :key="macro.id"
        class="macro-item"
        :class="{ 'macro-executing': executingMacroId === macro.id }"
      >
        <n-text class="macro-name">{{ macro.name }}</n-text>
        <n-space :size="4">
          <n-button
            size="tiny"
            type="success"
            quaternary
            :loading="executingMacroId === macro.id"
            :disabled="disabled || (isExecuting && executingMacroId !== macro.id)"
            title="執行巨集"
            @click="executeMacro(macro)"
          >
            <template #icon
              ><n-icon><PlayCircleOutlined /></n-icon
            ></template>
          </n-button>
          <n-button
            size="tiny"
            quaternary
            :disabled="isExecuting"
            title="編輯"
            @click="openEditor(macro)"
          >
            <template #icon
              ><n-icon><EditOutlined /></n-icon
            ></template>
          </n-button>
          <n-popconfirm @positive-click="deleteMacro(macro.id)">
            <template #trigger>
              <n-button size="tiny" type="error" quaternary :disabled="isExecuting" title="刪除">
                <template #icon
                  ><n-icon><DeleteOutlined /></n-icon
                ></template>
              </n-button>
            </template>
            確定要刪除此巨集嗎？
          </n-popconfirm>
        </n-space>
      </div>
    </div>

    <!-- 編輯 Modal -->
    <n-modal
      v-model:show="showEditor"
      preset="card"
      :title="editingMacro ? '編輯巨集' : '新增巨集'"
      class="macro-editor-modal"
      :closable="!isExecuting"
      :mask-closable="!isExecuting"
    >
      <n-space vertical :size="16">
        <div>
          <n-text depth="3" style="font-size: 12px; margin-bottom: 4px; display: block"
            >巨集名稱</n-text
          >
          <n-input v-model:value="macroName" placeholder="例如：Restart Countdown" />
        </div>

        <div>
          <n-text depth="3" style="font-size: 12px; margin-bottom: 4px; display: block">
            指令序列 (每行一條指令，使用 WAIT n 來等待 n 秒)
          </n-text>
          <n-input
            v-model:value="macroCommands"
            type="textarea"
            :rows="8"
            placeholder="say Hello World
WAIT 3
say Goodbye"
            class="commands-input"
          />
        </div>
      </n-space>

      <template #footer>
        <n-space justify="end">
          <n-button @click="showEditor = false">取消</n-button>
          <n-button type="primary" @click="saveMacro">儲存</n-button>
        </n-space>
      </template>
    </n-modal>
  </n-card>
</template>

<style scoped>
.macro-panel {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
}

.panel-title {
  font-family: 'Fira Code', monospace;
  font-size: 12px;
  letter-spacing: 1px;
}

.macro-list {
  max-height: 200px;
  overflow-y: auto;
}

.macro-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.2);
  margin-bottom: 4px;
  transition: all 0.2s ease;
}

.macro-item:hover {
  background: rgba(0, 0, 0, 0.3);
}

.macro-item.macro-executing {
  background: rgba(245, 158, 11, 0.15);
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.macro-name {
  font-family: 'Fira Code', monospace;
  font-size: 12px;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 8px;
}

.macro-editor-modal {
  max-width: 500px;
}

.commands-input :deep(.n-input__textarea-el) {
  font-family: 'Fira Code', monospace !important;
  font-size: 13px;
  line-height: 1.6;
}
</style>
