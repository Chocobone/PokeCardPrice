<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'

import CardHeader from '@/components/card-detail/CardHeader.vue'
import StateView from '@/components/common/StateView.vue'
import { SITE_NAME } from '@/router'
import { useCardDetailStore } from '@/stores/cardDetail'

// 카드 상세 화면 (/cards/:id). 카드를 바꾸면(같은 화면에서 id 만 바뀜) 이전 카드의 요청을 취소한다.
const route = useRoute()
const store = useCardDetailStore()

/** 주소의 카드 id. 양의 정수가 아니면 null (요청하지 않고 찾을 수 없음으로 보인다). */
const id = computed<number | null>(() => {
  const raw = route.params.id
  const s = Array.isArray(raw) ? raw[0] : raw
  if (typeof s !== 'string' || !/^[1-9]\d{0,15}$/.test(s)) return null
  const n = Number(s)
  return Number.isSafeInteger(n) ? n : null
})

const card = computed(() => (id.value === null ? undefined : store.cards[id.value]))
const loading = computed(() => id.value !== null && !!store.cardLoading[id.value])
const error = computed(() => (id.value === null ? undefined : store.cardErrors[id.value]))
const notFound = computed(() => id.value === null || !!error.value?.notFound)

watch(
  id,
  (v) => {
    if (v !== null) void store.loadCard(v)
  },
  { immediate: true },
)

function retry() {
  if (id.value !== null) void store.loadCard(id.value, { force: true })
}

// 탭 제목: 카드를 받으면 "{이름} ({세트} {번호})", 없는 카드면 "카드를 찾을 수 없습니다".
// 라우터의 afterEach 가 먼저 meta.title 을 쓰므로 그 뒤에 덮는다.
watch(
  [card, notFound, () => route.fullPath],
  ([c, nf]) => {
    if (c) document.title = `${c.name} (${c.set.name} ${c.localId}) · ${SITE_NAME}`
    else if (nf) document.title = `카드를 찾을 수 없습니다 · ${SITE_NAME}`
  },
  { immediate: true, flush: 'post' },
)
</script>

<template>
  <section class="card-detail">
    <div
      v-if="notFound"
      class="not-found"
    >
      <h1>카드를 찾을 수 없습니다</h1>
      <p class="muted">
        주소가 잘못되었거나 카드 정보가 바뀌었습니다. 검색으로 다시 찾아 주세요.
      </p>
      <RouterLink to="/">
        홈으로
      </RouterLink>
    </div>
    <StateView
      v-else-if="error"
      kind="error"
      :message="error.message"
      retryable
      @retry="retry"
    />
    <template v-else-if="card">
      <CardHeader :card="card" />
    </template>
    <StateView
      v-else-if="loading"
      kind="loading"
      message="카드 정보를 불러오는 중…"
    />
  </section>
</template>

<style scoped>
.card-detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.not-found {
  text-align: center;
  padding-block: var(--space-8);
}
</style>
