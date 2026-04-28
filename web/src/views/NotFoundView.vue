<script setup>
import { useRouter } from 'vue-router';
import { NButton, NCard, NSpace, NText } from 'naive-ui';

const router = useRouter();

const goHome = () => router.push('/');
const goLogin = () => router.push('/login');
</script>

<template>
  <div class="notfound-page">
    <n-card class="notfound-card" :bordered="false">
      <n-space vertical :size="16">
        <n-text depth="3">[ 404 ]</n-text>
        <h1>頁面不存在</h1>
        <n-text depth="2">
          你輸入的網址沒有對應頁面，請返回主頁或重新登入。
        </n-text>
        <n-space>
          <n-button type="primary" @click="goHome">回主頁</n-button>
          <n-button @click="goLogin">去登入</n-button>
        </n-space>
      </n-space>
    </n-card>
  </div>
</template>

<style scoped>
.notfound-page {
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: radial-gradient(circle at center, #171a22 0%, #0a0a0d 100%);
  padding: 16px;
}

.notfound-card {
  width: 100%;
  max-width: 520px;
  background: rgba(20, 20, 25, 0.86) !important;
  border: 1px solid #333 !important;
}

h1 {
  margin: 0;
  color: #f3f4f6;
}
</style>
