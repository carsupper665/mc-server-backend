<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue';
import { 
  NSpace, NCard, NGrid, NGridItem, NButton, NText, NTag, NIcon,
  NModal, NInput, NSpin, NEmpty, NCollapse, NCollapseItem, useMessage
} from 'naive-ui';
import { 
  PlusOutlined, 
  TeamOutlined, 
  CloseCircleOutlined,
  ReloadOutlined,
  UserOutlined
} from '@vicons/antd';
import api from '../api';
import { useActivityLogStore } from '../store/activityLog';

const message = useMessage();
const activityLog = useActivityLogStore();
const loading = ref(true);
const rooms = ref([]);
const expandedRooms = ref({});
const creatingRoom = ref(false);

// 獲取房間列表
const fetchRooms = async () => {
  loading.value = true;
  try {
    const res = await api.get('/LOL-AmongUs/a/ls');
    
    rooms.value = (res?.games || []).map(id => ({
      id,
      name: `Room #${id.substring(0, 6)}...`
    }));
  } catch (err) {
    // 靜默處理錯誤
    rooms.value = [];
  } finally {
    loading.value = false;
  }
};

// 獲取房間玩家
const fetchPlayers = async (roomId) => {
  if (!roomId || roomId === 'unknown') return [];
  try {
    const res = await api.get(`/LOL-AmongUs/a/ls-p/${roomId}`);
    
    return Array.isArray(res?.message) ? res.message.map(player => ({ ID: player.ID })) : [];
  } catch (err) {
    return [];
  }
};

// 展開/收合房間
const toggleRoom = async (room) => {
  const id = room.id;
  if (!id || id === 'unknown') return;
  
  if (expandedRooms.value[id]) {
    delete expandedRooms.value[id];
  } else {
    expandedRooms.value[id] = { loading: true, players: [] };
    const players = await fetchPlayers(id);
    expandedRooms.value[id] = { loading: false, players };
  }
};

// 快速建立房間
const quickCreate = async () => {
  creatingRoom.value = true;
  try {
    const res = await api.get('/LOL-AmongUs/a/c');
    message.success('房間已建立！');
    activityLog.logAmongUsCreate(res?.game_id || res?.id || 'new');
    setTimeout(() => fetchRooms(), 500);
  } catch (err) {
    message.error('建立房間失敗: ' + (err.response?.data?.error || err.message || '未知錯誤'));
  } finally {
    creatingRoom.value = false;
  }
};

// 結束遊戲
const endGame = async (roomId) => {
  if (!roomId || roomId === 'unknown') {
    message.warning('無法結束：無效的房間 ID');
    return;
  }
  try {
    await api.get(`/LOL-AmongUs/a/end/${roomId}`);
    message.success('遊戲已結束');
    activityLog.logAmongUsEnd(roomId);
    await fetchRooms();
  } catch (err) {
    message.error('結束遊戲失敗');
  }
};

// 自動輪詢 - 30 秒間隔以避免 429 錯誤
let pollInterval = null;

onMounted(() => {
  fetchRooms();
  pollInterval = setInterval(fetchRooms, 30000);
});

onBeforeUnmount(() => {
  if (pollInterval) clearInterval(pollInterval);
});
</script>

<template>
  <div class="among-us-view">
    <n-space vertical :size="24">
      <!-- 標題區 -->
      <div class="header-section fade-in-up">
        <n-space justify="space-between" align="center">
          <div class="title-area">
            <h2 class="page-title">
              <span class="title-icon">🚀</span>
              AMONG US LOBBY
            </h2>
            <n-text depth="3" class="subtitle">管理 Among Us 遊戲房間</n-text>
          </div>
          
          <n-space :size="12">
            <n-button 
              quaternary 
              circle 
              @click="fetchRooms"
              :disabled="loading"
            >
              <template #icon><n-icon><ReloadOutlined /></n-icon></template>
            </n-button>
            <n-button 
              type="primary"
              :loading="creatingRoom"
              @click="quickCreate"
            >
              <template #icon><n-icon><PlusOutlined /></n-icon></template>
              QUICK CREATE
            </n-button>
          </n-space>
        </n-space>
      </div>
      
      <!-- 載入中 -->
      <div v-if="loading" class="loading-area">
        <n-spin size="large" />
        <n-text depth="3">正在載入房間列表...</n-text>
      </div>
      
      <!-- 無房間 -->
      <n-card v-else-if="rooms.length === 0" class="empty-card fade-in-up">
        <n-empty description="目前沒有進行中的遊戲">
          <template #extra>
            <n-button type="primary" @click="quickCreate" :loading="creatingRoom">
              建立第一個房間
            </n-button>
          </template>
        </n-empty>
      </n-card>
      
      <!-- 房間卡片網格 -->
      <n-grid v-else cols="1 s:2 m:3" responsive="screen" :x-gap="20" :y-gap="20">
        <n-grid-item v-for="room in rooms" :key="room.id">
          <n-card 
            class="room-card fade-in-up"
            :class="{ 'room-expanded': expandedRooms[room.id] }"
            hoverable
          >
            <template #header>
              <n-space align="center" :size="8">
                <div class="room-avatar">
                  <n-icon size="20" color="#818cf8"><TeamOutlined /></n-icon>
                </div>
                <n-text strong class="room-name">{{ room.name }}</n-text>
              </n-space>
            </template>
            
            <template #header-extra>
              <n-tag v-if="expandedRooms[room.id] && !expandedRooms[room.id].loading" type="info" size="small" round>
                <template #icon>
                  <n-icon><UserOutlined /></n-icon>
                </template>
                {{ expandedRooms[room.id].players.length }} 玩家
              </n-tag>
            </template>
            
            <div class="room-content">
              <n-text depth="3" class="room-id">ID: {{ room.id }}</n-text>
              
              <!-- 展開的玩家列表 -->
              <div v-if="expandedRooms[room.id]" class="players-section">
                <n-spin v-if="expandedRooms[room.id].loading" size="small" />
                <template v-else>
                  <div 
                    v-for="player in expandedRooms[room.id].players"
                    :key="player.ID"
                    class="player-item"
                  >
                    <n-icon size="14"><UserOutlined /></n-icon>
                    <n-text>{{ player.ID }}</n-text>
                  </div>
                  <n-text v-if="expandedRooms[room.id].players.length === 0" depth="3">
                    目前無玩家
                  </n-text>
                </template>
              </div>
            </div>
            
            <template #footer>
              <n-space justify="space-between">
                <n-button 
                  size="small" 
                  quaternary
                  @click="toggleRoom(room)"
                >
                  {{ expandedRooms[room.id] ? '收合' : '查看玩家' }}
                </n-button>
                <n-button 
                  size="small" 
                  type="error" 
                  ghost
                  @click="endGame(room.id)"
                >
                  <template #icon><n-icon><CloseCircleOutlined /></n-icon></template>
                  結束
                </n-button>
              </n-space>
            </template>
          </n-card>
        </n-grid-item>
      </n-grid>
    </n-space>
  </div>
</template>

<style scoped>
.among-us-view {
  padding: 8px;
}

.header-section {
  margin-bottom: 8px;
}

.title-area {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.page-title {
  margin: 0;
  font-size: 24px;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 12px;
}

.title-icon {
  font-size: 28px;
}

.subtitle {
  font-family: 'Fira Code', monospace;
  font-size: 12px;
  letter-spacing: 0.5px;
}

.loading-area {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 80px 0;
}

.empty-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
}

/* Room Card - Among Us 風格 */
.room-card {
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%) !important;
  border: 1px solid rgba(99, 102, 241, 0.3) !important;
  transition: all 0.3s ease;
}

.room-card:hover {
  border-color: rgba(99, 102, 241, 0.6) !important;
  box-shadow: 0 0 20px rgba(99, 102, 241, 0.15);
  transform: translateY(-2px);
}

.room-card.room-expanded {
  border-color: rgba(99, 102, 241, 0.8) !important;
}

.room-avatar {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: rgba(99, 102, 241, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.room-name {
  font-family: 'Fira Code', monospace;
  letter-spacing: 0.5px;
}

.room-content {
  min-height: 40px;
}

.room-id {
  font-family: 'Fira Code', monospace;
  font-size: 11px;
}

.players-section {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.player-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 13px;
}

.player-item:not(:last-child) {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
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

@media (max-width: 768px) {
  .page-title {
    font-size: 20px;
  }
  
  .title-icon {
    font-size: 24px;
  }
}
</style>
