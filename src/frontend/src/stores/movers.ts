// 급등·급락 순위 상태. 기간·방향별로 결과를 보관해 홈에 다시 와도 다시 요청하지 않는다.
import { defineStore } from 'pinia'
import { ref } from 'vue'

import { getMovers } from '@/api/cards'
import { errorMessage, isAbortError } from '@/api/client'
import type { MoverDirection, MoverPeriod, MoversResponse } from '@/api/types'

export type MoverKey = `${MoverPeriod}:${MoverDirection}`
export const MOVERS_LIMIT = 10

export const moverKey = (period: MoverPeriod, direction: MoverDirection): MoverKey => `${period}:${direction}`

export const useMoversStore = defineStore('movers', () => {
  const results = ref<Partial<Record<MoverKey, MoversResponse>>>({})
  const loading = ref<Partial<Record<MoverKey, boolean>>>({})
  const errors = ref<Partial<Record<MoverKey, string>>>({})

  /** 순위를 불러온다. 불러오는 중이거나, force 가 아닌데 이미 결과가 있으면 건너뛴다. */
  async function load(period: MoverPeriod, direction: MoverDirection, opts: { force?: boolean } = {}) {
    const key = moverKey(period, direction)
    if (loading.value[key] || (!opts.force && results.value[key])) return
    loading.value[key] = true
    delete errors.value[key]
    try {
      results.value[key] = await getMovers({ period, direction, limit: MOVERS_LIMIT })
    } catch (e) {
      if (!isAbortError(e)) errors.value[key] = errorMessage(e)
    } finally {
      loading.value[key] = false
    }
  }

  return { results, loading, errors, load }
})
