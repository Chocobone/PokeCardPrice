<script setup lang="ts">
import type { MoversResponse } from '@/api/types'
import StateView from '@/components/common/StateView.vue'

import MoverRow from './MoverRow.vue'

// 급등 또는 급락 목록 하나. 로딩·오류·빈 목록 상태를 함께 다룬다.
defineProps<{
  title: string
  tone: 'up' | 'down'
  result?: MoversResponse
  loading?: boolean
  error?: string
}>()
defineEmits<{ retry: [] }>()
</script>

<template>
  <section
    class="mover-list"
    :aria-labelledby="`movers-${tone}`"
  >
    <h3
      :id="`movers-${tone}`"
      :class="tone"
    >
      {{ title }}
    </h3>
    <StateView
      v-if="error"
      kind="error"
      :message="error"
      retryable
      @retry="$emit('retry')"
    />
    <StateView
      v-else-if="loading || !result"
      kind="loading"
    />
    <StateView
      v-else-if="result.items.length === 0"
      kind="empty"
      message="시세 데이터가 쌓이면 표시됩니다."
    />
    <ol v-else>
      <MoverRow
        v-for="(m, i) in result.items"
        :key="`${m.card.id}:${m.variant}`"
        :rank="i + 1"
        :mover="m"
      />
    </ol>
  </section>
</template>

<style scoped>
.mover-list {
  min-width: 0;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-2);
}

h3 {
  margin: 0 var(--space-3) var(--space-2);
  font-size: var(--font-lg);
}

h3.up {
  color: var(--color-up);
}

h3.down {
  color: var(--color-down);
}

ol {
  list-style: none;
  margin: 0;
  padding: 0;
}
</style>
