<template>
  <div class="space-y-8">
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
      <div>
        <HandDrawnUnderline>
          <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
            Selamat Datang, {{ authStore.user?.name }}
          </h1>
        </HandDrawnUnderline>
        <p class="text-sm text-ink-muted mt-2">
          {{ authStore.user?.institution }} — Kelola tugas esai, rubrik kriteria, dan model penilaian Anda.
        </p>
      </div>
    </div>

    <!-- Kartu Status Langganan & Kuota -->
    <div class="grid sm:grid-cols-3 gap-5">
      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Paket Aktif</span>
          <Award class="w-4 h-4 text-brand-700 dark:text-brand-400" />
        </div>
        <p class="text-xl font-extrabold">{{ authStore.user?.subscription?.plan_name || 'Paket Gratis' }}</p>
        <p class="text-xs text-ink-muted">
          <template v-if="authStore.user?.subscription?.expires_at">
            Berlaku hingga {{ formatDate(authStore.user.subscription.expires_at) }}
          </template>
          <template v-else-if="authStore.user?.subscription?.quota_reset_at">
            Reset kuota berikutnya: {{ formatDate(authStore.user.subscription.quota_reset_at) }}
          </template>
          <template v-else>
            Aktif untuk akun guru Anda
          </template>
        </p>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Sisa Kuota Penilaian</span>
          <FileCheck2 class="w-4 h-4 text-criteria-terpenuhi" />
        </div>
        <div class="flex items-baseline gap-2">
          <ScoreCircleHighlight>
            <span class="text-2xl font-extrabold text-brand-700 dark:text-brand-400">
              {{ authStore.user?.subscription?.quota_remaining ?? 0 }}
            </span>
          </ScoreCircleHighlight>
          <span class="text-xs text-ink-muted num-tabular">
            dari {{ authStore.user?.subscription?.quota_total ?? 0 }} jawaban siswa
          </span>
        </div>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Model AI Penilaian</span>
          <Cpu class="w-4 h-4 text-brand-700 dark:text-brand-400" />
        </div>
        <p class="text-xl font-extrabold">IndoBERT-lite</p>
        <p class="text-xs text-ink-muted">Siap digunakan untuk penilaian rubrik otomatis</p>
      </div>
    </div>

    <!-- Empty State Tugas -->
    <div class="card-paper p-10 text-center space-y-4">
      <div class="w-12 h-12 rounded-xl bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center mx-auto">
        <BookOpenCheck class="w-6 h-6" />
      </div>
      <div class="max-w-md mx-auto space-y-1.5">
        <h2 class="font-extrabold text-lg">Ruang Kerja Guru Siap Digunakan</h2>
        <p class="text-xs sm:text-sm text-ink-muted">
          Akun Anda telah aktif dengan Paket Gratis. Fitur manajemen Tugas, Rubrik, Upload Excel, dan Penilaian Otomatis diaktifkan pada Fase 2.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { Award, FileCheck2, Cpu, BookOpenCheck } from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import ScoreCircleHighlight from '@/components/ScoreCircleHighlight.vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()

function formatDate(isoStr) {
  if (!isoStr) return '-'
  return new Date(isoStr).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'long',
    year: 'numeric'
  })
}
</script>

