<script setup>
import { h, ref, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { useAuthStore } from '../store/auth';
import { 
  NLayout, NLayoutHeader, NLayoutSider, NLayoutContent, 
  NMenu, NSpace, NAvatar, NDropdown, NText, NIcon, NDrawer, NDrawerContent, NButton,
  NBreadcrumb, NBreadcrumbItem
} from 'naive-ui';
import {
  DashboardOutlined,
  HddOutlined,
  UserOutlined,
  LogoutOutlined,
  MenuOutlined,
  TeamOutlined
} from '@vicons/antd';

const router = useRouter();
const route = useRoute();
const authStore = useAuthStore();
const activeKey = computed({
  get: () => route.name,
  set: (val) => {
    // Menu handles navigation via onClick
  }
});
const showMobileMenu = ref(false);

const menuOptions = [
  {
    label: 'Dashboard',
    key: 'Dashboard',
    icon: () => h(NIcon, null, { default: () => h(DashboardOutlined) }),
    onClick: () => {
      router.push({ name: 'Dashboard' });
      showMobileMenu.value = false;
    }
  },
  {
    label: 'Servers',
    key: 'Servers',
    icon: () => h(NIcon, null, { default: () => h(HddOutlined) }),
    onClick: () => {
      router.push({ name: 'Servers' });
      showMobileMenu.value = false;
    }
  },
  {
    label: 'Among Us',
    key: 'AmongUs',
    icon: () => h(NIcon, null, { default: () => h(TeamOutlined) }),
    onClick: () => {
      router.push({ name: 'AmongUs' });
      showMobileMenu.value = false;
    }
  }
];

const userOptions = [
  {
    label: 'Profile',
    key: 'profile',
    icon: () => h(NIcon, null, { default: () => h(UserOutlined) })
  },
  {
    label: 'Logout',
    key: 'logout',
    icon: () => h(NIcon, null, { default: () => h(LogoutOutlined) }),
    onClick: () => authStore.logout()
  }
];

// 動態麵包屑
const breadcrumbs = computed(() => {
  const crumbs = [];
  const name = route.name;
  
  if (name === 'Dashboard') {
    crumbs.push({ label: 'Dashboard', to: '/' });
  } else if (name === 'Servers') {
    crumbs.push({ label: 'Servers', to: '/servers' });
  } else if (name === 'ServerDetail') {
    crumbs.push({ label: 'Servers', to: '/servers' });
    crumbs.push({ label: route.params.id || 'Detail', to: null });
  }
  
  return crumbs;
});

const goToSystem = () => {
  router.push('/');
};
</script>

<template>
  <n-layout has-sider class="main-container">
    <!-- Desktop Sider -->
    <n-layout-sider
      bordered
      collapse-mode="width"
      :collapsed-width="64"
      :width="240"
      show-trigger
      class="sider desktop-sider"
    >
      <div class="logo-area">
        <div class="logo-text">MC-ADMIN</div>
      </div>
      <n-menu
        v-model:value="activeKey"
        :collapsed-width="64"
        :collapsed-icon-size="22"
        :options="menuOptions"
      />
    </n-layout-sider>
    
    <!-- Mobile Drawer -->
    <n-drawer v-model:show="showMobileMenu" placement="left" width="240">
      <n-drawer-content body-style="padding: 0; background-color: #101014;">
        <div class="logo-area">
          <div class="logo-text">MC-ADMIN</div>
        </div>
        <n-menu
          v-model:value="activeKey"
          :options="menuOptions"
        />
      </n-drawer-content>
    </n-drawer>
    
    <n-layout>
      <n-layout-header bordered class="header">
        <div class="header-content">
          <div class="left-section">
            <n-button 
              text 
              class="mobile-menu-btn" 
              @click="showMobileMenu = true"
            >
              <n-icon size="24"><MenuOutlined /></n-icon>
            </n-button>
            
            <div class="page-info">
              <n-breadcrumb class="nav-breadcrumb">
                <n-breadcrumb-item @click="goToSystem" class="system-link">
                  SYSTEM
                </n-breadcrumb-item>
                <n-breadcrumb-item 
                  v-for="(crumb, idx) in breadcrumbs" 
                  :key="idx"
                  @click="crumb.to && router.push(crumb.to)"
                  :class="{ 'active-crumb': !crumb.to, 'clickable-crumb': crumb.to }"
                >
                  {{ crumb.label }}
                </n-breadcrumb-item>
              </n-breadcrumb>
            </div>
          </div>
          
          <n-space align="center" :size="20">
            <n-dropdown :options="userOptions">
              <div class="user-profile">
                <n-avatar round size="small" :style="{ backgroundColor: '#18a058' }">
                  {{ authStore.user?.username?.charAt(0).toUpperCase() }}
                </n-avatar>
                <n-text class="username">{{ authStore.user?.username }}</n-text>
              </div>
            </n-dropdown>
          </n-space>
        </div>
      </n-layout-header>
      
      <n-layout-content content-style="padding: 24px;" class="content-area">
        <router-view v-slot="{ Component }">
          <transition name="fade-transform" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </n-layout-content>
    </n-layout>
  </n-layout>
</template>

<style scoped>
.main-container {
  height: 100vh;
}

.sider {
  background-color: #101014;
}

.desktop-sider {
  display: block;
}

.mobile-menu-btn {
  display: none;
  margin-right: 16px;
  color: #fff;
}

.logo-area {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.logo-text {
  font-family: 'Fira Code', monospace;
  font-weight: 700;
  color: #18a058;
  letter-spacing: 2px;
}

.header {
  height: 64px;
  padding: 0 24px;
  background-color: #101014;
}

.header-content {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.left-section {
  display: flex;
  align-items: center;
}

.page-title, .page-info {
  font-family: 'Fira Code', monospace;
  font-size: 14px;
}

/* 麵包屑導航樣式 */
.nav-breadcrumb {
  font-family: 'Fira Code', monospace;
}

.nav-breadcrumb :deep(.n-breadcrumb-item__link) {
  color: #666;
  font-weight: 500;
  letter-spacing: 1px;
  transition: all 0.2s ease;
}

.nav-breadcrumb :deep(.n-breadcrumb-item__separator) {
  color: #444;
  margin: 0 8px;
}

.system-link :deep(.n-breadcrumb-item__link) {
  color: #888;
  cursor: pointer;
}

.system-link :deep(.n-breadcrumb-item__link):hover {
  color: #18a058;
  text-shadow: 0 0 8px rgba(24, 160, 88, 0.4);
}

.clickable-crumb :deep(.n-breadcrumb-item__link) {
  color: #888;
  cursor: pointer;
}

.clickable-crumb :deep(.n-breadcrumb-item__link):hover {
  color: #18a058;
  text-shadow: 0 0 8px rgba(24, 160, 88, 0.4);
}

.active-crumb :deep(.n-breadcrumb-item__link) {
  color: #18a058 !important;
  font-weight: 600;
  text-shadow: 0 0 10px rgba(24, 160, 88, 0.3);
}

.user-profile {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background 0.2s;
}

.user-profile:hover {
  background: rgba(255, 255, 255, 0.05);
}

.username {
  font-size: 14px;
  font-weight: 500;
  display: block;
}

.content-area {
  background-color: #0c0c0e;
}

/* Transitions */
.fade-transform-enter-active,
.fade-transform-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.fade-transform-enter-from {
  opacity: 0;
  transform: translateX(-10px);
}

.fade-transform-leave-to {
  opacity: 0;
  transform: translateX(10px);
}

/* Responsive */
@media (max-width: 768px) {
  .desktop-sider {
    display: none;
  }
  
  .mobile-menu-btn {
    display: flex;
  }
  
  .username {
    display: none;
  }
  
  .header {
    padding: 0 16px;
  }
}
</style>
