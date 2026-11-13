import { afterEach, describe, expect, it } from 'vitest'
import { nextTick } from 'vue'
import {
  DEFAULT_MENU_KEY,
  MENU_ITEMS,
  breadcrumbFor,
  hashForKey,
  menuKeyFromHash
} from '../types/navigation'
import { resetHashRoute, useHashRoute } from '../composables/useHashRoute'

afterEach(() => {
  resetHashRoute()
})

describe('菜单定义', () => {
  it('覆盖文档要求的 12 个产品菜单', () => {
    const labels = MENU_ITEMS.map((m) => m.label)
    expect(MENU_ITEMS).toHaveLength(12)
    for (const expected of [
      '网站概况',
      '百度推广',
      '流量分析',
      '来源分析',
      '访问分析',
      '转化分析',
      '访客分析',
      '优化分析',
      '热力图与点击分析',
      '自定义报告',
      '数据导出与开放平台',
      '账户与权限管理'
    ]) {
      expect(labels).toContain(expected)
    }
    // key 与 path 均唯一
    expect(new Set(MENU_ITEMS.map((m) => m.key)).size).toBe(12)
    expect(new Set(MENU_ITEMS.map((m) => m.path)).size).toBe(12)
  })

  it('默认路由是访客分析（原 PV/UV 页），保证首屏与旧测试不变', () => {
    expect(DEFAULT_MENU_KEY).toBe('visitor')
  })

  it('面包屑为两级：数据统计 / 菜单名', () => {
    expect(breadcrumbFor('overview')).toEqual(['数据统计', '网站概况'])
    expect(breadcrumbFor('account')).toEqual(['数据统计', '账户与权限管理'])
  })
})

describe('哈希解析', () => {
  it('空哈希与未知路径回退到默认路由', () => {
    expect(menuKeyFromHash('')).toBe('visitor')
    expect(menuKeyFromHash('#/')).toBe('visitor')
    expect(menuKeyFromHash('#/not-exist')).toBe('visitor')
  })

  it('正常路径与 key 双向映射', () => {
    for (const m of MENU_ITEMS) {
      expect(menuKeyFromHash(`#/${m.path}`)).toBe(m.key)
      expect(hashForKey(m.key)).toBe(`#/${m.path}`)
    }
  })
})

describe('路由状态同步', () => {
  it('navigate 更新状态并写入 hash', async () => {
    const { route, navigate } = useHashRoute()
    navigate('traffic')
    await nextTick()
    expect(route.value).toBe('traffic')
    expect(window.location.hash).toBe('#/traffic')
  })

  it('浏览器 hash 变化（回退/前进）同步回菜单状态', async () => {
    const { route } = useHashRoute()
    window.location.hash = '#/source'
    window.dispatchEvent(new Event('hashchange'))
    await nextTick()
    expect(route.value).toBe('source')
  })
})
