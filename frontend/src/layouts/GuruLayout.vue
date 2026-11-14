<template>
  <div class="min-h-screen flex flex-col bg-ink-bg dark:bg-ink-darkBg text-ink dark:text-ink-darkText">
    <!-- Topbar -->
    <header class="sticky top-0 z-30 border-b border-ink-line dark:border-ink-darkLine bg-ink-card/95 dark:bg-ink-darkCard/95 backdrop-blur-md">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between gap-3">
        <div class="flex items-center gap-4 lg:gap-6 min-w-0">
          <RouterLink to="/guru" class="flex items-center gap-2.5 shrink-0">
            <div class="w-9 h-9 rounded-lg bg-brand-500 flex items-center justify-center text-white shrink-0">
              <PenTool class="w-5 h-5" />
            </div>
            <div class="whitespace-nowrap">
              <span class="font-extrabold text-lg tracking-tight leading-none block">
                Koreksi<span class="text-brand-700 dark:text-brand-400">.in</span>
              </span>
              <span class="text-[11px] font-semibold text-ink-muted block">Ruang Kerja Guru</span>
            </div>
          </RouterLink>

          <nav class="hidden md:flex items-center gap-1 text-xs font-semibold shrink-0">
            <RouterLink
              to="/guru"
              exact-active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
              class="px-2.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-1.5 whitespace-nowrap"
            >
              <LayoutDashboard class="w-4 h-4 shrink-0" />
              <span>Daftar Tugas</span>
            </RouterLink>
            <RouterLink
              to="/guru/tugas/baru"
              active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
              class="px-2.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-1.5 whitespace-nowrap"
            >
              <PlusCircle class="w-4 h-4 shrink-0" />
              <span>Buat Tugas</span>
            </RouterLink>
            <RouterLink
              to="/guru/model-custom"
              active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
              class="px-2.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-1.5 whitespace-nowrap"
            >
              <Sliders class="w-4 h-4 shrink-0" />
              <span>Model Custom</span>
            </RouterLink>
            <RouterLink
              to="/guru/langganan"
              active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
              class="px-2.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-1.5 whitespace-nowrap"
            >
              <CreditCard class="w-4 h-4 shrink-0" />
              <span>Langganan &amp; Kuota</span>
            </RouterLink>
          </nav>
        </div>

        <div class="flex items-center gap-2 sm:gap-2.5 shrink-0">
          <!-- Indikator Kuota Desktop/Tablet -->
          <div
            v-if="authStore.user?.subscription"
            class="hidden sm:flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 text-xs whitespace-nowrap"
          >
            <Sparkles class="w-3.5 h-3.5 text-brand-700 dark:text-brand-400 shrink-0" />
            <span class="font-semibold text-ink dark:text-ink-darkText hidden 2xl:inline">
              {{ authStore.user.subscription.plan_name }}:
            </span>
            <span class="font-extrabold num-tabular text-brand-700 dark:text-brand-300">
              Sisa {{ authStore.user.subscription.quota_remaining }} / {{ authStore.user.subscription.quota_total }} esai
            </span>
          </div>

          <!-- Indikator Kuota Ponsel -->
          <div
            v-if="authStore.user?.subscription"
            class="sm:hidden flex items-center gap-1 px-2 py-1 rounded-lg bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 text-[11px] font-extrabold text-brand-700 dark:text-brand-300 num-tabular whitespace-nowrap"
          >
            <Sparkles class="w-3 h-3 shrink-0" />
            <span>{{ authStore.user.subscription.quota_remaining }}/{{ authStore.user.subscription.quota_total }}</span>
          </div>

          <button
            type="button"
            class="p-2 rounded-lg border border-ink-line dark:border-ink-darkLine text-ink-muted hover:text-ink dark:hover:text-ink-darkText shrink-0"
            @click="themeStore.toggleTheme()"
            :aria-label="themeStore.isDark ? 'Mode terang' : 'Mode gelap'"
          >
            <Sun v-if="themeStore.isDark" class="w-4 h-4 text-brand-400" />
            <Moon v-else class="w-4 h-4" />
          </button>

          <div class="hidden xl:block text-right whitespace-nowrap">
            <p class="text-xs font-bold leading-tight">{{ authStore.user?.name }}</p>
            <p class="text-[11px] text-ink-muted leading-tight">{{ authStore.user?.institution }}</p>
          </div>

          <button
            type="button"
            class="btn-secondary px-3 py-2 text-xs whitespace-nowrap shrink-0"
            @click="handleLogout"
          >
            <LogOut class="w-4 h-4" />
            <span class="hidden sm:inline">Keluar</span>
          </button>
        </div>
      </div>

      <!-- Navigasi Ponsel (Mobile Subbar) -->
      <div class="md:hidden border-t border-ink-line dark:border-ink-darkLine px-2 py-1.5">
        <nav class="grid grid-cols-4 gap-1 text-[11px] font-bold">
          <RouterLink
            to="/guru"
            exact-active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="py-1.5 px-1 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText flex items-center justify-center gap-1 whitespace-nowrap"
          >
            <LayoutDashboard class="w-3.5 h-3.5 shrink-0" />
            <span>Tugas</span>
          </RouterLink>
          <RouterLink
            to="/guru/tugas/baru"
            active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="py-1.5 px-1 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText flex items-center justify-center gap-1 whitespace-nowrap"
          >
            <PlusCircle class="w-3.5 h-3.5 shrink-0" />
            <span>Buat</span>
          </RouterLink>
          <RouterLink
            to="/guru/model-custom"
            active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="py-1.5 px-1 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText flex items-center justify-center gap-1 whitespace-nowrap"
          >
            <Sliders class="w-3.5 h-3.5 shrink-0" />
            <span>Model</span>
          </RouterLink>
          <RouterLink
            to="/guru/langganan"
            active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="py-1.5 px-1 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText flex items-center justify-center gap-1 whitespace-nowrap"
          >
            <CreditCard class="w-3.5 h-3.5 shrink-0" />
            <span>Kuota</span>
          </RouterLink>
        </nav>
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
  PenTool,
  LayoutDashboard,
  PlusCircle,
  Sliders,
  CreditCard,
  Sparkles,
  LogOut,
  Sun,
  Moon
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useToastStore } from '@/stores/toast'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const toastStore = useToastStore()

function handleLogout() {
  authStore.logout()
  toastStore.show('Anda telah keluar dari sesi.')
  router.push('/masuk')
}
</script>

