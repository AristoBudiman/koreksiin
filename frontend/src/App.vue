<template>
  <div>
    <RouterView v-slot="{ Component }">
      <Transition name="page-fade" mode="out-in">
        <component :is="Component" />
      </Transition>
    </RouterView>
    <ToastContainer />
  </div>
</template>

<script setup>
import ToastContainer from '@/components/ToastContainer.vue'
import { useThemeStore } from '@/stores/theme'

// Pastikan preferensi tema diterapkan sejak awal
useThemeStore()
</script>

