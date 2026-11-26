// 카드 상세의 가격 관련 계산 (화면 공용).
import type { CardDetail, Price } from '@/api/types'

const samePrice = (a: Price, b: Price) =>
  a.source === b.source && a.date === b.date && a.amount === b.amount && a.currency === b.currency

/** 대표가가 나온 변형. API 가 변형을 알려 주지 않으므로 같은 가격(소스·기준일·금액)을 찾는다. 없으면 null. */
export function representativeVariant(card: CardDetail): string | null {
  const rep = card.price
  if (!rep) return null
  return card.variants.find((v) => v.prices.some((p) => samePrice(p, rep)))?.variant ?? null
}

/** 변형 가운데 하나라도 시세가 있는지. */
export function hasAnyPrice(card: CardDetail): boolean {
  return card.variants.some((v) => v.prices.length > 0)
}
