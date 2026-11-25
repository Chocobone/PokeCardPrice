// 카드 검색·순위·상세·시세 추이 API. 화면과 스토어는 이 함수만 쓰고, 목업 여부를 모른다.
// VITE_API_MOCK=true 이면 내장 목업을 쓴다 (백엔드 없이 화면을 볼 때). 운영 빌드에서는 목업 코드가 번들에서 빠진다.
import { apiGet } from './client'
import type {
  CardDetail,
  CardSummary,
  MoversParams,
  MoversResponse,
  Page,
  Period,
  PriceHistoryResponse,
  SearchParams,
} from './types'

export const USE_MOCK = import.meta.env.VITE_API_MOCK === 'true'

export async function searchCards(params: SearchParams, signal?: AbortSignal): Promise<Page<CardSummary>> {
  if (USE_MOCK) {
    const mock = await import('./mock/handlers')
    return mock.searchCards(params, signal)
  }
  return apiGet<Page<CardSummary>>('/cards', {
    params: { q: params.q, lang: params.lang, page: params.page, size: params.size },
    signal,
  })
}

export async function getMovers(params: MoversParams, signal?: AbortSignal): Promise<MoversResponse> {
  if (USE_MOCK) {
    const mock = await import('./mock/handlers')
    return mock.getMovers(params, signal)
  }
  return apiGet<MoversResponse>('/rankings/movers', {
    params: { period: params.period, direction: params.direction, limit: params.limit },
    signal,
  })
}

export async function getCard(id: number, signal?: AbortSignal): Promise<CardDetail> {
  if (USE_MOCK) {
    const mock = await import('./mock/handlers')
    return mock.getCard(id, signal)
  }
  return apiGet<CardDetail>(`/cards/${id}`, { signal })
}

export async function getCardPrices(id: number, period: Period, signal?: AbortSignal): Promise<PriceHistoryResponse> {
  if (USE_MOCK) {
    const mock = await import('./mock/handlers')
    return mock.getCardPrices(id, period, signal)
  }
  return apiGet<PriceHistoryResponse>(`/cards/${id}/prices`, { params: { period }, signal })
}
