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
          Kelola kuota penilaian esai siswa dan beli paket prabayar atau kuota tambahan (Add-on) secara instan melalui Midtrans Snap (QRIS &amp; GoPay).
        </p>
      </div>

      <button
        type="button"
        class="btn-secondary text-xs"
        :disabled="refreshing"
        @click="refreshAll"
      >
        <RefreshCw class="w-4 h-4" :class="refreshing ? 'animate-spin' : ''" />
        <span>Muat Ulang Status Kuota</span>
      </button>
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

    <!-- Pilihan Paket Prabayar & Add-on -->
    <div class="space-y-4">
      <div class="flex flex-col sm:flex-row sm:items-end justify-between gap-2">
        <div>
          <h2 class="text-lg font-extrabold tracking-tight">Pilih Paket Langganan atau Add-on Kuota</h2>
          <p class="text-xs text-ink-muted mt-0.5">
            Pembayaran prabayar via Midtrans Snap Sandbox (QRIS / GoPay). Kuota dan masa aktif diperbarui otomatis setelah notifikasi webhook diterima.
          </p>
        </div>
      </div>

      <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-5">
        <div
          v-for="plan in plans"
          :key="plan.code"
          class="card-paper p-5 flex flex-col justify-between space-y-5"
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
                {{ plan.is_addon ? 'Add-on Kuota' : plan.code === 'monthly' ? 'Paling Populer' : 'Paket Prabayar' }}
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
                <span>Model Custom (Kalibrasi &amp; LoRA)</span>
              </li>
            </ul>
          </div>

          <div class="pt-3 border-t border-ink-line dark:border-ink-darkLine">
            <button
              v-if="plan.price_idr > 0"
              type="button"
              class="w-full"
              :class="plan.code === 'monthly' ? 'btn-primary' : 'btn-secondary'"
              :disabled="checkingOutCode === plan.code"
              @click="handleCheckout(plan)"
            >
              <Loader2 v-if="checkingOutCode === plan.code" class="w-4 h-4 animate-spin" />
              <CreditCard v-else class="w-4 h-4" />
              <span>{{ plan.is_addon ? 'Beli Add-on Kuota' : 'Beli Paket Ini' }}</span>
            </button>
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
      <div class="flex items-center justify-between gap-4">
        <div>
          <h2 class="font-extrabold text-base flex items-center gap-2">
            <Receipt class="w-4 h-4 text-brand-700 dark:text-brand-400" />
            <span>Riwayat Transaksi Pembayaran</span>
          </h2>
          <p class="text-xs text-ink-muted mt-0.5">
            Daftar pesanan paket dan add-on Anda. Transaksi berstatus <strong>Menunggu Pembayaran</strong> dapat dibuka kembali melalui popup Snap atau tautan pembayaran Midtrans, lalu disimulasikan melalui
            <a
              href="https://simulator.sandbox.midtrans.com/v2/qris/index"
              target="_blank"
              rel="noopener noreferrer"
              class="text-brand-700 dark:text-brand-400 underline font-semibold"
            >Simulator QRIS Midtrans Sandbox (v2)</a>.
          </p>
        </div>
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
              <td class="py-2.5 px-3.5 text-center uppercase text-[11px] font-semibold text-ink-muted">
                {{ tx.payment_type || '-' }}
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
                    v-if="tx.snap_token"
                    type="button"
                    class="btn-primary py-1 px-2.5 text-xs"
                    @click="openSnapToken(tx.snap_token, tx.snap_redirect_url)"
                  >
                    Bayar (Snap)
                  </button>
                  <a
                    v-if="tx.snap_redirect_url"
                    :href="tx.snap_redirect_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="btn-secondary py-1 px-2.5 text-xs inline-flex items-center gap-1"
                  >
                    <span>Link Snap</span>
                    <ExternalLink class="w-3 h-3" />
                  </a>
                </div>
                <span v-else class="text-[11px] text-ink-muted">-</span>
              </td>
            </tr>
          </tbody>
        </table>
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
  Receipt,
  ExternalLink
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
const checkingOutCode = ref('')
const snapReady = ref(false)

let pollTimer = null

const sub = computed(() => authStore.user?.subscription)

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
  if (st === 'settlement' || st === 'capture') return 'Lunas (Settlement)'
  if (st === 'pending') return 'Menunggu Pembayaran'
  if (st === 'expire') return 'Kedaluwarsa'
  if (st === 'cancel' || st === 'deny') return 'Dibatalkan'
  return st
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

async function refreshAll(silent = false) {
  if (!silent) refreshing.value = true
  try {
    const [plansRes, txRes] = await Promise.all([
      apiFetch('/api/plans'),
      apiFetch('/api/payments/transactions'),
      authStore.refreshProfile()
    ])
    plans.value = plansRes.plans || []
    transactions.value = txRes.transactions || []
  } catch (err) {
    if (!silent) toastStore.error(err.message || 'Gagal memuat data langganan.')
  } finally {
    if (!silent) refreshing.value = false
  }
}

function openSnapToken(token, redirectUrl) {
  if (window.snap && token) {
    window.snap.pay(token, {
      onSuccess: () => {
        toastStore.show('Pembayaran dikirim. Menunggu konfirmasi webhook Midtrans...')
        setTimeout(() => refreshAll(true), 2000)
      },
      onPending: () => {
        toastStore.show('Transaksi dibuat. Selesaikan pembayaran QRIS/GoPay Anda.')
        refreshAll(true)
      },
      onError: () => {
        toastStore.error('Terjadi kendala pada jendela pembayaran Snap.')
        refreshAll(true)
      },
      onClose: () => {
        toastStore.show('Jendela pembayaran ditutup. Status paket hanya berubah setelah pembayaran terkonfirmasi via webhook.')
        refreshAll(true)
      }
    })
  } else if (redirectUrl) {
    window.open(redirectUrl, '_blank', 'noopener,noreferrer')
  }
}

async function handleCheckout(plan) {
  checkingOutCode.value = plan.code
  try {
    const res = await apiFetch('/api/payments/checkout', {
      method: 'POST',
      body: JSON.stringify({ plan_code: plan.code })
    })
    toastStore.success(`Pesanan ${res.order_id} dibuat.`)
    await refreshAll(true)
    openSnapToken(res.snap_token, res.snap_redirect_url)
  } catch (err) {
    toastStore.error(err.message || 'Gagal memulai pembayaran Midtrans.')
  } finally {
    checkingOutCode.value = ''
  }
}

onMounted(async () => {
  await Promise.all([loadSnapScript(), refreshAll()])
  pollTimer = setInterval(() => {
    const hasPending = transactions.value.some((t) => t.status === 'pending')
    if (hasPending) {
      refreshAll(true)
    }
  }, 5000)
})

onBeforeUnmount(() => {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
})
</script>
