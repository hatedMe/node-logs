<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElMessage } from 'element-plus'
import type { AccountData, TableRow } from '../types/modules'
import { formatCellValue } from '../types/modules'
import { ACCOUNT_ROLES, genAccount } from '../mocks/modules'
import { useModulePage } from '../composables/useModulePage'
import { toCsv } from '../utils/csv'
import FilterBar from '../components/common/FilterBar.vue'
import StatCards from '../components/common/StatCards.vue'
import PageTable from '../components/common/PageTable.vue'

// 角色筛选与新增成员是本页局部状态：筛选实时过滤表格，新增只存内存。
const roleFilter = ref(ACCOUNT_ROLES[0])
const addedUsers = ref<TableRow[]>([])
const dialogVisible = ref(false)
const newName = ref('')
const newRole = ref('只读')

function buildCsv(d: AccountData): string {
  const t = d.users
  const all = [...t.rows, ...addedUsers.value]
  return toCsv(
    t.columns.map((c) => c.label),
    all.map((r) => t.columns.map((c) => formatCellValue(c.format, r[c.prop] ?? '-')))
  )
}

const page = useModulePage<AccountData>({
  fetchData: (ctx) => genAccount(ctx),
  buildCsv,
  exportFilename: 'account-users.csv',
  hasContent: (d) => !!d && (d.cards.length > 0 || d.users.rows.length + addedUsers.value.length > 0)
})

const filteredUsers = computed(() => {
  const all = [...(page.data.value?.users.rows ?? []), ...addedUsers.value]
  if (roleFilter.value === ACCOUNT_ROLES[0]) return all
  return all.filter((r) => r.role === roleFilter.value)
})

function onAddMember(): void {
  const name = newName.value.trim()
  if (!name) {
    ElMessage.warning('请填写成员名称')
    return
  }
  if ([...(page.data.value?.users.rows ?? []), ...addedUsers.value].some((r) => r.name === name)) {
    ElMessage.warning('成员已存在')
    return
  }
  addedUsers.value = [
    ...addedUsers.value,
    { name, role: newRole.value, sites: '演示站点A', status: '启用', lastLogin: '从未登录' }
  ]
  dialogVisible.value = false
  newName.value = ''
  ElMessage.success(`已添加成员 ${name}`)
}
</script>

<template>
  <div class="module-page" data-testid="view-account">
    <header class="module-header">
      <h2>账户与权限管理</h2>
      <p>多账户统一管理：成员角色、报告授权与网站数据权限控制</p>
    </header>

    <el-card class="module-section" shadow="never">
      <FilterBar
        v-model:filters="page.filters.value"
        :loading="page.loading.value"
        :can-export="page.hasContent.value"
        :show-source="false"
        :show-device="false"
        show-keyword
        keyword-placeholder="搜索成员/角色"
        @submit="page.onQuery"
        @reset="page.onReset"
        @export="page.onExport"
      />
      <el-form label-width="84px">
        <el-form-item label="角色筛选">
          <el-select v-model="roleFilter" data-testid="account-role">
            <el-option v-for="r in ACCOUNT_ROLES" :key="r" :value="r" :label="r" />
          </el-select>
          <el-button type="primary" class="add-btn" @click="dialogVisible = true">添加成员</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-alert
      v-if="page.error.value"
      class="module-section"
      type="error"
      :title="page.error.value"
      show-icon
      :closable="false"
    >
      <el-button size="small" @click="page.onRetry">重试</el-button>
    </el-alert>

    <section class="module-section" aria-label="账户指标">
      <StatCards :cards="page.data.value?.cards ?? []" :loading="page.loading.value" />
    </section>

    <el-card class="module-section" shadow="never">
      <template #header><span>成员列表</span></template>
      <PageTable
        :rows="filteredUsers"
        :columns="page.data.value?.users.columns ?? []"
        :loading="page.loading.value"
        drawer-title="成员明细"
      />
    </el-card>

    <el-dialog v-model="dialogVisible" title="添加成员" width="360px">
      <el-form label-width="70px">
        <el-form-item label="成员名">
          <el-input v-model="newName" placeholder="如 li.si" />
        </el-form-item>
        <el-form-item label="角色">
          <el-select v-model="newRole">
            <el-option v-for="r in ACCOUNT_ROLES.slice(1)" :key="r" :value="r" :label="r" />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="onAddMember">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.add-btn {
  margin-left: 12px;
}
</style>
