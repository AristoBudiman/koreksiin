<template>
  <div class="space-y-6">
    <div>
      <HandDrawnUnderline>
        <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
          Langganan &amp; Kuota Penilaian
        </h1>
      </HandDrawnUnderline>
      <p class="text-sm text-ink-muted mt-2">
        Kelola kuota penilaian esai Anda dan pilih paket prabayar sesuai kebutuhan evaluasi kelas.
      </p>
    </div>

    <div class="card-paper p-6 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div>
          <span class="text-xs font-bold uppercase text-ink-muted">Paket Saat Ini</span>
          <h2 class="text-xl font-extrabold">{{ authStore.user?.subscription?.plan_name || 'Paket Gratis' }}</h2>
        </div>
        <div class="text-right">
          <span class="text-xs font-bold uppercase text-ink-muted block">Sisa Kuota</span>
          <span class="text-2xl font-extrabold num-tabular text-brand-700 dark:text-brand-400">
            {{ authStore.user?.subscription?.quota_remaining ?? 0 }} / {{ authStore.user?.subscription?.quota_total ?? 0 }} esai
          </span>
        </div>
      </div>
      <p class="text-xs text-ink-muted">
        Integrasi pembayaran Midtrans Snap QRIS Sandbox dan Add-on Kuota diaktifkan penuh pada Fase 5.
      </p>
    </div>
  </div>
</template>

<script setup>
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
</script>

