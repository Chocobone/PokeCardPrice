// API 계약 타입. mydocs/tech/api_contract.md 와 같아야 한다 (#8 이 이 형식으로 구현한다).

export type Language = 'en' | 'ja' | 'ko'
export type Currency = 'USD' | 'JPY' | 'KRW'
export type MoverPeriod = '7d' | '30d'
export type MoverDirection = 'up' | 'down'

export interface SetRef {
  /** TCGdex 세트 ID (예: "sv03.5") */
  tcgdexId: string
  name: string
}

export interface Price {
  /** 금액. 정밀도를 지키기 위해 문자열이다 (예: "12.34"). */
  amount: string
  currency: Currency
  /** price_sources 코드 (예: "tcgcsv") */
  source: string
  /** 가격 기준일 (YYYY-MM-DD) */
  date: string
}

export interface CardSummary {
  /** cards.id */
  id: number
  language: Language
  /** TCGdex 카드 ID (예: "sv03.5-025") */
  tcgdexId: string
  name: string
  /** 세트 안 카드 번호 (예: "025") */
  localId: string
  set: SetRef
  /** TCGdex 이미지 기본 주소. 화면에서 "/low.webp" 등을 붙인다. 없으면 null */
  imageUrl: string | null
  /** 일본판 카드에 한글판이 있는지 */
  hasKoEdition: boolean
  /** 첫 도감 번호의 한글 종 이름. 트레이너스·에너지는 null */
  speciesNameKo: string | null
  /** 대표가. 시세가 없으면 null */
  price: Price | null
}

export interface Page<T> {
  items: T[]
  /** 1부터 시작 */
  page: number
  size: number
  total: number
}

export interface Mover {
  card: CardSummary
  /** card_variants.variant (예: "holofoil") */
  variant: string
  current: Price
  previous: Price
  /** (current - previous) / previous. 0.125 = 12.5% */
  changeRate: number
}

export interface MoversResponse {
  period: MoverPeriod
  direction: MoverDirection
  /** 순위 기준일 (YYYY-MM-DD). 시세가 없으면 null */
  asOf: string | null
  items: Mover[]
}

export interface SearchParams {
  q: string
  lang?: Language
  page?: number
  size?: number
}

export interface MoversParams {
  period: MoverPeriod
  direction: MoverDirection
  limit?: number
}

/** 오류 응답 본문 */
export interface ErrorBody {
  error: { code: string; message: string }
}
