import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import App from '../App.vue'
import FilterBar from '../components/common/FilterBar.vue'
import { MENU_ITEMS } from '../types/navigation'
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

describe('统一布局与路由同步', () => {
  it('默认首屏为访客分析 PV/UV 页，侧边栏展示 12 个菜单', async () => {
    const wrapper = await mountApp()
    await flush()
    const items = wrapper.findAll('.side-menu .el-menu-item')
    expect(items).toHaveLength(12)
    for (const m of MENU_ITEMS) {
      expect(items.map((i) => i.text())).toContain(m.label)
    }
    expect(crumbsText(wrapper)).toContain('访客分析')
    expect(wrapper.text()).toContain('PV/UV 统计报表')
    expect(wrapper.find('.side-menu .el-menu-item.is-active').text()).toContain('访客分析')
  })

  it('点击侧边菜单切换页面，路由/面包屑/高亮同步', async () => {
    const wrapper = await mountApp()
    await flush()
    const target = wrapper
      .findAll('.side-menu .el-menu-item')
      .find((i) => i.text().includes('网站概况'))
    await target?.trigger('click')
    await flush()
    expect(window.location.hash).toBe('#/overview')
    expect(crumbsText(wrapper)).toContain('网站概况')
    expect(wrapper.find('.side-menu .el-menu-item.is-active').text()).toContain('网站概况')
    expect(wrapper.text()).toContain('TOP 来源')
    expect(wrapper.find('[data-testid="view-overview"]').exists()).toBe(true)
  })

  it('浏览器 hash 变化同步回菜单与页面', async () => {
    const wrapper = await mountApp()
    await flush()
    window.location.hash = '#/traffic'
    window.dispatchEvent(new Event('hashchange'))
    await flush()
    expect(wrapper.find('.side-menu .el-menu-item.is-active').text()).toContain('流量分析')
    expect(crumbsText(wrapper)).toContain('流量分析')
    expect(wrapper.text()).toContain('实时访客')
  })

  it('未知路由回退到访客分析页', async () => {
    const wrapper = await mountApp()
    await flush()
    window.location.hash = '#/no-such-page'
    window.dispatchEvent(new Event('hashchange'))
    await flush()
    expect(wrapper.text()).toContain('PV/UV 统计报表')
    expect(wrapper.find('.side-menu .el-menu-item.is-active').text()).toContain('访客分析')
  })

  it('窄屏抽屉菜单可打开并导航，导航后自动关闭', async () => {
    const wrapper = await mountApp()
    await flush()
    // 平时抽屉不渲染（v-if，避免干扰报表抽屉测试）
    expect(wrapper.find('[data-testid="drawer-menu"]').exists()).toBe(false)
    await wrapper.find('[data-testid="menu-toggle"]').trigger('click')
    await flush(60)
    const drawerItems = wrapper.findAll('[data-testid="drawer-menu"] .el-menu-item')
    expect(drawerItems).toHaveLength(12)
    await drawerItems.find((el) => el.text().includes('转化分析'))?.trigger('click')
    await flush()
    expect(window.location.hash).toBe('#/conversion')
    expect(wrapper.text()).toContain('转化漏斗')
    expect(wrapper.find('[data-testid="drawer-menu"]').exists()).toBe(false)
  })
})

describe('顶部站点/日期上下文', () => {
  it('站点切换写入全局上下文，模块页自动重查', async () => {
    const wrapper = await mountApp()
    await flush()
    window.location.hash = '#/overview'
    window.dispatchEvent(new Event('hashchange'))
    await flush()
    expect(wrapper.text()).toContain('网站概况')

    const siteSelect = wrapper.findAllComponents({ name: 'ElSelect' })[0]
    expect(siteSelect.exists()).toBe(true)
    siteSelect.vm.$emit('update:modelValue', '5503017')
    await flush()
    expect(useAppContext().siteId.value).toBe('5503017')
    // 重查后内容仍完整（mock 按站点种子变化但形状不变）
    expect(wrapper.text()).toContain('TOP 搜索词')
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('顶部日期预设同步到模块页筛选', async () => {
    const wrapper = await mountApp()
    await flush()
    window.location.hash = '#/source'
    window.dispatchEvent(new Event('hashchange'))
    await flush()
    const presetSelect = wrapper.findAllComponents({ name: 'ElSelect' })[1]
    presetSelect.vm.$emit('update:modelValue', 'last30')
    await flush()
    expect(useAppContext().rangePreset.value).toBe('last30')
    const filterBar = wrapper.findComponent(FilterBar)
    expect(filterBar.props('filters').rangePreset).toBe('last30')
    expect(wrapper.text()).toContain('搜索词明细')
  })
})

describe('模块页在布局内的通用交互', () => {
  it('错误重试：失败显示重试入口，恢复后展示数据', async () => {
    setModuleMockScenario({ mode: 'error', delayMs: 1 })
    const wrapper = await mountApp()
    await flush()
    window.location.hash = '#/overview'
    window.dispatchEvent(new Event('hashchange'))
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
    window.location.hash = '#/overview'
    window.dispatchEvent(new Event('hashchange'))
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
