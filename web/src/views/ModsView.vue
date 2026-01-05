<script setup>
import { NSpace, NText } from 'naive-ui';
import ModBrowser from '../components/ModBrowser.vue';

const handleSelectMod = (mod) => {
  console.log('Selected mod:', mod);
};
</script>

<template>
  <div class="mods-view fade-in-up">
    <n-space vertical :size="24">
      <div class="page-header">
        <h2 class="title">MOD BROWSER</h2>
        <n-text depth="3">EXPLORE AND DISCOVER MINECRAFT MODS FROM MODRINTH</n-text>
      </div>
      
      <ModBrowser @select-mod="handleSelectMod" />
    </n-space>
  </div>
</template>

<style scoped>
.mods-view {
  max-width: 1400px;
  margin: 0 auto;
}

.title {
  margin: 0;
  font-size: 24px;
  font-family: 'Fira Code', monospace;
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
</style>
