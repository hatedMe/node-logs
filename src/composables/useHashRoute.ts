// 轻量哈希路由：读写 location.hash，与侧边菜单双向同步。
// 路由粒度 = 百度统计演示站的报表页（如 #/source/searchword），模块级单例避免重复注册监听。

import { ref } from 'vue'
import type { ReportPageKey } from '../types/navigation'
import { DEFAULT_PAGE_KEY, hashForPage, pageKeyFromHash } from '../types/navigation'

const route = ref<ReportPageKey>(DEFAULT_PAGE_KEY)
let started = false

function syncFromLocation(): void {
  route.value = pageKeyFromHash(window.location.hash)
}

export function useHashRoute() {
  if (!started) {
    started = true
    syncFromLocation()
    window.addEventListener('hashchange', syncFromLocation)
  }

  /** 跳转：先同步更新状态（测试与界面立即生效），再写 hash（可回退/可分享）。 */
  function navigate(key: ReportPageKey): void {
    route.value = key
    const target = hashForPage(key)
    if (window.location.hash !== target) window.location.hash = target
  }

  return { route, navigate }
}

/** 测试用：回到默认路由并清空 hash。 */
export function resetHashRoute(): void {
  route.value = DEFAULT_PAGE_KEY
  if (window.location.hash !== '' && window.location.hash !== '#/') {
    window.location.hash = ''
  }
  route.value = pageKeyFromHash(window.location.hash)
}
