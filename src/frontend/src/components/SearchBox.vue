<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

// 검색창. 제출하면 /search?q=... 로 이동한다. 검색 화면에서는 URL 의 q 를 입력값으로 보여 준다.
withDefaults(defineProps<{ variant?: 'compact' | 'hero' }>(), { variant: 'compact' })

const MAX_LENGTH = 100
const route = useRoute()
const router = useRouter()
const text = ref('')

watch(
  () => [route.name, route.query.q] as const,
  ([name, q]) => {
    text.value = name === 'search' && typeof q === 'string' ? q : ''
  },
  { immediate: true },
)

function submit() {
  const q = text.value.trim()
  if (!q) return
  // 언어 필터는 유지하고 쪽은 처음으로 되돌린다.
  const lang = route.name === 'search' ? route.query.lang : undefined
  void router.push({ name: 'search', query: { q, ...(typeof lang === 'string' ? { lang } : {}) } })
}
</script>

<template>
  <form
    class="search-box"
    :class="variant"
    role="search"
    @submit.prevent="submit"
  >
    <label
      class="visually-hidden"
      :for="`search-${variant}`"
    >카드 검색</label>
    <input
      :id="`search-${variant}`"
      v-model="text"
      type="search"
      :maxlength="MAX_LENGTH"
      enterkeyhint="search"
      autocomplete="off"
      :placeholder="variant === 'hero' ? '포켓몬 이름(피카츄), 카드 이름, 세트 번호로 검색' : '카드 검색'"
    >
    <button
      type="submit"
      :disabled="!text.trim()"
    >
      검색
    </button>
  </form>
</template>

<style scoped>
.search-box {
  display: flex;
  width: 100%;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  overflow: hidden;
}

.search-box:focus-within {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-accent) 20%, transparent);
}

input {
  flex: 1;
  min-width: 0;
  border: 0;
  background: transparent;
  padding: var(--space-2) var(--space-3);
  outline: none;
}

button {
  flex: none;
  border: 0;
  padding: 0 var(--space-4);
  background: var(--color-accent);
  color: var(--color-accent-contrast);
  font-weight: 600;
}

button:disabled {
  opacity: 0.5;
  cursor: default;
}

.hero {
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
}

.hero input {
  padding: var(--space-3) var(--space-4);
  font-size: var(--font-lg);
}

.hero button {
  padding: 0 var(--space-5);
  font-size: var(--font-lg);
}
</style>
