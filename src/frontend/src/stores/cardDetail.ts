// 카드 상세·시세 추이 상태. 카드는 id 별, 추이는 "id:기간" 별로 보관해 같은 화면에 돌아와도 다시 요청하지 않는다.
// 보관은 최근 카드 MAX_CARDS 장까지이고, 넘으면 가장 오래 쓰지 않은 카드의 상세·추이를 함께 지운다 (1년 추이 약 78 KB).
import { defineStore } from 'pinia'
import { ref } from 'vue'

import { getCard, getCardPrices } from '@/api/cards'
import { ApiError, errorMessage, isAbortError } from '@/api/client'
import type { CardDetail, Period, PriceHistoryResponse } from '@/api/types'

export const MAX_CARDS = 20

export type HistoryKey = `${number}:${Period}`
export const historyKey = (id: number, period: Period): HistoryKey => `${id}:${period}`

/** 불러오기 오류. notFound 는 없는 카드(404)이다. */
export interface LoadError {
  notFound: boolean
  message: string
}

function toLoadError(e: unknown): LoadError {
  return { notFound: e instanceof ApiError && e.status === 404, message: errorMessage(e) }
}

export const useCardDetailStore = defineStore('cardDetail', () => {
  const cards = ref<Record<number, CardDetail>>({})
  const cardLoading = ref<Record<number, boolean>>({})
  const cardErrors = ref<Record<number, LoadError>>({})
  const histories = ref<Partial<Record<HistoryKey, PriceHistoryResponse>>>({})
  const historyLoading = ref<Partial<Record<HistoryKey, boolean>>>({})
  const historyErrors = ref<Partial<Record<HistoryKey, LoadError>>>({})

  // 진행 중인 요청. 키별로 하나이고, 끝날 때 자기 요청일 때만 로딩 상태를 정리한다.
  const pending = new Map<string, AbortController>()
  // 최근에 쓴 카드 id (뒤가 최신)
  let recent: number[] = []

  function touch(id: number) {
    recent = [...recent.filter((x) => x !== id), id]
    while (recent.length > MAX_CARDS) {
      const old = recent.shift()
      if (old === undefined) break
      delete cards.value[old]
      delete cardErrors.value[old]
      for (const k of Object.keys(histories.value) as HistoryKey[]) {
        if (k.startsWith(`${old}:`)) delete histories.value[k]
      }
      for (const k of Object.keys(historyErrors.value) as HistoryKey[]) {
        if (k.startsWith(`${old}:`)) delete historyErrors.value[k]
      }
    }
  }

  /** id 가 아닌 카드의 진행 중인 요청을 모두 취소한다 (다른 카드로 이동할 때). */
  function cancelOthers(id: number) {
    for (const [k, c] of pending) {
      if (k !== `card:${id}` && !k.startsWith(`history:${id}:`)) {
        c.abort()
        pending.delete(k)
      }
    }
  }

  async function run<T>(
    key: string,
    fetcher: (signal: AbortSignal) => Promise<T>,
    done: (v: T) => void,
    fail: (e: LoadError) => void,
    setLoading: (v: boolean) => void,
  ) {
    const c = new AbortController()
    pending.set(key, c)
    setLoading(true)
    try {
      done(await fetcher(c.signal))
    } catch (e) {
      if (!isAbortError(e) && pending.get(key) === c) fail(toLoadError(e))
    } finally {
      // 취소되었거나 같은 키의 새 요청이 있으면 그쪽이 로딩 상태를 맡는다.
      if (pending.get(key) === c) {
        pending.delete(key)
        setLoading(false)
      } else if (c.signal.aborted && !pending.has(key)) {
        setLoading(false)
      }
    }
  }

  /** 카드 상세를 불러온다. 불러오는 중이거나, force 가 아닌데 결과가 있으면 건너뛴다. */
  async function loadCard(id: number, opts: { force?: boolean } = {}) {
    cancelOthers(id)
    touch(id)
    if (cardLoading.value[id] || (!opts.force && cards.value[id])) return
    delete cardErrors.value[id]
    await run(
      `card:${id}`,
      (s) => getCard(id, s),
      (v) => (cards.value[id] = v),
      (e) => (cardErrors.value[id] = e),
      (v) => (cardLoading.value[id] = v),
    )
  }

  /** 기간 추이를 불러온다. 규칙은 loadCard 와 같다. */
  async function loadHistory(id: number, period: Period, opts: { force?: boolean } = {}) {
    const key = historyKey(id, period)
    if (historyLoading.value[key] || (!opts.force && histories.value[key])) return
    delete historyErrors.value[key]
    await run(
      `history:${key}`,
      (s) => getCardPrices(id, period, s),
      (v) => (histories.value[key] = v),
      (e) => (historyErrors.value[key] = e),
      (v) => (historyLoading.value[key] = v),
    )
  }

  return { cards, cardLoading, cardErrors, histories, historyLoading, historyErrors, loadCard, loadHistory, cancelOthers }
})
