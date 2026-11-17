// 설정 파일은 JS 로 둔다 (TS 설정 파일은 ESLint 가 jiti 패키지를 추가로 요구한다).
import pluginVue from 'eslint-plugin-vue'
import { vueTsConfigs, withVueTs } from '@vue/eslint-config-typescript'

// Vue 3 권장 규칙 + typescript-eslint 권장 규칙. 모든 .vue 의 <script> 는 lang="ts" 여야 한다.
export default withVueTs(
  { ignores: ['dist/**', 'node_modules/**'] },
  pluginVue.configs['flat/recommended'],
  vueTsConfigs.recommended,
)
