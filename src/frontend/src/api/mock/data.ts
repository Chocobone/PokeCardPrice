// 목업 카드 표본 (36장). 자동 생성 파일이므로 직접 고치지 않는다.
// 출처: #5 카탈로그 수집기로 적재한 개발 DB (2026-11-17 추출).
// 추출 방법: mydocs/tech/api_contract.md "목업 데이터" 절. 가격은 handlers.ts 가 임의로 만든다.
import type { CardSummary } from '../types'

export type MockCard = Omit<CardSummary, 'price'>

export const MOCK_CARDS: MockCard[] = [
  {id: 16946,language: "en",tcgdexId: "sv03.5-001",name: "Bulbasaur",localId: "001",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/001",hasKoEdition: false,speciesNameKo: "이상해씨"},
  {id: 16949,language: "en",tcgdexId: "sv03.5-004",name: "Charmander",localId: "004",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/004",hasKoEdition: false,speciesNameKo: "파이리"},
  {id: 16951,language: "en",tcgdexId: "sv03.5-006",name: "Charizard ex",localId: "006",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/006",hasKoEdition: false,speciesNameKo: "리자몽"},
  {id: 16952,language: "en",tcgdexId: "sv03.5-007",name: "Squirtle",localId: "007",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/007",hasKoEdition: false,speciesNameKo: "꼬부기"},
  {id: 16970,language: "en",tcgdexId: "sv03.5-025",name: "Pikachu",localId: "025",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/025",hasKoEdition: false,speciesNameKo: "피카츄"},
  {id: 16984,language: "en",tcgdexId: "sv03.5-039",name: "Jigglypuff",localId: "039",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/039",hasKoEdition: false,speciesNameKo: "푸린"},
  {id: 17039,language: "en",tcgdexId: "sv03.5-094",name: "Gengar",localId: "094",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/094",hasKoEdition: false,speciesNameKo: "팬텀"},
  {id: 17078,language: "en",tcgdexId: "sv03.5-133",name: "Eevee",localId: "133",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/133",hasKoEdition: false,speciesNameKo: "이브이"},
  {id: 17088,language: "en",tcgdexId: "sv03.5-143",name: "Snorlax",localId: "143",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/143",hasKoEdition: false,speciesNameKo: "잠만보"},
  {id: 17095,language: "en",tcgdexId: "sv03.5-150",name: "Mewtwo",localId: "150",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/150",hasKoEdition: false,speciesNameKo: "뮤츠"},
  {id: 17096,language: "en",tcgdexId: "sv03.5-151",name: "Mew ex",localId: "151",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/151",hasKoEdition: false,speciesNameKo: "뮤"},
  {id: 17111,language: "en",tcgdexId: "sv03.5-166",name: "Bulbasaur",localId: "166",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/166",hasKoEdition: false,speciesNameKo: "이상해씨"},
  {id: 17113,language: "en",tcgdexId: "sv03.5-168",name: "Charmander",localId: "168",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/168",hasKoEdition: false,speciesNameKo: "파이리"},
  {id: 17115,language: "en",tcgdexId: "sv03.5-170",name: "Squirtle",localId: "170",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/170",hasKoEdition: false,speciesNameKo: "꼬부기"},
  {id: 17118,language: "en",tcgdexId: "sv03.5-173",name: "Pikachu",localId: "173",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/173",hasKoEdition: false,speciesNameKo: "피카츄"},
  {id: 17128,language: "en",tcgdexId: "sv03.5-183",name: "Charizard ex",localId: "183",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/183",hasKoEdition: false,speciesNameKo: "리자몽"},
  {id: 17138,language: "en",tcgdexId: "sv03.5-193",name: "Mew ex",localId: "193",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/193",hasKoEdition: false,speciesNameKo: "뮤"},
  {id: 17141,language: "en",tcgdexId: "sv03.5-196",name: "Erika's Invitation",localId: "196",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/196",hasKoEdition: false,speciesNameKo: null},
  {id: 17144,language: "en",tcgdexId: "sv03.5-199",name: "Charizard ex",localId: "199",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/199",hasKoEdition: false,speciesNameKo: "리자몽"},
  {id: 17150,language: "en",tcgdexId: "sv03.5-205",name: "Mew ex",localId: "205",set: {tcgdexId: "sv03.5",name: "151"},imageUrl: "https://assets.tcgdex.net/en/sv/sv03.5/205",hasKoEdition: false,speciesNameKo: "뮤"},
  {id: 11,language: "en",tcgdexId: "base1-10",name: "Mewtwo",localId: "10",set: {tcgdexId: "base1",name: "Base Set"},imageUrl: "https://assets.tcgdex.net/en/base/base1/10",hasKoEdition: false,speciesNameKo: "뮤츠"},
  {id: 5,language: "en",tcgdexId: "base1-4",name: "Charizard",localId: "4",set: {tcgdexId: "base1",name: "Base Set"},imageUrl: "https://assets.tcgdex.net/en/base/base1/4",hasKoEdition: false,speciesNameKo: "리자몽"},
  {id: 47,language: "en",tcgdexId: "base1-46",name: "Charmander",localId: "46",set: {tcgdexId: "base1",name: "Base Set"},imageUrl: "https://assets.tcgdex.net/en/base/base1/46",hasKoEdition: false,speciesNameKo: "파이리"},
  {id: 59,language: "en",tcgdexId: "base1-58",name: "Pikachu",localId: "58",set: {tcgdexId: "base1",name: "Base Set"},imageUrl: "https://assets.tcgdex.net/en/base/base1/58",hasKoEdition: false,speciesNameKo: "피카츄"},
  {id: 27477,language: "ja",tcgdexId: "SM10-033",name: "ゲンガー",localId: "033",set: {tcgdexId: "SM10",name: "ダブルブレイズ"},imageUrl: null,hasKoEdition: false,speciesNameKo: "팬텀"},
  {id: 27480,language: "ja",tcgdexId: "SM10-036",name: "ミュウツー",localId: "036",set: {tcgdexId: "SM10",name: "ダブルブレイズ"},imageUrl: null,hasKoEdition: false,speciesNameKo: "뮤츠"},
  {id: 27481,language: "ja",tcgdexId: "SM10-037",name: "ミュウ",localId: "037",set: {tcgdexId: "SM10",name: "ダブルブレイズ"},imageUrl: null,hasKoEdition: false,speciesNameKo: "뮤"},
  {id: 27319,language: "ja",tcgdexId: "SM9a-014",name: "ピカチュウ",localId: "014",set: {tcgdexId: "SM9a",name: "ナイトユニゾン"},imageUrl: null,hasKoEdition: false,speciesNameKo: "피카츄"},
  {id: 27200,language: "ja",tcgdexId: "SM9-013",name: "リザードン",localId: "013",set: {tcgdexId: "SM9",name: "タッグボルト"},imageUrl: null,hasKoEdition: false,speciesNameKo: "리자몽"},
  {id: 27218,language: "ja",tcgdexId: "SM9-031",name: "ピカチュウ&ゼクロムGX",localId: "031",set: {tcgdexId: "SM9",name: "タッグボルト"},imageUrl: null,hasKoEdition: false,speciesNameKo: "피카츄"},
  {id: 27225,language: "ja",tcgdexId: "SM9-038",name: "ゲンガー&ミミッキュGX",localId: "038",set: {tcgdexId: "SM9",name: "タッグボルト"},imageUrl: null,hasKoEdition: false,speciesNameKo: "팬텀"},
  {id: 27253,language: "ja",tcgdexId: "SM9-066",name: "イーブイ&カビゴンGX",localId: "066",set: {tcgdexId: "SM9",name: "タッグボルト"},imageUrl: null,hasKoEdition: false,speciesNameKo: "이브이"},
  {id: 27293,language: "ja",tcgdexId: "SM9-106",name: "イーブイ&カビゴンGX",localId: "106",set: {tcgdexId: "SM9",name: "タッグボルト"},imageUrl: null,hasKoEdition: false,speciesNameKo: "이브이"},
  {id: 27081,language: "ja",tcgdexId: "SM8b-209",name: "リザードンGX",localId: "209",set: {tcgdexId: "SM8b",name: "GXウルトラシャイニー"},imageUrl: null,hasKoEdition: false,speciesNameKo: "리자몽"},
  {id: 27091,language: "ja",tcgdexId: "SM8b-219",name: "ミュウツーGX",localId: "219",set: {tcgdexId: "SM8b",name: "GXウルトラシャイニー"},imageUrl: null,hasKoEdition: false,speciesNameKo: "뮤츠"},
  {id: 25336,language: "ja",tcgdexId: "PCG5-041",name: "Mew Ex",localId: "041",set: {tcgdexId: "PCG5",name: "まぼろしの森"},imageUrl: null,hasKoEdition: false,speciesNameKo: "뮤"},
]
