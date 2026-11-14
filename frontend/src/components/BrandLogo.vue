<template>
  <span class="inline-flex items-center shrink-0">
    <!-- Ikon saja untuk header ponsel sempit bila compactOnMobile aktif -->
    <img
      v-if="compactOnMobile"
      src="/brand/koreksiin-icon.svg"
      alt="Koreksi.in"
      class="sm:hidden rounded-lg object-contain select-none"
      :class="iconSizeClass"
    />
    <!-- Logo horizontal (otomatis berganti sesuai mode terang/gelap) -->
    <img
      :src="themeStore.isDark ? '/brand/koreksiin-logo-gelap.svg' : '/brand/koreksiin-logo.svg'"
      alt="Koreksi.in"
      class="rounded-md object-contain select-none"
      :class="[compactOnMobile ? 'hidden sm:block' : 'block', sizeClass]"
    />
  </span>
</template>

<script setup>
import { computed } from 'vue'
import { useThemeStore } from '@/stores/theme'

const props = defineProps({
  size: {
    type: String,
    default: 'md' // 'sm' | 'md' | 'lg'
  },
  compactOnMobile: {
    type: Boolean,
    default: false
  }
})

const themeStore = useThemeStore()

const sizeClass = computed(() => {
  if (props.size === 'sm') return 'h-7 w-auto'
  if (props.size === 'lg') return 'h-11 sm:h-12 w-auto'
  return 'h-9 sm:h-10 w-auto'
})

const iconSizeClass = computed(() => {
  if (props.size === 'sm') return 'w-7 h-7'
  if (props.size === 'lg') return 'w-10 h-10'
  return 'w-8 h-8'
})
</script>

