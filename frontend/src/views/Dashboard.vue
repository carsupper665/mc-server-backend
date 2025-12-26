<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { useRouter } from 'vue-router';
import { useServerStore } from '../stores/server';
import { useAuthStore } from '../stores/auth';
import ServerStatusCard from '../components/dashboard/ServerStatusCard.vue';
import CreateServerModal from '../components/dashboard/CreateServerModal.vue';
import { LogOut, Plus, Server } from 'lucide-vue-next';
import { minecraftApi } from '../api/minecraft';

const router = useRouter();
const serverStore = useServerStore();
const authStore = useAuthStore();
const showCreateModal = ref(false);

onMounted(() => {
  serverStore.fetchMyServers();
});

const handleLogout = () => {
  authStore.logout();
  router.push('/login');
};

const handleStart = async (id: string) => {
  await minecraftApi.startServer(id);
  serverStore.updateServerStatus(id);
};

const handleStop = async (id: string) => {
  await minecraftApi.stopServer(id);
  serverStore.updateServerStatus(id);
};

const manageServer = (id: string) => {
  router.push(`/server/${id}`);
};
</script>

<template>
  <div class="min-h-screen bg-zinc-950 text-zinc-100">
    <nav class="border-b border-zinc-800 bg-black/50 backdrop-blur-md sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-6 h-20 flex justify-between items-center">
        <div class="flex items-center gap-4">
          <div class="w-10 h-10 bg-zinc-100 flex items-center justify-center">
            <Server class="text-zinc-950 w-6 h-6" />
          </div>
          <div>
            <h2 class="font-black uppercase tracking-tighter text-xl italic leading-none">Command Deck</h2>
            <span class="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">Active Operations</span>
          </div>
        </div>
        
        <div class="flex items-center gap-6">
          <button @click="handleLogout" class="text-zinc-500 hover:text-rose-400 transition-colors flex items-center gap-2 font-mono text-xs uppercase tracking-widest">
            <LogOut class="w-4 h-4" />
            Disconnect
          </button>
        </div>
      </div>
    </nav>

    <main class="max-w-7xl mx-auto px-6 py-12">
      <header class="mb-12 flex justify-between items-end">
        <div>
          <h1 class="text-5xl font-black uppercase tracking-tighter mb-2 italic">Infrastructure</h1>
          <p class="text-zinc-500 font-mono text-sm uppercase tracking-widest">Manage your remote instances and services</p>
        </div>
        <button 
          @click="showCreateModal = true"
          class="bg-emerald-500 text-zinc-950 px-6 py-3 font-black uppercase text-sm tracking-widest flex items-center gap-2 hover:bg-emerald-400 transition-all shadow-[4px_4px_0px_0px_rgba(5,150,105,1)] active:translate-y-1 active:shadow-none"
        >
          <Plus class="w-5 h-5" />
          Deploy Node
        </button>
      </header>

      <!-- Modal -->
      <CreateServerModal 
        v-if="showCreateModal" 
        @close="showCreateModal = false" 
        @created="serverStore.fetchMyServers()" 
      />

      <div v-if="serverStore.loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
        <div v-for="i in 3" :key="i" class="h-64 bg-zinc-900 animate-pulse border-2 border-zinc-800"></div>
      </div>

      <div v-else-if="serverStore.servers.length === 0" class="border-2 border-dashed border-zinc-800 p-20 text-center">
        <p class="text-zinc-500 font-mono uppercase tracking-[0.2em]">No active server nodes detected in your cluster.</p>
      </div>

      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
        <ServerStatusCard 
          v-for="server in serverStore.servers" 
          :key="server.server_id"
          :id="server.server_id"
          :name="server.display_name"
          :status="serverStore.currentServerStatus[server.server_id]?.status || 'Offline'"
          :players="serverStore.currentServerStatus[server.server_id]?.onlinePlayers"
          :maxPlayers="serverStore.currentServerStatus[server.server_id]?.maxPlayers"
          @start="handleStart(server.server_id)"
          @stop="handleStop(server.server_id)"
          @manage="manageServer(server.server_id)"
        />
      </div>
    </main>
  </div>
</template>
