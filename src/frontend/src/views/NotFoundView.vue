<template>
  <section class="not-found">
    <h1>페이지를 찾을 수 없습니다</h1>
    <p class="muted">
      주소가 바뀌었거나 없는 페이지입니다.
    </p>
    <RouterLink to="/">
      홈으로
    </RouterLink>
  </section>
</template>

<style scoped>
.not-found {
  text-align: center;
  padding-block: var(--space-8);
}
</style>
