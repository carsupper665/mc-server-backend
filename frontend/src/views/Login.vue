<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { authApi } from '../api/minecraft';
import { Shield, Lock, User, KeyRound } from 'lucide-vue-next';

const router = useRouter();
const authStore = useAuthStore();
const username = ref('');
const password = ref('');
const code = ref('');
const step = ref<'login' | 'verify'>('login');
const error = ref('');
const loading = ref(false);

const handleLogin = async () => {
  if (!username.value || (!password.value && step.value === 'login')) return;
  loading.value = true;
  error.value = '';
  try {
    if (step.value === 'login') {
      const res = await authApi.login({ username: username.value, password: password.value });
      if (res.status === 202) {
        step.value = 'verify';
      } else {
        authStore.setToken(res.data.token);
        router.push('/dashboard');
      }
    } else {
      const res = await authApi.verify({ code: code.value });
      authStore.setToken(res.data.token);
      router.push('/dashboard');
    }
  } catch (err: any) {
    error.value = err.response?.data?.error || 'Authentication failed';
  } finally {
    loading.value = false;
  }
};
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4 bg-[radial-gradient(circle_at_center,_#18181b_0%,_#09090b_100%)]">
    <div class="w-full max-w-md">
      <div class="text-center mb-12">
        <div class="inline-flex items-center justify-center w-16 h-16 bg-zinc-900 border-2 border-zinc-800 mb-6 rotate-45 group hover:border-zinc-100 transition-all duration-500">
          <Shield class="w-8 h-8 text-zinc-100 -rotate-45" />
        </div>
        <h1 class="text-4xl font-black uppercase tracking-tighter text-zinc-100 italic">Central Command</h1>
        <p class="text-zinc-500 font-mono text-xs uppercase tracking-[0.3em] mt-2">
          {{ step === 'login' ? 'Authentication Required' : 'Multi-Factor Validation' }}
        </p>
      </div>

      <div class="bg-zinc-900 border-2 border-zinc-800 p-8 shadow-[12px_12px_0px_0px_rgba(24,24,27,1)]">
        <form @submit.prevent="handleLogin" class="space-y-6">
          <template v-if="step === 'login'">
            <div>
              <label class="block text-[10px] font-black uppercase tracking-widest text-zinc-500 mb-2">Access Identity</label>
              <div class="relative">
                <User class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-600" />
                <input 
                  v-model="username"
                  type="text" 
                  class="w-full bg-black border-2 border-zinc-800 py-3 pl-10 pr-4 text-zinc-100 focus:border-zinc-100 outline-none transition-all font-mono text-sm"
                  placeholder="USERNAME"
                >
              </div>
            </div>

            <div>
              <label class="block text-[10px] font-black uppercase tracking-widest text-zinc-500 mb-2">Security Key</label>
              <div class="relative">
                <Lock class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-600" />
                <input 
                  v-model="password"
                  type="password" 
                  class="w-full bg-black border-2 border-zinc-800 py-3 pl-10 pr-4 text-zinc-100 focus:border-zinc-100 outline-none transition-all font-mono text-sm"
                  placeholder="PASSWORD"
                >
              </div>
            </div>
          </template>

          <template v-else>
            <div>
              <label class="block text-[10px] font-black uppercase tracking-widest text-zinc-500 mb-2">Verification Code</label>
              <div class="relative">
                <KeyRound class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-600" />
                <input 
                  v-model="code"
                  type="text" 
                  maxlength="6"
                  class="w-full bg-black border-2 border-zinc-800 py-3 pl-10 pr-4 text-zinc-100 focus:border-zinc-100 outline-none transition-all font-mono text-sm tracking-[1em] text-center"
                  placeholder="000000"
                >
              </div>
              <p class="mt-4 text-[10px] font-mono text-zinc-500 uppercase">
                A validation code was dispatched to your registered relay.
              </p>
            </div>
          </template>

          <div v-if="error" class="bg-rose-500/10 border border-rose-500/50 p-3 text-rose-400 text-xs font-mono">
            Error: {{ error }}
          </div>

          <button 
            type="submit"
            :disabled="loading"
            class="w-full bg-zinc-100 text-zinc-950 py-4 font-black uppercase tracking-[0.2em] text-sm hover:bg-emerald-400 transition-all active:translate-y-1 active:shadow-none"
          >
            {{ loading ? 'Authorizing...' : (step === 'login' ? 'Establish Connection' : 'Verify Identity') }}
          </button>

          <button 
            v-if="step === 'verify'"
            type="button"
            @click="step = 'login'"
            class="w-full text-zinc-500 font-mono text-[10px] uppercase tracking-widest hover:text-zinc-100 transition-colors"
          >
            Back to Initial Access
          </button>
        </form>
      </div>
      
      <p class="text-center mt-8 text-zinc-600 font-mono text-[10px] uppercase tracking-widest">
        &copy; 2025 MC-SERVER-BACKEND // SECURE ACCESS ONLY
      </p>
    </div>
  </div>
</template>
