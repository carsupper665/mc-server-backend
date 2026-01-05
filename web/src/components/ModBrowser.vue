<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { 
  NSpace, NInput, NSelect, NButton, NIcon, NSpin, NPagination,
  NEmpty, NGrid, NGi, NInputGroup, useMessage
} from 'naive-ui';
import { SearchOutlined, ReloadOutlined, FilterOutlined } from '@vicons/antd';
import { searchMods, getGameVersions } from '../api/modrinth';
import ModCard from './ModCard.vue';

const message = useMessage();
const emit = defineEmits(['select-mod']);

// 搜尋狀態
const query = ref('');
const loading = ref(false);
const mods = ref([]);
const totalHits = ref(0);
const currentPage = ref(1);
const pageSize = ref(20);

// 過濾器
const selectedLoaders = ref([]);
const selectedVersion = ref(null);
const sortBy = ref('relevance');

// 版本選項 (動態載入)
const gameVersions = ref([]);
const loadingVersions = ref(false);

// 載入器選項
const loaderOptions = [
  { label: 'Fabric', value: 'fabric' },
  { label: 'Forge', value: 'forge' },
  { label: 'NeoForge', value: 'neoforge' },
  { label: 'Quilt', value: 'quilt' }
];

// 排序選項
const sortOptions = [
  { label: '相關性', value: 'relevance' },
  { label: '下載數', value: 'downloads' },
  { label: '追蹤數', value: 'follows' },
  { label: '最新', value: 'newest' },
  { label: '最近更新', value: 'updated' }
];

// 版本選項 (計算)
const versionOptions = computed(() => {
  return gameVersions.value
    .filter(v => v.version_type === 'release')
    .map(v => ({ label: v.version, value: v.version }));
});

// 搜尋模組
const handleSearch = async () => {
  loading.value = true;
  try {
    const result = await searchMods({
      query: query.value,
      loaders: selectedLoaders.value,
      versions: selectedVersion.value ? [selectedVersion.value] : [],
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value,
      index: sortBy.value
    });
    
    mods.value = result.hits || [];
    totalHits.value = result.total_hits || 0;
  } catch (err) {
    console.error('Search failed:', err);
    message.error('搜尋模組失敗，請稍後再試');
    mods.value = [];
    totalHits.value = 0;
  } finally {
    loading.value = false;
  }
};

// 載入遊戲版本
const loadGameVersions = async () => {
  loadingVersions.value = true;
  try {
    gameVersions.value = await getGameVersions();
  } catch (err) {
    console.error('Failed to load game versions:', err);
  } finally {
    loadingVersions.value = false;
  }
};

// 重置過濾器
const resetFilters = () => {
  query.value = '';
  selectedLoaders.value = [];
  selectedVersion.value = null;
  sortBy.value = 'relevance';
  currentPage.value = 1;
  handleSearch();
};

// 處理分頁
const handlePageChange = (page) => {
  currentPage.value = page;
  handleSearch();
};

// 處理卡片點擊
const handleModClick = (mod) => {
  emit('select-mod', mod);
  // 開啟 Modrinth 頁面
  window.open(`https://modrinth.com/mod/${mod.slug}`, '_blank');
};

// 監聽過濾器變化
watch([selectedLoaders, selectedVersion, sortBy], () => {
  currentPage.value = 1;
  handleSearch();
});

// 初始化
onMounted(() => {
  loadGameVersions();
  handleSearch();
});
</script>

<template>
  <div class="mod-browser">
    <div class="search-section">
      <n-input-group>
        <n-input
          v-model:value="query"
          placeholder="搜尋模組..."
          clearable
          @keyup.enter="handleSearch"
        >
          <template #prefix>
            <n-icon><SearchOutlined /></n-icon>
          </template>
        </n-input>
        <n-button type="primary" @click="handleSearch" :loading="loading">
          搜尋
        </n-button>
      </n-input-group>
      
      <div class="filters">
        <n-space :size="12" align="center" wrap>
          <n-select
            v-model:value="selectedLoaders"
            :options="loaderOptions"
            placeholder="載入器"
            clearable
            multiple
            style="min-width: 180px;"
          />
          
          <n-select
            v-model:value="selectedVersion"
            :options="versionOptions"
            :loading="loadingVersions"
            placeholder="遊戲版本"
            clearable
            filterable
            style="min-width: 140px;"
          />
          
          <n-select
            v-model:value="sortBy"
            :options="sortOptions"
            style="min-width: 120px;"
          />
          
          <n-button ghost @click="resetFilters">
            <template #icon><n-icon><ReloadOutlined /></n-icon></template>
            重置
          </n-button>
        </n-space>
      </div>
    </div>
    
    <div class="results-info" v-if="!loading && totalHits > 0">
      <span class="total-count">找到 {{ totalHits.toLocaleString() }} 個模組</span>
    </div>
    
    <n-spin :show="loading" class="content-spin">
      <div class="mods-grid" v-if="mods.length > 0">
        <ModCard
          v-for="mod in mods"
          :key="mod.project_id"
          :mod="mod"
          @click="handleModClick"
        />
      </div>
      
      <n-empty 
        v-else-if="!loading" 
        description="沒有找到模組，試試其他關鍵字吧！"
        class="empty-state"
      />
    </n-spin>
    
    <div class="pagination-wrapper" v-if="totalHits > pageSize">
      <n-pagination
        v-model:page="currentPage"
        :page-count="Math.ceil(totalHits / pageSize)"
        :page-slot="5"
        @update:page="handlePageChange"
      />
    </div>
  </div>
</template>

<style scoped>
.mod-browser {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.search-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
  background: #1a1a20;
  padding: 20px;
  border-radius: 12px;
  border: 1px solid #333;
}

.filters {
  padding-top: 8px;
}

.results-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.total-count {
  color: #888;
  font-size: 14px;
  font-family: 'Fira Code', monospace;
}

.content-spin {
  min-height: 200px;
}

.mods-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(400px, 1fr));
  gap: 16px;
}

.empty-state {
  padding: 60px 20px;
}

.pagination-wrapper {
  display: flex;
  justify-content: center;
  padding: 20px 0;
}

/* Responsive */
@media (max-width: 768px) {
  .mods-grid {
    grid-template-columns: 1fr;
  }
  
  .filters :deep(.n-space) {
    flex-direction: column;
    align-items: stretch;
  }
  
  .filters :deep(.n-select) {
    width: 100% !important;
    min-width: unset !important;
  }
}
</style>
