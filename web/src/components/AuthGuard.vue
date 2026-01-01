<template>
  <!-- 只在權限足夠時渲染 slot 內容 -->
  <slot v-if="hasPermission" />
</template>

<script setup>
import { computed } from 'vue';
import { useAuthStore } from '../store/auth';

/**
 * AuthGuard 元件
 * 
 * 根據使用者角色權限控制 UI 顯示。
 * 權限不足時直接從 DOM 移除內容 (非 Disable)。
 * 
 * 使用方式：
 * <AuthGuard :minRole="4">
 *   <n-button>只有 Admin 才能看到的按鈕</n-button>
 * </AuthGuard>
 * 
 * 角色對照：
 * - Guest: 0
 * - Common: 1
 * - Admin: 4
 * - Root: 6
 */

const props = defineProps({
  /**
   * 最低角色權限要求
   */
  minRole: {
    type: Number,
    default: 1
  },
  /**
   * 允許的角色列表 (優先於 minRole)
   */
  allowedRoles: {
    type: Array,
    default: () => []
  },
  /**
   * 是否反轉邏輯 (權限足夠時隱藏)
   */
  inverted: {
    type: Boolean,
    default: false
  }
});

const authStore = useAuthStore();

const hasPermission = computed(() => {
  const userRole = authStore.user?.role ?? 0;

  let permitted = false;

  // 優先使用 allowedRoles
  if (props.allowedRoles.length > 0) {
    permitted = props.allowedRoles.includes(userRole);
  } else {
    // 使用 minRole
    permitted = userRole >= props.minRole;
  }

  // 反轉邏輯
  return props.inverted ? !permitted : permitted;
});
</script>
