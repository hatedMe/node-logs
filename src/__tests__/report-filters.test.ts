import { afterEach, describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ElementPlus from 'element-plus'
import type { ReportFilters as ReportFiltersState } from '../types/report'
import { MAX_METRICS } from '../types/report'
import { defaultFilters, useReport, validateFilters } from '../composables/useReport'
import { eachDay, getPresetRange } from '../utils/date'
import { MOCK_ERROR_MESSAGE, resetMockScenario, setMockScenario } from '../mocks/report'
import ReportFilters from '../components/report/ReportFilters.vue'

afterEach(() => {
  resetMockScenario()
  document.body.innerHTML = ''
})

function invalidRange(): ReportFiltersState {
  return { ...defaultFilters(), startDate: '2026-11-13', endDate: '2026-11-12' }
}

describe('筛选校验规则', () => {
  it('默认筛选合法', () => {
    expect(validateFilters(defaultFilters()).valid).toBe(true)
  })

  it('结束早于开始时阻止查询并说明原因', () => {
    const v = validateFilters(invalidRange())
    expect(v.valid).toBe(false)
    expect(v.errors[0]).toContain('不得早于')
  })

  it('对比日期段为空或重叠时阻止查询', () => {
    const empty = validateFilters({ ...defaultFilters(), compareEnabled: true })
    expect(empty.valid).toBe(false)
    expect(empty.errors[0]).toContain('对比日期段')

    const overlap = validateFilters({
      ...defaultFilters(),
      startDate: '2026-11-01',
      endDate: '2026-11-07',
      compareEnabled: true,
      compareStartDate: '2026-11-05',
      compareEndDate: '2026-11-10'
    })
    expect(overlap.valid).toBe(false)
    expect(overlap.errors[0]).toContain('重叠')
  })

  it('指标为空或超过 6 项时阻止查询', () => {
    expect(validateFilters({ ...defaultFilters(), metrics: [] }).valid).toBe(false)
    const over = validateFilters({
      ...defaultFilters(),
      metrics: ['pv', 'uv', 'ip', 'visits', 'avgPages', 'bounceRate', 'avgDuration']
    })
    expect(over.valid).toBe(false)
    expect(over.errors[0]).toContain(`${MAX_METRICS}`)
  })
})

describe('useReport 查询状态', () => {
  it('成功查询设置结果并结束 loading', async () => {
    setMockScenario({ delayMs: 1 })
    const r = useReport()
    const ok = await r.query()
    expect(ok).toBe(true)
    expect(r.loading.value).toBe(false)
    expect(r.error.value).toBeNull()
    expect(r.result.value).not.toBeNull()
    expect(r.hasData.value).toBe(true)
  })

  it('非法筛选阻止查询且不清空旧结果', async () => {
    setMockScenario({ delayMs: 1 })
    const r = useReport()
    await r.query()
    const first = r.result.value
    r.filters.value = invalidRange()
    const ok = await r.query()
    expect(ok).toBe(false)
    expect(r.error.value).toContain('不得早于')
    expect(r.result.value).toBe(first)
  })

  it('失败时清空旧结果并保留筛选条件', async () => {
    setMockScenario({ delayMs: 1 })
    const r = useReport()
    r.filters.value = { ...r.filters.value, source: 'search' }
    await r.query()
    expect(r.result.value).not.toBeNull()
    setMockScenario({ mode: 'error', delayMs: 1 })
    const ok = await r.query()
    expect(ok).toBe(false)
    expect(r.result.value).toBeNull()
    expect(r.error.value).toBe(MOCK_ERROR_MESSAGE)
    expect(r.filters.value.source).toBe('search')
    expect(r.hasData.value).toBe(false)
  })

  it('连续查询只采用最后一次结果', async () => {
    setMockScenario({ delayMs: 30 })
    const r = useReport()
    r.filters.value = { ...r.filters.value, source: 'all' }
    const p1 = r.query()
    r.filters.value = { ...r.filters.value, source: 'search' }
    const p2 = r.query()
    await Promise.all([p1, p2])
    expect(r.loading.value).toBe(false)
    expect(r.result.value?.queryKey).toContain('search')
  })

  it('reset 恢复默认筛选并重新查询', async () => {
    setMockScenario({ delayMs: 1 })
    const r = useReport()
    r.filters.value = { ...r.filters.value, source: 'search', geoLevel: 'country' }
    await r.reset()
    expect(r.filters.value).toEqual(defaultFilters())
    expect(r.result.value).not.toBeNull()
  })
})

describe('ReportFilters 组件', () => {
  function mountFilters(filters: ReportFiltersState = defaultFilters(), extra = {}) {
    return mount(ReportFilters, {
      props: { filters, loading: false, canDownload: true, ...extra },
      global: { plugins: [ElementPlus] }
    })
  }

  function lastUpdate(wrapper: ReturnType<typeof mountFilters>): ReportFiltersState {
    const emitted = wrapper.emitted('update:filters')
    expect(emitted).toBeTruthy()
    return emitted?.[emitted.length - 1][0] as ReportFiltersState
  }

  it('渲染日期预设、地域层级与 8 个指标选项', () => {
    const wrapper = mountFilters()
    const text = wrapper.text()
    for (const t of ['今天', '昨天', '最近 7 天', '最近 30 天', '自定义', '按省', '按国家']) {
      expect(text).toContain(t)
    }
    expect(
      wrapper.findComponent({ name: 'ElCheckboxGroup' }).findAll('input[type="checkbox"]')
    ).toHaveLength(8)
  })

  it('切换日期预设自动填充对应区间', async () => {
    const wrapper = mountFilters()
    const groups = wrapper.findAllComponents({ name: 'ElRadioGroup' })
    await groups[0].setValue('last30')
    const next = lastUpdate(wrapper)
    expect(next.rangePreset).toBe('last30')
    const expected = getPresetRange('last30')
    expect(next.startDate).toBe(expected.startDate)
    expect(next.endDate).toBe(expected.endDate)
    expect(eachDay(next.startDate, next.endDate)).toHaveLength(30)
  })

  it('手动修改日期后预设切换为自定义', async () => {
    const wrapper = mountFilters()
    const pickers = wrapper.findAllComponents({ name: 'ElDatePicker' })
    await pickers[0].setValue('2026-01-05')
    const next = lastUpdate(wrapper)
    expect(next.startDate).toBe('2026-01-05')
    expect(next.rangePreset).toBe('custom')
  })

  it('来源、访客、地域层级可交互', async () => {
    const wrapper = mountFilters()
    const selects = wrapper.findAllComponents({ name: 'ElSelect' })
    await selects[0].setValue('search')
    expect(lastUpdate(wrapper).source).toBe('search')
    await selects[1].setValue('new')
    expect(lastUpdate(wrapper).visitorType).toBe('new')
    const groups = wrapper.findAllComponents({ name: 'ElRadioGroup' })
    await groups[1].setValue('country')
    expect(lastUpdate(wrapper).geoLevel).toBe('country')
  })

  it('第 7 个指标不可选：守卫拦截且已选 6 项时其余禁用', async () => {
    const wrapper = mountFilters()
    const group = wrapper.findComponent({ name: 'ElCheckboxGroup' })
    const before = (wrapper.emitted('update:filters') ?? []).length
    await group.setValue(['pv', 'uv', 'ip', 'visits', 'avgPages', 'bounceRate', 'avgDuration'])
    expect((wrapper.emitted('update:filters') ?? []).length).toBe(before)

    const six: ReportFiltersState = {
      ...defaultFilters(),
      metrics: ['pv', 'uv', 'ip', 'visits', 'avgPages', 'bounceRate']
    }
    await wrapper.setProps({ filters: six })
    const boxes = wrapper.findAll('input[type="checkbox"]')
    const disabledUnchecked = boxes.filter((b) => {
      const input = b.element as HTMLInputElement
      return !input.checked && input.disabled
    })
    expect(disabledUnchecked).toHaveLength(2)
    expect(wrapper.text()).toContain('已选 6 项')
  })

  it('开启对比自动填充上一周期，再次提交合法', async () => {
    const wrapper = mountFilters()
    const sw = wrapper.findComponent({ name: 'ElSwitch' })
    await sw.setValue(true)
    const next = lastUpdate(wrapper)
    expect(next.compareEnabled).toBe(true)
    expect(next.compareStartDate).toBeTruthy()
    expect(next.compareEndDate).toBeTruthy()
    expect(validateFilters(next).valid).toBe(true)
  })

  it('合法时点击查询发出 submit，非法时阻止并显示校验信息', async () => {
    const wrapper = mountFilters()
    const buttons = wrapper.findAll('button')
    const queryBtn = buttons.find((b) => b.text().includes('查询'))
    expect(queryBtn).toBeTruthy()
    await queryBtn?.trigger('click')
    expect(wrapper.emitted('submit')).toHaveLength(1)

    await wrapper.setProps({ filters: invalidRange() })
    await queryBtn?.trigger('click')
    expect(wrapper.emitted('submit')).toHaveLength(1)
    expect(wrapper.text()).toContain('不得早于')
  })

  it('重置与下载按钮发出对应事件，无数据时下载禁用', async () => {
    const wrapper = mountFilters()
    const buttons = wrapper.findAll('button')
    const resetBtn = buttons.find((b) => b.text().includes('重置'))
    const downloadBtn = buttons.find((b) => b.text().includes('下载'))
    await resetBtn?.trigger('click')
    expect(wrapper.emitted('reset')).toHaveLength(1)
    await downloadBtn?.trigger('click')
    expect(wrapper.emitted('download')).toHaveLength(1)

    await wrapper.setProps({ canDownload: false })
    const disabled = wrapper
      .findAll('button')
      .find((b) => b.text().includes('下载'))
      ?.element as HTMLButtonElement
    expect(disabled.disabled).toBe(true)
  })
})
