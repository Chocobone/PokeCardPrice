<script setup lang="ts">
import type { PriceSeries } from '@/api/types'
import {
  changeDirection,
  formatAmount,
  formatChangeRate,
  formatKrw,
  formatPrice,
  formatDate,
  formatShortDate,
  sourceLabel,
  todayKst,
} from '@/utils/format'

// 선택한 변형의 소스별 기간 요약. 평균·최저·최고는 API 가 소스 통화로만 주므로 원래 통화로 쓴다 (환산하지 않음).
defineProps<{
  series: PriceSeries[]
  /** 요청 시작일. 첫 가격이 이보다 늦으면 "이력이 짧음" */
  from: string
  colors: Record<string, string>
}>()

// 올해가 아니면 연도까지 쓴다 (1년 기간).
const thisYear = todayKst().slice(0, 4)
const since = (ymd: string) => (ymd.startsWith(thisYear) ? formatShortDate(ymd) : formatDate(ymd))
</script>

<template>
  <ul class="price-summary">
    <li
      v-for="s in series"
      :key="s.source"
    >
      <p class="head">
        <span
          class="swatch"
          :style="{ background: colors[s.source] }"
        />
        {{ sourceLabel(s.source) }}
      </p>
      <p class="current">
        {{ formatKrw(s.summary.current.krw) ?? formatPrice(s.summary.current) }}
        <small v-if="formatKrw(s.summary.current.krw)">{{ formatPrice(s.summary.current) }}</small>
      </p>
      <dl>
        <div>
          <dt>평균</dt>
          <dd>{{ formatAmount(s.summary.average, s.currency) }}</dd>
        </div>
        <div>
          <dt>최저</dt>
          <dd>{{ formatAmount(s.summary.min, s.currency) }}</dd>
        </div>
        <div>
          <dt>최고</dt>
          <dd>{{ formatAmount(s.summary.max, s.currency) }}</dd>
        </div>
        <div>
          <dt>변동률</dt>
          <dd
            v-if="s.summary.changeRate !== null"
            :class="changeDirection(s.summary.changeRate)"
          >
            {{ formatChangeRate(s.summary.changeRate) }}
          </dd>
          <dd
            v-else
            class="muted"
          >
            —
          </dd>
        </div>
      </dl>
      <p class="since muted">
        {{ since(s.summary.from) }}부터 {{ s.points.length }}일치<span v-if="s.summary.from > from"> · 이력이 짧음</span>
      </p>
    </li>
  </ul>
</template>

<style scoped>
.price-summary {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-3);
  margin: 0;
  padding: 0;
  list-style: none;
  font-variant-numeric: tabular-nums;
}

.price-summary li {
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
}

.price-summary p {
  margin: 0;
}

.head {
  font-weight: 600;
}

.swatch {
  display: inline-block;
  width: 0.9em;
  height: 3px;
  margin-right: var(--space-1);
  vertical-align: middle;
  border-radius: 2px;
}

.current {
  font-size: var(--font-lg);
  font-weight: 700;
}

.current small {
  margin-left: var(--space-1);
  font-size: var(--font-sm);
  font-weight: 400;
  color: var(--color-text-muted);
}

dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: var(--space-1) var(--space-3);
  margin: var(--space-2) 0;
  font-size: var(--font-sm);
}

dl div {
  display: flex;
  justify-content: space-between;
  gap: var(--space-2);
}

dt {
  color: var(--color-text-muted);
}

dd {
  margin: 0;
}

.up {
  color: var(--color-up);
}

.down {
  color: var(--color-down);
}

.since {
  font-size: 0.75rem;
}
</style>
