import { createRouter, createWebHistory } from 'vue-router';
import { useAuthStore } from '../store/auth';

const routes = [
    {
        path: '/login',
        name: 'Login',
        component: () => import('../views/LoginView.vue'),
        meta: { guest: true }
    },
    {
        path: '/',
        component: () => import('../layout/MainLayout.vue'),
        children: [
            {
                path: '',
                name: 'Dashboard',
                component: () => import('../views/DashboardView.vue'),
            },
            {
                path: 'servers',
                name: 'Servers',
                component: () => import('../views/ServersView.vue'),
            },
            {
                path: 'servers/:id',
                name: 'ServerDetail',
                component: () => import('../views/ServerDetailView.vue'),
                props: true
            },
            {
                path: 'mods',
                name: 'Mods',
                component: () => import('../views/ModsView.vue'),
            },
            {
                path: 'among-us',
                name: 'AmongUs',
                component: () => import('../views/AmongUsView.vue'),
            }
        ],
        meta: { auth: true }
    }
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

router.beforeEach(async (to, from, next) => {
    const authStore = useAuthStore();

    if (!authStore.user && to.meta.auth) {
        try {
            await authStore.fetchUser();
        } catch (e) {
            console.error('Failed to fetch user context:', e);
            // If fetch fails, user is likely not authenticated
        }
    }

    if (to.meta.auth && !authStore.isLoggedIn) {
        next('/login');
    } else if (to.meta.guest && authStore.isLoggedIn) {
        next('/');
    } else {
        next();
    }
});

export default router;
