<script setup lang="ts">
import { computed, ref, watch } from 'vue'

import { dateTicks, linePath, nearestIndex, valueAxis, xOf, yOf, type ChartSeries } from '@/utils/chart'
import { formatKrw, formatShortDate } from '@/utils/format'

// 원화 추이 선 그래프 (#13 Q1·Q3). 패키지 없이 SVG 로 그린다.
// 선은 viewBox(0~1000) 를 늘려 그리고, 글자·점은 HTML 을 백분율 위치에 겹쳐 늘어나지 않게 한다.
// 점 정보: 마우스 이동·터치로 가장 가까운 날짜, 키보드는 ←/→ (Home/End), Esc 로 닫는다.
const props = defineProps<{
  from: string
  to: string
  series: ChartSeries[]
  /** 화면 읽기 프로그램용 요약 */
  label: string
}>()

const SIZE = 1000

const axis = computed(() =>
  valueAxis(props.series.flatMap((s) => s.points.flatMap((p) => (p.krw === null ? [] : [p.krw])))),
)
const xTicks = computed(() => dateTicks(props.from, props.to))
// 세로축 글자는 겹쳐 그리므로(absolute) 칸 너비를 가장 긴 글자에 맞춰 직접 정한다.
const yLabelWidth = computed(() => {
  const n = Math.max(0, ...(axis.value?.ticks ?? []).map((t) => (formatKrw(String(t)) ?? '').length))
  return `${n * 0.62 + 0.4}em`
})

const lines = computed(() => {
  const a = axis.value
  if (!a) return []
  return props.series.map((s) => {
    const { d, isolated } = linePath(
      s.points.map((p) => ({ x: xOf(p.date, props.from, props.to), y: p.krw === null ? null : yOf(p.krw, a) })),
      SIZE,
    )
    return { key: s.key, color: s.color, d, isolated }
  })
})

/** 모든 계열의 날짜 (점 정보 이동 단위) */
const dates = computed(() => [...new Set(props.series.flatMap((s) => s.points.map((p) => p.date)))].sort())

const active = ref<number | null>(null)
watch(dates, () => (active.value = null))

const activeDate = computed(() => (active.value === null ? null : (dates.value[active.value] ?? null)))
const activeX = computed(() => (activeDate.value ? xOf(activeDate.value, props.from, props.to) : 0))
const activeRows = computed(() => {
  const d = activeDate.value
  const a = axis.value
  if (!d || !a) return []
  return props.series.flatMap((s) => {
    const p = s.points.find((q) => q.date === d)
    if (!p) return []
    return [{ key: s.key, label: s.label, color: s.color, krw: p.krw, original: p.original, y: p.krw === null ? null : yOf(p.krw, a) }]
  })
})

function onPointer(e: PointerEvent) {
  const el = e.currentTarget as HTMLElement
  const r = el.getBoundingClientRect()
  if (!r.width || !dates.value.length) return
  active.value = nearestIndex(dates.value, (e.clientX - r.left) / r.width, props.from, props.to)
}

function onKey(e: KeyboardEvent) {
  const n = dates.value.length
  if (!n) return
  const cur = active.value
  let next: number | null = cur
  if (e.key === 'ArrowLeft') next = cur === null ? n - 1 : Math.max(0, cur - 1)
  else if (e.key === 'ArrowRight') next = cur === null ? n - 1 : Math.min(n - 1, cur + 1)
  else if (e.key === 'Home') next = 0
  else if (e.key === 'End') next = n - 1
  else if (e.key === 'Escape') next = null
  else return
  e.preventDefault()
  active.value = next
}

const pct = (v: number) => `${(v * 100).toFixed(3)}%`
</script>

<template>
  <div class="price-chart">
    <ul class="legend">
      <li
        v-for="s in series"
        :key="s.key"
      >
        <span
          class="swatch"
          :style="{ background: s.color }"
        />{{ s.label }}
      </li>
    </ul>
    <p
      class="readout"
      aria-live="polite"
    >
      <template v-if="activeDate">
        <strong>{{ formatShortDate(activeDate) }}</strong>
        <span
          v-for="r in activeRows"
          :key="`t-${r.key}`"
          class="row"
        >
          <span
            class="swatch"
            :style="{ background: r.color }"
          />{{ r.label }}
          <b>{{ formatKrw(r.krw === null ? null : String(r.krw)) ?? '환율 없음' }}</b>
          <small>{{ r.original }}</small>
        </span>
      </template>
      <span
        v-else
        class="hint"
      >차트를 가리키거나 누르면 날짜별 값이 보입니다</span>
    </p>
    <div
      v-if="axis"
      class="frame"
    >
      <div
        class="y-labels"
        aria-hidden="true"
        :style="{ width: yLabelWidth }"
      >
        <span
          v-for="t in axis.ticks"
          :key="t"
          :style="{ top: pct(yOf(t, axis)) }"
        >{{ formatKrw(String(t)) }}</span>
      </div>
      <div
        class="plot"
        tabindex="0"
        role="img"
        :aria-label="label"
        @pointermove="onPointer"
        @pointerdown="onPointer"
        @pointerleave="(e) => e.pointerType === 'mouse' && (active = null)"
        @keydown="onKey"
        @blur="active = null"
      >
        <svg
          :viewBox="`0 0 ${SIZE} ${SIZE}`"
          preserveAspectRatio="none"
          aria-hidden="true"
        >
          <line
            v-for="t in axis.ticks"
            :key="t"
            class="grid"
            x1="0"
            :x2="SIZE"
            :y1="yOf(t, axis) * SIZE"
            :y2="yOf(t, axis) * SIZE"
          />
          <path
            v-for="l in lines"
            :key="l.key"
            :d="l.d"
            :stroke="l.color"
            class="line"
          />
        </svg>
        <template
          v-for="l in lines"
          :key="`iso-${l.key}`"
        >
          <span
            v-for="(p, i) in l.isolated"
            :key="i"
            class="dot"
            :style="{ left: pct(p.x), top: pct(p.y ?? 0), background: l.color }"
          />
        </template>
        <template v-if="activeDate">
          <span
            class="cursor"
            :style="{ left: pct(activeX) }"
          />
          <template
            v-for="r in activeRows"
            :key="`a-${r.key}`"
          >
            <span
              v-if="r.y !== null"
              class="dot active"
              :style="{ left: pct(activeX), top: pct(r.y), background: r.color }"
            />
          </template>
        </template>
      </div>
      <div
        class="x-labels"
        aria-hidden="true"
      >
        <span
          v-for="t in xTicks"
          :key="t.date"
          :style="{ left: pct(xOf(t.date, from, to)) }"
        >{{ t.label }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.price-chart {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
}

.legend {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1) var(--space-4);
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: var(--font-sm);
}

.swatch {
  display: inline-block;
  width: 0.9em;
  height: 3px;
  margin-right: var(--space-1);
  vertical-align: middle;
  border-radius: 2px;
}

.frame {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  grid-template-rows: 200px auto;
  column-gap: var(--space-2);
  padding-top: var(--space-2);
}

.y-labels {
  position: relative;
  font-size: 0.75rem;
  color: var(--color-text-muted);
  font-variant-numeric: tabular-nums;
}

.y-labels span {
  position: absolute;
  right: 0;
  transform: translateY(-50%);
  white-space: nowrap;
}

.plot {
  position: relative;
  touch-action: pan-y;
  cursor: crosshair;
  border-radius: var(--radius-sm);
}

.plot svg {
  display: block;
  width: 100%;
  height: 100%;
  overflow: visible;
}

.grid {
  stroke: var(--color-border);
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}

.line {
  fill: none;
  stroke-width: 2;
  stroke-linejoin: round;
  stroke-linecap: round;
  vector-effect: non-scaling-stroke;
}

.dot {
  position: absolute;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  transform: translate(-50%, -50%);
  pointer-events: none;
}

.dot.active {
  width: 10px;
  height: 10px;
  border: 2px solid var(--color-surface);
}

.cursor {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 1px;
  background: var(--color-text-muted);
  pointer-events: none;
}

.readout {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-1) var(--space-4);
  min-height: 1.6em;
  margin: 0;
  font-size: var(--font-sm);
  font-variant-numeric: tabular-nums;
}

.readout .row {
  display: inline-flex;
  align-items: baseline;
  gap: var(--space-1);
}

.readout small,
.hint {
  color: var(--color-text-muted);
}

.hint {
  font-size: 0.75rem;
}

.x-labels {
  position: relative;
  grid-column: 2;
  height: 1.6em;
  font-size: 0.75rem;
  color: var(--color-text-muted);
}

.x-labels span {
  position: absolute;
  top: var(--space-1);
  transform: translateX(-50%);
  white-space: nowrap;
}

@media (min-width: 1024px) {
  .frame {
    grid-template-rows: 280px auto;
  }
}
</style>
