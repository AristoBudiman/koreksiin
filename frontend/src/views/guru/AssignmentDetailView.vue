<template>
  <div class="space-y-6">
    <!-- Top Navigation -->
    <div class="flex flex-wrap items-center justify-between gap-3">
      <RouterLink
        to="/guru"
        class="inline-flex items-center gap-1.5 text-xs font-semibold text-ink-muted hover:text-brand-700 dark:hover:text-brand-400 transition-colors"
      >
        <ArrowLeft class="w-4 h-4" />
        <span>Kembali ke Daftar Tugas</span>
      </RouterLink>

      <div v-if="assignment" class="flex flex-wrap items-center gap-2">
        <a
          :href="`/api/templates/excel?type=submissions`"
          class="btn-secondary text-xs px-3 py-2"
          download
        >
          <FileSpreadsheet class="w-4 h-4 text-brand-700 dark:text-brand-400" />
          <span>Unduh Template Excel</span>
        </a>

        <button
          v-if="submissions.length > 0"
          type="button"
          class="btn-secondary text-xs px-3.5 py-2"
          :disabled="exporting"
          @click="handleExportExcel"
        >
          <Loader2 v-if="exporting" class="w-4 h-4 animate-spin" />
          <Download v-else class="w-4 h-4" />
          <span>Ekspor Hasil ke Excel</span>
        </button>

        <RouterLink
          :to="`/guru/tugas/${assignment.id}/edit`"
          class="btn-secondary text-xs px-3 py-2"
        >
          <Edit3 class="w-4 h-4" />
          <span>Edit Rubrik</span>
        </RouterLink>

        <button
          type="button"
          class="btn-primary text-xs px-4 py-2"
          :disabled="enqueuing || isJobRunning || submissions.length === 0"
          @click="handleStartScoring"
        >
          <Loader2 v-if="enqueuing || isJobRunning" class="w-4 h-4 animate-spin" />
          <Play v-else class="w-4 h-4" />
          <span>
            {{
              isJobRunning
                ? 'Sedang Menilai...'
                : assignment.scored_count > 0
                  ? 'Nilai Ulang Jawaban'
                  : 'Mulai Penilaian AI'
            }}
          </span>
        </button>
      </div>
    </div>

    <!-- Skeleton Loading -->
    <div v-if="loading" class="space-y-4">
      <div class="card-paper p-6 animate-pulse space-y-3">
        <div class="h-6 bg-ink-line dark:bg-ink-darkLine rounded w-1/3"></div>
        <div class="h-4 bg-ink-line dark:bg-ink-darkLine rounded w-2/3"></div>
        <div class="h-16 bg-ink-line dark:bg-ink-darkLine rounded w-full"></div>
      </div>
    </div>

    <template v-else-if="assignment">
      <!-- Header Tugas & Ringkasan Nilai -->
      <div class="card-paper p-5 sm:p-6 space-y-5">
        <div class="flex flex-col lg:flex-row lg:items-start justify-between gap-6">
          <div class="space-y-2 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <span class="px-2.5 py-0.5 rounded bg-brand-50 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 text-xs font-bold">
                {{ assignment.subject }}
              </span>
              <span class="px-2.5 py-0.5 rounded bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-xs font-semibold text-ink-muted">
                Model: {{ assignment.model_type === 'custom' ? assignment.custom_model_name || 'Model Custom' : 'Model Bawaan (IndoBERT v4)' }}
              </span>
            </div>

            <HandDrawnUnderline>
              <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
                {{ assignment.title }}
              </h1>
            </HandDrawnUnderline>

            <div class="p-3.5 rounded-lg bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-xs sm:text-sm leading-relaxed">
              <span class="font-bold text-ink-muted block text-[11px] uppercase tracking-wider mb-1">Soal Esai:</span>
              {{ assignment.question_text }}
            </div>
          </div>

          <!-- Statistik Ringkas Tugas -->
          <div class="grid grid-cols-3 gap-3 sm:w-auto shrink-0">
            <div class="p-3.5 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-center">
              <span class="text-[11px] font-bold uppercase text-ink-muted block">Siswa</span>
              <span class="text-xl sm:text-2xl font-extrabold num-tabular mt-1 block">
                {{ submissions.length }}
              </span>
              <span class="text-[11px] text-ink-muted num-tabular">{{ assignment.scored_count }} dinilai</span>
            </div>

            <div class="p-3.5 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-center">
              <span class="text-[11px] font-bold uppercase text-ink-muted block">Rata-rata</span>
              <ScoreCircleHighlight v-if="assignment.average_score !== null">
                <CountUpNumber
                  :value="assignment.average_score"
                  :decimals="1"
                  class="text-xl sm:text-2xl font-extrabold text-brand-700 dark:text-brand-400"
                />
              </ScoreCircleHighlight>
              <span v-else class="text-xl sm:text-2xl font-extrabold text-ink-muted mt-1 block">-</span>
              <span class="text-[11px] text-ink-muted num-tabular block">Maks {{ assignment.max_score }}</span>
            </div>

            <div class="p-3.5 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-center">
              <span class="text-[11px] font-bold uppercase text-ink-muted block">Kemiripan</span>
              <span
                class="text-xl sm:text-2xl font-extrabold num-tabular mt-1 block"
                :class="similarityFlags.length > 0 ? 'text-brand-700 dark:text-brand-400' : 'text-criteria-terpenuhi'"
              >
                {{ similarityFlags.length }}
              </span>
              <span class="text-[11px] text-ink-muted">pasangan</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Peringatan Kuota Habis -->
      <div
        v-if="quotaExceededBanner"
        class="card-paper p-4 border-l-4 border-l-brand-700 bg-brand-50/90 dark:bg-brand-950/30 flex flex-col sm:flex-row sm:items-center justify-between gap-3"
      >
        <div class="flex items-start gap-2.5 text-xs">
          <CircleAlert class="w-5 h-5 text-brand-700 dark:text-brand-400 shrink-0 mt-0.5" />
          <div>
            <p class="font-bold text-brand-800 dark:text-brand-200">Kuota Penilaian Tidak Mencukupi</p>
            <p class="text-brand-700 dark:text-brand-300 mt-0.5">{{ quotaExceededBanner }}</p>
          </div>
        </div>
        <RouterLink to="/guru/langganan" class="btn-primary text-xs px-4 py-2 shrink-0">
          <span>Buka Halaman Langganan</span>
          <ArrowRight class="w-4 h-4" />
        </RouterLink>
      </div>

      <!-- Panel Progres Job Penilaian (antri -> memecah teks -> menilai -> cek kemiripan -> selesai) -->
      <div
        v-if="activeJob && (isJobRunning || activeJob.status === 'failed')"
        class="card-paper p-5 space-y-4 border-2 border-brand-500/40"
      >
        <div class="flex items-center justify-between gap-4">
          <div class="flex items-center gap-2.5">
            <Loader2 v-if="isJobRunning" class="w-5 h-5 text-brand-500 animate-spin" />
            <CircleAlert v-else class="w-5 h-5 text-brand-700" />
            <div>
              <h2 class="font-extrabold text-sm">
                {{ isJobRunning ? 'Memproses Penilaian Esai Otomatis...' : 'Penilaian Gagal' }}
              </h2>
              <p class="text-xs text-ink-muted">
                Tahap saat ini: <strong class="uppercase text-brand-700 dark:text-brand-400">{{ activeJob.stage }}</strong>
              </p>
            </div>
          </div>
          <span class="text-lg font-extrabold num-tabular text-brand-700 dark:text-brand-400">
            {{ activeJob.progress || 0 }}%
          </span>
        </div>

        <!-- Progress Bar Halus -->
        <div class="w-full h-3 rounded-full bg-ink-line dark:bg-ink-darkLine overflow-hidden">
          <div
            class="h-full bg-brand-500 transition-all duration-300 ease-out rounded-full"
            :style="{ width: `${Math.max(activeJob.progress || 5, 5)}%` }"
          ></div>
        </div>

        <!-- Indikator 5 Tahap -->
        <div class="grid grid-cols-5 gap-2 text-[11px] font-bold text-center">
          <div
            v-for="st in jobStages"
            :key="st.key"
            class="py-1.5 px-2 rounded-lg border transition-colors"
            :class="
              isStageReached(st.key)
                ? 'bg-brand-50 dark:bg-brand-900/40 border-brand-300 dark:border-brand-700 text-brand-700 dark:text-brand-300'
                : 'border-ink-line dark:border-ink-darkLine text-ink-muted'
            "
          >
            {{ st.label }}
          </div>
        </div>

        <p v-if="activeJob.error_message" class="text-xs font-semibold text-brand-700 dark:text-brand-400">
          Detail: {{ activeJob.error_message }}
        </p>
      </div>

      <!-- Navigasi Tab -->
      <div class="flex flex-wrap items-center gap-2 border-b border-ink-line dark:border-ink-darkLine pb-3">
        <button
          type="button"
          class="px-4 py-2 rounded-lg text-xs font-bold flex items-center gap-2 transition-colors"
          :class="
            activeTab === 'hasil'
              ? 'bg-brand-500 text-white'
              : 'bg-ink-card dark:bg-ink-darkCard text-ink-muted hover:text-ink dark:hover:text-ink-darkText border border-ink-line dark:border-ink-darkLine'
          "
          @click="activeTab = 'hasil'"
        >
          <CheckSquare class="w-4 h-4" />
          <span>Hasil Penilaian Siswa ({{ submissions.length }})</span>
        </button>

        <button
          type="button"
          class="px-4 py-2 rounded-lg text-xs font-bold flex items-center gap-2 transition-colors"
          :class="
            activeTab === 'input'
              ? 'bg-brand-500 text-white'
              : 'bg-ink-card dark:bg-ink-darkCard text-ink-muted hover:text-ink dark:hover:text-ink-darkText border border-ink-line dark:border-ink-darkLine'
          "
          @click="activeTab = 'input'"
        >
          <Upload class="w-4 h-4" />
          <span>Tambah / Upload Jawaban Siswa</span>
        </button>

        <button
          type="button"
          class="px-4 py-2 rounded-lg text-xs font-bold flex items-center gap-2 transition-colors"
          :class="
            activeTab === 'kemiripan'
              ? 'bg-brand-500 text-white'
              : 'bg-ink-card dark:bg-ink-darkCard text-ink-muted hover:text-ink dark:hover:text-ink-darkText border border-ink-line dark:border-ink-darkLine'
          "
          @click="activeTab = 'kemiripan'"
        >
          <CopyCheck class="w-4 h-4" />
          <span>Cek Kemiripan ({{ similarityFlags.length }})</span>
        </button>

        <button
          type="button"
          class="px-4 py-2 rounded-lg text-xs font-bold flex items-center gap-2 transition-colors"
          :class="
            activeTab === 'rubrik'
              ? 'bg-brand-500 text-white'
              : 'bg-ink-card dark:bg-ink-darkCard text-ink-muted hover:text-ink dark:hover:text-ink-darkText border border-ink-line dark:border-ink-darkLine'
          "
          @click="activeTab = 'rubrik'"
        >
          <ListChecks class="w-4 h-4" />
          <span>Rubrik Kriteria ({{ criteria.length }})</span>
        </button>
      </div>

      <!-- TAB 1: INPUT / UPLOAD JAWABAN SISWA -->
      <div v-if="activeTab === 'input'" class="grid lg:grid-cols-2 gap-6">
        <!-- Upload Excel -->
        <div class="card-paper p-5 sm:p-6 space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <h2 class="font-extrabold text-base flex items-center gap-2">
                <FileSpreadsheet class="w-4 h-4 text-brand-700 dark:text-brand-400" />
                <span>Opsi A: Unggah File Excel (.xlsx)</span>
              </h2>
              <p class="text-xs text-ink-muted mt-0.5">
                Unggah banyak jawaban siswa sekaligus dari template Excel.
              </p>
            </div>
            <a
              :href="`/api/templates/excel?type=submissions`"
              class="text-xs font-bold text-brand-700 dark:text-brand-400 hover:underline inline-flex items-center gap-1"
              download
            >
              <Download class="w-3.5 h-3.5" />
              <span>Template</span>
            </a>
          </div>

          <ExcelDropzone
            :assignment-id="assignment.id"
            @uploaded="onSubmissionsAdded"
          />
        </div>

        <!-- Input Manual Satu per Satu -->
        <div class="card-paper p-5 sm:p-6 space-y-4">
          <div>
            <h2 class="font-extrabold text-base flex items-center gap-2">
              <FileText class="w-4 h-4 text-brand-700 dark:text-brand-400" />
              <span>Opsi B: Tempel Jawaban Siswa Satu per Satu</span>
            </h2>
            <p class="text-xs text-ink-muted mt-0.5">
              Panjang teks jawaban siswa tidak dibatasi.
            </p>
          </div>

          <form class="space-y-3.5" @submit.prevent="handleAddManual">
            <div>
              <label class="block text-xs font-bold mb-1">Nama Siswa</label>
              <input
                v-model.trim="manualForm.student_name"
                type="text"
                required
                placeholder="Contoh: Nadia Putri Rahmawati"
                class="input-field"
              />
            </div>

            <div>
              <label class="block text-xs font-bold mb-1">Teks Jawaban Esai</label>
              <textarea
                v-model.trim="manualForm.answer_text"
                rows="5"
                required
                placeholder="Tempelkan jawaban lengkap siswa di sini..."
                class="input-field"
              ></textarea>
            </div>

            <button type="submit" class="btn-primary w-full" :disabled="addingManual">
              <Loader2 v-if="addingManual" class="w-4 h-4 animate-spin" />
              <Plus v-else class="w-4 h-4" />
              <span>{{ addingManual ? 'Menyimpan...' : 'Tambahkan Jawaban Siswa' }}</span>
            </button>
          </form>
        </div>
      </div>

      <!-- TAB 2: HASIL PENILAIAN SISWA -->
      <div v-else-if="activeTab === 'hasil'" class="space-y-5">
        <div v-if="submissions.length === 0" class="card-paper p-10 text-center space-y-4">
          <div class="w-12 h-12 rounded-xl bg-brand-50 dark:bg-brand-900/30 text-brand-700 dark:text-brand-400 flex items-center justify-center mx-auto">
            <Upload class="w-6 h-6" />
          </div>
          <div class="max-w-md mx-auto space-y-1.5">
            <h3 class="font-extrabold text-base">Belum Ada Jawaban Siswa</h3>
            <p class="text-xs text-ink-muted">
              Unggah file Excel jawaban siswa atau tempel jawaban secara manual terlebih dahulu untuk mulai menilai.
            </p>
          </div>
          <button type="button" class="btn-primary text-xs" @click="activeTab = 'input'">
            <Upload class="w-4 h-4" />
            <span>Tambah Jawaban Siswa Sekarang</span>
          </button>
        </div>

        <div v-else class="grid lg:grid-cols-12 gap-6 items-start">
          <!-- Kolom Kiri: Daftar Siswa -->
          <div class="lg:col-span-5 card-paper overflow-hidden">
            <div class="p-3.5 border-b border-ink-line dark:border-ink-darkLine flex items-center justify-between gap-2">
              <div class="relative flex-1">
                <Search class="w-4 h-4 text-ink-muted absolute left-3 top-2.5" />
                <input
                  v-model.trim="searchQuery"
                  type="text"
                  placeholder="Cari nama siswa..."
                  class="input-field pl-9 py-1.5 text-xs"
                />
              </div>
              <span class="text-xs font-bold text-ink-muted num-tabular shrink-0">
                {{ filteredSubmissions.length }} siswa
              </span>
            </div>

            <div class="divide-y divide-ink-line dark:divide-ink-darkLine max-h-[600px] overflow-y-auto">
              <button
                v-for="(sub, idx) in filteredSubmissions"
                :key="sub.id"
                type="button"
                class="w-full text-left p-3.5 transition-colors flex items-center justify-between gap-3"
                :class="
                  selectedSubmission?.id === sub.id
                    ? 'bg-brand-50 dark:bg-brand-900/30 border-l-4 border-l-brand-500'
                    : 'hover:bg-ink-bg dark:hover:bg-ink-darkBg'
                "
                @click="selectSubmission(sub)"
              >
                <div class="min-w-0 flex-1">
                  <div class="flex items-center gap-2">
                    <span class="text-xs font-bold text-ink-muted num-tabular">#{{ idx + 1 }}</span>
                    <p class="text-sm font-bold truncate">{{ sub.student_name }}</p>
                    <ShieldAlert
                      v-if="hasSimilarityFlag(sub.id)"
                      class="w-4 h-4 text-brand-700 dark:text-brand-400 shrink-0"
                      title="Terindikasi mirip dengan jawaban siswa lain"
                    />
                  </div>
                  <p class="text-xs text-ink-muted truncate mt-0.5">{{ sub.answer_text }}</p>
                </div>

                <div class="text-right shrink-0">
                  <template v-if="sub.status === 'scored' && sub.final_score !== null">
                    <span class="text-base font-extrabold num-tabular text-brand-700 dark:text-brand-400">
                      {{ Number(sub.final_score).toFixed(1) }}
                    </span>
                    <span
                      v-if="sub.revised_score !== null"
                      class="block text-[10px] font-bold text-ink-muted"
                    >
                      Direvisi
                    </span>
                  </template>
                  <span
                    v-else
                    class="px-2 py-0.5 rounded text-[11px] font-semibold bg-ink-bg dark:bg-ink-darkBg text-ink-muted border border-ink-line dark:border-ink-darkLine"
                  >
                    Belum dinilai
                  </span>
                </div>
              </button>
            </div>
          </div>

          <!-- Kolom Kanan: Rincian Jawaban & Kriteria Siswa Terpilih -->
          <div class="lg:col-span-7 space-y-5">
            <div v-if="selectedSubmission" class="card-paper p-5 sm:p-6 space-y-6">
              <!-- Header Siswa & Nilai Total -->
              <div class="flex flex-col sm:flex-row sm:items-start justify-between gap-4 border-b border-ink-line dark:border-ink-darkLine pb-4">
                <div class="space-y-1">
                  <div class="flex items-center gap-2">
                    <h2 class="text-xl font-extrabold">{{ selectedSubmission.student_name }}</h2>
                    <span
                      v-if="selectedSubmission.revised_score !== null"
                      class="px-2 py-0.5 rounded bg-brand-100 dark:bg-brand-900/50 text-brand-800 dark:text-brand-200 text-[11px] font-bold"
                    >
                      Telah Direvisi Guru
                    </span>
                  </div>
                  <p class="text-xs text-ink-muted num-tabular">
                    Nilai AI:
                    <strong>{{ selectedSubmission.predicted_score !== null ? Number(selectedSubmission.predicted_score).toFixed(1) : '-' }}</strong>
                    <template v-if="selectedSubmission.revised_score !== null">
                      • Nilai Revisi Guru: <strong>{{ Number(selectedSubmission.revised_score).toFixed(1) }}</strong>
                    </template>
                  </p>
                </div>

                <div class="flex items-center gap-4">
                  <div class="text-right">
                    <span class="text-[11px] font-bold uppercase tracking-wider text-ink-muted block">Nilai Akhir</span>
                    <ScoreCircleHighlight v-if="selectedSubmission.final_score !== null">
                      <CountUpNumber
                        :value="selectedSubmission.final_score"
                        :decimals="1"
                        class="text-2xl sm:text-3xl font-extrabold text-brand-700 dark:text-brand-400"
                      />
                    </ScoreCircleHighlight>
                    <span v-else class="text-2xl font-extrabold text-ink-muted">-</span>
                  </div>

                  <button
                    type="button"
                    class="text-xs text-ink-muted hover:text-brand-700 p-1.5 rounded-lg border border-ink-line dark:border-ink-darkLine"
                    title="Hapus jawaban siswa ini"
                    @click="handleDeleteSubmission(selectedSubmission.id)"
                  >
                    <Trash2 class="w-4 h-4" />
                  </button>
                </div>
              </div>

              <!-- Peringatan Kemiripan pada Siswa Ini -->
              <div
                v-if="studentSimilarities.length > 0"
                class="p-3.5 rounded-xl bg-brand-50 dark:bg-brand-950/40 border border-brand-200 dark:border-brand-800 space-y-1.5 text-xs"
              >
                <div class="flex items-center gap-2 font-bold text-brand-700 dark:text-brand-300">
                  <ShieldAlert class="w-4 h-4 shrink-0" />
                  <span>Indikasi Kemiripan Jawaban Terdeteksi:</span>
                </div>
                <ul class="space-y-1 pl-6 list-disc text-brand-800 dark:text-brand-200">
                  <li v-for="sf in studentSimilarities" :key="sf.id" class="num-tabular">
                    Mirip dengan
                    <strong>{{ sf.submission_a_id === selectedSubmission.id ? sf.student_b_name : sf.student_a_name }}</strong>
                    — Skor Semantik: <strong>{{ (sf.semantic_score * 100).toFixed(1) }}%</strong>,
                    Skor Leksikal: <strong>{{ (sf.lexical_score * 100).toFixed(1) }}%</strong>
                  </li>
                </ul>
              </div>

              <!-- Teks Jawaban Siswa -->
              <div class="space-y-1.5">
                <h3 class="text-xs font-extrabold uppercase tracking-wider text-ink-muted">Jawaban Lengkap Siswa</h3>
                <div class="p-4 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-sm leading-relaxed whitespace-pre-wrap">
                  {{ selectedSubmission.answer_text }}
                </div>
              </div>

              <!-- Form Revisi Nilai Guru (Menyimpan ke Data Latih) -->
              <div class="p-4 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine flex flex-col sm:flex-row sm:items-end justify-between gap-3">
                <div class="space-y-1 flex-1">
                  <label class="block text-xs font-extrabold">
                    Revisi Nilai Total Guru (0 – {{ assignment.max_score }})
                  </label>
                  <p class="text-[11px] text-ink-muted">
                    Nilai revisi akan menggantikan nilai akhir dan otomatis disimpan sebagai sampel data latih untuk model custom.
                  </p>
                </div>
                <form class="flex items-center gap-2 shrink-0" @submit.prevent="handleReviseScore">
                  <input
                    v-model.number="reviseInput"
                    type="number"
                    min="0"
                    :max="assignment.max_score"
                    step="0.5"
                    required
                    class="input-field w-24 num-tabular font-bold text-center py-2"
                  />
                  <button type="submit" class="btn-primary py-2 px-4 text-xs" :disabled="revising">
                    <Loader2 v-if="revising" class="w-3.5 h-3.5 animate-spin" />
                    <Check v-else class="w-3.5 h-3.5" />
                    <span>Simpan Revisi</span>
                  </button>
                </form>
              </div>

              <!-- Umpan Balik (Feedback Template per Kalimat dengan Ikon Status Lucide) -->
              <div v-if="selectedSubmission.criteria_scores?.length" class="space-y-2.5">
                <h3 class="text-xs font-extrabold uppercase tracking-wider text-ink-muted">
                  Ringkasan Umpan Balik Otomatis
                </h3>
                <div class="p-4 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine space-y-2">
                  <div
                    v-for="cs in selectedSubmission.criteria_scores"
                    :key="`fb-${cs.criterion_id}`"
                    class="flex items-start gap-2.5 text-xs sm:text-sm"
                  >
                    <CircleCheck
                      v-if="cs.status === 'terpenuhi'"
                      class="w-4 h-4 text-criteria-terpenuhi shrink-0 mt-0.5"
                    />
                    <CircleAlert
                      v-else-if="cs.status === 'sebagian'"
                      class="w-4 h-4 text-criteria-sebagian shrink-0 mt-0.5"
                    />
                    <CircleX
                      v-else
                      class="w-4 h-4 text-criteria-tidakAda shrink-0 mt-0.5"
                    />
                    <span class="leading-relaxed">{{ cs.feedback_sentence }}</span>
                  </div>
                </div>
              </div>

              <!-- Rincian Cakupan per Kriteria Rubrik (Staggered Cards) -->
              <div v-if="selectedSubmission.criteria_scores?.length" class="space-y-3">
                <h3 class="text-xs font-extrabold uppercase tracking-wider text-ink-muted">
                  Rincian Analisis per Kriteria Rubrik
                </h3>

                <div class="space-y-3">
                  <div
                    v-for="(cs, cIdx) in selectedSubmission.criteria_scores"
                    :key="cs.criterion_id"
                    class="p-4 rounded-xl border border-ink-line dark:border-ink-darkLine bg-ink-card dark:bg-ink-darkCard space-y-2.5 transition-all duration-300"
                    :style="{ transitionDelay: `${cIdx * 60}ms` }"
                  >
                    <div class="flex flex-wrap items-center justify-between gap-2">
                      <div class="flex items-center gap-2">
                        <CircleCheck
                          v-if="cs.status === 'terpenuhi'"
                          class="w-4 h-4 text-criteria-terpenuhi shrink-0"
                        />
                        <CircleAlert
                          v-else-if="cs.status === 'sebagian'"
                          class="w-4 h-4 text-criteria-sebagian shrink-0"
                        />
                        <CircleX
                          v-else
                          class="w-4 h-4 text-criteria-tidakAda shrink-0"
                        />
                        <span class="font-bold text-sm">{{ cs.criterion_name }}</span>
                        <span class="text-xs text-ink-muted num-tabular">(Bobot {{ cs.criterion_weight }})</span>
                      </div>

                      <div class="flex items-center gap-2">
                        <span class="text-xs font-bold num-tabular text-ink-muted">
                          Cakupan {{ (cs.coverage_score * 100).toFixed(0) }}%
                        </span>
                        <span
                          class="px-2.5 py-0.5 rounded text-xs font-bold"
                          :class="statusBadgeClass(cs.status)"
                        >
                          {{ statusLabel(cs.status) }}
                        </span>
                      </div>
                    </div>

                    <div
                      v-if="cs.matched_snippet"
                      class="p-2.5 rounded-lg bg-ink-bg dark:bg-ink-darkBg text-xs text-ink-muted border-l-2 border-l-brand-400"
                    >
                      <span class="font-semibold text-ink dark:text-ink-darkText block mb-0.5">
                        Potongan kalimat siswa yang paling relevan:
                      </span>
                      "{{ cs.matched_snippet }}"
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- TAB 3: CEK KEMIRIPAN ANTAR SISWA -->
      <div v-else-if="activeTab === 'kemiripan'" class="space-y-4">
        <div v-if="similarityFlags.length === 0" class="card-paper p-10 text-center space-y-3">
          <div class="w-12 h-12 rounded-xl bg-green-50 dark:bg-green-950/40 text-criteria-terpenuhi flex items-center justify-center mx-auto">
            <ShieldCheck class="w-6 h-6" />
          </div>
          <h3 class="font-extrabold text-base">Tidak Ditemukan Pasangan Jawaban yang Terindikasi Menyontek</h3>
          <p class="text-xs text-ink-muted max-w-md mx-auto">
            Seluruh pasangan jawaban siswa berada di bawah ambang gabungan kemiripan semantik (S_SEM) dan kemiripan leksikal (S_LEX).
          </p>
        </div>

        <div v-else class="space-y-4">
          <div
            v-for="sf in similarityFlags"
            :key="sf.id"
            class="card-paper p-5 space-y-4 border-l-4 border-l-brand-600"
          >
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div class="flex items-center gap-2">
                <ShieldAlert class="w-5 h-5 text-brand-700 dark:text-brand-400" />
                <h3 class="font-extrabold text-sm sm:text-base">
                  {{ sf.student_a_name }} &amp; {{ sf.student_b_name }}
                </h3>
              </div>

              <div class="flex items-center gap-2 text-xs font-bold num-tabular">
                <span class="px-2.5 py-1 rounded bg-brand-50 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300">
                  Kemiripan Semantik: {{ (sf.semantic_score * 100).toFixed(1) }}%
                </span>
                <span class="px-2.5 py-1 rounded bg-brand-50 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300">
                  Kemiripan Leksikal: {{ (sf.lexical_score * 100).toFixed(1) }}%
                </span>
              </div>
            </div>

            <div class="grid sm:grid-cols-2 gap-4 text-xs">
              <div class="p-3.5 rounded-lg bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine space-y-1">
                <span class="font-bold text-ink dark:text-ink-darkText block">{{ sf.student_a_name }}</span>
                <p class="text-ink-muted leading-relaxed">{{ sf.student_a_text }}</p>
              </div>
              <div class="p-3.5 rounded-lg bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine space-y-1">
                <span class="font-bold text-ink dark:text-ink-darkText block">{{ sf.student_b_name }}</span>
                <p class="text-ink-muted leading-relaxed">{{ sf.student_b_text }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- TAB 4: RUBRIK KRITERIA -->
      <div v-else-if="activeTab === 'rubrik'" class="space-y-4">
        <div
          v-for="(crit, idx) in criteria"
          :key="crit.id"
          class="card-paper p-5 space-y-2"
        >
          <div class="flex items-center justify-between">
            <h3 class="font-extrabold text-base">
              {{ idx + 1 }}. {{ crit.name }}
            </h3>
            <span class="px-2.5 py-1 rounded bg-brand-50 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 text-xs font-bold num-tabular">
              Bobot: {{ crit.weight }}
            </span>
          </div>
          <p class="text-xs font-bold text-ink-muted uppercase tracking-wider pt-1">Poin Kunci Jawaban:</p>
          <pre class="p-3.5 rounded-lg bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-xs font-sans whitespace-pre-wrap leading-relaxed">{{ crit.answer_key }}</pre>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import {
  ArrowLeft,
  ArrowRight,
  FileSpreadsheet,
  Download,
  Edit3,
  Play,
  Loader2,
  CircleCheck,
  CircleAlert,
  CircleX,
  CheckSquare,
  Upload,
  CopyCheck,
  ListChecks,
  FileText,
  Plus,
  Search,
  ShieldAlert,
  ShieldCheck,
  Trash2,
  Check
} from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import ScoreCircleHighlight from '@/components/ScoreCircleHighlight.vue'
import CountUpNumber from '@/components/CountUpNumber.vue'
import ExcelDropzone from '@/components/ExcelDropzone.vue'
import { apiFetch, getToken } from '@/utils/api'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'

const route = useRoute()
const authStore = useAuthStore()
const toastStore = useToastStore()

const loading = ref(true)
const assignment = ref(null)
const criteria = ref([])
const submissions = ref([])
const similarityFlags = ref([])
const activeJob = ref(null)

const activeTab = ref('hasil')
const searchQuery = ref('')
const selectedSubmission = ref(null)
const reviseInput = ref(0)
const revising = ref(false)

const addingManual = ref(false)
const manualForm = reactive({
  student_name: '',
  answer_text: ''
})

const enqueuing = ref(false)
const exporting = ref(false)
const quotaExceededBanner = ref('')

let pollTimer = null

const jobStages = [
  { key: 'antri', label: '1. Antri' },
  { key: 'memecah teks', label: '2. Memecah Teks' },
  { key: 'menilai', label: '3. Menilai' },
  { key: 'cek kemiripan', label: '4. Cek Kemiripan' },
  { key: 'selesai', label: '5. Selesai' }
]

const isJobRunning = computed(() =>
  activeJob.value && (activeJob.value.status === 'queued' || activeJob.value.status === 'processing')
)

const filteredSubmissions = computed(() => {
  const q = searchQuery.value.toLowerCase()
  if (!q) return submissions.value
  return submissions.value.filter((s) => s.student_name.toLowerCase().includes(q))
})

const studentSimilarities = computed(() => {
  if (!selectedSubmission.value) return []
  const sid = selectedSubmission.value.id
  return similarityFlags.value.filter(
    (sf) => sf.submission_a_id === sid || sf.submission_b_id === sid
  )
})

function hasSimilarityFlag(subId) {
  return similarityFlags.value.some(
    (sf) => sf.submission_a_id === subId || sf.submission_b_id === subId
  )
}

function isStageReached(stageKey) {
  if (!activeJob.value) return false
  const order = ['antri', 'memecah teks', 'menilai', 'cek kemiripan', 'selesai']
  const currentIdx = order.indexOf(activeJob.value.stage)
  const targetIdx = order.indexOf(stageKey)
  return currentIdx >= targetIdx
}

function statusLabel(st) {
  if (st === 'terpenuhi') return 'Terpenuhi'
  if (st === 'sebagian') return 'Sebagian'
  return 'Tidak ada'
}

function statusBadgeClass(st) {
  if (st === 'terpenuhi') {
    return 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
  }
  if (st === 'sebagian') {
    return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
  }
  return 'bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
}

function selectSubmission(sub) {
  selectedSubmission.value = sub
  reviseInput.value = sub.final_score !== null ? Number(sub.final_score) : 0
}

async function fetchDetail(keepTab = false) {
  try {
    const res = await apiFetch(`/api/assignments/${route.params.id}`)
    assignment.value = res.assignment
    criteria.value = res.criteria || []
    submissions.value = res.submissions || []
    similarityFlags.value = res.similarity_flags || []

    if (res.latest_job) {
      activeJob.value = res.latest_job
      if (isJobRunning.value) {
        startJobPolling(res.latest_job.id)
      }
    }

    if (!keepTab && submissions.value.length === 0) {
      activeTab.value = 'input'
    }

    if (submissions.value.length > 0) {
      const prevId = selectedSubmission.value?.id
      const found = submissions.value.find((s) => s.id === prevId) || submissions.value[0]
      selectSubmission(found)
    } else {
      selectedSubmission.value = null
    }
  } catch (err) {
    toastStore.error(err.message || 'Gagal memuat detail tugas.')
  } finally {
    loading.value = false
  }
}

function onSubmissionsAdded() {
  activeTab.value = 'hasil'
  fetchDetail(true)
}

async function handleAddManual() {
  addingManual.value = true
  try {
    const res = await apiFetch(`/api/assignments/${assignment.value.id}/submissions/manual`, {
      method: 'POST',
      body: JSON.stringify(manualForm)
    })
    toastStore.success(res.message)
    manualForm.student_name = ''
    manualForm.answer_text = ''
    activeTab.value = 'hasil'
    await fetchDetail(true)
  } catch (err) {
    toastStore.error(err.message)
  } finally {
    addingManual.value = false
  }
}

async function handleDeleteSubmission(subId) {
  try {
    await apiFetch(`/api/submissions/${subId}`, { method: 'DELETE' })
    toastStore.success('Jawaban siswa dihapus.')
    await fetchDetail(true)
  } catch (err) {
    toastStore.error(err.message)
  }
}

async function handleStartScoring() {
  quotaExceededBanner.value = ''
  enqueuing.value = true
  try {
    const res = await apiFetch(`/api/assignments/${assignment.value.id}/score`, {
      method: 'POST'
    })
    toastStore.success(res.message)
    activeJob.value = {
      id: res.job_id,
      status: res.status,
      stage: res.stage,
      progress: 5
    }
    startJobPolling(res.job_id)
  } catch (err) {
    if (err.data?.quota_exceeded) {
      quotaExceededBanner.value = err.message
    }
    toastStore.error(err.message)
  } finally {
    enqueuing.value = false
  }
}

function startJobPolling(jobId) {
  stopJobPolling()
  pollTimer = setInterval(async () => {
    try {
      const res = await apiFetch(`/api/jobs/${jobId}`)
      activeJob.value = res.job
      if (res.job.status === 'completed') {
        stopJobPolling()
        toastStore.success('Penilaian otomatis telah selesai!')
        activeTab.value = 'hasil'
        await fetchDetail(true)
        await authStore.refreshProfile()
      } else if (res.job.status === 'failed') {
        stopJobPolling()
        toastStore.error(res.job.error_message || 'Penilaian gagal.')
      }
    } catch {
      stopJobPolling()
    }
  }, 1500)
}

function stopJobPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

async function handleReviseScore() {
  if (!selectedSubmission.value) return
  revising.value = true
  try {
    const res = await apiFetch(`/api/submissions/${selectedSubmission.value.id}/revise-score`, {
      method: 'PUT',
      body: JSON.stringify({ revised_score: Number(reviseInput.value) })
    })
    toastStore.success(res.message)
    await fetchDetail(true)
  } catch (err) {
    toastStore.error(err.message)
  } finally {
    revising.value = false
  }
}

async function handleExportExcel() {
  exporting.value = true
  try {
    const token = getToken()
    const res = await fetch(`/api/assignments/${assignment.value.id}/export`, {
      headers: { Authorization: `Bearer ${token}` }
    })
    if (!res.ok) {
      throw new Error('Gagal mengunduh file Excel.')
    }
    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `hasil_koreksi_${assignment.value.id}.xlsx`
    document.body.appendChild(a)
    a.click()
    a.remove()
    window.URL.revokeObjectURL(url)
    toastStore.success('File Excel hasil penilaian berhasil diunduh.')
  } catch (err) {
    toastStore.error(err.message || 'Gagal mengekspor Excel.')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  fetchDetail()
})

onUnmounted(() => {
  stopJobPolling()
})
</script>

