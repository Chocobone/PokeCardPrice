<script setup lang="ts">
import { computed } from 'vue'

// 쪽 넘김. 640px 이상은 현재 쪽 주변 번호를, 모바일은 이전·다음과 "n / N" 만 보여 준다.
const props = defineProps<{ page: number; totalPages: number }>()
const emit = defineEmits<{ go: [page: number] }>()

type Item = number | 'gap'

// 첫 쪽, 마지막 쪽, 현재 쪽 ±2 를 보이고 사이는 "…" 로 줄인다.
const items = computed<Item[]>(() => {
  const { page, totalPages: n } = props
  const pages = new Set([1, n])
  for (let p = page - 2; p <= page + 2; p++) if (p >= 1 && p <= n) pages.add(p)
  const sorted = [...pages].sort((a, b) => a - b)
  const out: Item[] = []
  sorted.forEach((p, i) => {
    const prev = sorted[i - 1]
    if (prev !== undefined && p - prev > 1) out.push('gap')
    out.push(p)
  })
  return out
})

function go(p: number) {
  if (p >= 1 && p <= props.totalPages && p !== props.page) emit('go', p)
}
</script>

<template>
  <nav
    v-if="totalPages > 1"
    class="pagination"
    aria-label="쪽 넘김"
  >
    <button
      type="button"
      :disabled="page <= 1"
      @click="go(page - 1)"
    >
      <span aria-hidden="true">‹</span> 이전
    </button>
    <ol class="numbers">
      <li
        v-for="(it, i) in items"
        :key="it === 'gap' ? `gap-${i}` : it"
      >
        <span
          v-if="it === 'gap'"
          class="gap"
          aria-hidden="true"
        >…</span>
        <button
          v-else
          type="button"
          :class="{ current: it === page }"
          :aria-current="it === page ? 'page' : undefined"
          :aria-label="`${it}쪽`"
          @click="go(it)"
        >
          {{ it }}
        </button>
      </li>
    </ol>
    <span class="compact">{{ page }} / {{ totalPages }}</span>
    <button
      type="button"
      :disabled="page >= totalPages"
      @click="go(page + 1)"
    >
      다음 <span aria-hidden="true">›</span>
    </button>
  </nav>
</template>

<style scoped>
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  margin-top: var(--space-6);
}

button {
  min-width: 2.25rem;
  min-height: 2.25rem;
  padding: 0 var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
}

button:disabled {
  opacity: 0.4;
  cursor: default;
}

button.current {
  background: var(--color-accent);
  border-color: var(--color-accent);
  color: var(--color-accent-contrast);
  font-weight: 700;
}

.numbers {
  display: none;
  gap: var(--space-1);
  list-style: none;
  margin: 0;
  padding: 0;
}

.numbers button {
  padding: 0 var(--space-2);
}

.gap {
  display: inline-block;
  padding: 0 var(--space-1);
  color: var(--color-text-muted);
}

.compact {
  font-variant-numeric: tabular-nums;
  color: var(--color-text-muted);
}

@media (min-width: 640px) {
  .numbers {
    display: flex;
  }

  .compact {
    display: none;
  }
}
</style>
