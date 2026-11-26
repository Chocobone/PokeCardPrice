<script setup lang="ts">
import type { CardSummary, Language } from '@/api/types'
import CardGrid from '@/components/cards/CardGrid.vue'

// 한글판 ↔ 일본판 연결 카드. 한글판 카드에는 일본판 시세를 참고하라는 안내를 붙인다 (한글판 시세는 #7).
defineProps<{
  language: Language
  cards: CardSummary[]
}>()
</script>

<template>
  <section class="linked">
    <h2>{{ language === 'ko' ? '같은 카드의 일본판' : '같은 카드의 한글판' }}</h2>
    <p
      v-if="language === 'ko'"
      class="muted"
    >
      한글판 시세가 준비될 때까지 같은 카드의 일본판 시세를 참고할 수 있습니다.
    </p>
    <CardGrid :cards="cards" />
  </section>
</template>

<style scoped>
.linked h2 {
  font-size: var(--font-lg);
}
</style>
