// 목업 API. 계약(mydocs/tech/api_contract.md)대로 검색·순위·카드 상세·시세 추이를 흉내 낸다.
// 카드 정보는 개발 DB 표본(data.ts), 가격·변동률은 카드 ID 로 만든 임의 값이다.
import { ApiError } from '../client'
import type {
  CardDetail,
  CardSummary,
  Currency,
  Language,
  Mover,
  MoversParams,
  MoversResponse,
  Page,
  Period,
  Price,
  PriceHistoryResponse,
  PricePoint,
  SearchParams,
} from '../types'
import { MOCK_CARDS, type MockCard } from './data'

const DELAY_MS = 300
/** 이 검색어로 검색하면 서버 오류를 흉내 낸다 (오류 화면 확인용). */
export const MOCK_ERROR_QUERY = '__error'
const AS_OF = '2026-11-16'
const PREV_DAYS: Record<MoversParams['period'], number> = { '7d': 7, '30d': 30 }

/** 목업 호출 수 (개발 확인용). */
export const mockStats = { search: 0, movers: 0, card: 0, prices: 0 }
// 개발 확인 도구(헤드리스 브라우저)가 모듈 경로와 상관없이 읽을 수 있게 전역에도 둔다.
Object.assign(globalThis, { __mockStats: mockStats })

/**
 * 개발용 상황 전환. 주소에 ?mock=empty 또는 ?mock=error 를 붙이면 모든 목업 응답이 빈 결과·서버 오류가 된다.
 * 화면의 빈 상태·오류 상태를 확인할 때 쓴다. 목업 코드라 운영 빌드에는 없다.
 */
function scenario(): 'empty' | 'error' | null {
  const v = new URLSearchParams(globalThis.location?.search ?? '').get('mock')
  return v === 'empty' || v === 'error' ? v : null
}

/** 같은 입력이면 같은 값을 내는 0~1 난수 (mulberry32). */
function rand(seed: number): number {
  let t = (seed + 0x6d2b79f5) | 0
  t = Math.imul(t ^ (t >>> 15), t | 1)
  t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
  return ((t ^ (t >>> 14)) >>> 0) / 4294967296
}

// 실제 API 와 같이 영어판은 TCGCSV(USD), 일본판은 Cardmarket(EUR), 한글판은 대표가 없음(#7)으로 흉내 낸다.
const CURRENCY: Record<Language, Currency> = { en: 'USD', ja: 'EUR', ko: 'KRW' }
const SOURCE: Record<Language, string> = { en: 'tcgcsv', ja: 'tcgdex_cardmarket', ko: 'mock_kr' }
const KRW_RATE: Record<Currency, number> = { USD: 1400, EUR: 1500, KRW: 1 }

function dateBefore(daysAgo: number): string {
  const d = new Date(`${AS_OF}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() - daysAgo)
  return d.toISOString().slice(0, 10)
}

function priceOf(card: MockCard, daysAgo: number, amount: number): Price {
  const currency = CURRENCY[card.language]
  const date = dateBefore(daysAgo)
  return {
    amount: currency === 'KRW' ? Math.round(amount).toString() : amount.toFixed(2),
    currency,
    source: SOURCE[card.language],
    date,
    krw: Math.round(amount * KRW_RATE[currency]).toString(),
    fxDate: currency === 'KRW' ? null : date,
    updatedAt: null,
  }
}

/** 카드의 현재 대표가. 한글판과 일부 카드는 시세가 없다 (null). */
function baseAmount(card: MockCard): number | null {
  if (card.language === 'ko' || rand(card.id) < 0.12) return null
  const usd = 0.25 + rand(card.id * 7) ** 3 * 400
  return { USD: usd, EUR: usd * 0.9, KRW: usd * 1400 }[CURRENCY[card.language]]
}

function withPrice(card: MockCard): CardSummary {
  const amount = baseAmount(card)
  return { ...card, price: amount === null ? null : priceOf(card, 0, amount) }
}

/** 지정한 시간만큼 기다린다. signal 이 취소되면 AbortError 로 끝난다. */
function delay(signal?: AbortSignal): Promise<void> {
  return new Promise((resolve, reject) => {
    const abort = () => {
      clearTimeout(timer)
      reject(new DOMException('aborted', 'AbortError'))
    }
    const timer = setTimeout(() => {
      signal?.removeEventListener('abort', abort)
      resolve()
    }, DELAY_MS)
    if (signal?.aborted) abort()
    else signal?.addEventListener('abort', abort, { once: true })
  })
}

const norm = (s: string) => s.normalize('NFKC').toLowerCase()
const trimZeros = (s: string) => s.replace(/^0+(?=\d)/, '')

/** 검색어 낱말 하나가 카드와 맞는지: 이름·한글 종 이름·세트 이름 부분 일치, 세트 ID·번호 일치. */
function matchToken(card: MockCard, token: string): boolean {
  return (
    norm(card.name).includes(token) ||
    (card.speciesNameKo !== null && norm(card.speciesNameKo).includes(token)) ||
    norm(card.set.name).includes(token) ||
    norm(card.set.tcgdexId) === token ||
    (/^\d+$/.test(token) && trimZeros(card.localId) === trimZeros(token))
  )
}

export async function searchCards(params: SearchParams, signal?: AbortSignal): Promise<Page<CardSummary>> {
  await delay(signal)
  mockStats.search++
  if (scenario() === 'error') throw new ApiError(500, 'internal', '목업 서버 오류')
  const q = params.q.trim()
  if (q === MOCK_ERROR_QUERY) throw new ApiError(500, 'internal', '목업 서버 오류')
  if (q.length === 0 || q.length > 100) throw new ApiError(400, 'invalid_query', '검색어는 1~100자여야 합니다')
  const page = Math.max(1, params.page ?? 1)
  const size = Math.min(60, Math.max(1, params.size ?? 24))
  const tokens = norm(q).split(/\s+/)
  const hits = MOCK_CARDS.filter(
    (c) => scenario() !== 'empty' && (!params.lang || c.language === params.lang) && tokens.every((t) => matchToken(c, t)),
  )
  return {
    items: hits.slice((page - 1) * size, page * size).map(withPrice),
    page,
    size,
    total: hits.length,
  }
}

export async function getMovers(params: MoversParams, signal?: AbortSignal): Promise<MoversResponse> {
  await delay(signal)
  mockStats.movers++
  if (scenario() === 'error') throw new ApiError(500, 'internal', '목업 서버 오류')
  if (scenario() === 'empty') return { period: params.period, direction: params.direction, asOf: null, items: [] }
  const limit = Math.min(50, Math.max(1, params.limit ?? 10))
  const days = PREV_DAYS[params.period]
  const movers: Mover[] = []
  for (const card of MOCK_CARDS) {
    const amount = baseAmount(card)
    if (amount === null) continue
    // 기간별로 다른 변동률 (-45% ~ +60%)
    const changeRate = Math.round((rand(card.id * 31 + days) * 1.05 - 0.45) * 1000) / 1000
    const previous = amount / (1 + changeRate)
    movers.push({
      card: withPrice(card),
      variant: card.language === 'en' ? 'holofoil' : 'normal',
      current: priceOf(card, 0, amount),
      previous: priceOf(card, days, previous),
      changeRate,
    })
  }
  const sign = params.direction === 'up' ? 1 : -1
  const items = movers
    .filter((m) => sign * m.changeRate > 0)
    .sort((a, b) => sign * (b.changeRate - a.changeRate))
    .slice(0, limit)
  return { period: params.period, direction: params.direction, asOf: AS_OF, items }
}

function findCard(id: number): MockCard {
  const card = MOCK_CARDS.find((c) => c.id === id)
  if (!card || scenario() === 'empty') throw new ApiError(404, 'not_found', '카드를 찾을 수 없습니다')
  return card
}

/**
 * 목업 가격 계열. 실제 API 처럼 변형 × 소스별로 날짜 가격을 만든다 (#13 차트 확인용).
 * - 영어판: 일반/TCGplayer(대표가), 일반/Cardmarket, 리버스 홀로/TCGplayer
 * - 일본판: 일반/Cardmarket(대표가), 리버스 홀로/Cardmarket
 * - 일반/Cardmarket 는 7일마다 하루씩 빠지고, 200일보다 오래된 날은 원화가 없다 (선 끊김 확인).
 * - 일부 카드(약 20%)는 대표가 계열이 내내 같은 값이다 (최소 = 최대 확인).
 * 가격은 기준일(AS_OF)에서 대표가와 같아지도록 만든다 (검색 목업의 대표가와 맞춤).
 */
interface SeriesSpec {
  variant: string
  source: string
  currency: Currency
  /** 대표가 대비 배율 */
  factor: number
  /** 이 일수보다 오래된 날은 원화 null */
  noFxBefore?: number
  /** 이 간격마다 하루를 뺀다 */
  skipEvery?: number
  flat?: boolean
}

function seriesSpecs(card: MockCard): SeriesSpec[] {
  if (baseAmount(card) === null) return []
  const flat = rand(card.id * 3) < 0.2
  if (card.language === 'en') {
    return [
      { variant: 'normal', source: 'tcgcsv', currency: 'USD', factor: 1, flat },
      { variant: 'normal', source: 'tcgdex_cardmarket', currency: 'EUR', factor: 0.8, noFxBefore: 200, skipEvery: 7 },
      { variant: 'reverse_holofoil', source: 'tcgcsv', currency: 'USD', factor: 2.4 },
    ]
  }
  return [
    { variant: 'normal', source: 'tcgdex_cardmarket', currency: 'EUR', factor: 1, flat },
    { variant: 'reverse_holofoil', source: 'tcgdex_cardmarket', currency: 'EUR', factor: 2.1 },
  ]
}

/** 기준일로부터 daysAgo 일 전의 가격. 기준일(0)에는 대표가 × 배율이다. 빠진 날은 null. */
function seriesPrice(card: MockCard, spec: SeriesSpec, daysAgo: number): Price | null {
  if (spec.skipEvery && daysAgo > 0 && daysAgo % spec.skipEvery === 0) return null
  const base = baseAmount(card)
  if (base === null) return null
  const seed = card.id * 31 + spec.factor * 1000
  const wave = (i: number) => 0.6 * Math.sin(i / 11 + seed) + 0.4 * rand(seed + i)
  const drift = (rand(seed) - 0.5) * 0.9
  const m = spec.flat ? 1 : Math.exp((drift * daysAgo) / 365 + 0.12 * (wave(daysAgo) - wave(0)))
  const amount = base * spec.factor * m
  const date = dateBefore(daysAgo)
  const hasFx = !(spec.noFxBefore && daysAgo > spec.noFxBefore)
  return {
    amount: amount.toFixed(2),
    currency: spec.currency,
    source: spec.source,
    date,
    krw: hasFx ? Math.round(amount * KRW_RATE[spec.currency]).toString() : null,
    fxDate: hasFx ? date : null,
    updatedAt: null,
  }
}

/** 카드 상세. 변형·소스는 seriesSpecs, 연결 카드는 없다. */
export async function getCard(id: number, signal?: AbortSignal): Promise<CardDetail> {
  await delay(signal)
  mockStats.card++
  if (scenario() === 'error') throw new ApiError(500, 'internal', '목업 서버 오류')
  const card = findCard(id)
  const summary = withPrice(card)
  const specs = seriesSpecs(card)
  const variants = [...new Set(['normal', ...specs.map((s) => s.variant)])].map((variant) => ({
    variant,
    prices: specs
      .filter((s) => s.variant === variant)
      .map((s) => seriesPrice(card, s, 0))
      .filter((p): p is Price => p !== null),
  }))
  return {
    ...summary,
    price: variants[0]?.prices[0] ?? null,
    set: { ...card.set, releaseDate: null, logoUrl: null, cardCount: null },
    rarity: null,
    dexIds: [],
    linked: [],
    variants,
  }
}

const PERIOD_DAYS: Record<Period, number> = { '7d': 7, '30d': 30, '90d': 90, '1y': 365 }

const avg = (xs: number[]) => xs.reduce((a, b) => a + b, 0) / xs.length

/** 시세 추이. 변형 × 소스별 날짜 가격과 요약 (계약의 요약 규칙대로). */
export async function getCardPrices(id: number, period: Period, signal?: AbortSignal): Promise<PriceHistoryResponse> {
  await delay(signal)
  mockStats.prices++
  if (scenario() === 'error') throw new ApiError(500, 'internal', '목업 서버 오류')
  const card = findCard(id)
  const days = PERIOD_DAYS[period]
  const res: PriceHistoryResponse = { cardId: id, period, from: dateBefore(days), series: [] }
  for (const spec of seriesSpecs(card)) {
    const prices: Price[] = []
    for (let i = days; i >= 0; i--) {
      const p = seriesPrice(card, spec, i)
      if (p) prices.push(p)
    }
    const firstPrice = prices[0]
    const lastPrice = prices.at(-1)
    if (!firstPrice || !lastPrice) continue
    const amounts = prices.map((p) => Number(p.amount))
    const first = Number(firstPrice.amount)
    const last = Number(lastPrice.amount)
    res.series.push({
      variant: spec.variant,
      source: spec.source,
      currency: spec.currency,
      points: prices.map((p): PricePoint => ({ date: p.date, amount: p.amount, krw: p.krw })),
      summary: {
        current: lastPrice,
        average: avg(amounts).toFixed(2),
        min: Math.min(...amounts).toFixed(2),
        max: Math.max(...amounts).toFixed(2),
        changeRate: prices.length > 1 && first > 0 ? Math.round(((last - first) / first) * 1e6) / 1e6 : null,
        from: firstPrice.date,
      },
    })
  }
  return res
}
