import { defineStore } from 'pinia';
import api from '../api';
import { ensureDeviceId, getAccessToken } from '../utils/authStorage';

const jobStreams = new Map();

export const useModInstallStore = defineStore('modInstall', {
  state: () => ({
    jobs: []
  }),

  actions: {
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
      const headerValue = api.defaults.headers.common['C-MPMC-WEB-Header'] || '';
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
