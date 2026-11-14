<template>
  <div class="space-y-8">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-4">
      <div>
        <HandDrawnUnderline>
          <h1 class="text-2xl sm:text-3xl font-extrabold tracking-tight">
            Langganan &amp; Kuota Penilaian
          </h1>
        </HandDrawnUnderline>
        <p class="text-xs sm:text-sm text-ink-muted mt-2 max-w-2xl">
          Kelola kuota penilaian esai siswa dan beli paket atau kuota tambahan (Add-on). Pembayaran lewat QRIS, e-wallet, transfer bank (Virtual Account), gerai retail, atau kartu.
        </p>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="btn-primary text-xs"
          :disabled="reconciling || refreshing"
          @click="handleReconcilePayments"
        >
          <RefreshCw class="w-4 h-4" :class="reconciling ? 'animate-spin' : ''" />
          <span>Cek status pembayaran</span>
        </button>
        <button
          type="button"
          class="btn-secondary text-xs"
          :disabled="refreshing || reconciling"
          @click="refreshAll(false, true)"
        >
          <RefreshCw class="w-4 h-4" :class="refreshing ? 'animate-spin' : ''" />
          <span>Muat Ulang Kuota</span>
        </button>
      </div>
    </div>

    <!-- Kartu Status Langganan Saat Ini -->
    <div class="card-paper p-6 space-y-5">
      <div class="flex flex-col md:flex-row md:items-center justify-between gap-6">
        <div class="space-y-1.5">
          <div class="flex items-center gap-2">
            <span class="px-2.5 py-0.5 rounded bg-brand-50 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 text-xs font-bold uppercase tracking-wider">
              Paket Aktif Saat Ini
            </span>
            <span
              v-if="sub?.plan_code && sub.plan_code !== 'free'"
              class="px-2 py-0.5 rounded bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300 text-xs font-bold inline-flex items-center gap-1"
            >
              <CircleCheck class="w-3.5 h-3.5" />
              <span>Berbayar Aktif</span>
            </span>
          </div>
          <h2 class="text-2xl font-extrabold">{{ sub?.plan_name || 'Paket Gratis' }}</h2>
          <p class="text-xs text-ink-muted">
            <template v-if="sub?.expires_at">
              Masa aktif paket berakhir pada: <strong class="text-ink dark:text-ink-darkText">{{ formatDateTime(sub.expires_at) }}</strong>
            </template>
            <template v-else-if="sub?.quota_reset_at">
              Kuota gratis direset otomatis setiap bulan. Jadwal reset berikutnya: <strong class="text-ink dark:text-ink-darkText">{{ formatDateTime(sub.quota_reset_at) }}</strong>
            </template>
          </p>
        </div>

        <div class="flex items-center gap-6">
          <div class="text-right">
            <span class="text-xs font-bold uppercase text-ink-muted block">Sisa Kuota Esai</span>
            <div class="flex items-baseline justify-end gap-1.5 mt-0.5">
              <ScoreCircleHighlight>
                <span class="text-2xl sm:text-3xl font-extrabold text-brand-700 dark:text-brand-400 num-tabular">
                  <CountUpNumber :value="sub?.quota_remaining ?? 0" :decimals="0" />
                </span>
              </ScoreCircleHighlight>
              <span class="text-xs text-ink-muted num-tabular">
                / {{ sub?.quota_total ?? 0 }} esai
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Bar Penggunaan Kuota -->
      <div class="space-y-1.5 pt-2 border-t border-ink-line dark:border-ink-darkLine">
        <div class="flex items-center justify-between text-xs">
          <span class="text-ink-muted font-medium">
            Terpakai: <strong class="text-ink dark:text-ink-darkText num-tabular">{{ sub?.quota_used ?? 0 }} jawaban siswa</strong>
            (Menilai ulang jawaban siswa yang sama tidak memotong kuota lagi)
          </span>
          <span class="font-bold num-tabular text-brand-700 dark:text-brand-400">
            {{ usagePercent }}% terpakai
          </span>
        </div>
        <div class="w-full h-2.5 rounded-full bg-ink-line dark:bg-ink-darkLine overflow-hidden">
          <div
            class="h-full bg-brand-500 transition-all duration-300"
            :style="{ width: `${usagePercent}%` }"
          ></div>
        </div>
      </div>
    </div>

    <!-- Banner Transaksi Menunggu Pembayaran -->
    <div
      v-if="pendingTx"
      class="card-paper p-5 border-2 border-amber-400 dark:border-amber-600 bg-amber-50/70 dark:bg-amber-950/30 flex flex-col sm:flex-row sm:items-center justify-between gap-4"
    >
      <div class="space-y-1">
        <div class="flex items-center gap-2">
          <Clock class="w-4 h-4 text-amber-700 dark:text-amber-300 shrink-0" />
          <span class="text-xs font-extrabold uppercase tracking-wider text-amber-800 dark:text-amber-200">
            Transaksi Menunggu Pembayaran
          </span>
        </div>
        <p class="text-sm font-bold text-ink dark:text-ink-darkText">
          {{ pendingTx.plan_name }} — {{ formatRupiah(pendingTx.gross_amount) }}
        </p>
        <p class="text-xs text-ink-muted">
          Order ID: <span class="font-mono font-semibold">{{ pendingTx.order_id }}</span>. Selesaikan atau batalkan transaksi ini sebelum membuat pesanan baru.
        </p>
      </div>
      <div class="flex flex-wrap items-center gap-2 shrink-0">
        <button
          type="button"
          class="btn-primary text-xs"
          @click="openPaymentWindow(pendingTx.snap_token, pendingTx.snap_redirect_url)"
        >
          <CreditCard class="w-3.5 h-3.5" />
          <span>Lanjutkan pembayaran</span>
        </button>
        <button
          type="button"
          class="btn-secondary text-xs text-red-700 dark:text-red-300 border-red-300 dark:border-red-800 hover:bg-red-50 dark:hover:bg-red-950/40"
          :disabled="cancellingId === pendingTx.id"
          @click="handleCancelPending(pendingTx)"
        >
          <Loader2 v-if="cancellingId === pendingTx.id" class="w-3.5 h-3.5 animate-spin" />
          <CircleX v-else class="w-3.5 h-3.5" />
          <span>Batalkan</span>
        </button>
      </div>
    </div>

    <!-- Pilihan Paket & Add-on -->
    <div class="space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-2">
        <div>
          <h2 class="text-lg font-extrabold tracking-tight">Pilih Paket Langganan atau Add-on Kuota</h2>
          <p class="text-xs text-ink-muted mt-0.5">
            Pembayaran lewat QRIS, e-wallet, transfer bank (Virtual Account), gerai retail, atau kartu. Kuota dan masa aktif diperbarui otomatis setelah pembayaran lunas.
          </p>
        </div>
      </div>

      <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div
          v-for="plan in plans"
          :key="plan.code"
          class="card-paper p-5 flex flex-col justify-between space-y-5 transition-all duration-200 hover:-translate-y-1 hover:border-brand-500 hover:shadow-paperHover"
          :class="plan.code === 'monthly' ? 'border-2 border-brand-500 ring-4 ring-brand-500/10' : ''"
        >
          <div class="space-y-3.5">
            <div class="flex items-center justify-between gap-2">
              <span
                class="px-2.5 py-0.5 rounded text-xs font-bold"
                :class="
                  plan.is_addon
                    ? 'bg-amber-100 text-amber-900 dark:bg-amber-900/40 dark:text-amber-200'
                    : plan.code === 'monthly'
                      ? 'bg-brand-100 text-brand-800 dark:bg-brand-900/50 dark:text-brand-200'
                      : 'bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-ink-muted'
                "
              >
                {{
                  plan.is_addon
                    ? 'Add-on Kuota'
                    : plan.code === 'free'
                      ? 'Gratis'
                      : plan.code === 'monthly'
                        ? 'Paling Populer'
                        : 'Paket Berbayar'
                }}
              </span>
              <span
                v-if="sub?.plan_code === plan.code && !plan.is_addon"
                class="text-[11px] font-extrabold text-criteria-terpenuhi inline-flex items-center gap-1"
              >
                <CircleCheck class="w-3.5 h-3.5" />
                <span>Aktif</span>
              </span>
            </div>

            <div>
              <h3 class="font-extrabold text-base">{{ plan.name }}</h3>
              <p class="text-xs text-ink-muted mt-1 min-h-[34px] leading-relaxed">
                {{ plan.description }}
              </p>
            </div>

            <div class="pt-2 border-t border-ink-line dark:border-ink-darkLine">
              <span class="text-2xl font-extrabold num-tabular">
                {{ formatRupiah(plan.price_idr) }}
              </span>
              <span class="text-[11px] text-ink-muted block mt-0.5">
                {{
                  plan.is_addon
                    ? 'Mengikuti masa aktif paket saat ini'
                    : plan.code === 'free'
                      ? 'Direset setiap 30 hari'
                      : `${plan.duration_days} hari masa aktif`
                }}
              </span>
            </div>

            <ul class="space-y-2 text-xs pt-1">
              <li class="flex items-center gap-2">
                <CircleCheck class="w-4 h-4 text-criteria-terpenuhi shrink-0" />
                <span class="font-bold num-tabular">
                  {{ plan.is_addon ? `+${plan.quota_essays}` : plan.quota_essays }} kuota jawaban siswa
                </span>
              </li>
              <li class="flex items-center gap-2">
                <CircleCheck class="w-4 h-4 text-criteria-terpenuhi shrink-0" />
                <span>Deteksi kemiripan &amp; ekspor Excel</span>
              </li>
              <li class="flex items-center gap-2">
                <CircleCheck class="w-4 h-4 text-criteria-terpenuhi shrink-0" />
                <span>Model custom yang belajar dari gaya menilai Anda</span>
              </li>
            </ul>
          </div>

          <div class="pt-3 border-t border-ink-line dark:border-ink-darkLine space-y-2">
            <template v-if="plan.price_idr > 0">
              <button
                type="button"
                class="w-full disabled:opacity-50 disabled:cursor-not-allowed"
                :class="plan.code === 'monthly' ? 'btn-primary' : 'btn-secondary'"
                :disabled="checkingOutCode === plan.code || getPlanActionState(plan).disabled"
                @click="openCheckoutPreview(plan)"
              >
                <Loader2 v-if="checkingOutCode === plan.code" class="w-4 h-4 animate-spin" />
                <CreditCard v-else class="w-4 h-4" />
                <span>{{ plan.is_addon ? 'Beli Add-on Kuota' : 'Beli Paket Ini' }}</span>
              </button>
              <p
                v-if="getPlanActionState(plan).reason"
                class="text-[11px] text-ink-muted leading-snug"
              >
                {{ getPlanActionState(plan).reason }}
              </p>
            </template>
            <div
              v-else
              class="w-full py-2 px-3 rounded-lg bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-center text-xs font-semibold text-ink-muted"
            >
              Paket Dasar Bawaan
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Riwayat Transaksi Pembayaran Guru -->
    <div class="card-paper p-5 sm:p-6 space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 class="font-extrabold text-base flex items-center gap-2">
            <Receipt class="w-4 h-4 text-brand-700 dark:text-brand-400" />
            <span>Riwayat Transaksi Pembayaran</span>
          </h2>
          <p class="text-xs text-ink-muted mt-0.5">
            Daftar pesanan paket dan add-on Anda. Transaksi berstatus <strong>Menunggu Pembayaran</strong> dapat dilanjutkan atau dibatalkan sebelum membuat pesanan baru. Jika status belum berubah setelah membayar, klik <strong>Cek status pembayaran</strong>.
          </p>
        </div>

        <button
          type="button"
          class="btn-secondary text-xs shrink-0 self-start sm:self-center"
          :disabled="reconciling"
          @click="handleReconcilePayments"
        >
          <RefreshCw class="w-3.5 h-3.5" :class="reconciling ? 'animate-spin' : ''" />
          <span>Cek status pembayaran</span>
        </button>
      </div>

      <div v-if="transactions.length === 0" class="p-8 text-center text-xs text-ink-muted border border-dashed border-ink-line dark:border-ink-darkLine rounded-xl">
        Belum ada riwayat transaksi pembayaran.
      </div>

      <div v-else class="overflow-x-auto border border-ink-line dark:border-ink-darkLine rounded-xl">
        <table class="w-full text-left border-collapse text-xs">
          <thead class="bg-ink-bg dark:bg-ink-darkBg border-b border-ink-line dark:border-ink-darkLine">
            <tr>
              <th class="py-3 px-3.5 font-bold">Order ID</th>
              <th class="py-3 px-3.5 font-bold">Paket / Add-on</th>
              <th class="py-3 px-3.5 font-bold text-right">Nominal</th>
              <th class="py-3 px-3.5 font-bold text-center">Metode</th>
              <th class="py-3 px-3.5 font-bold text-center">Status</th>
              <th class="py-3 px-3.5 font-bold">Waktu</th>
              <th class="py-3 px-3.5 font-bold text-right">Tindakan</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-ink-line dark:divide-ink-darkLine">
            <tr
              v-for="tx in transactions"
              :key="tx.id"
              class="hover:bg-ink-bg/60 dark:hover:bg-ink-darkBg/40"
            >
              <td class="py-2.5 px-3.5 font-mono font-bold whitespace-nowrap">
                {{ tx.order_id }}
              </td>
              <td class="py-2.5 px-3.5">
                <span class="font-bold block">{{ tx.plan_name }}</span>
                <span class="text-[11px] text-ink-muted num-tabular">+{{ tx.quota_essays }} kuota esai</span>
              </td>
              <td class="py-2.5 px-3.5 text-right font-extrabold num-tabular whitespace-nowrap">
                {{ formatRupiah(tx.gross_amount) }}
              </td>
              <td class="py-2.5 px-3.5 text-center text-[11px] font-semibold text-ink dark:text-ink-darkText whitespace-nowrap">
                {{ formatPaymentMethod(tx.payment_type) }}
              </td>
              <td class="py-2.5 px-3.5 text-center">
                <span
                  class="px-2 py-0.5 rounded text-[11px] font-bold inline-flex items-center gap-1"
                  :class="statusBadgeClass(tx.status)"
                >
                  <CircleCheck v-if="tx.status === 'settlement'" class="w-3 h-3" />
                  <Clock v-else-if="tx.status === 'pending'" class="w-3 h-3" />
                  <CircleX v-else class="w-3 h-3" />
                  <span>{{ formatStatusLabel(tx.status) }}</span>
                </span>
              </td>
              <td class="py-2.5 px-3.5 text-ink-muted whitespace-nowrap num-tabular">
                {{ formatDateTime(tx.paid_at || tx.created_at) }}
              </td>
              <td class="py-2.5 px-3.5 text-right whitespace-nowrap">
                <div v-if="tx.status === 'pending'" class="inline-flex items-center gap-1.5">
                  <button
                    type="button"
                    class="btn-primary py-1 px-2.5 text-xs"
                    @click="openPaymentWindow(tx.snap_token, tx.snap_redirect_url)"
                  >
                    Lanjutkan pembayaran
                  </button>
                  <button
                    type="button"
                    class="btn-secondary py-1 px-2.5 text-xs text-red-700 dark:text-red-300 border-red-300 dark:border-red-800"
                    :disabled="cancellingId === tx.id"
                    @click="handleCancelPending(tx)"
                  >
                    Batalkan
                  </button>
                </div>
                <span v-else class="text-[11px] text-ink-muted">-</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Modal Ringkasan Sebelum Pembayaran -->
    <div
      v-if="previewPlan"
      class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4"
      @click.self="previewPlan = null"
    >
      <div class="card-paper max-w-md w-full p-6 space-y-5 shadow-2xl">
        <div class="space-y-1">
          <h3 class="text-lg font-extrabold tracking-tight">Konfirmasi Pembelian</h3>
          <p class="text-xs text-ink-muted">
            Periksa detail paket dan kuota Anda sebelum melanjutkan ke halaman pembayaran.
          </p>
        </div>

        <div class="p-4 rounded-xl bg-brand-50/70 dark:bg-brand-900/30 border border-brand-200 dark:border-brand-800 space-y-2 text-xs">
          <p class="font-bold text-ink dark:text-ink-darkText leading-relaxed">
            Setelah pembayaran, paket Anda:
            <span class="text-brand-700 dark:text-brand-300">{{ previewSummary.planName }}</span>,
            aktif sampai:
            <span class="text-brand-700 dark:text-brand-300">{{ previewSummary.expiresLabel }}</span>,
            kuota:
            <span class="text-brand-700 dark:text-brand-300">{{ previewSummary.quotaRemaining }} esai</span>.
          </p>
          <p class="text-ink-muted">
            Total pembayaran: <strong class="text-ink dark:text-ink-darkText">{{ formatRupiah(previewPlan.price_idr) }}</strong>
          </p>
        </div>

        <div class="flex items-center justify-end gap-2">
          <button
            type="button"
            class="btn-secondary text-xs"
            :disabled="checkingOutCode === previewPlan.code"
            @click="previewPlan = null"
          >
            Batal
          </button>
          <button
            type="button"
            class="btn-primary text-xs"
            :disabled="checkingOutCode === previewPlan.code"
            @click="confirmCheckout"
          >
            <Loader2 v-if="checkingOutCode === previewPlan.code" class="w-4 h-4 animate-spin" />
            <CreditCard v-else class="w-4 h-4" />
            <span>Lanjutkan ke Pembayaran</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import {
  RefreshCw,
  CircleCheck,
  CircleX,
  Clock,
  CreditCard,
  Loader2,
  Receipt
} from 'lucide-vue-next'
import HandDrawnUnderline from '@/components/HandDrawnUnderline.vue'
import ScoreCircleHighlight from '@/components/ScoreCircleHighlight.vue'
import CountUpNumber from '@/components/CountUpNumber.vue'
import { useAuthStore } from '@/stores/auth'
import { useToastStore } from '@/stores/toast'
import { apiFetch } from '@/utils/api'

const authStore = useAuthStore()
const toastStore = useToastStore()

const plans = ref([])
const transactions = ref([])
const refreshing = ref(false)
const reconciling = ref(false)
const checkingOutCode = ref('')
const cancellingId = ref(null)
const previewPlan = ref(null)
const snapReady = ref(false)

let pollTimer = null

const sub = computed(() => authStore.user?.subscription)

const pendingTx = computed(() => transactions.value.find((t) => t.status === 'pending') || null)

const isPaidActive = computed(() => {
  if (!sub.value || !sub.value.plan_code || sub.value.plan_code === 'free') return false
  if (!sub.value.expires_at) return true
  return new Date(sub.value.expires_at) > new Date()
})

const previewSummary = computed(() => {
  const plan = previewPlan.value
  if (!plan) return { planName: '-', expiresLabel: '-', quotaRemaining: 0 }

  const now = new Date()
  const curRemaining = Math.max(0, Number(sub.value?.quota_remaining) || 0)

  if (plan.is_addon) {
    const exp = sub.value?.expires_at ? formatDateTime(sub.value.expires_at) : '-'
    return {
      planName: sub.value?.plan_name || plan.name,
      expiresLabel: exp,
      quotaRemaining: curRemaining + Number(plan.quota_essays || 0)
    }
  }

  let base = now
  if (isPaidActive.value && sub.value?.expires_at && new Date(sub.value.expires_at) > now) {
    base = new Date(sub.value.expires_at)
  }
  const nextExp = new Date(base.getTime())
  nextExp.setDate(nextExp.getDate() + Number(plan.duration_days || 0))

  const nextQuota = isPaidActive.value
    ? curRemaining + Number(plan.quota_essays || 0)
    : Number(plan.quota_essays || 0)

  return {
    planName: plan.name,
    expiresLabel: formatDateTime(nextExp.toISOString()),
    quotaRemaining: nextQuota
  }
})

function getPlanActionState(plan) {
  if (pendingTx.value) {
    return {
      disabled: true,
      reason: 'Selesaikan atau batalkan transaksi yang menunggu pembayaran terlebih dahulu.'
    }
  }
  if (plan.is_addon && !isPaidActive.value) {
    return {
      disabled: true,
      reason: 'Add-on hanya tersedia saat paket Bulanan atau Tahunan sedang aktif.'
    }
  }
  if (!plan.is_addon && plan.code === 'monthly' && isPaidActive.value && sub.value?.plan_code === 'yearly') {
    return {
      disabled: true,
      reason: 'Paket Sekolah Tahunan masih aktif. Beli Add-on jika kuota kurang.'
    }
  }
  return { disabled: false, reason: '' }
}

const usagePercent = computed(() => {
  const total = Number(sub.value?.quota_total) || 0
  const used = Number(sub.value?.quota_used) || 0
  if (total <= 0) return 0
  return Math.min(100, Math.round((used / total) * 100))
})

function formatRupiah(amount) {
  if (!amount || amount === 0) return 'Rp 0'
  return 'Rp ' + Number(amount).toLocaleString('id-ID')
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

function formatStatusLabel(st) {
  if (st === 'settlement' || st === 'capture') return 'Lunas'
  if (st === 'pending') return 'Menunggu Pembayaran'
  if (st === 'expire') return 'Kedaluwarsa'
  if (st === 'cancel' || st === 'deny') return 'Dibatalkan'
  return st
}

function formatPaymentMethod(pt) {
  if (!pt) return 'Belum dipilih'
  const key = String(pt).toLowerCase().trim()
  const labels = {
    qris: 'QRIS',
    other_qris: 'QRIS',
    gopay: 'GoPay',
    shopeepay: 'ShopeePay',
    va_bca: 'BCA Virtual Account',
    bca_va: 'BCA Virtual Account',
    va_bni: 'BNI Virtual Account',
    bni_va: 'BNI Virtual Account',
    va_bri: 'BRI Virtual Account',
    bri_va: 'BRI Virtual Account',
    va_permata: 'Permata Virtual Account',
    permata_va: 'Permata Virtual Account',
    va_cimb: 'CIMB Virtual Account',
    cimb_va: 'CIMB Virtual Account',
    va_danamon: 'Danamon Virtual Account',
    va_bsi: 'BSI Virtual Account',
    echannel: 'Mandiri Bill / VA',
    bank_transfer: 'Transfer Bank (VA)',
    cstore_indomaret: 'Indomaret',
    cstore_alfamart: 'Alfamart',
    cstore: 'Gerai Retail',
    credit_card: 'Kartu Kredit / Debit',
    akulaku: 'Akulaku',
    kredivo: 'Kredivo'
  }
  if (labels[key]) return labels[key]
  if (key.startsWith('va_')) return key.slice(3).toUpperCase() + ' Virtual Account'
  if (key.startsWith('cstore_')) return 'Gerai ' + key.slice(7).toUpperCase()
  return pt.replace(/_/g, ' ').toUpperCase()
}

function statusBadgeClass(st) {
  if (st === 'settlement' || st === 'capture') {
    return 'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300'
  }
  if (st === 'pending') {
    return 'bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300'
  }
  return 'bg-ink-bg dark:bg-ink-darkBg border border-ink-line dark:border-ink-darkLine text-ink-muted'
}

async function loadSnapScript() {
  try {
    const cfg = await apiFetch('/api/payments/config')
    if (!cfg.client_key || !cfg.snap_js_url) return
    if (window.snap) {
      snapReady.value = true
      return
    }
    const existing = document.getElementById('midtrans-snap-script')
    if (existing) {
      snapReady.value = true
      return
    }
    const script = document.createElement('script')
    script.id = 'midtrans-snap-script'
    script.src = cfg.snap_js_url
    script.setAttribute('data-client-key', cfg.client_key)
    script.onload = () => {
      snapReady.value = true
    }
    document.head.appendChild(script)
  } catch {
    // Abaikan bila gagal memuat konfigurasi awal
  }
}

async function refreshAll(silent = false, reconcile = false) {
  if (!silent) refreshing.value = true
  try {
    const txUrl = reconcile ? '/api/payments/transactions?reconcile=1' : '/api/payments/transactions'
    const [plansRes, txRes] = await Promise.all([
      apiFetch('/api/plans'),
      apiFetch(txUrl)
    ])
    await authStore.refreshProfile()
    plans.value = plansRes.plans || []
    transactions.value = txRes.transactions || []
  } catch (err) {
    if (!silent) toastStore.error(err.message || 'Gagal memuat data langganan.')
  } finally {
    if (!silent) refreshing.value = false
  }
}

async function handleReconcilePayments() {
  reconciling.value = true
  try {
    const res = await apiFetch('/api/payments/reconcile', {
      method: 'POST'
    })
    await refreshAll(true, false)
    if (res.settled_count > 0) {
      toastStore.success(res.message)
    } else {
      toastStore.show(res.message || 'Status pembayaran telah diperiksa.')
    }
  } catch (err) {
    toastStore.error(err.message || 'Gagal mengecek status pembayaran.')
  } finally {
    reconciling.value = false
  }
}

function openPaymentWindow(token, redirectUrl) {
  if (window.snap && token) {
    window.snap.pay(token, {
      onSuccess: () => {
        toastStore.show('Pembayaran dikirim. Memperbarui status paket Anda...')
        setTimeout(() => refreshAll(true, true), 1500)
      },
      onPending: () => {
        toastStore.show('Transaksi dibuat. Selesaikan pembayaran Anda.')
        refreshAll(true, false)
      },
      onError: () => {
        toastStore.error('Terjadi kendala pada jendela pembayaran.')
        refreshAll(true, true)
      },
      onClose: () => {
        toastStore.show('Jendela pembayaran ditutup. Memeriksa status pembayaran terbaru...')
        refreshAll(true, true)
      }
    })
  } else if (redirectUrl) {
    window.open(redirectUrl, '_blank', 'noopener,noreferrer')
  }
}

function openCheckoutPreview(plan) {
  const state = getPlanActionState(plan)
  if (state.disabled) {
    toastStore.error(state.reason)
    return
  }
  previewPlan.value = plan
}

async function confirmCheckout() {
  if (!previewPlan.value) return
  const plan = previewPlan.value
  checkingOutCode.value = plan.code
  try {
    const res = await apiFetch('/api/payments/checkout', {
      method: 'POST',
      body: JSON.stringify({ plan_code: plan.code })
    })
    previewPlan.value = null
    toastStore.success(`Pesanan ${res.order_id} dibuat.`)
    await refreshAll(true, false)
    openPaymentWindow(res.snap_token, res.snap_redirect_url)
  } catch (err) {
    toastStore.error(err.message || 'Gagal memulai pembayaran.')
    await refreshAll(true, false)
  } finally {
    checkingOutCode.value = ''
  }
}

async function handleCancelPending(tx) {
  cancellingId.value = tx.id
  try {
    await apiFetch(`/api/payments/transactions/${tx.id}/cancel`, {
      method: 'POST'
    })
    toastStore.success('Transaksi menunggu pembayaran berhasil dibatalkan.')
    await refreshAll(true, false)
  } catch (err) {
    toastStore.error(err.message || 'Gagal membatalkan transaksi.')
  } finally {
    cancellingId.value = null
  }
}

function handleVisibilityChange() {
  if (document.visibilityState === 'visible') {
    refreshAll(true, true)
  }
}

onMounted(async () => {
  await Promise.all([loadSnapScript(), refreshAll(false, true)])
  document.addEventListener('visibilitychange', handleVisibilityChange)
  pollTimer = setInterval(() => {
    const hasPending = transactions.value.some((t) => t.status === 'pending')
    if (hasPending) {
      refreshAll(true, true)
    }
  }, 5000)
})

onBeforeUnmount(() => {
  document.removeEventListener('visibilitychange', handleVisibilityChange)
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})
</script>
