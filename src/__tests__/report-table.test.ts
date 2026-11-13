import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { GeoRow, MetricValues } from '../types/report'
import { zeroMetricValues } from '../types/report'
import { exportColumns, exportRows } from '../utils/export'
import ReportTable from '../components/report/ReportTable.vue'

const plugins = { plugins: [ElementPlus] }

afterEach(() => {
  document.body.innerHTML = ''
})

function values(patch: Partial<MetricValues>): MetricValues {
  return { ...zeroMetricValues(), ...patch }
}

function makeRows(n: number): GeoRow[] {
  return Array.from({ length: n }, (_, i) => ({
    name: `地域${String(i + 1).padStart(2, '0')}`,
    code: `P${String(i + 1).padStart(2, '0')}`,
    values: values({ pv: (n - i) * 1000, uv: (n - i) * 700, ip: (n - i) * 600 })
  }))
}

async function mountTable(rows: GeoRow[], metrics: Array<'pv' | 'uv' | 'ip'> = ['pv', 'uv'], extra = {}) {
  const wrapper = mount(ReportTable, {
    props: { rows, metrics, loading: false, total: rows.length, subtitle: '筛选摘要', ...extra },
    global: plugins
  })
  // el-table 首次挂载经异步布局后才填充表头/单元格，等待其完成。
  await nextTick()
  await new Promise((r) => setTimeout(r, 30))
  await nextTick()
  return wrapper
}

describe('ReportTable', () => {
  it('列随指标变化', async () => {
    const wrapper = await mountTable(makeRows(3))
    expect(wrapper.text()).toContain('PV 浏览量')
    expect(wrapper.text()).toContain('UV 访客数')
    expect(wrapper.text()).not.toContain('IP 数')
    await wrapper.setProps({ metrics: ['pv', 'ip'] })
    expect(wrapper.text()).toContain('IP 数')
    expect(wrapper.text()).not.toContain('UV 访客数')
  })

  it('点击表头发出排序事件并重排行', async () => {
    const wrapper = await mountTable(makeRows(3))
    const pvHeader = wrapper.findAll('thead th')[1]
    await pvHeader.find('.sort-caret.ascending').trigger('click')
    expect(wrapper.emitted('sort-change')).toEqual([[{ prop: 'pv', order: 'ascending' }]])
    const firstCell = wrapper.find('tbody tr.el-table__row td .cell')
    expect(firstCell.text()).toBe('地域03')
  })

  it('分页切换发出事件并只显示当页行', async () => {
    const wrapper = await mountTable(makeRows(12))
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(10)
    const pages = wrapper.findAll('.el-pager li.number')
    await pages[1].trigger('click')
    expect(wrapper.emitted('page-change')).toEqual([[{ page: 2, pageSize: 10 }]])
    expect(wrapper.findAll('tbody tr.el-table__row')).toHaveLength(2)
  })

  it('空数据与加载状态', async () => {
    const empty = await mountTable([])
    expect(empty.text()).toContain('暂无明细数据')
    const loading = await mountTable([], ['pv', 'uv'], { loading: true })
    expect(loading.find('.el-skeleton').exists()).toBe(true)
  })

  it('有数据时加载显示遮罩', async () => {
    const wrapper = await mountTable(makeRows(3), ['pv', 'uv'], { loading: true })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('.el-loading-mask').exists()).toBe(true)
  })

  it('点击行发出 drilldown 并打开地域抽屉', async () => {
    const rows = makeRows(3)
    const wrapper = await mountTable(rows)
    await wrapper.find('tbody tr.el-table__row').trigger('click')
    expect(wrapper.emitted('drilldown')?.[0][0]).toMatchObject({ name: '地域01' })
    const drawer = wrapper.findComponent({ name: 'ElDrawer' })
    expect(drawer.props('modelValue')).toBe(true)
    expect(drawer.props('title')).toContain('地域01')
  })

  it('外部 revealCode 跳到所在页、高亮行并打开抽屉', async () => {
    const wrapper = await mountTable(makeRows(12))
    await wrapper.setProps({ revealCode: 'P12' })
    await new Promise((r) => setTimeout(r, 30))
    await nextTick()
    // 主表跳到第 2 页（限定主表选择器，排除抽屉内来源表）
    expect(wrapper.findAll('.report-table > div > .el-table tbody tr.el-table__row')).toHaveLength(2)
    const highlighted = wrapper.find('tbody tr.revealed-row')
    expect(highlighted.exists()).toBe(true)
    expect(highlighted.text()).toContain('地域12')
    const drawer = wrapper.findComponent({ name: 'ElDrawer' })
    expect(drawer.props('modelValue')).toBe(true)
    expect(drawer.props('title')).toContain('地域12')
    expect(wrapper.emitted('drilldown')?.[0][0]).toMatchObject({ code: 'P12' })
  })

  it('关闭抽屉发出 drawer-close', async () => {
    const wrapper = await mountTable(makeRows(3))
    await wrapper.find('tbody tr.el-table__row').trigger('click')
    expect(wrapper.findComponent({ name: 'ElDrawer' }).props('modelValue')).toBe(true)
    // 模拟抽屉关闭事件（绕开 el-drawer 内部焦点逻辑，只测本组件处理）
    await wrapper.findComponent({ name: 'ElDrawer' }).vm.$emit('update:modelValue', false)
    await nextTick()
    expect(wrapper.emitted('drawer-close')).toHaveLength(1)
    expect(wrapper.findComponent({ name: 'ElDrawer' }).props('modelValue')).toBe(false)
  })

  it('新查询数据到达时关闭抽屉，避免残留旧地域', async () => {
    const wrapper = await mountTable(makeRows(3))
    await wrapper.find('tbody tr.el-table__row').trigger('click')
    expect(wrapper.findComponent({ name: 'ElDrawer' }).props('modelValue')).toBe(true)
    await wrapper.setProps({ rows: makeRows(5), total: 5 })
    expect(wrapper.findComponent({ name: 'ElDrawer' }).props('modelValue')).toBe(false)
  })
})

describe('exportRows', () => {
  it('表头包含地域、PV、UV 及当前选择指标（去重）', () => {
    expect(exportColumns(['pv', 'uv'])).toEqual(['pv', 'uv'])
    expect(exportColumns(['ip', 'pv'])).toEqual(['pv', 'uv', 'ip'])
    const csv = exportRows(makeRows(1), ['pv', 'uv'])
    const header = csv.replace(/^﻿/, '').split('\r\n')[0]
    expect(header).toBe('地域,PV 浏览量,UV 访客数')
  })

  it('单元格含逗号/引号/换行时正确转义', () => {
    const rows: GeoRow[] = [
      { name: 'A"B,C\nD', code: 'X', values: values({ pv: 100, uv: 80 }) }
    ]
    const csv = exportRows(rows, ['pv', 'uv'])
    expect(csv).toContain('"A""B,C\nD",100,80')
  })

  it('输出带 UTF-8 BOM 且行数与数据一致', () => {
    const csv = exportRows(makeRows(2), ['pv'])
    expect(csv.startsWith('﻿')).toBe(true)
    expect(csv.split('\r\n')).toHaveLength(3)
  })
})
