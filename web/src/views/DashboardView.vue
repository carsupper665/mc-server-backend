<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { NGrid, NGridItem, NCard, NStatistic, NNumberAnimation, NText, NSpace, NSkeleton, NTag, NIcon, NEmpty, useMessage } from 'naive-ui';
import { CheckCircleOutlined, CloseCircleOutlined, ClockCircleOutlined } from '@vicons/antd';
import { useAuthStore } from '../store/auth';
import { useVersionCacheStore } from '../store/versionCache';
import { useActivityLogStore, formatRelativeTime } from '../store/activityLog';
import MetricsChart from '../components/MetricsChart.vue';
import api from '../api';

const authStore = useAuthStore();
const versionCache = useVersionCacheStore();
const activityLog = useActivityLogStore();
const message = useMessage();
const loading = ref(true);

const stats = ref({
  totalServers: 0,
  activeServers: 0,
  ramUsed: 0,
  uptime: 100
});

// 偽即時走勢圖歷史數據 (最多 30 點)
const metricsHistory = ref([]);
const METRICS_HISTORY_MAX = 30;

// 從活動紀錄 Store 取得最近 10 筆活動
const recentActivities = computed(() => {
  return activityLog.recentActivities(10);
});

// 更新走勢圖數據
const updateMetrics = () => {
  const cpu = Math.round(20 + Math.random() * 40); // 模擬 CPU (20-60%)
  const ram = Number((1 + Math.random() * 3).toFixed(1)); // 模擬 RAM (1-4 GB)
  
  metricsHistory.value.push({
    timestamp: Math.floor(Date.now() / 1000),
    cpu,
    ram
  });
  
  // 保持最大長度
  if (metricsHistory.value.length > METRICS_HISTORY_MAX) {
    metricsHistory.value.shift();
  }
};

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
    let runningCount = 0;
    
    await Promise.all(servers.map(async (srv) => {
      try {
        const statusRes = await api.post(`/mc-api/a/status/${srv.server_id}`);
        const status = (statusRes && statusRes.status) ? statusRes.status : statusRes;
        
        if (status && status.toLowerCase() === 'running') {
          runningCount++;
        }
      } catch (e) {
        // Ignore errors for individual server status checks
      }
    }));
    
    stats.value.activeServers = runningCount;
    stats.value.ramUsed = runningCount * 2;
    
    // 更新走勢圖
    updateMetrics();
    
  } catch (err) {
    console.error('Failed to load dashboard data', err);
    message.error('無法載入儀表板數據');
  } finally {
    loading.value = false;
  }
};

// 輪詢間隔
let metricsInterval = null;

onMounted(() => {
  // 初始化活動紀錄
  activityLog.init();
  
  fetchDashboardData();
  versionCache.prefetchVersions();
  
  // 每 10 秒更新一次走勢圖
  metricsInterval = setInterval(updateMetrics, 10000);
});

onBeforeUnmount(() => {
  if (metricsInterval) clearInterval(metricsInterval);
});

// 格式化相對時間
const getRelativeTime = (timestamp) => {
  return formatRelativeTime(timestamp);
};

// 取得操作圖示
const getActionIcon = (status) => {
  return status === 'success' ? CheckCircleOutlined : CloseCircleOutlined;
};
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
      
      <!-- 系統走勢圖 -->
      <n-grid cols="1" :x-gap="24" :y-gap="24">
        <n-grid-item>
          <MetricsChart 
            :metrics-history="metricsHistory"
            title="SYSTEM METRICS"
            class="fade-in-up"
            style="animation-delay: 0.45s;"
          />
        </n-grid-item>
      </n-grid>
      
      <!-- 近期活動紀錄 Widget -->
      <n-card title="RECENT ACTIVITY" class="activity-card fade-in-up" style="animation-delay: 0.5s;">
        <template v-if="recentActivities.length === 0">
          <div class="empty-state">
            <n-empty description="尚無操作紀錄" />
          </div>
        </template>
        <template v-else>
          <div class="activity-list">
            <div 
              v-for="activity in recentActivities" 
              :key="activity.id" 
              class="activity-item"
            >
              <div class="activity-icon">
                <n-icon 
                  :component="getActionIcon(activity.status)" 
                  :color="activity.status === 'success' ? '#18a058' : '#e88080'"
                  size="18"
                />
              </div>
              <div class="activity-content">
                <div class="activity-action">
                  <n-text strong>{{ activity.action }}</n-text>
                  <n-tag size="tiny" :type="activity.status === 'success' ? 'success' : 'error'">
                    {{ activity.status === 'success' ? '成功' : '失敗' }}
                  </n-tag>
                </div>
                <n-text depth="3" class="activity-target">{{ activity.targetName }}</n-text>
                <n-text v-if="activity.details" depth="3" class="activity-details">{{ activity.details }}</n-text>
              </div>
              <div class="activity-time">
                <n-icon :component="ClockCircleOutlined" size="12" />
                <n-text depth="3">{{ getRelativeTime(activity.timestamp) }}</n-text>
              </div>
            </div>
          </div>
        </template>
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
}

.activity-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.activity-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 4px;
  transition: background 0.2s;
}

.activity-item:hover {
  background: rgba(0, 0, 0, 0.3);
}

.activity-icon {
  flex-shrink: 0;
  padding-top: 2px;
}

.activity-content {
  flex: 1;
  min-width: 0;
}

.activity-action {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.activity-target {
  font-size: 12px;
  font-family: 'Fira Code', monospace;
}

.activity-details {
  display: block;
  font-size: 11px;
  margin-top: 2px;
}

.activity-time {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: #666;
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

