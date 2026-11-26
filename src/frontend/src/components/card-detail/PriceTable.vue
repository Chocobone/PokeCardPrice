<script setup lang="ts">
import { computed } from 'vue'

import type { Price, VariantPrices } from '@/api/types'
import {
  formatKrw,
  formatPrice,
  formatShortDate,
  isStale,
  SOURCE_ATTRIBUTION,
  sourceLabel,
  STALE_DAYS,
  todayKst,
  variantLabel,
} from '@/utils/format'

// 변형 × 소스별 최신가 표. 소스마다 붙는 변형과 가격 차이가 크므로 섞지 않고 줄마다 보여 준다.
// 모바일(640px 미만)에서는 줄을 카드형 목록으로 바꾼다.
const props = defineProps<{
  variants: VariantPrices[]
  /** 대표가 (해당 줄을 강조한다) */
  representative: Price | null
}>()

interface Row {
  key: string
  variant: string
  /** 변형의 첫 줄인지 (데스크톱에서 변형 이름을 이 줄에만 보인다) */
  first: boolean
  price: Price | null
}

const rows = computed<Row[]>(() =>
  props.variants.flatMap((v): Row[] =>
    v.prices.length
      ? v.prices.map((p, i) => ({ key: `${v.variant}/${p.source}`, variant: v.variant, first: i === 0, price: p }))
      : [{ key: `${v.variant}/-`, variant: v.variant, first: true, price: null }],
  ),
)

const today = todayKst()
const sources = computed(() => [...new Set(props.variants.flatMap((v) => v.prices.map((p) => p.source)))])

function isRep(p: Price | null): boolean {
  const r = props.representative
  return !!p && !!r && p.source === r.source && p.date === r.date && p.amount === r.amount
}
</script>

<template>
  <section class="price-table">
    <h2>가격 비교</h2>
    <table>
      <thead>
        <tr>
          <th scope="col">
            변형
          </th>
          <th scope="col">
            소스
          </th>
          <th
            scope="col"
            class="num"
          >
            원화
          </th>
          <th
            scope="col"
            class="num"
          >
            원래 금액
          </th>
          <th scope="col">
            기준일
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="r in rows"
          :key="r.key"
          :class="{ rep: isRep(r.price), first: r.first }"
        >
          <th
            scope="row"
            class="variant"
            :class="{ repeat: !r.first }"
          >
            {{ variantLabel(r.variant) }}
            <span
              v-if="isRep(r.price)"
              class="tag"
            >대표</span>
          </th>
          <template v-if="r.price">
            <td data-label="소스">
              {{ sourceLabel(r.price.source) }}
            </td>
            <td
              data-label="원화"
              class="num krw"
            >
              <template v-if="formatKrw(r.price.krw)">
                {{ formatKrw(r.price.krw) }}
                <small
                  v-if="r.price.fxDate && r.price.fxDate !== r.price.date"
                  class="muted"
                >환율 {{ formatShortDate(r.price.fxDate) }}</small>
              </template>
              <span
                v-else
                class="muted"
              >환율 없음</span>
            </td>
            <td
              data-label="원래 금액"
              class="num"
            >
              {{ formatPrice(r.price) }}
            </td>
            <td data-label="기준일">
              <span>
                {{ formatShortDate(r.price.date) }}
                <span
                  v-if="isStale(r.price.date, today)"
                  class="stale"
                  :title="`가격 기준일이 ${STALE_DAYS}일 넘게 지났습니다`"
                >오래된 가격</span>
              </span>
            </td>
          </template>
          <td
            v-else
            colspan="4"
            class="muted"
          >
            시세 없음
          </td>
        </tr>
      </tbody>
    </table>
    <p
      v-if="sources.length"
      class="attribution muted"
    >
      가격 출처: {{ sources.map((s) => SOURCE_ATTRIBUTION[s] ?? sourceLabel(s)).join(' · ') }}.
      원화는 가격 기준일 이전 가장 가까운 날의 환율(Frankfurter)로 환산한 참고값입니다.
    </p>
  </section>
</template>

<style scoped>
.price-table h2 {
  font-size: var(--font-lg);
}

table {
  width: 100%;
  border-collapse: collapse;
  font-variant-numeric: tabular-nums;
}

th,
td {
  padding: var(--space-2) var(--space-3);
  text-align: left;
  vertical-align: top;
}

thead th {
  font-size: var(--font-sm);
  font-weight: 600;
  color: var(--color-text-muted);
  border-bottom: 1px solid var(--color-border);
}

tbody tr.first:not(:first-child) {
  border-top: 1px solid var(--color-border);
}

tbody th {
  font-weight: 600;
}

.variant.repeat {
  color: transparent;
}

.num {
  text-align: right;
}

.krw {
  font-weight: 700;
}

.krw small {
  display: block;
  font-weight: 400;
}

.rep {
  background: var(--color-surface-muted);
}

.tag,
.stale {
  display: inline-block;
  margin-left: var(--space-1);
  padding: 0 0.4em;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 400;
  color: var(--color-text-muted);
  white-space: nowrap;
}

.tag {
  border-color: var(--color-accent);
  color: var(--color-accent);
}

.attribution {
  margin-top: var(--space-3);
  font-size: var(--font-sm);
}

/* 모바일: 줄마다 2열 카드형 (왼쪽 변형·소스·기준일, 오른쪽 원화·원래 금액) */
@media (max-width: 639px) {
  thead {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }

  table,
  tbody {
    display: block;
  }

  tbody tr {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    column-gap: var(--space-3);
    padding: var(--space-2) var(--space-3);
    border-top: 1px solid var(--color-border);
  }

  tbody tr:first-child {
    border-top: 0;
  }

  th,
  td {
    padding: 0;
  }

  .variant {
    grid-column: 1;
    grid-row: 1;
  }

  .variant.repeat {
    color: inherit;
  }

  td[data-label='소스'] {
    grid-column: 1;
    grid-row: 2;
    color: var(--color-text-muted);
    font-size: var(--font-sm);
  }

  td[data-label='원화'] {
    grid-column: 2;
    grid-row: 1;
  }

  td[data-label='원래 금액'] {
    grid-column: 2;
    grid-row: 2;
    color: var(--color-text-muted);
    font-size: var(--font-sm);
  }

  td[data-label='기준일'] {
    grid-column: 1 / -1;
    grid-row: 3;
    color: var(--color-text-muted);
    font-size: var(--font-sm);
  }

  td[colspan] {
    grid-column: 2;
    grid-row: 1;
  }

  .krw small {
    display: inline;
    margin-right: var(--space-1);
  }
}
</style>
