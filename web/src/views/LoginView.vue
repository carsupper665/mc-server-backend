<script setup>
import { ref, computed, onBeforeUnmount } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../store/auth';
import { NCard, NForm, NFormItem, NInput, NButton, NSpace, NText, NProgress, useMessage } from 'naive-ui';

const router = useRouter();
const authStore = useAuthStore();
const message = useMessage();

const loginForm = ref({
  username: '',
  password: ''
});

const verificationCode = ref('');
const isVerifying = ref(false);

// 倒數計時器邏輯 (5 分鐘 = 300 秒)
const countdown = ref(0);
let countdownInterval = null;

const countdownDisplay = computed(() => {
  const mins = Math.floor(countdown.value / 60);
  const secs = countdown.value % 60;
  return `${mins}:${secs.toString().padStart(2, '0')}`;
});

const countdownPercent = computed(() => {
  return (countdown.value / 300) * 100;
});

const isExpired = computed(() => {
  return isVerifying.value && countdown.value === 0;
});

const startCountdown = () => {
  countdown.value = 300; // 5 分鐘
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

// 重新發送驗證碼
const resendCode = async () => {
  stopCountdown();
  try {
    const res = await authStore.login(loginForm.value.username, loginForm.value.password);
    if (res.message && res.message.includes('verification code sent')) {
      startCountdown();
      message.success('驗證碼已重新發送，請查看 Email');
    }
  } catch (err) {
    message.error('發送失敗：' + (err.response?.data?.error || '未知錯誤'));
  }
};

const handleLogin = async () => {
  if (isVerifying.value) {
    if (!verificationCode.value) {
      message.warning('請輸入驗證碼');
      return;
    }
    if (isExpired.value) {
      message.warning('驗證碼已過期，請重新發送');
      return;
    }
    try {
      stopCountdown();
      await authStore.verifyCode(verificationCode.value);
      message.success('驗證成功');
      router.push('/');
    } catch (err) {
      message.error('驗證失敗：' + (err.response?.data?.error || '驗證碼錯誤'));
    }
    return;
  }

  if (!loginForm.value.username || !loginForm.value.password) {
    message.warning('請輸入帳號與密碼');
    return;
  }

  try {
    const res = await authStore.login(loginForm.value.username, loginForm.value.password);
    // Backend returns 202 if verification is needed
    // res here is already unwrapped data due to axios interceptor
    if (res.message && res.message.includes('verification code sent')) {
      isVerifying.value = true;
      startCountdown();
      message.info('驗證碼已寄出，請查看 Email');
    } else {
      message.success('登入成功');
      router.push('/');
    }
  } catch (err) {
    message.error('登入失敗：' + (err.response?.data?.error || '未知錯誤'));
  }
};

// 返回登入時清除計時器
const backToLogin = () => {
  stopCountdown();
  isVerifying.value = false;
  verificationCode.value = '';
};

onBeforeUnmount(() => {
  stopCountdown();
});
</script>

<template>
  <div class="login-container">
    <div class="scanlines"></div>
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
          <n-form-item label="VERIFICATION CODE" path="code">
            <n-input 
              v-model:value="verificationCode" 
              placeholder="6-Digit Code"
              class="terminal-input"
              maxlength="6"
              :disabled="isExpired"
            />
          </n-form-item>
          
          <!-- 倒數計時器區塊 -->
          <div class="countdown-section">
            <n-space justify="space-between" align="center">
              <n-text :depth="isExpired ? 1 : 3" :class="{ 'expired-text': isExpired }">
                {{ isExpired ? '⚠ 驗證碼已過期' : `驗證碼有效時間：${countdownDisplay}` }}
              </n-text>
              <n-button 
                v-if="isExpired" 
                size="tiny" 
                type="warning"
                ghost
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
              :height="4"
              class="countdown-progress"
            />
          </div>
        </template>
        
        <n-space vertical :size="20">
          <n-button 
            type="primary" 
            block 
            :loading="authStore.loading"
            @click="handleLogin"
            class="terminal-button"
          >
            {{ isVerifying ? 'VERIFY CODE' : 'INITIALIZE AUTHENTICATION' }}
          </n-button>
          
          <n-button 
            v-if="isVerifying"
            text 
            @click="backToLogin"
            class="back-link"
          >
            BACK TO LOGIN
          </n-button>
          
          <div class="terminal-footer">
            <n-text depth="3">AUTHORIZED PERSONNEL ONLY</n-text>
          </div>
        </n-space>
      </n-form>
    </n-card>
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
  z-index: 10;
}

.login-card {
  width: 90%;
  max-width: 400px;
  background: rgba(20, 20, 25, 0.8) !important;
  backdrop-filter: blur(10px);
  border: 1px solid #333 !important;
  box-shadow: 0 0 40px rgba(0, 0, 0, 0.5), inset 0 0 20px rgba(24, 160, 88, 0.05);
  z-index: 20;
  animation: cardEntry 0.8s cubic-bezier(0.19, 1, 0.22, 1);
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

.terminal-footer {
  text-align: center;
  font-size: 10px;
  letter-spacing: 1px;
  opacity: 0.5;
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
  z-index: 11;
}

/* 倒數計時器樣式 */
.countdown-section {
  margin-top: 12px;
  padding: 12px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.countdown-progress {
  margin-top: 8px;
}

.expired-text {
  color: #e88080 !important;
  font-weight: 600;
}

.back-link {
  opacity: 0.7;
  transition: opacity 0.2s;
}

.back-link:hover {
  opacity: 1;
}
</style>
