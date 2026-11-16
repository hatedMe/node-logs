<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { MenuKey } from '../../types/navigation'
import { NAVIGATION_GROUPS, NAVIGATION_ITEMS, breadcrumbFor } from '../../types/navigation'
import { RANGE_PRESET_OPTIONS } from '../../types/report'
import type { RangePreset } from '../../types/report'
import { useAppContext } from '../../composables/useAppContext'
import NavigationReportView from './NavigationReportView.vue'

const props = defineProps<{
  activeKey: MenuKey
}>()

const emit = defineEmits<{
  (e: 'navigate', value: MenuKey): void
}>()

const app = useAppContext()
const drawerOpen = ref(false)
const selectedUrl = ref(NAVIGATION_GROUPS[0].children[0].url)
const selectedItem = computed(() => NAVIGATION_ITEMS.find((item) => item.url === selectedUrl.value) ?? NAVIGATION_ITEMS[0])

const crumbs = computed(() => breadcrumbFor(props.activeKey))
const activeNavigationKey = computed(() => {
  const selected = NAVIGATION_ITEMS.find((item) => item.url === selectedUrl.value)
  return selected?.parentKey === props.activeKey
    ? selected.key
    : NAVIGATION_GROUPS.find((group) => group.key === props.activeKey)?.children[0]?.key ?? props.activeKey
})
const openNavigationKeys = computed(() =>
  NAVIGATION_GROUPS.filter((group) => group.children.length > 1).map((group) => group.key)
)
const activeGlobalNav = computed(() => props.activeKey === 'overview' ? 'overview' : 'analysis')

watch(() => props.activeKey, (key) => {
  const selected = NAVIGATION_ITEMS.find((item) => item.url === selectedUrl.value)
  if (selected?.parentKey === key) return
  const group = NAVIGATION_GROUPS.find((item) => item.key === key)
  if (group?.children[0]) selectedUrl.value = group.children[0].url
}, { immediate: true })

function onSelect(key: string): void {
  const item = NAVIGATION_ITEMS.find((m) => m.key === key)
  if (!item) return
  drawerOpen.value = false
  selectedUrl.value = item.url
  if (item.parentKey !== props.activeKey) emit('navigate', item.parentKey)
}

function onPreset(preset: RangePreset): void {
  app.setPreset(preset)
}
</script>

<template>
  <div class="app-layout">
    <div class="demo-banner">当前展示的是demo站点数据，如果您想进一步了解分析功能，请查看“售前问题”文档 <span>×</span></div>
    <aside class="app-sidebar" aria-label="产品菜单">
      <div class="sidebar-brand">
        <span class="brand-mark" aria-hidden="true"><i />百度<span>统计</span></span>
        <span class="brand-copy">
          <strong>免费版</strong>
          <small>数据统计</small>
        </span>
      </div>
      <el-menu
        :default-active="activeNavigationKey"
        :default-openeds="openNavigationKeys"
        class="side-menu"
        data-testid="side-menu"
        @select="onSelect"
      >
        <template v-for="group in NAVIGATION_GROUPS" :key="group.key">
          <el-menu-item
            v-if="group.children.length === 1"
            :index="group.children[0].key"
            :title="group.label"
          >
            <a class="nav-link" :href="group.children[0].url" @click.prevent="onSelect(group.children[0].key)">
              {{ group.label }}
            </a>
          </el-menu-item>
          <el-sub-menu v-else :index="group.key">
            <template #title>{{ group.label }}</template>
            <el-menu-item v-for="item in group.children" :key="item.key" :index="item.key">
              <a class="nav-link" :href="item.url" @click.prevent="onSelect(item.key)">{{ item.label }}</a>
            </el-menu-item>
          </el-sub-menu>
        </template>
      </el-menu>
    </aside>

    <div class="app-main">
      <header class="app-topbar">
        <el-button
          class="menu-toggle"
          data-testid="menu-toggle"
          aria-label="打开菜单"
          @click="drawerOpen = true"
        >
          ☰
        </el-button>
        <nav class="global-nav" aria-label="全局导航">
          <a class="global-nav__item" :class="{ 'is-active': activeGlobalNav === 'overview' }" href="#/overview">概况</a>
          <a class="global-nav__item" :class="{ 'is-active': activeGlobalNav === 'analysis' }" href="#/traffic">分析</a>
          <a class="global-nav__item" href="#/account">用户管理</a>
          <a class="global-nav__item" href="#/promotion">营销管理</a>
          <a class="global-nav__item" href="#/optimize">智能运营</a>
          <a class="global-nav__item" href="#/export">数据管理</a>
          <a class="global-nav__item" href="#/account">使用设置</a>
        </nav>
        <el-breadcrumb class="app-crumbs" data-testid="breadcrumb" separator="/">
          <el-breadcrumb-item v-for="c in crumbs" :key="c">{{ c }}</el-breadcrumb-item>
        </el-breadcrumb>
        <div class="topbar-context">
          <el-select
            :model-value="app.siteId.value"
            size="small"
            class="site-select"
            data-testid="site-select"
            aria-label="当前站点"
            @update:model-value="app.setSite($event)"
          >
            <el-option v-for="s in app.sites" :key="s.id" :value="s.id" :label="`${s.name}(${s.id})`" />
          </el-select>
          <el-select
            :model-value="app.rangePreset.value"
            size="small"
            class="preset-select"
            data-testid="date-preset"
            aria-label="日期预设"
            @update:model-value="onPreset($event)"
          >
            <el-option v-for="o in RANGE_PRESET_OPTIONS" :key="o.value" :value="o.value" :label="o.label" />
          </el-select>
          <el-date-picker
            :model-value="app.startDate.value"
            type="date"
            size="small"
            value-format="YYYY-MM-DD"
            class="topbar-date"
            aria-label="开始日期"
            @update:model-value="app.setRange($event ?? app.startDate.value, app.endDate.value)"
          />
          <span class="date-sep">至</span>
          <el-date-picker
            :model-value="app.endDate.value"
            type="date"
            size="small"
            value-format="YYYY-MM-DD"
            class="topbar-date"
            aria-label="结束日期"
            @update:model-value="app.setRange(app.startDate.value, $event ?? app.endDate.value)"
          />
          <span class="topbar-icon" aria-label="帮助">?</span>
          <span class="topbar-user">sky蜀朝阳⌄</span>
        </div>
      </header>

      <main class="app-content">
        <NavigationReportView :item="selectedItem" />
      </main>
    </div>

    <!-- 窄屏抽屉菜单：平时不渲染，避免干扰报表页的抽屉测试。 -->
    <el-drawer
      v-if="drawerOpen"
      :model-value="true"
      title="产品菜单"
      direction="ltr"
      size="240px"
      @update:model-value="(v: boolean) => { if (!v) drawerOpen = false }"
    >
      <el-menu
        :default-active="activeNavigationKey"
        :default-openeds="openNavigationKeys"
        data-testid="drawer-menu"
        @select="onSelect"
      >
        <template v-for="group in NAVIGATION_GROUPS" :key="group.key">
          <el-menu-item v-if="group.children.length === 1" :index="group.children[0].key">
            <a class="nav-link" :href="group.children[0].url" @click.prevent="onSelect(group.children[0].key)">{{ group.label }}</a>
          </el-menu-item>
          <el-sub-menu v-else :index="group.key">
            <template #title>{{ group.label }}</template>
            <el-menu-item v-for="item in group.children" :key="item.key" :index="item.key">
              <a class="nav-link" :href="item.url" @click.prevent="onSelect(item.key)">{{ item.label }}</a>
            </el-menu-item>
          </el-sub-menu>
        </template>
      </el-menu>
    </el-drawer>
  </div>
</template>
