<template>
  <span class="relative inline-flex items-center justify-center px-3 py-1">
    <svg
      class="absolute inset-0 w-full h-full text-brand-500 dark:text-brand-400 pointer-events-none overflow-visible"
      viewBox="0 0 120 60"
      fill="none"
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <path
        class="svg-draw-circle"
        d="M15 12C45 2 108 5 114 29C120 53 72 58 34 55C6 52 2 33 12 17C19 6 42 5 64 7"
        stroke="currentColor"
        stroke-width="2.75"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    <span class="relative z-10 num-tabular">
      <slot />
    </span>
  </span>
</template>

