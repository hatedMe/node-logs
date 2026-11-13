// 新模块通用异步查询：loading/错误/重试/竞态防护。
// 与 useReport 同构，但 fetcher 是同步纯函数；延迟与失败由模块 mock 场景统一模拟。

import { ref } from 'vue'
import {
  getModuleMockDelay,
  getModuleMockMode,
  MODULE_MOCK_ERROR_MESSAGE
} from '../mocks/modules'

export function useModuleQuery<T>(fetcher: () => T) {
  const data = ref<T | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  let requestId = 0

  async function query(): Promise<boolean> {
    const id = ++requestId
    loading.value = true
    error.value = null
    await new Promise((r) => setTimeout(r, getModuleMockDelay()))
    if (id !== requestId) return false
    if (getModuleMockMode() === 'error') {
      data.value = null
      error.value = MODULE_MOCK_ERROR_MESSAGE
      loading.value = false
      return false
    }
    try {
      data.value = fetcher()
    } catch (e) {
      data.value = null
      error.value = e instanceof Error ? e.message : MODULE_MOCK_ERROR_MESSAGE
      loading.value = false
      return false
    }
    loading.value = false
    return true
  }

  function retry(): Promise<boolean> {
    return query()
  }

  return { data, loading, error, query, retry }
}
