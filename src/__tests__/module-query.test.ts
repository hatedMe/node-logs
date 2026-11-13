import { afterEach, describe, expect, it } from 'vitest'
import { useModuleQuery } from '../composables/useModuleQuery'
import {
  MODULE_MOCK_ERROR_MESSAGE,
  resetModuleMockScenario,
  setModuleMockScenario
} from '../mocks/modules'

afterEach(() => {
  resetModuleMockScenario()
})

describe('useModuleQuery', () => {
  it('正常查询设置数据并清除 loading', async () => {
    setModuleMockScenario({ delayMs: 1 })
    const q = useModuleQuery(() => ({ n: 1 }))
    expect(q.data.value).toBeNull()
    const p = q.query()
    expect(q.loading.value).toBe(true)
    expect(await p).toBe(true)
    expect(q.data.value).toEqual({ n: 1 })
    expect(q.loading.value).toBe(false)
    expect(q.error.value).toBeNull()
  })

  it('error 模式清空旧数据并给出固定错误，重试成功后恢复', async () => {
    setModuleMockScenario({ delayMs: 1 })
    const q = useModuleQuery(() => ({ n: 1 }))
    expect(await q.query()).toBe(true)
    setModuleMockScenario({ mode: 'error', delayMs: 1 })
    expect(await q.query()).toBe(false)
    expect(q.data.value).toBeNull()
    expect(q.error.value).toBe(MODULE_MOCK_ERROR_MESSAGE)
    setModuleMockScenario({ mode: 'ok', delayMs: 1 })
    expect(await q.retry()).toBe(true)
    expect(q.data.value).toEqual({ n: 1 })
    expect(q.error.value).toBeNull()
  })

  it('连续查询只有最后一次生效（竞态防护）', async () => {
    let tag = 'first'
    const q = useModuleQuery(() => tag)
    setModuleMockScenario({ delayMs: 30 })
    const p1 = q.query()
    tag = 'second'
    setModuleMockScenario({ delayMs: 1 })
    const p2 = q.query()
    expect(await p2).toBe(true)
    expect(await p1).toBe(false)
    expect(q.data.value).toBe('second')
    expect(q.loading.value).toBe(false)
  })
})
