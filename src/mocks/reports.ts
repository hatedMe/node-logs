// 20 个还原报表页的 mock 数据生成器（纯函数）：每个菜单链接对应的报表页各一个。
// 规则与 mocks/modules.ts 一致：种子 = 页面 + 查询上下文，同条件稳定、异条件变化，
// 关键词做真实子串过滤；空态/失败态由 applyEmptyMode 与 useModuleQuery 的场景开关统一控制。

import { floatBetween, intBetween, rngFor } from './seed'
import {
  applyEmptyMode,
  card,
  filterByKeyword,
  fmtInt,
  fmtPct,
  funnelSteps,
  rankItems,
  scaleOf,
  seedKey,
  trendPoints
} from './modules'
import type { FunnelStep, HeatCell, ModuleQueryContext, RankItem, TableRow, TableData } from '../types/modules'
import type { GeoRow, MetricValues, TrendPoint } from '../types/report'
import { COUNTRY_NAMES, PROVINCE_NAMES, zeroMetricValues } from '../types/report'
import type { ReportPageData } from '../types/reports'
import type { ReportPageKey } from '../types/navigation'
import { addDays, eachDay } from '../utils/date'

// ---- 公共词库（各报表页共用的演示数据） ----

const SITE_PAGES = [
  '/index.html', '/product/list', '/product/detail', '/docs/help', '/docs/api',
  '/price', '/about', '/news/2026/11/13', '/case/a', '/case/b', '/download', '/contact'
]
const SOURCE_TYPES = ['搜索引擎', '直接访问', '外部链接', '站内来源', '自定义来源']
const ENGINES = ['百度', '360搜索', '搜狗', '必应', '谷歌', '神马搜索']
const SEARCH_WORDS = [
  '百度统计 demo', '网站数据分析', 'uv pv 统计', '免费统计工具', '流量来源报告',
  'seo 优化', '落地页设计', '转化率提升', '热力图工具', '访客画像', '搜索词报告', '实时访客监控'
]
const LINK_PAGES = [
  'zhihu.com/question/1024', 'weibo.com/1234567890', 'mp.weixin.qq.com/s/abc123',
  'juejin.cn/post/7102', 'bilibili.com/video/BV1xx', 'csdn.net/blog/8899',
  'v2ex.com/t/778899', 'gitee.com/explore', 'jianshu.com/p/6655', 'segmentfault.com/q/1001'
]
const LINK_DOMAINS = ['zhihu.com', 'weibo.com', 'mp.weixin.qq.com', 'juejin.cn', 'bilibili.com', 'csdn.net', 'v2ex.com', 'gitee.com']
const LINK_CATEGORIES = ['论坛博客', '社交媒体', '新闻资讯', '视频网站', '合作伙伴']
const REFERRER_PAGES = ['/index.html', '/product/list', '/docs/help', '/price', '/news/2026/11/13', '/about', '/case/a']
const AREA_NAMES = PROVINCE_NAMES.slice(0, 12)
const BROWSERS = ['Chrome', 'Safari', 'Edge', 'Firefox', 'QQ浏览器', '360安全浏览器', 'IE 11']
const OPERATING_SYSTEMS = ['Windows', 'Android', 'iOS', 'macOS', 'HarmonyOS', 'Linux']
const RESOLUTIONS = ['1920×1080', '1366×768', '1440×900', '2560×1440', '375×812', '414×896']
const HOUR_SLOTS = Array.from({ length: 12 }, (_, i) => `${String(i * 2).padStart(2, '0')}:00-${String(i * 2 + 2).padStart(2, '0')}:00`)
const EVENT_ROWS: Array<[string, string, string]> = [
  ['注册', '按钮点击', 'register-btn'],
  ['下载', '按钮点击', 'download-btn'],
  ['视频', '播放', 'intro-video'],
  ['咨询', '浮窗点击', 'side-consult'],
  ['搜索', '站内搜索', 'site-search'],
  ['表单', '提交', 'contact-form'],
  ['导航', '菜单点击', 'nav-menu'],
  ['收藏', '按钮点击', 'bookmark-btn']
]
const SUBDIRS = ['/product/', '/docs/', '/news/', '/case/', '/about/', '/download/', '/contact/']
const GOALS = ['注册成功', '提交咨询', '下单支付', '资料下载', '在线客服']
const CLICK_AREAS = [
  '首屏 Banner', '导航栏-产品', '导航栏-文档', '立即试用按钮', '价格表-企业版', '客户案例区',
  '页脚-联系方式', '侧边咨询浮窗', '搜索框', '下载按钮', '视频介绍', 'FAQ 折叠区'
]

function sortByValue(rows: TableRow[], prop: string): TableRow[] {
  return [...rows].sort((a, b) => Number(b[prop]) - Number(a[prop]))
}

function shareRows(rows: Array<{ name: string; value: number }>): RankItem[] {
  const total = rows.reduce((s, r) => s + r.value, 0)
  return rows
    .map((r) => ({
      name: r.name,
      value: r.value,
      display: fmtInt(r.value),
      share: total > 0 ? Number(((r.value / total) * 100).toFixed(1)) : 0
    }))
    .sort((a, b) => b.value - a.value)
}

// ---- 网站概况（overview/index） ----

export function genOverviewIndex(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('overview', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(88000 * scale * (0.8 + rng() * 0.4))
  const uv = Math.round(pv * (0.55 + rng() * 0.15))
  const keywords: TableData = {
    columns: [
      { prop: 'name', label: '搜索词', sortable: true },
      { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
      { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
      { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' }
    ],
    rows: sortByValue(
      filterByKeyword(
        SEARCH_WORDS.map((name) => {
          const kpv = Math.max(1, Math.round(pv * (0.008 + rng() * 0.05)))
          return { name, pv: kpv, uv: Math.round(kpv * (0.5 + rng() * 0.3)), bounceRate: floatBetween(rng, 0.25, 0.7) }
        }),
        ctx.keyword,
        (r) => String(r.name)
      ),
      'pv'
    )
  }
  const topPages: TableData = {
    columns: [
      { prop: 'name', label: '受访页面', sortable: true },
      { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
      { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
      { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' }
    ],
    rows: sortByValue(
      SITE_PAGES.slice(0, 10).map((name) => {
        const ppv = Math.max(1, Math.round((pv / 10) * (0.25 + rng() * 1.6)))
        return { name, pv: ppv, uv: Math.round(ppv * (0.5 + rng() * 0.3)), share: ppv / pv }
      }),
      'pv'
    )
  }
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '浏览量 PV', fmtInt(pv)),
      card(rng, 'uv', '访客数 UV', fmtInt(uv)),
      card(rng, 'ip', 'IP 数', fmtInt(uv * 0.87)),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.55))),
      card(rng, 'avgDuration', '平均访问时长', `${intBetween(rng, 1, 4)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`),
      card(rng, 'conversionRate', '转化次数', fmtInt(pv * floatBetween(rng, 0.01, 0.04)))
    ],
    trend: trendPoints('overview', ctx, 12500),
    rank: rankItems(rng, SOURCE_TYPES, pv / 5, 1),
    table: keywords,
    secondary: topPages
  })
}

// ---- 实时访客（trend/latest） ----

export function genTrafficLatest(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('traffic-latest', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(8600 * scale * (0.8 + rng() * 0.4))
  const online = Math.round(20 + rng() * 90)
  const realtimeRows = Array.from({ length: 18 }, (_, i) => {
    const page = SITE_PAGES[intBetween(rng, 0, SITE_PAGES.length - 1)]
    return {
      time: `10:${String(30 + Math.floor(i / 2)).padStart(2, '0')}:${String((i * 7) % 60).padStart(2, '0')}`,
      source: SOURCE_TYPES[intBetween(rng, 0, SOURCE_TYPES.length - 1)],
      area: AREA_NAMES[intBetween(rng, 0, AREA_NAMES.length - 1)],
      landing: SITE_PAGES[intBetween(rng, 0, SITE_PAGES.length - 1)],
      page,
      pages: intBetween(rng, 1, 12),
      duration: intBetween(rng, 5, 900),
      device: ctx.device === 'mobile' ? '移动设备' : ctx.device === 'pc' ? '计算机' : rng() > 0.4 ? '计算机' : '移动设备'
    }
  })
  const hourly: TableData = {
    columns: [
      { prop: 'name', label: '时段', sortable: true },
      { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
      { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' }
    ],
    rows: HOUR_SLOTS.map((name) => {
      const hpv = Math.max(1, Math.round((pv / 12) * (0.3 + rng() * 1.5)))
      return { name, pv: hpv, uv: Math.round(hpv * (0.5 + rng() * 0.3)) }
    })
  }
  return applyEmptyMode({
    cards: [
      card(rng, 'online', '当前在线访客', fmtInt(online)),
      card(rng, 'pv', '今日浏览量 PV', fmtInt(pv)),
      card(rng, 'uv', '今日访客数 UV', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'bounceRate', '今日跳出率', fmtPct(floatBetween(rng, 0.3, 0.55)))
    ],
    trend: trendPoints('traffic-latest', ctx, 8600),
    table: {
      columns: [
        { prop: 'time', label: '访问时间', sortable: true },
        { prop: 'source', label: '来源', sortable: true },
        { prop: 'area', label: '地域', sortable: true },
        { prop: 'landing', label: '入口页面', sortable: true },
        { prop: 'page', label: '当前页面', sortable: true },
        { prop: 'pages', label: '访问页数', sortable: true, align: 'right', format: 'int' },
        { prop: 'duration', label: '停留时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'device', label: '设备', sortable: true }
      ],
      rows: filterByKeyword(realtimeRows, ctx.keyword, (r) => `${r.page} ${r.source} ${r.area}`)
    },
    secondary: hourly,
    rank: shareRows(
      REFERRER_PAGES.map((name) => ({ name, value: Math.max(1, Math.round(online * (0.08 + rng() * 0.3))) }))
    )
  })
}

// ---- 趋势分析（trend/time） ----

export function genTrendTime(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('traffic-time', ctx))
  const trend = trendPoints('traffic-time', ctx, 10800)
  const dailyRows = trend.map((p: TrendPoint) => ({
    name: p.date,
    pv: p.values.pv,
    uv: p.values.uv,
    ip: p.values.ip,
    visits: p.values.visits,
    bounceRate: p.values.bounceRate,
    avgDuration: p.values.avgDuration,
    avgPages: p.values.avgPages,
    conversionRate: p.values.conversionRate
  }))
  const total = trend.reduce(
    (s, p) => {
      s.pv += p.values.pv
      s.uv += p.values.uv
      return s
    },
    { pv: 0, uv: 0 }
  )
  const hourlyRows = HOUR_SLOTS.map((name) => {
    const hpv = Math.max(1, Math.round((total.pv / 12) * (0.3 + rng() * 1.5)))
    return {
      name,
      pv: hpv,
      uv: Math.round(hpv * (0.5 + rng() * 0.3)),
      ip: Math.round(hpv * 0.45),
      bounceRate: floatBetween(rng, 0.3, 0.6)
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '浏览量 PV', fmtInt(total.pv)),
      card(rng, 'uv', '访客数 UV', fmtInt(total.uv)),
      card(rng, 'ip', 'IP 数', fmtInt(total.uv * 0.87)),
      card(rng, 'visits', '访问次数', fmtInt(total.pv * 0.7)),
      card(rng, 'avgPages', '平均访问页数', (2 + rng() * 2).toFixed(2)),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.55)))
    ],
    trend,
    table: {
      columns: [
        { prop: 'name', label: '日期', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'ip', label: 'IP 数', sortable: true, align: 'right', format: 'int' },
        { prop: 'visits', label: '访问次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均访问时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'avgPages', label: '平均访问页数', sortable: true, align: 'right', format: 'float2' },
        { prop: 'conversionRate', label: '转化率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(dailyRows, ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '时段', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'ip', label: 'IP 数', sortable: true, align: 'right', format: 'int' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: hourlyRows
    }
  })
}

// ---- 全部来源（source/all） ----

export function genSourceAll(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('source-all', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(69000 * scale * (0.8 + rng() * 0.4))
  const typeRows = SOURCE_TYPES.map((name) => {
    const tPv = Math.max(1, Math.round((pv / 5) * (0.3 + rng() * 1.5)))
    return {
      name,
      pv: tPv,
      uv: Math.round(tPv * (0.5 + rng() * 0.3)),
      bounceRate: floatBetween(rng, 0.25, 0.65),
      avgDuration: intBetween(rng, 30, 420),
      avgPages: Number((1.5 + rng() * 2.5).toFixed(2)),
      conv: Math.round(tPv * floatBetween(rng, 0.005, 0.05))
    }
  })
  const siteRows = LINK_DOMAINS.concat(['direct', 'baidu.com']).map((name) => {
    const sPv = Math.max(1, Math.round((pv / 10) * (0.2 + rng() * 1.8)))
    return {
      name: name === 'direct' ? '直接访问' : name,
      pv: sPv,
      uv: Math.round(sPv * (0.5 + rng() * 0.3)),
      bounceRate: floatBetween(rng, 0.25, 0.7),
      avgDuration: intBetween(rng, 20, 380)
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '浏览量 PV', fmtInt(pv)),
      card(rng, 'uv', '访客数 UV', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.6))),
      card(rng, 'avgDuration', '平均访问时长', `${intBetween(rng, 1, 4)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`),
      card(rng, 'newRate', '新访客比率', fmtPct(0.25 + rng() * 0.25)),
      card(rng, 'conv', '转化次数', fmtInt(pv * floatBetween(rng, 0.01, 0.04)))
    ],
    trend: trendPoints('source-all', ctx, 9800),
    rank: shareRows(typeRows.map((r) => ({ name: String(r.name), value: Number(r.pv) }))),
    table: {
      columns: [
        { prop: 'name', label: '来源类型', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均访问时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'avgPages', label: '平均访问页数', sortable: true, align: 'right', format: 'float2' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(typeRows, 'pv'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '来源网站', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' }
      ],
      rows: filterByKeyword(sortByValue(siteRows, 'pv'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 搜索引擎（source/engine） ----

export function genSourceEngine(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('source-engine', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(28000 * scale * (0.8 + rng() * 0.4))
  const engineRows = ENGINES.map((name) => {
    const ePv = Math.max(1, Math.round((pv / ENGINES.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      pv: ePv,
      uv: Math.round(ePv * (0.5 + rng() * 0.3)),
      bounceRate: floatBetween(rng, 0.25, 0.65),
      avgDuration: intBetween(rng, 30, 420),
      conv: Math.round(ePv * floatBetween(rng, 0.005, 0.05))
    }
  })
  const wordRows = SEARCH_WORDS.map((word) => {
    const wPv = Math.max(1, Math.round((pv / 12) * (0.2 + rng() * 1.8)))
    return {
      name: word,
      engine: ENGINES[intBetween(rng, 0, ENGINES.length - 1)],
      pv: wPv,
      uv: Math.round(wPv * (0.5 + rng() * 0.3)),
      conv: Math.round(wPv * floatBetween(rng, 0.005, 0.05))
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '搜索流量 PV', fmtInt(pv)),
      card(rng, 'uv', '搜索访客 UV', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.6))),
      card(rng, 'avgDuration', '平均访问时长', `${intBetween(rng, 1, 4)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`),
      card(rng, 'words', '搜索词数', fmtInt(1800 + rng() * 2600)),
      card(rng, 'conv', '转化次数', fmtInt(pv * floatBetween(rng, 0.01, 0.04)))
    ],
    trend: trendPoints('source-engine', ctx, 4200),
    rank: shareRows(engineRows.map((r) => ({ name: String(r.name), value: Number(r.pv) }))),
    table: {
      columns: [
        { prop: 'name', label: '搜索引擎', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均访问时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(engineRows, 'pv'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '搜索词', sortable: true },
        { prop: 'engine', label: '搜索引擎', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(wordRows, 'pv'), ctx.keyword, (r) => `${r.name} ${r.engine}`)
    }
  })
}

// ---- 搜索词（source/searchword） ----

export function genSourceSearchWord(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('source-searchword', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(18000 * scale * (0.8 + rng() * 0.4))
  const naturalRows = SEARCH_WORDS.map((word) => {
    const wPv = Math.max(1, Math.round((pv / 12) * (0.2 + rng() * 1.8)))
    return {
      name: word,
      engine: ENGINES[intBetween(rng, 0, ENGINES.length - 1)],
      pv: wPv,
      uv: Math.round(wPv * (0.5 + rng() * 0.3)),
      bounceRate: floatBetween(rng, 0.25, 0.7),
      avgDuration: intBetween(rng, 15, 360),
      conv: Math.round(wPv * floatBetween(rng, 0.005, 0.05))
    }
  })
  const bidRows = SEARCH_WORDS.slice(0, 8).map((word) => {
    const shows = Math.max(1, Math.round((pv / 8) * (1 + rng() * 4)))
    const clicks = Math.max(1, Math.round(shows * floatBetween(rng, 0.02, 0.12)))
    return {
      name: word,
      shows,
      clicks,
      cost: Number((clicks * (1.1 + rng() * 2.6)).toFixed(2)),
      ctr: shows > 0 ? clicks / shows : 0,
      conv: Math.round(clicks * floatBetween(rng, 0.02, 0.12))
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '搜索词带来 PV', fmtInt(pv)),
      card(rng, 'uv', '搜索访客 UV', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'words', '搜索词数', fmtInt(1800 + rng() * 2600)),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.6)))
    ],
    trend: trendPoints('source-searchword', ctx, 2600),
    rank: shareRows(naturalRows.slice(0, 6).map((r) => ({ name: String(r.name), value: Number(r.pv) }))),
    table: {
      columns: [
        { prop: 'name', label: '搜索词', sortable: true },
        { prop: 'engine', label: '搜索引擎', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均访问时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(naturalRows, 'pv'), ctx.keyword, (r) => `${r.name} ${r.engine}`)
    },
    secondary: {
      columns: [
        { prop: 'name', label: '竞价搜索词', sortable: true },
        { prop: 'shows', label: '展现次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'clicks', label: '点击次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'cost', label: '消费(元)', sortable: true, align: 'right', format: 'float2' },
        { prop: 'ctr', label: '点击率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(bidRows, 'clicks'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 外部链接（source/link） ----

export function genSourceLink(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('source-link', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(12400 * scale * (0.8 + rng() * 0.4))
  const linkRows = LINK_PAGES.map((name) => {
    const lPv = Math.max(1, Math.round((pv / LINK_PAGES.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      category: LINK_CATEGORIES[intBetween(rng, 0, LINK_CATEGORIES.length - 1)],
      pv: lPv,
      uv: Math.round(lPv * (0.5 + rng() * 0.3)),
      bounceRate: floatBetween(rng, 0.3, 0.75),
      avgDuration: intBetween(rng, 15, 360),
      conv: Math.round(lPv * floatBetween(rng, 0.002, 0.04))
    }
  })
  const domainRows = LINK_DOMAINS.map((name) => {
    const dPv = Math.max(1, Math.round((pv / LINK_DOMAINS.length) * (0.3 + rng() * 1.7)))
    return { name, pv: dPv, uv: Math.round(dPv * (0.5 + rng() * 0.3)), share: dPv / pv }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '外部链接 PV', fmtInt(pv)),
      card(rng, 'uv', '外链访客 UV', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'links', '链接页面数', fmtInt(120 + rng() * 260)),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.7))),
      card(rng, 'avgDuration', '平均停留时长', `${intBetween(rng, 0, 3)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`),
      card(rng, 'conv', '转化次数', fmtInt(pv * floatBetween(rng, 0.005, 0.03)))
    ],
    trend: trendPoints('source-link', ctx, 1800),
    rank: shareRows(linkRows.slice(0, 6).map((r) => ({ name: String(r.name), value: Number(r.pv) }))),
    table: {
      columns: [
        { prop: 'name', label: '外部链接', sortable: true },
        { prop: 'category', label: '来源分类', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(linkRows, 'pv'), ctx.keyword, (r) => `${r.name} ${r.category}`)
    },
    secondary: {
      columns: [
        { prop: 'name', label: '链接域名', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(sortByValue(domainRows, 'pv'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 站内来源（source/inner） ----

export function genSourceInner(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('source-inner', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(24000 * scale * (0.8 + rng() * 0.4))
  const innerRows = REFERRER_PAGES.map((name) => {
    const iPv = Math.max(1, Math.round((pv / REFERRER_PAGES.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      pv: iPv,
      uv: Math.round(iPv * (0.5 + rng() * 0.3)),
      share: iPv / pv,
      avgDuration: intBetween(rng, 15, 360)
    }
  })
  const pageRows = SITE_PAGES.slice(0, 8).map((name) => {
    const pPv = Math.max(1, Math.round((pv / 8) * (0.3 + rng() * 1.7)))
    return {
      name,
      pv: pPv,
      uv: Math.round(pPv * (0.5 + rng() * 0.3)),
      exitRate: floatBetween(rng, 0.15, 0.75)
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '站内来源 PV', fmtInt(pv)),
      card(rng, 'entries', '入口页面数', fmtInt(24 + rng() * 60)),
      card(rng, 'depth', '平均访问深度', (2 + rng() * 2).toFixed(2)),
      card(rng, 'returnRate', '回访率', fmtPct(0.18 + rng() * 0.2))
    ],
    trend: trendPoints('source-inner', ctx, 3400),
    rank: shareRows(innerRows.map((r) => ({ name: String(r.name), value: Number(r.pv) }))),
    table: {
      columns: [
        { prop: 'name', label: '来源页面', sortable: true },
        { prop: 'pv', label: '导入 PV', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '导入 UV', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' }
      ],
      rows: filterByKeyword(sortByValue(innerRows, 'pv'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '受访页面', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'exitRate', label: '退出率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(sortByValue(pageRows, 'pv'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 受访页面（visit/toppage） ----

export function genVisitTopPage(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('visit-toppage', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(82000 * scale * (0.8 + rng() * 0.4))
  const pageRows = SITE_PAGES.map((name) => {
    const pPv = Math.max(1, Math.round((pv / 12) * (0.25 + rng() * 1.8)))
    return {
      name,
      pv: pPv,
      uv: Math.round(pPv * (0.5 + rng() * 0.3)),
      exitRate: floatBetween(rng, 0.15, 0.75),
      avgStay: intBetween(rng, 8, 420),
      downstream: Math.round(pPv * (0.4 + rng() * 1.2))
    }
  })
  const titleRows = SITE_PAGES.slice(0, 8).map((name) => {
    const tPv = Math.max(1, Math.round((pv / 8) * (0.3 + rng() * 1.6)))
    return {
      name: `${name.replace(/^\//, '').replace(/[/.-]/g, ' ').trim() || '首页'} - 演示站点A`,
      pv: tPv,
      uv: Math.round(tPv * (0.5 + rng() * 0.3)),
      exitRate: floatBetween(rng, 0.15, 0.75),
      avgStay: intBetween(rng, 8, 420)
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'pv', '受访 PV', fmtInt(pv)),
      card(rng, 'uv', '访客数 UV', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'avgStay', '平均停留时长', `${intBetween(rng, 0, 4)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`),
      card(rng, 'exitRate', '退出率', fmtPct(floatBetween(rng, 0.15, 0.5)))
    ],
    trend: trendPoints('visit-toppage', ctx, 11600),
    rank: shareRows(pageRows.slice(0, 6).map((r) => ({ name: String(r.name), value: Number(r.pv) }))),
    table: {
      columns: [
        { prop: 'name', label: '页面 URL', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'exitRate', label: '退出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgStay', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'downstream', label: '贡献下游浏览量', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(pageRows, 'pv'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '页面标题', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'exitRate', label: '退出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgStay', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' }
      ],
      rows: filterByKeyword(sortByValue(titleRows, 'pv'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 入口页面（visit/landingpage） ----

export function genVisitLandingPage(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('visit-landingpage', ctx))
  const scale = scaleOf(ctx)
  const visits = Math.round(54000 * scale * (0.8 + rng() * 0.4))
  const rows = SITE_PAGES.slice(0, 10).map((name) => {
    const entries = Math.max(1, Math.round((visits / 10) * (0.3 + rng() * 1.7)))
    return {
      name,
      entries,
      share: entries / visits,
      bounceRate: floatBetween(rng, 0.25, 0.75),
      avgStay: intBetween(rng, 8, 420),
      downstream: Math.round(entries * (0.5 + rng() * 1.6))
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'entries', '入口次数', fmtInt(visits)),
      card(rng, 'visitors', '入口访客', fmtInt(visits * (0.6 + rng() * 0.15))),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.6))),
      card(rng, 'avgStay', '平均停留时长', `${intBetween(rng, 0, 3)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`)
    ],
    trend: trendPoints('visit-landingpage', ctx, 7600),
    rank: shareRows(rows.slice(0, 6).map((r) => ({ name: String(r.name), value: Number(r.entries) }))),
    table: {
      columns: [
        { prop: 'name', label: '入口页面', sortable: true },
        { prop: 'entries', label: '入口次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '入口占比', sortable: true, align: 'right', format: 'percent' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgStay', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'downstream', label: '带来浏览量', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(rows, 'entries'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 受访域名（visit/topdomain） ----

export function genVisitTopDomain(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('visit-topdomain', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(76000 * scale * (0.8 + rng() * 0.4))
  const domains = ['www.example.com', 'm.example.com', 'blog.example.com', 'docs.example.com', 'shop.example.com', 'cdn.example.com']
  const rows = domains.map((name) => {
    const dPv = Math.max(1, Math.round((pv / domains.length) * (0.3 + rng() * 1.6)))
    return {
      name,
      pv: dPv,
      uv: Math.round(dPv * (0.5 + rng() * 0.3)),
      avgStay: intBetween(rng, 20, 480),
      exitRate: floatBetween(rng, 0.15, 0.7)
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'domains', '受访域名数', fmtInt(domains.length)),
      card(rng, 'pv', '浏览量 PV', fmtInt(pv)),
      card(rng, 'uv', '访客数 UV', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'avgStay', '平均停留时长', `${intBetween(rng, 1, 4)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`)
    ],
    trend: trendPoints('visit-topdomain', ctx, 10800),
    rank: shareRows(rows.map((r) => ({ name: String(r.name), value: Number(r.pv) }))),
    table: {
      columns: [
        { prop: 'name', label: '受访域名', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'avgStay', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'exitRate', label: '退出率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(sortByValue(rows, 'pv'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 页面点击图（custom/pageclick） ----

export const CLICKABLE_PAGES = SITE_PAGES.slice(0, 5)

export function genPageClick(ctx: ModuleQueryContext, page = CLICKABLE_PAGES[0]): ReportPageData {
  const rng = rngFor(`${seedKey('visit-pageclick', ctx)}|${page}`)
  const scale = scaleOf(ctx)
  const clicks = Math.round(26000 * scale * (0.8 + rng() * 0.4))
  const cells: HeatCell[] = CLICK_AREAS.map((label) => {
    const value = Math.max(1, Math.round((clicks / CLICK_AREAS.length) * (0.15 + rng() * 1.8)))
    return { label, value, display: fmtInt(value), intensity: 0 }
  })
  const max = Math.max(1, ...cells.map((c) => c.value))
  cells.forEach((c) => {
    c.intensity = Number((c.value / max).toFixed(3))
  })
  const elementRows = cells.map((c) => ({
    name: c.label,
    clicks: c.value,
    share: c.value / clicks,
    page
  }))
  const hourRows = HOUR_SLOTS.map((name) => {
    const h = Math.max(1, Math.round((clicks / 12) * (0.3 + rng() * 1.5)))
    return { name, clicks: h, users: Math.round(h * (0.5 + rng() * 0.3)) }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'clicks', '点击次数', fmtInt(clicks)),
      card(rng, 'users', '点击用户数', fmtInt(clicks * (0.42 + rng() * 0.18))),
      card(rng, 'pages', '点击页面数', fmtInt(20 + rng() * 90)),
      card(rng, 'avg', '人均点击次数', (1.2 + rng() * 1.6).toFixed(2))
    ],
    trend: trendPoints('visit-pageclick', ctx, 3600),
    table: {
      columns: [
        { prop: 'name', label: '点击元素', sortable: true },
        { prop: 'clicks', label: '点击次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '点击占比', sortable: true, align: 'right', format: 'percent' },
        { prop: 'page', label: '所属页面', sortable: true }
      ],
      rows: filterByKeyword(sortByValue(elementRows, 'clicks'), ctx.keyword, (r) => `${r.name} ${r.page}`)
    },
    secondary: {
      columns: [
        { prop: 'name', label: '时段', sortable: true },
        { prop: 'clicks', label: '点击次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'users', label: '点击用户数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: hourRows
    },
    heat: filterByKeyword(cells, ctx.keyword, (c) => c.label),
    pages: [...CLICKABLE_PAGES]
  })
}

// ---- 页面上下游（custom/flow） ----

export const FLOWABLE_PAGES = SITE_PAGES.slice(0, 5)

export function genPageFlow(ctx: ModuleQueryContext, page = FLOWABLE_PAGES[0]): ReportPageData {
  const rng = rngFor(`${seedKey('visit-flow', ctx)}|${page}`)
  const scale = scaleOf(ctx)
  const pv = Math.round(18000 * scale * (0.8 + rng() * 0.4))
  const upRows = REFERRER_PAGES.map((name) => {
    const v = Math.max(1, Math.round((pv / REFERRER_PAGES.length) * (0.3 + rng() * 1.7)))
    return { name: name === page ? `${name}（当前页）` : name, pv: v, share: v / pv, avgDuration: intBetween(rng, 10, 300) }
  })
  const downRows = SITE_PAGES.slice(0, 8).map((name) => {
    const v = Math.max(1, Math.round((pv / 8) * (0.3 + rng() * 1.7)))
    return { name, pv: v, share: v / pv, exitRate: floatBetween(rng, 0.15, 0.75) }
  })
  const steps: FunnelStep[] = funnelSteps(rng, ['进入本页', '浏览本页', '流出到下游', '达成转化'], Math.round(pv))
  return applyEmptyMode({
    cards: [
      card(rng, 'up', '上游页面数', fmtInt(REFERRER_PAGES.length + intBetween(rng, 2, 30))),
      card(rng, 'down', '下游页面数', fmtInt(SITE_PAGES.length + intBetween(rng, 2, 26))),
      card(rng, 'users', '路径用户数', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'depth', '平均访问深度', (2 + rng() * 2.4).toFixed(2))
    ],
    trend: trendPoints('visit-flow', ctx, 2600),
    funnel: steps,
    table: {
      columns: [
        { prop: 'name', label: '上游页面', sortable: true },
        { prop: 'pv', label: '导入 PV', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' }
      ],
      rows: filterByKeyword(sortByValue(upRows, 'pv'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '下游页面', sortable: true },
        { prop: 'pv', label: '导出 PV', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' },
        { prop: 'exitRate', label: '退出率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(sortByValue(downRows, 'pv'), ctx.keyword, (r) => String(r.name))
    },
    pages: [...FLOWABLE_PAGES]
  })
}

// ---- 事件分析（custom/event_track） ----

export function genEventTrack(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('visit-event-track', ctx))
  const scale = scaleOf(ctx)
  const total = Math.round(9800 * scale * (0.8 + rng() * 0.4))
  const rows = EVENT_ROWS.map(([category, action, label]) => {
    const times = Math.max(1, Math.round((total / EVENT_ROWS.length) * (0.3 + rng() * 1.7)))
    return {
      name: category,
      action,
      label,
      times,
      users: Math.round(times * (0.4 + rng() * 0.35)),
      avgDuration: intBetween(rng, 5, 240),
      conv: Math.round(times * floatBetween(rng, 0.01, 0.08))
    }
  })
  const dailyRows = eachDayLabels(ctx).map((date) => {
    const t = Math.max(1, Math.round((total / 7) * (0.5 + rng() * 1.1)))
    return { name: date, times: t, users: Math.round(t * (0.4 + rng() * 0.35)) }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'times', '事件次数', fmtInt(total)),
      card(rng, 'users', '触发用户数', fmtInt(total * (0.42 + rng() * 0.18))),
      card(rng, 'events', '事件总数', fmtInt(EVENT_ROWS.length)),
      card(rng, 'convRate', '事件转化率', fmtPct(floatBetween(rng, 0.02, 0.12)))
    ],
    trend: trendPoints('visit-event-track', ctx, 1400),
    rank: shareRows(rows.map((r) => ({ name: `${r.name}-${r.action}`, value: Number(r.times) }))),
    table: {
      columns: [
        { prop: 'name', label: '事件类别', sortable: true },
        { prop: 'action', label: '事件动作', sortable: true },
        { prop: 'label', label: '事件标签', sortable: true },
        { prop: 'times', label: '事件次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'users', label: '独立访客', sortable: true, align: 'right', format: 'int' },
        { prop: 'avgDuration', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(rows, 'times'), ctx.keyword, (r) => `${r.name} ${r.action} ${r.label}`)
    },
    secondary: {
      columns: [
        { prop: 'name', label: '日期', sortable: true },
        { prop: 'times', label: '事件次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'users', label: '触发用户数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: dailyRows
    }
  })
}

/** 事件按天明细用的日期序列：区间末端往前 7 天（不足 7 天按区间起点截断）。 */
function eachDayLabels(ctx: ModuleQueryContext): string[] {
  return eachDay(addDays(ctx.endDate, -6), ctx.endDate, 7)
}

// ---- 子目录（custom/subdir） ----

export function genSubDir(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('visit-subdir', ctx))
  const scale = scaleOf(ctx)
  const pv = Math.round(64000 * scale * (0.8 + rng() * 0.4))
  const rows = SUBDIRS.map((name) => {
    const sPv = Math.max(1, Math.round((pv / SUBDIRS.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      pv: sPv,
      uv: Math.round(sPv * (0.5 + rng() * 0.3)),
      avgStay: intBetween(rng, 10, 420),
      bounceRate: floatBetween(rng, 0.25, 0.7),
      exitRate: floatBetween(rng, 0.15, 0.7),
      conv: Math.round(sPv * floatBetween(rng, 0.002, 0.04))
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'subdirs', '子目录数', fmtInt(SUBDIRS.length)),
      card(rng, 'pv', '浏览量 PV', fmtInt(pv)),
      card(rng, 'uv', '访客数 UV', fmtInt(pv * (0.55 + rng() * 0.15))),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.6)))
    ],
    trend: trendPoints('visit-subdir', ctx, 9200),
    rank: shareRows(rows.map((r) => ({ name: String(r.name), value: Number(r.pv) }))),
    table: {
      columns: [
        { prop: 'name', label: '子目录', sortable: true },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'avgStay', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'exitRate', label: '退出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'conv', label: '转化次数', sortable: true, align: 'right', format: 'int' }
      ],
      rows: filterByKeyword(sortByValue(rows, 'pv'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 转化概况（trans/overview） ----

export function genTransOverview(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('trans-overview', ctx))
  const scale = scaleOf(ctx)
  const visits = Math.round(42000 * scale * (0.8 + rng() * 0.4))
  const conv = Math.max(1, Math.round(visits * floatBetween(rng, 0.02, 0.07)))
  const revenue = Math.round(conv * (60 + rng() * 220))
  const goalRows = GOALS.map((name) => {
    const g = Math.max(1, Math.round((conv / GOALS.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      times: g,
      rate: g / visits,
      revenue: Math.round(g * (50 + rng() * 240)),
      cost: Number((8 + rng() * 46).toFixed(2))
    }
  })
  const sourceRows = SOURCE_TYPES.map((name) => {
    const s = Math.max(1, Math.round((conv / SOURCE_TYPES.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      times: s,
      revenue: Math.round(s * (50 + rng() * 240)),
      share: s / conv
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'conv', '转化次数', fmtInt(conv)),
      card(rng, 'convRate', '转化率', fmtPct(conv / visits)),
      card(rng, 'revenue', '转化收益(元)', fmtInt(revenue)),
      card(rng, 'cost', '平均转化成本(元)', (8 + rng() * 40).toFixed(2))
    ],
    trend: trendPoints('trans-overview', ctx, 5800),
    funnel: funnelSteps(rng, ['访问网站', '浏览商品', '提交订单', '支付成功'], visits),
    rank: shareRows(goalRows.map((r) => ({ name: String(r.name), value: Number(r.times) }))),
    table: {
      columns: [
        { prop: 'name', label: '转化目标', sortable: true },
        { prop: 'times', label: '转化次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'rate', label: '转化率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'revenue', label: '转化收益(元)', sortable: true, align: 'right', format: 'int' },
        { prop: 'cost', label: '平均转化成本(元)', sortable: true, align: 'right', format: 'float2' }
      ],
      rows: filterByKeyword(sortByValue(goalRows, 'times'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '来源渠道', sortable: true },
        { prop: 'times', label: '转化次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'revenue', label: '转化收益(元)', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(sortByValue(sourceRows, 'times'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 转化路径（custom/newpath） ----

export function genConversionPath(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('conversion-path', ctx))
  const scale = scaleOf(ctx)
  const visits = Math.round(26000 * scale * (0.8 + rng() * 0.4))
  const pathNames = [
    '入口页 → 产品页 → 注册页',
    '搜索 → 详情页 → 下单支付',
    '活动页 → 咨询 → 留资',
    '首页 → 价格页 → 注册页',
    '文章页 → 下载页 → 激活',
    '案例页 → 咨询 → 下单支付'
  ]
  const rows = pathNames.map((name) => {
    const users = Math.max(1, Math.round((visits / pathNames.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      users,
      times: Math.round(users * floatBetween(rng, 0.2, 0.8)),
      rate: floatBetween(rng, 0.15, 0.75),
      avgDuration: intBetween(rng, 60, 900)
    }
  })
  const stepRows = ['入口页', '产品/详情页', '注册/下单页', '支付成功页'].map((name, i) => {
    const enter = Math.max(1, Math.round((visits / (i + 1)) * (0.4 + rng() * 0.6)))
    const lost = Math.round(enter * floatBetween(rng, 0.15, 0.6))
    return { name, enter, lost, rate: lost / enter }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'users', '转化用户数', fmtInt(visits * floatBetween(rng, 0.02, 0.08))),
      card(rng, 'paths', '转化路径数', fmtInt(24 + rng() * 80)),
      card(rng, 'steps', '平均步骤数', (2.6 + rng() * 1.8).toFixed(2)),
      card(rng, 'finish', '路径完成率', fmtPct(floatBetween(rng, 0.15, 0.55)))
    ],
    trend: trendPoints('conversion-path', ctx, 3400),
    funnel: funnelSteps(rng, ['入口页', '产品页', '注册页', '支付成功'], visits),
    table: {
      columns: [
        { prop: 'name', label: '转化路径', sortable: true },
        { prop: 'users', label: '用户数', sortable: true, align: 'right', format: 'int' },
        { prop: 'times', label: '转化次数', sortable: true, align: 'right', format: 'int' },
        { prop: 'rate', label: '完成率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均转化时长', sortable: true, align: 'right', format: 'duration' }
      ],
      rows: filterByKeyword(sortByValue(rows, 'users'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '步骤', sortable: true },
        { prop: 'enter', label: '进入人数', sortable: true, align: 'right', format: 'int' },
        { prop: 'lost', label: '流失人数', sortable: true, align: 'right', format: 'int' },
        { prop: 'rate', label: '流失率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(stepRows, ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 地域分布（visit/district） ----

export function genVisitDistrict(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('visit-district', ctx))
  const scale = scaleOf(ctx)
  const uv = Math.round(46000 * scale * (0.8 + rng() * 0.4))
  const geoRows: GeoRow[] = AREA_NAMES.map((name, i) => {
    const values: MetricValues = {
      ...zeroMetricValues(),
      pv: Math.max(1, Math.round((uv / AREA_NAMES.length) * (0.3 + rng() * 1.7) * 1.6)),
      uv: Math.max(1, Math.round((uv / AREA_NAMES.length) * (0.3 + rng() * 1.7))),
      ip: Math.max(1, Math.round((uv / AREA_NAMES.length) * (0.3 + rng() * 1.5))),
      visits: Math.max(1, Math.round((uv / AREA_NAMES.length) * (0.4 + rng() * 1.6))),
      bounceRate: floatBetween(rng, 0.25, 0.65),
      avgDuration: intBetween(rng, 30, 420),
      avgPages: Number((1.5 + rng() * 2.5).toFixed(2)),
      conversionRate: floatBetween(rng, 0.005, 0.05)
    }
    return { name, code: `CN-${String(i + 1).padStart(2, '0')}`, values }
  })
  const areaRows = geoRows.map((g) => ({
    name: g.name,
    uv: g.values.uv,
    pv: g.values.pv,
    bounceRate: g.values.bounceRate,
    avgDuration: g.values.avgDuration,
    newRate: floatBetween(rng, 0.3, 0.8)
  }))
  const countryRows = COUNTRY_NAMES.map((name) => {
    const c = Math.max(1, Math.round((uv / COUNTRY_NAMES.length) * (0.3 + rng() * 1.7)))
    return { name, uv: c, pv: Math.round(c * (1.3 + rng() * 0.8)), share: c / uv }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'uv', '访客数 UV', fmtInt(uv)),
      card(rng, 'provinces', '覆盖省份', fmtInt(AREA_NAMES.length)),
      card(rng, 'pv', '浏览量 PV', fmtInt(uv * (1.4 + rng() * 0.5))),
      card(rng, 'bounceRate', '跳出率', fmtPct(floatBetween(rng, 0.3, 0.6))),
      card(rng, 'avgDuration', '平均停留时长', `${intBetween(rng, 1, 4)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`),
      card(rng, 'newRate', '新访客比率', fmtPct(0.3 + rng() * 0.3))
    ],
    trend: trendPoints('visit-district', ctx, 6600),
    geo: geoRows,
    table: {
      columns: [
        { prop: 'name', label: '地域', sortable: true },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' },
        { prop: 'newRate', label: '新访客比率', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(sortByValue(areaRows, 'uv'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '国家/地区', sortable: true },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'pv', label: '浏览量(PV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(sortByValue(countryRows, 'uv'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 系统环境（visit/client） ----

export function genVisitClient(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('visit-client', ctx))
  const scale = scaleOf(ctx)
  const uv = Math.round(44000 * scale * (0.8 + rng() * 0.4))
  const browserRows = BROWSERS.map((name) => {
    const v = Math.max(1, Math.round((uv / BROWSERS.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      uv: v,
      share: v / uv,
      bounceRate: floatBetween(rng, 0.25, 0.7),
      avgDuration: intBetween(rng, 20, 420)
    }
  })
  const osRows = OPERATING_SYSTEMS.map((name) => {
    const v = Math.max(1, Math.round((uv / OPERATING_SYSTEMS.length) * (0.3 + rng() * 1.7)))
    return {
      name,
      uv: v,
      share: v / uv,
      bounceRate: floatBetween(rng, 0.25, 0.7),
      avgDuration: intBetween(rng, 20, 420)
    }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'uv', '访客数 UV', fmtInt(uv)),
      card(rng, 'mobile', '移动设备占比', fmtPct(0.4 + rng() * 0.3)),
      card(rng, 'pc', '计算机占比', fmtPct(0.25 + rng() * 0.25)),
      card(rng, 'avgDuration', '平均停留时长', `${intBetween(rng, 1, 4)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`)
    ],
    trend: trendPoints('visit-client', ctx, 6300),
    rank: shareRows(
      RESOLUTIONS.map((name) => ({ name, value: Math.max(1, Math.round((uv / RESOLUTIONS.length) * (0.3 + rng() * 1.6))) }))
    ),
    table: {
      columns: [
        { prop: 'name', label: '浏览器', sortable: true },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' }
      ],
      rows: filterByKeyword(sortByValue(browserRows, 'uv'), ctx.keyword, (r) => String(r.name))
    },
    secondary: {
      columns: [
        { prop: 'name', label: '操作系统', sortable: true },
        { prop: 'uv', label: '访客数(UV)', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' },
        { prop: 'bounceRate', label: '跳出率', sortable: true, align: 'right', format: 'percent' },
        { prop: 'avgDuration', label: '平均停留时长', sortable: true, align: 'right', format: 'duration' }
      ],
      rows: filterByKeyword(sortByValue(osRows, 'uv'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 新老访客（visit/type） ----

export function genVisitType(ctx: ModuleQueryContext): ReportPageData {
  const rng = rngFor(seedKey('visit-type', ctx))
  const scale = scaleOf(ctx)
  const uv = Math.round(46000 * scale * (0.8 + rng() * 0.4))
  const newUv = Math.round(uv * floatBetween(rng, 0.55, 0.8))
  const oldUv = uv - newUv
  const pv = Math.round(uv * (1.4 + rng() * 0.5))
  const newPv = Math.round(pv * floatBetween(rng, 0.5, 0.75))
  const compareRows: TableRow[] = [
    { name: '访客数(UV)', newValue: newUv, oldValue: oldUv, newShare: newUv / uv },
    { name: '浏览量(PV)', newValue: newPv, oldValue: pv - newPv, newShare: newPv / pv },
    { name: '访问次数', newValue: Math.round(newUv * (1.1 + rng() * 0.5)), oldValue: Math.round(oldUv * (1.3 + rng() * 0.7)), newShare: 0.6 },
    { name: '跳出率', newValue: fmtPct(floatBetween(rng, 0.35, 0.6)), oldValue: fmtPct(floatBetween(rng, 0.2, 0.45)), newShare: 0 },
    { name: '平均停留时长', newValue: `${intBetween(rng, 0, 3)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`, oldValue: `${intBetween(rng, 1, 5)}:${String(intBetween(rng, 10, 59)).padStart(2, '0')}`, newShare: 0 },
    { name: '转化次数', newValue: Math.round(newUv * floatBetween(rng, 0.01, 0.05)), oldValue: Math.round(oldUv * floatBetween(rng, 0.02, 0.08)), newShare: 0.55 }
  ]
  const sourceRows = SOURCE_TYPES.map((name) => {
    const n = Math.max(1, Math.round((newUv / SOURCE_TYPES.length) * (0.3 + rng() * 1.7)))
    const o = Math.max(1, Math.round((oldUv / SOURCE_TYPES.length) * (0.3 + rng() * 1.7)))
    return { name, newValue: n, oldValue: o, share: (n + o) / uv }
  })
  return applyEmptyMode({
    cards: [
      card(rng, 'uv', '访客数 UV', fmtInt(uv)),
      card(rng, 'new', '新访客', fmtInt(newUv)),
      card(rng, 'old', '老访客', fmtInt(oldUv)),
      card(rng, 'returnRate', '回访率', fmtPct(uv > 0 ? oldUv / uv : 0))
    ],
    trend: trendPoints('visit-type', ctx, 6600),
    rank: shareRows([
      { name: '新访客', value: newUv },
      { name: '老访客', value: oldUv }
    ]),
    table: {
      columns: [
        { prop: 'name', label: '指标', sortable: true },
        { prop: 'newValue', label: '新访客', sortable: true, align: 'right' },
        { prop: 'oldValue', label: '老访客', sortable: true, align: 'right' },
        { prop: 'newShare', label: '新访客占比', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: compareRows
    },
    secondary: {
      columns: [
        { prop: 'name', label: '来源', sortable: true },
        { prop: 'newValue', label: '新访客', sortable: true, align: 'right', format: 'int' },
        { prop: 'oldValue', label: '老访客', sortable: true, align: 'right', format: 'int' },
        { prop: 'share', label: '占比', sortable: true, align: 'right', format: 'percent' }
      ],
      rows: filterByKeyword(sortByValue(sourceRows, 'newValue'), ctx.keyword, (r) => String(r.name))
    }
  })
}

// ---- 汇总分发：按报表页 key 取数（测试与通用入口用） ----

const GENERATORS: Record<ReportPageKey, (ctx: ModuleQueryContext) => ReportPageData> = {
  overview: genOverviewIndex,
  'traffic-latest': genTrafficLatest,
  'traffic-time': genTrendTime,
  'source-all': genSourceAll,
  'source-engine': genSourceEngine,
  'source-searchword': genSourceSearchWord,
  'source-link': genSourceLink,
  'source-inner': genSourceInner,
  'visit-toppage': genVisitTopPage,
  'visit-landingpage': genVisitLandingPage,
  'visit-topdomain': genVisitTopDomain,
  'visit-pageclick': genPageClick,
  'visit-flow': genPageFlow,
  'visit-event-track': genEventTrack,
  'visit-subdir': genSubDir,
  'conversion-overview': genTransOverview,
  'conversion-path': genConversionPath,
  'visitor-district': genVisitDistrict,
  'visitor-client': genVisitClient,
  'visitor-type': genVisitType
}

export function genReportPage(key: ReportPageKey, ctx: ModuleQueryContext): ReportPageData {
  return GENERATORS[key](ctx)
}
