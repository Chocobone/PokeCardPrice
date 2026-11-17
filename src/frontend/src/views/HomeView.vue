<script setup lang="ts">
import MoversPanel from '@/components/movers/MoversPanel.vue'
import SearchBox from '@/components/SearchBox.vue'
</script>

<template>
  <section class="hero">
    <h1>포켓몬 카드 시세</h1>
    <p class="muted">
      영어판·일본판·한글판 카드의 시세를 한곳에서 비교합니다.
    </p>
    <SearchBox variant="hero" />
  </section>
  <MoversPanel />
</template>

<style scoped>
.hero {
  max-width: 720px;
  margin: 0 auto var(--space-8);
  text-align: center;
}

.hero p {
  margin-bottom: var(--space-5);
}
</style>
