<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'

import SearchBox from '@/components/SearchBox.vue'
import { SITE_NAME } from '@/router'

// 홈에는 큰 검색창이 있어 헤더 검색창을 숨긴다.
const route = useRoute()
const showSearch = computed(() => route.name !== 'home')
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
      <div
        v-if="showSearch"
        class="header-search"
      >
        <SearchBox variant="compact" />
      </div>
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
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-4);
  min-height: var(--header-height);
  padding-block: var(--space-2);
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

/* 모바일: 검색창은 다음 줄 전체 폭. 640px 이상: 이름 옆 */
.header-search {
  flex: 1 1 100%;
  min-width: 0;
}

@media (min-width: 640px) {
  .header-search {
    flex: 1 1 auto;
    max-width: 480px;
    margin-left: auto;
  }
}
</style>
