import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import App from '../App.vue'
import { resetMockScenario, setMockScenario } from '../mocks/report'
import { readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

// 直接读样式源码断言窄屏规则，不依赖打包器 CSS 处理。
const reportCss = readFileSync(
  join(dirname(fileURLToPath(import.meta.url)), '../styles/report.css'),
  'utf-8'
)

const plugins = { plugins: [ElementPlus] }

afterEach(() => {
  resetMockScenario()
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

describe('报表页面', () => {
  it('首次加载展示默认 PV/UV 报表', async () => {
    setMockScenario({ delayMs: 1 })
    const wrapper = await mountApp()
    await flushQuery()
    const text = wrapper.text()
    expect(text).toContain('PV/UV 统计报表')
    expect(text).toContain('PV 浏览量')
    expect(text).toContain('UV 访客数')
    expect(text).toContain('地域排行（按省）')
    expect(text).toContain('地域明细')
    expect(wrapper.find('svg').exists()).toBe(true)
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('查询中显示 loading，完成后展示数据', async () => {
    setMockScenario({ delayMs: 60 })
    const wrapper = await mountApp()
    expect(wrapper.findAll('.el-skeleton').length).toBeGreaterThan(0)
    await flushQuery(120)
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('失败显示重试入口，重试成功后恢复数据', async () => {
    setMockScenario({ mode: 'error', delayMs: 1 })
    const wrapper = await mountApp()
    await flushQuery()
    expect(wrapper.text()).toContain('模拟查询失败')
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(0)

    setMockScenario({ mode: 'ok', delayMs: 1 })
    const retryBtn = wrapper.findAll('button').find((b) => b.text().includes('重试'))
    expect(retryBtn).toBeTruthy()
    await retryBtn?.trigger('click')
    await flushQuery()
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
    expect(wrapper.text()).not.toContain('模拟查询失败')
  })

  it('空结果显示空状态且下载禁用', async () => {
    setMockScenario({ mode: 'empty', delayMs: 1 })
    const wrapper = await mountApp()
    await flushQuery()
    const text = wrapper.text()
    expect(text).toContain('暂无指标数据')
    expect(text).toContain('暂无趋势数据')
    expect(text).toContain('暂无明细数据')
    const downloadBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('下载'))
      ?.element as HTMLButtonElement
    expect(downloadBtn.disabled).toBe(true)
  })

  it('筛选提交后所有视图同步更新', async () => {
    setMockScenario({ delayMs: 1 })
    const wrapper = await mountApp()
    await flushQuery()
    const groups = wrapper.findAllComponents({ name: 'ElRadioGroup' })
    const geoGroup = groups[groups.length - 1]
    await geoGroup.setValue('country')
    const queryBtn = wrapper.findAll('button').find((b) => b.text().includes('查询'))
    await queryBtn?.trigger('click')
    await flushQuery()
    expect(wrapper.text()).toContain('地域排行（按国家）')
  })

  it('有数据时下载导出当前结果，无数据时禁用下载', async () => {
    setMockScenario({ delayMs: 1 })
    const wrapper = await mountApp()
    await flushQuery()
    const createObjectURL = vi.fn(() => 'blob:mock')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL })
    // jsdom 不支持 blob 链接导航，stub 掉避免噪音错误。
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    const queryDownload = () => wrapper.findAll('button').find((b) => b.text().includes('下载'))
    await queryDownload()?.trigger('click')
    expect(createObjectURL).toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
    clickSpy.mockRestore()
    expect(document.body.textContent).toContain('已导出当前结果')

    setMockScenario({ mode: 'empty', delayMs: 1 })
    const queryBtn = wrapper.findAll('button').find((b) => b.text().includes('查询'))
    await queryBtn?.trigger('click')
    await flushQuery()
    // 无数据时下载按钮禁用（App 内 warning 分支为防御性兜底）
    const disabledBtn = queryDownload()?.element as HTMLButtonElement
    expect(disabledBtn.disabled).toBe(true)
  })

  it('地域排行查看明细联动表格抽屉钻取', async () => {
    setMockScenario({ delayMs: 1 })
    const wrapper = await mountApp()
    await flushQuery()
    const firstRegion = wrapper.find('.geo-view .geo-name').text()
    const detailBtn = wrapper.findAll('button').find((b) => b.text().includes('查看明细'))
    await detailBtn?.trigger('click')
    await flushQuery(40)
    // 明细表为该地域打开抽屉并高亮所在行
    const drawer = wrapper.findComponent({ name: 'ElDrawer' })
    expect(drawer.props('modelValue')).toBe(true)
    expect(String(drawer.props('title'))).toContain(firstRegion)
    expect(wrapper.find('tbody tr.revealed-row').exists()).toBe(true)

    // 表格行点击仍可独立钻取
    await wrapper.find('tbody tr.el-table__row').trigger('click')
    expect(drawer.props('modelValue')).toBe(true)
    expect(String(drawer.props('title'))).toContain('明细')
  })

  it('窄屏布局关键行为：栅格断点、窄屏样式与视口切换渲染稳定', async () => {
    setMockScenario({ delayMs: 1 })
    const wrapper = await mountApp()
    await flushQuery()
    // 栅格断点声明：窄屏 xs=24 纵向堆叠，桌面 md 分栏（按 md 值精确找到图表两列）
    const allCols = wrapper.findAllComponents({ name: 'ElCol' })
    const trendCol = allCols.find((c) => c.props('md') === 14)
    const geoCol = allCols.find((c) => c.props('md') === 10)
    expect(trendCol?.props()).toMatchObject({ xs: 24, md: 14 })
    expect(geoCol?.props()).toMatchObject({ xs: 24, md: 10 })
    // 窄屏样式规则真实存在
    expect(reportCss).toContain('@media (max-width: 768px)')
    expect(reportCss).toContain('.report-filters .el-form-item__content')
    expect(reportCss).toContain('flex-wrap')
    // 实际改变视口并触发 resize：关键区域仍完整渲染（jsdom 无真实布局，故以上述声明断言为布局依据）
    const originalWidth = window.innerWidth
    Object.defineProperty(window, 'innerWidth', { configurable: true, writable: true, value: 375 })
    window.dispatchEvent(new Event('resize'))
    await nextTick()
    expect(window.innerWidth).toBe(375)
    expect(wrapper.find('header.report-header').exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ElRadioGroup' }).exists()).toBe(true)
    expect(wrapper.find('.metric-cards').exists()).toBe(true)
    expect(wrapper.find('.trend-chart').exists()).toBe(true)
    expect(wrapper.find('.geo-view').exists()).toBe(true)
    expect(wrapper.find('.report-table').exists()).toBe(true)
    const buttons = wrapper.findAll('button').map((b) => b.text())
    expect(buttons.some((t) => t.includes('查询'))).toBe(true)
    expect(buttons.some((t) => t.includes('重置'))).toBe(true)
    Object.defineProperty(window, 'innerWidth', { configurable: true, writable: true, value: originalWidth })
    window.dispatchEvent(new Event('resize'))
  })
})
