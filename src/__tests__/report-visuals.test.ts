import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { GeoRow, MetricValues, ReportResult, TrendPoint } from '../types/report'
import { zeroMetricValues } from '../types/report'
import MetricCards from '../components/report/MetricCards.vue'
import TrendChart from '../components/report/TrendChart.vue'
import GeoView from '../components/report/GeoView.vue'

const plugins = { plugins: [ElementPlus] }

function values(patch: Partial<MetricValues>): MetricValues {
  return { ...zeroMetricValues(), ...patch }
}

function sampleTrend(withCompare = false): TrendPoint[] {
  return ['2026-11-10', '2026-11-11', '2026-11-12'].map((date, i) => ({
    date,
    values: values({ pv: 10000 + i * 1000, uv: 7000 + i * 500 }),
    compareValues: withCompare ? values({ pv: 9000 + i * 800, uv: 6300 + i * 400 }) : null
  }))
}

function sampleRows(): GeoRow[] {
  return [
    { name: '广东', code: 'P01', values: values({ pv: 20000, uv: 14000 }) },
    { name: '江苏', code: 'P02', values: values({ pv: 15000, uv: 10000 }) },
    { name: '浙江', code: 'P03', values: values({ pv: 12000, uv: 8000 }) }
  ]
}

function sampleResult(): ReportResult {
  return {
    summary: [
      { key: 'pv', value: 33000, compareValue: 30000, changeRate: 0.1 },
      { key: 'uv', value: 22500, compareValue: 23000, changeRate: -0.0217 }
    ],
    trend: sampleTrend(),
    geoRows: sampleRows(),
    total: 3,
    queryKey: 'test'
  }
}

describe('MetricCards', () => {
  it('默认展示 PV、UV 卡片与环比值', () => {
    const wrapper = mount(MetricCards, {
      props: { result: sampleResult(), metrics: ['pv', 'uv'] },
      global: plugins
    })
    const text = wrapper.text()
    expect(text).toContain('PV 浏览量')
    expect(text).toContain('33000')
    expect(text).toContain('UV 访客数')
    expect(text).toContain('环比 +10.00%')
    expect(text).toContain('环比 -2.17%')
  })

  it('只展示当前选中指标', () => {
    const wrapper = mount(MetricCards, {
      props: { result: sampleResult(), metrics: ['pv'] },
      global: plugins
    })
    expect(wrapper.text()).toContain('PV 浏览量')
    expect(wrapper.text()).not.toContain('UV 访客数')
  })

  it('无结果时显示空状态，加载中显示骨架', () => {
    const empty = mount(MetricCards, {
      props: { result: null, metrics: ['pv', 'uv'] },
      global: plugins
    })
    expect(empty.text()).toContain('暂无指标数据')
    const loading = mount(MetricCards, {
      props: { result: null, metrics: ['pv', 'uv'], loading: true },
      global: plugins
    })
    expect(loading.find('.el-skeleton').exists()).toBe(true)
  })
})

describe('TrendChart', () => {
  it('每个指标一条实线，图例点击显隐', async () => {
    const wrapper = mount(TrendChart, {
      props: { trend: sampleTrend(), metrics: ['pv', 'uv'] },
      global: plugins
    })
    expect(wrapper.findAll('polyline.trend-line:not(.compare-line)')).toHaveLength(2)
    const legends = wrapper.findAll('.legend-item')
    expect(legends).toHaveLength(2)
    await legends[0].trigger('click')
    expect(wrapper.findAll('polyline.trend-line:not(.compare-line)')).toHaveLength(1)
    await legends[0].trigger('click')
    expect(wrapper.findAll('polyline.trend-line:not(.compare-line)')).toHaveLength(2)
  })

  it('对比区间显示虚线并标注图例', () => {
    const wrapper = mount(TrendChart, {
      props: { trend: sampleTrend(true), metrics: ['pv', 'uv'], compareEnabled: true },
      global: plugins
    })
    const dashed = wrapper.findAll('polyline.compare-line')
    expect(dashed).toHaveLength(2)
    expect(dashed[0].attributes('stroke-dasharray')).toBe('5 4')
    expect(wrapper.text()).toContain('虚线：对比区间')
  })

  it('开启对比但无对比数据时不画虚线', () => {
    const wrapper = mount(TrendChart, {
      props: { trend: sampleTrend(false), metrics: ['pv'], compareEnabled: true },
      global: plugins
    })
    expect(wrapper.findAll('polyline.compare-line')).toHaveLength(0)
    expect(wrapper.text()).not.toContain('虚线：对比区间')
  })

  it('单天区间显示数据圆点', () => {
    const wrapper = mount(TrendChart, {
      props: { trend: sampleTrend().slice(0, 1), metrics: ['pv', 'uv'] },
      global: plugins
    })
    expect(wrapper.findAll('circle.single-dot')).toHaveLength(2)
  })

  it('百分比指标按自身峰值归一化，不再被 PV 压平', () => {
    const trend: TrendPoint[] = [
      { date: '2026-11-11', values: values({ pv: 10000, bounceRate: 0.5 }), compareValues: null },
      { date: '2026-11-12', values: values({ pv: 12000, bounceRate: 0.4 }), compareValues: null }
    ]
    const wrapper = mount(TrendChart, {
      props: { trend, metrics: ['pv', 'bounceRate'] },
      global: plugins
    })
    const line = wrapper.find('polyline.trend-line[data-metric="bounceRate"]')
    expect(line.exists()).toBe(true)
    const ys = (line.attributes('points') ?? '').split(' ').map((s) => Number(s.split(',')[1]))
    // 归一化后峰值点位于顶部（y=12）；若与 PV 共用单一量级会被压平到底部（y≈232）
    expect(ys[0]).toBeLessThan(50)
    expect(ys[1]).toBeGreaterThan(ys[0])
  })

  it('坐标轴与图例标明百分比归一化单位', () => {
    const wrapper = mount(TrendChart, {
      props: { trend: sampleTrend(), metrics: ['pv', 'uv'] },
      global: plugins
    })
    expect(wrapper.find('svg').text()).toContain('100%')
    expect(wrapper.text()).toContain('归一化')
  })

  it('图例按钮暴露 aria-pressed 且随显隐切换', async () => {
    const wrapper = mount(TrendChart, {
      props: { trend: sampleTrend(), metrics: ['pv', 'uv'] },
      global: plugins
    })
    const legends = wrapper.findAll('.legend-item')
    expect(legends[0].attributes('aria-pressed')).toBe('true')
    await legends[0].trigger('click')
    expect(legends[0].attributes('aria-pressed')).toBe('false')
  })

  it('图表提供读屏等效文本，tooltip 为 status 实时区', async () => {
    const wrapper = mount(TrendChart, {
      props: { trend: sampleTrend(), metrics: ['pv', 'uv'] },
      global: plugins
    })
    const label = wrapper.find('svg').attributes('aria-label') ?? ''
    expect(label).toContain('PV 浏览量')
    expect(label).toContain('10000')
    expect(label).toContain('归一化')
    await wrapper.findAll('.hover-dot')[0].trigger('mouseover')
    const tip = wrapper.find('.tooltip')
    expect(tip.attributes('role')).toBe('status')
    expect(tip.text()).toContain('2026-11-10')
  })

  it('数据点 hover 显示日期与指标 tooltip', async () => {
    const wrapper = mount(TrendChart, {
      props: { trend: sampleTrend(), metrics: ['pv', 'uv'] },
      global: plugins
    })
    expect(wrapper.find('.tooltip').exists()).toBe(false)
    await wrapper.findAll('.hover-dot')[1].trigger('mouseover')
    const tip = wrapper.find('.tooltip')
    expect(tip.exists()).toBe(true)
    expect(tip.text()).toContain('2026-11-11')
    expect(tip.text()).toContain('PV 浏览量')
  })

  it('无趋势数据时显示空状态', () => {
    const wrapper = mount(TrendChart, {
      props: { trend: [], metrics: ['pv', 'uv'] },
      global: plugins
    })
    expect(wrapper.text()).toContain('暂无趋势数据')
  })
})

describe('GeoView', () => {
  it('标题随地域层级同步并展示总量摘要', () => {
    const province = mount(GeoView, {
      props: { rows: sampleRows(), geoLevel: 'province' },
      global: plugins
    })
    expect(province.text()).toContain('地域排行（按省）')
    expect(province.text()).toContain('共 3 个地域')
    expect(province.text()).toContain('47000')
  })

  it('按国家切换标题', () => {
    const country = mount(GeoView, {
      props: { rows: sampleRows(), geoLevel: 'country' },
      global: plugins
    })
    expect(country.text()).toContain('地域排行（按国家）')
  })

  it('查看明细发出地域参数', async () => {
    const wrapper = mount(GeoView, {
      props: { rows: sampleRows(), geoLevel: 'province' },
      global: plugins
    })
    const buttons = wrapper.findAll('button')
    await buttons[0].trigger('click')
    expect(wrapper.emitted('view-detail')).toEqual([[{ name: '广东', code: 'P01' }]])
  })

  it('无地域数据时显示空状态', () => {
    const wrapper = mount(GeoView, {
      props: { rows: [], geoLevel: 'province' },
      global: plugins
    })
    expect(wrapper.text()).toContain('暂无地域数据')
  })
})
