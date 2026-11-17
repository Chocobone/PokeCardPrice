// 목업 API. 계약(mydocs/tech/api_contract.md)대로 검색·순위를 흉내 낸다.
// 카드 정보는 개발 DB 표본(data.ts), 가격·변동률은 카드 ID 로 만든 임의 값이다.
import { ApiError } from '../client'
import type {
  CardSummary,
  Currency,
  Language,
  Mover,
  MoversParams,
  MoversResponse,
  Page,
  Price,
  SearchParams,
} from '../types'
import { MOCK_CARDS, type MockCard } from './data'

const DELAY_MS = 300
/** 이 검색어로 검색하면 서버 오류를 흉내 낸다 (오류 화면 확인용). */
export const MOCK_ERROR_QUERY = '__error'
const AS_OF = '2026-11-16'
const PREV_DAYS: Record<MoversParams['period'], number> = { '7d': 7, '30d': 30 }

/** 목업 호출 수 (개발 확인용). */
export const mockStats = { search: 0, movers: 0 }
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

const CURRENCY: Record<Language, Currency> = { en: 'USD', ja: 'JPY', ko: 'KRW' }
const SOURCE: Record<Language, string> = { en: 'tcgcsv', ja: 'mock_jp', ko: 'mock_kr' }

function priceOf(card: MockCard, daysAgo: number, amount: number): Price {
  const currency = CURRENCY[card.language]
  const d = new Date(`${AS_OF}T00:00:00Z`)
  d.setUTCDate(d.getUTCDate() - daysAgo)
  return {
    amount: currency === 'USD' ? amount.toFixed(2) : Math.round(amount).toString(),
    currency,
    source: SOURCE[card.language],
    date: d.toISOString().slice(0, 10),
  }
}

/** 카드의 현재 대표가. 일부 카드는 시세가 없다 (null). */
function baseAmount(card: MockCard): number | null {
  if (rand(card.id) < 0.12) return null
  const usd = 0.25 + rand(card.id * 7) ** 3 * 400
  return { USD: usd, JPY: usd * 150, KRW: usd * 1400 }[CURRENCY[card.language]]
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
