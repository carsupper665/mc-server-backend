<script setup>
import { computed } from 'vue';
import { NCard, NDescriptions, NDescriptionsItem, NSpace, NText } from 'naive-ui';
import { useAuthStore } from '../store/auth';
import { formatMemoryGB, UNLIMITED_LABEL } from '../utils/roles';

const authStore = useAuthStore();

const accountRows = computed(() => [
  { label: '顯示名稱', value: authStore.displayName || '—' },
  { label: '帳號', value: authStore.user?.username || '—' },
  { label: '角色', value: authStore.roleName },
]);

const limitRows = computed(() => {
  const perm = authStore.user?.permissions;
  const unlimited = !!perm?.unlimited;
  const limit = (value, format = v => v) => {
    if (unlimited) return UNLIMITED_LABEL;
    return value == null ? '—' : format(value);
  };
  const flag = (value) => {
    if (!unlimited && value == null) return '—';
    return unlimited || value ? '可以' : '不可以';
  };
  return [
    { label: '單台記憶體上限', value: limit(perm?.server_max_mem, formatMemoryGB) },
    { label: '伺服器數量上限', value: limit(perm?.max_server_count) },
    { label: '同時運行上限', value: limit(perm?.max_online_count) },
    { label: '模組數量上限', value: limit(perm?.max_mods_count) },
    { label: '備份數量上限', value: limit(perm?.max_back_up_count) },
    { label: '建立備份', value: flag(perm?.can_back_up_server) },
    { label: '修改 JVM 參數', value: flag(perm?.can_edit_server_args) },
  ];
});
</script>

<template>
  <div class="profile-view">
    <n-space vertical :size="24">
      <n-card title="PROFILE" class="profile-card">
        <n-descriptions label-placement="left" :column="1" bordered>
          <n-descriptions-item v-for="row in accountRows" :key="row.label" :label="row.label">
            <n-text>{{ row.value }}</n-text>
          </n-descriptions-item>
        </n-descriptions>
      </n-card>
      <n-card title="LIMITS" class="profile-card">
        <n-descriptions label-placement="left" :column="1" bordered>
          <n-descriptions-item v-for="row in limitRows" :key="row.label" :label="row.label">
            <n-text>{{ row.value }}</n-text>
          </n-descriptions-item>
        </n-descriptions>
      </n-card>
    </n-space>
  </div>
</template>

<style scoped>
.profile-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
}
</style>
