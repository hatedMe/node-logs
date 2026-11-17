// 各统计模块的 mock 数据生成器（纯函数）：新模块模拟数据只能出现在本文件与
// mocks/reports.ts（20 个还原报表页）里，其余地方禁止散写假数据。
// 规则：种子 = 模块名 + 查询上下文（站点/日期/来源/设备/关键词），
// 同条件稳定返回，不同条件稳定变化；关键词做真实子串过滤。

import type {
  AccountData,
  ConversionData,
  CustomMetric,
  CustomReportData,
  ExportData,
  FunnelStep,
  HeatCell,
  HeatmapData,
  ModuleQueryContext,
  OptimizeData,
  OverviewData,
  PromotionData,
  RankItem,
  SourceData,
  StatCard,
  TableData,
  TrafficData,
  VisitData
} from '../types/modules'
import { floatBetween, intBetween, rngFor } from './seed'
import { CUSTOM_METRIC_LABELS } from '../types/modules'
import type { MetricValues, TrendPoint } from '../types/report'
import type { ReportPageData } from '../types/reports'
import { eachDay } from '../utils/date'

export const MODULE_MOCK_ERROR_MESSAGE = '模块数据加载失败，请重试'

export type ModuleMockMode = 'ok' | 'empty' | 'error'

interface ModuleMockScenario {
  mode: ModuleMockMode
  delayMs: number
}

const scenario: ModuleMockScenario = { mode: 'ok', delayMs: 15 }

/** 测试/演示用：切换新模块 mock 场景（正常/空结果/失败）与延迟。 */
export function setModuleMockScenario(patch: Partial<ModuleMockScenario>): void {
  Object.assign(scenario, patch)
}

export function resetModuleMockScenario(): void {
  scenario.mode = 'ok'
  scenario.delayMs = 15
}

export function getModuleMockMode(): ModuleMockMode {
  return scenario.mode
}

export function getModuleMockDelay(): number {
  return scenario.delayMs
}

const SOURCE_SCALE: Record<string, number> = {
  all: 1,
  direct: 0.32,
  search: 0.41,
  external: 0.18,
  internal: 0.22,
  custom: 0.12
}

const DEVICE_SCALE: Record<string, number> = { all: 1, pc: 0.62, mobile: 0.38 }

export function scaleOf(ctx: ModuleQueryContext): number {
  return (SOURCE_SCALE[ctx.source] ?? 1) * (DEVICE_SCALE[ctx.device] ?? 1)
}

export function seedKey(module: string, ctx: ModuleQueryContext): string {
  return [module, ctx.siteId, ctx.startDate, ctx.endDate, ctx.source, ctx.device].join('|')
}

export function fmtInt(n: number): string {
  return String(Math.max(0, Math.round(n)))
}

export function fmtPct(ratio: number): string {
  return `${(ratio * 100).toFixed(2)}%`
}

function changeText(rng: () => number): { change: string; up: boolean } {
  const v = (rng() - 0.45) * 0.4
  const up = v >= 0
  return { change: `${up ? '+' : ''}${(v * 100).toFixed(2)}%`, up }
}

export function card(
  rng: () => number,
  key: string,
  label: string,
  value: string
): StatCard {
  const { change, up } = changeText(rng)
  return { key, label, value, change, up }
}

function isTableData(v: unknown): v is { columns: unknown[]; rows: unknown[] } {
  if (typeof v !== 'object' || v === null) return false
  const o = v as Record<string, unknown>
  return Array.isArray(o.columns) && Array.isArray(o.rows)
}

/**
 * empty 场景：清空一层数组字段，并清空嵌套表格的 rows（保留列头）。
 * 各模块数据都是扁平结构（cards/rank/table），一层处理即够。
 * 用重载而非泛型：调用点按重载签名做上下文类型推断，columns 的
 * align/format 字面量不会被拓宽为 string。
 */
export function applyEmptyMode(data: OverviewData): OverviewData
export function applyEmptyMode(data: PromotionData): PromotionData
export function applyEmptyMode(data: TrafficData): TrafficData
export function applyEmptyMode(data: SourceData): SourceData
export function applyEmptyMode(data: VisitData): VisitData
export function applyEmptyMode(data: ConversionData): ConversionData
export function applyEmptyMode(data: OptimizeData): OptimizeData
export function applyEmptyMode(data: HeatmapData): HeatmapData
export function applyEmptyMode(data: CustomReportData): CustomReportData
export function applyEmptyMode(data: ExportData): ExportData
export function applyEmptyMode(data: AccountData): AccountData
export function applyEmptyMode(data: ReportPageData): ReportPageData
export function applyEmptyMode(data: object): object {
  if (scenario.mode !== 'empty') return data
  const out: Record<string, unknown> = { ...(data as Record<string, unknown>) }
  for (const k of Object.keys(out)) {
    const v = out[k]
    if (Array.isArray(v)) {
      // HeatmapData.pages 等选项数组保留，否则下拉框在空状态下无选项。
      if (k === 'pages') continue
      out[k] = []
    } else if (isTableData(v)) {
      out[k] = { ...v, rows: [] }
    }
  }
  return out
}

export function rankItems(
  rng: () => number,
  names: string[],
  base: number,
  scale: number
): RankItem[] {
  const values = names.map((name) => ({ name, value: Math.max(1, Math.round(base * scale * (0.25 + rng()))) }))
  const total = values.reduce((s, r) => s + r.value, 0)
  return values
    .map((r) => ({
      ...r,
      display: fmtInt(r.value),
      share: total > 0 ? Number(((r.value / total) * 100).toFixed(1)) : 0
    }))
    .sort((a, b) => b.value - a.value)
}

export function funnelSteps(rng: () => number, labels: string[], first: number): FunnelStep[] {
  let current = first
  return labels.map((label, i) => {
    if (i > 0) current = Math.max(1, Math.round(current * (0.18 + rng() * 0.5)))
    return {
      label,
      value: current,
      display: fmtInt(current),
      rate: i === 0 ? '100%' : fmtPct(current / first)
    }
  })
}

export function filterByKeyword<T>(rows: T[], keyword: string, pick: (r: T) => string): T[] {
  const kw = keyword.trim().toLowerCase()
  if (!kw) return rows
  return rows.filter((r) => pick(r).toLowerCase().includes(kw))
}

/**
 * 生成区间内每日 PV/UV 趋势点（供 TrendChart 复用）。
 * 天数上限 90，避免超长自定义区间拖慢渲染；compareValues 恒为 null（新模块暂无对比）。
 */
export function trendPoints(module: string, ctx: ModuleQueryContext, basePv: number): TrendPoint[] {
  const rng = rngFor(`${seedKey(module, ctx)}|trend`)
  const scale = scaleOf(ctx)
  return eachDay(ctx.startDate, ctx.endDate, 90).map((date) => {
    const pv = Math.max(1, Math.round(basePv * scale * (0.7 + rng() * 0.6)))
    const uv = Math.round(pv * (0.55 + rng() * 0.15))
    const values: MetricValues = {
      pv,
      uv,
      ip: Math.round(uv * 0.87),
      visits: Math.round(pv * 0.7),
      avgPages: Number((2 + rng() * 2).toFixed(2)),
      bounceRate: Number((0.3 + rng() * 0.25).toFixed(4)),
      avgDuration: Math.round(80 + rng() * 200),
      conversionRate: Number((0.01 + rng() * 0.05).toFixed(4))
    }
    return { date, values, compareValues: null }
  })
}

// ---- 网站概况 ----

const OVERVIEW_KEYWORDS = [
  '百度统计', '网站流量分析', 'pv uv 区别', '跳出率优化', '落地页转化',
  '热力图工具', '搜索词报告', '访客地域分布', '转化漏斗', '实时访客'
]

export function genOverview(ctx: ModuleQueryContext): OverviewData {
  const rng = rngFor(seedKey('overview', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(88000 * scale * (0.8 + rng() * 0.4))
  const uv = Math.round(pv * (0.55 + rng() * 0.15))
  const table: TableData = {
    columns: [
      { prop: 'name', label: '搜索词', sortable: true },
      { prop: 'pv', label: 'PV', sortable: true, align: 'right', format: 'int' },
      { prop: 'uv', label: 'UV', sortable: true, align: 'right', format: 'int' },
      { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' }
    ],
    rows: filterByKeyword(
      OVERVIEW_KEYWORDS.map((name) => {
        const kpv = Math.max(1, Math.round(pv * (0.01 + rng() * 0.06)))
        return {
          name,
          pv: kpv,
          uv: Math.round(kpv * (0.5 + rng() * 0.3)),
          bounceRate: floatBetween(rng, 0.25, 0.7)
        }
      }),
      ctx.keyword,
      (r) => String(r.name)
    ).sort((a, b) => Number(b.pv) - Number(a.pv))
  }
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '浏览量 PV', fmtInt(pv)),
      card(rng, 'uv', '访客数 UV', fmtInt(uv)),
      card(rng, 'ip', 'IP 数', fmtInt(uv * 0.87)),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.55))),
      card(rng, 'avgDuration', '平均访问时长', `${intBetween(rng, 1, 4)}:${String(intBetween(rng, 10, 59))}`),
      card(rng, 'conversionRate', '转化率', fmtPct(floatBetween(rng, 0.01, 0.06)))
    ],
    trend: trendPoints('overview', ctx, 12500),
    topSources: filterByKeyword(
      rankItems(rng, ['搜索引擎', '直接访问', '外部链接', '站内来源', '自定义来源'], pv / 5, 1),
      ctx.keyword,
      (r) => r.name
    ),
    topPages: rankItems(rng, ['/index.html', '/product/list', '/docs/help', '/price', '/about'], pv / 6, 1),
    keywords: { ...table }
  })
}

// ---- 百度推广 ----

const PROMO_PLANS = ['品牌词计划', '核心业务计划', '竞品词计划', '长尾词计划', '活动推广计划', '网盟再营销', '应用下载计划', '本地门店计划']

export function genPromotion(ctx: ModuleQueryContext): PromotionData {
  const rng = rngFor(seedKey('promotion', ctx))
  const scale = scaleOf(ctx)
  const click = Math.round(12000 * scale * (0.8 + rng() * 0.4))
  return applyEmptyMode({
    cards: [
      card(rng, 'cost', '推广消费（元）', fmtInt(click * (1.2 + rng()))),
      card(rng, 'shows', '展示次数', fmtInt(click * (18 + rng() * 8))),
      card(rng, 'click', '点击次数', fmtInt(click)),
      card(rng, 'conv', '转化次数', fmtInt(click * 0.06)),
      card(rng, 'convRate', '点击转化率', fmtPct(floatBetween(rng, 0.03, 0.09))),
      card(rng, 'cpc', '平均点击价格（元）', (1.2 + rng() * 2.4).toFixed(2))
    ],
    funnel: funnelSteps(rng, ['广告点击', '到达网站', '有效咨询', '达成转化'], click),
    plans: {
      columns: [
        { prop: 'name', label: '推广计划', sortable: true },
        { prop: 'click', label: '点击', sortable: true, align: 'right', format: 'int' },
        { prop: 'cost', label: '消费（元）', sortable: true, align: 'right', format: 'float2' },
        { prop: 'conv', label: '转化', sortable: true, align: 'right', format: 'int' },
        { prop: 'convRate', label: '转化率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(
        PROMO_PLANS.map((name) => {
          const c = Math.max(1, Math.round((click / PROMO_PLANS.length) * (0.3 + rng() * 1.6)))
          const conv = Math.round(c * floatBetween(rng, 0.02, 0.1))
          return {
            name,
            click: c,
            cost: Number((c * (1.1 + rng() * 2.2)).toFixed(2)),
            conv,
            convRate: c > 0 ? conv / c : 0
          }
        }),
        ctx.keyword,
        (r) => String(r.name)
      ).sort((a, b) => Number(b.click) - Number(a.click))
    }
  })
}

// ---- 流量分析 ----

const TRAFFIC_SOURCES = ['搜索引擎', '直接访问', '外部链接', '站内来源']
const TRAFFIC_PAGES = ['/index.html', '/product/list', '/docs/help', '/price', '/news/2026/11/13', '/about']
const TRAFFIC_AREAS = ['广东', '江苏', '浙江', '山东', '四川', '湖北', '北京', '上海']

export function genTraffic(ctx: ModuleQueryContext): TrafficData {
  const rng = rngFor(seedKey('traffic', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(76000 * scale * (0.8 + rng() * 0.4))
  const rows = Array.from({ length: 18 }, (_, i) => {
    const page = TRAFFIC_PAGES[intBetween(rng, 0, TRAFFIC_PAGES.length - 1)]
    return {
      time: `10:${String(30 + Math.floor(i / 2)).padStart(2, '0')}:${String((i * 7) % 60).padStart(2, '0')}`,
      page,
      source: TRAFFIC_SOURCES[intBetween(rng, 0, TRAFFIC_SOURCES.length - 1)],
      area: TRAFFIC_AREAS[intBetween(rng, 0, TRAFFIC_AREAS.length - 1)],
      device: ctx.device === 'mobile' ? '移动设备' : ctx.device === 'pc' ? '计算机' : rng() > 0.4 ? '计算机' : '移动设备',
      duration: intBetween(rng, 5, 900)
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '浏览量 PV', fmtInt(pv)),
      card(rng, 'online', '当前在线访客', fmtInt(20 + rng() * 80)),
      card(rng, 'visits', '访问次数', fmtInt(pv * 0.7)),
      card(rng, 'avgPages', '平均访问页数', (2 + rng() * 3).toFixed(2))
    ],
    trend: trendPoints('traffic', ctx, 10800),    realtime: {
      columns: [
        { prop: 'time', label: '访问时间', sortable: true },
        { prop: 'page', label: '当前页面', sortable: true },
        { prop: 'source', label: '来源', sortable: true },
        { prop: 'area', label: '地域', sortable: true },
        { prop: 'device', label: '设备', sortable: true },
        { prop: 'duration', label: '停留', sortable: true, align: 'right', format: 'duration' }
      ],
      rows: filterByKeyword(rows, ctx.keyword, (r) => `${r.page} ${r.source} ${r.area}`)
    },
    topPages: rankItems(rng, TRAFFIC_PAGES, pv / 8, 1)
  })
}

// ---- 来源分析 ----

const SEARCH_ENGINES = ['百度', '搜狗', '360搜索', '必应', '谷歌']
const SEARCH_WORDS = ['百度统计 demo', '网站数据分析', 'uv pv 统计', '免费统计工具', '流量来源报告', 'seo 优化', '落地页设计', '转化率提升', '热力图', '访客画像']

export function genSource(ctx: ModuleQueryContext): SourceData {
  const rng = rngFor(seedKey('source', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(69000 * scale * (0.8 + rng() * 0.4))
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '来源带来 PV', fmtInt(pv)),
      card(rng, 'searchShare', '搜索占比', fmtPct(0.35 + rng() * 0.15)),
      card(rng, 'newRate', '新访客比率', fmtPct(0.25 + rng() * 0.25)),
      card(rng, 'bounceRate', '来源跳出率', fmtPct(floatBetween(rng, 0.3, 0.6)))
    ],
    sourceTypes: rankItems(rng, ['搜索引擎', '直接访问', '外部链接', '站内来源', '自定义来源'], pv / 5, 1),
    searchWords: {
      columns: [
        { prop: 'word', label: '搜索词', sortable: true },
        { prop: 'engine', label: '搜索引擎', sortable: true },
        { prop: 'pv', label: 'PV', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: 'UV', sortable: true, align: 'right', format: 'int' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(
        SEARCH_WORDS.map((word) => {
          const wPv = Math.max(1, Math.round((pv / 12) * (0.2 + rng() * 1.8)))
          return {
            word,
            engine: SEARCH_ENGINES[intBetween(rng, 0, SEARCH_ENGINES.length - 1)],
            pv: wPv,
            uv: Math.round(wPv * (0.5 + rng() * 0.3)),
            conv: Math.round(wPv * floatBetween(rng, 0.005, 0.05))
          }
        }),
        ctx.keyword,
        (r) => `${r.word} ${r.engine}`
      ).sort((a, b) => Number(b.pv) - Number(a.pv))
    }
  })
}

// ---- 访问分析 ----

const VISIT_PAGES = [
  '/index.html', '/product/list', '/product/detail', '/docs/help', '/docs/api',
  '/price', '/about', '/news/2026/11/13', '/case/a', '/case/b', '/download', '/contact'
]

export function genVisit(ctx: ModuleQueryContext): VisitData {
  const rng = rngFor(seedKey('visit', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(82000 * scale * (0.8 + rng() * 0.4))
  const rows = filterByKeyword(
    VISIT_PAGES.map((name) => {
      const pPv = Math.max(1, Math.round((pv / 14) * (0.25 + rng() * 1.8)))
      return {
        name,
        pv: pPv,
        uv: Math.round(pPv * (0.5 + rng() * 0.3)),
        avgStay: intBetween(rng, 8, 420),
        exitRate: floatBetween(rng, 0.15, 0.75)
      }
    }),
    ctx.keyword,
    (r) => String(r.name)
  ).sort((a, b) => Number(b.pv) - Number(a.pv))
  const byPv = [...rows].sort((a, b) => Number(b.pv) - Number(a.pv))
  const total = byPv.reduce((s, r) => s + Number(r.pv), 0)
  const toRank = (list: typeof rows): RankItem[] =>
    list.slice(0, 6).map((r) => ({
      name: String(r.name),
      value: Number(r.pv),
      display: fmtInt(Number(r.pv)),
      share: total > 0 ? Number(((Number(r.pv) / total) * 100).toFixed(1)) : 0
    }))
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '受访 PV', fmtInt(pv)),
      card(rng, 'entries', '入口页次数', fmtInt(pv * 0.68)),
      card(rng, 'exits', '退出页次数', fmtInt(pv * 0.66)),
      card(rng, 'avgStay', '平均停留时长', `${intBetween(rng, 0, 3)}:${String(intBetween(rng, 10, 59))}`)
    ],
    pages: {
      columns: [
        { prop: 'name', label: '页面 URL', sortable: true },
        { prop: 'pv', label: 'PV', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: 'UV', sortable: true, align: 'right', format: 'int' },
        { prop: 'avgStay', label: '平均停留', sortable: true, align: 'right', format: 'duration' },
        { prop: 'exitRate', label: '退出率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows
    },
    entries: toRank(byPv),
    exits: toRank([...rows].sort((a, b) => Number(b.exitRate) - Number(a.exitRate)))
  })
}

// ---- 转化分析 ----

export function genConversion(ctx: ModuleQueryContext): ConversionData {
  const rng = rngFor(seedKey('conversion', ctx))
  const scale = scaleOf(ctx)
  const visits = Math.round(30000 * scale * (0.8 + rng() * 0.4))
  const conv = Math.round(visits * floatBetween(rng, 0.02, 0.07))
  const rows = Array.from({ length: 10 }, (_, i) => {
    const amount = Number(floatBetween(rng, 49, 1999, 2).toFixed(2))
    return {
      order: `ORD-202611${String(10 + (i % 3))}-${1000 + intBetween(rng, 0, 8999)}`,
      channel: ['搜索引擎', '直接访问', '百度推广', '外部链接'][intBetween(rng, 0, 3)],
      amount,
      time: `2026-11-${String(10 + (i % 3)).padStart(2, '0')} 1${i % 10}:${String((i * 11) % 60).padStart(2, '0')}`
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'conv', '转化次数', fmtInt(conv)),
      card(rng, 'convRate', '转化率', fmtPct(conv / visits)),
      card(rng, 'revenue', '转化收益（元）', fmtInt(rows.reduce((s, r) => s + Number(r.amount), 0))),
      card(rng, 'cost', '平均转化成本（元）', (8 + rng() * 40).toFixed(2))
    ],
    funnel: funnelSteps(rng, ['进入落地页', '浏览商品', '提交订单', '支付成功'], visits),
    orders: {
      columns: [
        { prop: 'order', label: '订单号', sortable: true },
        { prop: 'channel', label: '来源渠道', sortable: true },
        { prop: 'amount', label: '金额（元）', sortable: true, align: 'right', format: 'float2' },
        { prop: 'time', label: '转化时间', sortable: true }
      ],
      rows: filterByKeyword(
        rows,
        ctx.keyword,
        (r) => `${r.order} ${r.channel}`
      )
    }
  })
}

// ---- 优化分析 ----

const CRAWL_URLS = ['/old-page', '/tmp/share?id=1', '/assets/big.js', '/api/debug', '/news/2024/01/01', '/case/old', '/sitemap.xml', '/feed.xml']
const CRAWL_ERRORS = ['抓取超时', '404 未找到', '500 服务器错误', 'DNS 解析失败', 'robots 拦截']

export function genOptimize(ctx: ModuleQueryContext): OptimizeData {
  const rng = rngFor(seedKey('optimize', ctx))
  return applyEmptyMode({
    cards: [
      card(rng, 'indexed', '索引量', fmtInt(12000 + rng() * 4000)),
      card(rng, 'crawlErr', '抓取异常', fmtInt(5 + rng() * 30)),
      card(rng, 'slow', '慢页面数（>3s）', fmtInt(3 + rng() * 20)),
      card(rng, 'load', '平均加载耗时', `${(0.8 + rng() * 2.4).toFixed(2)}s`)
    ],
    crawlErrors: {
      columns: [
        { prop: 'url', label: '异常 URL', sortable: true },
        { prop: 'error', label: '异常类型', sortable: true },
        { prop: 'count', label: '次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'firstSeen', label: '首次发现', sortable: true }
      ],
      rows: filterByKeyword(
        CRAWL_URLS.map((url) => ({
          url,
          error: CRAWL_ERRORS[intBetween(rng, 0, CRAWL_ERRORS.length - 1)],
          count: intBetween(rng, 1, 120),
          firstSeen: `2026-11-${String(intBetween(rng, 1, 12)).padStart(2, '0')}`
        })),
        ctx.keyword,
        (r) => `${r.url} ${r.error}`
      ).sort((a, b) => Number(b.count) - Number(a.count))
    },
    slowPages: rankItems(rng, VISIT_PAGES.slice(0, 8), 3200, 1).map((r) => ({
      ...r,
      display: `${(r.value / 1000).toFixed(2)}s`
    }))
  })
}

// ---- 热力图与点击分析 ----

const HEAT_PAGES = ['/index.html', '/product/list', '/price']
const HEAT_AREAS = ['首屏 Banner', '导航栏-产品', '导航栏-文档', '立即试用按钮', '价格表-企业版', '客户案例区', '页脚-联系方式', '侧边咨询浮窗', '搜索框', '下载按钮', '视频介绍', 'FAQ 折叠区']

function heatCellsFor(page: string, ctx: ModuleQueryContext): HeatCell[] {
  const rng = rngFor(`${seedKey('heatmap', ctx)}|${page}`)
  return HEAT_AREAS.map((label) => {
    const value = intBetween(rng, 50, 5200)
    return { label, value, display: fmtInt(value), intensity: Number((value / 5200).toFixed(3)) }
  })
}

export function genHeatmap(ctx: ModuleQueryContext, page = HEAT_PAGES[0]): HeatmapData {
  const cells = heatCellsFor(page, ctx)
  const max = Math.max(1, ...cells.map((c) => c.value))
  const normalized = cells.map((c) => ({ ...c, intensity: Number((c.value / max).toFixed(3)) }))
  return applyEmptyMode({
    pages: [...HEAT_PAGES],
    cells: filterByKeyword(normalized, ctx.keyword, (c) => c.label),
    elements: {
      columns: [
        { prop: 'label', label: '点击元素', sortable: true },
        { prop: 'clicks', label: '点击次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '点击占比', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(
        normalized
          .map((c) => ({
            label: c.label,
            clicks: c.value,
            share: c.value / cells.reduce((s, x) => s + x.value, 0)
          }))
          .sort((a, b) => b.clicks - a.clicks),
        ctx.keyword,
        (r) => String(r.label)
      )
    }
  })
}

// ---- 自定义报告 ----

export const CUSTOM_DIMENSIONS = ['地域', '来源', '设备', '页面']

const DIM_VALUES: Record<string, string[]> = {
  地域: ['广东', '江苏', '浙江', '山东', '四川', '湖北'],
  来源: ['搜索引擎', '直接访问', '外部链接', '站内来源'],
  设备: ['计算机', '移动设备'],
  页面: ['/index.html', '/product/list', '/price', '/about']
}

export function genCustomReport(
  ctx: ModuleQueryContext,
  dims: string[],
  metrics: CustomMetric[]
): CustomReportData {
  const picked = dims.length > 0 ? dims : ['地域']
  const pickedMetrics = metrics.length > 0 ? metrics : (['pv'] as CustomMetric[])
  const rng = rngFor(`${seedKey('custom', ctx)}|${picked.join(',')}|${pickedMetrics.join(',')}`)
  const scale = scaleOf(ctx)
  const dimLists = picked.map((d) => DIM_VALUES[d] ?? ['-'])
  // 笛卡尔组合取前 12 行，保证行数可控。
  const combos: string[][] = []
  const walk = (i: number, cur: string[]) => {
    if (combos.length >= 12) return
    if (i === dimLists.length) {
      combos.push([...cur])
      return
    }
    for (const v of dimLists[i]) {
      cur.push(v)
      walk(i + 1, cur)
      cur.pop()
      if (combos.length >= 12) return
    }
  }
  walk(0, [])
  const columns = [
    ...picked.map((d) => ({ prop: d, label: d, sortable: true as const })),
    ...pickedMetrics.map((m) => ({
      prop: m,
      label: CUSTOM_METRIC_LABELS[m],
      sortable: true as const,
      align: 'right' as const,
      format: 'int' as const
    }))
  ]
  const rows = combos.map((combo) => {
    const row: Record<string, string | number> = {}
    picked.forEach((d, i) => {
      row[d] = combo[i]
    })
    const pv = Math.max(1, Math.round(4000 * scale * (0.2 + rng() * 1.6)))
    if (pickedMetrics.includes('pv')) row.pv = pv
    if (pickedMetrics.includes('uv')) row.uv = Math.round(pv * (0.5 + rng() * 0.3))
    if (pickedMetrics.includes('conv')) row.conv = Math.round(pv * floatBetween(rng, 0.005, 0.05))
    return row
  })
  const keywordRows =
    ctx.keyword.trim() === ''
      ? rows
      : rows.filter((r) => Object.values(r).join(' ').toLowerCase().includes(ctx.keyword.trim().toLowerCase()))
  return applyEmptyMode({ table: { columns, rows: keywordRows } })
}

// ---- 数据导出与开放平台 ----

export function genExport(ctx: ModuleQueryContext): ExportData {
  const rng = rngFor(seedKey('export', ctx))
  const taskNames = ['地域分布日报', '来源周报', '转化漏斗月报', '搜索词明细', '实时访客快照']
  const statuses = ['已完成', '已完成', '处理中', '已完成', '排队中']
  return applyEmptyMode({
    cards: [
      card(rng, 'apiCalls', 'API 调用（次/日）', fmtInt(8000 + rng() * 9000)),
      card(rng, 'tasks', '导出任务', fmtInt(taskNames.length)),
      card(rng, 'success', '导出成功率', fmtPct(0.96 + rng() * 0.03)),
      card(rng, 'quota', '剩余配额（次）', fmtInt(40000 + rng() * 20000))
    ],
    tasks: {
      columns: [
        { prop: 'name', label: '任务名称', sortable: true },
        { prop: 'range', label: '数据区间', sortable: true },
        { prop: 'status', label: '状态', sortable: true },
        { prop: 'created', label: '创建时间', sortable: true }
      ],
      rows: taskNames.map((name, i) => ({
        name,
        range: `${ctx.startDate} 至 ${ctx.endDate}`,
        status: statuses[i],
        created: `2026-11-1${i % 3} 09:1${i}`
      }))
    },
    apiKeys: {
      columns: [
        { prop: 'name', label: '密钥名称', sortable: true },
        { prop: 'prefix', label: '前缀', sortable: false },
        { prop: 'calls', label: '今日调用', sortable: true, align: 'right', format: 'int' },
        { prop: 'status', label: '状态', sortable: true }
      ],
      rows: [
        { name: '内部报表同步', prefix: 'ak_live_8f31…', calls: intBetween(rng, 1000, 5000), status: '启用' },
        { name: '数据仓库 ETL', prefix: 'ak_live_77c2…', calls: intBetween(rng, 1000, 5000), status: '启用' },
        { name: '测试联调', prefix: 'ak_test_19ab…', calls: intBetween(rng, 0, 200), status: '停用' }
      ]
    }
  })
}

// ---- 账户与权限管理 ----

const ACCOUNT_USERS = [
  { name: 'zhang.san', role: '管理员', sites: '全部站点', status: '启用' },
  { name: 'li.si', role: '分析师', sites: '演示站点A', status: '启用' },
  { name: 'wang.wu', role: '分析师', sites: '演示站点A', status: '启用' },
  { name: 'zhao.liu', role: '只读', sites: '演示站点B', status: '启用' },
  { name: 'qian.qi', role: '只读', sites: '演示站点A', status: '停用' },
  { name: 'sun.ba', role: '分析师', sites: '演示站点B', status: '启用' }
]

export const ACCOUNT_ROLES = ['全部角色', '管理员', '分析师', '只读']

export function genAccount(ctx: ModuleQueryContext): AccountData {
  const rng = rngFor(seedKey('account', ctx))
  const rows = filterByKeyword(
    ACCOUNT_USERS.map((u, i) => ({ ...u, lastLogin: `2026-11-1${i % 3} 1${i % 10}:0${i % 10}` })),
    ctx.keyword,
    (r) => `${r.name} ${r.role}`
  )
  return applyEmptyMode({
    cards: [
      card(rng, 'users', '成员总数', fmtInt(ACCOUNT_USERS.length)),
      card(rng, 'admins', '管理员', fmtInt(ACCOUNT_USERS.filter((u) => u.role === '管理员').length)),
      card(rng, 'enabled', '启用中', fmtInt(ACCOUNT_USERS.filter((u) => u.status === '启用').length)),
      card(rng, 'pending', '待审核', fmtInt(intBetween(rng, 0, 3)))
    ],
    users: {
      columns: [
        { prop: 'name', label: '成员', sortable: true },
        { prop: 'role', label: '角色', sortable: true },
        { prop: 'sites', label: '数据权限', sortable: true },
        { prop: 'status', label: '状态', sortable: true },
        { prop: 'lastLogin', label: '最近登录', sortable: true }
      ],
      rows
    }
  })
}
