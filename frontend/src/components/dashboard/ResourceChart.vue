<script setup lang="ts">
import { computed } from 'vue';
import { Line } from 'vue-chartjs';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
} from 'chart.js';

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

const props = defineProps<{
  data: number[];
  label: string;
  color: string;
}>();

const chartData = computed(() => ({
  labels: new Array(props.data.length).fill(''),
  datasets: [
    {
      label: props.label,
      data: props.data,
      borderColor: props.color,
      backgroundColor: `${props.color}22`,
      borderWidth: 2,
      pointRadius: 0,
      tension: 0.4,
      fill: true,
    }
  ]
}));

const chartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: { enabled: true }
  },
  scales: {
    x: { display: false },
    y: {
      beginAtZero: true,
      max: 100,
      grid: {
        color: '#27272a',
        drawBorder: false,
      },
      ticks: {
        color: '#52525b',
        font: { family: 'JetBrains Mono', size: 10 }
      }
    }
  }
};
</script>

<template>
  <div class="h-full w-full">
    <Line :data="chartData" :options="chartOptions" />
  </div>
</template>
