<template>
  <div
    class="fixed bottom-5 right-5 z-50 flex flex-col gap-2.5 max-w-sm w-full pointer-events-none px-4 sm:px-0"
    aria-live="polite"
  >
    <TransitionGroup
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="translate-y-3 opacity-0"
      enter-to-class="translate-y-0 opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="translate-y-0 opacity-100"
      leave-to-class="translate-y-2 opacity-0"
    >
      <div
        v-for="t in toastStore.toasts"
        :key="t.id"
        class="pointer-events-auto card-paper p-3.5 flex items-start gap-3 border-l-4"
        :class="
          t.type === 'error'
            ? 'border-l-brand-700 bg-brand-50/95 dark:bg-ink-darkCard'
            : 'border-l-criteria-terpenuhi bg-white dark:bg-ink-darkCard'
        "
      >
        <CircleAlert
          v-if="t.type === 'error'"
          class="w-5 h-5 text-brand-700 dark:text-brand-400 shrink-0 mt-0.5"
        />
        <CircleCheck
          v-else
          class="w-5 h-5 text-criteria-terpenuhi shrink-0 mt-0.5"
        />
        <p class="text-sm font-medium text-ink dark:text-ink-darkText flex-1 leading-snug">
          {{ t.message }}
        </p>
        <button
          type="button"
          class="text-ink-muted hover:text-ink dark:hover:text-ink-darkText p-0.5 rounded"
          @click="toastStore.remove(t.id)"
          aria-label="Tutup notifikasi"
        >
          <X class="w-4 h-4" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { CircleAlert, CircleCheck, X } from 'lucide-vue-next'
import { useToastStore } from '@/stores/toast'

const toastStore = useToastStore()
</script>

