<script setup lang="ts">
import { computed, ref, watch } from 'vue'

// TCGdex 카드 이미지. imageUrl 은 확장자 없는 기본 주소이고, 크기에 맞춰 "/low.webp"·"/high.webp" 를 붙인다.
// 이미지가 없거나(한글판 등) 불러오지 못하면 대체 화면을 보여 준다.
const props = withDefaults(
  defineProps<{
    imageUrl: string | null
    alt: string
    size?: 'low' | 'high'
  }>(),
  { size: 'low' },
)

const failed = ref(false)
watch(
  () => props.imageUrl,
  () => (failed.value = false),
)
const src = computed(() => (props.imageUrl && !failed.value ? `${props.imageUrl}/${props.size}.webp` : null))
</script>

<template>
  <div class="card-image">
    <img
      v-if="src"
      :src="src"
      :alt="alt"
      loading="lazy"
      decoding="async"
      @error="failed = true"
    >
    <div
      v-else
      class="placeholder"
      role="img"
      :aria-label="`${alt} (이미지 없음)`"
    >
      <span aria-hidden="true">이미지<br>없음</span>
    </div>
  </div>
</template>

<style scoped>
.card-image {
  aspect-ratio: 63 / 88;
  width: 100%;
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: var(--color-surface-muted);
}

img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.placeholder {
  display: grid;
  place-items: center;
  width: 100%;
  height: 100%;
  border: 1px dashed var(--color-border);
  border-radius: inherit;
  color: var(--color-text-muted);
  font-size: 0.7rem;
  text-align: center;
  line-height: 1.3;
}
</style>
