<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { minecraftApi } from '../../api/minecraft';
import { X, Server, Zap, Download } from 'lucide-vue-next';

const emit = defineEmits(['close', 'created']);

const step = ref(1);
const loading = ref(false);
const error = ref('');

const formData = ref({
  display_name: '',
  server_type: 'vanilla',
  server_ver: '',
  fabric_loader: '',
  fabric_installer: ''
});

const vanillaVersions = ref<string[]>([]);
const fabricVersions = ref<string[]>([]);

onMounted(async () => {
  try {
    const [vRes, fRes] = await Promise.all([
      minecraftApi.getVanillaVersions(),
      minecraftApi.getFabricVersions()
    ]);
    
    // Handle map/object from vanilla version API
    if (vRes.data.versions && typeof vRes.data.versions === 'object' && !Array.isArray(vRes.data.versions)) {
      vanillaVersions.value = Object.keys(vRes.data.versions).sort((a, b) => b.localeCompare(a, undefined, { numeric: true }));
    } else {
      vanillaVersions.value = vRes.data.versions || [];
    }
    
    fabricVersions.value = fRes.data.versions || [];
  } catch (err) {
    console.error('Failed to load versions');
  }
});

const handleCreate = async () => {
  if (!formData.value.display_name || !formData.value.server_ver) return;
  
  loading.value = true;
  error.value = '';
  try {
    await minecraftApi.createServer(formData.value);
    emit('created');
    emit('close');
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Failed to initiate deployment';
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="fixed inset-0 z-[100] flex items-center justify-center p-4 backdrop-blur-sm bg-black/60">
    <div class="w-full max-w-2xl bg-zinc-950 border-2 border-zinc-800 shadow-[20px_20px_0px_0px_rgba(0,0,0,0.5)] flex flex-col max-h-[90vh]">
      <!-- Header -->
      <div class="p-6 border-b-2 border-zinc-800 flex justify-between items-center bg-zinc-900/50">
        <div class="flex items-center gap-4">
          <div class="p-2 bg-emerald-500 text-zinc-950">
            <Zap class="w-5 h-5" />
          </div>
          <div>
            <h2 class="text-xl font-black uppercase italic tracking-tighter">Deploy New Node</h2>
            <p class="text-[10px] font-mono text-zinc-500 uppercase tracking-widest">Step {{ step }} of 2: Infrastructure Setup</p>
          </div>
        </div>
        <button @click="$emit('close')" class="text-zinc-500 hover:text-zinc-100 transition-colors">
          <X class="w-6 h-6" />
        </button>
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-y-auto p-8 custom-scrollbar">
        <div v-if="step === 1" class="space-y-8">
          <div class="space-y-2">
            <label class="block text-[10px] font-black uppercase tracking-widest text-zinc-500">Node Designation</label>
            <input 
              v-model="formData.display_name"
              placeholder="e.g. Survival-Alpha-01"
              class="w-full bg-black border-2 border-zinc-800 p-4 text-zinc-100 font-mono focus:border-emerald-500 outline-none transition-all shadow-[inset_0_2px_4px_rgba(0,0,0,0.5)]"
            />
          </div>

          <div class="grid grid-cols-2 gap-6">
            <button 
              @click="formData.server_type = 'vanilla'"
              :class="formData.server_type === 'vanilla' ? 'border-emerald-500 bg-emerald-500/10 text-emerald-400' : 'border-zinc-800 bg-zinc-900/30 text-zinc-500'"
              class="border-2 p-6 flex flex-col items-center gap-3 transition-all group"
            >
              <Server class="w-8 h-8 transition-transform group-hover:scale-110" />
              <span class="font-black uppercase italic tracking-tighter text-lg">Vanilla</span>
            </button>
            <button 
              @click="formData.server_type = 'fabric'"
              :class="formData.server_type === 'fabric' ? 'border-emerald-500 bg-emerald-500/10 text-emerald-400' : 'border-zinc-800 bg-zinc-900/30 text-zinc-500'"
              class="border-2 p-6 flex flex-col items-center gap-3 transition-all group"
            >
              <Zap class="w-8 h-8 transition-transform group-hover:scale-110" />
              <span class="font-black uppercase italic tracking-tighter text-lg">Fabric</span>
            </button>
          </div>

          <button 
            @click="step = 2"
            :disabled="!formData.display_name"
            class="w-full bg-zinc-100 text-zinc-950 py-4 font-black uppercase tracking-[0.2em] text-sm hover:bg-emerald-400 transition-all disabled:opacity-30 disabled:grayscale"
          >
            Continue to Versioning
          </button>
        </div>

        <div v-if="step === 2" class="space-y-8">
          <div class="space-y-2">
            <label class="block text-[10px] font-black uppercase tracking-widest text-zinc-500">Target Environment Version</label>
            <select 
              v-model="formData.server_ver"
              class="w-full bg-black border-2 border-zinc-800 p-4 text-zinc-100 font-mono focus:border-emerald-500 outline-none appearance-none cursor-pointer"
            >
              <option value="" disabled>Select Minecraft Version</option>
              <option v-for="v in vanillaVersions" :key="v" :value="v">{{ v }}</option>
            </select>
          </div>

          <template v-if="formData.server_type === 'fabric'">
            <div class="space-y-2">
              <label class="block text-[10px] font-black uppercase tracking-widest text-zinc-500">Fabric Loader</label>
              <select 
                v-model="formData.fabric_loader"
                class="w-full bg-black border-2 border-zinc-800 p-4 text-zinc-100 font-mono focus:border-emerald-500 outline-none appearance-none"
              >
                <option value="" disabled>Select Loader Version</option>
                <option v-for="v in fabricVersions" :key="v" :value="v">{{ v }}</option>
              </select>
            </div>
          </template>

          <div v-if="error" class="bg-rose-500/10 border border-rose-500/50 p-4 text-rose-400 text-xs font-mono">
            CRITICAL_ERROR: {{ error }}
          </div>

          <div class="grid grid-cols-2 gap-4">
            <button @click="step = 1" class="border-2 border-zinc-800 py-4 font-black uppercase text-xs tracking-widest hover:bg-zinc-900 transition-colors">
              Go Back
            </button>
            <button 
              @click="handleCreate"
              :disabled="loading || !formData.server_ver || (formData.server_type === 'fabric' && !formData.fabric_loader)"
              class="bg-emerald-500 text-zinc-950 py-4 font-black uppercase tracking-[0.2em] text-sm hover:bg-emerald-400 transition-all flex items-center justify-center gap-2 disabled:opacity-30"
            >
              <Download v-if="!loading" class="w-4 h-4" />
              {{ loading ? 'Initializing...' : 'Execute Deployment' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Footer -->
      <div class="p-4 bg-zinc-900/80 border-t-2 border-zinc-800 text-[9px] font-mono text-zinc-600 flex justify-between uppercase tracking-widest">
        <span>Subsystem: Deployment_Module_V1</span>
        <span>Status: Waiting_For_Input</span>
      </div>
    </div>
  </div>
</template>
