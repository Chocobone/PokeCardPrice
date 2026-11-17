<script setup lang="ts">
import { computed } from 'vue'

import type { Mover } from '@/api/types'
import CardImage from '@/components/common/CardImage.vue'
import LangBadge from '@/components/common/LangBadge.vue'
import { changeDirection, formatChangeRate, formatPrice } from '@/utils/format'

const props = defineProps<{ rank: number; mover: Mover }>()
const dir = computed(() => changeDirection(props.mover.changeRate))
</script>

<template>
  <li>
    <RouterLink
      class="row"
      :to="{ name: 'card', params: { id: mover.card.id } }"
    >
      <span class="rank">{{ rank }}</span>
      <CardImage
        class="thumb"
        :image-url="mover.card.imageUrl"
        :alt="mover.card.name"
      />
      <span class="info">
        <span class="name">{{ mover.card.name }}</span>
        <span class="meta">
          <LangBadge
            :language="mover.card.language"
            :has-ko-edition="mover.card.hasKoEdition"
          />
          <span class="set">{{ mover.card.set.name }} · {{ mover.card.localId }}</span>
        </span>
      </span>
      <span class="price">
        <span class="current">{{ formatPrice(mover.current) }}</span>
        <span
          class="change"
          :class="dir"
        >
          <span aria-hidden="true">{{ dir === 'up' ? '▲' : dir === 'down' ? '▼' : '' }}</span>
          {{ formatChangeRate(mover.changeRate) }}
        </span>
      </span>
    </RouterLink>
  </li>
</template>

<style scoped>
.row {
  display: grid;
  grid-template-columns: 1.5rem 40px minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  color: inherit;
  text-decoration: none;
  border-radius: var(--radius-sm);
}

.row:hover {
  background: var(--color-surface-muted);
}

.rank {
  font-weight: 700;
  color: var(--color-text-muted);
  text-align: center;
}

.info {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.name {
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
  font-size: var(--font-sm);
  color: var(--color-text-muted);
}

.set {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.price {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.current {
  font-weight: 600;
}

.change {
  font-size: var(--font-sm);
  font-weight: 600;
}

.up {
  color: var(--color-up);
}

.down {
  color: var(--color-down);
}
</style>
