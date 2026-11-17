<script setup lang="ts">
import type { CardSummary } from '@/api/types'

import CardTile from './CardTile.vue'

// 카드 격자. 열 수는 화면 폭에 따라 바뀐다 (모바일 2열 ~ 데스크톱 6열).
defineProps<{ cards: CardSummary[] }>()
</script>

<template>
  <ul class="grid">
    <CardTile
      v-for="c in cards"
      :key="c.id"
      :card="c"
    />
  </ul>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-3);
  list-style: none;
  margin: 0;
  padding: 0;
}

@media (min-width: 640px) {
  .grid {
    grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
    gap: var(--space-4);
  }
}
</style>
