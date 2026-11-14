<template>
  <div class="min-h-screen flex flex-col bg-ink-bg dark:bg-ink-darkBg text-ink dark:text-ink-darkText">
    <!-- Topbar -->
    <header class="sticky top-0 z-30 border-b border-ink-line dark:border-ink-darkLine bg-ink-card/95 dark:bg-ink-darkCard/95 backdrop-blur-md">
      <div class="max-w-7xl mx-auto px-3 sm:px-6 h-16 flex items-center justify-between gap-3">
        <div class="shrink-0 flex items-center gap-3 min-w-0">
          <RouterLink to="/admin" class="flex items-center gap-2 shrink-0">
            <BrandLogo size="md" compactOnMobile />
            <div class="sm:hidden whitespace-nowrap">
              <span class="font-extrabold text-sm tracking-tight leading-none block">
                Koreksi<span class="text-brand-700 dark:text-brand-400">.in</span>
              </span>
              <span class="text-[10px] font-semibold text-brand-700 dark:text-brand-400 block">Panel Admin</span>
            </div>
            <span class="hidden sm:inline-block px-2 py-0.5 rounded bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 text-[11px] font-bold text-brand-700 dark:text-brand-300 whitespace-nowrap">
              Panel Administrator
            </span>
          </RouterLink>
        </div>

        <nav class="hidden md:flex flex-1 min-w-0 items-center gap-1 text-sm font-semibold overflow-hidden">
          <RouterLink
            to="/admin"
            exact-active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="px-3.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-2 whitespace-nowrap shrink-0"
          >
            <LayoutDashboard class="w-4 h-4 shrink-0" />
            <span>Dashboard Admin</span>
          </RouterLink>
        </nav>

        <div class="shrink-0 flex items-center gap-2 sm:gap-3">
          <button
            type="button"
            class="p-2 rounded-lg border border-ink-line dark:border-ink-darkLine text-ink-muted hover:text-ink dark:hover:text-ink-darkText shrink-0"
            @click="themeStore.toggleTheme()"
            :aria-label="themeStore.isDark ? 'Mode terang' : 'Mode gelap'"
          >
            <Sun v-if="themeStore.isDark" class="w-4 h-4 text-brand-400" />
            <Moon v-else class="w-4 h-4" />
          </button>

          <div class="hidden sm:block max-w-[180px] text-right min-w-0">
            <p class="text-xs font-bold leading-tight truncate" :title="authStore.user?.name">{{ authStore.user?.name }}</p>
            <p class="text-[11px] text-ink-muted leading-tight truncate" :title="authStore.user?.email">{{ authStore.user?.email }}</p>
          </div>

          <button
            type="button"
            class="btn-secondary px-2.5 sm:px-3 py-2 text-xs whitespace-nowrap shrink-0"
            @click="handleLogout"
          >
            <LogOut class="w-4 h-4 shrink-0" />
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

