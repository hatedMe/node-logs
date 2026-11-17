import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import App from '../App.vue'
import { NAVIGATION_ITEMS, PAGE_KEYS, hashForPage, pageMeta } from '../types/navigation'
import type { ModuleQueryContext } from '../types/modules'
import { genPageClick, genPageFlow, genReportPage } from '../mocks/reports'
import { resetModuleMockScenario, setModuleMockScenario } from '../mocks/modules'
import { resetHashRoute } from '../composables/useHashRoute'
import { resetAppContext } from '../composables/useAppContext'

const plugins = { plugins: [ElementPlus] }

beforeEach(() => {
  resetModuleMockScenario()
  resetAppContext()
  resetHashRoute()
  window.dispatchEvent(new Event('hashchange'))
  setModuleMockScenario({ delayMs: 1 })
  document.body.innerHTML = ''
})

afterEach(() => {
  while (liveWrappers.length > 0) liveWrappers.pop()?.unmount()
  ElMessage.closeAll()
  resetModuleMockScenario()
  resetAppContext()
  resetHashRoute()
  document.body.innerHTML = ''
})

const liveWrappers: VueWrapper[] = []

async function mountAt(hash: string) {
  window.location.hash = hash
  window.dispatchEvent(new Event('hashchange'))
  const wrapper = mount(App, { global: plugins })
  liveWrappers.push(wrapper as unknown as VueWrapper)
  await nextTick()
  await new Promise((r) => setTimeout(r, 60))
  await nextTick()
  await nextTick()
  return wrapper
}

describe('20 个还原报表页逐一渲染', () => {
  it.each(PAGE_KEYS)('%s：菜单标题、指标条与明细表齐备', async (key) => {
    const wrapper = await mountAt(hashForPage(key))
    const root = wrapper.find(`[data-testid="view-${key}"]`)
    expect(root.exists()).toBe(true)
    // 页面标题与菜单名一致（还原目录逐页对应）
    expect(root.find('h1').text()).toBe(pageMeta(key).label)
    expect(root.find('.navigation-report__metrics').exists()).toBe(true)
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('菜单 20 个链接全部可路由（key/path 一一映射）', () => {
    expect(PAGE_KEYS).toHaveLength(NAVIGATION_ITEMS.length)
    for (const item of NAVIGATION_ITEMS) {
      expect(hashForPage(item.key)).toBe(`#/${item.path}`)
    }
  })
})

function baseCtx(patch: Partial<ModuleQueryContext> = {}): ModuleQueryContext {
  return {
    siteId: '16847648',
    rangePreset: 'last7',
    startDate: '2026-11-10',
    endDate: '2026-11-16',
    source: 'all',
    device: 'all',
    keyword: '',
    ...patch
  }
}

describe('报表页 mock 形状与确定性', () => {
  it('20 个生成器都返回非空卡片与明细表', () => {
    for (const key of PAGE_KEYS) {
      const d = genReportPage(key, baseCtx())
      expect(d.cards.length, key).toBeGreaterThan(0)
      expect(d.table.columns.length, key).toBeGreaterThan(0)
      expect(d.table.rows.length, key).toBeGreaterThan(0)
      expect(d.trend.length, key).toBeGreaterThan(0)
    }
  })

  it('同条件稳定返回，站点/设备/日期变化时数据变化', () => {
    expect(genReportPage('source-all', baseCtx())).toEqual(genReportPage('source-all', baseCtx()))
    expect(genReportPage('source-all', baseCtx({ siteId: '5503017' }))).not.toEqual(genReportPage('source-all', baseCtx()))
    expect(genReportPage('source-all', baseCtx({ device: 'mobile' }))).not.toEqual(genReportPage('source-all', baseCtx()))
    expect(genReportPage('source-all', baseCtx({ endDate: '2026-11-17' }))).not.toEqual(genReportPage('source-all', baseCtx()))
  })

  it('关键词做真实子串过滤，无命中时返回空行', () => {
    expect(genReportPage('source-searchword', baseCtx({ keyword: '百度' })).table.rows.length).toBeGreaterThan(0)
    expect(genReportPage('source-searchword', baseCtx({ keyword: 'zzz-no-such-word' })).table.rows).toEqual([])
  })

  it('受访页面切换改变点击图与上下游数据', () => {
    expect(genPageClick(baseCtx(), '/index.html')).not.toEqual(genPageClick(baseCtx(), '/price'))
    expect(genPageFlow(baseCtx(), '/index.html')).not.toEqual(genPageFlow(baseCtx(), '/price'))
  })

  it('empty 模式清空卡片/排行/明细行，保留列头与页面选项', () => {
    setModuleMockScenario({ mode: 'empty' })
    const d = genReportPage('visit-pageclick', baseCtx())
    expect(d.cards).toEqual([])
    expect(d.trend).toEqual([])
    expect(d.heat ?? []).toEqual([])
    expect(d.table.rows).toEqual([])
    expect(d.table.columns.length).toBeGreaterThan(0)
    expect((d.pages ?? []).length).toBeGreaterThan(0)
    resetModuleMockScenario()
  })
})
