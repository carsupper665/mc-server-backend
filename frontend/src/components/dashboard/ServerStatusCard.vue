<script setup lang="ts">
defineProps<{
  id: string;
  name: string;
  status: 'Online' | 'Offline' | 'Starting' | string;
  players?: number;
  maxPlayers?: number;
}>();

defineEmits(['start', 'stop', 'manage']);
</script>

<template>
  <div class="group bg-zinc-900 border-2 border-zinc-800 p-6 transition-all hover:border-zinc-600 shadow-[4px_4px_0px_0px_rgba(39,39,42,1)] hover:shadow-[8px_8px_0px_0px_rgba(63,63,70,1)]">
    <div class="flex justify-between items-start mb-6">
      <div>
        <h3 class="text-xl font-bold tracking-tight text-zinc-100 mb-1">{{ name }}</h3>
        <p class="text-xs font-mono text-zinc-500 uppercase tracking-widest">Node: {{ id }}</p>
      </div>
      <div :class="{
        'bg-emerald-500/10 text-emerald-400 border-emerald-500/50': status === 'Online',
        'bg-rose-500/10 text-rose-400 border-rose-500/50': status === 'Offline',
        'bg-amber-500/10 text-amber-400 border-amber-500/50 animate-pulse': status === 'Starting'
      }" class="px-2 py-1 border text-[10px] font-black uppercase tracking-[0.2em]">
        {{ status }}
      </div>
    </div>

    <div class="flex items-center gap-4 mb-8">
      <div class="flex-1">
        <div class="flex justify-between text-[10px] font-mono uppercase text-zinc-500 mb-1">
          <span>Capacity</span>
          <span>{{ players || 0 }}/{{ maxPlayers || 20 }}</span>
        </div>
        <div class="h-1 bg-zinc-800 w-full overflow-hidden">
          <div class="h-full bg-zinc-400 transition-all duration-500" :style="{ width: `${((players || 0) / (maxPlayers || 20)) * 100}%` }"></div>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <button 
        v-if="status === 'Offline'"
        @click="$emit('start')"
        class="bg-zinc-100 text-zinc-950 py-2 font-black uppercase text-xs tracking-widest hover:bg-emerald-400 transition-colors"
      >
        Boot System
      </button>
      <button 
        v-else
        @click="$emit('stop')"
        class="border-2 border-zinc-700 text-zinc-300 py-2 font-black uppercase text-xs tracking-widest hover:border-rose-500 hover:text-rose-500 transition-all"
      >
        Terminate
      </button>
      <button 
        @click="$emit('manage')"
        class="border-2 border-zinc-700 text-zinc-300 py-2 font-black uppercase text-xs tracking-widest hover:border-zinc-100 hover:text-zinc-100 transition-all"
      >
        Control Panel
      </button>
    </div>
  </div>
</template>
