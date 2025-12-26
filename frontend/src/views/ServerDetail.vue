<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import { minecraftApi } from '../api/minecraft';
import ConsoleViewer from '../components/terminal/ConsoleViewer.vue';
import PropertyVisualEditor from '../components/dashboard/PropertyVisualEditor.vue';
import ResourceChart from '../components/dashboard/ResourceChart.vue';
import { ChevronLeft, Settings, Database, Activity, Save, FileText, Cpu, HardDrive } from 'lucide-vue-next';

const route = useRoute();
const serverId = route.params.id as string;

const MAX_LOG_LINES = 500; // Limit the number of log lines to prevent performance issues
const logs = ref<string[]>([]);
const properties = ref<string>('');
const status = ref<any>(null);
const cpuHistory = ref<number[]>(new Array(20).fill(0));
const ramHistory = ref<number[]>(new Array(20).fill(0));
const activeTab = ref<'console' | 'properties' | 'backups'>('console');
const propertyViewMode = ref<'visual' | 'raw'>('visual');
const loadingLogs = ref(false);
const savingProps = ref(false);

const fetchData = async () => {
  try {
    const statusRes = await minecraftApi.getStatus(serverId);
    status.value = statusRes.data;

    if (status.value?.status !== 'stopped') {
      const logsRes = await minecraftApi.getLogs(serverId);
      logs.value = logsRes.data?.logs || [];
      if (logs.value.length > MAX_LOG_LINES) {
        logs.value = logs.value.slice(-MAX_LOG_LINES);
      }
    }
    
    // Simulate metrics if not provided by backend
    if (statusRes.data?.cpuUsage !== undefined || statusRes.data?.ramUsage !== undefined) {
      const newCpu = [...cpuHistory.value, statusRes.data?.cpuUsage || 0];
      const newRam = [...ramHistory.value, statusRes.data?.ramUsage || 0];
      cpuHistory.value = newCpu.slice(-20);
      ramHistory.value = newRam.slice(-20);
    } else {
      const newCpu = [...cpuHistory.value, 0];
      const newRam = [...ramHistory.value, 0];
      cpuHistory.value = newCpu.slice(-20);
      ramHistory.value = newRam.slice(-20);
    }
  } catch (err) {
    console.error('Fetch error', err);
  }
};

const handleCommand = async (cmd: string) => {
  try {
    await minecraftApi.sendCommand(serverId, cmd);
    // Optimistic log entry with special marker
    logs.value.push(`USER_COMMAND: ${cmd}`);
    if (logs.value.length > MAX_LOG_LINES) {
      logs.value.shift(); // Remove the oldest log if exceeding the limit
    }
  } catch (err) {
    logs.value.push(`ERROR: Failed to send command`);
    if (logs.value.length > MAX_LOG_LINES) {
      logs.value.shift(); // Remove the oldest log if exceeding the limit
    }
  }
};

const loadProperties = async () => {
  try {
    const res = await minecraftApi.getProperties(serverId);
    properties.value = res.data.property || '';
  } catch (err) {
    console.error('Props error', err);
  }
};

const saveProperties = async (newContent?: string | Event) => {
  const contentToSave = typeof newContent === 'string' ? newContent : properties.value;
  savingProps.value = true;
  try {
    await minecraftApi.updateProperties(serverId, { texts: contentToSave });
    properties.value = contentToSave;
    alert('Properties updated successfully');
  } catch (err) {
    alert('Failed to save properties');
  } finally {
    savingProps.value = false;
  }
};

let pollInterval: any;

onMounted(() => {
  fetchData();
  pollInterval = setInterval(fetchData, 5000);
});

onUnmounted(() => {
  clearInterval(pollInterval);
});
</script>

<template>
  <div class="min-h-screen bg-zinc-950 text-zinc-100 flex flex-col">
    <nav class="border-b border-zinc-800 bg-black/50 backdrop-blur-md h-16 flex items-center px-6">
      <router-link to="/dashboard" class="mr-6 text-zinc-500 hover:text-zinc-100 transition-colors">
        <ChevronLeft />
      </router-link>
      <div class="flex items-center gap-3">
        <h2 class="font-black uppercase tracking-tighter italic">Server Node: {{ serverId }}</h2>
        <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]"></span>
      </div>
    </nav>

    <div class="flex-1 flex flex-col lg:flex-row">
      <!-- Sidebar -->
      <aside class="w-full lg:w-64 border-r border-zinc-800 p-4 space-y-2">
        <button 
          @click="activeTab = 'console'"
          :class="activeTab === 'console' ? 'bg-zinc-100 text-zinc-950' : 'text-zinc-500 hover:bg-zinc-900'"
          class="w-full flex items-center gap-3 px-4 py-3 font-black uppercase text-xs tracking-widest transition-all"
        >
          <Activity class="w-4 h-4" />
          Live Console
        </button>
        <button 
          @click="activeTab = 'properties'; loadProperties()"
          :class="activeTab === 'properties' ? 'bg-zinc-100 text-zinc-950' : 'text-zinc-500 hover:bg-zinc-900'"
          class="w-full flex items-center gap-3 px-4 py-3 font-black uppercase text-xs tracking-widest transition-all"
        >
          <Settings class="w-4 h-4" />
          Config Editor
        </button>
        <button 
          @click="activeTab = 'backups'"
          :class="activeTab === 'backups' ? 'bg-zinc-100 text-zinc-950' : 'text-zinc-500 hover:bg-zinc-900'"
          class="w-full flex items-center gap-3 px-4 py-3 font-black uppercase text-xs tracking-widest transition-all"
        >
          <Database class="w-4 h-4" />
          Data Backups
        </button>
      </aside>

      <!-- Content -->
      <main class="flex-1 p-8 bg-[url('https://www.transparenttextures.com/patterns/carbon-fibre.png')]">
        <div v-if="activeTab === 'console'" class="max-w-6xl mx-auto space-y-6">
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            <div class="bg-zinc-900 border-2 border-zinc-800 p-4 relative overflow-hidden group">
              <div class="absolute right-0 top-0 p-2 opacity-10 group-hover:opacity-20 transition-opacity">
                <Activity class="w-12 h-12" />
              </div>
              <p class="text-[10px] font-mono uppercase text-zinc-500 mb-1 tracking-widest">Node Status</p>
              <p class="text-2xl font-black uppercase text-emerald-400 italic tracking-tighter">{{ status?.status || 'Unknown' }}</p>
            </div>
            
            <div class="bg-zinc-900 border-2 border-zinc-800 p-4 relative overflow-hidden group">
              <div class="absolute right-0 top-0 p-2 opacity-10 group-hover:opacity-20 transition-opacity">
                <Database class="w-12 h-12" />
              </div>
              <p class="text-[10px] font-mono uppercase text-zinc-500 mb-1 tracking-widest">Active Players</p>
              <p class="text-2xl font-black uppercase text-zinc-100 italic tracking-tighter">{{ status?.onlinePlayers || 0 }} <span class="text-xs text-zinc-600">/ {{ status?.maxPlayers || 0 }}</span></p>
            </div>

            <div class="bg-zinc-900 border-2 border-zinc-800 p-4 flex flex-col h-24">
              <div class="flex justify-between items-center mb-1">
                <p class="text-[10px] font-mono uppercase text-zinc-500 tracking-widest flex items-center gap-1">
                  <Cpu class="w-3 h-3" /> CPU Load
                </p>
                <span class="text-[10px] font-mono text-emerald-500">{{ cpuHistory[cpuHistory.length-1] }}%</span>
              </div>
              <div class="flex-1 min-h-0">
                <ResourceChart :data="cpuHistory" label="CPU" color="#10b981" />
              </div>
            </div>

            <div class="bg-zinc-900 border-2 border-zinc-800 p-4 flex flex-col h-24">
              <div class="flex justify-between items-center mb-1">
                <p class="text-[10px] font-mono uppercase text-zinc-500 tracking-widest flex items-center gap-1">
                  <HardDrive class="w-3 h-3" /> RAM Usage
                </p>
                <span class="text-[10px] font-mono text-blue-500">{{ ramHistory[ramHistory.length-1] }}%</span>
              </div>
              <div class="flex-1 min-h-0">
                <ResourceChart :data="ramHistory" label="RAM" color="#3b82f6" />
              </div>
            </div>
          </div>
          
          <ConsoleViewer :logs="logs" :loading="loadingLogs" @command="handleCommand" />
        </div>

        <div v-if="activeTab === 'properties'" class="max-w-5xl mx-auto">
          <div class="flex flex-col md:flex-row justify-between items-start md:items-center gap-4 mb-8">
            <div>
              <h3 class="text-3xl font-black uppercase italic tracking-tighter text-zinc-100 leading-none mb-2">Configuration</h3>
              <p class="text-[10px] font-mono text-zinc-500 uppercase tracking-[0.2em]">Editing server.properties</p>
            </div>
            
            <div class="flex bg-zinc-900 border border-zinc-800 p-1">
              <button 
                @click="propertyViewMode = 'visual'"
                :class="propertyViewMode === 'visual' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'"
                class="px-4 py-1.5 text-[10px] font-black uppercase tracking-widest transition-all flex items-center gap-2"
              >
                <Settings class="w-3 h-3" /> Visual
              </button>
              <button 
                @click="propertyViewMode = 'raw'"
                :class="propertyViewMode === 'raw' ? 'bg-zinc-800 text-zinc-100' : 'text-zinc-500 hover:text-zinc-300'"
                class="px-4 py-1.5 text-[10px] font-black uppercase tracking-widest transition-all flex items-center gap-2"
              >
                <FileText class="w-3 h-3" /> Raw Editor
              </button>
            </div>
          </div>

          <div v-if="propertyViewMode === 'visual'">
            <PropertyVisualEditor 
              :raw-properties="properties" 
              :saving="savingProps" 
              @save="saveProperties" 
            />
          </div>

          <div v-else class="space-y-4">
            <div class="bg-black border-2 border-zinc-800 p-1">
              <textarea 
                v-model="properties"
                class="w-full h-[600px] bg-zinc-950 text-zinc-300 font-mono text-sm p-6 outline-none focus:bg-black transition-colors resize-none custom-scrollbar"
                placeholder="# Minecraft server properties..."
              ></textarea>
            </div>
            <div class="flex justify-end">
              <button 
                @click="saveProperties"
                :disabled="savingProps"
                class="bg-zinc-100 text-zinc-950 px-8 py-3 font-black uppercase text-xs tracking-[0.2em] hover:bg-emerald-400 flex items-center gap-3 transition-all shadow-[4px_4px_0px_0px_rgba(24,24,27,1)] active:translate-y-1 active:shadow-none"
              >
                <Save class="w-4 h-4" />
                {{ savingProps ? 'Writing to disk...' : 'Save Raw Configuration' }}
              </button>
            </div>
          </div>
        </div>

        <div v-if="activeTab === 'backups'" class="max-w-4xl mx-auto">
          <div class="border-2 border-dashed border-zinc-800 p-20 text-center">
            <Database class="w-12 h-12 text-zinc-800 mx-auto mb-4" />
            <p class="text-zinc-500 font-mono uppercase tracking-[0.2em]">Backup subsystem initializing...</p>
          </div>
        </div>
      </main>
    </div>
  </div>
</template>
