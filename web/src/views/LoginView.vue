<script setup>
import { ref, computed, onBeforeUnmount, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../store/auth';
import { getSanitizedErrorMessage } from '../api';
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NText, NProgress, useMessage } from 'naive-ui';

const router = useRouter();
const authStore = useAuthStore();
const message = useMessage();

const loginForm = ref({
  username: '',
  password: ''
});

const isVerifying = ref(false);

// 倒數計時器邏輯（5 分鐘 = 300 秒）
const COUNTDOWN_TOTAL_SECONDS = 300;
const countdown = ref(0);
let countdownInterval = null;

const countdownDisplay = computed(() => {
  const mins = Math.floor(countdown.value / 60);
  const secs = countdown.value % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
});

const countdownPercent = computed(() => {
  return (countdown.value / COUNTDOWN_TOTAL_SECONDS) * 100;
});

const isExpired = computed(() => {
  return isVerifying.value && countdown.value === 0;
});

const startCountdown = () => {
  countdown.value = COUNTDOWN_TOTAL_SECONDS;
  if (countdownInterval) clearInterval(countdownInterval);
  countdownInterval = setInterval(() => {
    if (countdown.value > 0) {
      countdown.value--;
    } else {
      clearInterval(countdownInterval);
      countdownInterval = null;
    }
  }, 1000);
};

const stopCountdown = () => {
  if (countdownInterval) {
    clearInterval(countdownInterval);
    countdownInterval = null;
  }
};

const errorText = (err, fallback = '未知錯誤') => {
  const explicit = err.response?.data?.error;
  if (explicit) return explicit;
  const sanitized = getSanitizedErrorMessage(err);
  return sanitized || fallback;
};

// 重新發送驗證碼
const resendCode = async () => {
  stopCountdown();
  try {
    const res = await authStore.login(loginForm.value.username, loginForm.value.password);
    if (res.requiresVerification) {
      startCountdown();
      message.success('驗證碼已重新發送，請查看 Email');
      return;
    }
    message.success('已登入');
    router.push('/');
  } catch (err) {
    message.error('發送失敗：' + errorText(err));
  }
};

const handleLogin = async () => {
  if (isVerifying.value) {
    message.info('請到信箱點擊驗證連結完成登入');
    return;
  }

  if (!loginForm.value.username || !loginForm.value.password) {
    message.warning('請輸入帳號與密碼');
    return;
  }

  try {
    const res = await authStore.login(loginForm.value.username, loginForm.value.password);
    if (res.requiresVerification) {
      isVerifying.value = true;
      startCountdown();
      message.info('驗證碼已寄出，請查看 Email');
    } else {
      message.success('登入成功');
      router.push('/');
    }
  } catch (err) {
    message.error('登入失敗：' + errorText(err));
  }
};

const autoLogin = async () => {
  try {
    await authStore.fetchUser();
    if (authStore.isLoggedIn) {
      router.replace('/');
    }
  } catch {
    // ignore if not logged in
  }
};

// 返回登入時清除計時器
const backToLogin = () => {
  stopCountdown();
  isVerifying.value = false;
};

onMounted(() => {
  autoLogin();
});

onBeforeUnmount(() => {
  stopCountdown();
});
</script>

<template>
  <div class="login-container">
    <div class="scanlines"></div>
    <div class="login-card-wrap">
      <div v-if="isVerifying && !isExpired" class="card-back-countdown">{{ countdownDisplay }}</div>
      <n-card class="login-card" :bordered="false">
        <template #header>
          <div class="login-header">
            <div class="brand-logo">MC-SERVER</div>
            <n-text depth="3" class="system-tag">{{ isVerifying ? '[ VERIFICATION REQUIRED ]' : '[ SYSTEM ACCESS ]' }}</n-text>
          </div>
        </template>
        
        <n-form @keyup.enter="handleLogin">
          <template v-if="!isVerifying">
            <n-form-item label="OPERATOR ID" path="username">
              <n-input 
                v-model:value="loginForm.username" 
                placeholder="Username / Email"
                class="terminal-input"
              />
            </n-form-item>
            <n-form-item label="ACCESS KEY" path="password">
              <n-input 
                v-model:value="loginForm.password" 
                type="password" 
                show-password-on="mousedown"
                placeholder="Password"
                class="terminal-input"
              />
            </n-form-item>
          </template>
          
          <template v-else>
            <div class="verification-message">
              <n-text depth="2">
                驗證連結已寄出，請前往 Email 點擊登入連結。
              </n-text>
              <n-text depth="3">
                完成驗證後會自動導回系統。
              </n-text>
            </div>
            
            <!-- 倒數計時器區塊 -->
            <div class="countdown-section" :class="{ 'countdown-section-expired': isExpired }">
              <n-space justify="space-between" align="center">
                <n-text :depth="isExpired ? 1 : 3" :class="{ 'expired-text': isExpired }">
                  <template v-if="isExpired">
                    驗證碼已過期
                  </template>
                  <template v-else>
                    驗證碼有效時間
                  </template>
                </n-text>
                <n-button 
                  v-if="isExpired" 
                  size="tiny" 
                  text
                  class="resend-link back-link"
                  @click="resendCode"
                  :loading="authStore.loading"
                >
                  重新發送
                </n-button>
              </n-space>
              <n-progress 
                type="line" 
                :percentage="countdownPercent" 
                :show-indicator="false"
                :status="isExpired ? 'error' : 'success'"
                :color="isExpired ? '#ef4444' : undefined"
                :height="4"
                class="countdown-progress"
              />
            </div>
          </template>
          
          <n-space vertical :size="20">
            <n-button 
              v-if="!isVerifying"
              type="primary" 
              block 
              :loading="authStore.loading"
              @click="handleLogin"
              class="terminal-button"
            >
              INITIALIZE AUTHENTICATION
            </n-button>
            
            <n-button 
              v-if="isVerifying"
              text 
              @click="backToLogin"
              class="back-link"
            >
              ← BACK TO LOGIN
            </n-button>
            
          </n-space>
        </n-form>
      </n-card>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: radial-gradient(circle at center, #1a1a20 0%, #0a0a0c 100%);
  position: relative;
  overflow: hidden;
}

.scanlines {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: linear-gradient(
    rgba(18, 16, 16, 0) 50%,
    rgba(0, 0, 0, 0.1) 50%
  ),
  linear-gradient(
    90deg,
    rgba(255, 0, 0, 0.02),
    rgba(0, 255, 0, 0.01),
    rgba(0, 0, 255, 0.02)
  );
  background-size: 100% 4px, 3px 100%;
  pointer-events: none;
  z-NameIndex: 10;
}

.login-card-wrap {
  position: relative;
  width: 90%;
  max-width: 400px;
  z-index: 20;
}

.card-back-countdown {
  position: absolute;
  left: 50%;
  top: 50%;
  transform: translate(-50%, -50%);
  z-index: 0;
  pointer-events: none;
  user-select: none;
  color: rgba(255, 255, 255, 0.28);
  font-size: clamp(170px, 34vw, 300px);
  font-weight: 900;
  line-height: 1;
  letter-spacing: 3px;
  white-space: nowrap;
}

.login-card {
  width: 100%;
  position: relative;
  z-index: 1;
  background: rgba(20, 20, 25, 0.37) !important;
  backdrop-filter: blur(10px);
  border: 1px solid #333 !important;
  box-shadow: 0 0 40px rgba(0, 0, 0, 0.5), inset 0 0 20px rgba(24, 160, 88, 0.05);
  animation: cardEntry 0.8s cubic-bezier(0.19, 1, 0.22, 1);
  transition: background-color 0.2s ease;
}

.login-card:hover {
  background: rgba(20, 20, 25, 0.8) !important;
}

@keyframes cardEntry {
  from {
    opacity: 0;
    transform: translateY(30px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.login-header {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 10px 0;
}

.brand-logo {
  font-family: 'Fira Code', monospace;
  font-size: 24px;
  font-weight: 700;
  letter-spacing: 4px;
  color: #18a058;
  text-shadow: 0 0 10px rgba(24, 160, 88, 0.5);
}

.system-tag {
  font-size: 10px;
  letter-spacing: 2px;
}

.terminal-input :deep(.n-input) {
  background-color: rgba(0, 0, 0, 0.3);
  border-radius: 2px;
}

.terminal-button {
  font-weight: 600;
  letter-spacing: 1px;
  border-radius: 2px;
}

@keyframes flicker {
  0% { opacity: 0.97; }
  5% { opacity: 0.95; }
  10% { opacity: 0.9; }
  15% { opacity: 0.95; }
  20% { opacity: 0.98; }
  25% { opacity: 0.95; }
  30% { opacity: 0.9; }
  100% { opacity: 1; }
}

.login-container::after {
  content: "";
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(18, 16, 16, 0.05);
  pointer-events: none;
  animation: flicker 0.1s infinite;
  z-NameIndex: 11;
}

/* 倒數計時器樣式 */
.countdown-section {
  margin-top: 12px;
  padding: 12px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.05);
  position: relative;
  overflow: hidden;
}

.countdown-section > * {
  position: relative;
  z-index: 1;
}

.countdown-section-expired::before {
  content: "⚠︎";
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(239, 68, 68, 0.2);
  font-size: 150px;
  font-weight: 100;
  line-height: 1;
  z-index: 0;
  pointer-events: none;
}

.countdown-progress {
  margin-top: 8px;
}

.verification-message {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 4px;
}

.expired-text {
  color: #ef4444 !important;
  font-weight: 600;
}

.back-link {
  margin-top: 12px;
  opacity: 0.7;
  transition: opacity 0.2s;
}

.back-link:hover {
  opacity: 1;
}

.resend-link {
  margin-top: 0 !important;
  border: 2px solid currentColor !important;
  padding: 2px 10px !important;
  height: 30px !important;
  opacity: 0.95;
  --n-text-color: #c6ccb1 !important;
  /* --n-text-color-hover: #b8bea9 !important;
  --n-text-color-pressed: #aab09b !important;
  --n-text-color-focus: #c6ccb1 !important; */
}
</style>
