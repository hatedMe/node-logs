import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import type { VueWrapper } from '@vue/test-utils'
import ElementPlus, { ElMessage } from 'element-plus'
import OverviewView from '../views/OverviewView.vue'
import PromotionView from '../views/PromotionView.vue'
import TrafficView from '../views/TrafficView.vue'
import SourceView from '../views/SourceView.vue'
import VisitView from '../views/VisitView.vue'
import ConversionView from '../views/ConversionView.vue'
import VisitorView from '../views/VisitorView.vue'
import OptimizeView from '../views/OptimizeView.vue'
import HeatmapView from '../views/HeatmapView.vue'
import CustomView from '../views/CustomView.vue'
import ExportView from '../views/ExportView.vue'
import AccountView from '../views/AccountView.vue'
import { resetMockScenario, setMockScenario } from '../mocks/report'
import { resetModuleMockScenario, setModuleMockScenario } from '../mocks/modules'
import { resetAppContext } from '../composables/useAppContext'

const plugins = { plugins: [ElementPlus] }

// 跟踪挂载的 wrapper，用例结束即卸载，避免 Element Plus 内部异步回调
// （popper 定位、消息定时器）在环境销毁后触发导致偶发 Unhandled Errors。
const liveWrappers: VueWrapper[] = []

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function mountTracked(comp: any): VueWrapper {
  const w = mount(comp, { global: plugins })
  liveWrappers.push(w as unknown as VueWrapper)
  return w as unknown as VueWrapper
}

async function flush(ms = 60) {
  await new Promise((r) => setTimeout(r, ms))
  await nextTick()
  await nextTick()
}

beforeEach(() => {
  resetMockScenario()
  resetModuleMockScenario()
  resetAppContext()
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
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})
describe('各菜单页面渲染', () => {
  const cases: Array<[string, unknown, string[]]> = [
    ['网站概况', OverviewView, ['网站概况', 'PV/UV 趋势', 'TOP 来源', 'TOP 搜索词']],
    ['百度推广', PromotionView, ['百度推广', '推广转化漏斗', '推广计划效果']],
    ['流量分析', TrafficView, ['流量分析', 'PV/UV 趋势', '实时访客', '受访页面 TOP']],
    ['来源分析', SourceView, ['来源分析', '来源类型排行', '搜索词明细']],
    ['访问分析', VisitView, ['访问分析', '受访页面明细', '入口页面 TOP']],
    ['转化分析', ConversionView, ['转化分析', '转化漏斗', '订单明细']],
    ['访客分析', VisitorView, ['PV/UV 统计报表', '地域明细']],
    ['优化分析', OptimizeView, ['优化分析', '抓取异常明细', '慢页面 TOP']],
    ['热力图', HeatmapView, ['热力图与点击分析', '点击热图', '点击元素排行']],
    ['自定义报告', CustomView, ['自定义报告', '报表结果', '已保存报表']],
    ['导出开放平台', ExportView, ['数据导出与开放平台', '导出任务', 'API 密钥', '接入文档']],
    ['账户权限', AccountView, ['账户与权限管理', '成员列表']]
  ]

  for (const [name, comp, markers] of cases) {
    it(`${name}页展示标题与核心区块`, async () => {
      const wrapper = mountTracked(comp)
      await flush()
      const text = wrapper.text()
      for (const m of markers as string[]) {
        expect(text).toContain(m)
      }
      // 通用交互：查询/重置/导出按钮都存在
      const buttons = wrapper.findAll('button').map((b) => b.text())
      expect(buttons.some((t) => t.includes('查询'))).toBe(true)
      expect(buttons.some((t) => t.includes('重置'))).toBe(true)
    })
  }
})

describe('模块页通用状态', () => {
  it('失败显示错误与重试入口，重试成功后恢复', async () => {
    setModuleMockScenario({ mode: 'error', delayMs: 1 })
    const wrapper = mountTracked(OverviewView)
    await flush()
    expect(wrapper.text()).toContain('模块数据加载失败')
    setModuleMockScenario({ mode: 'ok', delayMs: 1 })
    await wrapper.findAll('button').find((b) => b.text() === '重试')?.trigger('click')
    await flush()
    expect(wrapper.text()).not.toContain('模块数据加载失败')
    expect(wrapper.findAll('tbody tr.el-table__row').length).toBeGreaterThan(0)
  })

  it('空结果显示空状态且导出禁用', async () => {
    setModuleMockScenario({ mode: 'empty', delayMs: 1 })
    const wrapper = mountTracked(TrafficView)
    await flush()
    const text = wrapper.text()
    expect(text).toContain('暂无指标数据')
    expect(text).toContain('暂无明细数据')
    const exportBtn = wrapper
      .findAll('button')
      .find((b) => b.text().includes('导出'))?.element as HTMLButtonElement
    expect(exportBtn.disabled).toBe(true)
  })

  it('非法日期阻止提交并提示', async () => {
    const wrapper = mountTracked(SourceView)
    await flush()
    // 通过子组件回传非法日期（模拟用户手工改到非法范围）
    const filterBar = wrapper.findComponent({ name: 'FilterBar' })
    filterBar.vm.$emit('update:filters', {
      rangePreset: 'custom',
      startDate: '2026-11-12',
      endDate: '2026-11-06',
      source: 'all',
      device: 'all',
      keyword: ''
    })
    await nextTick()
    await wrapper.findAll('button').find((b) => b.text() === '查询')?.trigger('click')
    await nextTick()
    expect(wrapper.text()).toContain('日期范围无效')
  })
})

describe('各页特色交互', () => {
  it('热力图：点击色块打开明细，受访页面可切换', async () => {
    const wrapper = mountTracked(HeatmapView)
    await flush()
    expect(wrapper.findAll('.heat-cell').length).toBeGreaterThan(0)
    await wrapper.findAll('.heat-cell')[0].trigger('click')
    await nextTick()
    const drawers = wrapper.findAllComponents({ name: 'ElDrawer' })
    expect(drawers.some((d) => d.props('modelValue') === true)).toBe(true)
    // 第三个选择器是受访页面（前两个是来源/设备）
    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    selects[selects.length - 1].vm.$emit('update:modelValue', '/price')
    await flush()
    expect(selects[selects.length - 1].props('modelValue')).toBe('/price')
    expect(wrapper.findAll('.heat-cell').length).toBeGreaterThan(0)
  })

  it('自定义报告：保存组合新增一行，删除可移除', async () => {
    const wrapper = mountTracked(CustomView)
    await flush()
    await wrapper.findAll('button').find((b) => b.text() === '保存当前组合')?.trigger('click')
    await nextTick()
    expect(wrapper.text()).toContain('自定义报表2')
    const before = wrapper.findAll('button').filter((b) => b.text() === '删除').length
    await wrapper.findAll('button').find((b) => b.text() === '删除')?.trigger('click')
    await nextTick()
    const after = wrapper.findAll('button').filter((b) => b.text() === '删除').length
    expect(after).toBe(before - 1)
  })

  it('导出开放平台：新建任务/刷新状态/新建密钥', async () => {
    const wrapper = mountTracked(ExportView)
    await flush()
    await wrapper.findAll('button').find((b) => b.text() === '新建任务')?.trigger('click')
    await nextTick()
    expect(wrapper.text()).toContain('手动导出任务1')
    expect(wrapper.text()).toContain('排队中')
    await wrapper.findAll('button').find((b) => b.text() === '刷新状态')?.trigger('click')
    await nextTick()
    // 刷新后新建任务变为已完成（预置行中仍有处理中/排队中样本）
    await wrapper.findAll('button').find((b) => b.text() === '新建密钥')?.trigger('click')
    await nextTick()
    expect(wrapper.text()).toContain('新密钥1')
  })

  it('账户权限：角色筛选过滤成员，添加成员写入表格', async () => {
    const wrapper = mountTracked(AccountView)
    await flush()
    expect(wrapper.text()).toContain('li.si')
    wrapper.findAllComponents({ name: 'ElSelect' })[0].vm.$emit('update:modelValue', '管理员')
    await nextTick()
    await nextTick()
    expect(wrapper.text()).toContain('zhang.san')
    expect(wrapper.text()).not.toContain('li.si')

    // 打开添加成员对话框（ teleported 到 body，用 document 查询）
    wrapper.findAllComponents({ name: 'ElSelect' })[0].vm.$emit('update:modelValue', '全部角色')
    await wrapper.findAll('button').find((b) => b.text() === '添加成员')?.trigger('click')
    await flush(60)
    // 对话框内容 inline 渲染在 wrapper 内（非 body），直接在 wrapper 中操作。
    const input = wrapper.find('.el-dialog .el-input__inner')
    expect(input.exists()).toBe(true)
    await input.setValue('new.member')
    const confirm = wrapper
      .findAll('.el-dialog__footer button')
      .find((b) => b.text().includes('确定'))
    await confirm?.trigger('click')
    await flush()
    expect(wrapper.text()).toContain('new.member')
  })

  it('表格行点击打开详情抽屉（以访问分析为例）', async () => {
    const wrapper = mountTracked(VisitView)
    await flush()
    await wrapper.find('tbody tr.el-table__row').trigger('click')
    await nextTick()
    const drawers = wrapper.findAllComponents({ name: 'ElDrawer' })
    const opened = drawers.find((d) => d.props('modelValue') === true)
    expect(opened).toBeTruthy()
    expect(String(opened?.props('title'))).toContain('页面明细')
  })
})
