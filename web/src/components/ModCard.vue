<script setup>
import { computed } from 'vue';
import { NCard, NTag, NSpace, NIcon, NAvatar, NText, NEllipsis } from 'naive-ui';
import { DownloadOutlined, HeartOutlined } from '@vicons/antd';
import { formatDownloads } from '../api/modrinth';

const props = defineProps({
  mod: {
    type: Object,
    required: true
  }
});

const emit = defineEmits(['click']);

// 格式化下載數
const downloads = computed(() => formatDownloads(props.mod.downloads || 0));

// 格式化追蹤數
const follows = computed(() => formatDownloads(props.mod.follows || 0));

// 取得主要載入器標籤
const loaderTags = computed(() => {
  const categories = props.mod.categories || [];
  const loaders = ['fabric', 'forge', 'neoforge', 'quilt'];
  return categories.filter(c => loaders.includes(c.toLowerCase()));
});

// 取得其他類別標籤 (最多顯示 2 個)
const categoryTags = computed(() => {
  const categories = props.mod.display_categories || props.mod.categories || [];
  const loaders = ['fabric', 'forge', 'neoforge', 'quilt'];
  return categories.filter(c => !loaders.includes(c.toLowerCase())).slice(0, 2);
});

// 最新支援的版本
const latestVersion = computed(() => {
  const versions = props.mod.versions || [];
  return versions.length > 0 ? versions[versions.length - 1] : null;
});

const handleClick = () => {
  emit('click', props.mod);
};
</script>

<template>
  <n-card 
    class="mod-card" 
    hoverable 
    :bordered="false"
    @click="handleClick"
  >
    <div class="mod-content">
      <n-avatar
        :src="mod.icon_url"
        :size="64"
        :fallback-src="'https://cdn.modrinth.com/placeholder.svg'"
        class="mod-icon"
        object-fit="cover"
      />
      
      <div class="mod-info">
        <div class="mod-header">
          <h3 class="mod-title">{{ mod.title }}</h3>
          <n-text depth="3" class="mod-author">by {{ mod.author }}</n-text>
        </div>
        
        <n-ellipsis :line-clamp="2" class="mod-description">
          {{ mod.description }}
        </n-ellipsis>
        
        <div class="mod-meta">
          <n-space :size="8" align="center">
            <!-- 載入器標籤 -->
            <n-tag 
              v-for="loader in loaderTags" 
              :key="loader"
              :type="loader === 'fabric' ? 'info' : loader === 'forge' ? 'warning' : 'success'"
              size="small"
              round
            >
              {{ loader }}
            </n-tag>
            
            <!-- 類別標籤 -->
            <n-tag 
              v-for="cat in categoryTags" 
              :key="cat"
              type="default"
              size="small"
              round
            >
              {{ cat }}
            </n-tag>
          </n-space>
          
          <n-space :size="16" align="center" class="mod-stats">
            <span class="stat-item">
              <n-icon :size="14"><DownloadOutlined /></n-icon>
              {{ downloads }}
            </span>
            <span class="stat-item">
              <n-icon :size="14"><HeartOutlined /></n-icon>
              {{ follows }}
            </span>
            <n-tag v-if="latestVersion" size="small" type="success" round>
              {{ latestVersion }}
            </n-tag>
          </n-space>
        </div>
      </div>
    </div>
  </n-card>
</template>

<style scoped>
.mod-card {
  background-color: #1a1a20;
  border: 1px solid #333;
  border-radius: 12px;
  transition: all 0.3s ease;
  cursor: pointer;
}

.mod-card:hover {
  border-color: #18a058;
  box-shadow: 0 8px 24px rgba(24, 160, 88, 0.15);
  transform: translateY(-4px);
}

.mod-content {
  display: flex;
  gap: 16px;
}

.mod-icon {
  flex-shrink: 0;
  border-radius: 12px;
  background-color: #2a2a30;
}

.mod-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mod-header {
  display: flex;
  align-items: baseline;
  gap: 8px;
  flex-wrap: wrap;
}

.mod-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
  font-family: 'Fira Code', monospace;
}

.mod-author {
  font-size: 12px;
}

.mod-description {
  font-size: 13px;
  color: #aaa;
  line-height: 1.5;
}

.mod-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: auto;
}

.mod-stats {
  color: #888;
  font-size: 12px;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

/* Responsive */
@media (max-width: 640px) {
  .mod-content {
    flex-direction: column;
    align-items: center;
    text-align: center;
  }
  
  .mod-header {
    justify-content: center;
  }
  
  .mod-meta {
    flex-direction: column;
    align-items: center;
  }
}
</style>
