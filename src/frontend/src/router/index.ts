import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

import HomeView from '@/views/HomeView.vue'

declare module 'vue-router' {
  interface RouteMeta {
    /** 브라우저 탭 제목. 없으면 서비스 이름만 쓴다. */
    title?: string
  }
}

export const SITE_NAME = 'Card-Market'

const routes: RouteRecordRaw[] = [
  { path: '/', name: 'home', component: HomeView },
  {
    path: '/search',
    name: 'search',
    component: () => import('@/views/SearchView.vue'),
    meta: { title: '검색' },
  },
  {
    // 카드 상세 화면은 #13 에서 만든다. 지금은 자리 화면이다.
    path: '/cards/:id',
    name: 'card',
    component: () => import('@/views/CardPlaceholderView.vue'),
    meta: { title: '카드 상세' },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { title: '페이지를 찾을 수 없습니다' },
  },
]

const router = createRouter({
  // 배포 시 서버가 모든 경로에 index.html 을 돌려줘야 한다 (#16).
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
  scrollBehavior(_to, _from, saved) {
    return saved ?? { top: 0 }
  },
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · ${SITE_NAME}` : SITE_NAME
})

export default router
