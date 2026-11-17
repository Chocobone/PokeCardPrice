<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

// 4단계에서 검색 결과 화면으로 채운다.
const route = useRoute()
const q = computed(() => (typeof route.query.q === 'string' ? route.query.q : ''))
</script>

<template>
  <section>
    <h1>검색</h1>
    <p class="muted">
      검색어: {{ q || '(없음)' }}
    </p>
  </section>
</template>
