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
          {{ authStore.user?.institution }} — Kelola tugas esai, rubrik kriteria, dan penilaian otomatis siswa.
        </p>
      </div>

      <div class="flex items-center gap-2.5">
        <a
          href="/api/templates/excel?type=submissions"
          class="btn-secondary text-xs"
          download
        >
          <FileSpreadsheet class="w-4 h-4 text-brand-700 dark:text-brand-400" />
          <span>Template Excel</span>
        </a>
        <RouterLink to="/guru/tugas/baru" class="btn-primary text-xs sm:text-sm">
          <Plus class="w-4 h-4" />
          <span>Buat Tugas Baru</span>
        </RouterLink>
      </div>
    </div>

    <!-- Kartu Status Langganan & Statistik -->
    <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-5 stagger-grid">
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
            Reset bulanan: {{ formatDate(authStore.user.subscription.quota_reset_at) }}
          </template>
          <template v-else>
            Aktif untuk akun guru Anda
          </template>
        </p>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Sisa Kuota Penilaian</span>
          <Sparkles class="w-4 h-4 text-brand-700 dark:text-brand-400" />
        </div>
        <div class="flex items-baseline gap-2">
          <ScoreCircleHighlight>
            <span class="text-2xl font-extrabold text-brand-700 dark:text-brand-400">
              <CountUpNumber :value="authStore.user?.subscription?.quota_remaining ?? 0" :decimals="0" />
            </span>
          </ScoreCircleHighlight>
          <span class="text-xs text-ink-muted num-tabular">
            dari {{ authStore.user?.subscription?.quota_total ?? 0 }} esai
          </span>
        </div>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Total Tugas</span>
          <BookOpen class="w-4 h-4 text-brand-700 dark:text-brand-400" />
        </div>
        <p class="text-2xl font-extrabold num-tabular">
          <CountUpNumber :value="assignments.length" :decimals="0" />
        </p>
        <p class="text-xs text-ink-muted">Tugas esai beserta rubrik</p>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Esai Dinilai</span>
          <FileCheck2 class="w-4 h-4 text-criteria-terpenuhi" />
        </div>
        <p class="text-2xl font-extrabold num-tabular">
          <CountUpNumber :value="stats.total_scored" :decimals="0" />
        </p>
        <p class="text-xs text-ink-muted num-tabular">dari {{ stats.total_submissions }} total jawaban</p>
      </div>
    </div>

    <!-- Daftar Tugas Guru -->
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-extrabold tracking-tight">Daftar Tugas Penilaian Esai</h2>
      </div>

      <!-- Skeleton Loading -->
      <div v-if="loading" class="grid sm:grid-cols-2 lg:grid-cols-3 gap-5">
        <div v-for="n in 3" :key="n" class="card-paper p-5 animate-pulse space-y-3">
          <div class="h-4 bg-ink-line dark:bg-ink-darkLine rounded w-1/3"></div>
          <div class="h-6 bg-ink-line dark:bg-ink-darkLine rounded w-3/4"></div>
          <div class="h-12 bg-ink-line dark:bg-ink-darkLine rounded w-full"></div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-else-if="assignments.length === 0" class="card-paper p-10 text-center space-y-4">
        <div class="w-12 h-12 rounded-xl bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center mx-auto">
          <BookOpenCheck class="w-6 h-6" />
        </div>
        <div class="max-w-md mx-auto space-y-1.5">
          <h3 class="font-extrabold text-lg">Belum Ada Tugas, Buat Tugas Pertamamu</h3>
          <p class="text-xs sm:text-sm text-ink-muted">
            Mulai dengan membuat soal esai dan rubrik kriteria, lalu unggah jawaban siswa dari file Excel untuk dinilai secara otomatis.
          </p>
        </div>
        <RouterLink to="/guru/tugas/baru" class="btn-primary">
          <Plus class="w-4 h-4" />
          <span>Buat Tugas Pertamamu</span>
        </RouterLink>
      </div>

      <!-- Kartu Tugas -->
      <div v-else class="grid sm:grid-cols-2 lg:grid-cols-3 gap-5 stagger-grid">
        <div
          v-for="item in assignments"
          :key="item.id"
          class="card-paper p-5 flex flex-col justify-between hover:shadow-paperHover space-y-4"
        >
          <div class="space-y-3">
            <div class="flex items-center justify-between gap-2">
              <div class="flex flex-wrap items-center gap-1.5">
                <span class="px-2.5 py-0.5 rounded bg-brand-50 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 text-xs font-bold">
                  {{ item.subject }}
                </span>
                <span class="px-2 py-0.5 rounded bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-[11px] font-semibold text-ink-muted">
                  {{ item.model_type === 'custom' ? (item.custom_model_name || 'Model Custom') : 'all-indobert-base-v4' }}
                </span>
              </div>
              <span class="text-xs text-ink-muted num-tabular">
                {{ formatDate(item.created_at) }}
              </span>
            </div>

            <div>
              <RouterLink
                :to="`/guru/tugas/${item.id}`"
                class="font-extrabold text-base hover:text-brand-700 dark:hover:text-brand-400 transition-colors line-clamp-1"
              >
                {{ item.title }}
              </RouterLink>
              <p class="text-xs text-ink-muted line-clamp-2 mt-1 leading-relaxed">
                {{ item.question_text }}
              </p>
            </div>

            <div class="grid grid-cols-3 gap-2 pt-2 border-t border-ink-line dark:border-ink-darkLine text-center text-xs">
              <div>
                <span class="text-ink-muted block text-[11px]">Kriteria</span>
                <strong class="num-tabular">{{ item.criteria_count }}</strong>
              </div>
              <div>
                <span class="text-ink-muted block text-[11px]">Siswa</span>
                <strong class="num-tabular">{{ item.scored_count }}/{{ item.submission_count }}</strong>
              </div>
              <div>
                <span class="text-ink-muted block text-[11px]">Rata-rata</span>
                <strong class="num-tabular text-brand-700 dark:text-brand-400">
                  {{ item.average_score !== null ? Number(item.average_score).toFixed(1) : '-' }}
                </strong>
              </div>
            </div>
          </div>

          <div class="pt-3 border-t border-ink-line dark:border-ink-darkLine flex items-center justify-between gap-2">
            <button
              type="button"
              class="text-xs font-semibold text-ink-muted hover:text-brand-700 inline-flex items-center gap-1"
              @click="handleDelete(item)"
            >
              <Trash2 class="w-3.5 h-3.5" />
              <span>Hapus</span>
            </button>

            <RouterLink :to="`/guru/tugas/${item.id}`" class="btn-primary py-1.5 px-3.5 text-xs">
              <span>Buka Tugas</span>
              <ArrowRight class="w-3.5 h-3.5" />
            </RouterLink>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import {
  Award,
  Sparkles,
  BookOpen,
  FileCheck2,
  BookOpenCheck,
  FileSpreadsheet,
  Plus,
  ArrowRight,
  Trash2
} from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import ScoreCircleHighlight from '@/components/ScoreCircleHighlight.vue'
import CountUpNumber from '@/components/CountUpNumber.vue'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import { apiFetch } from '@/utils/api'

const authStore = useAuthStore()
const toastStore = useToastStore()

const loading = ref(true)
const assignments = ref([])
const stats = reactive({
  total_assignments: 0,
  total_submissions: 0,
  total_scored: 0,
  total_models: 0
})

function formatDate(isoStr) {
  if (!isoStr) return '-'
  return new Date(isoStr).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric'
  })
}

async function fetchDashboard() {
  loading.value = true
  try {
    const [listRes, statsRes] = await Promise.all([
      apiFetch('/api/assignments'),
      apiFetch('/api/guru/stats')
    ])
    assignments.value = listRes.assignments || []
    Object.assign(stats, statsRes)
    await authStore.refreshProfile()
  } catch (err) {
    toastStore.error(err.message || 'Gagal memuat daftar tugas.')
  } finally {
    loading.value = false
  }
}

async function handleDelete(item) {
  try {
    await apiFetch(`/api/assignments/${item.id}`, { method: 'DELETE' })
    toastStore.success(`Tugas "${item.title}" dihapus.`)
    await fetchDashboard()
  } catch (err) {
    toastStore.error(err.message)
  }
}

onMounted(() => {
  fetchDashboard()
})
</script>
