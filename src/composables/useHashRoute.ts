// 轻量哈希路由：读写 location.hash，与侧边菜单双向同步。
// 模块级单例：多次调用共用同一状态，只注册一次 hashchange 监听。

import { ref } from 'vue'
import type { MenuKey } from '../types/navigation'
import { DEFAULT_MENU_KEY, hashForKey, menuKeyFromHash } from '../types/navigation'

const route = ref<MenuKey>(DEFAULT_MENU_KEY)
let started = false

function syncFromLocation(): void {
  route.value = menuKeyFromHash(window.location.hash)
}

export function useHashRoute() {
  if (!started) {
    started = true
    syncFromLocation()
    window.addEventListener('hashchange', syncFromLocation)
  }

  /** 跳转：先同步更新状态（测试与界面立即生效），再写 hash（可回退/可分享）。 */
  function navigate(key: MenuKey): void {
    route.value = key
    const target = hashForKey(key)
    if (window.location.hash !== target) window.location.hash = target
  }

  return { route, navigate }
}

/** 测试用：回到默认路由并清空 hash。 */
export function resetHashRoute(): void {
  route.value = DEFAULT_MENU_KEY
  if (window.location.hash !== '' && window.location.hash !== '#/') {
    window.location.hash = ''
  }
  route.value = menuKeyFromHash(window.location.hash)
}
