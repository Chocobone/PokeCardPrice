<script setup>
import { onMounted, ref } from 'vue'

// 백엔드 연결 확인용 임시 화면. #12에서 라우터와 실제 화면으로 교체한다.
const status = ref('확인 중...')
const ok = ref(null)

async function checkBackend() {
  try {
    const res = await fetch('/api/healthz')
    if (!res.ok) throw new Error(`HTTP ${res.status}`)
    const body = await res.json()
    status.value = body.status
    ok.value = body.status === 'ok'
  } catch (err) {
    status.value = `연결 실패 (${err.message})`
    ok.value = false
  }
}

onMounted(checkBackend)
</script>

<template>
  <main>
    <h1>Card-Market</h1>
    <p>포켓몬 카드 시세 비교</p>
    <p class="status">
      백엔드 상태:
      <strong :class="{ ok: ok === true, fail: ok === false }">{{ status }}</strong>
    </p>
  </main>
</template>

<style scoped>
main {
  max-width: 40rem;
  margin: 4rem auto;
  padding: 0 1rem;
}

.ok {
  color: #1a7f37;
}

.fail {
  color: #cf222e;
}
</style>
