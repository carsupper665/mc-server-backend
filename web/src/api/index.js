import axios from 'axios';

const api = axios.create({
    baseURL: '', // Remove /api prefix to handle different endpoint groups
    timeout: 10000,
    withCredentials: true,
});

// Response interceptor
api.interceptors.response.use(
    (response) => response.data,
    (error) => {
        if (error.response && error.response.status === 401) {
            // Unauthorized, redirect to login if not already there
            if (!window.location.pathname.startsWith('/login')) {
                window.location.href = '/login';
            }
        } else if (error.code === 'ECONNREFUSED') {
            // This is a network error, backend is likely down
            // We can use a global event bus or a pinia store to show a single, non-spammy message
            // For now, let's just log it to avoid spamming the user with toasts
            console.error('Backend connection refused. Please ensure the backend server is running.');
        }
        return Promise.reject(error);
    }
);

export default api;
