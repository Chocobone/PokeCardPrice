<script setup lang="ts">
import { computed } from 'vue'

import type { CardSummary } from '@/api/types'
import CardImage from '@/components/common/CardImage.vue'
import LangBadge from '@/components/common/LangBadge.vue'
import { formatPrice } from '@/utils/format'

const props = defineProps<{ card: CardSummary }>()

// 카드 이름과 다를 때만 한글 종 이름을 덧붙인다 (예: Pikachu → 피카츄).
const species = computed(() => {
  const s = props.card.speciesNameKo
  return s && !props.card.name.includes(s) ? s : null
})
</script>

<template>
  <li class="tile">
    <RouterLink
      class="link"
      :to="{ name: 'card', params: { id: card.id } }"
    >
      <CardImage
        :image-url="card.imageUrl"
        :alt="card.name"
      />
      <span class="body">
        <span class="name">{{ card.name }}</span>
        <span
          v-if="species"
          class="species"
        >{{ species }}</span>
        <span class="set">{{ card.set.name }} · {{ card.localId }}</span>
        <LangBadge
          :language="card.language"
          :has-ko-edition="card.hasKoEdition"
        />
        <span
          class="price"
          :class="{ none: !card.price }"
        >{{ formatPrice(card.price) }}</span>
      </span>
    </RouterLink>
  </li>
</template>

<style scoped>
.tile {
  min-width: 0;
}

.link {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  height: 100%;
  padding: var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: inherit;
  text-decoration: none;
  transition: box-shadow 0.15s;
}

.link:hover {
  box-shadow: var(--shadow-sm);
}

.body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.name {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.species,
.set {
  font-size: var(--font-sm);
  color: var(--color-text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.price {
  margin-top: var(--space-1);
  font-weight: 700;
  font-variant-numeric: tabular-nums;
}

.price.none {
  font-weight: 400;
  font-size: var(--font-sm);
  color: var(--color-text-muted);
}
</style>
