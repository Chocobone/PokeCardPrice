// 카드 검색 상태. 질의(검색어·언어·쪽)는 URL 이 기준이고, 화면이 URL 에서 만든 질의로 run() 을 부른다.
import { defineStore } from 'pinia'
import { ref } from 'vue'

import { searchCards } from '@/api/cards'
import { errorMessage, isAbortError } from '@/api/client'
import type { CardSummary, Language, Page } from '@/api/types'

export const SEARCH_PAGE_SIZE = 24

export interface SearchQuery {
  q: string
  lang?: Language
  page: number
}

const keyOf = (q: SearchQuery) => JSON.stringify([q.q, q.lang ?? '', q.page])

export const useSearchStore = defineStore('search', () => {
  const query = ref<SearchQuery | null>(null)
  const result = ref<Page<CardSummary> | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  let controller: AbortController | null = null
  let seq = 0

  /**
   * 검색한다. 같은 질의의 결과가 있거나 불러오는 중이면 건너뛴다 (force 면 다시 부른다).
   * 새 검색을 시작하면 이전 요청은 취소해, 빠르게 검색어를 바꿔도 마지막 결과만 남는다.
   */
  async function run(next: SearchQuery, opts: { force?: boolean } = {}) {
    if (!opts.force && query.value && keyOf(query.value) === keyOf(next) && (loading.value || result.value)) return
    controller?.abort()
    controller = new AbortController()
    const mine = ++seq
    query.value = next
    loading.value = true
    error.value = null
    try {
      const page = await searchCards({ ...next, size: SEARCH_PAGE_SIZE }, controller.signal)
      if (mine === seq) result.value = page
    } catch (e) {
      if (isAbortError(e) || mine !== seq) return
      result.value = null
      error.value = errorMessage(e)
    } finally {
      if (mine === seq) loading.value = false
    }
  }

  return { query, result, loading, error, run }
})
