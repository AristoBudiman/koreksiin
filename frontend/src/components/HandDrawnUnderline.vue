<template>
  <div class="inline-block">
    <slot />
    <svg
      class="w-full h-2.5 mt-1 text-brand-500 dark:text-brand-400 overflow-visible"
      viewBox="0 0 200 10"
      fill="none"
      preserveAspectRatio="none"
      aria-hidden="true"
    >
      <path
        class="svg-draw-line"
        d="M2 7.5C45 3.2 135 2.5 198 6.8"
        stroke="currentColor"
        stroke-width="3.5"
        stroke-linecap="round"
      />
      <path
        class="svg-draw-line"
        d="M14 8.8C70 6.2 140 5.9 188 8.2"
        stroke="currentColor"
        stroke-width="1.8"
        stroke-linecap="round"
        opacity="0.7"
      />
    </svg>
  </div>
</template>

