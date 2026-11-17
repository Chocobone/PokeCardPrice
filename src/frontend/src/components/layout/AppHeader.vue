<script setup lang="ts">
import { SITE_NAME } from '@/router'
</script>

<template>
  <header class="header">
    <div class="container header-inner">
      <RouterLink
        to="/"
        class="brand"
        aria-label="홈으로"
      >
        <span
          class="brand-mark"
          aria-hidden="true"
        >◆</span>
        {{ SITE_NAME }}
      </RouterLink>
      <!-- 검색창은 3단계에서 SearchBox 로 채운다 -->
      <div class="header-search" />
    </div>
  </header>
</template>

<style scoped>
.header {
  position: sticky;
  top: 0;
  z-index: 5;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.header-inner {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  min-height: var(--header-height);
}

.brand {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-weight: 700;
  font-size: var(--font-lg);
  color: var(--color-text);
  text-decoration: none;
  white-space: nowrap;
}

.brand-mark {
  color: var(--color-accent);
}

.header-search {
  flex: 1;
  min-width: 0;
}
</style>
