<script setup>
import { computed, ref, watch } from 'vue';
import { NButton, NIcon, NText, NTag } from 'naive-ui';
import { MinusOutlined, ExpandOutlined } from '@vicons/antd';

const props = defineProps({
  jobs: {
    type: Array,
    default: () => []
  }
});

const collapsed = ref(false);

const activeCount = computed(() => {
  return props.jobs.filter(job => job.status !== 'completed' && job.status !== 'failed').length;
});

watch(() => props.jobs.length, (len) => {
  if (len === 0) {
    collapsed.value = false;
  }
});

const toggle = () => {
  collapsed.value = !collapsed.value;
};

const stageLabel = (stage, status) => {
  if (status === 'failed') return '失敗';
  if (status === 'completed') return '完成';
  switch (stage) {
    case 'queued':
      console.log("queued")
      return '排隊中';
    case 'resolving':
      console.log("resolving")
      return '解析相依';
    case 'downloading':
      console.log('下載中')
      return '下載中';
    case 'installed':
      console.log('已安裝')
      return '已安裝';
    case 'skipped':
      console.log("skip");
      return '略過';
    case 'completed':
      console.log('completed')
      return '完成';
    default:
      return stage || '處理中';
  }
};

const stageClass = (stage, status) => {
  if (status === 'failed') return 'fail';
  if (status === 'completed') return 'done';
  switch (stage) {
    case 'downloading':
      return 'warn';
    case 'installed':
      return 'done';
    default:
      return 'info';
  }
};
</script>

<template>
  <div v-if="jobs.length" class="install-floating">
    <div v-if="collapsed" class="install-mini" @click="toggle">
      <n-icon :size="16"><ExpandOutlined /></n-icon>
      <n-text>MODS</n-text>
      <n-tag size="small" type="success" round>{{ activeCount }}</n-tag>
    </div>

    <div v-else class="install-panel">
      <div class="panel-header">
        <div class="title">
          <n-text strong>MOD INSTALLS</n-text>
          <n-tag size="small" type="success" round>{{ activeCount }}</n-tag>
        </div>
        <n-button size="tiny" quaternary @click="toggle">
          <template #icon><n-icon><MinusOutlined /></n-icon></template>
        </n-button>
      </div>

      <div class="panel-body">
        <div v-for="job in jobs" :key="job.jobId" class="job-card">
          <div class="job-header">
            <div class="job-summary">
              <img :src="job.modIcon" class="job-icon" alt="mod icon" />
              <div>
                <div class="job-title">{{ job.modTitle }}</div>
                <div class="job-sub">{{ job.serverName }} · {{ stageLabel(job.stage, job.status) }}</div>
              </div>
            </div>
            <n-tag size="small" :type="job.status === 'failed' ? 'error' : job.status === 'completed' ? 'success' : 'warning'" round>
              {{ job.status || 'running' }}
            </n-tag>
          </div>

          <div class="progress-bar" :class="stageClass(job.stage, job.status)">
            <div
              class="progress-fill"
              :class="{ indeterminate: job.percent === null || job.percent === undefined }"
              :style="job.percent !== null && job.percent !== undefined ? { width: `${job.percent}%` } : {}"
            ></div>
          </div>

          <div v-if="job.message" class="job-message">
            {{ job.message }}
          </div>
          <div v-if="job.error" class="job-error">
            {{ job.error }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.install-floating {
  position: fixed;
  right: 20px;
  bottom: 20px;
  z-NameIndex: 1000;
  font-family: 'Fira Code', monospace;
}

.install-mini {
  background: #1a1a20;
  border: 1px solid #333;
  border-radius: 999px;
  padding: 8px 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.35);
}

.install-panel {
  width: 340px;
  background: #1a1a20;
  border: 1px solid #333;
  border-radius: 12px;
  box-shadow: 0 16px 30px rgba(0, 0, 0, 0.35);
  overflow: hidden;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 12px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-body {
  max-height: 320px;
  overflow-y: auto;
  padding: 10px 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.job-card {
  background: rgba(0, 0, 0, 0.2);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 10px;
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.job-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 10px;
}

.job-summary {
  display: flex;
  gap: 10px;
  align-items: center;
}

.job-icon {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  object-fit: cover;
  background: #2a2a30;
}

.job-title {
  font-size: 12px;
  color: #fff;
}

.job-sub {
  font-size: 11px;
  color: #888;
}

.progress-bar {
  height: 6px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  overflow: hidden;
}

.progress-fill {
  height: 100%;
  width: 0;
  background: linear-gradient(90deg, #18a058, #37d67a);
  transition: width 0.25s ease;
}

.progress-fill.indeterminate {
  width: 100%;
  background-size: 200% 100%;
  animation: shimmer 1.4s linear infinite;
}

.progress-bar.warn .progress-fill {
  background: linear-gradient(90deg, #f2b705, #ffdc7a);
}

.progress-bar.fail .progress-fill {
  background: linear-gradient(90deg, #ef4444, #f87171);
}

.progress-bar.done .progress-fill {
  background: linear-gradient(90deg, #18a058, #37d67a);
}

.job-message {
  font-size: 11px;
  color: #c9c9c9;
}

.job-error {
  font-size: 11px;
  color: #f87171;
}

@keyframes shimmer {
  0% {
    background-position: 0% 50%;
  }
  100% {
    background-position: 200% 50%;
  }
}
</style>
