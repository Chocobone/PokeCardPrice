// fetch 래퍼. 모든 API 호출은 이 함수를 거친다.
// 오류는 ApiError 로 통일하고, 호출 측 취소(AbortError)는 그대로 던져 화면이 무시할 수 있게 한다.
import type { ErrorBody } from './types'

/** 프론트엔드 기준 주소. Vite 프록시(개발)·배포 서버가 /api 를 떼어 백엔드로 넘긴다. */
export const API_BASE = '/api'
const DEFAULT_TIMEOUT_MS = 10_000

/**
 * API 오류. status 0 은 응답을 받지 못한 경우이다 (code: network, timeout).
 */
export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

export type QueryValue = string | number | undefined

export interface RequestOptions {
  params?: Record<string, QueryValue>
  signal?: AbortSignal
  timeoutMs?: number
}

/** 호출 측이 요청을 취소했는지. 취소된 요청의 오류는 화면에 보이지 않는다. */
export function isAbortError(e: unknown): boolean {
  return e instanceof DOMException && e.name === 'AbortError'
}

/** params 에서 undefined·빈 문자열을 빼고 쿼리 문자열을 만든다. */
export function buildQuery(params: Record<string, QueryValue> = {}): string {
  const sp = new URLSearchParams()
  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === '') continue
    sp.set(k, String(v))
  }
  const s = sp.toString()
  return s ? `?${s}` : ''
}

export async function apiGet<T>(path: string, opts: RequestOptions = {}): Promise<T> {
  const timeout = AbortSignal.timeout(opts.timeoutMs ?? DEFAULT_TIMEOUT_MS)
  const signal = opts.signal ? AbortSignal.any([opts.signal, timeout]) : timeout

  let res: Response
  try {
    res = await fetch(API_BASE + path + buildQuery(opts.params), {
      headers: { Accept: 'application/json' },
      signal,
    })
  } catch (e) {
    if (e instanceof DOMException && e.name === 'TimeoutError') {
      throw new ApiError(0, 'timeout', '응답 시간이 초과되었습니다')
    }
    if (isAbortError(e)) throw e
    throw new ApiError(0, 'network', '서버에 연결할 수 없습니다')
  }

  if (!res.ok) {
    let code = `http_${res.status}`
    let message = `요청이 실패했습니다 (HTTP ${res.status})`
    try {
      const body = (await res.json()) as Partial<ErrorBody>
      if (body.error?.code) code = body.error.code
      if (body.error?.message) message = body.error.message
    } catch {
      // 본문이 JSON 이 아니면 기본 문구를 쓴다
    }
    throw new ApiError(res.status, code, message)
  }
  return (await res.json()) as T
}

/** 화면에 보여 줄 오류 문구. */
export function errorMessage(e: unknown): string {
  if (e instanceof ApiError) {
    if (e.status >= 500) return '서버에 문제가 생겼습니다. 잠시 후 다시 시도해 주세요.'
    return e.message
  }
  return '알 수 없는 오류가 발생했습니다.'
}
