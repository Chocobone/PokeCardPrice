<script setup lang="ts">
import AppFooter from '@/components/layout/AppFooter.vue'
import AppHeader from '@/components/layout/AppHeader.vue'
</script>

<template>
  <div class="app">
    <a
      class="skip-link"
      href="#main"
    >본문으로 건너뛰기</a>
    <AppHeader />
    <main
      id="main"
      class="app-main container"
    >
      <RouterView />
    </main>
    <AppFooter />
  </div>
</template>

<style scoped>
.app {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  min-height: 100dvh;
}

.app-main {
  flex: 1;
  width: 100%;
  padding-block: var(--space-6) var(--space-8);
}

.skip-link {
  position: absolute;
  left: var(--space-4);
  top: -100px;
  z-index: 10;
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.skip-link:focus {
  top: var(--space-2);
}
</style>
