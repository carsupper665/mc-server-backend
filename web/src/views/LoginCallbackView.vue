<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { NButton, NCard, NProgress, NText } from 'naive-ui';
import { useAuthStore } from '../store/auth';
import { getSanitizedErrorMessage } from '../api';

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();

const status = ref('loading'); // loading | success | error
const title = ref('驗證中');
const message = ref('正在向伺服器交換登入資訊...');
const seconds = ref(5);
const shellRef = ref(null);
const mouseX = ref('50%');
let timer = null;

const percent = computed(() => (seconds.value / 5) * 100);
const isCompactError = computed(() => status.value === 'error' && message.value.trim().length <= 12);
const shellStyle = computed(() => ({
  '--mouse-x': mouseX.value,
}));

const goHome = () => {
  router.replace('/');
};

const goLogin = () => {
  router.replace('/login');
};

const clearTimer = () => {
  if (timer) {
    clearInterval(timer);
    timer = null;
  }
};

const handlePointerMove = (event) => {
  if (!shellRef.value) return;
  const rect = shellRef.value.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) return;
  const x = Math.max(0, Math.min(event.clientX - rect.left, rect.width));
  mouseX.value = `${((x / rect.width) * 100).toFixed(2)}%`;
};

const resetPointerGlow = () => {
  mouseX.value = '50%';
};

const startAutoRedirect = () => {
  clearTimer();
  seconds.value = 5;
  timer = setInterval(() => {
    if (seconds.value <= 1) {
      clearTimer();
      goHome();
      return;
    }
    seconds.value -= 1;
  }, 1000);
};

const run = async () => {
  try {
    await authStore.exchangeCallbackToken(route.query);
    status.value = 'success';
    title.value = '登入成功';
    message.value = '已完成登入，系統將在 5 秒後自動跳轉到主頁。';
    startAutoRedirect();
  } catch (err) {
    status.value = 'error';
    title.value = '登入失敗';
    message.value = getSanitizedErrorMessage(err) || '登入連結已失效或不正確，請重新登入。';
  }
};

onMounted(() => {
  run();
});

onBeforeUnmount(() => {
  clearTimer();
  resetPointerGlow();
});
</script>

<template>
  <div class="callback-page">
    <div
      ref="shellRef"
      :class="[
        'callback-card-shell',
        `callback-card-shell--${status}`,
        { 'callback-card-shell--compact': isCompactError },
      ]"
      :style="shellStyle"
      @pointermove="handlePointerMove"
      @pointerleave="resetPointerGlow"
    >
      <n-card class="callback-card" :bordered="false">
        <template #header>
          <div class="callback-header">
            <n-text class="callback-tag">[ CALLBACK ]</n-text>
            <h2>{{ title }}</h2>
          </div>
        </template>

        <div :class="['callback-content', `callback-content--${status}`]">
          <n-text>{{ message }}</n-text>

          <template v-if="status === 'loading'">
            <n-progress type="line" :percentage="90" processing :show-indicator="false" :height="6" />
          </template>

          <template v-else-if="status === 'success'">
            <n-text depth="3">自動跳轉倒數：{{ seconds }} 秒</n-text>
            <n-progress type="line" :percentage="percent" :show-indicator="false" :height="6" />
            <n-button type="primary" block @click="goHome">立即前往主頁</n-button>
          </template>

          <template v-else>
            <div class="callback-actions callback-actions--bottom">
              <n-button class="error-action" type="warning" block @click="goLogin">返回登入頁</n-button>
            </div>
          </template>
        </div>
      </n-card>
    </div>
  </div>
</template>

<style scoped>
.callback-page {
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: radial-gradient(circle at center, #1a1a20 0%, #0a0a0c 100%);
  padding: 16px;
}

.callback-card-shell {
  --glow-opacity: 0.32;
  --edge-opacity: 0.42;
  --glow-rgb: 56, 189, 248;
  --frame-color: #4b5563;
  --mouse-x: 50%;
  position: relative;
  width: 100%;
  max-width: 460px;
}

.callback-card-shell::before {
  content: '';
  position: absolute;
  width: 110%;
  height: 88%;
  left: 50%;
  top: 56%;
  transform: translate(-50%, -50%);
  z-index: 0;
  pointer-events: none;
  border-radius: 999px;
  filter: blur(42px);
  opacity: var(--glow-opacity);
  background: radial-gradient(circle at var(--mouse-x) 56%, rgba(var(--glow-rgb), 0.58), rgba(var(--glow-rgb), 0) 72%);
  transition: none;
}

.callback-card-shell::after {
  content: '';
  position: absolute;
  inset: -10px;
  z-index: 0;
  pointer-events: none;
  border-radius: 18px;
  opacity: var(--edge-opacity);
  box-shadow:
    0 0 0 1px rgba(var(--glow-rgb), 0.24),
    0 0 24px 5px rgba(var(--glow-rgb), 0.2),
    0 0 46px 14px rgba(var(--glow-rgb), 0.15);
  transition: none;
}

.callback-card {
  --n-color: #161a24;
  --n-color-embedded: #161a24;
  position: relative;
  z-index: 1;
  min-height: 340px;
  background: #161a24 !important;
  border: 1px solid var(--frame-color) !important;
  transition: border-color 0.28s ease, box-shadow 0.28s ease;
}

.callback-card :deep(.n-card__content),
.callback-card :deep(.n-card-header),
.callback-card :deep(.n-card-header__main),
.callback-card :deep(.n-card__action) {
  background: #161a24 !important;
}

.callback-card :deep(.n-card__content) {
  min-height: 220px;
}

.callback-content {
  min-height: 220px;
  display: flex;
  flex-direction: column;
  gap: 18px;
}

.callback-actions--bottom {
  margin-top: auto;
}

.error-action {
  --n-color: #cfd5b8 !important;
  --n-color-hover: #d7ddc0 !important;
  --n-color-pressed: #bcc3a7 !important;
  --n-border: 1px solid #c7ceaf !important;
  --n-border-hover: 1px solid #d0d6ba !important;
  --n-border-pressed: 1px solid #b4bc9f !important;
  --n-text-color: #1f2937 !important;
  --n-text-color-hover: #1f2937 !important;
  --n-text-color-pressed: #1f2937 !important;
  --n-ripple-color: #cfd5b8 !important;
}

.callback-card-shell--success {
  --glow-opacity: 0.58;
  --edge-opacity: 0.52;
  --glow-rgb: 16, 185, 129;
  --frame-color: rgba(16, 185, 129, 0.9);
}

.callback-card-shell--success::before {
  background: radial-gradient(circle at var(--mouse-x) 56%, rgba(var(--glow-rgb), 0.8), rgba(var(--glow-rgb), 0) 72%);
}

.callback-card-shell--success .callback-card {
  box-shadow: inset 0 0 0 1px rgba(16, 185, 129, 0.28);
}

.callback-card-shell--error {
  --glow-opacity: 0.58;
  --edge-opacity: 0.52;
  --glow-rgb: 239, 68, 68;
  --frame-color: rgba(239, 68, 68, 0.9);
}

.callback-card-shell--error::before {
  background: radial-gradient(circle at var(--mouse-x) 56%, rgba(var(--glow-rgb), 0.8), rgba(var(--glow-rgb), 0) 72%);
}

.callback-card-shell--error .callback-card {
  box-shadow: inset 0 0 0 1px rgba(239, 68, 68, 0.28);
}

.callback-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.callback-header h2 {
  margin: 0;
  color: #e5e7eb;
  font-size: 22px;
}

.callback-tag {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  padding: 2px 7px;
  border-radius: 999px;
  font-size: 11px;
  line-height: 1;
  color: var(--frame-color) !important;
  letter-spacing: 0.06em;
  font-weight: 700;
  background: rgba(var(--glow-rgb), 0.08);
  text-shadow: 0 0 10px rgba(var(--glow-rgb), 0.45);
  box-shadow:
    0 0 10px rgba(var(--glow-rgb), 0.24),
    inset 0 0 10px rgba(var(--glow-rgb), 0.08);
}

.callback-card-shell--error .callback-header h2 {
  text-align: center;
}

.callback-card-shell--compact .callback-card {
  min-height: 0;
}

.callback-card-shell--compact .callback-card :deep(.n-card__content) {
  min-height: 0;
}

.callback-card-shell--compact .callback-content {
  min-height: 0;
}
</style>
