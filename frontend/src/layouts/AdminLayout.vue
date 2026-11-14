<template>
  <div class="min-h-screen flex flex-col bg-ink-bg dark:bg-ink-darkBg text-ink dark:text-ink-darkText">
    <!-- Topbar -->
    <header class="sticky top-0 z-30 border-b border-ink-line dark:border-ink-darkLine bg-ink-card/95 dark:bg-ink-darkCard/95 backdrop-blur-md">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between gap-4">
        <div class="flex items-center gap-6">
          <RouterLink to="/admin" class="flex items-center gap-2.5">
            <BrandLogo size="md" compactOnMobile />
            <div class="sm:hidden whitespace-nowrap">
              <span class="font-extrabold text-sm tracking-tight leading-none block">
                Koreksi<span class="text-brand-700 dark:text-brand-400">.in</span>
              </span>
              <span class="text-[10px] font-semibold text-brand-700 dark:text-brand-400 block">Panel Administrator</span>
            </div>
            <span class="hidden sm:inline-block px-2 py-0.5 rounded bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 text-[11px] font-bold text-brand-700 dark:text-brand-300 whitespace-nowrap">
              Panel Administrator
            </span>
          </RouterLink>

          <nav class="hidden md:flex items-center gap-1 text-sm font-semibold">
            <RouterLink
              to="/admin"
              exact-active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
              class="px-3.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-2"
            >
              <LayoutDashboard class="w-4 h-4" />
              <span>Dashboard Admin</span>
            </RouterLink>
          </nav>
        </div>

        <div class="flex items-center gap-3">
          <button
            type="button"
            class="p-2 rounded-lg border border-ink-line dark:border-ink-darkLine text-ink-muted hover:text-ink dark:hover:text-ink-darkText"
            @click="themeStore.toggleTheme()"
            :aria-label="themeStore.isDark ? 'Mode terang' : 'Mode gelap'"
          >
            <Sun v-if="themeStore.isDark" class="w-4 h-4 text-brand-400" />
            <Moon v-else class="w-4 h-4" />
          </button>

          <div class="hidden sm:block text-right">
            <p class="text-xs font-bold leading-tight">{{ authStore.user?.name }}</p>
            <p class="text-[11px] text-ink-muted leading-tight">{{ authStore.user?.email }}</p>
          </div>

          <button
            type="button"
            class="btn-secondary px-3 py-2 text-xs"
            @click="handleLogout"
          >
            <LogOut class="w-4 h-4" />
            <span class="hidden sm:inline">Keluar</span>
          </button>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 py-8">
      <RouterView />
    </main>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import {
  LayoutDashboard,
  LogOut,
  Sun,
  Moon
} from 'lucide-vue-next'
import BrandLogo from '@/components/BrandLogo.vue'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useToastStore } from '@/stores/toast'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const toastStore = useToastStore()

function handleLogout() {
  authStore.logout()
  toastStore.show('Anda telah keluar dari panel Administrator.')
  router.push('/masuk')
}
</script>

