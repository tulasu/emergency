<script setup lang="ts">
import type { TbIconName } from '~/types/ui';

defineProps<{
  name: TbIconName;
}>();
</script>

<template>
  <span class="tb-icon">
    <svg v-if="name === 'arrow-left'" viewBox="0 0 24 24">
      <path d="M15 18l-6-6 6-6" />
    </svg>
    <svg v-else-if="name === 'plus'" viewBox="0 0 24 24">
      <path d="M12 5v14M5 12h14" />
    </svg>
    <svg v-else-if="name === 'layers'" viewBox="0 0 24 24">
      <path d="M12 3l9 5-9 5-9-5 9-5zM3 12l9 5 9-5M3 16l9 5 9-5" />
    </svg>
    <svg v-else-if="name === 'folder'" viewBox="0 0 24 24">
      <path d="M3 7h6l2 2h10v10H3z" />
    </svg>
    <svg v-else-if="name === 'users'" viewBox="0 0 24 24">
      <path
        d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM22 21v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75"
      />
    </svg>
    <svg v-else-if="name === 'user-plus'" viewBox="0 0 24 24">
      <path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M19 8v6M22 11h-6" />
    </svg>
    <svg v-else-if="name === 'settings'" viewBox="0 0 24 24">
      <circle cx="12" cy="12" r="3" />
      <path
        d="M19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.8-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.8.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.8 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.8l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.8.3H9a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5 1.7 1.7 0 0 0 1.8-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.8V9c0 .7.4 1.3 1.1 1.5H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z"
      />
    </svg>
    <svg v-else-if="name === 'log-out'" viewBox="0 0 24 24">
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9" />
    </svg>
    <svg v-else-if="name === 'user'" viewBox="0 0 24 24">
      <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" />
      <circle cx="12" cy="7" r="4" />
    </svg>
    <svg v-else-if="name === 'copy'" viewBox="0 0 24 24">
      <rect x="9" y="9" width="13" height="13" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
    <svg v-else-if="name === 'chevron-down'" viewBox="0 0 24 24">
      <path d="m6 9 6 6 6-6" />
    </svg>
    <svg v-else-if="name === 'chevron-right'" viewBox="0 0 24 24">
      <path d="m9 18 6-6-6-6" />
    </svg>
    <svg v-else-if="name === 'check'" viewBox="0 0 24 24">
      <path d="M20 6 9 17l-5-5" />
    </svg>
    <svg v-else-if="name === 'x'" viewBox="0 0 24 24">
      <path d="M18 6 6 18M6 6l12 12" />
    </svg>
    <svg v-else-if="name === 'file'" viewBox="0 0 24 24">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <path d="M14 2v6h6" />
    </svg>
    <svg v-else-if="name === 'search'" viewBox="0 0 24 24">
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.5-3.5" />
    </svg>
    <svg v-else-if="name === 'chart'" viewBox="0 0 24 24">
      <path d="M4 19V5M4 19h16M8 17V10M12 17V7M16 17v-4" />
    </svg>
    <svg v-else-if="name === 'book'" viewBox="0 0 24 24">
      <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" />
      <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
    </svg>
    <svg v-else-if="name === 'home'" viewBox="0 0 24 24">
      <path d="M3 10.5 12 3l9 7.5" />
      <path d="M5 9.5V21h14V9.5" />
    </svg>
    <svg v-else-if="name === 'archive'" viewBox="0 0 24 24">
      <rect x="3" y="4" width="18" height="4" rx="1" />
      <path d="M5 8v11a1 1 0 0 0 1 1h12a1 1 0 0 0 1-1V8M10 12h4" />
    </svg>
    <svg v-else-if="name === 'calendar'" viewBox="0 0 24 24">
      <rect x="3" y="5" width="18" height="16" rx="2" />
      <path d="M16 3v4M8 3v4M3 11h18" />
    </svg>
    <svg v-else-if="name === 'pencil'" viewBox="0 0 24 24">
      <path d="M12 20h9" />
      <path d="M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4 12.5-12.5z" />
    </svg>
    <svg v-else-if="name === 'play'" viewBox="0 0 24 24">
      <path d="M8 5v14l11-7z" fill="currentColor" stroke="none" />
    </svg>
    <svg v-else-if="name === 'clock'" viewBox="0 0 24 24">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 2" />
    </svg>
    <svg v-else-if="name === 'lock'" viewBox="0 0 24 24">
      <rect x="5" y="11" width="14" height="10" rx="2" />
      <path d="M8 11V8a4 4 0 0 1 8 0v3" />
    </svg>
    <svg v-else-if="name === 'info'" viewBox="0 0 24 24">
      <circle cx="12" cy="12" r="9" />
      <path d="M12 10v6M12 7h.01" />
    </svg>
  </span>
</template>

<style scoped>
.tb-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  color: inherit;
  flex-shrink: 0;
  vertical-align: middle;
}

svg {
  display: block;
  width: 100%;
  height: 100%;
  fill: none;
  stroke: currentColor;
  stroke-width: 2;
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>
