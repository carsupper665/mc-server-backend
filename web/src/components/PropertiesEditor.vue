<template>
  <div class="properties-editor">
    <!-- 模式切換 -->
    <div class="editor-header">
      <n-space justify="space-between" align="center">
        <n-text depth="3" class="mode-label">
          {{ isAdvancedMode ? 'ADVANCED MODE' : 'VISUAL MODE' }}
        </n-text>
        <n-button 
          size="small" 
          quaternary 
          @click="toggleMode"
        >
          {{ isAdvancedMode ? '切換至視覺化模式' : '切換至進階模式' }}
        </n-button>
      </n-space>
    </div>

    <!-- 進階模式：純文字編輯 -->
    <template v-if="isAdvancedMode">
      <n-input
        v-model:value="rawContent"
        type="textarea"
        :rows="20"
        placeholder="server.properties 內容..."
        class="raw-editor"
        @update:value="emitUpdate"
      />
    </template>

    <!-- 視覺化模式 -->
    <template v-else>
      <div class="visual-editor">
        <!-- 遊戲模式 -->
        <div class="property-row">
          <div class="property-label">
            <n-text strong>遊戲模式</n-text>
            <n-text depth="3" class="property-key">gamemode</n-text>
          </div>
          <n-select
            v-model:value="parsedProps.gamemode"
            :options="gamemodeOptions"
            class="property-control"
            @update:value="updateProperty('gamemode', $event)"
          />
        </div>

        <!-- 難度 -->
        <div class="property-row">
          <div class="property-label">
            <n-text strong>難度</n-text>
            <n-text depth="3" class="property-key">difficulty</n-text>
          </div>
          <n-select
            v-model:value="parsedProps.difficulty"
            :options="difficultyOptions"
            class="property-control"
            @update:value="updateProperty('difficulty', $event)"
          />
        </div>

        <!-- PVP -->
        <div class="property-row">
          <div class="property-label">
            <n-text strong>玩家對戰 (PVP)</n-text>
            <n-text depth="3" class="property-key">pvp</n-text>
          </div>
          <n-switch
            :value="parsedProps.pvp === 'true'"
            @update:value="updateProperty('pvp', $event ? 'true' : 'false')"
          />
        </div>

        <!-- 白名單 -->
        <div class="property-row">
          <div class="property-label">
            <n-text strong>白名單</n-text>
            <n-text depth="3" class="property-key">white-list</n-text>
          </div>
          <n-switch
            :value="parsedProps['white-list'] === 'true'"
            @update:value="updateProperty('white-list', $event ? 'true' : 'false')"
          />
        </div>

        <!-- 線上模式 -->
        <div class="property-row">
          <div class="property-label">
            <n-text strong>正版驗證</n-text>
            <n-text depth="3" class="property-key">online-mode</n-text>
          </div>
          <n-switch
            :value="parsedProps['online-mode'] === 'true'"
            @update:value="updateProperty('online-mode', $event ? 'true' : 'false')"
          />
        </div>

        <!-- 最大玩家數 -->
        <div class="property-row">
          <div class="property-label">
            <n-text strong>最大玩家數</n-text>
            <n-text depth="3" class="property-key">max-players</n-text>
          </div>
          <n-input-number
            :value="parseInt(parsedProps['max-players']) || 20"
            :min="1"
            :max="1000"
            class="property-control number-input"
            @update:value="updateProperty('max-players', String($event))"
          />
        </div>

        <!-- 視野距離 -->
        <div class="property-row">
          <div class="property-label">
            <n-text strong>視野距離</n-text>
            <n-text depth="3" class="property-key">view-distance</n-text>
          </div>
          <n-input-number
            :value="parseInt(parsedProps['view-distance']) || 10"
            :min="2"
            :max="32"
            class="property-control number-input"
            @update:value="updateProperty('view-distance', String($event))"
          />
        </div>

        <!-- MOTD -->
        <div class="property-row full-width">
          <div class="property-label">
            <n-text strong>伺服器描述 (MOTD)</n-text>
            <n-text depth="3" class="property-key">motd</n-text>
          </div>
          <n-input
            :value="parsedProps.motd || 'A Minecraft Server'"
            placeholder="A Minecraft Server"
            class="property-control"
            @update:value="updateProperty('motd', $event)"
          />
        </div>

        <!-- 其他屬性提示 -->
        <div class="other-props-hint">
          <n-text depth="3">
            其他 {{ otherPropsCount }} 個設定項目可在進階模式中編輯
          </n-text>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue';
import { NInput, NSelect, NSwitch, NInputNumber, NButton, NSpace, NText } from 'naive-ui';

const props = defineProps({
  modelValue: {
    type: String,
    default: ''
  }
});

const emit = defineEmits(['update:modelValue']);

// 模式切換
const isAdvancedMode = ref(false);

// 原始內容
const rawContent = ref(props.modelValue);

// 解析後的屬性
const parsedProps = ref({});

// 已解析的行 (保留順序與註解)
const parsedLines = ref([]);

// 選項定義
const gamemodeOptions = [
  { label: 'Survival (生存)', value: 'survival' },
  { label: 'Creative (創造)', value: 'creative' },
  { label: 'Adventure (冒險)', value: 'adventure' },
  { label: 'Spectator (觀察者)', value: 'spectator' }
];

const difficultyOptions = [
  { label: 'Peaceful (和平)', value: 'peaceful' },
  { label: 'Easy (簡單)', value: 'easy' },
  { label: 'Normal (普通)', value: 'normal' },
  { label: 'Hard (困難)', value: 'hard' }
];

// 視覺化編輯支援的 key
const visualKeys = [
  'gamemode', 'difficulty', 'pvp', 'white-list', 
  'online-mode', 'max-players', 'view-distance', 'motd'
];

// 計算其他屬性數量
const otherPropsCount = computed(() => {
  const allKeys = Object.keys(parsedProps.value);
  return allKeys.filter(k => !visualKeys.includes(k)).length;
});

/**
 * 解析 server.properties 格式
 */
const parseProperties = (raw) => {
  const lines = raw.split('\n');
  const result = {};
  const lineData = [];

  lines.forEach((line, index) => {
    const trimmed = line.trim();
    
    // 保留空行和註解
    if (!trimmed || trimmed.startsWith('#')) {
      lineData.push({ type: 'comment', value: line, index });
      return;
    }

    // 解析 key=value
    const eqIndex = line.indexOf('=');
    if (eqIndex === -1) {
      lineData.push({ type: 'comment', value: line, index });
      return;
    }

    const key = line.substring(0, eqIndex).trim();
    const value = line.substring(eqIndex + 1).trim();
    
    result[key] = value;
    lineData.push({ type: 'property', key, value, index });
  });

  return { props: result, lines: lineData };
};

/**
 * 將解析後的屬性重新序列化為字串
 */
const serializeProperties = () => {
  return parsedLines.value.map(line => {
    if (line.type === 'comment') {
      return line.value;
    }
    return `${line.key}=${parsedProps.value[line.key] ?? line.value}`;
  }).join('\n');
};

/**
 * 更新單一屬性
 */
const updateProperty = (key, value) => {
  parsedProps.value[key] = value;
  rawContent.value = serializeProperties();
  emitUpdate();
};

/**
 * 發送更新事件
 */
const emitUpdate = () => {
  emit('update:modelValue', rawContent.value);
};

/**
 * 切換模式
 */
const toggleMode = () => {
  if (isAdvancedMode.value) {
    // 從進階模式切換回視覺化模式，重新解析
    const { props: parsed, lines } = parseProperties(rawContent.value);
    parsedProps.value = parsed;
    parsedLines.value = lines;
  }
  isAdvancedMode.value = !isAdvancedMode.value;
};

// 監聽 modelValue 變化
watch(() => props.modelValue, (newVal) => {
  rawContent.value = newVal;
  const { props: parsed, lines } = parseProperties(newVal);
  parsedProps.value = parsed;
  parsedLines.value = lines;
}, { immediate: true });
</script>

<style scoped>
.properties-editor {
  background: #1a1a20;
  border: 1px solid #333;
  border-radius: 6px;
  padding: 16px;
}

.editor-header {
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.mode-label {
  font-family: 'Fira Code', monospace;
  font-size: 11px;
  letter-spacing: 1px;
}

.raw-editor :deep(.n-input) {
  font-family: 'Fira Code', monospace;
  font-size: 13px;
  background: #0c0c0e;
}

.visual-editor {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.property-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.03);
}

.property-row.full-width {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}

.property-row.full-width .property-control {
  width: 100%;
}

.property-label {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.property-key {
  font-family: 'Fira Code', monospace;
  font-size: 10px;
}

.property-control {
  width: 180px;
}

.number-input {
  width: 120px;
}

.other-props-hint {
  text-align: center;
  padding: 12px;
  background: rgba(24, 160, 88, 0.05);
  border-radius: 4px;
  border: 1px dashed rgba(24, 160, 88, 0.2);
}

@media (max-width: 600px) {
  .property-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  
  .property-control {
    width: 100%;
  }
  
  .number-input {
    width: 100%;
  }
}
</style>
