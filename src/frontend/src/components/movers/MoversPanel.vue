<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import type { MoverPeriod } from '@/api/types'
import { moverKey, useMoversStore } from '@/stores/movers'
import { formatDate } from '@/utils/format'

import MoverList from './MoverList.vue'

// 홈의 급등·급락 영역. 기간 탭(7일·30일)을 바꾸면 해당 기간의 두 목록을 불러온다.
const PERIODS: { value: MoverPeriod; label: string }[] = [
  { value: '7d', label: '7일' },
  { value: '30d', label: '30일' },
]

const store = useMoversStore()
const period = ref<MoverPeriod>('7d')

watch(
  period,
  (p) => {
    void store.load(p, 'up')
    void store.load(p, 'down')
  },
  { immediate: true },
)

const up = computed(() => moverKey(period.value, 'up'))
const down = computed(() => moverKey(period.value, 'down'))
const asOf = computed(() => store.results[up.value]?.asOf ?? store.results[down.value]?.asOf ?? null)
</script>

<template>
  <section
    class="movers"
    aria-labelledby="movers-title"
  >
    <div class="head">
      <h2 id="movers-title">
        급등·급락
      </h2>
      <div
        class="tabs"
        role="tablist"
        aria-label="기간"
      >
        <button
          v-for="p in PERIODS"
          :key="p.value"
          type="button"
          role="tab"
          :aria-selected="period === p.value"
          :class="{ active: period === p.value }"
          @click="period = p.value"
        >
          {{ p.label }}
        </button>
      </div>
    </div>
    <p
      v-if="asOf"
      class="muted as-of"
    >
      {{ formatDate(asOf) }} 기준 · 기간 시작 대비 변동률
    </p>
    <div class="lists">
      <MoverList
        title="급등"
        tone="up"
        :result="store.results[up]"
        :loading="store.loading[up]"
        :error="store.errors[up]"
        @retry="store.load(period, 'up', { force: true })"
      />
      <MoverList
        title="급락"
        tone="down"
        :result="store.results[down]"
        :loading="store.loading[down]"
        :error="store.errors[down]"
        @retry="store.load(period, 'down', { force: true })"
      />
    </div>
  </section>
</template>

<style scoped>
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
}

h2 {
  margin: 0;
  font-size: var(--font-xl);
}

.tabs {
  display: inline-flex;
  padding: 2px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface-muted);
}

.tabs button {
  border: 0;
  border-radius: calc(var(--radius-md) - 2px);
  padding: var(--space-1) var(--space-4);
  background: transparent;
  color: var(--color-text-muted);
  font-weight: 600;
}

.tabs button.active {
  background: var(--color-surface);
  color: var(--color-text);
  box-shadow: var(--shadow-sm);
}

.as-of {
  font-size: var(--font-sm);
}

.lists {
  display: grid;
  gap: var(--space-4);
}

@media (min-width: 1024px) {
  .lists {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
