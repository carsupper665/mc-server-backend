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
        path: '/login/callback',
        name: 'LoginCallback',
        component: () => import('../views/LoginCallbackView.vue'),
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
                props: route => ({
                    id: route.params.id,
                    display_name: route.query.display_name
                })
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
    },
    {
        path: '/:pathMatch(.*)*',
        name: 'NotFound',
        component: () => import('../views/NotFoundView.vue')
    }
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

router.beforeEach(async (to, from, next) => {
    const authStore = useAuthStore();
    const isLoginCallback = to.name === 'LoginCallback';

    if (!authStore.user && to.meta.auth) {
        try {
            await authStore.fetchUser({ skipAuthRedirect: true });
        } catch (e) {
            console.error('Failed to fetch user context:', e);
            // If fetch fails, user is likely not authenticated
        }
    }

    if (to.meta.auth && !authStore.isLoggedIn) {
        next('/login');
    } else if (to.meta.guest && authStore.isLoggedIn && !isLoginCallback) {
        next('/');
    } else {
        next();
    }
});

export default router;
