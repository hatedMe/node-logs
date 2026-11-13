import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import VisitorView from '../views/VisitorView.vue'
import type { ReportFilters } from '../types/report'
import { DEFAULT_METRICS } from '../types/report'
import { buildQueryKey, fetchReport, resetMockScenario, setMockScenario } from '../mocks/report'

const plugins = { plugins: [ElementPlus] }
const liveWrappers: VueWrapper[] = []

function mountView(): VueWrapper {
  const w = mount(VisitorView as never, { global: plugins })
  liveWrappers.push(w as unknown as VueWrapper)
  return w as unknown as VueWrapper
}

async function flush(ms = 80) {
  await new Promise((r) => setTimeout(r, ms))
  await nextTick()
  await nextTick()
}

function baseFilters(patch: Partial<ReportFilters> = {}): ReportFilters {
  return {
    rangePreset: 'last7',
    startDate: '2026-11-06',
    endDate: '2026-11-12',
    compareEnabled: false,
    compareStartDate: '',
    compareEndDate: '',
    source: 'all',
    visitorType: 'all',
    geoLevel: 'province',
    metrics: [...DEFAULT_METRICS],
    ...patch
  }
}

beforeEach(() => {
  resetMockScenario()
  setMockScenario({ delayMs: 1 })
  document.body.innerHTML = ''
})

afterEach(() => {
  while (liveWrappers.length > 0) liveWrappers.pop()?.unmount()
  ElMessage.closeAll()
  resetMockScenario()
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('mock 层地域/引擎筛选', () => {
  it('queryKey 包含生效的 areas 与 engines，同条件稳定、异条件变化', () => {
    const a = buildQueryKey(baseFilters({ source: 'search', areas: ['广东'], engines: ['baidu'] }))
    expect(buildQueryKey(baseFilters({ source: 'search', areas: ['广东'], engines: ['baidu'] }))).toBe(a)
    expect(buildQueryKey(baseFilters())).not.toBe(a)
    expect(buildQueryKey(baseFilters({ source: 'search', areas: ['江苏'], engines: ['baidu'] }))).not.toBe(a)
    expect(a).toContain('广东')
    expect(a).toContain('baidu')
  })

  it('未生效的残留值不进 key：engines 不影响非搜索结果，areas 不影响按国家结果', async () => {
    setMockScenario({ delayMs: 1 })
    expect(buildQueryKey(baseFilters({ source: 'direct', engines: ['baidu'] }))).toBe(
      buildQueryKey(baseFilters({ source: 'direct' }))
    )
    expect(buildQueryKey(baseFilters({ geoLevel: 'country', areas: ['广东'] }))).toBe(
      buildQueryKey(baseFilters({ geoLevel: 'country' }))
    )
    const direct = await fetchReport(baseFilters({ source: 'direct', engines: ['baidu'] }))
    const directNoEngines = await fetchReport(baseFilters({ source: 'direct' }))
    expect(direct).toEqual(directNoEngines)
  })

  it('areas 过滤省份行：指定省份/全国/按国家忽略', async () => {
    setMockScenario({ delayMs: 1 })
    const picked = await fetchReport(baseFilters({ areas: ['广东', '江苏'] }))
    expect(picked.geoRows.map((r) => r.name).sort()).toEqual(['广东', '江苏'])
    const all = await fetchReport(baseFilters({ areas: ['全国'] }))
    expect(all.geoRows.length).toBe(20)
    const def = await fetchReport(baseFilters())
    expect(def.geoRows.length).toBe(20)
    const country = await fetchReport(baseFilters({ geoLevel: 'country', areas: ['广东'] }))
    expect(country.geoRows.length).toBe(12)
  })

  it('engines 改变搜索来源的数据量级', async () => {
    setMockScenario({ delayMs: 1 })
    const allEngines = await fetchReport(baseFilters({ source: 'search' }))
    const oneEngine = await fetchReport(baseFilters({ source: 'search', engines: ['baidu'] }))
    expect(oneEngine.trend[0].values.pv).toBeLessThan(allEngines.trend[0].values.pv)
    expect(allEngines.trend).toHaveLength(7)
    expect(allEngines.geoRows).toHaveLength(20)
  })
})

describe('工具栏：下载/帮助/问题搜索', () => {
  it('有数据时工具栏下载可用，无数据时禁用', async () => {
    const wrapper = mountView()
    await flush()
    const createObjectURL = vi.fn(() => 'blob:mock')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL })
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    await wrapper.find('[data-testid="toolbar-download"]').trigger('click')
    expect(createObjectURL).toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
    clickSpy.mockRestore()

    setMockScenario({ mode: 'empty', delayMs: 1 })
    await wrapper.findAll('button').find((b) => b.text() === '查询')?.trigger('click')
    await flush()
    expect(
      (wrapper.find('[data-testid="toolbar-download"]').element as HTMLButtonElement).disabled
    ).toBe(true)
  })

  it('帮助弹层可打开关闭，问题搜索框过滤常见问题', async () => {
    const wrapper = mountView()
    await flush()
    await wrapper.find('[data-testid="toolbar-help"]').trigger('click')
    await nextTick()
    const dialog = wrapper.findComponent({ name: 'ElDialog' })
    expect(dialog.props('modelValue')).toBe(true)
    expect(wrapper.text()).toContain('地域分布报告说明')
    expect(wrapper.findAll('.faq-item').length).toBeGreaterThan(3)

    await wrapper.find('[data-testid="toolbar-search"]').setValue('下载')
    await nextTick()
    const items = wrapper.findAll('.faq-item')
    expect(items.length).toBeGreaterThan(0)
    for (const item of items) {
      expect(item.text()).toContain('下载')
    }

    await wrapper.findAll('.el-dialog__footer button').find((b) => b.text() === '关闭')?.trigger('click')
    await nextTick()
    expect(wrapper.findComponent({ name: 'ElDialog' }).props('modelValue')).toBe(false)
  })
})

describe('高级筛选区', () => {
  it('可收起/展开，收起后 DOM 保留（v-show）', async () => {
    const wrapper = mountView()
    await flush()
    const toggle = () => wrapper.findAll('button').find((b) => b.text().includes('高级筛选'))
    expect(toggle()?.text()).toContain('收起高级筛选')
    await toggle()?.trigger('click')
    await nextTick()
    expect(toggle()?.text()).toContain('展开高级筛选')
    // 指标选择区被隐藏但 DOM 保留
    const metricItem = wrapper
      .findAll('.el-form-item')
      .find((i) => i.text().includes('网站基础指标'))
    expect(metricItem?.attributes('style')).toContain('display: none')
    expect(wrapper.findAll('input[type="checkbox"]').length).toBeGreaterThan(0)
  })

  it('地域筛选：选省份后查询只返回所选省份，全国恢复全部', async () => {
    const wrapper = mountView()
    await flush()
    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    const areas = selects[selects.length - 1]
    areas.vm.$emit('update:modelValue', ['广东', '江苏'])
    await nextTick()
    await wrapper.findAll('button').find((b) => b.text() === '查询')?.trigger('click')
    await flush()
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(2)
    expect(wrapper.text()).toContain('广东')
    expect(wrapper.text()).not.toContain('浙江')

    areas.vm.$emit('update:modelValue', ['全国'])
    await nextTick()
    await wrapper.findAll('button').find((b) => b.text() === '查询')?.trigger('click')
    await flush()
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(20)
  })

  it('搜索引擎二级选项仅在来源=搜索引擎时出现，选择后数据变化', async () => {
    const wrapper = mountView()
    await flush()
    expect(wrapper.text()).not.toContain('360搜索')
    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    selects[0].vm.$emit('update:modelValue', 'search')
    await nextTick()
    for (const e of ['百度', 'Google', '搜狗', 'Bing', '360搜索']) {
      expect(wrapper.text()).toContain(e)
    }
    const before = wrapper.find('.metric-cards').text()
    const engineGroup = wrapper.findAllComponents({ name: 'ElCheckboxGroup' })[0]
    await engineGroup.setValue(['baidu'])
    await wrapper.findAll('button').find((b) => b.text() === '查询')?.trigger('click')
    await flush()
    expect(wrapper.find('.metric-cards').text()).not.toBe(before)
  })

  it('默认勾选 PV/UV/IP/跳出率/平均访问时长 5 项，指标分组与说明齐全', async () => {
    const wrapper = mountView()
    await flush()
    for (const g of ['网站基础指标', '流量质量指标', '转化指标']) {
      expect(wrapper.text()).toContain(g)
    }
    expect(wrapper.text()).toContain('去重后的独立访客数')
    expect(wrapper.text()).toContain('已选 5 项')
    const checked = wrapper
      .findAll('.metric-group input[type="checkbox"]')
      .filter((b) => (b.element as HTMLInputElement).checked)
    expect(checked).toHaveLength(5)
    // 第 7 项仍被拦截
    const group = wrapper.findComponent({ name: 'ElCheckboxGroup' })
    const before = (wrapper.findComponent({ name: 'ReportFilters' }).emitted('update:filters') ?? []).length
    await group.setValue(['pv', 'uv', 'ip', 'visits', 'avgPages', 'bounceRate', 'avgDuration'])
    expect((wrapper.findComponent({ name: 'ReportFilters' }).emitted('update:filters') ?? []).length).toBe(before)
  })

  it('重置恢复 5 项默认指标并清空地域/引擎选择', async () => {
    const wrapper = mountView()
    await flush()
    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    selects[0].vm.$emit('update:modelValue', 'search')
    selects[selects.length - 1].vm.$emit('update:modelValue', ['广东'])
    await nextTick()
    await wrapper.findAll('button').find((b) => b.text() === '重置')?.trigger('click')
    await flush()
    expect(selects[0].props('modelValue')).toBe('all')
    expect(selects[selects.length - 1].props('modelValue')).toEqual([])
    expect(wrapper.text()).toContain('已选 5 项')
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(20)
  })
})

describe('地域 tab 与地图/列表切换', () => {
  it('按国家 tab 切换后自动重查，无需点查询', async () => {
    const wrapper = mountView()
    await flush()
    await wrapper.findAll('.el-tabs__item').find((t) => t.text() === '按国家')?.trigger('click')
    await flush()
    expect(wrapper.text()).toContain('地域排行（按国家）')
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(12)
  })

  it('默认展示地图占位，可切换到地域排行', async () => {
    const wrapper = mountView()
    await flush()
    expect(wrapper.text()).toContain('地域分布地图（占位）')
    expect(wrapper.find('.map-placeholder').attributes('style') ?? '').not.toContain('display: none')
    await wrapper.findAll('.el-tabs__item').find((t) => t.text() === '地域排行')?.trigger('click')
    await nextTick()
    expect(wrapper.find('.map-placeholder').attributes('style')).toContain('display: none')
    expect(wrapper.text()).toContain('地域排行（按省）')
  })

  it('地图地域按钮点击联动明细表抽屉', async () => {
    const wrapper = mountView()
    await flush()
    const firstRegion = wrapper.find('.map-regions button').text()
    await wrapper.find('.map-regions button').trigger('click')
    await flush(40)
    const drawer = wrapper.findComponent({ name: 'ElDrawer' })
    expect(drawer.props('modelValue')).toBe(true)
    expect(String(drawer.props('title'))).toContain(firstRegion)
    expect(wrapper.find('tbody tr.revealed-row').exists()).toBe(true)
  })
})

describe('地域明细列表', () => {
  it('默认按 PV 降序、每页 20 条', async () => {
    const wrapper = mountView()
    await flush()
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(20)
    expect(wrapper.findComponent({ name: 'ElPagination' }).props('pageSize')).toBe(20)
    // 列表首行与 PV 排行第一同为 PV 最大地域
    const firstRow = wrapper.find('tbody tr.el-table__row').text()
    expect(firstRow).toContain(wrapper.find('.geo-view .geo-name').text())
  })

  it('点击 PV 表头可切换升序', async () => {
    const wrapper = mountView()
    await flush()
    const pvHeader = wrapper.findAll('.report-table > div > .el-table thead th')[1]
    await pvHeader.find('.sort-caret.ascending').trigger('click')
    await flush(40)
    // 升序后首行不再是 PV 最大地域
    const firstRow = wrapper.find('tbody tr.el-table__row').text()
    expect(firstRow).not.toContain(wrapper.find('.geo-view .geo-name').text())
  })
})
