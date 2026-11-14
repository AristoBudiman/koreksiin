import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiFetch, getToken, setToken } from '@/utils/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref(null)
  const initialized = ref(false)
  const loading = ref(false)

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')
  const isGuru = computed(() => user.value?.role === 'guru')

  async function init() {
    if (initialized.value) return
    const token = getToken()
    if (!token) {
      initialized.value = true
      return
    }
    try {
      const res = await apiFetch('/api/auth/me')
      user.value = res.user
    } catch {
      setToken(null)
      user.value = null
    } finally {
      initialized.value = true
    }
  }

  async function login(email, password) {
    loading.value = true
    try {
      const res = await apiFetch('/api/auth/login', {
        method: 'POST',
        body: JSON.stringify({ email, password })
      })
      setToken(res.token)
      user.value = res.user
      return res.user
    } finally {
      loading.value = false
    }
  }

  async function register({ name, email, institution, password }) {
    loading.value = true
    try {
      const res = await apiFetch('/api/auth/register', {
        method: 'POST',
        body: JSON.stringify({ name, email, institution, password })
      })
      setToken(res.token)
      user.value = res.user
      return res.user
    } finally {
      loading.value = false
    }
  }

  async function refreshProfile() {
    if (!getToken()) return null
    const res = await apiFetch('/api/auth/me')
    user.value = res.user
    return res.user
  }

  function logout() {
    setToken(null)
    user.value = null
  }

  return {
    user,
    initialized,
    loading,
    isAuthenticated,
    isAdmin,
    isGuru,
    init,
    login,
    register,
    refreshProfile,
    logout
  }
})

