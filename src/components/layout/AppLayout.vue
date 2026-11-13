<script setup lang="ts">
import { computed, ref } from 'vue'
import type { MenuKey } from '../../types/navigation'
import { MENU_ITEMS, breadcrumbFor } from '../../types/navigation'
import { RANGE_PRESET_OPTIONS } from '../../types/report'
import type { RangePreset } from '../../types/report'
import { useAppContext } from '../../composables/useAppContext'

const props = defineProps<{
  activeKey: MenuKey
}>()

const emit = defineEmits<{
  (e: 'navigate', value: MenuKey): void
}>()

const app = useAppContext()
const drawerOpen = ref(false)

const crumbs = computed(() => breadcrumbFor(props.activeKey))

function onSelect(key: string): void {
  const item = MENU_ITEMS.find((m) => m.key === key)
  if (!item) return
  drawerOpen.value = false
  if (item.key !== props.activeKey) emit('navigate', item.key)
}

function onPreset(preset: RangePreset): void {
  app.setPreset(preset)
}
</script>

<template>
  <div class="app-layout">
    <aside class="app-sidebar" aria-label="产品菜单">
      <div class="sidebar-brand">
        <span class="brand-mark" aria-hidden="true">N</span>
        <span class="brand-copy">
          <strong>网站统计</strong>
          <small>NODE LOGS / ANALYTICS</small>
        </span>
      </div>
      <el-menu
        :default-active="activeKey"
        class="side-menu"
        data-testid="side-menu"
        @select="onSelect"
      >
        <el-menu-item v-for="m in MENU_ITEMS" :key="m.key" :index="m.key" :title="m.desc">
          {{ m.label }}
        </el-menu-item>
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
        <div class="topbar-kicker">WORKSPACE / 2026</div>
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
        </div>
      </header>

      <main class="app-content">
        <slot />
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
      <el-menu :default-active="activeKey" data-testid="drawer-menu" @select="onSelect">
        <el-menu-item v-for="m in MENU_ITEMS" :key="m.key" :index="m.key">
          {{ m.label }}
        </el-menu-item>
      </el-menu>
    </el-drawer>
  </div>
</template>
