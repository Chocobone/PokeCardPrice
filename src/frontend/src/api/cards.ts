// 카드 검색·순위 API. 화면과 스토어는 이 함수만 쓰고, 목업 여부를 모른다.
// VITE_API_MOCK=true 이면 내장 목업을 쓴다 (#8 API 완성 전). 운영 빌드에서는 목업 코드가 번들에서 빠진다.
import { apiGet } from './client'
import type { CardSummary, MoversParams, MoversResponse, Page, SearchParams } from './types'

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
