<template>
  <div class="min-h-screen flex flex-col bg-ink-bg dark:bg-ink-darkBg text-ink dark:text-ink-darkText">
    <!-- Topbar -->
    <header class="sticky top-0 z-30 border-b border-ink-line dark:border-ink-darkLine bg-ink-card/95 dark:bg-ink-darkCard/95 backdrop-blur-md">
      <div class="max-w-7xl mx-auto px-3 sm:px-6 h-16 flex items-center justify-between gap-3">
        <!-- Kiri: Logo -->
        <div class="shrink-0 flex items-center gap-3 min-w-0">
          <RouterLink to="/guru" class="flex items-center gap-2 shrink-0">
            <BrandLogo size="md" compactOnMobile />
            <div class="sm:hidden whitespace-nowrap">
              <span class="font-extrabold text-sm tracking-tight leading-none block">
                Koreksi<span class="text-brand-700 dark:text-brand-400">.in</span>
              </span>
              <span class="text-[10px] font-semibold text-ink-muted block">Ruang Guru</span>
            </div>
            <span class="hidden 2xl:inline-block px-2 py-0.5 rounded bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-[11px] font-bold text-ink-muted whitespace-nowrap">
              Ruang Kerja Guru
            </span>
          </RouterLink>
        </div>

        <!-- Tengah: Navigasi Desktop (flex-1 min-w-0 agar tidak pernah menimpa kanan) -->
        <nav class="hidden md:flex flex-1 min-w-0 items-center gap-1 text-xs font-semibold overflow-hidden">
          <RouterLink
            to="/guru"
            title="Daftar Tugas"
            exact-active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="px-2.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-1.5 whitespace-nowrap shrink-0"
          >
            <LayoutDashboard class="w-4 h-4 shrink-0" />
            <span class="hidden xl:inline">Daftar Tugas</span>
          </RouterLink>
          <RouterLink
            to="/guru/tugas/baru"
            title="Buat Tugas"
            active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="px-2.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-1.5 whitespace-nowrap shrink-0"
          >
            <PlusCircle class="w-4 h-4 shrink-0" />
            <span class="hidden xl:inline">Buat Tugas</span>
          </RouterLink>
          <RouterLink
            to="/guru/model-custom"
            title="Model Custom"
            active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="px-2.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-1.5 whitespace-nowrap shrink-0"
          >
            <Sliders class="w-4 h-4 shrink-0" />
            <span class="hidden xl:inline">Model Custom</span>
          </RouterLink>
          <RouterLink
            to="/guru/langganan"
            title="Langganan & Kuota"
            active-class="bg-brand-50 text-brand-700 dark:bg-brand-900/30 dark:text-brand-300"
            class="px-2.5 py-2 rounded-lg text-ink-muted hover:text-ink dark:hover:text-ink-darkText transition-colors flex items-center gap-1.5 whitespace-nowrap shrink-0"
          >
            <CreditCard class="w-4 h-4 shrink-0" />
            <span class="hidden xl:inline">Langganan</span>
          </RouterLink>
        </nav>

        <!-- Kanan: Kuota Ringkas, Tema, Profil (ellipsis), Keluar -->
        <div class="shrink-0 flex items-center gap-1.5 sm:gap-2.5">
          <RouterLink
            v-if="authStore.user?.subscription"
            to="/guru/langganan"
            :title="`${authStore.user.subscription.plan_name}: Sisa ${authStore.user.subscription.quota_remaining} / ${authStore.user.subscription.quota_total} esai`"
            class="flex items-center gap-1.5 px-2 sm:px-2.5 py-1.5 rounded-lg bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 text-[11px] sm:text-xs font-extrabold num-tabular text-brand-700 dark:text-brand-300 whitespace-nowrap hover:border-brand-400 transition-colors shrink-0"
          >
            <Sparkles class="w-3.5 h-3.5 text-brand-700 dark:text-brand-400 shrink-0" />
            <span>{{ authStore.user.subscription.quota_remaining }}/{{ authStore.user.subscription.quota_total }}</span>
          </RouterLink>

          <button
            type="button"
            class="p-2 rounded-lg border border-ink-line dark:border-ink-darkLine text-ink-muted hover:text-ink dark:hover:text-ink-darkText shrink-0"
            @click="themeStore.toggleTheme()"
            :aria-label="themeStore.isDark ? 'Mode terang' : 'Mode gelap'"
          >
            <Sun v-if="themeStore.isDark" class="w-4 h-4 text-brand-400" />
            <Moon v-else class="w-4 h-4" />
          </button>

          <div class="hidden 2xl:block max-w-[160px] text-right min-w-0">
            <p class="text-xs font-bold leading-tight truncate" :title="authStore.user?.name">{{ authStore.user?.name }}</p>
            <p class="text-[11px] text-ink-muted leading-tight truncate" :title="authStore.user?.institution">{{ authStore.user?.institution }}</p>
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
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  LayoutDashboard,
  PlusCircle,
  Sliders,
  CreditCard,
  Sparkles,
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
  toastStore.show('Anda telah keluar dari sesi.')
  router.push('/masuk')
}

onMounted(() => {
  if (authStore.isAuthenticated) {
    authStore.refreshProfile().catch(() => {})
  }
})
</script>

