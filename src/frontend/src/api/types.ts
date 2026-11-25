// API 계약 타입. mydocs/tech/api_contract.md 와 같아야 한다 (#8 이 이 형식으로 구현한다).

export type Language = 'en' | 'ja' | 'ko'
export type Currency = 'USD' | 'EUR' | 'KRW'
export type MoverPeriod = '7d' | '30d'
/** 시세 추이 기간 */
export type Period = '7d' | '30d' | '90d' | '1y'
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
  /** 원화 환산 (원 단위 정수 문자열, 예: "16602"). 환율이 없으면 null */
  krw: string | null
  /** 환산에 쓴 환율 날짜 (YYYY-MM-DD). 환율이 없으면 null */
  fxDate: string | null
  /** 소스 갱신 시각 (RFC 3339). 오래된 가격 판단용 */
  updatedAt: string | null
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

export interface SetDetail extends SetRef {
  releaseDate: string | null
  logoUrl: string | null
  cardCount: number | null
}

export interface VariantPrices {
  /** card_variants.variant */
  variant: string
  /** 소스별 최신가. 소스 우선순위(TCGCSV → Cardmarket) 순. 시세가 없으면 [] */
  prices: Price[]
}

/** GET /cards/{id} (#13 카드 상세 화면에서 쓴다) */
export interface CardDetail extends Omit<CardSummary, 'set'> {
  set: SetDetail
  rarity: string | null
  /** 도감 번호. 트레이너스·에너지는 [] */
  dexIds: number[]
  /** 한글판 ↔ 일본판 연결 카드 */
  linked: CardSummary[]
  /** 변형 id 순 */
  variants: VariantPrices[]
}

export interface PricePoint {
  date: string
  amount: string
  krw: string | null
}

export interface PriceSeries {
  variant: string
  source: string
  currency: Currency
  /** 날짜 오름차순 */
  points: PricePoint[]
  summary: {
    /** 기간 안 마지막 가격 */
    current: Price
    /** 날짜별 가격의 산술 평균 (소수 둘째 자리) */
    average: string
    min: string
    max: string
    /** (마지막 - 첫) / 첫. 가격이 하나뿐이거나 첫 가격이 0이면 null */
    changeRate: number | null
    /** 기간 안 첫 가격의 날짜 */
    from: string
  }
}

/** GET /cards/{id}/prices (#13 카드 상세 화면에서 쓴다) */
export interface PriceHistoryResponse {
  cardId: number
  period: Period
  /** 요청 시작일 (오늘 KST - 기간) */
  from: string
  /** 변형 id 순 → 소스 우선순위 순 */
  series: PriceSeries[]
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
