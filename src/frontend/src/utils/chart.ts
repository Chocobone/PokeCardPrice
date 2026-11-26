// 추이 차트 계산 (#13). 좌표는 0~1 비율로 내고, 화면은 SVG viewBox·CSS 백분율로 그린다.
import { daysBetween } from './format'

export interface Axis {
  lo: number
  hi: number
  ticks: number[]
}

/** 1·2·5 × 10ⁿ 가운데 raw 이상인 가장 작은 값. */
function niceStep(raw: number): number {
  const exp = Math.floor(Math.log10(raw))
  const base = 10 ** exp
  const f = raw / base
  return (f <= 1 ? 1 : f <= 2 ? 2 : f <= 5 ? 5 : 10) * base
}

/**
 * 세로축 범위와 눈금. 값 범위에 위아래 8% 여백을 두고 보기 좋은 눈금(4~5개)에 맞춘다.
 * 최소 = 최대면 ±5%(0이면 0~1)로 넓힌다. 음수는 없으므로 아래쪽은 0에서 자른다.
 */
export function valueAxis(values: number[], count = 5): Axis | null {
  if (!values.length) return null
  let min = Math.min(...values)
  let max = Math.max(...values)
  if (min === max) {
    const d = min === 0 ? 1 : Math.abs(min) * 0.05
    min -= d
    max += d
  }
  const pad = (max - min) * 0.08
  min = Math.max(0, min - pad)
  max += pad
  const step = niceStep((max - min) / (count - 1))
  const lo = Math.floor(min / step) * step
  const hi = Math.ceil(max / step) * step
  const ticks: number[] = []
  for (let v = lo; v <= hi + step / 2; v += step) ticks.push(Math.round(v / step) * step)
  return { lo, hi, ticks }
}

/** "YYYY-MM-DD" 에 days 일을 더한다. */
export function addDays(ymd: string, days: number): string {
  const d = new Date(`${ymd}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() + days)
  return d.toISOString().slice(0, 10)
}

export interface DateTick {
  date: string
  label: string
}

/**
 * 가로축 눈금. 90일 이하는 기간을 고르게 나눈 날짜(최대 count 개, "11/3"),
 * 그보다 길면 매월 1일("3월", 1월은 "2026년 1월")을 쓰고 많으면 건너뛴다.
 */
export function dateTicks(from: string, to: string, count = 5): DateTick[] {
  const total = daysBetween(from, to)
  if (total <= 0) return [{ date: from, label: shortLabel(from) }]
  if (total <= 90) {
    const step = Math.max(1, Math.round(total / (count - 1)))
    const out: DateTick[] = []
    for (let i = 0; i <= total; i += step) out.push({ date: addDays(from, i), label: shortLabel(addDays(from, i)) })
    return out
  }
  const months: string[] = []
  const [y0, m0] = from.split('-').map(Number) as [number, number]
  for (let y = y0, m = m0 + 1; ; m++) {
    if (m > 12) {
      m = 1
      y++
    }
    const d = `${y}-${String(m).padStart(2, '0')}-01`
    if (d > to) break
    months.push(d)
  }
  const every = Math.ceil(months.length / count)
  return months
    .filter((_, i) => i % every === 0)
    .map((d) => ({ date: d, label: d.endsWith('-01-01') ? `${d.slice(0, 4)}년 1월` : `${Number(d.slice(5, 7))}월` }))
}

function shortLabel(ymd: string): string {
  return `${Number(ymd.slice(5, 7))}/${Number(ymd.slice(8, 10))}`
}

/** 날짜의 가로 위치 (0~1). */
export function xOf(date: string, from: string, to: string): number {
  const total = daysBetween(from, to)
  return total <= 0 ? 0.5 : daysBetween(from, date) / total
}

/** 값의 세로 위치 (0 = 위, 1 = 아래). */
export function yOf(value: number, axis: Axis): number {
  return axis.hi === axis.lo ? 0.5 : 1 - (value - axis.lo) / (axis.hi - axis.lo)
}

export interface PlotPoint {
  x: number
  /** null 이면 값이 없는 점 (선을 끊는다) */
  y: number | null
}

/**
 * SVG path (viewBox 0~size). y 가 null 인 점에서 선을 끊는다.
 * 앞뒤가 끊겨 혼자 남은 점은 path 로 보이지 않으므로 isolated 로 따로 돌려준다.
 */
export function linePath(points: PlotPoint[], size = 1000): { d: string; isolated: PlotPoint[] } {
  const segments: PlotPoint[][] = []
  let cur: PlotPoint[] = []
  for (const p of points) {
    if (p.y === null) {
      if (cur.length) segments.push(cur)
      cur = []
    } else cur.push(p)
  }
  if (cur.length) segments.push(cur)
  const fmt = (n: number) => Math.round(n * size * 10) / 10
  const d = segments
    .filter((s) => s.length > 1)
    .map((s) => s.map((p, i) => `${i ? 'L' : 'M'}${fmt(p.x)} ${fmt(p.y ?? 0)}`).join(''))
    .join('')
  return { d, isolated: segments.filter((s) => s.length === 1).flat() }
}

/** 정렬된 날짜 목록에서 비율 x 에 가장 가까운 날짜의 위치. */
export function nearestIndex(dates: string[], x: number, from: string, to: string): number {
  let best = 0
  let bestD = Infinity
  dates.forEach((d, i) => {
    const dist = Math.abs(xOf(d, from, to) - x)
    if (dist < bestD) {
      bestD = dist
      best = i
    }
  })
  return best
}

/** 차트에 그릴 계열 하나 (소스 하나). */
export interface ChartSeries {
  key: string
  label: string
  /** CSS 색 (변수 가능) */
  color: string
  /** 날짜 오름차순. krw 가 null 이면 원화가 없는 날이다. original 은 소스 통화 표시 문자열 */
  points: { date: string; krw: number | null; original: string }[]
}
