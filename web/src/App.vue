<script setup>
import { NConfigProvider, NGlobalStyle, darkTheme, NMessageProvider, NDialogProvider } from 'naive-ui';
import { zhTW, dateZhTW } from 'naive-ui';
</script>

<template>
  <n-config-provider :theme="darkTheme" :locale="zhTW" :date-locale="dateZhTW">
    <n-message-provider>
      <n-dialog-provider>
        <router-view />
      </n-dialog-provider>
    </n-message-provider>
    <n-global-style />
  </n-config-provider>
</template>

<style>
body {
  margin: 0;
  padding: 0;
  font-family: v-sans, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  background-color: #101014;
}

/* Industrial Terminal Aesthetic Styles */
::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}
::-webkit-scrollbar-track {
  background: #101014;
}
::-webkit-scrollbar-thumb {
  background: #2a2a30;
  border-radius: 3px;
}
::-webkit-scrollbar-thumb:hover {
  background: #3a3a40;
}
</style>