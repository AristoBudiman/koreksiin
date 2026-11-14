<template>
  <span class="num-tabular">{{ formattedValue }}</span>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'

const props = defineProps({
  value: {
    type: Number,
    default: 0
  },
  decimals: {
    type: Number,
    default: 1
  },
  duration: {
    type: Number,
    default: 650
  }
})

const displayValue = ref(0)

const formattedValue = computed(() => {
  const num = Number(displayValue.value || 0)
  if (Number.isInteger(num) && props.decimals === 0) {
    return num.toString()
  }
  return Number.isInteger(num) ? num.toString() : num.toFixed(props.decimals)
})

function animateTo(target) {
  const prefersReduced =
    typeof window !== 'undefined' &&
    window.matchMedia &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches

  if (prefersReduced || props.duration <= 0) {
    displayValue.value = target
    return
  }

  const startVal = displayValue.value
  const diff = target - startVal
  const startTime = performance.now()

  function step(now) {
    const elapsed = now - startTime
    const progress = Math.min(elapsed / props.duration, 1)
    const eased = 1 - Math.pow(1 - progress, 3)
    displayValue.value = startVal + diff * eased
    if (progress < 1) {
      requestAnimationFrame(step)
    } else {
      displayValue.value = target
    }
  }

  requestAnimationFrame(step)
}

watch(
  () => props.value,
  (newVal) => {
    animateTo(Number(newVal || 0))
  }
)

onMounted(() => {
  animateTo(Number(props.value || 0))
})
</script>

