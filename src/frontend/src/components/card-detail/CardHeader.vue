<script setup lang="ts">
import { computed } from 'vue'

import type { CardDetail } from '@/api/types'
import CardImage from '@/components/common/CardImage.vue'
import LangBadge from '@/components/common/LangBadge.vue'
import { formatDate } from '@/utils/format'

// 카드 정보: 큰 이미지, 이름, 한글 종 이름, 세트·번호·희귀도·발매일. 대표가 등 가격은 slot 으로 받는다.
const props = defineProps<{ card: CardDetail }>()

// 카드 이름과 다를 때만 한글 종 이름을 덧붙인다 (CardTile 과 같은 규칙).
const species = computed(() => {
  const s = props.card.speciesNameKo
  return s && !props.card.name.includes(s) ? s : null
})
</script>

<template>
  <div class="card-header">
    <div class="image">
      <CardImage
        :image-url="card.imageUrl"
        :alt="card.name"
        size="high"
      />
    </div>
    <div class="info">
      <LangBadge
        :language="card.language"
        :has-ko-edition="card.hasKoEdition"
      />
      <h1 class="name">
        {{ card.name }}
      </h1>
      <p
        v-if="species"
        class="species"
      >
        {{ species }}
      </p>
      <dl class="facts">
        <div>
          <dt>세트</dt>
          <dd class="set">
            <img
              v-if="card.set.logoUrl"
              class="logo"
              :src="`${card.set.logoUrl}.webp`"
              alt=""
              loading="lazy"
            >
            <span>{{ card.set.name }}</span>
            <span class="muted">({{ card.set.tcgdexId }}<template v-if="card.set.cardCount"> · {{ card.set.cardCount }}장</template>)</span>
          </dd>
        </div>
        <div>
          <dt>번호</dt>
          <dd>{{ card.localId }}</dd>
        </div>
        <div v-if="card.rarity">
          <dt>희귀도</dt>
          <dd>{{ card.rarity }}</dd>
        </div>
        <div v-if="card.set.releaseDate">
          <dt>발매일</dt>
          <dd>{{ formatDate(card.set.releaseDate) }}</dd>
        </div>
        <div v-if="card.dexIds.length">
          <dt>도감 번호</dt>
          <dd>{{ card.dexIds.map((d) => `No.${d}`).join(', ') }}</dd>
        </div>
      </dl>
      <slot />
    </div>
  </div>
</template>

<style scoped>
.card-header {
  display: grid;
  gap: var(--space-5);
}

.image {
  width: min(100%, 300px);
  margin-inline: auto;
}

.info {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  min-width: 0;
}

.name {
  margin: 0;
  overflow-wrap: anywhere;
}

.species {
  margin: 0;
  font-size: var(--font-lg);
  color: var(--color-text-muted);
}

.facts {
  display: grid;
  gap: var(--space-1);
  margin: var(--space-2) 0 0;
}

.facts > div {
  display: grid;
  grid-template-columns: 5.5em 1fr;
  gap: var(--space-2);
}

.facts dt {
  color: var(--color-text-muted);
}

.facts dd {
  margin: 0;
  min-width: 0;
  overflow-wrap: anywhere;
}

.set {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-1) var(--space-2);
}

.logo {
  height: 1.5em;
  width: auto;
  max-width: 6em;
  object-fit: contain;
}

@media (min-width: 640px) {
  .card-header {
    grid-template-columns: 260px 1fr;
    align-items: start;
  }

  .image {
    width: 100%;
  }
}

@media (min-width: 1024px) {
  .card-header {
    grid-template-columns: 320px 1fr;
    gap: var(--space-6);
  }
}
</style>
