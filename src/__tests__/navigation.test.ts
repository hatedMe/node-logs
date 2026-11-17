import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import {
  DEFAULT_PAGE_KEY,
  NAVIGATION_GROUPS,
  NAVIGATION_ITEMS,
  breadcrumbForPage,
  hashForPage,
  pageKeyFromHash
} from '../types/navigation'
import { resetHashRoute, useHashRoute } from '../composables/useHashRoute'

afterEach(() => {
  resetHashRoute()
})

describe('菜单定义', () => {
  it('覆盖百度统计演示站目录：6 个菜单组、20 个报表页', () => {
    expect(NAVIGATION_GROUPS.map((g) => g.label)).toEqual([
      '网站概况',
      '流量分析',
      '来源分析',
      '访问分析',
      '转化分析',
      '访客分析'
    ])
    expect(NAVIGATION_ITEMS).toHaveLength(20)
    for (const expected of [
      '网站概况',
      '实时访客',
      '趋势分析',
      '全部来源',
      '搜索引擎',
      '搜索词',
      '外部链接',
      '站内来源',
      '受访页面',
      '入口页面',
      '受访域名',
      '页面点击图',
      '页面上下游',
      '事件分析',
      '子目录',
      '转化概况',
      '转化路径',
      '地域分布',
      '系统环境',
      '新老访客'
    ]) {
      expect(NAVIGATION_ITEMS.map((i) => i.label)).toContain(expected)
    }
    // key 与 path 均唯一
    expect(new Set(NAVIGATION_ITEMS.map((i) => i.key)).size).toBe(20)
    expect(new Set(NAVIGATION_ITEMS.map((i) => i.path)).size).toBe(20)
  })

  it('每个报表页保留演示站原始 URL（还原来源）', () => {
    const item = NAVIGATION_ITEMS.find((i) => i.key === 'source-searchword')
    expect(item?.path).toBe('source/searchword')
    expect(item?.url).toBe('https://tongji.baidu.com/main/overview/demo/source/searchword?siteId=16847648')
  })

  it('默认路由是网站概况（演示站目录第一项）', () => {
    expect(DEFAULT_PAGE_KEY).toBe('overview')
  })

  it('面包屑：数据统计 / 菜单组 / 报表页（单页组省略中间级）', () => {
    expect(breadcrumbForPage('overview')).toEqual(['数据统计', '网站概况'])
    expect(breadcrumbForPage('source-searchword')).toEqual(['数据统计', '来源分析', '搜索词'])
    expect(breadcrumbForPage('visitor-type')).toEqual(['数据统计', '访客分析', '新老访客'])
  })
})

describe('哈希解析', () => {
  it('空哈希与未知路径回退到默认页', () => {
    expect(pageKeyFromHash('')).toBe('overview')
    expect(pageKeyFromHash('#/')).toBe('overview')
    expect(pageKeyFromHash('#/not-exist')).toBe('overview')
  })

  it('报表页路径与 key 双向映射（哈希路径 = 演示站 URL 路径）', () => {
    for (const item of NAVIGATION_ITEMS) {
      expect(pageKeyFromHash(`#/${item.path}`)).toBe(item.key)
      expect(hashForPage(item.key)).toBe(`#/${item.path}`)
    }
  })

  it('旧顶层模块路径兼容落到该组首个报表页', () => {
    expect(pageKeyFromHash('#/traffic')).toBe('traffic-latest')
    expect(pageKeyFromHash('#/source')).toBe('source-all')
    expect(pageKeyFromHash('#/visitor')).toBe('visitor-district')
  })
})

describe('路由状态同步', () => {
  it('navigate 更新状态并写入 hash', async () => {
    const { route, navigate } = useHashRoute()
    navigate('source-searchword')
    await nextTick()
    expect(route.value).toBe('source-searchword')
    expect(window.location.hash).toBe('#/source/searchword')
  })

  it('浏览器 hash 变化（回退/前进）同步回菜单状态', async () => {
    const { route } = useHashRoute()
    window.location.hash = '#/visit/toppage'
    window.dispatchEvent(new Event('hashchange'))
    await nextTick()
    expect(route.value).toBe('visit-toppage')
  })
})
