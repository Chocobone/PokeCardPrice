<script setup lang="ts">
import { computed } from 'vue'

import type { Language, Price } from '@/api/types'
import { formatKrw, formatPrice, formatShortDate, isStale, sourceLabel, STALE_DAYS, variantLabel } from '@/utils/format'

// 대표가 (#8 Q1 규칙으로 API 가 고른 가격). 원화를 크게, 원래 금액·소스·변형·기준일을 작게 쓴다 (#13 Q2).
const props = defineProps<{
  price: Price | null
  variant: string | null
  language: Language
}>()

const krw = computed(() => formatKrw(props.price?.krw))
const stale = computed(() => !!props.price && isStale(props.price.date))
</script>

<template>
  <div class="headline">
    <p class="label">
      대표가
    </p>
    <template v-if="price">
      <p class="main">
        {{ krw ?? formatPrice(price) }}
      </p>
      <p class="sub">
        <span v-if="krw">{{ formatPrice(price) }} · </span>{{ sourceLabel(price.source) }}<span v-if="variant"> · {{ variantLabel(variant) }}</span>
        · {{ formatShortDate(price.date) }} 기준
        <span v-if="!krw"> · 환율 정보 없음</span>
        <span
          v-if="stale"
          class="stale"
          :title="`가격 기준일이 ${STALE_DAYS}일 넘게 지났습니다`"
        >오래된 가격</span>
      </p>
    </template>
    <p
      v-else
      class="none"
    >
      {{ language === 'ko' ? '한글판 시세는 준비 중입니다' : '아직 시세가 없습니다' }}
    </p>
  </div>
</template>

<style scoped>
.headline {
  margin-top: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.headline p {
  margin: 0;
}

.label {
  font-size: var(--font-sm);
  color: var(--color-text-muted);
}

.main {
  font-size: var(--font-2xl);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  line-height: 1.2;
}

.sub {
  font-size: var(--font-sm);
  color: var(--color-text-muted);
  font-variant-numeric: tabular-nums;
}

.none {
  color: var(--color-text-muted);
}

.stale {
  display: inline-block;
  margin-left: var(--space-1);
  padding: 0 0.4em;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 0.75rem;
}
</style>
