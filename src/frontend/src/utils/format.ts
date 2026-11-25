// 화면 표시용 형식 함수. 금액은 API 에서 문자열로 오므로 표시할 때만 숫자로 바꾼다.
import type { Currency, Price } from '@/api/types'

const FRACTION: Record<Currency, number> = { USD: 2, EUR: 2, KRW: 0 }
const moneyFormats = new Map<Currency, Intl.NumberFormat>()

function moneyFormat(currency: Currency): Intl.NumberFormat {
  let f = moneyFormats.get(currency)
  if (!f) {
    f = new Intl.NumberFormat('ko-KR', {
      style: 'currency',
      currency,
      currencyDisplay: 'narrowSymbol',
      minimumFractionDigits: FRACTION[currency],
      maximumFractionDigits: FRACTION[currency],
    })
    moneyFormats.set(currency, f)
  }
  return f
}

/** "$135.57", "€12.30", "₩12,000" (소스 통화). 가격이 없으면 "시세 없음". */
export function formatPrice(price: Price | null | undefined): string {
  if (!price) return '시세 없음'
  const n = Number(price.amount)
  return Number.isFinite(n) ? moneyFormat(price.currency).format(n) : '시세 없음'
}

const percent = new Intl.NumberFormat('ko-KR', {
  style: 'percent',
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
  signDisplay: 'exceptZero',
})

/** 0.125 → "+12.5%", -0.3 → "-30.0%" */
export function formatChangeRate(rate: number): string {
  return percent.format(rate)
}

/** 변동 방향. 화면 색(상승 빨강·하락 파랑)과 기호에 쓴다. */
export function changeDirection(rate: number): 'up' | 'down' | 'flat' {
  if (rate > 0) return 'up'
  if (rate < 0) return 'down'
  return 'flat'
}

const dateFormat = new Intl.DateTimeFormat('ko-KR', { year: 'numeric', month: 'long', day: 'numeric', timeZone: 'UTC' })

/** "2026-11-16" → "2026년 11월 16일" */
export function formatDate(ymd: string): string {
  const d = new Date(`${ymd}T00:00:00Z`)
  return Number.isNaN(d.getTime()) ? ymd : dateFormat.format(d)
}
