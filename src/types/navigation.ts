// 菜单与哈希路由的唯一定义：侧边菜单、路由解析、面包屑都从这里派生。
// 不引入 vue-router：需求只是菜单切换与状态同步，轻量哈希路由足够且零依赖。

export type MenuKey =
  | 'overview'
  | 'promotion'
  | 'traffic'
  | 'source'
  | 'visit'
  | 'conversion'
  | 'visitor'
  | 'optimize'
  | 'heatmap'
  | 'custom'
  | 'export'
  | 'account'

export interface MenuItem {
  key: MenuKey
  label: string
  /** 哈希路径：#/overview */
  path: string
  desc: string
}

export const MENU_ITEMS: MenuItem[] = [
  { key: 'overview', label: '网站概况', path: 'overview', desc: '汇总网站运营表现与 TOP 榜单' },
  { key: 'promotion', label: '百度推广', path: 'promotion', desc: '搜索/网盟推广从点击到转化的效果' },
  { key: 'traffic', label: '流量分析', path: 'traffic', desc: '按时间变化的访问情况与实时访客' },
  { key: 'source', label: '来源分析', path: 'source', desc: '直接访问/搜索引擎/外部链接等渠道' },
  { key: 'visit', label: '访问分析', path: 'visit', desc: '受访/入口/退出页面与页面路径' },
  { key: 'conversion', label: '转化分析', path: 'conversion', desc: '转化概况、漏斗、订单与收益' },
  { key: 'visitor', label: '访客分析', path: 'visitor', desc: '地域分布、新老访客与 PV/UV 报表' },
  { key: 'optimize', label: '优化分析', path: 'optimize', desc: '搜索词、索引量、速度诊断与抓取异常' },
  { key: 'heatmap', label: '热力图与点击分析', path: 'heatmap', desc: '页面点击热图与元素交互' },
  { key: 'custom', label: '自定义报告', path: 'custom', desc: '自由组合维度与指标的日常监控报表' },
  { key: 'export', label: '数据导出与开放平台', path: 'export', desc: '报告下载、Data API 与 JS API' },
  { key: 'account', label: '账户与权限管理', path: 'account', desc: '多账户管理、报告授权与数据权限' }
]

export const MENU_MAP: Record<MenuKey, MenuItem> = Object.fromEntries(
  MENU_ITEMS.map((m) => [m.key, m])
) as Record<MenuKey, MenuItem>

/** 默认路由：访客分析（即原 PV/UV 地域报表页），保证首屏与旧测试不变。 */
export const DEFAULT_MENU_KEY: MenuKey = 'visitor'

export function menuKeyFromHash(hash: string): MenuKey {
  const path = hash.replace(/^#\/?/, '').split('?')[0].trim()
  return MENU_ITEMS.find((m) => m.path === path)?.key ?? DEFAULT_MENU_KEY
}

export function hashForKey(key: MenuKey): string {
  return `#/${MENU_MAP[key].path}`
}

export function breadcrumbFor(key: MenuKey): string[] {
  return ['数据统计', MENU_MAP[key].label]
}
