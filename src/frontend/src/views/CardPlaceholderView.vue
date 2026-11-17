<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()
</script>

<template>
  <section>
    <h1>카드 상세</h1>
    <p class="muted">
      카드 #{{ route.params.id }} 의 상세 화면은 준비 중입니다.
    </p>
    <RouterLink to="/">
      홈으로
    </RouterLink>
  </section>
</template>
