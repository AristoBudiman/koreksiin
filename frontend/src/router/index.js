import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import LandingView from '@/views/LandingView.vue'
import LoginView from '@/views/LoginView.vue'
import RegisterView from '@/views/RegisterView.vue'
import GuruLayout from '@/layouts/GuruLayout.vue'
import AdminLayout from '@/layouts/AdminLayout.vue'
import GuruDashboardView from '@/views/guru/GuruDashboardView.vue'
import AdminDashboardView from '@/views/admin/AdminDashboardView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'landing',
      component: LandingView
    },
    {
      path: '/masuk',
      name: 'login',
      component: LoginView,
      meta: { guestOnly: true }
    },
    {
      path: '/daftar',
      name: 'register',
      component: RegisterView,
      meta: { guestOnly: true }
    },
    {
      path: '/guru',
      component: GuruLayout,
      meta: { requiresAuth: true, role: 'guru' },
      children: [
        {
          path: '',
          name: 'guru-dashboard',
          component: GuruDashboardView
        }
      ]
    },
    {
      path: '/admin',
      component: AdminLayout,
      meta: { requiresAuth: true, role: 'admin' },
      children: [
        {
          path: '',
          name: 'admin-dashboard',
          component: AdminDashboardView
        }
      ]
    }
  ],
  scrollBehavior() {
    return { top: 0 }
  }
})

router.beforeEach(async (to) => {
  const authStore = useAuthStore()
  if (!authStore.initialized) {
    await authStore.init()
  }

  if (to.meta.requiresAuth && !authStore.isAuthenticated) {
    return { name: 'login' }
  }

  if (to.meta.role && authStore.user?.role !== to.meta.role) {
    return authStore.isAdmin ? { name: 'admin-dashboard' } : { name: 'guru-dashboard' }
  }

  if (to.meta.guestOnly && authStore.isAuthenticated) {
    return authStore.isAdmin ? { name: 'admin-dashboard' } : { name: 'guru-dashboard' }
  }

  return true
})

export default router

