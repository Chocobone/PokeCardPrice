/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** "true" 이면 API 클라이언트가 내장 목업 응답을 쓴다 (#8 API 완성 전 개발용). */
  readonly VITE_API_MOCK?: string
}

interface ImportMeta {
  readonly env: ImportMetaEnv
}
