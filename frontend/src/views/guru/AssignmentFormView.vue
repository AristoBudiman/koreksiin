<template>
  <div class="max-w-4xl mx-auto space-y-6">
    <div class="flex items-center justify-between gap-4">
      <RouterLink
        :to="isEdit ? `/guru/tugas/${assignmentId}` : '/guru'"
        class="inline-flex items-center gap-1.5 text-xs font-semibold text-ink-muted hover:text-brand-700 dark:hover:text-brand-400 transition-colors"
      >
        <ArrowLeft class="w-4 h-4" />
        <span>{{ isEdit ? 'Kembali ke Detail Tugas' : 'Kembali ke Daftar Tugas' }}</span>
      </RouterLink>

      <button
        v-if="!isEdit"
        type="button"
        class="btn-secondary text-xs px-3 py-1.5"
        @click="fillPhotosynthesisSample"
      >
        <Sparkles class="w-3.5 h-3.5 text-brand-700 dark:text-brand-400" />
        <span>Isi Contoh Soal &amp; Rubrik Fotosintesis</span>
      </button>
    </div>

    <div>
      <HandDrawnUnderline>
        <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
          {{ isEdit ? 'Edit Tugas & Rubrik Penilaian' : 'Buat Tugas & Rubrik Baru' }}
        </h1>
      </HandDrawnUnderline>
      <p class="text-xs sm:text-sm text-ink-muted mt-2">
        Tentukan soal esai, nilai maksimal, model AI yang digunakan, serta kriteria rubrik beserta poin kunci jawabannya.
      </p>
    </div>

    <div
      v-if="errorMessage"
      class="p-3.5 rounded-lg bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 flex items-start gap-2.5 text-xs font-semibold text-brand-700 dark:text-brand-300"
    >
      <CircleAlert class="w-4 h-4 shrink-0 mt-0.5" />
      <span>{{ errorMessage }}</span>
    </div>

    <form class="space-y-6" @submit.prevent="handleSubmit">
      <!-- Informasi Dasar Tugas -->
      <div class="card-paper p-5 sm:p-6 space-y-4">
        <h2 class="font-extrabold text-base flex items-center gap-2">
          <FileText class="w-4 h-4 text-brand-700 dark:text-brand-400" />
          <span>1. Informasi Tugas &amp; Soal Esai</span>
        </h2>

        <div class="grid sm:grid-cols-2 gap-4">
          <div>
            <label class="block text-xs font-bold mb-1.5">Judul Tugas</label>
            <input
              v-model.trim="form.title"
              type="text"
              required
              placeholder="Contoh: Uji Pemahaman Konsep Fotosintesis"
              class="input-field"
            />
          </div>

          <div>
            <label class="block text-xs font-bold mb-1.5">Mata Pelajaran &amp; Kelas</label>
            <input
              v-model.trim="form.subject"
              type="text"
              required
              placeholder="Contoh: Biologi Kelas X"
              class="input-field"
            />
          </div>
        </div>

        <div>
          <label class="block text-xs font-bold mb-1.5">Teks Soal Esai</label>
          <textarea
            v-model.trim="form.question_text"
            rows="3"
            required
            placeholder="Tuliskan pertanyaan esai yang diberikan kepada siswa..."
            class="input-field"
          ></textarea>
        </div>

        <div class="grid sm:grid-cols-2 gap-4">
          <div>
            <label class="block text-xs font-bold mb-1.5">Nilai Maksimal</label>
            <input
              v-model.number="form.max_score"
              type="number"
              min="10"
              max="1000"
              step="1"
              required
              class="input-field num-tabular"
            />
          </div>

          <div>
            <label class="block text-xs font-bold mb-1.5">Model Penilaian AI</label>
            <select v-model="form.model_type" class="input-field">
              <option value="base">Model Bawaan (LazarusNLP/all-indobert-base-v4)</option>
              <option value="custom" disabled>Model Custom Guru (Tersedia setelah dilatih di menu Model Custom)</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Rubrik Penilaian -->
      <div class="card-paper p-5 sm:p-6 space-y-5">
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
          <div>
            <h2 class="font-extrabold text-base flex items-center gap-2">
              <ListChecks class="w-4 h-4 text-brand-700 dark:text-brand-400" />
              <span>2. Rubrik Kriteria &amp; Kunci Jawaban</span>
            </h2>
            <p class="text-xs text-ink-muted mt-0.5">
              Total Bobot Saat Ini:
              <strong class="num-tabular text-brand-700 dark:text-brand-400">{{ totalWeight }}</strong>
              (Nilai tiap kriteria dihitung proporsional terhadap total bobot)
            </p>
          </div>

          <button
            type="button"
            class="btn-secondary text-xs px-3.5 py-2"
            @click="addCriterion"
          >
            <Plus class="w-4 h-4" />
            <span>Tambah Kriteria</span>
          </button>
        </div>

        <!-- Panduan Penulisan Poin Kunci (Wajib & Jelas) -->
        <div class="p-4 rounded-xl bg-brand-50/70 dark:bg-brand-950/30 border border-brand-200 dark:border-brand-800/80 space-y-2.5 text-xs">
          <div class="flex items-start gap-2 font-bold text-brand-800 dark:text-brand-200">
            <Info class="w-4 h-4 shrink-0 mt-0.5 text-brand-700 dark:text-brand-400" />
            <span>Petunjuk Penulisan Kunci Jawaban agar Akurasi AI Optimal:</span>
          </div>
          <p class="text-ink dark:text-ink-darkText leading-relaxed pl-6">
            Tulis satu poin per baris sebagai kalimat utuh yang menyebut subjeknya. Contoh:
            <code class="font-bold text-brand-700 dark:text-brand-300">Fotosintesis terjadi di kloroplas.</code>
          </p>
          <div class="grid sm:grid-cols-2 gap-3 pl-6 pt-1">
            <div class="p-2.5 rounded-lg bg-white dark:bg-ink-darkCard border border-green-300 dark:border-green-800/70 space-y-1">
              <div class="flex items-center gap-1.5 font-bold text-criteria-terpenuhi">
                <CircleCheck class="w-3.5 h-3.5" />
                <span>Contoh Baik (Kalimat Utuh + Subjek):</span>
              </div>
              <p class="text-[11px] text-ink-muted font-mono leading-snug">
                Reaksi berlangsung di dalam organel kloroplas pada daun.<br />
                Daun memiliki pigmen klorofil untuk menyerap cahaya matahari.
              </p>
            </div>
            <div class="p-2.5 rounded-lg bg-white dark:bg-ink-darkCard border border-amber-300 dark:border-amber-800/70 space-y-1">
              <div class="flex items-center gap-1.5 font-bold text-criteria-sebagian">
                <CircleAlert class="w-3.5 h-3.5" />
                <span>Contoh Kurang Baik (Potongan Kata Tanpa Subjek):</span>
              </div>
              <p class="text-[11px] text-ink-muted font-mono leading-snug">
                kloroplas<br />
                cahaya matahari
              </p>
            </div>
          </div>
        </div>

        <!-- Daftar Kriteria -->
        <div class="space-y-4">
          <div
            v-for="(crit, idx) in form.criteria"
            :key="idx"
            class="p-4 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine space-y-3"
          >
            <div class="flex items-center justify-between gap-2">
              <span class="text-xs font-extrabold uppercase tracking-wider text-brand-700 dark:text-brand-400 num-tabular">
                Kriteria #{{ idx + 1 }}
              </span>
              <button
                v-if="form.criteria.length > 1"
                type="button"
                class="text-xs font-semibold text-brand-700 dark:text-brand-400 hover:underline inline-flex items-center gap-1"
                @click="removeCriterion(idx)"
              >
                <Trash2 class="w-3.5 h-3.5" />
                <span>Hapus</span>
              </button>
            </div>

            <div class="grid sm:grid-cols-12 gap-3">
              <div class="sm:col-span-9">
                <label class="block text-xs font-bold mb-1">Nama Kriteria</label>
                <input
                  v-model.trim="crit.name"
                  type="text"
                  required
                  placeholder="Contoh: Tempat Terjadinya dan Peran Klorofil"
                  class="input-field"
                />
              </div>
              <div class="sm:col-span-3">
                <label class="block text-xs font-bold mb-1">Bobot Kriteria</label>
                <input
                  v-model.number="crit.weight"
                  type="number"
                  min="1"
                  max="100"
                  step="1"
                  required
                  class="input-field num-tabular"
                />
              </div>
            </div>

            <div>
              <label class="block text-xs font-bold mb-1">
                Kunci Jawaban (Satu poin kalimat utuh per baris)
              </label>
              <textarea
                v-model="crit.answer_key"
                rows="3"
                required
                placeholder="Tulis satu poin per baris sebagai kalimat utuh yang menyebut subjeknya. Contoh: Fotosintesis terjadi di kloroplas."
                class="input-field text-xs leading-relaxed"
              ></textarea>
              <p class="text-[11px] text-ink-muted mt-1">
                Tulis satu poin per baris sebagai kalimat utuh yang menyebut subjeknya. Contoh: Fotosintesis terjadi di kloroplas.
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- Tombol Simpan -->
      <div class="flex items-center justify-end gap-3">
        <RouterLink
          :to="isEdit ? `/guru/tugas/${assignmentId}` : '/guru'"
          class="btn-secondary"
        >
          Batal
        </RouterLink>
        <button type="submit" class="btn-primary px-6" :disabled="saving">
          <Loader2 v-if="saving" class="w-4 h-4 animate-spin" />
          <Save v-else class="w-4 h-4" />
          <span>{{ saving ? 'Menyimpan...' : isEdit ? 'Simpan Perubahan' : 'Buat Tugas & Lanjut Upload Jawaban' }}</span>
        </button>
      </div>
    </form>
  </div>
</template>

<script setup>
import { reactive, ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  Sparkles,
  FileText,
  ListChecks,
  Plus,
  Trash2,
  Info,
  CircleCheck,
  CircleAlert,
  Save,
  Loader2
} from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import { apiFetch } from '@/utils/api'
import { useToastStore } from '@/stores/toast'

const route = useRoute()
const router = useRouter()
const toastStore = useToastStore()

const assignmentId = computed(() => route.params.id)
const isEdit = computed(() => !!assignmentId.value)

const saving = ref(false)
const errorMessage = ref('')

const form = reactive({
  title: '',
  subject: '',
  question_text: '',
  max_score: 100,
  model_type: 'base',
  custom_model_id: null,
  criteria: [
    { name: '', weight: 35, answer_key: '' },
    { name: '', weight: 35, answer_key: '' },
    { name: '', weight: 30, answer_key: '' }
  ]
})

const totalWeight = computed(() =>
  form.criteria.reduce((acc, item) => acc + (Number(item.weight) || 0), 0)
)

function addCriterion() {
  form.criteria.push({
    name: '',
    weight: 25,
    answer_key: ''
  })
}

function removeCriterion(index) {
  if (form.criteria.length > 1) {
    form.criteria.splice(index, 1)
  }
}

function fillPhotosynthesisSample() {
  form.title = 'Uji Pemahaman Konsep Fotosintesis'
  form.subject = 'Biologi Kelas X'
  form.question_text =
    'Jelaskan secara lengkap apa yang dimaksud dengan proses fotosintesis pada tumbuhan hijau, sebutkan bahan baku serta hasil akhirnya, dan jelaskan di mana proses tersebut berlangsung beserta peran pigmen yang terlibat!'
  form.max_score = 100
  form.model_type = 'base'
  form.criteria = [
    {
      name: 'Definisi Proses Fotosintesis',
      weight: 35,
      answer_key:
        'Fotosintesis adalah proses pembuatan atau pembentukan makanan berupa glukosa oleh tumbuhan hijau.\nTumbuhan hijau menggunakan bantuan energi cahaya matahari.'
    },
    {
      name: 'Bahan Baku dan Hasil',
      weight: 35,
      answer_key:
        'Tumbuhan membutuhkan bahan baku berupa karbon dioksida (CO2) dan air (H2O).\nReaksi ini menghasilkan glukosa dan gas oksigen (O2).'
    },
    {
      name: 'Tempat Terjadinya dan Peran Klorofil',
      weight: 30,
      answer_key:
        'Reaksi berlangsung di dalam organel kloroplas pada daun.\nDaun memiliki pigmen klorofil untuk menangkap atau menyerap cahaya matahari.'
    }
  ]
  toastStore.success('Contoh soal dan rubrik Fotosintesis berhasil diisikan.')
}

async function loadExistingAssignment() {
  if (!isEdit.value) return
  try {
    const res = await apiFetch(`/api/assignments/${assignmentId.value}`)
    const a = res.assignment
    form.title = a.title
    form.subject = a.subject
    form.question_text = a.question_text
    form.max_score = a.max_score
    form.model_type = a.model_type || 'base'
    form.custom_model_id = a.custom_model_id || null
    if (res.criteria?.length) {
      form.criteria = res.criteria.map((c) => ({
        name: c.name,
        weight: c.weight,
        answer_key: c.answer_key
      }))
    }
  } catch (err) {
    errorMessage.value = err.message || 'Gagal memuat data tugas.'
  }
}

async function handleSubmit() {
  errorMessage.value = ''
  saving.value = true
  try {
    if (isEdit.value) {
      await apiFetch(`/api/assignments/${assignmentId.value}`, {
        method: 'PUT',
        body: JSON.stringify(form)
      })
      toastStore.success('Tugas dan rubrik berhasil diperbarui.')
      router.push(`/guru/tugas/${assignmentId.value}`)
    } else {
      const res = await apiFetch('/api/assignments', {
        method: 'POST',
        body: JSON.stringify(form)
      })
      toastStore.success('Tugas baru berhasil dibuat.')
      router.push(`/guru/tugas/${res.id}`)
    }
  } catch (err) {
    errorMessage.value = err.message || 'Gagal menyimpan tugas.'
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadExistingAssignment()
})
</script>

