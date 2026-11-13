import { afterEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import RankList from '../components/common/RankList.vue'
import FunnelSteps from '../components/common/FunnelSteps.vue'
import HeatGrid from '../components/common/HeatGrid.vue'
import PageTable from '../components/common/PageTable.vue'
import type { ModuleFilters, TableColumn } from '../types/modules'

const plugins = { plugins: [ElementPlus] }

function baseFilters(): ModuleFilters {
  return {
    rangePreset: 'last7',
    startDate: '2026-11-06',
    endDate: '2026-11-12',
    source: 'all',
    device: 'all',
    keyword: ''
  }
}

afterEach(() => {
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('FilterBar', () => {
  it('日期预设切换回填起止日期并向上传递', async () => {
    const wrapper = mount(FilterBar, { props: { filters: baseFilters() }, global: plugins })
    await wrapper.findComponent({ name: 'ElRadioGroup' }).setValue('today')
    const emitted = wrapper.emitted('update:filters')
    expect(emitted).toBeTruthy()
    const last = emitted?.[emitted.length - 1][0] as ModuleFilters
    expect(last.rangePreset).toBe('today')
    expect(last.startDate).toBe(last.endDate)
  })

  it('非法日期范围阻止提交并提示', async () => {
    const wrapper = mount(FilterBar, {
      props: { filters: { ...baseFilters(), startDate: '2026-11-12', endDate: '2026-11-06' } },
      global: plugins
    })
    await wrapper.findAll('button').find((b) => b.text() === '查询')?.trigger('click')
    await nextTick()
    expect(wrapper.emitted('submit')).toBeFalsy()
    expect(wrapper.text()).toContain('日期范围无效')
  })

  it('来源/设备/关键词变更向上传递，查询重置导出正常触发', async () => {
    const wrapper = mount(FilterBar, {
      props: { filters: baseFilters(), showKeyword: true, canExport: true },
      global: plugins
    })
    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    selects[0].vm.$emit('update:modelValue', 'search')
    selects[1].vm.$emit('update:modelValue', 'mobile')
    await nextTick()
    const emitted = wrapper.emitted('update:filters') ?? []
    expect(emitted.some((a) => (a[0] as ModuleFilters).source === 'search')).toBe(true)
    expect(emitted.some((a) => (a[0] as ModuleFilters).device === 'mobile')).toBe(true)
    await wrapper.findAll('button').find((b) => b.text() === '查询')?.trigger('click')
    await wrapper.findAll('button').find((b) => b.text() === '重置')?.trigger('click')
    await wrapper.findAll('button').find((b) => b.text() === '导出 CSV')?.trigger('click')
    expect(wrapper.emitted('submit')).toBeTruthy()
    expect(wrapper.emitted('reset')).toBeTruthy()
    expect(wrapper.emitted('export')).toBeTruthy()
  })
})

describe('StatCards', () => {
  it('渲染指标卡片与环比颜色，空数据展示空状态', () => {
    const cards = [
      { key: 'a', label: '指标A', value: '123', change: '+1.00%', up: true as const },
      { key: 'b', label: '指标B', value: '45', change: '-2.00%', up: false as const }
    ]
    const wrapper = mount(StatCards, { props: { cards }, global: plugins })
    expect(wrapper.text()).toContain('指标A')
    expect(wrapper.text()).toContain('123')
    expect(wrapper.find('.stat-change.up').exists()).toBe(true)
    expect(wrapper.find('.stat-change.down').exists()).toBe(true)
    const empty = mount(StatCards, { props: { cards: [] }, global: plugins })
    expect(empty.text()).toContain('暂无指标数据')
  })
})

describe('RankList', () => {
  const items = [
    { name: '广东', value: 100, display: '100', share: 50 },
    { name: '江苏', value: 60, display: '60', share: 30 }
  ]

  it('渲染排行条与查看明细抽屉', async () => {
    const wrapper = mount(RankList, { props: { title: '地域排行', items }, global: plugins })
    expect(wrapper.text()).toContain('地域排行')
    expect(wrapper.findAll('.rank-item')).toHaveLength(2)
    await wrapper.findAll('button').find((b) => b.text() === '查看明细')?.trigger('click')
    await nextTick()
    const drawer = wrapper.findComponent({ name: 'ElDrawer' })
    expect(drawer.props('modelValue')).toBe(true)
    expect(String(drawer.props('title'))).toContain('广东')
    expect(wrapper.emitted('select')).toBeTruthy()
  })

  it('空数据展示空状态', () => {
    const wrapper = mount(RankList, { props: { title: '排行', items: [] }, global: plugins })
    expect(wrapper.text()).toContain('暂无排行数据')
  })
})

describe('FunnelSteps', () => {
  it('渲染漏斗步骤与转化率，空数据展示空状态', () => {
    const steps = [
      { label: '点击', value: 100, display: '100', rate: '100%' },
      { label: '转化', value: 10, display: '10', rate: '10.00%' }
    ]
    const wrapper = mount(FunnelSteps, { props: { title: '漏斗', steps }, global: plugins })
    expect(wrapper.text()).toContain('转化')
    expect(wrapper.text()).toContain('10.00%')
    const empty = mount(FunnelSteps, { props: { title: '漏斗', steps: [] }, global: plugins })
    expect(empty.text()).toContain('暂无漏斗数据')
  })
})

describe('HeatGrid', () => {
  it('渲染热力色块，点击打开明细抽屉', async () => {
    const cells = [
      { label: '按钮A', value: 100, display: '100', intensity: 1 },
      { label: '按钮B', value: 10, display: '10', intensity: 0.1 }
    ]
    const wrapper = mount(HeatGrid, { props: { cells }, global: plugins })
    expect(wrapper.findAll('.heat-cell')).toHaveLength(2)
    await wrapper.findAll('.heat-cell')[1].trigger('click')
    await nextTick()
    const drawer = wrapper.findComponent({ name: 'ElDrawer' })
    expect(drawer.props('modelValue')).toBe(true)
    expect(String(drawer.props('title'))).toContain('按钮B')
  })
})

describe('PageTable', () => {
  const columns: TableColumn[] = [
    { prop: 'name', label: '名称', sortable: true },
    { prop: 'pv', label: 'PV', sortable: true, align: 'right', format: 'int' },
    { prop: 'rate', label: '比率', sortable: true, align: 'right', format: 'percent' }
  ]
  const rows = Array.from({ length: 25 }, (_, i) => ({
    name: `行${String(i + 1).padStart(2, '0')}`,
    pv: (i + 1) * 10,
    rate: 0.01 * (i + 1)
  }))

  async function flush(ms = 40) {
    await new Promise((r) => setTimeout(r, ms))
    await nextTick()
    await nextTick()
  }

  it('排序/分页/行点击抽屉联动', async () => {
    const wrapper = mount(PageTable, { props: { rows, columns }, global: plugins })
    await flush()
    // 默认第一页 10 行
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(10)

    // PV 降序：第 1 行应为最大值行
    const table = wrapper.findComponent({ name: 'ElTable' })
    table.vm.$emit('sort-change', { prop: 'pv', order: 'descending', column: {} })
    await flush()
    expect(wrapper.find('tbody tr.el-table__row').text()).toContain('行25')

    // 翻到第 2 页
    const pager = wrapper.findComponent({ name: 'ElPagination' })
    pager.vm.$emit('current-change', 2)
    await flush()
    expect(wrapper.find('tbody tr.el-table__row').text()).toContain('行15')

    // 行点击打开抽屉，标题含行名
    await wrapper.find('tbody tr.el-table__row').trigger('click')
    await nextTick()
    const drawer = wrapper.findComponent({ name: 'ElDrawer' })
    expect(drawer.props('modelValue')).toBe(true)
    expect(String(drawer.props('title'))).toContain('行15')
  })

  it('空数据展示空状态', () => {
    const wrapper = mount(PageTable, { props: { rows: [], columns }, global: plugins })
    expect(wrapper.text()).toContain('暂无明细数据')
  })
})
