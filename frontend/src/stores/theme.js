import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(localStorage.getItem('koreksiin_theme') === 'dark')

  function applyTheme() {
    const root = document.documentElement
    if (isDark.value) {
      root.classList.add('dark')
      localStorage.setItem('koreksiin_theme', 'dark')
    } else {
      root.classList.remove('dark')
      localStorage.setItem('koreksiin_theme', 'light')
    }
  }

  function toggleTheme() {
    isDark.value = !isDark.value
    applyTheme()
  }

  applyTheme()

  return {
    isDark,
    toggleTheme,
    applyTheme
  }
})

