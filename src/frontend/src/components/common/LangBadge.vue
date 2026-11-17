<script setup lang="ts">
import type { Language } from '@/api/types'

// 언어판 배지. hasKoEdition 이면 일본판 카드 옆에 "한글판" 배지를 함께 붙인다.
defineProps<{
  language: Language
  hasKoEdition?: boolean
}>()

const LABEL: Record<Language, { short: string; long: string }> = {
  en: { short: 'EN', long: '영어판' },
  ja: { short: 'JP', long: '일본판' },
  ko: { short: 'KR', long: '한글판' },
}
</script>

<template>
  <span class="badges">
    <span
      class="badge"
      :class="`lang-${language}`"
      :title="LABEL[language].long"
    >
      {{ LABEL[language].short }}<span class="visually-hidden"> {{ LABEL[language].long }}</span>
    </span>
    <span
      v-if="hasKoEdition && language === 'ja'"
      class="badge ko-edition"
      title="한글판도 발매된 카드"
    >한글판</span>
  </span>
</template>

<style scoped>
.badges {
  display: inline-flex;
  gap: var(--space-1);
  flex-wrap: wrap;
}

.badge {
  display: inline-block;
  padding: 0 0.4em;
  border-radius: 4px;
  font-size: 0.7rem;
  font-weight: 700;
  line-height: 1.6;
  border: 1px solid currentColor;
  white-space: nowrap;
}

.lang-en {
  color: #5c7cfa;
}

.lang-ja {
  color: #e8590c;
}

.lang-ko,
.ko-edition {
  color: #2f9e44;
}
</style>
