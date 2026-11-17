import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import App from '../App.vue'
import { resetHashRoute } from '../composables/useHashRoute'
import { resetAppContext } from '../composables/useAppContext'
import { resetModuleMockScenario, setModuleMockScenario } from '../mocks/modules'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

// 直接读样式源码断言窄屏规则，不依赖打包后 CSS 处理。
const layoutCss = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), '../styles/layout.css'),
  'utf-8'
)

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
  ElMessage.closeAll()
  resetModuleMockScenario()
  resetAppContext()
  resetHashRoute()
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

async function mountApp() {
  const wrapper = mount(App, { global: plugins })
  await nextTick()
  return wrapper
}

/** 等待 mock 查询与 el-table 异步渲染完成。 */
async function flushQuery(ms = 60) {
  await new Promise((r) => setTimeout(r, ms))
  await nextTick()
  await nextTick()
}

function findButton(wrapper: ReturnType<typeof mount>, text: string) {
  return wrapper.findAll('button').find((b) => b.text().includes(text))
}

describe('报表页面', () => {
  it('首次加载展示默认网站概况报表', async () => {
    const wrapper = await mountApp()
    await flushQuery()
    const text = wrapper.text()
    expect(text).toContain('网站概况')
    expect(text).toContain('核心指标')
    expect(text).toContain('来源构成')
    expect(text).toContain('Top10 搜索词')
    expect(text).toContain('Top10 受访页面')
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('查询中显示 loading，完成后展示数据', async () => {
    setModuleMockScenario({ delayMs: 60 })
    const wrapper = await mountApp()
    expect(wrapper.text()).toContain('加载中')
    await flushQuery(120)
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
    expect(wrapper.text()).not.toContain('加载中')
  })

  it('空结果显示空状态且下载禁用', async () => {
    setModuleMockScenario({ mode: 'empty', delayMs: 1 })
    const wrapper = await mountApp()
    await flushQuery()
    const text = wrapper.text()
    expect(text).toContain('暂无趋势数据')
    expect(text).toContain('暂无构成数据')
    expect(text).toContain('暂无明细数据')
    const downloadBtn = findButton(wrapper, '下载')?.element as HTMLButtonElement
    expect(downloadBtn.disabled).toBe(true)
  })

  it('关键词筛选提交后明细真实过滤，重置恢复', async () => {
    const wrapper = await mountApp()
    await flushQuery()
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
    const keyword = wrapper
      .findAllComponents({ name: 'ElInput' })
      .find((c) => c.classes().includes('filter-keyword'))
    expect(keyword).toBeTruthy()
    keyword?.vm.$emit('update:modelValue', 'zzz-no-such-word')
    await findButton(wrapper, '查询')?.trigger('click')
    await flushQuery()
    expect(wrapper.text()).toContain('暂无明细数据')
    await findButton(wrapper, '重置')?.trigger('click')
    await flushQuery()
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('有数据时下载导出当前结果，无数据时禁用下载', async () => {
    const wrapper = await mountApp()
    await flushQuery()
    const createObjectURL = vi.fn(() => 'blob:mock')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL })
    // jsdom 不支持 blob 链接导航，stub 掉避免噪音错误。
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    await findButton(wrapper, '下载')?.trigger('click')
    expect(createObjectURL).toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
    clickSpy.mockRestore()
    expect(document.body.textContent).toContain('已导出当前结果')

    setModuleMockScenario({ mode: 'empty', delayMs: 1 })
    await findButton(wrapper, '查询')?.trigger('click')
    await flushQuery()
    const disabledBtn = findButton(wrapper, '下载')?.element as HTMLButtonElement
    expect(disabledBtn.disabled).toBe(true)
  })

  it('数据 Tab 切换换表（搜索词：自然/竞价）', async () => {
    const wrapper = await mountApp()
    await flushQuery()
    window.location.hash = '#/source/searchword'
    window.dispatchEvent(new Event('hashchange'))
    await flushQuery()
    expect(wrapper.text()).toContain('搜索词明细')
    await findButton(wrapper, '竞价搜索词')?.trigger('click')
    await flushQuery()
    expect(wrapper.text()).toContain('竞价搜索词明细')
  })

  it('页面选择器切换后数据随之更新（页面上下游）', async () => {
    const wrapper = await mountApp()
    await flushQuery()
    window.location.hash = '#/custom/flow'
    window.dispatchEvent(new Event('hashchange'))
    await flushQuery()
    expect(wrapper.text()).toContain('/index.html（当前页）')
    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    await selects[selects.length - 1].vm.$emit('update:modelValue', '/price')
    await flushQuery()
    expect(wrapper.text()).toContain('/price（当前页）')
    expect(wrapper.text()).not.toContain('/index.html（当前页）')
  })

  it('地图点击联动钻取抽屉（地域分布）', async () => {
    const wrapper = await mountApp()
    await flushQuery()
    window.location.hash = '#/visit/district'
    window.dispatchEvent(new Event('hashchange'))
    await flushQuery()
    const regionBtns = wrapper.findAll('.map-regions button')
    expect(regionBtns.length).toBeGreaterThan(0)
    const firstRegion = regionBtns[0].text()
    await regionBtns[0].trigger('click')
    await flushQuery(40)
    const openDrawer = wrapper.findAllComponents({ name: 'ElDrawer' }).find((d) => d.props('modelValue') === true)
    expect(openDrawer).toBeTruthy()
    expect(String(openDrawer?.props('title'))).toContain(firstRegion)
    expect(wrapper.text()).toContain('访客数(UV)')
  })

  it('窄屏布局关键行为：栅格断点、窄屏样式与视口切换渲染稳定', async () => {
    const wrapper = await mountApp()
    await flushQuery()
    // 栅格断点声明：窄屏 xs=24 纵向堆叠，桌面 md 分栏（趋势图 14 列 + 构成 10 列）
    const allCols = wrapper.findAllComponents({ name: 'ElCol' })
    const trendCol = allCols.find((c) => c.props('md') === 14)
    const sideCol = allCols.find((c) => c.props('md') === 10)
    expect(trendCol?.props()).toMatchObject({ xs: 24, md: 14 })
    expect(sideCol?.props()).toMatchObject({ xs: 24, md: 10 })
    // 窄屏样式规则真实存在
    expect(layoutCss).toContain('@media (max-width: 768px)')
    expect(layoutCss).toContain('.navigation-report__grid')
    expect(layoutCss).toContain('grid-template-columns: 1fr')
    // 实际改变视口并触发 resize：关键区域仍完整渲染（jsdom 无真实布局，故以上述声明断言为布局依据）
    const originalWidth = window.innerWidth
    Object.defineProperty(window, 'innerWidth', { configurable: true, writable: true, value: 375 })
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(window.innerWidth).toBe(375)
    expect(wrapper.find('.navigation-report__metrics').exists()).toBe(true)
    expect(wrapper.find('.trend-chart').exists()).toBe(true)
    expect(wrapper.find('.navigation-report__filters').exists()).toBe(true)
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
    const buttons = wrapper.findAll('button').map((b) => b.text())
    expect(buttons.some((t) => t.includes('查询'))).toBe(true)
    expect(buttons.some((t) => t.includes('重置'))).toBe(true)
    Object.defineProperty(window, 'innerWidth', { configurable: true, writable: true, value: originalWidth })
    window.dispatchEvent(new Event('resize'))
  })
})
