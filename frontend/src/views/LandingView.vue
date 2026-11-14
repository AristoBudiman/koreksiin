<template>
  <div class="min-h-screen flex flex-col bg-ink-bg dark:bg-ink-darkBg text-ink dark:text-ink-darkText">
    <!-- Header -->
    <header class="sticky top-0 z-30 border-b border-ink-line dark:border-ink-darkLine bg-ink-card/90 dark:bg-ink-darkCard/90 backdrop-blur-md">
      <div class="max-w-6xl mx-auto px-4 sm:px-6 h-16 flex items-center justify-between">
        <RouterLink to="/" class="flex items-center gap-2.5 group">
          <BrandLogo size="md" compactOnMobile />
          <div class="sm:hidden flex flex-col">
            <span class="font-extrabold text-sm tracking-tight leading-none">
              Koreksi<span class="text-brand-700 dark:text-brand-400">.in</span>
            </span>
            <span class="text-[10px] font-medium text-ink-muted">Penilaian Esai Otomatis AI</span>
          </div>
        </RouterLink>

        <nav class="hidden md:flex items-center gap-6 text-sm font-semibold text-ink-muted">
          <a href="#fitur" class="hover:text-brand-700 dark:hover:text-brand-400 transition-colors">Fitur Utama</a>
          <a href="#cara-kerja" class="hover:text-brand-700 dark:hover:text-brand-400 transition-colors">Cara Kerja</a>
          <a href="#harga" class="hover:text-brand-700 dark:hover:text-brand-400 transition-colors">Paket Harga</a>
        </nav>

        <div class="flex items-center gap-2.5">
          <button
            type="button"
            class="p-2 rounded-lg border border-ink-line dark:border-ink-darkLine text-ink-muted hover:text-ink dark:hover:text-ink-darkText hover:bg-brand-50/40 dark:hover:bg-ink-darkLine transition-colors"
            @click="themeStore.toggleTheme()"
            :aria-label="themeStore.isDark ? 'Aktifkan mode terang' : 'Aktifkan mode gelap'"
          >
            <Sun v-if="themeStore.isDark" class="w-4 h-4 text-brand-400" />
            <Moon v-else class="w-4 h-4" />
          </button>

          <template v-if="authStore.isAuthenticated">
            <RouterLink
              :to="authStore.isAdmin ? '/admin' : '/guru'"
              class="btn-primary"
            >
              <LayoutDashboard class="w-4 h-4" />
              <span>Buka Dashboard</span>
            </RouterLink>
          </template>
          <template v-else>
            <RouterLink
              to="/masuk"
              class="btn-secondary px-3 py-2 sm:px-4 sm:py-2.5 text-xs sm:text-sm"
            >
              Masuk
            </RouterLink>
            <RouterLink
              to="/daftar"
              class="btn-primary px-3.5 py-2 sm:px-5 sm:py-2.5 text-xs sm:text-sm"
            >
              <span>Daftar Guru</span>
              <ArrowRight class="w-4 h-4 hidden sm:inline" />
            </RouterLink>
          </template>
        </div>
      </div>
    </header>

    <!-- Hero Section -->
    <main class="flex-1">
      <section class="relative overflow-hidden py-14 sm:py-20 border-b border-ink-line dark:border-ink-darkLine">
        <div class="max-w-6xl mx-auto px-4 sm:px-6 grid lg:grid-cols-12 gap-10 items-center">
          <div class="lg:col-span-7 space-y-6">
            <div class="inline-flex items-center gap-2 px-3 py-1.5 rounded-full bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 text-brand-700 dark:text-brand-300 text-xs font-bold">
              <Sparkles class="w-3.5 h-3.5" />
              <span>Ditenagai IndoBERT (all-indobert-base-v4) &amp; Adaptasi Gaya Penilaian Guru (LoRA)</span>
            </div>

            <h1 class="text-3xl sm:text-5xl font-extrabold tracking-tight leading-[1.15]">
              Koreksi Ratusan Esai Siswa Lebih Cepat, Objektif, dan
              <HandDrawnUnderline>
                <span class="text-brand-700 dark:text-brand-400">Konsisten</span>
              </HandDrawnUnderline>
            </h1>

            <p class="text-base sm:text-lg text-ink-muted leading-relaxed max-w-2xl">
              Bantu guru memeriksa jawaban esai berdasarkan poin kunci per kriteria rubrik, memberikan umpan balik terstruktur untuk setiap siswa, mendeteksi kemiripan jawaban, serta mempelajari pola koreksi Anda melalui model custom.
            </p>

            <div class="flex flex-wrap items-center gap-3 pt-2">
              <RouterLink to="/daftar" class="btn-primary px-6 py-3 text-base">
                <UserPlus class="w-5 h-5" />
                <span>Mulai Gratis Sekarang</span>
              </RouterLink>
              <RouterLink to="/masuk" class="btn-secondary px-5 py-3 text-base">
                <LogIn class="w-5 h-5" />
                <span>Masuk ke Akun</span>
              </RouterLink>
            </div>

            <div class="grid grid-cols-3 gap-4 pt-4 border-t border-ink-line dark:border-ink-darkLine max-w-lg">
              <div>
                <p class="text-xl sm:text-2xl font-extrabold num-tabular text-brand-700 dark:text-brand-400">35 Esai</p>
                <p class="text-xs text-ink-muted font-medium">Kuota gratis tiap bulan</p>
              </div>
              <div>
                <p class="text-xl sm:text-2xl font-extrabold num-tabular">Per Kriteria</p>
                <p class="text-xs text-ink-muted font-medium">Analisis cakupan poin kunci</p>
              </div>
              <div>
                <p class="text-xl sm:text-2xl font-extrabold num-tabular">100% Lokal</p>
                <p class="text-xs text-ink-muted font-medium">Privasi terjaga tanpa API luar</p>
              </div>
            </div>
          </div>

          <!-- Kartu Ilustrasi Kertas Koreksi Guru -->
          <div class="lg:col-span-5">
            <div class="card-paper p-5 sm:p-6 space-y-5 relative">
              <div class="flex items-start justify-between border-b border-ink-line dark:border-ink-darkLine pb-4">
                <div>
                  <span class="inline-block px-2.5 py-0.5 rounded bg-brand-50 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 text-xs font-bold mb-1">
                    Biologi Kelas X — Fotosintesis
                  </span>
                  <h2 class="font-bold text-base">Lembar Koreksi Jawaban Siswa</h2>
                  <p class="text-xs text-ink-muted">Siswa: Nadia Putri Rahmawati</p>
                </div>
                <div class="text-right">
                  <span class="text-[11px] uppercase tracking-wider font-bold text-ink-muted block">Nilai Akhir</span>
                  <ScoreCircleHighlight>
                    <span class="text-2xl sm:text-3xl font-extrabold text-brand-700 dark:text-brand-400">
                      <CountUpNumber :value="86.5" :decimals="1" />
                    </span>
                  </ScoreCircleHighlight>
                </div>
              </div>

              <div class="space-y-2.5 text-xs">
                <div class="p-3 rounded-lg bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine flex items-start gap-2.5">
                  <CircleCheck class="w-4 h-4 text-criteria-terpenuhi shrink-0 mt-0.5" />
                  <div>
                    <div class="flex items-center gap-2">
                      <span class="font-bold text-ink dark:text-ink-darkText">1. Definisi Proses</span>
                      <span class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300">Terpenuhi</span>
                    </div>
                    <p class="text-ink-muted mt-0.5">Definisi proses sudah tepat dan lengkap.</p>
                  </div>
                </div>

                <div class="p-3 rounded-lg bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine flex items-start gap-2.5">
                  <CircleAlert class="w-4 h-4 text-criteria-sebagian shrink-0 mt-0.5" />
                  <div>
                    <div class="flex items-center gap-2">
                      <span class="font-bold text-ink dark:text-ink-darkText">2. Bahan dan Hasil</span>
                      <span class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300">Sebagian</span>
                    </div>
                    <p class="text-ink-muted mt-0.5">Bahan dan hasil baru dijelaskan sebagian. Belum menyebut: menghasilkan oksigen.</p>
                  </div>
                </div>

                <div class="p-3 rounded-lg bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine flex items-start gap-2.5">
                  <CircleCheck class="w-4 h-4 text-criteria-terpenuhi shrink-0 mt-0.5" />
                  <div>
                    <div class="flex items-center gap-2">
                      <span class="font-bold text-ink dark:text-ink-darkText">3. Tempat &amp; Klorofil</span>
                      <span class="px-1.5 py-0.5 rounded text-[10px] font-bold bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300">Terpenuhi</span>
                    </div>
                    <p class="text-ink-muted mt-0.5">Tempat terjadinya sudah tepat dan lengkap.</p>
                  </div>
                </div>
              </div>

              <div class="pt-2 border-t border-ink-line dark:border-ink-darkLine flex items-center justify-between text-xs text-ink-muted">
                <span class="inline-flex items-center gap-1.5 font-medium">
                  <ShieldCheck class="w-4 h-4 text-criteria-terpenuhi" />
                  Tidak ada indikasi kemiripan jawaban
                </span>
                <span class="font-semibold text-brand-700 dark:text-brand-400">Siap Ekspor Excel</span>
              </div>
            </div>
          </div>
        </div>
      </section>

      <!-- Fitur Utama -->
      <section id="fitur" class="py-16 border-b border-ink-line dark:border-ink-darkLine">
        <div class="max-w-6xl mx-auto px-4 sm:px-6 space-y-10">
          <div class="max-w-2xl">
            <HandDrawnUnderline>
              <h2 class="text-2xl sm:text-3xl font-extrabold tracking-tight">Fitur Utama untuk Guru</h2>
            </HandDrawnUnderline>
            <p class="text-sm sm:text-base text-ink-muted mt-3">
              Dirancang khusus untuk kebutuhan evaluasi esai Bahasa Indonesia di tingkat sekolah maupun perguruan tinggi.
            </p>
          </div>

          <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-5 stagger-grid">
            <div class="card-paper p-5 space-y-3 hover:shadow-paperHover">
              <div class="w-10 h-10 rounded-lg bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center">
                <ListChecks class="w-5 h-5" />
              </div>
              <h3 class="font-bold text-base">Rubrik Multi-Kriteria</h3>
              <p class="text-xs sm:text-sm text-ink-muted leading-relaxed">
                Susun kriteria penilaian lengkap dengan bobot dan poin-poin kunci jawaban. Sistem mencocokkan kalimat siswa secara semantik.
              </p>
            </div>

            <div class="card-paper p-5 space-y-3 hover:shadow-paperHover">
              <div class="w-10 h-10 rounded-lg bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center">
                <FileSpreadsheet class="w-5 h-5" />
              </div>
              <h3 class="font-bold text-base">Impor &amp; Ekspor Excel</h3>
              <p class="text-xs sm:text-sm text-ink-muted leading-relaxed">
                Unggah jawaban satu kelas sekaligus menggunakan template Excel dan unduh rekap nilai beserta status tiap kriteria.
              </p>
            </div>

            <div class="card-paper p-5 space-y-3 hover:shadow-paperHover">
              <div class="w-10 h-10 rounded-lg bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center">
                <CopyCheck class="w-5 h-5" />
              </div>
              <h3 class="font-bold text-base">Deteksi Kemiripan Ganda</h3>
              <p class="text-xs sm:text-sm text-ink-muted leading-relaxed">
                Mendeteksi pasangan jawaban siswa yang terindikasi menyontek melalui kombinasi kemiripan makna dan kemiripan kata.
              </p>
            </div>

            <div class="card-paper p-5 space-y-3 hover:shadow-paperHover">
              <div class="w-10 h-10 rounded-lg bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center">
                <Sliders class="w-5 h-5" />
              </div>
              <h3 class="font-bold text-base">Model Custom LoRA</h3>
              <p class="text-xs sm:text-sm text-ink-muted leading-relaxed">
                Revisi nilai yang Anda berikan menjadi data latih untuk menyesuaikan model all-indobert-base-v4 dengan standar penilaian Anda.
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- Cara Kerja -->
      <section id="cara-kerja" class="py-16 border-b border-ink-line dark:border-ink-darkLine bg-ink-card/50 dark:bg-ink-darkCard/30">
        <div class="max-w-6xl mx-auto px-4 sm:px-6 space-y-10">
          <div class="max-w-2xl">
            <HandDrawnUnderline>
              <h2 class="text-2xl sm:text-3xl font-extrabold tracking-tight">Alur Penilaian 4 Langkah</h2>
            </HandDrawnUnderline>
          </div>

          <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-5 stagger-grid">
            <div v-for="(step, idx) in steps" :key="idx" class="card-paper p-5 space-y-2.5">
              <div class="flex items-center justify-between">
                <span class="text-xs font-extrabold uppercase tracking-wider text-brand-700 dark:text-brand-400">
                  Langkah 0{{ idx + 1 }}
                </span>
                <component :is="step.icon" class="w-5 h-5 text-ink-muted" />
              </div>
              <h3 class="font-bold text-base">{{ step.title }}</h3>
              <p class="text-xs sm:text-sm text-ink-muted leading-relaxed">{{ step.desc }}</p>
            </div>
          </div>
        </div>
      </section>

      <!-- Paket Harga -->
      <section id="harga" class="py-16">
        <div class="max-w-6xl mx-auto px-4 sm:px-6 space-y-10">
          <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
            <div>
              <HandDrawnUnderline>
                <h2 class="text-2xl sm:text-3xl font-extrabold tracking-tight">Paket Langganan Prabayar</h2>
              </HandDrawnUnderline>
              <p class="text-sm sm:text-base text-ink-muted mt-3">
                Transparan, tanpa perpanjangan otomatis yang mengejutkan. Mendukung pembayaran via QRIS, GoPay, ShopeePay, Virtual Account Bank, Gerai Retail, dan Kartu.
              </p>
            </div>
          </div>

          <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-5 stagger-grid">
            <div
              v-for="plan in plans"
              :key="plan.code"
              class="card-paper p-6 flex flex-col justify-between"
              :class="plan.code === 'monthly' ? 'border-2 border-brand-500 ring-4 ring-brand-500/10' : ''"
            >
              <div class="space-y-4">
                <div class="flex items-center justify-between gap-2">
                  <span
                    class="px-2.5 py-1 rounded text-xs font-bold"
                    :class="
                      plan.code === 'monthly'
                        ? 'bg-brand-100 text-brand-800 dark:bg-brand-900/50 dark:text-brand-200'
                        : 'bg-ink-bg dark:bg-ink-darkBg text-ink-muted border border-ink-line dark:border-ink-darkLine'
                    "
                  >
                    {{ plan.is_addon ? 'Kuota Tambahan' : plan.code === 'monthly' ? 'Paling Populer' : 'Paket Langganan' }}
                  </span>
                </div>

                <div>
                  <h3 class="font-extrabold text-lg">{{ plan.name }}</h3>
                  <p class="text-xs text-ink-muted mt-1 min-h-[34px]">{{ plan.description }}</p>
                </div>

                <div class="pt-2 border-t border-ink-line dark:border-ink-darkLine">
                  <span class="text-2xl sm:text-3xl font-extrabold num-tabular">
                    {{ formatRupiah(plan.price_idr) }}
                  </span>
                  <span class="text-xs text-ink-muted block mt-0.5">
                    {{ plan.is_addon ? 'Mengikuti masa aktif paket' : plan.duration_days + ' hari masa aktif' }}
                  </span>
                </div>

                <ul class="space-y-2 text-xs text-ink dark:text-ink-darkText pt-2">
                  <li class="flex items-center gap-2">
                    <CircleCheck class="w-4 h-4 text-criteria-terpenuhi shrink-0" />
                    <span class="num-tabular font-semibold">{{ plan.quota_essays }} kuota jawaban siswa</span>
                  </li>
                  <li class="flex items-center gap-2">
                    <CircleCheck class="w-4 h-4 text-criteria-terpenuhi shrink-0" />
                    <span>Deteksi kemiripan &amp; ekspor Excel</span>
                  </li>
                  <li class="flex items-center gap-2">
                    <CircleCheck class="w-4 h-4 text-criteria-terpenuhi shrink-0" />
                    <span>Pelatihan model custom LoRA</span>
                  </li>
                </ul>
              </div>

              <div class="pt-6">
                <RouterLink
                  to="/daftar"
                  :class="plan.code === 'monthly' ? 'btn-primary w-full' : 'btn-secondary w-full'"
                >
                  <span>{{ plan.price_idr === 0 ? 'Daftar Gratis' : 'Pilih Paket' }}</span>
                  <ArrowRight class="w-4 h-4" />
                </RouterLink>
              </div>
            </div>
          </div>
        </div>
      </section>
    </main>

    <!-- Footer -->
    <footer class="border-t border-ink-line dark:border-ink-darkLine bg-ink-card dark:bg-ink-darkCard py-8">
      <div class="max-w-6xl mx-auto px-4 sm:px-6 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-ink-muted">
        <div class="flex items-center gap-3">
          <BrandLogo size="sm" />
          <span>— Platform SaaS Penilaian Esai Otomatis Berbasis AI</span>
        </div>
        <p>Proyek Mata Kuliah Kewirausahaan — Informatika UNS</p>
      </div>
    </footer>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import {
  Sparkles,
  ArrowRight,
  UserPlus,
  LogIn,
  LayoutDashboard,
  CircleCheck,
  CircleAlert,
  ShieldCheck,
  ListChecks,
  FileSpreadsheet,
  CopyCheck,
  Sliders,
  FileText,
  Upload,
  Cpu,
  Download,
  Sun,
  Moon
} from 'lucide-vue-next'
import BrandLogo from '@/components/BrandLogo.vue'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import ScoreCircleHighlight from '@/components/ScoreCircleHighlight.vue'
import CountUpNumber from '@/components/CountUpNumber.vue'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { apiFetch } from '@/utils/api'

const authStore = useAuthStore()
const themeStore = useThemeStore()

const steps = [
  {
    title: 'Buat Tugas & Rubrik',
    desc: 'Tulis soal esai dan tentukan poin-poin kunci jawaban beserta bobot untuk setiap kriteria.',
    icon: FileText
  },
  {
    title: 'Unggah Jawaban Siswa',
    desc: 'Tempel teks jawaban secara langsung atau unggah file Excel seluruh siswa dalam satu langkah.',
    icon: Upload
  },
  {
    title: 'Penilaian AI & Cek Kemiripan',
    desc: 'Model all-indobert-base-v4 menilai cakupan tiap kriteria, menyusun feedback, dan menandai jawaban yang mirip.',
    icon: Cpu
  },
  {
    title: 'Revisi, Latih Model & Ekspor',
    desc: 'Sesuaikan nilai bila perlu untuk melatih model custom Anda, lalu unduh laporan lengkap ke Excel.',
    icon: Download
  }
]

const plans = ref([
  {
    code: 'free',
    name: 'Paket Gratis',
    description: 'Cocok untuk mencoba fitur penilaian otomatis dengan kuota esai yang direset setiap bulan.',
    price_idr: 0,
    quota_essays: 35,
    duration_days: 30,
    is_addon: false
  },
  {
    code: 'monthly',
    name: 'Paket Guru Bulanan',
    description: 'Pilihan tepat untuk penilaian ulangan harian dan ujian tengah semester beserta model custom.',
    price_idr: 49000,
    quota_essays: 300,
    duration_days: 30,
    is_addon: false
  },
  {
    code: 'yearly',
    name: 'Paket Sekolah Tahunan',
    description: 'Kuota besar untuk dua semester penuh dengan harga jauh lebih hemat.',
    price_idr: 450000,
    quota_essays: 4000,
    duration_days: 365,
    is_addon: false
  },
  {
    code: 'addon',
    name: 'Add-on Kuota +150 Esai',
    description: 'Tambahan kuota esai instan yang berlaku mengikuti masa aktif paket Anda saat ini.',
    price_idr: 20000,
    quota_essays: 150,
    duration_days: 0,
    is_addon: true
  }
])

function formatRupiah(amount) {
  if (!amount || amount === 0) return 'Rp 0'
  return 'Rp ' + Number(amount).toLocaleString('id-ID')
}

onMounted(async () => {
  try {
    const res = await apiFetch('/api/plans')
    if (res?.plans?.length) {
      plans.value = res.plans
    }
  } catch {
    // Gunakan daftar paket fallback bila API belum merespons
  }
})
</script>

