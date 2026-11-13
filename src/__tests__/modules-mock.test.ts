import { afterEach, describe, expect, it } from 'vitest'
import type { ModuleQueryContext } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { toCsv } from '../utils/csv'
import {
  genAccount,
  genConversion,
  genCustomReport,
  genExport,
  genHeatmap,
  genOptimize,
  genOverview,
  genPromotion,
  genSource,
  genTraffic,
  genVisit,
  resetModuleMockScenario,
  setModuleMockScenario
} from '../mocks/modules'

function baseCtx(patch: Partial<ModuleQueryContext> = {}): ModuleQueryContext {
  return {
    siteId: '16847648',
    rangePreset: 'last7',
    startDate: '2026-11-06',
    endDate: '2026-11-12',
    source: 'all',
    device: 'all',
    keyword: '',
    ...patch
  }
}

afterEach(() => {
  resetModuleMockScenario()
})

describe('各模块 mock 形状', () => {
  it('11 个新模块生成器都返回非空卡片与表格', () => {
    const ctx = baseCtx()
    const overview = genOverview(ctx)
    expect(overview.cards.length).toBeGreaterThan(0)
    expect(overview.trend).toHaveLength(7)
    expect(overview.trend[0].date).toBe('2026-11-06')
    expect(overview.trend[0].values.pv).toBeGreaterThan(0)
    expect(overview.trend[0].compareValues).toBeNull()
    expect(overview.keywords.columns.length).toBeGreaterThan(0)
    expect(overview.keywords.rows.length).toBeGreaterThan(0)
    expect(overview.topSources.length).toBeGreaterThan(0)

    expect(genPromotion(ctx).funnel.length).toBeGreaterThan(0)
    expect(genPromotion(ctx).plans.rows.length).toBeGreaterThan(0)
    expect(genTraffic(ctx).trend).toHaveLength(7)
    expect(genTraffic(ctx).realtime.rows.length).toBeGreaterThan(0)
    expect(genSource(ctx).searchWords.rows.length).toBeGreaterThan(0)
    expect(genVisit(ctx).pages.rows.length).toBeGreaterThan(0)
    expect(genVisit(ctx).entries.length).toBeGreaterThan(0)
    expect(genConversion(ctx).funnel.length).toBeGreaterThan(0)
    expect(genConversion(ctx).orders.rows.length).toBeGreaterThan(0)
    expect(genOptimize(ctx).crawlErrors.rows.length).toBeGreaterThan(0)
    expect(genOptimize(ctx).slowPages.length).toBeGreaterThan(0)
    expect(genHeatmap(ctx).cells.length).toBeGreaterThan(0)
    expect(genHeatmap(ctx).elements.rows.length).toBeGreaterThan(0)
    expect(genExport(ctx).tasks.rows.length).toBeGreaterThan(0)
    expect(genExport(ctx).apiKeys.rows.length).toBeGreaterThan(0)
    expect(genAccount(ctx).users.rows.length).toBeGreaterThan(0)
  })

  it('漏斗与热力数据单调/归一化合理', () => {
    const funnel = genPromotion(baseCtx()).funnel
    for (let i = 1; i < funnel.length; i++) {
      expect(funnel[i].value).toBeLessThanOrEqual(funnel[i - 1].value)
    }
    for (const c of genHeatmap(baseCtx()).cells) {
      expect(c.intensity).toBeGreaterThanOrEqual(0)
      expect(c.intensity).toBeLessThanOrEqual(1)
    }
    expect(Math.max(...genHeatmap(baseCtx()).cells.map((c) => c.intensity))).toBe(1)
  })
})

describe('mock 确定性与条件敏感', () => {
  it('同条件返回稳定数据', () => {
    expect(genOverview(baseCtx())).toEqual(genOverview(baseCtx()))
    expect(genVisit(baseCtx())).toEqual(genVisit(baseCtx()))
  })

  it('站点/来源/设备/日期变化时数据变化', () => {
    const a = JSON.stringify(genOverview(baseCtx()))
    expect(JSON.stringify(genOverview(baseCtx({ siteId: '5503017' })))).not.toBe(a)
    expect(JSON.stringify(genOverview(baseCtx({ source: 'search' })))).not.toBe(a)
    expect(JSON.stringify(genOverview(baseCtx({ device: 'mobile' })))).not.toBe(a)
    expect(JSON.stringify(genOverview(baseCtx({ startDate: '2026-10-01', endDate: '2026-10-07' })))).not.toBe(a)
  })

  it('关键词做真实子串过滤，无命中时返回空行', () => {
    const hit = genSource(baseCtx({ keyword: 'seo' }))
    expect(hit.searchWords.rows.length).toBeGreaterThan(0)
    for (const r of hit.searchWords.rows) {
      expect(`${r.word} ${r.engine}`.toLowerCase()).toContain('seo')
    }
    expect(genSource(baseCtx({ keyword: 'zzz-no-such-word' })).searchWords.rows).toEqual([])
    expect(genVisit(baseCtx({ keyword: 'zzz-no-such-word' })).pages.rows).toEqual([])
  })

  it('不同受访页面的热力数据不同', () => {
    const ctx = baseCtx()
    expect(genHeatmap(ctx, '/index.html').cells).not.toEqual(genHeatmap(ctx, '/price').cells)
  })
})

describe('自定义报告生成', () => {
  it('维度与指标决定表格列，行数可控', () => {
    const r = genCustomReport(baseCtx(), ['地域', '设备'], ['pv', 'conv'])
    expect(r.table.columns.map((c) => c.prop)).toEqual(['地域', '设备', 'pv', 'conv'])
    expect(r.table.rows.length).toBeGreaterThan(0)
    expect(r.table.rows.length).toBeLessThanOrEqual(12)
  })

  it('空维度/空指标时回退到默认 地域×PV', () => {
    const r = genCustomReport(baseCtx(), [], [])
    expect(r.table.columns.map((c) => c.prop)).toEqual(['地域', 'pv'])
  })
})

describe('empty 场景', () => {
  it('empty 模式清空卡片/排行/漏斗/热力与嵌套表格行，保留列头与页面选项', () => {
    setModuleMockScenario({ mode: 'empty' })
    const ctx = baseCtx()
    const o = genOverview(ctx)
    expect(o.cards).toEqual([])
    expect(o.topSources).toEqual([])
    expect(o.topPages).toEqual([])
    expect(o.keywords.rows).toEqual([])
    expect(o.keywords.columns.length).toBeGreaterThan(0)
    expect(o.trend).toEqual([])
    expect(genTraffic(ctx).realtime.rows).toEqual([])
    expect(genConversion(ctx).funnel).toEqual([])
    expect(genHeatmap(ctx).cells).toEqual([])
    expect(genHeatmap(ctx).elements.rows).toEqual([])
    expect(genHeatmap(ctx).pages.length).toBeGreaterThan(0)
  })
})

describe('单元格格式化', () => {
  it('百分比/时长/浮点/整数/非法值', () => {
    expect(formatCellValue('percent', 0.1234)).toBe('12.34%')
    expect(formatCellValue('duration', 125)).toBe('2:05')
    expect(formatCellValue('float2', 1.236)).toBe('1.24')
    expect(formatCellValue('int', 9.6)).toBe('10')
    expect(formatCellValue('int', '原文')).toBe('原文')
    expect(formatCellValue('percent', Number.NaN)).toBe('-')
  })
})

describe('通用 CSV 导出', () => {
  it('带 BOM、CRLF 换行，逗号与引号正确转义', () => {
    const csv = toCsv(['名称', '数值'], [['a,b', 1], ['c"d', 2]])
    expect(csv.charCodeAt(0)).toBe(0xfeff)
    expect(csv).toContain('\r\n')
    expect(csv).toContain('"a,b",1')
    expect(csv).toContain('"c""d",2')
  })
})
