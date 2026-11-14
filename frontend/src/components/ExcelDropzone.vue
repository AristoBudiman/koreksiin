<template>
  <div class="space-y-4">
    <!-- Area Drag and Drop -->
    <div
      class="relative border-2 border-dashed rounded-xl p-6 sm:p-8 text-center transition-all duration-200 cursor-pointer"
      :class="[
        isDragging
          ? 'border-brand-500 bg-brand-50/70 dark:bg-brand-900/20 scale-[1.01]'
          : selectedFile
            ? 'border-criteria-terpenuhi bg-green-50/40 dark:bg-green-950/20'
            : 'border-ink-line dark:border-ink-darkLine hover:border-brand-400 hover:bg-brand-50/20 dark:hover:bg-ink-darkLine/40'
      ]"
      @dragover.prevent="isDragging = true"
      @dragleave.prevent="isDragging = false"
      @drop.prevent="onDrop"
      @click="triggerFileInput"
    >
      <input
        ref="fileInput"
        type="file"
        accept=".xlsx"
        class="hidden"
        @change="onFileSelected"
      />

      <div class="flex flex-col items-center gap-3">
        <div
          class="w-12 h-12 rounded-xl flex items-center justify-center transition-transform duration-200"
          :class="[
            isDragging ? 'scale-110 bg-brand-500 text-white' : '',
            selectedFile && !isDragging
              ? 'bg-green-100 text-criteria-terpenuhi dark:bg-green-900/40 animate-bounce'
              : 'bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400'
          ]"
          style="animation-iteration-count: 1;"
        >
          <Loader2 v-if="validating" class="w-6 h-6 animate-spin" />
          <FileCheck2 v-else-if="selectedFile" class="w-6 h-6" />
          <FileSpreadsheet v-else class="w-6 h-6" />
        </div>

        <div v-if="!selectedFile" class="space-y-1">
          <p class="text-sm font-bold">
            Seret &amp; lepas file Excel (<code class="text-brand-700 dark:text-brand-400">.xlsx</code>) di sini, atau klik untuk memilih
          </p>
          <p class="text-xs text-ink-muted">
            Kolom wajib: <code class="font-semibold">nama_siswa</code> dan <code class="font-semibold">jawaban</code> (panjang jawaban tidak dibatasi)
          </p>
        </div>

        <div v-else class="space-y-1">
          <p class="text-sm font-bold text-ink dark:text-ink-darkText">
            {{ selectedFile.name }}
          </p>
          <p class="text-xs text-ink-muted num-tabular">
            {{ (selectedFile.size / 1024).toFixed(1) }} KB — Klik untuk mengganti file
          </p>
        </div>
      </div>
    </div>

    <!-- Ringkasan Validasi File -->
    <div v-if="validationSummary" class="card-paper p-4 space-y-3">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-center gap-4 text-xs font-bold">
          <span class="inline-flex items-center gap-1.5 text-criteria-terpenuhi">
            <CircleCheck class="w-4 h-4" />
            <span class="num-tabular">{{ validationSummary.valid_count }} baris valid</span>
          </span>
          <span
            v-if="validationSummary.error_count > 0"
            class="inline-flex items-center gap-1.5 text-brand-700 dark:text-brand-400"
          >
            <CircleAlert class="w-4 h-4" />
            <span class="num-tabular">{{ validationSummary.error_count }} baris bermasalah</span>
          </span>
        </div>

        <div class="flex items-center gap-2">
          <button
            type="button"
            class="btn-secondary px-3 py-1.5 text-xs"
            @click="resetFile"
          >
            <X class="w-3.5 h-3.5" />
            <span>Batal</span>
          </button>
          <button
            v-if="validationSummary.valid_count > 0 && validationSummary.error_count === 0"
            type="button"
            class="btn-primary px-4 py-1.5 text-xs"
            :disabled="uploading"
            @click="confirmUpload"
          >
            <Loader2 v-if="uploading" class="w-3.5 h-3.5 animate-spin" />
            <Upload v-else class="w-3.5 h-3.5" />
            <span>{{ uploading ? 'Mengunggah...' : `Simpan ${validationSummary.valid_count} Jawaban Siswa` }}</span>
          </button>
        </div>
      </div>

      <!-- Daftar Error per Baris -->
      <div
        v-if="validationSummary.row_errors?.length"
        class="p-3 rounded-lg bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 space-y-1.5"
      >
        <p class="text-xs font-bold text-brand-700 dark:text-brand-300 flex items-center gap-1.5">
          <CircleAlert class="w-4 h-4 shrink-0" />
          <span>Perbaiki baris berikut di file Excel Anda sebelum mengunggah:</span>
        </p>
        <ul class="text-xs text-brand-800 dark:text-brand-200 space-y-1 pl-5 list-disc max-h-40 overflow-y-auto">
          <li v-for="err in validationSummary.row_errors" :key="err.row" class="num-tabular">
            <strong>Baris {{ err.row }}:</strong> {{ err.message }}
          </li>
        </ul>
      </div>
    </div>

    <div
      v-if="errorMsg"
      class="p-3 rounded-lg bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 flex items-start gap-2 text-xs font-semibold text-brand-700 dark:text-brand-300"
    >
      <CircleAlert class="w-4 h-4 shrink-0 mt-0.5" />
      <span>{{ errorMsg }}</span>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import {
  FileSpreadsheet,
  FileCheck2,
  Loader2,
  CircleCheck,
  CircleAlert,
  Upload,
  X
} from 'lucide-vue-next'
import { apiFetch } from '@/utils/api'
import { useToastStore } from '@/stores/toast'

const props = defineProps({
  assignmentId: {
    type: [Number, String],
    required: true
  }
})

const emit = defineEmits(['uploaded'])
const toastStore = useToastStore()

const fileInput = ref(null)
const isDragging = ref(false)
const selectedFile = ref(null)
const validating = ref(false)
const uploading = ref(false)
const validationSummary = ref(null)
const errorMsg = ref('')

function triggerFileInput() {
  fileInput.value?.click()
}

function onFileSelected(e) {
  const file = e.target.files?.[0]
  if (file) {
    handleFile(file)
  }
}

function onDrop(e) {
  isDragging.value = false
  const file = e.dataTransfer?.files?.[0]
  if (file) {
    handleFile(file)
  }
}

function resetFile() {
  selectedFile.value = null
  validationSummary.value = null
  errorMsg.value = ''
  if (fileInput.value) {
    fileInput.value.value = ''
  }
}

async function handleFile(file) {
  errorMsg.value = ''
  validationSummary.value = null

  if (!file.name.toLowerCase().endsWith('.xlsx')) {
    errorMsg.value = 'Format file harus berupa dokumen Excel (.xlsx).'
    return
  }

  selectedFile.value = file
  validating.value = true
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await apiFetch(`/api/assignments/${props.assignmentId}/submissions/excel?dry_run=true`, {
      method: 'POST',
      body: fd
    })
    validationSummary.value = res
  } catch (err) {
    errorMsg.value = err.message || 'Gagal memvalidasi file Excel.'
  } finally {
    validating.value = false
  }
}

async function confirmUpload() {
  if (!selectedFile.value) return
  uploading.value = true
  errorMsg.value = ''
  try {
    const fd = new FormData()
    fd.append('file', selectedFile.value)
    const res = await apiFetch(`/api/assignments/${props.assignmentId}/submissions/excel`, {
      method: 'POST',
      body: fd
    })
    toastStore.success(res.message || 'Jawaban siswa berhasil diunggah.')
    resetFile()
    emit('uploaded', res)
  } catch (err) {
    errorMsg.value = err.message || 'Gagal menyimpan jawaban dari file Excel.'
  } finally {
    uploading.value = false
  }
}
</script>

