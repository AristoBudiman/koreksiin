<template>
  <div class="space-y-8">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
      <div>
        <HandDrawnUnderline>
          <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
            Dashboard Administrator
          </h1>
        </HandDrawnUnderline>
        <p class="text-xs sm:text-sm text-ink-muted mt-2">
          Pantau statistik platform, kelola akun guru, periksa transaksi &amp; log webhook Midtrans, serta atur paket dan ambang AI.
        </p>
      </div>

      <button
        type="button"
        class="btn-secondary text-xs"
        :disabled="loading"
        @click="fetchAllAdminData"
      >
        <RefreshCw class="w-4 h-4" :class="loading ? 'animate-spin' : ''" />
        <span>Muat Ulang Data</span>
      </button>
    </div>

    <!-- Kartu Statistik Utama -->
    <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-5">
      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Guru Terdaftar</span>
          <Users class="w-4 h-4 text-brand-700 dark:text-brand-400" />
        </div>
        <p class="text-2xl font-extrabold num-tabular">{{ overview.total_teachers }}</p>
        <p class="text-xs text-ink-muted num-tabular">
          <strong class="text-criteria-terpenuhi">{{ overview.active_teachers }} aktif</strong>,
          {{ overview.inactive_teachers }} nonaktif
        </p>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Esai Dinilai &amp; Model</span>
          <FileCheck2 class="w-4 h-4 text-criteria-terpenuhi" />
        </div>
        <p class="text-2xl font-extrabold num-tabular">{{ overview.total_scored_essays }}</p>
        <p class="text-xs text-ink-muted num-tabular">
          {{ overview.total_assignments }} tugas — {{ overview.total_custom_models }} model custom
        </p>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Pendapatan Lunas</span>
          <CreditCard class="w-4 h-4 text-brand-700 dark:text-brand-400" />
        </div>
        <ScoreCircleHighlight>
          <span class="text-xl sm:text-2xl font-extrabold text-brand-700 dark:text-brand-400 num-tabular">
            {{ formatRupiah(overview.total_revenue_idr) }}
          </span>
        </ScoreCircleHighlight>
        <p class="text-xs text-ink-muted num-tabular">
          {{ overview.settled_transactions }} transaksi settlement
        </p>
      </div>

      <div class="card-paper p-5 space-y-2">
        <div class="flex items-center justify-between text-xs text-ink-muted font-bold uppercase tracking-wider">
          <span>Service Worker &amp; AI</span>
          <Server class="w-4 h-4 text-criteria-terpenuhi" />
        </div>
        <div class="space-y-1 pt-0.5 text-xs">
          <div class="flex items-center justify-between">
            <span class="text-ink-muted">Worker Scoring:</span>
            <span
              class="font-bold inline-flex items-center gap-1"
              :class="overview.workers?.scoring_online ? 'text-criteria-terpenuhi' : 'text-criteria-sebagian'"
            >
              <CircleCheck v-if="overview.workers?.scoring_online" class="w-3.5 h-3.5" />
              <CircleAlert v-else class="w-3.5 h-3.5" />
              <span>{{ overview.workers?.scoring_online ? 'Siap' : 'Standby' }}</span>
            </span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-ink-muted">Worker Trainer:</span>
            <span
              class="font-bold inline-flex items-center gap-1"
              :class="overview.workers?.training_online ? 'text-criteria-terpenuhi' : 'text-criteria-sebagian'"
            >
              <CircleCheck v-if="overview.workers?.training_online" class="w-3.5 h-3.5" />
              <CircleAlert v-else class="w-3.5 h-3.5" />
              <span>{{ overview.workers?.training_online ? 'Siap' : 'Standby' }}</span>
            </span>
          </div>
        </div>
        <p class="text-[11px] text-ink-muted font-mono truncate">LazarusNLP/all-indobert-base-v4</p>
      </div>
    </div>

    <!-- URL Notifikasi Midtrans Aktif (Read-Only) -->
    <div class="card-paper p-5 space-y-2.5">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
        <div class="flex items-center gap-2">
          <Globe class="w-4 h-4 text-brand-700 dark:text-brand-400 shrink-0" />
          <h2 class="text-xs font-extrabold uppercase tracking-wider text-ink-muted">
            URL Notifikasi Webhook Midtrans Aktif (Read-Only)
          </h2>
        </div>
        <span
          class="px-2.5 py-0.5 rounded text-[11px] font-bold self-start sm:self-center"
          :class="
            overview.notification_url
              ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
              : 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
          "
        >
          {{
            overview.notification_url_source === 'public_base_url'
              ? 'Sumber: PUBLIC_BASE_URL (X-Override-Notification)'
              : overview.notification_url_source === 'tunnel'
                ? 'Sumber: Cloudflare Quick Tunnel Otomatis (X-Override-Notification)'
                : 'Sumber: Default Dashboard Midtrans'
          }}
        </span>
      </div>

      <input
        type="text"
        readonly
        :value="overview.notification_url || 'Belum ada tunnel aktif — Midtrans memakai Payment Notification URL di Dashboard'"
        class="input-field font-mono text-xs bg-ink-bg dark:bg-ink-darkBg cursor-default select-all"
      />
      <p class="text-[11px] text-ink-muted">
        URL ini dikirim otomatis pada header <code class="font-bold">X-Override-Notification</code> setiap kali guru membuat transaksi Snap baru, sehingga Anda tidak perlu memperbarui URL di Dashboard Midtrans saat tunnel dimulai ulang.
      </p>
    </div>

    <!-- Distribusi Guru per Paket -->
    <div v-if="overview.plan_distribution?.length" class="card-paper p-5 space-y-3">
      <h2 class="text-xs font-extrabold uppercase tracking-wider text-ink-muted">
        Distribusi Guru Berdasarkan Paket Langganan Aktif
      </h2>
      <div class="grid sm:grid-cols-3 gap-4">
        <div
          v-for="pd in overview.plan_distribution"
          :key="pd.plan_code"
          class="p-3.5 rounded-xl bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine flex items-center justify-between"
        >
          <div>
            <span class="font-bold text-sm block">{{ pd.plan_name }}</span>
            <span class="text-[11px] text-ink-muted uppercase font-mono">{{ pd.plan_code }}</span>
          </div>
          <span class="text-xl font-extrabold num-tabular text-brand-700 dark:text-brand-400">
            {{ pd.teacher_count }} guru
          </span>
        </div>
      </div>
    </div>

    <!-- Navigasi Tab Admin -->
    <div class="border-b border-ink-line dark:border-ink-darkLine flex flex-wrap gap-2">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        class="px-4 py-2.5 text-xs sm:text-sm font-extrabold border-b-2 transition-colors flex items-center gap-2 -mb-px"
        :class="
          activeTab === tab.key
            ? 'border-brand-500 text-brand-700 dark:text-brand-400'
            : 'border-transparent text-ink-muted hover:text-ink dark:hover:text-ink-darkText'
        "
        @click="activeTab = tab.key"
      >
        <component :is="tab.icon" class="w-4 h-4" />
        <span>{{ tab.label }}</span>
      </button>
    </div>

    <!-- TAB 1: KELOLA AKUN GURU -->
    <div v-if="activeTab === 'teachers'" class="card-paper p-5 sm:p-6 space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h2 class="font-extrabold text-base">Daftar Akun Guru</h2>
          <p class="text-xs text-ink-muted">
            Kelola status aktif/nonaktif akun guru serta pantau penggunaan kuota esai mereka.
          </p>
        </div>
        <div class="relative w-full sm:w-64">
          <Search class="w-4 h-4 text-ink-muted absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            v-model.trim="teacherSearch"
            type="text"
            placeholder="Cari nama, email, sekolah..."
            class="input-field pl-9 py-1.5 text-xs"
          />
        </div>
      </div>

      <div v-if="filteredTeachers.length === 0" class="p-8 text-center text-xs text-ink-muted border border-dashed border-ink-line dark:border-ink-darkLine rounded-xl">
        Tidak ada akun guru yang ditemukan.
      </div>

      <div v-else class="overflow-x-auto border border-ink-line dark:border-ink-darkLine rounded-xl">
        <table class="w-full text-left border-collapse text-xs">
          <thead class="bg-ink-bg dark:bg-ink-darkBg border-b border-ink-line dark:border-ink-darkLine">
            <tr>
              <th class="py-3 px-3.5 font-bold">Nama &amp; Email</th>
              <th class="py-3 px-3.5 font-bold">Institusi</th>
              <th class="py-3 px-3.5 font-bold">Paket</th>
              <th class="py-3 px-3.5 font-bold text-right">Sisa Kuota</th>
              <th class="py-3 px-3.5 font-bold text-center">Tugas / Model</th>
              <th class="py-3 px-3.5 font-bold text-center">Status</th>
              <th class="py-3 px-3.5 font-bold text-right">Aksi</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-ink-line dark:divide-ink-darkLine">
            <tr
              v-for="t in filteredTeachers"
              :key="t.id"
              class="hover:bg-ink-bg/60 dark:hover:bg-ink-darkBg/40"
            >
              <td class="py-2.5 px-3.5">
                <span class="font-bold block">{{ t.name }}</span>
                <span class="text-[11px] text-ink-muted">{{ t.email }}</span>
              </td>
              <td class="py-2.5 px-3.5 text-ink-muted">{{ t.institution || '-' }}</td>
              <td class="py-2.5 px-3.5">
                <span class="font-semibold block">{{ t.plan_name }}</span>
                <span v-if="t.expires_at" class="text-[10px] text-ink-muted num-tabular">
                  s/d {{ formatDate(t.expires_at) }}
                </span>
              </td>
              <td class="py-2.5 px-3.5 text-right font-extrabold num-tabular">
                {{ t.quota_remaining }} / {{ t.quota_total }}
              </td>
              <td class="py-2.5 px-3.5 text-center num-tabular">
                {{ t.assignment_count }} tugas / {{ t.model_count }} model
              </td>
              <td class="py-2.5 px-3.5 text-center">
                <span
                  class="px-2 py-0.5 rounded text-[11px] font-bold inline-flex items-center gap-1"
                  :class="
                    t.is_active
                      ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                      : 'bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-ink-muted'
                  "
                >
                  <CircleCheck v-if="t.is_active" class="w-3 h-3" />
                  <CircleX v-else class="w-3 h-3" />
                  <span>{{ t.is_active ? 'Aktif' : 'Nonaktif' }}</span>
                </span>
              </td>
              <td class="py-2.5 px-3.5 text-right">
                <button
                  type="button"
                  class="btn-secondary py-1 px-2.5 text-xs"
                  @click="handleToggleTeacher(t)"
                >
                  {{ t.is_active ? 'Nonaktifkan' : 'Aktifkan' }}
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- TAB 2: TRANSAKSI & LOG WEBHOOK MIDTRANS -->
    <div v-else-if="activeTab === 'transactions'" class="space-y-6">
      <div class="card-paper p-5 sm:p-6 space-y-4">
        <div>
          <h2 class="font-extrabold text-base">Daftar Transaksi Langganan &amp; Add-on</h2>
          <p class="text-xs text-ink-muted">
            Seluruh transaksi Midtrans Snap Sandbox yang dibuat oleh guru.
          </p>
        </div>

        <div v-if="transactions.length === 0" class="p-8 text-center text-xs text-ink-muted border border-dashed border-ink-line dark:border-ink-darkLine rounded-xl">
          Belum ada transaksi pembayaran.
        </div>

        <div v-else class="overflow-x-auto border border-ink-line dark:border-ink-darkLine rounded-xl">
          <table class="w-full text-left border-collapse text-xs">
            <thead class="bg-ink-bg dark:bg-ink-darkBg border-b border-ink-line dark:border-ink-darkLine">
              <tr>
                <th class="py-3 px-3.5 font-bold">Order ID</th>
                <th class="py-3 px-3.5 font-bold">Guru</th>
                <th class="py-3 px-3.5 font-bold">Paket</th>
                <th class="py-3 px-3.5 font-bold text-right">Nominal</th>
                <th class="py-3 px-3.5 font-bold text-center">Metode</th>
                <th class="py-3 px-3.5 font-bold text-center">Status</th>
                <th class="py-3 px-3.5 font-bold">Waktu</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-ink-line dark:divide-ink-darkLine">
              <tr v-for="tx in transactions" :key="tx.id" class="hover:bg-ink-bg/60 dark:hover:bg-ink-darkBg/40">
                <td class="py-2.5 px-3.5 font-mono font-bold whitespace-nowrap">{{ tx.order_id }}</td>
                <td class="py-2.5 px-3.5">
                  <span class="font-bold block">{{ tx.teacher_name }}</span>
                  <span class="text-[11px] text-ink-muted">{{ tx.teacher_email }}</span>
                </td>
                <td class="py-2.5 px-3.5 font-semibold">{{ tx.plan_name }}</td>
                <td class="py-2.5 px-3.5 text-right font-extrabold num-tabular whitespace-nowrap">
                  {{ formatRupiah(tx.gross_amount) }}
                </td>
                <td class="py-2.5 px-3.5 text-center uppercase text-[11px] font-semibold text-ink-muted">
                  {{ tx.payment_type || '-' }}
                </td>
                <td class="py-2.5 px-3.5 text-center">
                  <span
                    class="px-2 py-0.5 rounded text-[11px] font-bold"
                    :class="
                      tx.status === 'settlement'
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                        : tx.status === 'pending'
                          ? 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
                          : 'bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-ink-muted'
                    "
                  >
                    {{ tx.status }}
                  </span>
                </td>
                <td class="py-2.5 px-3.5 text-ink-muted num-tabular whitespace-nowrap">
                  {{ formatDateTime(tx.paid_at || tx.created_at) }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Tabel Log Webhook Midtrans -->
      <div class="card-paper p-5 sm:p-6 space-y-4">
        <div>
          <h2 class="font-extrabold text-base">Log Notifikasi Webhook Midtrans</h2>
          <p class="text-xs text-ink-muted">
            Audit penerimaan webhook `POST /api/payments/midtrans/notification` beserta status verifikasi signature SHA512 dan penanganan idempoten.
          </p>
        </div>

        <div v-if="webhookLogs.length === 0" class="p-8 text-center text-xs text-ink-muted border border-dashed border-ink-line dark:border-ink-darkLine rounded-xl">
          Belum ada log notifikasi webhook yang masuk.
        </div>

        <div v-else class="overflow-x-auto border border-ink-line dark:border-ink-darkLine rounded-xl">
          <table class="w-full text-left border-collapse text-xs">
            <thead class="bg-ink-bg dark:bg-ink-darkBg border-b border-ink-line dark:border-ink-darkLine">
              <tr>
                <th class="py-2.5 px-3 font-bold">Waktu</th>
                <th class="py-2.5 px-3 font-bold">Order ID</th>
                <th class="py-2.5 px-3 font-bold text-center">Status Midtrans</th>
                <th class="py-2.5 px-3 font-bold text-center">Signature SHA512</th>
                <th class="py-2.5 px-3 font-bold text-center">Idempoten</th>
                <th class="py-2.5 px-3 font-bold">Catatan Pemrosesan</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-ink-line dark:divide-ink-darkLine">
              <tr v-for="log in webhookLogs" :key="log.id" class="hover:bg-ink-bg/60 dark:hover:bg-ink-darkBg/40">
                <td class="py-2 px-3 text-ink-muted num-tabular whitespace-nowrap">
                  {{ formatDateTime(log.created_at) }}
                </td>
                <td class="py-2 px-3 font-mono font-bold whitespace-nowrap">{{ log.order_id }}</td>
                <td class="py-2 px-3 text-center font-mono">
                  {{ log.transaction_status }} ({{ log.status_code }})
                </td>
                <td class="py-2 px-3 text-center">
                  <span
                    class="px-2 py-0.5 rounded text-[10px] font-bold inline-flex items-center gap-1"
                    :class="
                      log.signature_valid
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
                        : 'bg-brand-100 text-brand-800 dark:bg-brand-900/40 dark:text-brand-300'
                    "
                  >
                    <CircleCheck v-if="log.signature_valid" class="w-3 h-3" />
                    <CircleX v-else class="w-3 h-3" />
                    <span>{{ log.signature_valid ? 'Valid' : 'Ditolak' }}</span>
                  </span>
                </td>
                <td class="py-2 px-3 text-center">
                  <span v-if="log.processed_idempotently" class="px-2 py-0.5 rounded bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-200 text-[10px] font-bold">
                    Duplikat (Idempoten)
                  </span>
                  <span v-else class="text-ink-muted">-</span>
                </td>
                <td class="py-2 px-3 text-ink-muted">{{ log.processed_note }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- TAB 3: PENGATURAN PAKET LANGGANAN & ADD-ON -->
    <div v-else-if="activeTab === 'plans'" class="grid md:grid-cols-2 gap-5">
      <div
        v-for="p in plans"
        :key="p.id"
        class="card-paper p-5 sm:p-6 space-y-4"
      >
        <div class="flex items-center justify-between border-b border-ink-line dark:border-ink-darkLine pb-3">
          <div>
            <span class="text-[11px] font-mono uppercase font-bold text-brand-700 dark:text-brand-400">
              Kode: {{ p.code }} {{ p.is_addon ? '(Add-on)' : '' }}
            </span>
            <h3 class="font-extrabold text-base">{{ p.name }}</h3>
          </div>
          <label class="flex items-center gap-2 text-xs font-bold cursor-pointer">
            <input
              v-model="p.is_active"
              type="checkbox"
              class="rounded border-ink-line text-brand-500 focus:ring-brand-500"
            />
            <span>Aktif</span>
          </label>
        </div>

        <div class="space-y-3 text-xs">
          <div>
            <label class="block font-bold mb-1">Nama Paket</label>
            <input v-model.trim="p.name" type="text" class="input-field text-xs" />
          </div>

          <div>
            <label class="block font-bold mb-1">Deskripsi</label>
            <textarea v-model.trim="p.description" rows="2" class="input-field text-xs"></textarea>
          </div>

          <div class="grid grid-cols-3 gap-3">
            <div>
              <label class="block font-bold mb-1">Harga (Rp)</label>
              <input
                v-model.number="p.price_idr"
                type="number"
                min="0"
                step="1000"
                class="input-field text-xs num-tabular"
              />
            </div>
            <div>
              <label class="block font-bold mb-1">Kuota Esai</label>
              <input
                v-model.number="p.quota_essays"
                type="number"
                min="1"
                step="5"
                class="input-field text-xs num-tabular"
              />
            </div>
            <div>
              <label class="block font-bold mb-1">Masa Aktif (Hari)</label>
              <input
                v-model.number="p.duration_days"
                type="number"
                min="0"
                step="1"
                class="input-field text-xs num-tabular"
              />
            </div>
          </div>
        </div>

        <div class="pt-2 flex justify-end">
          <button
            type="button"
            class="btn-primary text-xs py-1.5 px-4"
            :disabled="savingPlanId === p.id"
            @click="handleSavePlan(p)"
          >
            <Loader2 v-if="savingPlanId === p.id" class="w-3.5 h-3.5 animate-spin" />
            <Save v-else class="w-3.5 h-3.5" />
            <span>Simpan Paket</span>
          </button>
        </div>
      </div>
    </div>

    <!-- TAB 4: PENGATURAN AMBANG AI -->
    <div v-else-if="activeTab === 'ai'" class="max-w-2xl">
      <form class="card-paper p-6 space-y-5" @submit.prevent="handleSaveAISettings">
        <div class="border-b border-ink-line dark:border-ink-darkLine pb-3">
          <h2 class="font-extrabold text-base">Konfigurasi Ambang Penilaian &amp; Kemiripan AI</h2>
          <p class="text-xs text-ink-muted mt-0.5">
            Model Dasar: <code class="font-bold text-brand-700 dark:text-brand-400">LazarusNLP/all-indobert-base-v4</code> (Mean Pooling + L2 Norm di CPU)
          </p>
        </div>

        <div class="grid sm:grid-cols-2 gap-4 text-xs">
          <div>
            <label class="block font-bold mb-1">Ambang Tercakup Penuh (T_HIGH)</label>
            <input
              v-model.number="aiForm.t_high"
              type="number"
              step="0.01"
              min="0.1"
              max="0.99"
              required
              class="input-field num-tabular"
            />
            <p class="text-[11px] text-ink-muted mt-1">
              Default terkalibrasi: <strong>0.60</strong>. Poin kunci dengan cosine &gt;= T_HIGH berstatus tercakup penuh (s' = 1.0).
            </p>
          </div>

          <div>
            <label class="block font-bold mb-1">Ambang Batas Bawah (T_LOW)</label>
            <input
              v-model.number="aiForm.t_low"
              type="number"
              step="0.01"
              min="0.05"
              max="0.95"
              required
              class="input-field num-tabular"
            />
            <p class="text-[11px] text-ink-muted mt-1">
              Default terkalibrasi: <strong>0.45</strong>. Poin kunci dengan cosine &lt; T_LOW berstatus belum ada (s' = 0.0).
            </p>
          </div>

          <div>
            <label class="block font-bold mb-1">Ambang Kemiripan Semantik (S_SEM)</label>
            <input
              v-model.number="aiForm.s_sem"
              type="number"
              step="0.01"
              min="0.5"
              max="1.0"
              required
              class="input-field num-tabular"
            />
            <p class="text-[11px] text-ink-muted mt-1">
              Default terkalibrasi: <strong>0.92</strong>. Cosine similarity minimum antar embedding dokumen siswa.
            </p>
          </div>

          <div>
            <label class="block font-bold mb-1">Ambang Kemiripan Leksikal (S_LEX)</label>
            <input
              v-model.number="aiForm.s_lex"
              type="number"
              step="0.01"
              min="0.2"
              max="1.0"
              required
              class="input-field num-tabular"
            />
            <p class="text-[11px] text-ink-muted mt-1">
              Default terkalibrasi: <strong>0.60</strong>. TF-IDF n-gram cosine minimum untuk membedakan jawaban mirip vs kebetulan satu topik.
            </p>
          </div>

          <div class="sm:col-span-2">
            <label class="block font-bold mb-1">Minimal Sampel Pelatihan Model Custom (MIN_TRAIN_SAMPLES)</label>
            <input
              v-model.number="aiForm.min_train_samples"
              type="number"
              step="1"
              min="5"
              max="500"
              required
              class="input-field num-tabular max-w-xs"
            />
            <p class="text-[11px] text-ink-muted mt-1">
              Jumlah minimum sampel jawaban bertarget nilai (dari revisi guru atau unggahan Excel) sebelum model custom dapat dilatih (default: <strong>20</strong>).
            </p>
          </div>
        </div>

        <div class="pt-3 border-t border-ink-line dark:border-ink-darkLine flex items-center justify-end">
          <button type="submit" class="btn-primary text-xs px-5" :disabled="savingAI">
            <Loader2 v-if="savingAI" class="w-4 h-4 animate-spin" />
            <Save v-else class="w-4 h-4" />
            <span>Simpan Pengaturan AI</span>
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import {
  RefreshCw,
  Users,
  FileCheck2,
  CreditCard,
  Server,
  CircleCheck,
  CircleAlert,
  CircleX,
  Search,
  Receipt,
  Package,
  Sliders,
  Save,
  Loader2,
  Globe
} from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import ScoreCircleHighlight from '@/components/ScoreCircleHighlight.vue'
import { apiFetch } from '@/utils/api'
import { useToastStore } from '@/stores/toast'

const toastStore = useToastStore()

const loading = ref(true)
const activeTab = ref('teachers')
const teacherSearch = ref('')

const tabs = [
  { key: 'teachers', label: 'Kelola Guru', icon: Users },
  { key: 'transactions', label: 'Transaksi & Webhook', icon: Receipt },
  { key: 'plans', label: 'Pengaturan Paket', icon: Package },
  { key: 'ai', label: 'Pengaturan Ambang AI', icon: Sliders }
]

const overview = reactive({
  total_teachers: 0,
  active_teachers: 0,
  inactive_teachers: 0,
  total_assignments: 0,
  total_scored_essays: 0,
  total_custom_models: 0,
  settled_transactions: 0,
  total_revenue_idr: 0,
  plan_distribution: [],
  notification_url: '',
  notification_url_source: 'dashboard_default',
  workers: { scoring_online: true, training_online: true }
})

const teachers = ref([])
const transactions = ref([])
const webhookLogs = ref([])
const plans = ref([])
const savingPlanId = ref(null)
const savingAI = ref(false)

const aiForm = reactive({
  t_high: 0.60,
  t_low: 0.45,
  s_sem: 0.92,
  s_lex: 0.60,
  min_train_samples: 20
})

const filteredTeachers = computed(() => {
  const q = teacherSearch.value.toLowerCase()
  if (!q) return teachers.value
  return teachers.value.filter(
    (t) =>
      t.name?.toLowerCase().includes(q) ||
      t.email?.toLowerCase().includes(q) ||
      t.institution?.toLowerCase().includes(q)
  )
})

function formatRupiah(amount) {
  if (!amount || amount === 0) return 'Rp 0'
  return 'Rp ' + Number(amount).toLocaleString('id-ID')
}

function formatDate(isoStr) {
  if (!isoStr) return '-'
  return new Date(isoStr).toLocaleDateString('id-ID', {
    day: 'numeric',
    month: 'short',
    year: 'numeric'
  })
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

async function fetchAllAdminData() {
  loading.value = true
  try {
    const [ovRes, tRes, txRes, pRes, aiRes] = await Promise.all([
      apiFetch('/api/admin/overview'),
      apiFetch('/api/admin/teachers'),
      apiFetch('/api/admin/transactions'),
      apiFetch('/api/admin/plans'),
      apiFetch('/api/admin/ai-settings')
    ])
    Object.assign(overview, ovRes)
    teachers.value = tRes.teachers || []
    transactions.value = txRes.transactions || []
    webhookLogs.value = txRes.webhook_logs || []
    plans.value = pRes.plans || []
    if (aiRes.ai_settings) {
      Object.assign(aiForm, aiRes.ai_settings)
    }
  } catch (err) {
    toastStore.error(err.message || 'Gagal memuat data dashboard administrator.')
  } finally {
    loading.value = false
  }
}

async function handleToggleTeacher(teacher) {
  try {
    const nextState = !teacher.is_active
    const res = await apiFetch(`/api/admin/teachers/${teacher.id}/status`, {
      method: 'PUT',
      body: JSON.stringify({ is_active: nextState })
    })
    toastStore.success(res.message)
    await fetchAllAdminData()
  } catch (err) {
    toastStore.error(err.message)
  }
}

async function handleSavePlan(plan) {
  savingPlanId.value = plan.id
  try {
    const res = await apiFetch(`/api/admin/plans/${plan.id}`, {
      method: 'PUT',
      body: JSON.stringify(plan)
    })
    toastStore.success(res.message)
    await fetchAllAdminData()
  } catch (err) {
    toastStore.error(err.message)
  } finally {
    savingPlanId.value = null
  }
}

async function handleSaveAISettings() {
  savingAI.value = true
  try {
    const res = await apiFetch('/api/admin/ai-settings', {
      method: 'PUT',
      body: JSON.stringify(aiForm)
    })
    toastStore.success(res.message)
    await fetchAllAdminData()
  } catch (err) {
    toastStore.error(err.message)
  } finally {
    savingAI.value = false
  }
}

onMounted(() => {
  fetchAllAdminData()
})
</script>
