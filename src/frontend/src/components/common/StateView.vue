<script setup lang="ts">
// 로딩·빈 결과·오류 공통 표시. 오류일 때 retry 이벤트를 주면 "다시 시도" 버튼이 나온다.
defineProps<{
  kind: 'loading' | 'empty' | 'error'
  message?: string
  retryable?: boolean
}>()
defineEmits<{ retry: [] }>()
</script>

<template>
  <div
    class="state"
    :class="kind"
    :role="kind === 'error' ? 'alert' : 'status'"
    :aria-busy="kind === 'loading'"
  >
    <span
      v-if="kind === 'loading'"
      class="spinner"
      aria-hidden="true"
    />
    <p>{{ message ?? (kind === 'loading' ? '불러오는 중…' : kind === 'empty' ? '표시할 내용이 없습니다.' : '불러오지 못했습니다.') }}</p>
    <button
      v-if="kind === 'error' && retryable"
      type="button"
      class="retry"
      @click="$emit('retry')"
    >
      다시 시도
    </button>
    <slot />
  </div>
</template>

<style scoped>
.state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-6) var(--space-4);
  text-align: center;
  color: var(--color-text-muted);
}

.state p {
  margin: 0;
}

.error p {
  color: var(--color-danger);
}

.spinner {
  width: 24px;
  height: 24px;
  border: 3px solid var(--color-border);
  border-top-color: var(--color-accent);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinner {
    animation-duration: 2.4s;
  }
}

.retry {
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
}
</style>
