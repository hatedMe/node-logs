import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import App from '../App.vue'
import ReportFilterStrip from '../components/report/ReportFilterStrip.vue'
import { NAVIGATION_GROUPS, NAVIGATION_ITEMS } from '../types/navigation'
import { resetHashRoute } from '../composables/useHashRoute'
import { resetAppContext, useAppContext } from '../composables/useAppContext'
import { resetMockScenario, setMockScenario } from '../mocks/report'
import { resetModuleMockScenario, setModuleMockScenario } from '../mocks/modules'

const plugins = { plugins: [ElementPlus] }

beforeEach(() => {
  resetMockScenario()
  resetModuleMockScenario()
  resetAppContext()
  resetHashRoute()
  // 同步一次 hashchange，把可能残留的异步事件影响消除。
  window.dispatchEvent(new Event('hashchange'))
  setMockScenario({ delayMs: 1 })
  setModuleMockScenario({ delayMs: 1 })
  document.body.innerHTML = ''
})

afterEach(() => {
  while (liveWrappers.length > 0) liveWrappers.pop()?.unmount()
  ElMessage.closeAll()
  resetMockScenario()
  resetModuleMockScenario()
  resetAppContext()
  resetHashRoute()
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

async function flush(ms = 80) {
  await new Promise((r) => setTimeout(r, ms))
  await nextTick()
  await nextTick()
}

const liveWrappers: VueWrapper[] = []

async function mountApp() {
  const wrapper = mount(App, { global: plugins })
  liveWrappers.push(wrapper as unknown as VueWrapper)
  await nextTick()
  return wrapper
}

function crumbsText(wrapper: { find: (s: string) => { text: () => string } }): string {
  return wrapper.find('[data-testid="breadcrumb"]').text()
}

function sideItem(wrapper: VueWrapper, label: string) {
  return wrapper.findAll('.side-menu .el-menu-item').find((i) => i.text().includes(label))
}

describe('统一布局与路由同步', () => {
  it('默认首屏为网站概况页，侧边栏展示 20 个报表菜单', async () => {
    const wrapper = await mountApp()
    await flush()
    const items = wrapper.findAll('.side-menu .el-menu-item')
    expect(items).toHaveLength(NAVIGATION_ITEMS.length)
    for (const item of NAVIGATION_ITEMS) {
      expect(items.map((i) => i.text())).toContain(item.label)
    }
    // 多子项菜单折叠为菜单组标题
    expect(wrapper.findAll('.side-menu .el-sub-menu__title')).toHaveLength(
      NAVIGATION_GROUPS.filter((g) => g.children.length > 1).length
    )
    expect(crumbsText(wrapper)).toContain('网站概况')
    expect(wrapper.find('[data-testid="view-overview"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('核心指标')
    expect(wrapper.find('.side-menu .el-menu-item.is-active').text()).toContain('网站概况')
  })

  it('点击侧边菜单切换页面，路由/面包屑/高亮同步', async () => {
    const wrapper = await mountApp()
    await flush()
    await sideItem(wrapper, '搜索词')?.trigger('click')
    await flush()
    expect(window.location.hash).toBe('#/source/searchword')
    expect(crumbsText(wrapper)).toContain('来源分析')
    expect(crumbsText(wrapper)).toContain('搜索词')
    expect(wrapper.find('.side-menu .el-menu-item.is-active').text()).toContain('搜索词')
    expect(wrapper.find('[data-testid="view-source-searchword"]').exists()).toBe(true)
  })

  it('浏览器 hash 变化同步回菜单与页面', async () => {
    const wrapper = await mountApp()
    await flush()
    window.location.hash = '#/visit/toppage'
    window.dispatchEvent(new Event('hashchange'))
    await flush()
    expect(wrapper.find('.side-menu .el-menu-item.is-active').text()).toContain('受访页面')
    expect(crumbsText(wrapper)).toContain('受访页面')
    expect(wrapper.find('[data-testid="view-visit-toppage"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('受访页面明细')
  })

  it('未知路由回退到网站概况页', async () => {
    const wrapper = await mountApp()
    await flush()
    window.location.hash = '#/no-such-page'
    window.dispatchEvent(new Event('hashchange'))
    await flush()
    expect(wrapper.find('[data-testid="view-overview"]').exists()).toBe(true)
    expect(wrapper.find('.side-menu .el-menu-item.is-active').text()).toContain('网站概况')
  })

  it('窄屏抽屉菜单可打开并导航，导航后自动关闭', async () => {
    const wrapper = await mountApp()
    await flush()
    // 平时抽屉不渲染（v-if，避免干扰报表抽屉测试）
    expect(wrapper.find('[data-testid="drawer-menu"]').exists()).toBe(false)
    await wrapper.find('[data-testid="menu-toggle"]').trigger('click')
    await flush(60)
    const drawerItems = wrapper.findAll('[data-testid="drawer-menu"] .el-menu-item')
    expect(drawerItems).toHaveLength(NAVIGATION_ITEMS.length)
    await drawerItems.find((el) => el.text().includes('转化概况'))?.trigger('click')
    await flush()
    expect(window.location.hash).toBe('#/trans/overview')
    expect(wrapper.find('[data-testid="view-conversion-overview"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('转化漏斗')
    expect(wrapper.find('[data-testid="drawer-menu"]').exists()).toBe(false)
  })
})

describe('顶部站点/日期上下文', () => {
  it('站点切换写入全局上下文，报表页自动重查', async () => {
    const wrapper = await mountApp()
    await flush()
    expect(wrapper.text()).toContain('网站概况')

    const siteSelect = wrapper.findAllComponents({ name: 'ElSelect' })[0]
    expect(siteSelect.exists()).toBe(true)
    siteSelect.vm.$emit('update:modelValue', '5503017')
    await flush()
    expect(useAppContext().siteId.value).toBe('5503017')
    // 重查后内容仍完整（mock 按站点种子变化但形状不变）
    expect(wrapper.text()).toContain('Top10 搜索词')
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('顶部日期预设同步到报表页筛选', async () => {
    const wrapper = await mountApp()
    await flush()
    window.location.hash = '#/source/searchword'
    window.dispatchEvent(new Event('hashchange'))
    await flush()
    const presetSelect = wrapper.findAllComponents({ name: 'ElSelect' })[1]
    presetSelect.vm.$emit('update:modelValue', 'last30')
    await flush()
    expect(useAppContext().rangePreset.value).toBe('last30')
    const filterStrip = wrapper.findComponent(ReportFilterStrip)
    expect(filterStrip.props('filters').rangePreset).toBe('last30')
    expect(wrapper.text()).toContain('搜索词明细')
  })
})

describe('报表页在布局内的通用交互', () => {
  it('错误重试：失败显示重试入口，恢复后展示数据', async () => {
    setModuleMockScenario({ mode: 'error', delayMs: 1 })
    const wrapper = await mountApp()
    await flush()
    expect(wrapper.text()).toContain('模块数据加载失败')
    setModuleMockScenario({ mode: 'ok', delayMs: 1 })
    await wrapper.findAll('button').find((b) => b.text() === '重试')?.trigger('click')
    await flush()
    expect(wrapper.text()).not.toContain('模块数据加载失败')
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('导出按钮触发 CSV 下载', async () => {
    const wrapper = await mountApp()
    await flush()
    const createObjectURL = vi.fn(() => 'blob:mock')
    const revokeObjectURL = vi.fn()
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL })
    const clickSpy = vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
    await wrapper.findAll('button').find((b) => b.text().includes('导出'))?.trigger('click')
    expect(createObjectURL).toHaveBeenCalled()
    expect(clickSpy).toHaveBeenCalled()
    clickSpy.mockRestore()
    expect(document.body.textContent).toContain('已导出当前结果')
  })
})
