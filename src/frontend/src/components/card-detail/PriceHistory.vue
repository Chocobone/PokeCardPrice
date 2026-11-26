<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import type { CardDetail, Period, PriceHistoryResponse, PriceSeries } from '@/api/types'
import StateView from '@/components/common/StateView.vue'
import { historyKey, useCardDetailStore } from '@/stores/cardDetail'
import { representativeVariant } from '@/utils/cardPrice'
import { addDays, type ChartSeries } from '@/utils/chart'
import { formatAmount, formatKrw, sourceLabel, variantLabel } from '@/utils/format'

import PriceChart from './PriceChart.vue'
import PriceSummary from './PriceSummary.vue'

// 시세 추이: 기간 탭, 변형 선택, 원화 차트(소스별 선), 기간 요약 (#13 Q3).
// 기간·변형 선택은 화면 상태라 컴포넌트에 둔다. 추이 응답은 스토어가 기간별로 보관한다.
const props = defineProps<{ card: CardDetail }>()
const store = useCardDetailStore()

const PERIODS: { value: Period; label: string; days: number }[] = [
  { value: '7d', label: '7일', days: 7 },
  { value: '30d', label: '30일', days: 30 },
  { value: '90d', label: '90일', days: 90 },
  { value: '1y', label: '1년', days: 365 },
]
const SOURCE_COLOR: Record<string, string> = {
  tcgcsv: 'var(--color-accent)',
  tcgdex_cardmarket: 'var(--color-series-2)',
}
const colorOf = (source: string, i: number) => SOURCE_COLOR[source] ?? (i ? 'var(--color-series-2)' : 'var(--color-accent)')

const period = ref<Period>('30d')
const selected = ref<string | null>(null)

const key = computed(() => historyKey(props.card.id, period.value))
const history = computed(() => store.histories[key.value])
const loading = computed(() => !!store.historyLoading[key.value])
const error = computed(() => store.historyErrors[key.value])
// 기간을 바꿔 새 추이를 받는 동안에는 이전 기간 추이를 흐리게 보여 준다 (화면이 뛰지 않게).
const lastShown = ref<PriceHistoryResponse | null>(null)
watch(history, (h) => {
  if (h) lastShown.value = h
})
const shown = computed(() => history.value ?? (loading.value ? lastShown.value : null))

watch(
  key,
  () => {
    void store.loadHistory(props.card.id, period.value)
  },
  { immediate: true },
)
// 다른 카드로 바뀌면 기간·변형 선택을 처음으로 돌린다.
watch(
  () => props.card.id,
  () => {
    period.value = '30d'
    selected.value = null
    lastShown.value = null
  },
)

/** 이 기간에 계열이 있는 변형 (API 순서 = 변형 id 순) */
const variants = computed(() => [...new Set((shown.value?.series ?? []).map((s) => s.variant))])
/** 고른 변형. 없으면 대표가 변형, 그것도 없으면 첫 변형 */
const variant = computed(() => {
  const vs = variants.value
  for (const v of [selected.value, representativeVariant(props.card)]) {
    if (v && vs.includes(v)) return v
  }
  return vs[0] ?? null
})
const series = computed<PriceSeries[]>(() => (shown.value?.series ?? []).filter((s) => s.variant === variant.value))
const colors = computed(() => Object.fromEntries(series.value.map((s, i) => [s.source, colorOf(s.source, i)])))

const chartSeries = computed<ChartSeries[]>(() =>
  series.value.map((s, i) => ({
    key: s.source,
    label: sourceLabel(s.source),
    color: colorOf(s.source, i),
    points: s.points.map((p) => ({
      date: p.date,
      krw: p.krw === null ? null : Number(p.krw),
      original: formatAmount(p.amount, s.currency),
    })),
  })),
)

const hasKrw = computed(() => chartSeries.value.some((s) => s.points.some((p) => p.krw !== null)))
const maxPoints = computed(() => Math.max(0, ...series.value.map((s) => s.points.length)))
// 가로축 끝: 요청 시작일 + 기간 (= 오늘 KST, API 의 from 규칙). 마지막 가격이 더 늦으면 그 날.
const chartTo = computed(() => {
  const h = shown.value
  if (!h) return ''
  const end = addDays(h.from, PERIODS.find((p) => p.value === h.period)?.days ?? 30)
  const last = series.value.flatMap((s) => s.points.map((p) => p.date)).sort().at(-1)
  return last && last > end ? last : end
})
const periodLabel = computed(() => PERIODS.find((p) => p.value === period.value)?.label ?? '')
const chartLabel = computed(
  () =>
    `${periodLabel.value} 원화 시세 추이, ${variant.value ? variantLabel(variant.value) : ''}. ` +
    series.value
      .map((s) => `${sourceLabel(s.source)} 현재 ${formatKrw(s.summary.current.krw) ?? formatAmount(s.summary.current.amount, s.currency)}`)
      .join(', '),
)

function retry() {
  void store.loadHistory(props.card.id, period.value, { force: true })
}
</script>

<template>
  <section class="price-history">
    <div class="toolbar">
      <h2>시세 추이</h2>
      <div
        class="periods"
        role="group"
        aria-label="기간"
      >
        <button
          v-for="p in PERIODS"
          :key="p.value"
          type="button"
          :aria-pressed="period === p.value"
          @click="period = p.value"
        >
          {{ p.label }}
        </button>
      </div>
      <label
        v-if="variants.length > 1"
        class="variant"
      >
        <span class="muted">변형</span>
        <select
          :value="variant ?? ''"
          @change="selected = ($event.target as HTMLSelectElement).value"
        >
          <option
            v-for="v in variants"
            :key="v"
            :value="v"
          >{{ variantLabel(v) }}</option>
        </select>
      </label>
      <span
        v-else-if="variant"
        class="variant muted"
      >변형: {{ variantLabel(variant) }}</span>
    </div>

    <StateView
      v-if="error"
      kind="error"
      :message="error.message"
      retryable
      @retry="retry"
    />
    <StateView
      v-else-if="!shown && loading"
      kind="loading"
      message="시세 추이를 불러오는 중…"
    />
    <template v-else-if="shown">
      <StateView
        v-if="!series.length"
        kind="empty"
        :message="`최근 ${periodLabel} 동안의 시세가 없습니다.`"
      />
      <div
        v-else
        class="body"
        :class="{ busy: loading }"
      >
        <div class="chart">
          <PriceChart
            v-if="hasKrw"
            :from="shown.from"
            :to="chartTo"
            :series="chartSeries"
            :label="chartLabel"
          />
          <p
            v-else
            class="muted notice"
          >
            환율 정보가 없어 원화 추이를 그릴 수 없습니다. 아래 요약은 소스 통화 기준입니다.
          </p>
          <p
            v-if="maxPoints <= 1"
            class="muted notice"
          >
            이력이 쌓이면 추이가 표시됩니다.
          </p>
        </div>
        <PriceSummary
          :series="series"
          :from="shown.from"
          :colors="colors"
        />
      </div>
    </template>
  </section>
</template>

<style scoped>
.toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-4);
  margin-bottom: var(--space-3);
}

.toolbar h2 {
  margin: 0;
  font-size: var(--font-lg);
}

.periods {
  display: inline-flex;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  overflow: hidden;
}

.periods button {
  padding: var(--space-1) var(--space-3);
  border: 0;
  border-left: 1px solid var(--color-border);
  background: var(--color-surface);
}

.periods button:first-child {
  border-left: 0;
}

.periods button[aria-pressed='true'] {
  background: var(--color-accent);
  color: var(--color-accent-contrast);
  font-weight: 600;
}

.variant {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-sm);
}

.variant select {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
}

.body {
  display: grid;
  gap: var(--space-4);
  transition: opacity 0.15s;
}

.body.busy {
  opacity: 0.6;
}

.chart {
  min-width: 0;
}

.notice {
  margin: var(--space-2) 0 0;
  font-size: var(--font-sm);
}

@media (min-width: 1024px) {
  .body {
    grid-template-columns: minmax(0, 2fr) minmax(0, 1fr);
    align-items: start;
  }

  .body :deep(.price-summary) {
    grid-template-columns: 1fr;
  }
}
</style>
