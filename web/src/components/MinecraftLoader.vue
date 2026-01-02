<template>
  <div class="minecraft-loader">
    <div class="blocks-container">
      <div class="block grass"></div>
      <div class="block dirt"></div>
      <div class="block stone"></div>
    </div>
    <p class="loader-message">{{ message }}</p>
    <div class="progress-dots">
      <span class="dot"></span>
      <span class="dot"></span>
      <span class="dot"></span>
    </div>
  </div>
</template>

<script setup>
defineProps({
  message: {
    type: String,
    default: '正在處理中...'
  }
});
</script>

<style scoped>
.minecraft-loader {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
}

.blocks-container {
  display: flex;
  gap: 8px;
  margin-bottom: 24px;
}

.block {
  width: 40px;
  height: 40px;
  border-radius: 4px;
  animation: bounce 0.6s ease-in-out infinite;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.block.grass {
  background: linear-gradient(180deg, #7cb342 0%, #558b2f 40%, #8d6e63 40%, #8d6e63 100%);
  animation-delay: 0s;
}

.block.dirt {
  background: linear-gradient(180deg, #8d6e63 0%, #6d4c41 100%);
  animation-delay: 0.15s;
}

.block.stone {
  background: linear-gradient(180deg, #78909c 0%, #546e7a 100%);
  animation-delay: 0.3s;
}

@keyframes bounce {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-20px);
  }
}

.loader-message {
  color: #a0aec0;
  font-size: 14px;
  margin: 0 0 16px 0;
  text-align: center;
  letter-spacing: 0.5px;
}

.progress-dots {
  display: flex;
  gap: 6px;
}

.dot {
  width: 8px;
  height: 8px;
  background: #18a058;
  border-radius: 50%;
  animation: pulse-dot 1.4s ease-in-out infinite;
}

.dot:nth-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-child(3) {
  animation-delay: 0.4s;
}

@keyframes pulse-dot {
  0%, 80%, 100% {
    opacity: 0.3;
    transform: scale(0.8);
  }
  40% {
    opacity: 1;
    transform: scale(1);
  }
}
</style>
