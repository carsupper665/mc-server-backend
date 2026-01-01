<script setup>
import { ref, onMounted } from 'vue';
import { NGrid, NGridItem, NCard, NStatistic, NNumberAnimation, NText, NSpace, NSkeleton, useMessage } from 'naive-ui';
import { useAuthStore } from '../store/auth';
import api from '../api';

const authStore = useAuthStore();
const message = useMessage();
const loading = ref(true);

const stats = ref({
  totalServers: 0,
  activeServers: 0,
  ramUsed: 0,
  uptime: 100
});

const fetchDashboardData = async () => {
  try {
    // 1. Fetch user's servers
    const res = await api.get('/user/myservers');
    let servers = [];
    if (Array.isArray(res)) {
      servers = res;
    } else {
      servers = res.data || [];
    }
    
    stats.value.totalServers = servers.length;
    
    // 2. Check status for each server to count running ones
    // Note: This causes N+1 requests but is necessary without backend aggregation
    let runningCount = 0;
    
    await Promise.all(servers.map(async (srv) => {
      try {
        const statusRes = await api.post(`/mc-api/a/status/${srv.server_id}`);
        // Check both object.status and raw string response
        const status = (statusRes && statusRes.status) ? statusRes.status : statusRes;
        
        if (status && status.toLowerCase() === 'running') {
          runningCount++;
        }
      } catch (e) {
        // Ignore errors for individual server status checks
      }
    }));
    
    stats.value.activeServers = runningCount;
    // Estimate RAM: 2GB per running server (based on backend default)
    stats.value.ramUsed = runningCount * 2;
    
  } catch (err) {
    console.error('Failed to load dashboard data', err);
    message.error('無法載入儀表板數據');
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  fetchDashboardData();
});
</script>

<template>
  <div class="dashboard-view">
    <n-space vertical :size="24">
      <div class="welcome-section fade-in-up">
        <n-text depth="3" class="system-time">SYSTEM STATUS: [ NOMINAL ]</n-text>
        <h1 class="welcome-text">WELCOME BACK, OPERATOR <span class="highlight">{{ authStore.user?.username }}</span></h1>
      </div>

      <n-grid cols="1 s:2 m:4" responsive="screen" :x-gap="24" :y-gap="24">
        <n-grid-item>
          <n-card class="stat-card fade-in-up" style="animation-delay: 0.1s;">
            <template v-if="loading">
              <n-skeleton text :repeat="1" style="width: 60%" />
              <n-skeleton text style="width: 40%; margin-top: 12px; height: 32px" />
            </template>
            <n-statistic v-else label="ACTIVE SERVERS" :value="stats.activeServers">
              <template #prefix>
                <div class="stat-icon server"></div>
              </template>
              <template #suffix>
                / {{ stats.totalServers }}
              </template>
            </n-statistic>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card class="stat-card fade-in-up" style="animation-delay: 0.2s;">
            <template v-if="loading">
              <n-skeleton text :repeat="1" style="width: 60%" />
              <n-skeleton text style="width: 40%; margin-top: 12px; height: 32px" />
            </template>
            <n-statistic v-else label="TOTAL RAM USED" :value="stats.ramUsed">
              <template #suffix> GB </template>
            </n-statistic>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card class="stat-card fade-in-up" style="animation-delay: 0.3s;">
            <template v-if="loading">
              <n-skeleton text :repeat="1" style="width: 60%" />
              <n-skeleton text style="width: 40%; margin-top: 12px; height: 32px" />
            </template>
            <n-statistic v-else label="SYSTEM UPTIME" :value="stats.uptime">
              <template #suffix> % </template>
            </n-statistic>
          </n-card>
        </n-grid-item>
        <n-grid-item>
          <n-card class="stat-card fade-in-up" style="animation-delay: 0.4s;">
            <template v-if="loading">
              <n-skeleton text :repeat="1" style="width: 60%" />
              <n-skeleton text style="width: 40%; margin-top: 12px; height: 32px" />
            </template>
            <n-statistic v-else label="SECURITY LEVEL" :value="(authStore.user?.username === 'root' || authStore.user?.role === 'admin') ? 'ROOT' : 'USER'">
            </n-statistic>
          </n-card>
        </n-grid-item>
      </n-grid>
      
      <n-card title="RECENT ACTIVITY" class="activity-card fade-in-up" style="animation-delay: 0.5s;">
        <div class="empty-state">
          <n-text depth="3">NO RECENT LOGS RECORDED</n-text>
        </div>
      </n-card>
    </n-space>
  </div>
</template>

<style scoped>
.welcome-section {
  margin-bottom: 12px;
}

.system-time {
  font-family: 'Fira Code', monospace;
  font-size: 12px;
  letter-spacing: 1px;
}

.welcome-text {
  margin: 8px 0;
  font-size: 28px;
  font-weight: 300;
  letter-spacing: -0.5px;
}

.highlight {
  color: #18a058;
  font-weight: 700;
  text-shadow: 0 0 10px rgba(24, 160, 88, 0.3);
}

.stat-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
}

.activity-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
  min-height: 300px;
}

.empty-state {
  height: 200px;
  display: flex;
  justify-content: center;
  align-items: center;
  border: 1px dashed #333;
  border-radius: 4px;
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
