// 菜单与哈希路由的唯一定义：侧边菜单、路由解析、面包屑都从这里派生。
// 菜单结构与百度统计演示站（siteId=16847648）的报表目录一一对应：
// 6 个菜单组、20 个报表页，每个报表页的哈希路径 = 演示站 URL 的路径段。

export type ReportPageKey =
  | 'overview'
  | 'traffic-latest'
  | 'traffic-time'
  | 'source-all'
  | 'source-engine'
  | 'source-searchword'
  | 'source-link'
  | 'source-inner'
  | 'visit-toppage'
  | 'visit-landingpage'
  | 'visit-topdomain'
  | 'visit-pageclick'
  | 'visit-flow'
  | 'visit-event-track'
  | 'visit-subdir'
  | 'conversion-overview'
  | 'conversion-path'
  | 'visitor-district'
  | 'visitor-client'
  | 'visitor-type'

export type MenuGroupKey = 'overview' | 'traffic' | 'source' | 'visit' | 'conversion' | 'visitor'

export interface NavigationChild {
  key: ReportPageKey
  label: string
  /** 哈希路径段：#/trend/latest */
  path: string
  /** 百度统计演示站原始 URL，还原页面时的对照来源。 */
  url: string
  /** 所属菜单组。 */
  parentKey: MenuGroupKey
}

export interface NavigationGroup {
  key: MenuGroupKey
  label: string
  children: NavigationChild[]
}

const TONGJI_BASE = 'https://tongji.baidu.com/main/overview/demo'
const demoUrl = (path: string): string => `${TONGJI_BASE}/${path}?siteId=16847648`

/** 百度统计演示站点的报表目录：URL 与演示站菜单保持一致。 */
export const NAVIGATION_GROUPS: NavigationGroup[] = [
  {
    key: 'overview',
    label: '网站概况',
    children: [{ key: 'overview', label: '网站概况', parentKey: 'overview', path: 'overview/index', url: demoUrl('overview/index') }]
  },
  {
    key: 'traffic',
    label: '流量分析',
    children: [
      { key: 'traffic-latest', label: '实时访客', parentKey: 'traffic', path: 'trend/latest', url: demoUrl('trend/latest') },
      { key: 'traffic-time', label: '趋势分析', parentKey: 'traffic', path: 'trend/time', url: demoUrl('trend/time') }
    ]
  },
  {
    key: 'source',
    label: '来源分析',
    children: [
      { key: 'source-all', label: '全部来源', parentKey: 'source', path: 'source/all', url: demoUrl('source/all') },
      { key: 'source-engine', label: '搜索引擎', parentKey: 'source', path: 'source/engine', url: demoUrl('source/engine') },
      { key: 'source-searchword', label: '搜索词', parentKey: 'source', path: 'source/searchword', url: demoUrl('source/searchword') },
      { key: 'source-link', label: '外部链接', parentKey: 'source', path: 'source/link', url: demoUrl('source/link') },
      { key: 'source-inner', label: '站内来源', parentKey: 'source', path: 'source/inner', url: demoUrl('source/inner') }
    ]
  },
  {
    key: 'visit',
    label: '访问分析',
    children: [
      { key: 'visit-toppage', label: '受访页面', parentKey: 'visit', path: 'visit/toppage', url: demoUrl('visit/toppage') },
      { key: 'visit-landingpage', label: '入口页面', parentKey: 'visit', path: 'visit/landingpage', url: demoUrl('visit/landingpage') },
      { key: 'visit-topdomain', label: '受访域名', parentKey: 'visit', path: 'visit/topdomain', url: demoUrl('visit/topdomain') },
      { key: 'visit-pageclick', label: '页面点击图', parentKey: 'visit', path: 'custom/pageclick', url: demoUrl('custom/pageclick') },
      { key: 'visit-flow', label: '页面上下游', parentKey: 'visit', path: 'custom/flow', url: demoUrl('custom/flow') },
      { key: 'visit-event-track', label: '事件分析', parentKey: 'visit', path: 'custom/event_track', url: demoUrl('custom/event_track') },
      { key: 'visit-subdir', label: '子目录', parentKey: 'visit', path: 'custom/subdir', url: demoUrl('custom/subdir') }
    ]
  },
  {
    key: 'conversion',
    label: '转化分析',
    children: [
      { key: 'conversion-overview', label: '转化概况', parentKey: 'conversion', path: 'trans/overview', url: demoUrl('trans/overview') },
      { key: 'conversion-path', label: '转化路径', parentKey: 'conversion', path: 'custom/newpath', url: demoUrl('custom/newpath') }
    ]
  },
  {
    key: 'visitor',
    label: '访客分析',
    children: [
      { key: 'visitor-district', label: '地域分布', parentKey: 'visitor', path: 'visit/district', url: demoUrl('visit/district') },
      { key: 'visitor-client', label: '系统环境', parentKey: 'visitor', path: 'visit/client', url: demoUrl('visit/client') },
      { key: 'visitor-type', label: '新老访客', parentKey: 'visitor', path: 'visit/type', url: demoUrl('visit/type') }
    ]
  }
]

export const NAVIGATION_ITEMS: NavigationChild[] = NAVIGATION_GROUPS.flatMap((group) => group.children)

export const PAGE_KEYS: ReportPageKey[] = NAVIGATION_ITEMS.map((item) => item.key)

const CHILD_BY_KEY: Record<ReportPageKey, NavigationChild> = Object.fromEntries(
  NAVIGATION_ITEMS.map((item) => [item.key, item])
) as Record<ReportPageKey, NavigationChild>

const CHILD_BY_PATH: Record<string, NavigationChild> = Object.fromEntries(
  NAVIGATION_ITEMS.map((item) => [item.path, item])
)

/** 旧顶层模块路径的兼容映射：#/#/traffic 等旧链接落到该组的首个报表页。 */
const LEGACY_PATH_ALIASES: Record<string, ReportPageKey> = {
  overview: 'overview',
  traffic: 'traffic-latest',
  source: 'source-all',
  visit: 'visit-toppage',
  conversion: 'conversion-overview',
  visitor: 'visitor-district'
}

/** 默认路由：网站概况（演示站目录的第一项）。 */
export const DEFAULT_PAGE_KEY: ReportPageKey = 'overview'

export function pageMeta(key: ReportPageKey): NavigationChild {
  return CHILD_BY_KEY[key]
}

export function groupOf(key: ReportPageKey): NavigationGroup {
  const child = CHILD_BY_KEY[key]
  return NAVIGATION_GROUPS.find((g) => g.key === child.parentKey) ?? NAVIGATION_GROUPS[0]
}

/** 哈希 → 报表页 key：#/trend/latest → traffic-latest，未知路径回退默认页。 */
export function pageKeyFromHash(hash: string): ReportPageKey {
  const path = hash.replace(/^#\/?/, '').split('?')[0].trim().replace(/\/$/, '')
  const hit = CHILD_BY_PATH[path] ?? CHILD_BY_PATH[`${path}/index`]
  if (hit) return hit.key
  return LEGACY_PATH_ALIASES[path] ?? DEFAULT_PAGE_KEY
}

export function hashForPage(key: ReportPageKey): string {
  return `#/${CHILD_BY_KEY[key].path}`
}

/** 面包屑：数据统计 / 菜单组 / 报表页（单页组省略中间级）。 */
export function breadcrumbForPage(key: ReportPageKey): string[] {
  const child = CHILD_BY_KEY[key]
  const group = groupOf(key)
  return group.children.length > 1 ? ['数据统计', group.label, child.label] : ['数据统计', child.label]
}
