<script setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue';
import { NCard, NText, NSpace, NSpin } from 'naive-ui';
import { Chart, registerables } from 'chart.js';

// 註冊 Chart.js 所有元件
Chart.register(...registerables);

const props = defineProps({
  metricsHistory: {
    type: Array,
    default: () => []
  },
  title: {
    type: String,
    default: 'SYSTEM METRICS'
  }
});

const chartRef = ref(null);
let chartInstance = null;

// 圖表配置
const chartConfig = {
  type: 'line',
  data: {
    labels: [],
    datasets: [
      {
        label: 'CPU %',
        data: [],
        borderColor: '#18a058',
        backgroundColor: 'rgba(24, 160, 88, 0.1)',
        fill: true,
        tension: 0.4,
        yAxisID: 'y',
        pointRadius: 2,
        pointHoverRadius: 4
      },
      {
        label: 'RAM GB',
        data: [],
        borderColor: '#2080f0',
        backgroundColor: 'rgba(32, 128, 240, 0.1)',
        fill: true,
        tension: 0.4,
        yAxisID: 'y1',
        pointRadius: 2,
        pointHoverRadius: 4
      }
    ]
  },
  options: {
    responsive: true,
    maintainAspectRatio: false,
    interaction: {
      mode: 'index',
      intersect: false
    },
    plugins: {
      legend: {
        display: true,
        position: 'top',
        labels: {
          color: '#999',
          font: {
            family: "'Fira Code', monospace",
            size: 11
          },
          boxWidth: 12
        }
      },
      tooltip: {
        backgroundColor: '#1a1a20',
        titleColor: '#fff',
        bodyColor: '#ccc',
        borderColor: '#333',
        borderWidth: 1,
        titleFont: {
          family: "'Fira Code', monospace"
        },
        bodyFont: {
          family: "'Fira Code', monospace"
        }
      }
    },
    scales: {
      x: {
        display: true,
        grid: {
          color: 'rgba(255, 255, 255, 0.05)'
        },
        ticks: {
          color: '#666',
          font: {
            family: "'Fira Code', monospace",
            size: 10
          },
          maxRotation: 0,
          maxTicksLimit: 6
        }
      },
      y: {
        type: 'linear',
        display: true,
        position: 'left',
        min: 0,
        max: 100,
        grid: {
          color: 'rgba(255, 255, 255, 0.05)'
        },
        ticks: {
          color: '#18a058',
          font: {
            family: "'Fira Code', monospace",
            size: 10
          },
          callback: (value) => value + '%'
        },
        title: {
          display: true,
          text: 'CPU',
          color: '#18a058',
          font: {
            family: "'Fira Code', monospace",
            size: 10
          }
        }
      },
      y1: {
        type: 'linear',
        display: true,
        position: 'right',
        min: 0,
        grid: {
          drawOnChartArea: false
        },
        ticks: {
          color: '#2080f0',
          font: {
            family: "'Fira Code', monospace",
            size: 10
          },
          callback: (value) => value + 'G'
        },
        title: {
          display: true,
          text: 'RAM',
          color: '#2080f0',
          font: {
            family: "'Fira Code', monospace",
            size: 10
          }
        }
      }
    }
  }
};

// 初始化圖表
const initChart = () => {
  if (!chartRef.value) return;
  
  const ctx = chartRef.value.getContext('2d');
  chartInstance = new Chart(ctx, chartConfig);
};

// 更新圖表數據
const updateChart = () => {
  if (!chartInstance || !props.metricsHistory.length) return;
  
  const labels = props.metricsHistory.map((m, i) => {
    if (m.timestamp) {
      const date = new Date(m.timestamp * 1000);
      return date.toLocaleTimeString('zh-TW', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
    }
    return `T-${props.metricsHistory.length - i}`;
  });
  
  const cpuData = props.metricsHistory.map(m => m.cpu || 0);
  const ramData = props.metricsHistory.map(m => m.ram || 0);
  
  chartInstance.data.labels = labels;
  chartInstance.data.datasets[0].data = cpuData;
  chartInstance.data.datasets[1].data = ramData;
  chartInstance.update('none'); // 無動畫更新
};

// 監聽數據變化
watch(() => props.metricsHistory, () => {
  updateChart();
}, { deep: true });

onMounted(() => {
  initChart();
  updateChart();
});

onBeforeUnmount(() => {
  if (chartInstance) {
    chartInstance.destroy();
    chartInstance = null;
  }
});
</script>

<template>
  <n-card class="metrics-chart-card" size="small">
    <template #header>
      <n-space align="center" :size="8">
        <div class="chart-indicator"></div>
        <n-text strong class="chart-title">{{ title }}</n-text>
      </n-space>
    </template>
    
    <div class="chart-container">
      <div v-if="metricsHistory.length === 0" class="chart-empty">
        <n-spin size="small" />
        <n-text depth="3">等待數據...</n-text>
      </div>
      <canvas ref="chartRef"></canvas>
    </div>
  </n-card>
</template>

<style scoped>
.metrics-chart-card {
  background-color: #1a1a20 !important;
  border: 1px solid #333 !important;
}

.chart-title {
  font-family: 'Fira Code', monospace;
  font-size: 12px;
  letter-spacing: 1px;
}

.chart-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: linear-gradient(135deg, #18a058, #2080f0);
  animation: pulse-chart 2s ease-in-out infinite;
}

@keyframes pulse-chart {
  0%, 100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.6;
    transform: scale(0.8);
  }
}

.chart-container {
  position: relative;
  height: 180px;
}

.chart-empty {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: rgba(0, 0, 0, 0.3);
  z-index: 10;
}
</style>
