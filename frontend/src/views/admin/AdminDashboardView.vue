<template>
  <div class="space-y-8">
    <div>
      <HandDrawnUnderline>
        <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
          Dashboard Administrator
        </h1>
      </HandDrawnUnderline>
      <p class="text-sm text-ink-muted mt-2">
        Pantau guru terdaftar, esai yang dinilai, transaksi langganan, dan konfigurasi ambang AI.
      </p>
    </div>

    <div class="grid sm:grid-cols-3 gap-5">
      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Status Sistem</span>
          <Server class="w-4 h-4 text-criteria-terpenuhi" />
        </div>
        <p class="text-xl font-extrabold">Aktif &amp; Terhubung</p>
        <p class="text-xs text-ink-muted">PostgreSQL, Redis, Worker, dan Trainer berjalan</p>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Akun Administrator</span>
          <ShieldCheck class="w-4 h-4 text-brand-700 dark:text-brand-400" />
        </div>
        <p class="text-xl font-extrabold">{{ authStore.user?.name }}</p>
        <p class="text-xs text-ink-muted">{{ authStore.user?.email }}</p>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Pengaturan Ambang AI</span>
          <Sliders class="w-4 h-4 text-brand-700 dark:text-brand-400" />
        </div>
        <p class="text-xl font-extrabold num-tabular">T_HIGH 0.72 / T_LOW 0.52</p>
        <p class="text-xs text-ink-muted">Ambang kemiripan: S_SEM 0.85 / S_LEX 0.60</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { Server, ShieldCheck, Sliders } from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
</script>

