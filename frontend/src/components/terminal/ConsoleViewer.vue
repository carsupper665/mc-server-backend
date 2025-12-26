<script setup lang="ts">
import { ref, watch, computed, onMounted } from 'vue';
import { Terminal, Send, ChevronRight, Trash2, ArrowDown } from 'lucide-vue-next';

const props = defineProps<{
  logs: string[];
  loading?: boolean;
}>();

const emit = defineEmits(['command']);
const command = ref('');
const showSuggestions = ref(false);
const logContainer = ref<HTMLElement | null>(null);
const commandInput = ref<HTMLInputElement | null>(null);
const isUserScrolling = ref(false);
const localLogs = ref<string[]>([]);

// Command history state
const commandHistory = ref<string[]>([]);
const historyIndex = ref(-1);

const commonCommands = [
  { cmd: 'help', desc: 'Show all commands' },
  { cmd: 'stop', desc: 'Gracefully shutdown' },
  { cmd: 'list', desc: 'List online players' },
  { cmd: 'op ', desc: 'Grant operator status' },
  { cmd: 'deop ', desc: 'Revoke operator status' },
  { cmd: 'whitelist add ', desc: 'Add player to whitelist' },
  { cmd: 'whitelist remove ', desc: 'Remove player from whitelist' },
  { cmd: 'say ', desc: 'Broadcast message' },
  { cmd: 'save-all', desc: 'Force world save' },
  { cmd: 'clear', desc: 'Clear terminal view' },
];

const suggestions = computed(() => {
  if (!command.value.startsWith('/')) return [];
  const query = command.value.slice(1).toLowerCase();
  return commonCommands.filter(c => c.cmd.startsWith(query));
});

const selectSuggestion = (cmd: string) => {
  command.value = '/' + cmd;
  showSuggestions.value = false;
  commandInput.value?.focus();
};

const handleTab = (e: KeyboardEvent) => {
  const firstSuggestion = suggestions.value[0];
  if (firstSuggestion) {
    e.preventDefault();
    selectSuggestion(firstSuggestion.cmd);
  }
};

const scrollToBottom = (force = false) => {
  const container = logContainer.value;
  if (container && (force || !isUserScrolling.value)) {
    container.scrollTop = container.scrollHeight;
  }
};

const handleScroll = () => {
  const container = logContainer.value;
  if (!container) return;
  const { scrollTop, scrollHeight, clientHeight } = container;
  // If user is more than 50px away from bottom, consider them "scrolling up"
  isUserScrolling.value = scrollHeight - scrollTop - clientHeight > 50;
};

const clearView = () => {
  localLogs.value = [];
};

watch(() => props.logs, (newLogs) => {
  // Sync local logs with props, but allow local clear
  if (localLogs.value.length === 0 && newLogs.length > 0) {
    localLogs.value = [...newLogs];
  } else if (newLogs.length > localLogs.value.length) {
    // Only append new ones
    const diff = newLogs.slice(localLogs.value.length);
    localLogs.value.push(...diff);
  } else if (newLogs.length < localLogs.value.length) {
      // If props was truncated or reset, reset local
      localLogs.value = [...newLogs];
  }
  
  setTimeout(() => scrollToBottom(), 50);
}, { deep: true, immediate: true });

const handleSend = () => {
  if (!command.value.trim()) return;
  
  const rawCmd = command.value.trim();
  
  if (rawCmd === '/clear' || rawCmd === 'clear') {
      clearView();
      command.value = '';
      return;
  }
  
  // Save to history if not duplicate of last command
  if (commandHistory.value[0] !== rawCmd) {
    commandHistory.value.unshift(rawCmd);
    if (commandHistory.value.length > 50) commandHistory.value.pop();
  }
  historyIndex.value = -1;

  // Remove leading / if present for API consistency
  const finalCmd = rawCmd.startsWith('/') ? rawCmd.slice(1) : rawCmd;
  emit('command', finalCmd);
  command.value = '';
  showSuggestions.value = false;
  scrollToBottom(true);
};

const navigateHistory = (direction: 'up' | 'down') => {
  if (commandHistory.value.length === 0) return;

  if (direction === 'up') {
    if (historyIndex.value < commandHistory.value.length - 1) {
      historyIndex.value++;
      command.value = commandHistory.value[historyIndex.value] || '';
    }
  } else {
    if (historyIndex.value > 0) {
      historyIndex.value--;
      command.value = commandHistory.value[historyIndex.value] || '';
    } else {
      historyIndex.value = -1;
      command.value = '';
    }
  }
};

onMounted(() => {
    commandInput.value?.focus();
});
</script>

<template>
  <div 
    class="bg-[#0c0c0c] border-2 border-zinc-800 flex flex-col font-mono text-xs shadow-2xl h-[600px] relative overflow-hidden group/terminal"
    @click="commandInput?.focus()"
  >
    <!-- Scanline effect -->
    <div class="absolute inset-0 pointer-events-none bg-[linear-gradient(rgba(18,16,16,0)_50%,rgba(0,0,0,0.1)_50%),linear-gradient(90deg,rgba(255,0,0,0.02),rgba(0,255,0,0.01),rgba(0,0,255,0.02))] bg-[length:100%_2px,3px_100%] z-50 opacity-20"></div>

    <!-- Suggestions Overlay -->
    <div v-if="suggestions.length > 0" class="absolute bottom-16 left-12 right-12 bg-zinc-900 border-2 border-zinc-700 p-2 z-[60] shadow-2xl backdrop-blur-md bg-opacity-90">
      <div v-for="s in suggestions" :key="s.cmd" 
           @click.stop="selectSuggestion(s.cmd)"
           class="flex justify-between items-center px-3 py-2 hover:bg-emerald-500/20 hover:border-l-2 hover:border-emerald-500 cursor-pointer group transition-all">
        <div class="flex items-center gap-2">
          <ChevronRight class="w-3 h-3 text-emerald-500" />
          <span class="text-zinc-200 font-bold">/{{ s.cmd }}</span>
        </div>
        <span class="text-zinc-600 text-[10px] uppercase group-hover:text-zinc-300">{{ s.desc }}</span>
      </div>
    </div>

    <!-- Scroll to bottom button -->
    <button 
        v-if="isUserScrolling"
        @click.stop="scrollToBottom(true)"
        class="absolute bottom-20 right-8 bg-emerald-500 text-zinc-950 px-4 py-2 rounded-full flex items-center gap-2 text-[10px] font-black uppercase tracking-widest shadow-lg hover:bg-emerald-400 transition-all z-50 animate-bounce"
    >
        <ArrowDown class="w-3 h-3" />
        New Activity
    </button>

    <div class="bg-[#1a1a1a] px-4 py-3 border-b border-zinc-800 flex justify-between items-center relative z-10">
      <div class="flex items-center gap-2">
        <div class="p-1 bg-zinc-800 rounded">
            <Terminal class="w-3 h-3 text-emerald-500" />
        </div>
        <div class="flex flex-col">
            <span class="text-zinc-200 font-black uppercase tracking-widest text-[9px] leading-none">System Virtual Terminal</span>
            <span class="text-zinc-600 text-[7px] uppercase tracking-[0.2em]">Connection: Secure_SSH_v2</span>
        </div>
      </div>
      <div class="flex items-center gap-4">
        <button 
            @click.stop="clearView"
            class="text-zinc-600 hover:text-rose-400 transition-colors flex items-center gap-1 group/btn"
            title="Clear Terminal (Local)"
        >
            <span class="text-[8px] font-black uppercase opacity-0 group-hover/btn:opacity-100 transition-opacity">Clear</span>
            <Trash2 class="w-3 h-3" />
        </button>
        <div class="flex gap-1.5">
          <div class="w-2.5 h-2.5 rounded-full bg-zinc-800 border border-zinc-700"></div>
          <div class="w-2.5 h-2.5 rounded-full bg-zinc-800 border border-zinc-700"></div>
          <div class="w-2.5 h-2.5 rounded-full bg-zinc-800 border border-zinc-700"></div>
        </div>
      </div>
    </div>
    
    <div 
      ref="logContainer"
      @scroll="handleScroll"
      class="flex-1 overflow-y-auto p-6 space-y-1.5 custom-scrollbar bg-[radial-gradient(#1a1a1a_1px,transparent_1px)] bg-[size:30px_30px] relative z-10"
    >
      <div v-if="loading" class="text-emerald-500/50 flex items-center gap-2">
          <span class="animate-spin w-3 h-3 border-2 border-emerald-500 border-t-transparent rounded-full"></span>
          Initializing secure stream...
      </div>
      <div v-for="(log, i) in localLogs" :key="i" class="flex gap-4 group/line">
        <span class="text-zinc-800 select-none w-10 text-right font-mono text-[9px] pt-0.5">{{ i + 1 }}</span>
        <div class="flex-1 text-[#d4d4d4] break-all leading-relaxed">
          <template v-if="log.startsWith('USER_COMMAND: ')">
            <div class="flex items-center gap-2 bg-emerald-500/5 py-0.5 px-2 -ml-2 border-l-2 border-emerald-500">
                <span class="text-emerald-500 font-bold">$</span>
                <span class="text-emerald-400 font-bold">{{ log.replace('USER_COMMAND: ', '') }}</span>
            </div>
          </template>
          <template v-else>
            <span v-if="log.includes('INFO')" class="text-blue-400 font-bold">[INFO]</span>
            <span v-else-if="log.includes('WARN')" class="text-amber-400 font-bold">[WARN]</span>
            <span v-else-if="log.includes('ERROR')" class="text-rose-500 font-bold">[ERROR]</span>
            <span class="ml-1">{{ log.replace(/\[(INFO|WARN|ERROR)\]/, '') }}</span>
          </template>
        </div>
      </div>
      <!-- Virtual cursor at end of logs if focused -->
      <div v-if="localLogs.length > 0" class="h-4"></div>
    </div>

    <div class="p-4 bg-[#141414] border-t border-zinc-800 flex items-center gap-3 relative z-10">
      <div class="flex items-center gap-2 group/prompt">
          <span class="text-emerald-500 font-black tracking-tighter">admin@node-01</span>
          <span class="text-zinc-600 font-bold">:</span>
          <span class="text-blue-500 font-bold">~</span>
          <span class="text-zinc-400 font-bold">$</span>
      </div>
      <div class="flex-1 relative flex items-center">
          <input 
            ref="commandInput"
            v-model="command" 
            @keyup.enter="handleSend"
            @keyup.up.prevent="navigateHistory('up')"
            @keyup.down.prevent="navigateHistory('down')"
            @keydown.tab="handleTab"
            placeholder="TYPE_COMMAND_HERE..." 
            class="bg-transparent border-none outline-none w-full text-emerald-400 placeholder:text-zinc-800 font-bold caret-transparent"
          >
          <!-- Custom caret -->
          <div 
            class="absolute pointer-events-none bg-emerald-500 w-2 h-4 transition-all duration-75 animate-[blink_1s_infinite]"
            :style="{ left: `${(command.length * 7.2) + 2}px` }"
          ></div>
      </div>
      <button @click="handleSend" class="text-zinc-600 hover:text-emerald-400 transition-all hover:scale-110">
        <Send class="w-4 h-4" />
      </button>
    </div>

    <!-- Background glow -->
    <div class="absolute inset-0 bg-emerald-500/5 pointer-events-none"></div>
  </div>
</template>

<style scoped>
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}

.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}

.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #27272a;
  border-radius: 10px;
}

.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #3f3f46;
}

/* For the caret positioning - approximate character width */
input {
    letter-spacing: 0.01em;
}
</style>
