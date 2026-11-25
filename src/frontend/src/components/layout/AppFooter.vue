<template>
  <footer class="footer">
    <div class="container footer-inner">
      <p>
        카드 정보: <a
          href="https://tcgdex.dev"
          target="_blank"
          rel="noopener"
        >TCGdex</a>
        · 포켓몬 이름: <a
          href="https://pokeapi.co"
          target="_blank"
          rel="noopener"
        >PokéAPI</a>
        · 가격: <a
          href="https://www.tcgplayer.com"
          target="_blank"
          rel="noopener"
        >TCGplayer</a> (<a
          href="https://tcgcsv.com"
          target="_blank"
          rel="noopener"
        >TCGCSV</a> 경유) · <a
          href="https://www.cardmarket.com"
          target="_blank"
          rel="noopener"
        >Cardmarket</a> (TCGdex 경유)
        · 환율: <a
          href="https://frankfurter.dev"
          target="_blank"
          rel="noopener"
        >Frankfurter</a>
      </p>
      <p class="muted">
        시세는 참고용이며 실제 거래가와 다를 수 있습니다. 원화는 환율로 환산한 참고값입니다.
        Pokémon 및 관련 명칭은 Nintendo, Creatures, GAME FREAK, The Pokémon Company 의 상표입니다.
        이 사이트는 이들과 관련이 없습니다.
      </p>
    </div>
  </footer>
</template>

<style scoped>
.footer {
  border-top: 1px solid var(--color-border);
  background: var(--color-surface-muted);
  font-size: var(--font-sm);
}

.footer-inner {
  padding-block: var(--space-5);
}

.footer p {
  margin: 0 0 var(--space-2);
}

.muted {
  color: var(--color-text-muted);
}
</style>
