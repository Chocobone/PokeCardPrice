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

const krwFormat = new Intl.NumberFormat('ko-KR', { style: 'currency', currency: 'KRW', maximumFractionDigits: 0 })

/** 원화 환산값("16441") → "₩16,441". 없으면 null. */
export function formatKrw(krw: string | null | undefined): string | null {
  if (!krw) return null
  const n = Number(krw)
  return Number.isFinite(n) ? krwFormat.format(n) : null
}

/** 소스 통화 금액 + 통화 → "$12.30" (요약 평균·최저·최고처럼 Price 가 아닌 금액용). */
export function formatAmount(amount: string, currency: Currency): string {
  const n = Number(amount)
  return Number.isFinite(n) ? moneyFormat(currency).format(n) : amount
}

const shortDateFormat = new Intl.DateTimeFormat('ko-KR', { month: 'long', day: 'numeric', timeZone: 'UTC' })

/** "2026-11-23" → "11월 23일" */
export function formatShortDate(ymd: string): string {
  const d = new Date(`${ymd}T00:00:00Z`)
  return Number.isNaN(d.getTime()) ? ymd : shortDateFormat.format(d)
}

const kstDate = new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Seoul', year: 'numeric', month: '2-digit', day: '2-digit' })

/** 오늘 날짜 (KST, "YYYY-MM-DD"). */
export function todayKst(now: Date = new Date()): string {
  return kstDate.format(now)
}

/** 두 날짜("YYYY-MM-DD") 사이 일수 (to - from). */
export function daysBetween(from: string, to: string): number {
  return Math.round((Date.parse(`${to}T00:00:00Z`) - Date.parse(`${from}T00:00:00Z`)) / 86_400_000)
}

/** 오래된 가격 기준: 가격 기준일이 오늘(KST)보다 이 일수를 넘게 이전이면 오래된 가격이다 (#13 Q4). */
export const STALE_DAYS = 3

export function isStale(date: string, today: string = todayKst()): boolean {
  return daysBetween(date, today) > STALE_DAYS
}

/** 가격 소스 표시 이름. */
export const SOURCE_LABEL: Record<string, string> = {
  tcgcsv: 'TCGplayer',
  tcgdex_cardmarket: 'Cardmarket',
}

/**
 * 가격 소스 출처 문구 (#13 Q5). DB price_sources.attribution 과 같은 문구이다.
 * API 응답에 없는 값이라 상수로 두며, 마이그레이션 문구가 바뀌면 함께 고친다.
 */
export const SOURCE_ATTRIBUTION: Record<string, string> = {
  tcgcsv: 'TCGplayer 가격 (TCGCSV 경유)',
  tcgdex_cardmarket: 'Cardmarket 가격 (TCGdex 경유)',
}

export function sourceLabel(code: string): string {
  return SOURCE_LABEL[code] ?? code
}

const VARIANT_LABEL: Record<string, string> = {
  normal: '일반',
  holofoil: '홀로',
  reverse_holofoil: '리버스 홀로',
  first_edition: '1st 에디션',
  first_edition_holofoil: '1st 에디션 홀로',
  unlimited: '언리미티드',
  unlimited_holofoil: '언리미티드 홀로',
}

/** 변형 표시 이름. 모르는 변형은 코드 그대로. */
export function variantLabel(code: string): string {
  return VARIANT_LABEL[code] ?? code
}
