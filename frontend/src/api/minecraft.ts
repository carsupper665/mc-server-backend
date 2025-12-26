import api from './index';

export const authApi = {
  login: (data: any) => api.post('/Authentication/login', data),
  verify: (data: any) => api.post('/Authentication/verify', data),
  logout: () => api.post('/logout'),
};

export const minecraftApi = {
  getVanillaVersions: () => api.get('/mc-api/vinfo'),
  getFabricVersions: () => api.get('/mc-api/finfo'),
  getMyServers: () => api.get('/user/myservers'),
  
  createServer: (data: any) => api.post('/mc-api/a/create', data),
  getStatus: (id: string) => api.post(`/mc-api/a/status/${id}`),
  startServer: (id: string) => api.post(`/mc-api/a/start/${id}`),
  stopServer: (id: string) => api.post(`/mc-api/a/stop/${id}`),
  
  getProperties: (id: string) => api.post(`/mc-api/a/property/${id}`),
  updateProperties: (id: string, data: any) => api.post(`/mc-api/a/UploadProperty/${id}`, data),
  sendCommand: (id: string, command: string) => api.post(`/mc-api/a/cmd/${id}`, { command }),
  getLogs: (id: string) => api.get(`/server-api/a/log/${id}`),
  backup: (id: string) => api.get(`/mc-api/a/backup/${id}`),
};
