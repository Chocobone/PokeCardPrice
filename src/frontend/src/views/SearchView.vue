<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery, type LocationQueryRaw } from 'vue-router'

import type { Language } from '@/api/types'
import CardGrid from '@/components/cards/CardGrid.vue'
import PageNav from '@/components/common/PageNav.vue'
import StateView from '@/components/common/StateView.vue'
import { SITE_NAME } from '@/router'
import { useSearchStore, type SearchQuery } from '@/stores/search'

// 검색 결과 화면. URL 쿼리(q, lang, page)가 상태의 기준이다.
// 필터·쪽을 바꾸면 URL 만 바꾸고, URL 이 바뀌면 검색한다 (새로고침·뒤로 가기·링크 공유 대응).
const LANGS: { value: Language | undefined; label: string }[] = [
  { value: undefined, label: '전체' },
  { value: 'en', label: '영어판' },
  { value: 'ja', label: '일본판' },
  { value: 'ko', label: '한글판' },
]

const route = useRoute()
const router = useRouter()
const store = useSearchStore()

const first = (v: unknown) => (Array.isArray(v) ? v[0] : v)

/** URL 쿼리를 검사한 질의. q 가 없으면 null. */
const query = computed<SearchQuery | null>(() => {
  const q = first(route.query.q)
  if (typeof q !== 'string' || !q.trim()) return null
  const lang = first(route.query.lang)
  const page = Number(first(route.query.page) ?? 1)
  return {
    q: q.trim().slice(0, 100),
    lang: lang === 'en' || lang === 'ja' || lang === 'ko' ? lang : undefined,
    page: Number.isInteger(page) && page >= 1 ? page : 1,
  }
})

function toLocation(q: SearchQuery): LocationQueryRaw {
  return { q: q.q, lang: q.lang, page: q.page > 1 ? String(q.page) : undefined }
}

/** URL 쿼리가 want(값이 undefined 인 키는 없는 것으로 봄)와 같은지. */
function sameQuery(current: LocationQuery, want: LocationQueryRaw): boolean {
  const keys = new Set([...Object.keys(current), ...Object.keys(want).filter((k) => want[k] !== undefined)])
  return [...keys].every((k) => current[k] === want[k])
}

watch(
  query,
  (q) => {
    if (!q) return
    // 잘못된 값(page=abc, 모르는 lang, 앞뒤 공백)은 고친 주소로 바꾼다. 바뀐 주소로 다시 이 감시가 돈다.
    // 문자열(fullPath)이 아니라 값으로 비교한다. 공백 인코딩(%20·+)만 다른 주소를 다르다고 보면
    // 같은 위치로의 replace 가 중복 이동으로 무시되어 검색이 실행되지 않는다.
    if (!sameQuery(route.query, toLocation(q))) {
      void router.replace({ name: 'search', query: toLocation(q) })
      return
    }
    document.title = `"${q.q}" 검색 · ${SITE_NAME}`
    void store.run(q)
  },
  { immediate: true },
)

const result = computed(() => (query.value && store.query?.q === query.value.q ? store.result : null))
const totalPages = computed(() => (result.value ? Math.max(1, Math.ceil(result.value.total / result.value.size)) : 1))

// 결과보다 큰 쪽 번호로 들어오면 마지막 쪽으로 보낸다.
watch(result, (r) => {
  const q = query.value
  if (r && q && r.total > 0 && q.page > totalPages.value) {
    void router.replace({ name: 'search', query: toLocation({ ...q, page: totalPages.value }) })
  }
})

function setLang(lang: Language | undefined) {
  if (!query.value || query.value.lang === lang) return
  void router.push({ name: 'search', query: toLocation({ ...query.value, lang, page: 1 }) })
}

function goPage(page: number) {
  if (!query.value) return
  void router.push({ name: 'search', query: toLocation({ ...query.value, page }) })
}

function retry() {
  if (query.value) void store.run(query.value, { force: true })
}
</script>

<template>
  <section v-if="!query">
    <h1>카드 검색</h1>
    <StateView
      kind="empty"
      message="위 검색창에 포켓몬 이름(피카츄), 카드 이름, 세트 번호를 입력하세요."
    />
  </section>
  <section v-else>
    <div class="head">
      <h1>
        <span class="q">“{{ query.q }}”</span> 검색 결과
        <span
          v-if="result"
          class="count"
        >{{ result.total.toLocaleString('ko-KR') }}장</span>
      </h1>
      <div
        class="langs"
        role="group"
        aria-label="언어판"
      >
        <button
          v-for="l in LANGS"
          :key="l.label"
          type="button"
          :class="{ active: query.lang === l.value }"
          :aria-pressed="query.lang === l.value"
          @click="setLang(l.value)"
        >
          {{ l.label }}
        </button>
      </div>
    </div>

    <StateView
      v-if="store.error"
      kind="error"
      :message="store.error"
      retryable
      @retry="retry"
    />
    <StateView
      v-else-if="store.loading && !result"
      kind="loading"
      message="검색 중…"
    />
    <StateView
      v-else-if="result && result.total === 0"
      kind="empty"
      message="검색 결과가 없습니다."
    >
      <p class="tips">
        한글 포켓몬 이름(예: 피카츄), 영어·일본어 카드 이름, 세트 ID와 번호(예: sv03.5 25)로 검색할 수 있습니다.
        <template v-if="query.lang">
          언어판 필터를 "전체"로 바꿔 보세요.
        </template>
      </p>
    </StateView>
    <div
      v-else-if="result"
      :class="{ stale: store.loading }"
      :aria-busy="store.loading"
    >
      <CardGrid :cards="result.items" />
      <PageNav
        :page="result.page"
        :total-pages="totalPages"
        @go="goPage"
      />
    </div>
  </section>
</template>

<style scoped>
.head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}

h1 {
  margin: 0;
  font-size: var(--font-xl);
  overflow-wrap: anywhere;
}

.count {
  margin-left: var(--space-2);
  font-size: var(--font-md);
  font-weight: 400;
  color: var(--color-text-muted);
}

.langs {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.langs button {
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: 999px;
  background: var(--color-surface);
  font-size: var(--font-sm);
}

.langs button.active {
  border-color: var(--color-accent);
  background: var(--color-accent);
  color: var(--color-accent-contrast);
  font-weight: 600;
}

.tips {
  max-width: 32rem;
  font-size: var(--font-sm);
}

/* 쪽·필터를 바꾸는 동안 이전 결과를 흐리게 보여 준다 */
.stale {
  opacity: 0.55;
  transition: opacity 0.15s;
}
</style>
