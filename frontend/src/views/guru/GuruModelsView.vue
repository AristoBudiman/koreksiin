<template>
  <div class="space-y-8">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
      <div>
        <HandDrawnUnderline>
          <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
            Model Custom Guru
          </h1>
        </HandDrawnUnderline>
        <p class="text-xs sm:text-sm text-ink-muted mt-2 max-w-3xl">
          Sesuaikan standar penilaian esai mata pelajaran atau kelas Anda di atas model dasar
          <strong class="text-ink dark:text-ink-darkText">LazarusNLP/all-indobert-base-v4</strong>
          menggunakan mode <strong>Kalibrasi</strong> (default, cepat &amp; akurat) maupun opsi
          <strong>Fine-tune mendalam (LoRA)</strong> dengan evaluasi <strong>5-Fold Cross-Validation</strong>.
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2.5">
        <a
          href="/api/templates/excel?type=training"
          class="btn-secondary text-xs"
          download
        >
          <FileSpreadsheet class="w-4 h-4 text-brand-700 dark:text-brand-400" />
          <span>Template Excel Data Latih</span>
        </a>
        <button
          type="button"
          class="btn-primary text-xs sm:text-sm"
          @click="openCreateModal"
        >
          <Plus class="w-4 h-4" />
          <span>Buat Model Custom Baru</span>
        </button>
      </div>
    </div>

    <!-- Seksi Unggah Data Latih Excel ke Tugas -->
    <div class="card-paper p-5 sm:p-6 space-y-4">
      <div class="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        <div class="space-y-1">
          <h2 class="font-extrabold text-base flex items-center gap-2">
            <Database class="w-4 h-4 text-brand-700 dark:text-brand-400" />
            <span>Bank Data Latih per Tugas (Revisi Nilai &amp; Impor Excel)</span>
          </h2>
          <p class="text-xs text-ink-muted">
            Pilih tugas untuk melihat sampel data latih atau mengunggah file Excel berisi kolom
            <code class="font-bold">nama_siswa</code>, <code class="font-bold">jawaban</code>, dan <code class="font-bold">nilai</code>.
          </p>
        </div>

        <div class="flex flex-wrap items-center gap-2.5">
          <select
            v-model="selectedAssignmentId"
            class="input-field text-xs py-2 min-w-[240px]"
            @change="fetchAssignmentSamples"
          >
            <option :value="null">-- Pilih Tugas Sumber Data Latih --</option>
            <option v-for="a in assignments" :key="a.id" :value="a.id">
              {{ a.title }} ({{ a.subject }}) — {{ a.training_sample_count || 0 }} sampel
            </option>
          </select>

          <button
            v-if="selectedAssignmentId"
            type="button"
            class="btn-secondary text-xs py-2 px-3"
            @click="showSamplesList = !showSamplesList"
          >
            <Eye class="w-4 h-4" />
            <span>{{ showSamplesList ? 'Sembunyikan Sampel' : `Lihat Sampel (${assignmentSamples.length})` }}</span>
          </button>
        </div>
      </div>

      <div v-if="selectedAssignmentId" class="pt-3 border-t border-ink-line dark:border-ink-darkLine space-y-4">
        <div class="grid lg:grid-cols-12 gap-5 items-start">
          <div class="lg:col-span-6 space-y-2">
            <label class="block text-xs font-bold">
              Unggah File Excel Nilai Guru (.xlsx) untuk Tugas Ini
            </label>
            <ExcelDropzone
              accept=".xlsx"
              label="Tarik & lepas file Excel nilai latih (.xlsx) atau klik untuk memilih"
              hint="Kolom wajib: nama_siswa, jawaban, nilai"
              @file-selected="handleTrainingExcelSelected"
            />
          </div>

          <div class="lg:col-span-6 space-y-3">
            <div v-if="previewLoading" class="p-4 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine flex items-center gap-2.5 text-xs">
              <Loader2 class="w-4 h-4 animate-spin text-brand-500" />
              <span>Memeriksa baris data latih di file Excel...</span>
            </div>

            <div
              v-else-if="excelPreview"
              class="p-4 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine space-y-3 text-xs"
            >
              <div class="flex items-center justify-between">
                <span class="font-extrabold">Pratinjau Validasi File Excel</span>
                <span class="text-ink-muted num-tabular">Maks Nilai: {{ excelPreview.max_score }}</span>
              </div>

              <div class="flex flex-wrap items-center gap-3">
                <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300 font-bold num-tabular">
                  <CircleCheck class="w-3.5 h-3.5" />
                  {{ excelPreview.valid_count }} baris valid
                </span>
                <span
                  v-if="excelPreview.error_count > 0"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300 font-bold num-tabular"
                >
                  <CircleAlert class="w-3.5 h-3.5" />
                  {{ excelPreview.error_count }} baris dilewati
                </span>
              </div>

              <div v-if="excelPreview.error_rows?.length" class="max-h-28 overflow-y-auto space-y-1 p-2 rounded bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine">
                <div v-for="errRow in excelPreview.error_rows" :key="errRow.row_number" class="text-[11px] text-brand-700 dark:text-brand-300">
                  Baris {{ errRow.row_number }} ({{ errRow.student_name }}): {{ errRow.reason }}
                </div>
              </div>

              <div class="flex items-center justify-end gap-2 pt-1">
                <button
                  type="button"
                  class="btn-secondary text-xs py-1.5 px-3"
                  @click="clearExcelPreview"
                >
                  Batal
                </button>
                <button
                  type="button"
                  class="btn-primary text-xs py-1.5 px-4"
                  :disabled="uploadingExcel || excelPreview.valid_count === 0"
                  @click="confirmUploadTrainingExcel"
                >
                  <Loader2 v-if="uploadingExcel" class="w-3.5 h-3.5 animate-spin" />
                  <Upload v-else class="w-3.5 h-3.5" />
                  <span>Simpan {{ excelPreview.valid_count }} Sampel Data Latih</span>
                </button>
              </div>
            </div>

            <div v-else class="p-4 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-xs text-ink-muted space-y-1.5">
              <p class="font-bold text-ink dark:text-ink-darkText">Cara Mengumpulkan Data Latih:</p>
              <p>1. Setiap kali Anda merevisi nilai siswa di halaman detail tugas, jawaban dan nilai tersebut otomatis tersimpan sebagai sampel data latih.</p>
              <p>2. Anda juga dapat mengunggah langsung file Excel berisi nilai koreksi guru untuk melatih model custom.</p>
            </div>
          </div>
        </div>

        <!-- Tabel Sampel Data Latih Tugas Terpilih -->
        <div v-if="showSamplesList" class="pt-2">
          <div v-if="assignmentSamples.length === 0" class="p-6 text-center text-xs text-ink-muted border border-dashed border-ink-line dark:border-ink-darkLine rounded-xl">
            Belum ada sampel data latih pada tugas ini.
          </div>
          <div v-else class="overflow-x-auto border border-ink-line dark:border-ink-darkLine rounded-xl max-h-72">
            <table class="w-full text-left border-collapse text-xs">
              <thead class="bg-ink-bg dark:bg-ink-darkBg border-b border-ink-line dark:border-ink-darkLine sticky top-0">
                <tr>
                  <th class="py-2.5 px-3 font-bold">Siswa</th>
                  <th class="py-2.5 px-3 font-bold">Potongan Jawaban</th>
                  <th class="py-2.5 px-3 font-bold text-right">Target Nilai</th>
                  <th class="py-2.5 px-3 font-bold text-center">Sumber</th>
                  <th class="py-2.5 px-3 font-bold text-right">Aksi</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-ink-line dark:divide-ink-darkLine">
                <tr v-for="s in assignmentSamples" :key="s.id" class="hover:bg-ink-bg/60 dark:hover:bg-ink-darkBg/40">
                  <td class="py-2 px-3 font-semibold whitespace-nowrap">{{ s.student_name }}</td>
                  <td class="py-2 px-3 text-ink-muted line-clamp-1 max-w-md">{{ s.answer_text }}</td>
                  <td class="py-2 px-3 text-right font-extrabold num-tabular text-brand-700 dark:text-brand-400">
                    {{ Number(s.target_score).toFixed(1) }}
                  </td>
                  <td class="py-2 px-3 text-center">
                    <span class="px-2 py-0.5 rounded text-[10px] font-bold uppercase bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine">
                      {{ s.source === 'revision' ? 'Revisi Guru' : 'Impor Excel' }}
                    </span>
                  </td>
                  <td class="py-2 px-3 text-right">
                    <button
                      type="button"
                      class="text-ink-muted hover:text-brand-700 p-1"
                      title="Hapus sampel"
                      @click="handleDeleteSample(s.id)"
                    >
                      <Trash2 class="w-3.5 h-3.5" />
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- Daftar Model Custom Guru -->
    <div class="space-y-4">
      <div class="flex items-center justify-between">
        <h2 class="text-lg font-extrabold tracking-tight">Daftar Model Custom Anda</h2>
        <span class="text-xs text-ink-muted num-tabular">
          Minimal sampel pelatihan: <strong>{{ minTrainSamples }} jawaban</strong>
        </span>
      </div>

      <div v-if="loading" class="grid lg:grid-cols-2 gap-5">
        <div v-for="n in 2" :key="n" class="card-paper p-6 animate-pulse space-y-4">
          <div class="h-5 bg-ink-line dark:bg-ink-darkLine rounded w-1/3"></div>
          <div class="h-4 bg-ink-line dark:bg-ink-darkLine rounded w-2/3"></div>
          <div class="h-24 bg-ink-line dark:bg-ink-darkLine rounded w-full"></div>
        </div>
      </div>

      <div v-else-if="models.length === 0" class="card-paper p-10 text-center space-y-4">
        <div class="w-12 h-12 rounded-xl bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center mx-auto">
          <Sliders class="w-6 h-6" />
        </div>
        <div class="max-w-md mx-auto space-y-1.5">
          <h3 class="font-extrabold text-lg">Belum Ada Model Custom</h3>
          <p class="text-xs sm:text-sm text-ink-muted">
            Buat model custom (misalnya "Biologi Kelas 10"), pilih tugas sumber data latihnya, dan jalankan pelatihan di atas model <strong>all-indobert-base-v4</strong>.
          </p>
        </div>
        <button type="button" class="btn-primary" @click="openCreateModal">
          <Plus class="w-4 h-4" />
          <span>Buat Model Custom Pertamamu</span>
        </button>
      </div>

      <div v-else class="grid lg:grid-cols-2 gap-6">
        <div
          v-for="m in models"
          :key="m.id"
          class="card-paper p-5 sm:p-6 flex flex-col justify-between space-y-5"
        >
          <div class="space-y-4">
            <!-- Header Kartu Model -->
            <div class="flex items-start justify-between gap-3">
              <div>
                <div class="flex flex-wrap items-center gap-2">
                  <h3 class="font-extrabold text-lg">{{ m.name }}</h3>
                  <span
                    v-if="m.active_version"
                    class="px-2.5 py-0.5 rounded bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300 text-xs font-bold inline-flex items-center gap-1"
                  >
                    <CircleCheck class="w-3.5 h-3.5" />
                    <span>
                      Aktif: v{{ m.active_version.version_num }}
                      ({{ formatModeLabel(m.active_version.training_mode) }})
                    </span>
                  </span>
                  <span
                    v-else
                    class="px-2.5 py-0.5 rounded bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-xs font-semibold text-ink-muted"
                  >
                    Belum Ada Versi Aktif
                  </span>
                </div>
                <p class="text-xs text-ink-muted mt-1">
                  {{ m.description || 'Model custom berbasis LazarusNLP/all-indobert-base-v4 (8 fitur agregat poin kunci)' }}
                </p>
              </div>

              <div class="flex items-center gap-1.5 shrink-0">
                <button
                  type="button"
                  class="btn-secondary p-2 text-xs"
                  title="Edit tugas sumber & nama model"
                  @click="openEditModal(m)"
                >
                  <Edit3 class="w-3.5 h-3.5" />
                </button>
                <button
                  type="button"
                  class="btn-secondary p-2 text-xs hover:text-brand-700"
                  title="Hapus model custom"
                  @click="handleDeleteModel(m)"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              </div>
            </div>

            <!-- Info Arsitektur & Sampel Data Latih -->
            <div class="p-3.5 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine space-y-2.5 text-xs">
              <div class="flex items-center justify-between">
                <span class="text-ink-muted font-semibold">Model Dasar:</span>
                <code class="font-bold text-brand-700 dark:text-brand-400">LazarusNLP/all-indobert-base-v4</code>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-ink-muted font-semibold">Total Sampel Tersedia:</span>
                <span
                  class="font-extrabold num-tabular"
                  :class="m.total_samples >= minTrainSamples ? 'text-criteria-terpenuhi' : 'text-criteria-sebagian'"
                >
                  {{ m.total_samples }} / {{ minTrainSamples }} sampel minimum
                </span>
              </div>

              <div class="pt-1 border-t border-ink-line dark:border-ink-darkLine">
                <span class="text-[11px] font-bold uppercase text-ink-muted block mb-1.5">
                  Tugas Sumber Data Latih ({{ m.linked_assignments?.length || 0 }} tugas):
                </span>
                <div v-if="m.linked_assignments?.length" class="flex flex-wrap gap-1.5">
                  <span
                    v-for="la in m.linked_assignments"
                    :key="la.id"
                    class="px-2 py-0.5 rounded bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine text-[11px] font-semibold num-tabular"
                  >
                    {{ la.title }} ({{ la.sample_count }} sampel)
                  </span>
                </div>
                <p v-else class="text-[11px] text-brand-700 dark:text-brand-300 font-semibold">
                  Belum ada tugas yang dihubungkan. Klik tombol edit untuk memilih tugas.
                </p>
              </div>
            </div>

            <!-- Pengaturan Mode Pelatihan (Default: Kalibrasi, Opsi Checkbox: Fine-tune mendalam LoRA) -->
            <div class="p-3.5 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine space-y-2.5 text-xs">
              <div class="flex items-center justify-between">
                <span class="font-bold text-ink dark:text-ink-darkText">Mode Pelatihan:</span>
                <span class="text-[11px] font-semibold text-ink-muted num-tabular">
                  Estimasi durasi:
                  <strong class="text-ink dark:text-ink-darkText">
                    {{ loraOptions[m.id] ? '~4 – 6 menit (CPU)' : '~5 – 10 detik' }}
                  </strong>
                </span>
              </div>

              <p class="text-[11px] text-ink-muted leading-relaxed">
                Mode default <strong>Kalibrasi</strong> melatih head differentiable di atas 8 fitur agregat baru tanpa mengubah bobot dasar IndoBERT (evaluasi 5-Fold Cross-Validation, lalu model final dilatih ulang dengan seluruh data).
              </p>

              <label class="flex items-start gap-2.5 pt-1 cursor-pointer select-none">
                <input
                  v-model="loraOptions[m.id]"
                  type="checkbox"
                  class="mt-0.5 rounded border-ink-line text-brand-500 focus:ring-brand-500"
                />
                <div class="space-y-0.5">
                  <span class="font-bold text-ink dark:text-ink-darkText block">
                    Sertakan Fine-tune mendalam (LoRA)
                  </span>
                  <span class="text-[11px] text-ink-muted block">
                    Melatih versi Kalibrasi dan versi LoRA sekaligus, membandingkan MAE CV keduanya, lalu menyimpan keduanya sebagai versi terpisah.
                  </span>
                </div>
              </label>

              <!-- Peringatan bila data latih < 50 sampel dan LoRA dicentang -->
              <div
                v-if="loraOptions[m.id] && m.total_samples < 50"
                class="p-2.5 rounded-lg bg-amber-50 dark:bg-amber-950/40 border border-amber-300 dark:border-amber-800/80 flex items-start gap-2 text-[11px] text-amber-900 dark:text-amber-200 font-medium"
              >
                <CircleAlert class="w-4 h-4 text-criteria-sebagian shrink-0 mt-0.5" />
                <span>
                  Fine-tune mendalam biasanya baru unggul dengan data yang lebih banyak, dan training butuh beberapa menit.
                </span>
              </div>
            </div>

            <!-- Progres Job Training bila sedang berjalan (per fold dan per epoch) -->
            <div
              v-if="m.latest_job && (m.latest_job.status === 'queued' || m.latest_job.status === 'processing')"
              class="p-4 rounded-xl border-2 border-brand-500/40 bg-brand-50/40 dark:bg-brand-950/20 space-y-2.5 text-xs"
            >
              <div class="flex items-center justify-between gap-2">
                <div class="flex items-center gap-2 font-bold text-brand-700 dark:text-brand-300">
                  <Loader2 class="w-4 h-4 animate-spin shrink-0" />
                  <span>{{ m.latest_job.stage }}</span>
                </div>
                <span class="font-extrabold num-tabular text-brand-700 dark:text-brand-400">
                  {{ m.latest_job.progress }}%
                </span>
              </div>
              <div class="w-full h-2 rounded-full bg-ink-line dark:bg-ink-darkLine overflow-hidden">
                <div
                  class="h-full bg-brand-500 transition-all duration-300"
                  :style="{ width: `${m.latest_job.progress || 5}%` }"
                ></div>
              </div>
              <div class="flex flex-wrap items-center justify-between gap-2 text-[11px] text-ink-muted num-tabular">
                <span v-if="m.latest_job.metadata?.fold">
                  Fold {{ m.latest_job.metadata.fold }}/{{ m.latest_job.metadata.total_folds }}
                  <template v-if="m.latest_job.metadata?.epoch">
                    — Epoch {{ m.latest_job.metadata.epoch }}/{{ m.latest_job.metadata.total_epochs }}
                  </template>
                </span>
                <span>
                  Perkiraan durasi: {{ m.latest_job.metadata?.use_lora ? '~4 – 6 menit' : '~5 – 10 detik' }}
                </span>
              </div>
            </div>

            <!-- Pesan Error bila job terakhir gagal -->
            <div
              v-else-if="m.latest_job && m.latest_job.status === 'failed'"
              class="p-3 rounded-lg bg-brand-50 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 text-xs text-brand-700 dark:text-brand-300 flex items-start gap-2"
            >
              <CircleAlert class="w-4 h-4 shrink-0 mt-0.5" />
              <span>Pelatihan terakhir gagal: {{ m.latest_job.error_message }}</span>
            </div>

            <!-- Daftar Versi & Metrik 5-Fold Cross-Validation (Tanpa Metrik Data Latih) -->
            <div v-if="m.versions?.length" class="space-y-3">
              <div class="flex items-center justify-between text-xs">
                <span class="font-extrabold uppercase tracking-wider text-ink-muted">
                  Versi Model &amp; Evaluasi 5-Fold Cross-Validation
                </span>
                <span class="text-[11px] text-ink-muted num-tabular">
                  {{ m.versions.length }} versi
                </span>
              </div>

              <div class="space-y-3 max-h-[420px] overflow-y-auto pr-1">
                <div
                  v-for="v in m.versions"
                  :key="v.id"
                  class="p-3.5 rounded-xl border space-y-3 text-xs transition-colors"
                  :class="
                    v.is_active
                      ? 'bg-brand-50/40 dark:bg-brand-950/25 border-brand-400 dark:border-brand-700'
                      : 'bg-ink-bg dark:bg-ink-darkBg border-ink-line dark:border-ink-darkLine'
                  "
                >
                  <!-- Baris Atas Versi: Nomor Versi, Mode, Badge Rekomendasi & Aktif, Tombol Aksi -->
                  <div class="flex flex-wrap items-center justify-between gap-2">
                    <div class="flex flex-wrap items-center gap-1.5">
                      <strong class="text-sm font-extrabold num-tabular">v{{ v.version_num }}</strong>
                      <span class="px-2 py-0.5 rounded bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine text-[11px] font-bold">
                        {{ formatModeLabel(v.training_mode) }}
                      </span>
                      <span
                        v-if="v.is_recommended"
                        class="px-2 py-0.5 rounded bg-brand-100 text-brand-800 dark:bg-brand-900/50 dark:text-brand-200 text-[10px] font-extrabold inline-flex items-center gap-1"
                      >
                        <Award class="w-3 h-3" />
                        <span>Direkomendasikan (MAE CV Terkecil)</span>
                      </span>
                      <span
                        v-if="v.is_active"
                        class="px-2 py-0.5 rounded bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300 text-[10px] font-bold"
                      >
                        Sedang Aktif
                      </span>
                    </div>

                    <div>
                      <button
                        v-if="!v.is_active"
                        type="button"
                        class="btn-primary py-1 px-3 text-xs"
                        @click="handleActivateVersion(m.id, v.id)"
                      >
                        Aktifkan v{{ v.version_num }}
                      </button>
                      <button
                        v-else
                        type="button"
                        class="btn-secondary py-1 px-2.5 text-xs"
                        @click="handleDeactivateModel(m.id)"
                      >
                        Nonaktifkan
                      </button>
                    </div>
                  </div>

                  <!-- Info Jumlah Sampel & Tanggal Training -->
                  <div class="flex flex-wrap items-center justify-between gap-2 text-[11px] text-ink-muted num-tabular">
                    <span>
                      Sampel: <strong>{{ v.sample_count }} jawaban</strong> (Evaluasi 5-Fold CV)
                      <template v-if="v.metadata?.elapsed_seconds">
                        — Durasi {{ v.metadata.elapsed_seconds }} dtk
                      </template>
                    </span>
                    <span>Dilatih: {{ formatDateTime(v.created_at) }}</span>
                  </div>

                  <!-- Perbandingan MAE CV & Pearson CV vs Model Bawaan -->
                  <div class="grid grid-cols-2 gap-2.5">
                    <div class="p-2.5 rounded-lg bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine space-y-1">
                      <span class="text-[10px] font-bold uppercase text-brand-700 dark:text-brand-300 block">
                        Model Custom ({{ formatModeLabel(v.training_mode) }})
                      </span>
                      <div class="flex items-baseline justify-between">
                        <span class="text-ink-muted text-[11px]">MAE 5-Fold CV:</span>
                        <strong class="font-extrabold num-tabular text-sm text-brand-700 dark:text-brand-400">
                          {{ Number(v.mae_custom).toFixed(2) }}
                        </strong>
                      </div>
                      <div class="flex items-baseline justify-between">
                        <span class="text-ink-muted text-[11px]">Pearson CV (r):</span>
                        <strong class="font-extrabold num-tabular text-xs">
                          {{ Number(v.pearson_custom).toFixed(4) }}
                        </strong>
                      </div>
                    </div>

                    <div class="p-2.5 rounded-lg bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine space-y-1">
                      <span class="text-[10px] font-bold uppercase text-ink-muted block">
                        Model Bawaan (all-indobert-base-v4)
                      </span>
                      <div class="flex items-baseline justify-between">
                        <span class="text-ink-muted text-[11px]">MAE 5-Fold CV:</span>
                        <strong class="font-extrabold num-tabular text-sm">
                          {{ Number(v.mae_base).toFixed(2) }}
                        </strong>
                      </div>
                      <div class="flex items-baseline justify-between">
                        <span class="text-ink-muted text-[11px]">Pearson CV (r):</span>
                        <strong class="font-extrabold num-tabular text-xs">
                          {{ Number(v.pearson_base).toFixed(4) }}
                        </strong>
                      </div>
                    </div>
                  </div>

                  <!-- Rata-rata Nilai per Rentang (Prediksi Out-of-Fold CV) -->
                  <div v-if="v.metadata?.range_summary" class="pt-1 space-y-1">
                    <span class="text-[10px] font-bold uppercase text-ink-muted block">
                      Rata-rata Nilai per Kelompok pada Data Uji CV (Target Guru vs Bawaan vs Custom):
                    </span>
                    <div class="grid grid-cols-3 gap-2 text-[11px] num-tabular">
                      <div
                        v-if="v.metadata.range_summary.tinggi?.count"
                        class="p-2 rounded bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine"
                      >
                        <span class="font-bold block text-criteria-terpenuhi">
                          Lengkap ({{ v.metadata.range_summary.tinggi.count }})
                        </span>
                        <span class="text-ink-muted block">Target: {{ v.metadata.range_summary.tinggi.target_mean }}</span>
                        <span class="text-ink-muted block">Bawaan: {{ v.metadata.range_summary.tinggi.base_mean }}</span>
                        <strong class="text-ink dark:text-ink-darkText block">Custom: {{ v.metadata.range_summary.tinggi.custom_mean }}</strong>
                      </div>

                      <div
                        v-if="v.metadata.range_summary.sedang?.count"
                        class="p-2 rounded bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine"
                      >
                        <span class="font-bold block text-criteria-sebagian">
                          Sebagian ({{ v.metadata.range_summary.sedang.count }})
                        </span>
                        <span class="text-ink-muted block">Target: {{ v.metadata.range_summary.sedang.target_mean }}</span>
                        <span class="text-ink-muted block">Bawaan: {{ v.metadata.range_summary.sedang.base_mean }}</span>
                        <strong class="text-brand-700 dark:text-brand-400 block">Custom: {{ v.metadata.range_summary.sedang.custom_mean }}</strong>
                      </div>

                      <div
                        v-if="v.metadata.range_summary.rendah?.count"
                        class="p-2 rounded bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine"
                      >
                        <span class="font-bold block text-ink-muted">
                          Melenceng ({{ v.metadata.range_summary.rendah.count }})
                        </span>
                        <span class="text-ink-muted block">Target: {{ v.metadata.range_summary.rendah.target_mean }}</span>
                        <span class="text-ink-muted block">Bawaan: {{ v.metadata.range_summary.rendah.base_mean }}</span>
                        <strong class="text-ink dark:text-ink-darkText block">Custom: {{ v.metadata.range_summary.rendah.custom_mean }}</strong>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- Footer Aksi Kartu Model -->
          <div class="pt-4 border-t border-ink-line dark:border-ink-darkLine flex items-center justify-between gap-3">
            <span class="text-[11px] text-ink-muted">
              {{ m.version_count || 0 }} versi tersimpan
            </span>

            <button
              type="button"
              class="btn-primary text-xs py-2 px-4"
              :disabled="
                m.total_samples < minTrainSamples ||
                (m.latest_job && (m.latest_job.status === 'queued' || m.latest_job.status === 'processing'))
              "
              @click="handleStartTraining(m)"
            >
              <Cpu class="w-4 h-4" />
              <span>
                {{
                  loraOptions[m.id]
                    ? 'Latih Kalibrasi + LoRA'
                    : m.version_count > 0
                      ? 'Latih Versi Kalibrasi Baru'
                      : 'Latih Model (Kalibrasi)'
                }}
              </span>
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Modal Buat / Edit Model Custom -->
    <div
      v-if="showModal"
      class="fixed inset-0 z-50 bg-black/50 backdrop-blur-xs flex items-center justify-center p-4"
    >
      <div class="card-paper max-w-lg w-full p-6 space-y-5">
        <div class="flex items-center justify-between border-b border-ink-line dark:border-ink-darkLine pb-3">
          <h3 class="font-extrabold text-lg">
            {{ editingModelId ? 'Edit Model Custom' : 'Buat Model Custom Baru' }}
          </h3>
          <button
            type="button"
            class="text-ink-muted hover:text-ink text-xs font-bold"
            @click="closeModal"
          >
            Tutup
          </button>
        </div>

        <form class="space-y-4" @submit.prevent="handleSaveModel">
          <div>
            <label class="block text-xs font-bold mb-1.5">Nama Model Custom</label>
            <input
              v-model.trim="modalForm.name"
              type="text"
              required
              placeholder="Contoh: Biologi Kelas 10"
              class="input-field"
            />
          </div>

          <div>
            <label class="block text-xs font-bold mb-1.5">Deskripsi Singkat (Opsional)</label>
            <input
              v-model.trim="modalForm.description"
              type="text"
              placeholder="Contoh: Standar penilaian esai Biologi SMA Kelas X"
              class="input-field"
            />
          </div>

          <div>
            <label class="block text-xs font-bold mb-1.5">
              Pilih Tugas Sumber Data Latih
            </label>
            <p class="text-[11px] text-ink-muted mb-2">
              Seluruh revisi nilai dan unggahan Excel nilai pada tugas yang dicentang akan digabungkan saat melatih model ini.
            </p>
            <div v-if="assignments.length === 0" class="p-3 rounded-lg bg-ink-bg dark:bg-ink-darkBg text-xs text-ink-muted">
              Belum ada tugas. Buat tugas terlebih dahulu di menu Buat Tugas.
            </div>
            <div v-else class="max-h-48 overflow-y-auto space-y-2 p-3 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine">
              <label
                v-for="a in assignments"
                :key="a.id"
                class="flex items-center justify-between gap-2 p-2 rounded-lg bg-white dark:bg-ink-darkCard border border-ink-line dark:border-ink-darkLine cursor-pointer text-xs"
              >
                <div class="flex items-center gap-2.5">
                  <input
                    v-model="modalForm.assignment_ids"
                    type="checkbox"
                    :value="a.id"
                    class="rounded border-ink-line text-brand-500 focus:ring-brand-500"
                  />
                  <div>
                    <span class="font-bold block">{{ a.title }}</span>
                    <span class="text-[11px] text-ink-muted">{{ a.subject }} (Maks {{ a.max_score }})</span>
                  </div>
                </div>
                <span class="px-2 py-0.5 rounded bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-300 font-bold num-tabular text-[11px]">
                  {{ a.training_sample_count || 0 }} sampel
                </span>
              </label>
            </div>
          </div>

          <div class="flex items-center justify-end gap-2.5 pt-2">
            <button type="button" class="btn-secondary text-xs" @click="closeModal">
              Batal
            </button>
            <button type="submit" class="btn-primary text-xs px-5" :disabled="savingModel">
              <Loader2 v-if="savingModel" class="w-4 h-4 animate-spin" />
              <span>{{ savingModel ? 'Menyimpan...' : 'Simpan Model Custom' }}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onBeforeUnmount } from 'vue'
import {
  FileSpreadsheet,
  Plus,
  Database,
  Eye,
  Loader2,
  CircleCheck,
  CircleAlert,
  Upload,
  Trash2,
  Sliders,
  Edit3,
  Cpu,
  Award
} from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import ExcelDropzone from '@/components/ExcelDropzone.vue'
import { apiFetch } from '@/utils/api'
import { useToastStore } from '@/stores/toast'

const toastStore = useToastStore()

const loading = ref(true)
const models = ref([])
const assignments = ref([])
const minTrainSamples = ref(20)
const loraOptions = reactive({})

const selectedAssignmentId = ref(null)
const assignmentSamples = ref([])
const showSamplesList = ref(false)
const selectedExcelFile = ref(null)
const previewLoading = ref(false)
const excelPreview = ref(null)
const uploadingExcel = ref(false)

const showModal = ref(false)
const editingModelId = ref(null)
const savingModel = ref(false)
const modalForm = reactive({
  name: '',
  description: '',
  assignment_ids: []
})

let pollInterval = null

function formatModeLabel(mode) {
  if (mode === 'lora') return 'Fine-tune mendalam (LoRA)'
  return 'Kalibrasi'
}

function formatDateTime(isoStr) {
  if (!isoStr) return '-'
  return new Date(isoStr).toLocaleString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

async function fetchAllData(silent = false) {
  if (!silent) loading.value = true
  try {
    const [modelsRes, assignRes] = await Promise.all([
      apiFetch('/api/models'),
      apiFetch('/api/assignments')
    ])
    models.value = modelsRes.models || []
    for (const m of models.value) {
      if (loraOptions[m.id] === undefined) {
        loraOptions[m.id] = false
      }
    }
    if (modelsRes.min_train_samples) {
      minTrainSamples.value = modelsRes.min_train_samples
    }
    assignments.value = assignRes.assignments || []
    if (!selectedAssignmentId.value && assignments.value.length > 0) {
      selectedAssignmentId.value = assignments.value[0].id
      await fetchAssignmentSamples()
    }
    setupPollingIfNeeded()
  } catch (err) {
    if (!silent) toastStore.error(err.message || 'Gagal memuat halaman model custom.')
  } finally {
    if (!silent) loading.value = false
  }
}

function setupPollingIfNeeded() {
  const hasRunning = models.value.some(
    (m) => m.latest_job && (m.latest_job.status === 'queued' || m.latest_job.status === 'processing')
  )
  if (hasRunning && !pollInterval) {
    pollInterval = setInterval(() => {
      fetchAllData(true)
    }, 1500)
  } else if (!hasRunning && pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
}

async function fetchAssignmentSamples() {
  if (!selectedAssignmentId.value) {
    assignmentSamples.value = []
    return
  }
  try {
    const res = await apiFetch(`/api/assignments/${selectedAssignmentId.value}/training-samples`)
    assignmentSamples.value = res.samples || []
  } catch {
    assignmentSamples.value = []
  }
}

async function handleTrainingExcelSelected(file) {
  if (!selectedAssignmentId.value) {
    toastStore.error('Pilih tugas terlebih dahulu.')
    return
  }
  selectedExcelFile.value = file
  previewLoading.value = true
  excelPreview.value = null
  try {
    const fd = new FormData()
    fd.append('file', file)
    const res = await apiFetch(
      `/api/assignments/${selectedAssignmentId.value}/training-samples/excel?dry_run=true`,
      {
        method: 'POST',
        body: fd
      }
    )
    excelPreview.value = res
  } catch (err) {
    toastStore.error(err.message || 'Gagal memeriksa file Excel.')
    selectedExcelFile.value = null
  } finally {
    previewLoading.value = false
  }
}

function clearExcelPreview() {
  selectedExcelFile.value = null
  excelPreview.value = null
}

async function confirmUploadTrainingExcel() {
  if (!selectedExcelFile.value || !selectedAssignmentId.value) return
  uploadingExcel.value = true
  try {
    const fd = new FormData()
    fd.append('file', selectedExcelFile.value)
    const res = await apiFetch(
      `/api/assignments/${selectedAssignmentId.value}/training-samples/excel`,
      {
        method: 'POST',
        body: fd
      }
    )
    toastStore.success(res.message || 'Data latih berhasil ditambahkan.')
    clearExcelPreview()
    await Promise.all([fetchAssignmentSamples(), fetchAllData(true)])
  } catch (err) {
    toastStore.error(err.message || 'Gagal menyimpan data latih Excel.')
  } finally {
    uploadingExcel.value = false
  }
}

async function handleDeleteSample(sampleId) {
  try {
    await apiFetch(`/api/training-samples/${sampleId}`, { method: 'DELETE' })
    toastStore.success('Sampel data latih dihapus.')
    await Promise.all([fetchAssignmentSamples(), fetchAllData(true)])
  } catch (err) {
    toastStore.error(err.message)
  }
}

function openCreateModal() {
  editingModelId.value = null
  modalForm.name = ''
  modalForm.description = ''
  modalForm.assignment_ids = assignments.value.map((a) => a.id)
  showModal.value = true
}

function openEditModal(model) {
  editingModelId.value = model.id
  modalForm.name = model.name
  modalForm.description = model.description || ''
  modalForm.assignment_ids = (model.linked_assignments || []).map((la) => la.id)
  showModal.value = true
}

function closeModal() {
  showModal.value = false
}

async function handleSaveModel() {
  savingModel.value = true
  try {
    if (editingModelId.value) {
      await apiFetch(`/api/models/${editingModelId.value}`, {
        method: 'PUT',
        body: JSON.stringify(modalForm)
      })
      toastStore.success('Model custom berhasil diperbarui.')
    } else {
      await apiFetch('/api/models', {
        method: 'POST',
        body: JSON.stringify(modalForm)
      })
      toastStore.success('Model custom baru berhasil dibuat.')
    }
    closeModal()
    await fetchAllData(true)
  } catch (err) {
    toastStore.error(err.message || 'Gagal menyimpan model custom.')
  } finally {
    savingModel.value = false
  }
}

async function handleDeleteModel(model) {
  try {
    await apiFetch(`/api/models/${model.id}`, { method: 'DELETE' })
    toastStore.success(`Model "${model.name}" berhasil dihapus.`)
    await fetchAllData(true)
  } catch (err) {
    toastStore.error(err.message)
  }
}

async function handleStartTraining(model) {
  try {
    const useLora = !!loraOptions[model.id]
    const res = await apiFetch(`/api/models/${model.id}/train`, {
      method: 'POST',
      body: JSON.stringify({ use_lora: useLora })
    })
    toastStore.success(res.message || 'Pelatihan model dimulai.')
    await fetchAllData(true)
  } catch (err) {
    toastStore.error(err.message || 'Gagal memulai pelatihan model.')
  }
}

async function handleActivateVersion(modelId, versionId) {
  try {
    const res = await apiFetch(`/api/models/${modelId}/versions/${versionId}/activate`, {
      method: 'POST'
    })
    toastStore.success(res.message || 'Versi model diaktifkan.')
    await fetchAllData(true)
  } catch (err) {
    toastStore.error(err.message)
  }
}

async function handleDeactivateModel(modelId) {
  try {
    const res = await apiFetch(`/api/models/${modelId}/deactivate`, {
      method: 'POST'
    })
    toastStore.success(res.message || 'Model custom dinonaktifkan.')
    await fetchAllData(true)
  } catch (err) {
    toastStore.error(err.message)
  }
}

onMounted(() => {
  fetchAllData()
})

onBeforeUnmount(() => {
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
})
</script>
