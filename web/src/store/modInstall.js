import { defineStore } from 'pinia';
import api from '../api';
import { ensureDeviceId, getAccessToken } from '../utils/authStorage';

const jobStreams = new Map();
const PLACEHOLDER_ICON = 'https://cdn.modrinth.com/placeholder.svg';

export const useModInstallStore = defineStore('modInstall', {
  state: () => ({
    jobs: []
  }),

  actions: {
    dismissJob(jobId) {
      const job = this.jobs.find(job => job.jobId === jobId);
      if (!job || !['completed', 'failed'].includes(job.status)) return;
      this.closeJobStream(jobId);
      this.jobs = this.jobs.filter(job => job.jobId !== jobId);
    },

    /**
     * 把 mod 加入伺服器的安裝佇列（ModsView 與伺服器詳情頁共用）
     * @returns {Promise<object>} 新增的工作；失敗時丟出 API 錯誤
     */
    async enqueueModInstall({ mod, server, versionId }) {
      const res = await api.post(`/api/v1/server/mod/add/${server.server_id}?async=true`, {
        mod_id: mod.project_id,
        version_id: versionId || '',
        auto_update: true
      }, { silent: true });
      if (!res?.job_id) throw new Error('無法取得安裝工作 ID');
      const job = {
        jobId: res.job_id,
        serverId: server.server_id,
        serverName: server.display_name || server.server_id,
        modId: mod.project_id,
        modTitle: mod.title || mod.project_id,
        modIcon: mod.icon_url || PLACEHOLDER_ICON,
        stage: res.status || 'queued',
        status: 'running',
        percent: 0,
        message: ''
      };
      this.addJob(job);
      return job;
    },

    // 追蹤伺服器安裝（建立伺服器、匯入整合包）的進度；同一個工作只加一次
    trackServerInstall({ jobId, serverId, serverName, status = 'queued' }) {
      if (!jobId || this.jobs.some(job => job.jobId === jobId)) return;
      this.addJob({ jobId, serverId, serverName, modTitle: serverName || serverId, status, stage: 'queued', percent: 0 });
    },

    addJob(job) {
      if (!job || !job.jobId) return;
      this.jobs = [job, ...this.jobs];
      this.subscribeJob(job.jobId);
    },

    updateJob(jobId, patch) {
      const idx = this.jobs.findIndex(job => job.jobId === jobId);
      if (idx === -1) return;
      this.jobs[idx] = { ...this.jobs[idx], ...patch };
    },

    closeJobStream(jobId) {
      const stream = jobStreams.get(jobId);
      if (stream) {
        stream.close();
        jobStreams.delete(jobId);
      }
    },

    createSseStream(url, onEvent, onError) {
      const controller = new AbortController();
      const headerValue = api.defaults?.headers?.common?.['C-MPMC-WEB-Header'] || '';
      const headers = {
        Accept: 'text/event-stream'
      };
      if (headerValue) {
        headers['C-MPMC-WEB-Header'] = headerValue;
      }
      const token = getAccessToken();
      const deviceId = ensureDeviceId();
      if (token) {
        headers.Authorization = `Bearer ${token}`;
      }
      if (deviceId) {
        headers['X-Device-ID'] = deviceId;
      }

      const decoder = new TextDecoder();
      let buffer = '';

      const processBuffer = () => {
        const parts = buffer.split(/\n\n/);
        buffer = parts.pop() || '';
        for (const part of parts) {
          const lines = part.split(/\r?\n/);
          const event = { event: 'message', data: '' };
          for (const line of lines) {
            if (!line || line.startsWith(':')) continue;
            if (line.startsWith('event:')) {
              event.event = line.slice(6).trim();
            } else if (line.startsWith('data:')) {
              const dataLine = line.slice(5).trim();
              event.data = event.data ? `${event.data}\n${dataLine}` : dataLine;
            }
          }
          if (event.data) {
            onEvent(event);
          }
        }
      };

      fetch(url, {
        method: 'GET',
        headers,
        signal: controller.signal
      })
        .then((response) => {
          if (!response.ok || !response.body) {
            throw new Error(`SSE failed with status ${response.status}`);
          }
          const reader = response.body.getReader();
          const read = () => reader.read().then(({ done, value }) => {
            if (done) { onError(new Error('SSE closed')); return; }
            buffer += decoder.decode(value, { stream: true });
            processBuffer();
            return read();
          });
          return read();
        })
        .catch((err) => {
          if (err.name === 'AbortError') return;
          onError(err);
        });

      return {
        close: () => controller.abort()
      };
    },

    subscribeJob(jobId) {
      if (jobStreams.has(jobId)) return;
      const store = this;
      const stream = this.createSseStream(
        `/api/v1/server/mod/subscribe/${jobId}`,
        (event) => {
          if (event.event !== 'progress') return;
          try {
            const data = JSON.parse(event.data || '{}');
            const status = data.stage === 'completed'
              ? 'completed'
              : (['failed', 'interrupted'].includes(data.stage) ? 'failed' : 'running');

            const patch = {
              stage: data.stage,
              percent: typeof data.percent === 'number' ? data.percent : null,
              message: data.message || '',
              error: data.error || '',
              status
            };

            if (data.server_id) patch.serverId = data.server_id;
            if (data.mod_name) {
              patch.modTitle = data.mod_name;
            }

            store.updateJob(jobId, patch);

            if (status === 'completed' || status === 'failed') {
              store.closeJobStream(jobId);
            }
          } catch (err) {
            store.updateJob(jobId, { status: 'failed', error: 'SSE 資料解析失敗' });
            store.closeJobStream(jobId);
          }
        },
        async () => {
          const current = store.jobs.find(job => job.jobId === jobId);
          if (current?.status === 'completed' || current?.status === 'failed') return;
          store.closeJobStream(jobId);
          try {
            const snapshot = await api.get(`/api/v1/server/mod/job/${jobId}`);
            const terminal = ['completed', 'failed', 'interrupted'].includes(snapshot.status);
            store.updateJob(jobId, { status: snapshot.status === 'interrupted' ? 'failed' : snapshot.status, percent: snapshot.progress, message: snapshot.message, error: snapshot.error || '' });
            if (!terminal) setTimeout(() => store.subscribeJob(jobId), 2000);
          } catch {
            store.updateJob(jobId, { status: 'failed', error: '無法取得安裝進度' });
          }
        }
      );

      jobStreams.set(jobId, stream);
    }
  }
});
