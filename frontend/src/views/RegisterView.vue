<template>
  <div class="min-h-screen flex flex-col justify-center py-12 px-4 sm:px-6 bg-ink-bg dark:bg-ink-darkBg">
    <div class="sm:mx-auto sm:w-full sm:max-w-md space-y-6">
      <div class="flex items-center justify-between">
        <RouterLink
          to="/"
          class="inline-flex items-center gap-1.5 text-xs font-semibold text-ink-muted hover:text-brand-700 dark:hover:text-brand-400 transition-colors"
        >
          <ArrowLeft class="w-4 h-4" />
          <span>Kembali ke Beranda</span>
        </RouterLink>

        <button
          type="button"
          class="p-2 rounded-lg border border-ink-line dark:border-ink-darkLine text-ink-muted hover:text-ink dark:hover:text-ink-darkText"
          @click="themeStore.toggleTheme()"
          :aria-label="themeStore.isDark ? 'Mode terang' : 'Mode gelap'"
        >
          <Sun v-if="themeStore.isDark" class="w-4 h-4 text-brand-400" />
          <Moon v-else class="w-4 h-4" />
        </button>
      </div>

      <div class="card-paper p-6 sm:p-8 space-y-6">
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2.5">
              <div class="w-9 h-9 rounded-lg bg-brand-500 flex items-center justify-center text-white">
                <PenTool class="w-5 h-5" />
              </div>
              <span class="font-extrabold text-lg">
                Koreksi<span class="text-brand-700 dark:text-brand-400">.in</span>
              </span>
            </div>
            <span class="px-2.5 py-1 rounded bg-brand-50 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 text-xs font-bold">
              Gratis 35 Esai / Bulan
            </span>
          </div>

          <HandDrawnUnderline>
            <h1 class="text-2xl font-extrabold tracking-tight pt-1">Daftar Akun Guru</h1>
          </HandDrawnUnderline>
          <p class="text-xs sm:text-sm text-ink-muted">
            Buat akun guru baru dan dapatkan Paket Gratis secara otomatis untuk mulai menilai esai siswa.
          </p>
        </div>

        <div
          v-if="errorMessage"
          class="p-3 rounded-lg bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 flex items-start gap-2.5 text-brand-700 dark:text-brand-300 text-xs font-semibold"
        >
          <CircleAlert class="w-4 h-4 shrink-0 mt-0.5" />
          <span>{{ errorMessage }}</span>
        </div>

        <form class="space-y-4" @submit.prevent="handleSubmit">
          <div>
            <label for="name" class="block text-xs font-bold mb-1.5">Nama Lengkap &amp; Gelar</label>
            <div class="relative">
              <User class="w-4 h-4 text-ink-muted absolute left-3.5 top-3" />
              <input
                id="name"
                v-model.trim="form.name"
                type="text"
                required
                placeholder="Contoh: Budi Santoso, S.Pd."
                class="input-field pl-10"
              />
            </div>
          </div>

          <div>
            <label for="email" class="block text-xs font-bold mb-1.5">Alamat Email Aktif</label>
            <div class="relative">
              <Mail class="w-4 h-4 text-ink-muted absolute left-3.5 top-3" />
              <input
                id="email"
                v-model.trim="form.email"
                type="email"
                required
                autocomplete="email"
                placeholder="budi@sman1surakarta.sch.id"
                class="input-field pl-10"
              />
            </div>
          </div>

          <div>
            <label for="institution" class="block text-xs font-bold mb-1.5">Sekolah / Instansi</label>
            <div class="relative">
              <Building2 class="w-4 h-4 text-ink-muted absolute left-3.5 top-3" />
              <input
                id="institution"
                v-model.trim="form.institution"
                type="text"
                required
                placeholder="Contoh: SMA Negeri 1 Surakarta"
                class="input-field pl-10"
              />
            </div>
          </div>

          <div>
            <label for="password" class="block text-xs font-bold mb-1.5">Kata Sandi (Minimal 6 Karakter)</label>
            <div class="relative">
              <Lock class="w-4 h-4 text-ink-muted absolute left-3.5 top-3" />
              <input
                id="password"
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                required
                minlength="6"
                autocomplete="new-password"
                placeholder="Minimal 6 karakter"
                class="input-field pl-10 pr-10"
              />
              <button
                type="button"
                class="absolute right-3 top-2.5 text-ink-muted hover:text-ink dark:hover:text-ink-darkText p-0.5"
                @click="showPassword = !showPassword"
                :aria-label="showPassword ? 'Sembunyikan sandi' : 'Tampilkan sandi'"
              >
                <EyeOff v-if="showPassword" class="w-4 h-4" />
                <Eye v-else class="w-4 h-4" />
              </button>
            </div>
          </div>

          <button type="submit" class="btn-primary w-full py-3" :disabled="authStore.loading">
            <Loader2 v-if="authStore.loading" class="w-4 h-4 animate-spin" />
            <UserPlus v-else class="w-4 h-4" />
            <span>{{ authStore.loading ? 'Mendaftarkan Akun...' : 'Daftar & Aktifkan Paket Gratis' }}</span>
          </button>
        </form>

        <div class="pt-4 border-t border-ink-line dark:border-ink-darkLine text-center text-xs text-ink-muted">
          Sudah memiliki akun?
          <RouterLink to="/masuk" class="font-bold text-brand-700 dark:text-brand-400 hover:underline ml-1">
            Masuk di Sini
          </RouterLink>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowLeft,
  PenTool,
  User,
  Mail,
  Building2,
  Lock,
  Eye,
  EyeOff,
  UserPlus,
  Loader2,
  CircleAlert,
  Sun,
  Moon
} from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useToastStore } from '@/stores/toast'

const router = useRouter()
const authStore = useAuthStore()
const themeStore = useThemeStore()
const toastStore = useToastStore()

const showPassword = ref(false)
const errorMessage = ref('')
const form = reactive({
  name: '',
  email: '',
  institution: '',
  password: ''
})

async function handleSubmit() {
  errorMessage.value = ''
  if (form.password.length < 6) {
    errorMessage.value = 'Kata sandi harus terdiri dari minimal 6 karakter.'
    return
  }
  try {
    const user = await authStore.register(form)
    toastStore.success(`Pendaftaran berhasil! Selamat datang, ${user.name}.`)
    router.push('/guru')
  } catch (err) {
    errorMessage.value = err.message || 'Gagal mendaftarkan akun baru.'
  }
}
</script>

