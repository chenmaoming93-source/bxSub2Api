<template>
  <AppLayout>
    <div class="space-y-6 p-4 sm:p-6">
      <header class="overflow-hidden rounded-2xl border border-primary-100 bg-gradient-to-br from-primary-50 via-white to-indigo-50 p-6 shadow-sm dark:border-primary-900/40 dark:from-primary-950/40 dark:via-dark-800 dark:to-indigo-950/30">
        <div class="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
          <div>
            <div class="mb-2 flex items-center gap-2 text-sm font-medium text-primary-600 dark:text-primary-400">
              <Icon name="chartBar" size="sm" />
              Token Governance
            </div>
            <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">可配置 Token 统计与限额</h1>
            <p class="mt-2 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">按业务需要组合用户、密钥、分组、路由和上游账号等维度。配置从启用时开始采集，历史数据不会回填；限额创建或重新启用后立即作用于当前自然周期。</p>
          </div>
          <div class="space-y-3 sm:min-w-96">
            <div class="grid grid-cols-3 gap-2">
              <div class="metric-tile"><span>统计项</span><strong>{{ projections.length }}</strong></div>
              <div class="metric-tile"><span>运行中</span><strong class="text-emerald-600">{{ activeProjectionCount }}</strong></div>
              <div class="metric-tile"><span>限额</span><strong>{{ quotas.length }}</strong></div>
            </div>
            <div class="flex items-center justify-between rounded-xl border border-white/70 bg-white/80 px-4 py-3 shadow-sm dark:border-dark-600 dark:bg-dark-800/80">
              <div>
                <div class="flex items-center gap-2 text-sm font-medium text-gray-800 dark:text-gray-100">
                  <span class="status-dot" :class="runtimeState.enabled ? 'bg-emerald-500' : 'bg-gray-400'"></span>
                  全局统计与限额
                </div>
                <p class="mt-1 text-xs text-gray-500">{{ runtimeState.enabled ? '正在采集并执行限额' : '已热关闭，不采集且不限额' }}</p>
              </div>
              <button
                v-if="can('token_usage.manage')"
                class="btn"
                :class="runtimeState.enabled ? 'btn-secondary' : 'btn-primary'"
                :disabled="runtimeSaving || !runtimeState.available"
                @click="toggleRuntime"
              >
                {{ runtimeSaving ? '处理中…' : runtimeState.enabled ? '立即关闭' : '重新开启' }}
              </button>
            </div>
          </div>
        </div>
      </header>

      <div v-if="error" class="flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/30 dark:text-red-300"><Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" /><span>{{ error }}</span></div>
      <div v-if="success" class="flex items-center gap-3 rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-700 dark:border-emerald-900/50 dark:bg-emerald-950/30 dark:text-emerald-300"><Icon name="checkCircle" size="sm" /><span>{{ success }}</span></div>
      <nav class="flex gap-1 overflow-x-auto rounded-xl border border-gray-200 bg-white p-1.5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <button v-for="item in tabs" :key="item.id" class="tab-button" :class="{ 'tab-button-active': tab === item.id }" @click="tab = item.id">
          <Icon :name="item.icon" size="sm" />
          {{ item.label }}
        </button>
      </nav>

      <section v-if="tab === 'projections'" class="space-y-4">
        <form v-if="can('token_usage.manage')" class="card space-y-5 p-5" @submit.prevent="saveProjection">
          <div class="flex items-center justify-between">
            <div><h2 class="section-title">{{ projectionDraft.id ? '编辑统计项草稿' : '新建统计项' }}</h2><p class="section-help">维度组合启用后不可直接修改，需要停用后新建组合。</p></div>
            <span class="rounded-full bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700 dark:bg-primary-950/50 dark:text-primary-300">指标：总 Token</span>
          </div>
          <div>
            <label class="field-label">统计项名称</label>
            <input v-model.trim="projectionDraft.name" class="input max-w-xl" required placeholder="例如：按用户统计" />
          </div>
          <fieldset>
            <legend class="field-label">选择统计维度</legend>
            <div class="grid gap-2 sm:grid-cols-2 xl:grid-cols-3">
              <label v-for="dimension in dimensions" :key="dimension.code" class="dimension-option" :class="{ 'dimension-option-active': projectionDraft.dimension_codes.includes(dimension.code) }">
                <input v-model="projectionDraft.dimension_codes" type="checkbox" :value="dimension.code" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
                <span><b>{{ dimension.display_name }}</b><small>{{ dimension.code }}</small></span>
              </label>
            </div>
          </fieldset>
          <div class="flex justify-end gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
            <button v-if="projectionDraft.id" type="button" class="btn btn-secondary" @click="resetProjection">取消</button>
            <button class="btn btn-primary" :disabled="saving || projectionDraft.dimension_codes.length === 0">{{ saving ? '保存中…' : '保存草稿' }}</button>
          </div>
        </form>
        <div class="card overflow-hidden">
          <div class="table-heading"><div><h2 class="section-title">统计项列表</h2><p class="section-help">只有“运行中”的统计项会接收新的模型调用数据。</p></div><button class="btn btn-secondary" :disabled="loading" @click="load"><Icon name="refresh" size="sm" />刷新</button></div>
          <div class="overflow-x-auto">
          <table class="stat-table">
            <thead><tr><th>名称</th><th>维度</th><th>状态</th><th>统计起点</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="item in projections" :key="item.id">
                <td><div class="font-medium text-gray-900 dark:text-white">{{ item.name }}</div><div class="mt-0.5 text-xs text-gray-400">#{{ item.id }}</div></td>
                <td><div class="flex max-w-xl flex-wrap gap-1.5"><span v-for="code in item.dimension_codes" :key="code" class="dimension-chip">{{ dimensionName(code) }}</span></div></td>
                <td><span class="status-pill" :class="projectionStatusClass(item.status)"><span class="status-dot"></span>{{ projectionStatusLabel(item.status) }}</span></td>
                <td>{{ formatTime(item.enabled_at) }}</td>
                <td><div class="flex flex-wrap gap-2">
                  <button v-if="can('token_usage.manage') && item.status === 'DRAFT'" class="link" @click="editProjection(item)">编辑</button>
                  <button v-if="can('token_usage.manage') && item.status === 'DRAFT'" class="link" @click="projectionAction(item.id, 'publish')">发布</button>
                  <button v-if="can('token_usage.manage') && ['PUBLISHED','DISABLED'].includes(item.status)" class="link" @click="projectionAction(item.id, 'activate')">{{ item.status === 'DISABLED' ? '重新启用' : '启用' }}</button>
                  <button v-if="can('token_usage.manage') && ['PUBLISHED','ACTIVE'].includes(item.status)" class="link text-red-600" @click="projectionAction(item.id, 'disable')">停用</button>
                </div></td>
              </tr>
              <tr v-if="!loading && projections.length === 0"><td colspan="5"><div class="empty-state"><Icon name="chartBar" size="lg" /><b>还没有统计项</b><span>创建一个维度组合并依次发布、启用后开始采集。</span></div></td></tr>
            </tbody>
          </table>
          </div>
        </div>
      </section>

      <section v-else-if="tab === 'quotas'" class="space-y-4">
        <form v-if="can('token_quota.update')" class="card space-y-5 p-5" @submit.prevent="createQuota">
          <div><h2 class="section-title">新建限额规则</h2><p class="section-help">限额与统计项相互独立；缺少相同维度组合时会先创建等待中的统计项。</p></div>
          <div class="grid gap-3 md:grid-cols-4">
            <input v-model.trim="quotaDraft.name" class="input" required placeholder="限额名称" />
            <select v-model="quotaDraft.period_type" class="input"><option value="D">日</option><option value="W">周</option><option value="M">月</option></select>
            <select v-model="quotaDraft.mode" class="input"><option value="OBSERVE">仅观察</option><option value="ENFORCE">强制限制</option></select>
            <input v-model.number="quotaDraft.limit_value" class="input" type="number" min="1" required placeholder="Token 上限" />
          </div>
          <div class="grid gap-3 md:grid-cols-3">
            <label v-for="dimension in dimensions" :key="dimension.code" class="dimension-option" :class="{ 'dimension-option-active': quotaDraft.dimension_codes.includes(dimension.code) }">
              <input v-model="quotaDraft.dimension_codes" type="checkbox" :value="dimension.code" @change="onQuotaDimensionToggle(dimension.code)" />
              <span class="min-w-0 flex-1"><b>{{ dimension.display_name }}</b><small>{{ dimension.code }}</small>
                <select
                  v-if="quotaDraft.dimension_codes.includes(dimension.code) && usesEnhancedSelector(dimension.code, quotaValues)"
                  v-model="quotaValues[dimension.code]"
                  class="input mt-2"
                  required
                  @change="onQuotaValueChange(dimension.code)"
                >
                  <option value="" disabled>请选择{{ dimension.display_name }}</option>
                  <option v-for="option in enhancedDimensionOptions(dimension.code, quotaValues, quotaAccountModels)" :key="String(option.value)" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
                <input v-else-if="quotaDraft.dimension_codes.includes(dimension.code)" v-model="quotaValues[dimension.code]" class="input mt-2" required :type="dimension.value_type === 'int64' ? 'number' : 'text'" placeholder="匹配值" />
              </span>
            </label>
          </div>
          <p class="text-xs text-amber-700">若维度组合尚无统计项，系统会创建草稿，限额保持等待；统计项启用后限额自动启用，并立即使用当前周期已经累计的 Token 用量进行判断。</p>
          <div class="flex justify-end gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
            <button class="btn btn-primary" :disabled="saving">{{ saving ? '创建中…' : '创建限额' }}</button>
          </div>
        </form>
        <div class="card overflow-x-auto">
          <table class="stat-table">
            <thead><tr><th>名称</th><th>适用范围</th><th>周期</th><th>上限</th><th>模式</th><th>状态</th><th>生效时间</th><th>操作</th></tr></thead>
            <tbody><tr v-for="item in quotas" :key="item.id">
              <td class="font-medium">{{ item.name }}</td>
              <td><div class="flex min-w-48 flex-wrap gap-1.5"><span v-for="entry in quotaDimensionEntries(item)" :key="entry.code" class="rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-700"><b>{{ entry.name }}：</b>{{ entry.value }}</span></div></td>
              <td>{{ periodLabel(item.period_type) }}</td><td class="font-mono">{{ item.limit_value.toLocaleString() }}</td><td>{{ item.enforcement_mode === 'ENFORCE' ? '强制限制' : '仅观察' }}</td><td><span class="status-pill" :class="quotaStatusClass(item.status)"><span class="status-dot"></span>{{ quotaStatusLabel(item.status) }}</span></td><td>{{ formatTime(item.effective_from) }}</td>
              <td><div v-if="can('token_quota.update')" class="flex flex-wrap gap-2">
                <button class="link" @click="editQuota(item)">编辑</button>
                <button class="link" @click="quotaAction(item.id, item.status === 'ENABLED' ? 'disable' : 'enable')">{{ item.status === 'ENABLED' ? '停用' : '启用' }}</button>
                <button class="link text-red-600" @click="deletingQuota = item">删除</button>
              </div></td>
            </tr><tr v-if="quotas.length === 0"><td colspan="8"><div class="empty-state"><Icon name="shield" size="lg" /><b>还没有限额规则</b><span>统计项可以独立运行，不创建限额也会持续记录。</span></div></td></tr></tbody>
          </table>
        </div>
      </section>

      <section v-else-if="tab === 'query'" class="space-y-4">
        <form class="card space-y-5 p-5" @submit.prevent="runQuery(1)">
          <div><h2 class="section-title">多维统计查询</h2><p class="section-help">选择统计项后，筛选条件和分组维度会按其配置动态生成。</p></div>
          <div class="grid gap-3 md:grid-cols-4">
            <select v-model.number="queryDraft.projection_id" class="input" required @change="resetQueryDimensions">
              <option :value="0" disabled>选择已采集统计项</option>
              <option v-for="item in queryableProjections" :key="item.id" :value="item.id">{{ item.name }}（{{ item.dimension_codes.join(' + ') }}）</option>
            </select>
            <select v-model="queryDraft.period_type" class="input" @change="alignNaturalQueryRange"><option value="D">日</option><option value="W">自然周</option><option value="M">自然月</option></select>
            <input v-model="queryDraft.start" class="input" type="date" required @change="alignNaturalQueryRange" />
            <input v-model="queryDraft.end" class="input" type="date" required @change="alignNaturalQueryRange" />
          </div>
          <p v-if="queryDraft.period_type !== 'D'" class="text-xs text-gray-500">自然周和自然月查询会自动将起止日期对齐到完整的自然周期边界。</p>
          <div v-if="selectedQueryProjection" class="grid gap-4 md:grid-cols-2">
            <fieldset><legend class="mb-2 text-sm font-medium">筛选条件</legend>
              <label v-for="dimension in queryDimensions" :key="dimension.code" class="mb-2 flex items-center gap-2 text-sm">
                <span class="w-28">{{ dimension.display_name }}</span>
                <select
                  v-if="usesEnhancedSelector(dimension.code)"
                  v-model="queryFilterValues[dimension.code]"
                  class="input flex-1"
                  @change="onQueryFilterChange(dimension.code)"
                >
                  <option value="">不限</option>
                  <option v-for="option in enhancedDimensionOptions(dimension.code, queryFilterValues, accountModels)" :key="String(option.value)" :value="option.value">
                    {{ option.label }}
                  </option>
                </select>
                <input v-else v-model="queryFilterValues[dimension.code]" class="input flex-1" :type="dimension.value_type === 'int64' ? 'number' : 'text'" placeholder="留空表示不限" />
              </label>
            </fieldset>
            <fieldset><legend class="mb-2 text-sm font-medium">分组维度</legend>
              <label v-for="dimension in queryDimensions" :key="dimension.code" class="mr-4 text-sm"><input v-model="queryDraft.group_by" type="checkbox" :value="dimension.code" /> {{ dimension.display_name }}</label>
            </fieldset>
          </div>
          <div class="flex gap-2"><button class="btn btn-primary" :disabled="queryLoading">查询</button><button type="button" class="btn btn-secondary" :disabled="!queryResult" @click="exportQuery">导出 CSV</button></div>
        </form>

        <div v-if="queryError" class="rounded-lg bg-red-50 p-3 text-sm text-red-700">{{ queryError }}</div>
        <div v-if="queryResult" class="space-y-4">
          <div class="grid gap-3 md:grid-cols-3">
            <div class="result-card"><div>Token 汇总</div><strong>{{ queryResult.summary.toLocaleString() }}</strong></div>
            <div class="result-card"><div>统计开始时间</div><strong class="text-base">{{ formatTime(queryResult.projection_enabled_at) }}</strong></div>
            <div class="result-card"><div>MySQL 最后同步</div><strong class="text-base">{{ formatTime(queryResult.last_synced_at) }}</strong></div>
          </div>
          <div v-if="!queryResult.complete" class="rounded-lg bg-amber-50 p-3 text-sm text-amber-800">数据仍在最终一致同步中，当前结果可能不完整。</div>
          <div v-if="queryResult.rows.length" class="grid gap-4 lg:grid-cols-2">
            <div class="card p-4"><h2 class="mb-3 font-medium">趋势</h2>
              <div v-for="row in trendRows" :key="row.period_start + dimensionLabel(row.dimensions)" class="mb-2 flex items-center gap-2 text-xs">
                <span class="w-36 truncate">{{ new Date(row.period_start).toLocaleDateString() }}</span>
                <div class="h-3 rounded bg-primary-500" :style="{ width: `${Math.max(2, row.value / maxQueryValue * 100)}%` }"></div><span>{{ row.value }}</span>
              </div>
            </div>
            <div class="card p-4"><h2 class="mb-3 font-medium">排行榜</h2>
              <ol><li v-for="(row, index) in rankingRows" :key="index" class="flex justify-between border-b py-2 text-sm"><span>{{ index + 1 }}. {{ dimensionLabel(row.dimensions) || new Date(row.period_start).toLocaleDateString() }}</span><b>{{ row.value.toLocaleString() }}</b></li></ol>
            </div>
          </div>
          <div v-else class="card p-8 text-center text-gray-500">该条件下暂无已同步数据。</div>
          <div class="card overflow-x-auto">
            <table class="stat-table"><thead><tr><th>周期开始</th><th>周期结束</th><th>维度</th><th>Token</th></tr></thead>
              <tbody><tr v-for="(row, index) in queryResult.rows" :key="index"><td>{{ formatTime(row.period_start) }}</td><td>{{ formatTime(row.period_end) }}</td><td>{{ dimensionLabel(row.dimensions) }}</td><td>{{ row.value.toLocaleString() }}</td></tr></tbody>
            </table>
            <div class="flex justify-end gap-2 p-3"><button class="btn btn-secondary" :disabled="queryPage <= 1" @click="runQuery(queryPage - 1)">上一页</button><span class="py-2 text-sm">第 {{ queryPage }} 页 / 共 {{ queryResult.total }} 条</span><button class="btn btn-secondary" :disabled="queryPage * 50 >= queryResult.total" @click="runQuery(queryPage + 1)">下一页</button></div>
          </div>
        </div>
      </section>

      <section v-else class="card overflow-hidden">
        <div class="table-heading"><div><h2 class="section-title">同步与封账状态</h2><p class="section-help">MySQL 最后同步：{{ formatTime(syncStatus?.last_synced_at) }}</p></div><button class="btn btn-secondary" :disabled="loading" @click="load"><Icon name="refresh" size="sm" />刷新</button></div>
        <div class="grid gap-3 border-y border-gray-100 bg-gray-50/60 p-4 sm:grid-cols-2 lg:grid-cols-4 dark:border-dark-700 dark:bg-dark-900/30"><div v-for="(value, key) in syncStatus?.metrics ?? {}" :key="key" class="rounded-xl border border-gray-200 bg-white p-3 text-xs dark:border-dark-700 dark:bg-dark-800"><div class="truncate text-gray-500">{{ key }}</div><b class="mt-1 block text-lg text-gray-900 dark:text-white">{{ value }}</b></div></div>
        <div class="overflow-x-auto"><table class="stat-table"><thead><tr><th>周期</th><th>开始</th><th>结束</th><th>状态</th><th>错误</th></tr></thead>
          <tbody><tr v-for="item in syncStatus?.periods ?? []" :key="item.id"><td>{{ item.period_type }}</td><td>{{ formatTime(item.period_start) }}</td><td>{{ formatTime(item.period_end) }}</td><td>{{ item.state }}</td><td>{{ item.last_error || '-' }}</td></tr></tbody>
        </table></div>
      </section>
    </div>
    <BaseDialog :show="Boolean(editingQuota)" title="编辑限额规则" width="normal" @close="closeQuotaEditor">
      <form class="space-y-5" @submit.prevent="saveQuotaEdit">
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="sm:col-span-2"><span class="field-label">限额名称</span><input v-model.trim="quotaEditDraft.name" class="input" required /></label>
          <label><span class="field-label">Token 上限</span><input v-model.number="quotaEditDraft.limit_value" class="input" type="number" min="1" required /></label>
          <label><span class="field-label">执行模式</span><select v-model="quotaEditDraft.mode" class="input"><option value="OBSERVE">仅观察</option><option value="ENFORCE">强制限制</option></select></label>
        </div>
        <div class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/40">
          <div class="mb-2 flex items-center justify-between"><span class="text-sm font-medium text-gray-800 dark:text-gray-100">适用范围</span><span class="text-xs text-gray-500">{{ editingQuota ? periodLabel(editingQuota.period_type) : '' }}</span></div>
          <div class="flex flex-wrap gap-2"><span v-for="entry in editingQuota ? quotaDimensionEntries(editingQuota) : []" :key="entry.code" class="dimension-chip"><b>{{ entry.name }}：</b>{{ entry.value }}</span></div>
          <p class="mt-3 text-xs text-gray-500">周期和维度范围属于统计口径，编辑时保持不变；需要调整时请新建限额。</p>
        </div>
        <div class="flex justify-end gap-2 border-t border-gray-100 pt-4 dark:border-dark-700">
          <button type="button" class="btn btn-secondary" :disabled="editSaving" @click="closeQuotaEditor">取消</button>
          <button class="btn btn-primary" :disabled="editSaving">{{ editSaving ? '保存中…' : '保存修改' }}</button>
        </div>
      </form>
    </BaseDialog>
    <ConfirmDialog
      :show="Boolean(deletingQuota)"
      title="删除限额规则"
      :message="`确定删除限额“${deletingQuota?.name ?? ''}”吗？已统计的用量和历史数据不会被删除。`"
      confirm-text="删除"
      cancel-text="取消"
      danger
      @confirm="confirmDeleteQuota"
      @cancel="deletingQuota = undefined"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import { dynamicTokenStatisticsAPI } from '@/api/admin/dynamicTokenStatistics'
import type { DimensionCode, DimensionDefinition, DimensionValue, PeriodType, Projection, ProjectionInput, Quota, RuntimeState, SyncStatus, UsageQueryInput, UsageQueryResult } from '@/api/admin/dynamicTokenStatistics'
import * as usersAPI from '@/api/admin/users'
import * as groupsAPI from '@/api/admin/groups'
import * as accountsAPI from '@/api/admin/accounts'
import { usePermission } from '@/composables/usePermission'
import type { Account, AdminGroup, AdminUser, SelectOption } from '@/types'

const { can } = usePermission()
const tabs = [
  { id: 'projections', label: '统计项', icon: 'chartBar' },
  { id: 'quotas', label: '限额', icon: 'shield' },
  { id: 'query', label: '通用查询', icon: 'search' },
  { id: 'status', label: '同步状态', icon: 'sync' }
] as const
const tab = ref<(typeof tabs)[number]['id']>('projections')
const dimensions = ref<DimensionDefinition[]>([])
const projections = ref<Projection[]>([])
const quotas = ref<Quota[]>([])
const syncStatus = ref<SyncStatus>()
const runtimeState = ref<RuntimeState>({ available: false, enabled: false })
const runtimeSaving = ref(false)
const saving = ref(false)
const loading = ref(false)
const error = ref('')
const success = ref('')
const projectionDraft = reactive<{ id?: number; name: string; dimension_codes: DimensionCode[] }>({ name: '', dimension_codes: [] })
const quotaDraft = reactive<{ name: string; dimension_codes: DimensionCode[]; period_type: PeriodType; mode: 'OBSERVE' | 'ENFORCE'; limit_value: number }>({ name: '', dimension_codes: [], period_type: 'D', mode: 'OBSERVE', limit_value: 0 })
const quotaValues = reactive<Partial<Record<DimensionCode, string | number>>>({})
const deletingQuota = ref<Quota>()
const editingQuota = ref<Quota>()
const quotaEditDraft = reactive<{ name: string; mode: 'OBSERVE' | 'ENFORCE'; limit_value: number }>({ name: '', mode: 'OBSERVE', limit_value: 0 })
const editSaving = ref(false)
const today = new Date()
const queryDraft = reactive({
  projection_id: Number(localStorage.getItem('token-stat-query-projection') || 0),
  period_type: 'D' as PeriodType,
  start: new Date(today.getTime() - 6 * 86400000).toISOString().slice(0, 10),
  end: new Date(today.getTime() + 86400000).toISOString().slice(0, 10),
  group_by: [] as DimensionCode[]
})
const queryFilterValues = reactive<Partial<Record<DimensionCode, string | number>>>({})
const queryResult = ref<UsageQueryResult>()
const queryLoading = ref(false)
const queryError = ref('')
const queryPage = ref(1)
const queryUsers = ref<AdminUser[]>([])
const queryGroups = ref<AdminGroup[]>([])
const queryAccounts = ref<Account[]>([])
const accountModels = ref<SelectOption[]>([])
const quotaAccountModels = ref<SelectOption[]>([])
const enhancedDimensionCodes = new Set<DimensionCode>(['user_id', 'group_id', 'account_id', 'route_alias', 'upstream_model'])
const queryableProjections = computed(() => projections.value.filter(item => item.status === 'ACTIVE' || item.status === 'DISABLED'))
const selectedQueryProjection = computed(() => projections.value.find(item => item.id === queryDraft.projection_id))
const queryDimensions = computed(() => dimensions.value.filter(item => selectedQueryProjection.value?.dimension_codes.includes(item.code)))
const rankingRows = computed(() => [...(queryResult.value?.rows ?? [])].sort((a, b) => b.value - a.value).slice(0, 10))
const trendRows = computed(() => [...(queryResult.value?.rows ?? [])].sort((a, b) => a.period_start.localeCompare(b.period_start)))
const maxQueryValue = computed(() => Math.max(1, ...trendRows.value.map(row => row.value)))
const activeProjectionCount = computed(() => projections.value.filter(item => item.status === 'ACTIVE').length)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [dimensionItems, projectionItems, quotaItems, status, runtime] = await Promise.all([
      dynamicTokenStatisticsAPI.dimensions(), dynamicTokenStatisticsAPI.projections(),
      dynamicTokenStatisticsAPI.quotas(), dynamicTokenStatisticsAPI.status(),
      dynamicTokenStatisticsAPI.runtime()
    ])
    dimensions.value = dimensionItems
    projections.value = projectionItems
    quotas.value = quotaItems
    syncStatus.value = status
    runtimeState.value = runtime
    void loadDimensionOptions()
  } catch (cause) {
    error.value = (cause as Error).message
  } finally {
    loading.value = false
  }
}
async function loadDimensionOptions() {
  const [users, groups, accounts] = await Promise.allSettled([
    usersAPI.list(1, 1000),
    groupsAPI.getAllIncludingInactive(),
    accountsAPI.list(1, 1000, { lite: 'true' })
  ])
  if (users.status === 'fulfilled') queryUsers.value = users.value.items
  if (groups.status === 'fulfilled') queryGroups.value = groups.value
  if (accounts.status === 'fulfilled') queryAccounts.value = accounts.value.items
}
function enhancedDimensionOptions(
  code: DimensionCode,
  values: Partial<Record<DimensionCode, string | number>>,
  upstreamModels: SelectOption[]
): SelectOption[] {
  if (code === 'user_id') {
    return queryUsers.value.map(user => ({ value: user.id, label: `${user.username || user.email}（#${user.id}）` }))
  }
  if (code === 'group_id') {
    return queryGroups.value.map(group => ({ value: group.id, label: `${group.name}（#${group.id}）` }))
  }
  if (code === 'account_id') {
    return queryAccounts.value.map(account => ({ value: account.id, label: `${account.name}（#${account.id}）` }))
  }
  if (code === 'route_alias') {
    const selectedGroupID = Number(values.group_id || 0)
    const groups = selectedGroupID
      ? queryGroups.value.filter(group => group.id === selectedGroupID)
      : queryGroups.value
    const aliases = new Set(groups.flatMap(group => Object.keys(group.model_routing || {})))
    return [...aliases].sort().map(alias => ({ value: alias, label: alias }))
  }
  if (code === 'upstream_model') return upstreamModels
  return []
}
function usesEnhancedSelector(
  code: DimensionCode,
  values: Partial<Record<DimensionCode, string | number>> = queryFilterValues
) {
  if (!enhancedDimensionCodes.has(code)) return false
  // 上游模型依赖模型账号动态加载；没有账号维度或尚未选择账号时保留通用文本查询能力。
  if (code === 'upstream_model') return Number(values.account_id || 0) > 0
  return true
}
async function loadAccountModels(accountID: number) {
  if (!accountID) return []
  const models = await accountsAPI.getAvailableModels(accountID, {})
  return models.map(model => ({ value: model.id, label: model.display_name || model.id }))
}
async function onQueryFilterChange(code: DimensionCode) {
  if (code === 'group_id') queryFilterValues.route_alias = ''
  if (code === 'account_id') {
    queryFilterValues.upstream_model = ''
    accountModels.value = []
    const accountID = Number(queryFilterValues.account_id || 0)
    if (!accountID) return
    try {
      accountModels.value = await loadAccountModels(accountID)
    } catch {
      accountModels.value = []
    }
  }
}
function onQuotaDimensionToggle(code: DimensionCode) {
  if (quotaDraft.dimension_codes.includes(code)) {
    quotaValues[code] = ''
    return
  }
  delete quotaValues[code]
  if (code === 'group_id') delete quotaValues.route_alias
  if (code === 'account_id') {
    delete quotaValues.upstream_model
    quotaAccountModels.value = []
  }
}
async function onQuotaValueChange(code: DimensionCode) {
  if (code === 'group_id') quotaValues.route_alias = ''
  if (code === 'account_id') {
    quotaValues.upstream_model = ''
    quotaAccountModels.value = []
    try {
      quotaAccountModels.value = await loadAccountModels(Number(quotaValues.account_id || 0))
    } catch {
      quotaAccountModels.value = []
    }
  }
}
async function toggleRuntime() {
  runtimeSaving.value = true
  error.value = ''
  success.value = ''
  try {
    const enabled = !runtimeState.value.enabled
    runtimeState.value = await dynamicTokenStatisticsAPI.updateRuntime(enabled)
    success.value = enabled
      ? '统计与限额机制已重新开启'
      : '统计与限额机制已热关闭；已有 Redis 数据仍会继续同步到 MySQL'
  } catch (cause) {
    error.value = (cause as Error).message
  } finally {
    runtimeSaving.value = false
  }
}
function resetProjection() { projectionDraft.id = undefined; projectionDraft.name = ''; projectionDraft.dimension_codes = [] }
function editProjection(item: Projection) { projectionDraft.id = item.id; projectionDraft.name = item.name; projectionDraft.dimension_codes = [...item.dimension_codes] }
async function saveProjection() {
  saving.value = true
  try {
    const input: ProjectionInput = { name: projectionDraft.name, dimension_codes: projectionDraft.dimension_codes, metric_codes: ['total_tokens'] }
    if (projectionDraft.id) await dynamicTokenStatisticsAPI.updateProjection(projectionDraft.id, input)
    else await dynamicTokenStatisticsAPI.createProjection(input)
    success.value = projectionDraft.id ? '统计项草稿已更新' : '统计项草稿已创建'
    resetProjection(); await load()
  } catch (cause) { error.value = (cause as Error).message } finally { saving.value = false }
}
async function projectionAction(id: number, action: 'publish' | 'activate' | 'disable') {
  error.value = ''; success.value = ''
  try {
    await dynamicTokenStatisticsAPI.projectionAction(id, action)
    success.value = action === 'publish' ? '统计项已发布' : action === 'activate' ? '统计项已启用，新的模型调用将开始计入' : '统计项已停用'
    await load()
  } catch (cause) { error.value = (cause as Error).message }
}
function resetQuota() {
  quotaDraft.name = ''
  quotaDraft.dimension_codes = []
  quotaDraft.period_type = 'D'
  quotaDraft.mode = 'OBSERVE'
  quotaDraft.limit_value = 0
  Object.keys(quotaValues).forEach(key => delete quotaValues[key as DimensionCode])
  quotaAccountModels.value = []
}
function editQuota(item: Quota) {
  editingQuota.value = item
  quotaEditDraft.name = item.name
  quotaEditDraft.mode = item.enforcement_mode
  quotaEditDraft.limit_value = item.limit_value
}
function closeQuotaEditor() {
  if (editSaving.value) return
  editingQuota.value = undefined
}
async function createQuota() {
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    const values = Object.fromEntries(quotaDraft.dimension_codes.map(code => {
      const definition = dimensions.value.find(item => item.code === code)!
      return [code, definition.value_type === 'int64' ? { type: 'int64', int64: Number(quotaValues[code]) } : { type: 'string', string: String(quotaValues[code] ?? '') }]
    }))
    await dynamicTokenStatisticsAPI.createQuota({ ...quotaDraft, dimension_values: values, metric_code: 'total_tokens' })
    success.value = '限额规则已创建'
    resetQuota()
    await load()
  } catch (cause) { error.value = (cause as Error).message } finally { saving.value = false }
}
async function saveQuotaEdit() {
  const item = editingQuota.value
  if (!item) return
  editSaving.value = true
  error.value = ''
  success.value = ''
  try {
    await dynamicTokenStatisticsAPI.updateQuota(item.id, {
      name: quotaEditDraft.name,
      limit_value: quotaEditDraft.limit_value,
      mode: quotaEditDraft.mode
    })
    editingQuota.value = undefined
    success.value = '限额规则已更新并热生效'
    await load()
  } catch (cause) {
    error.value = (cause as Error).message
  } finally {
    editSaving.value = false
  }
}
async function quotaAction(id: number, action: 'enable' | 'disable') {
  try {
    await dynamicTokenStatisticsAPI.quotaAction(id, action)
    success.value = action === 'enable' ? '限额已启用' : '限额已停用'
    await load()
  } catch (cause) { error.value = (cause as Error).message }
}
async function confirmDeleteQuota() {
  const item = deletingQuota.value
  if (!item) return
  error.value = ''
  success.value = ''
  try {
    await dynamicTokenStatisticsAPI.deleteQuota(item.id)
    if (editingQuota.value?.id === item.id) editingQuota.value = undefined
    deletingQuota.value = undefined
    success.value = '限额规则已删除；统计用量和历史数据保持不变'
    await load()
  } catch (cause) {
    error.value = (cause as Error).message
  }
}
function resetQueryDimensions() { queryDraft.group_by = []; Object.keys(queryFilterValues).forEach(key => delete queryFilterValues[key as DimensionCode]) }
function buildQuery(page = 1): UsageQueryInput {
  alignNaturalQueryRange()
  const filters: Partial<Record<DimensionCode, DimensionValue>> = {}
  for (const definition of queryDimensions.value) {
    const raw = queryFilterValues[definition.code]
    if (raw === '' || raw === undefined) continue
    filters[definition.code] = definition.value_type === 'int64' ? { type: 'int64', int64: Number(raw) } : { type: 'string', string: String(raw) }
  }
  return {
    projection_id: queryDraft.projection_id, metric_code: 'total_tokens', period_type: queryDraft.period_type,
    start: `${queryDraft.start}T00:00:00+08:00`, end: `${queryDraft.end}T00:00:00+08:00`,
    filters, group_by: queryDraft.group_by, sort: 'time_asc', page, page_size: 50
  }
}
function localDate(value: string) {
  const [year, month, day] = value.split('-').map(Number)
  return new Date(year, month - 1, day)
}
function dateValue(value: Date) {
  const year = value.getFullYear()
  const month = String(value.getMonth() + 1).padStart(2, '0')
  const day = String(value.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}
function alignNaturalQueryRange() {
  if (queryDraft.period_type === 'D') return
  const start = localDate(queryDraft.start)
  const end = localDate(queryDraft.end)
  if (queryDraft.period_type === 'W') {
    const startWeekday = start.getDay() || 7
    start.setDate(start.getDate() - startWeekday + 1)
    const endWeekday = end.getDay() || 7
    if (endWeekday !== 1) end.setDate(end.getDate() + (8 - endWeekday))
  } else {
    start.setDate(1)
    if (end.getDate() !== 1) end.setMonth(end.getMonth() + 1, 1)
  }
  queryDraft.start = dateValue(start)
  queryDraft.end = dateValue(end)
}
async function runQuery(page = 1) {
  queryLoading.value = true; queryError.value = ''
  try {
    queryResult.value = await dynamicTokenStatisticsAPI.query(buildQuery(page))
    queryPage.value = page
    localStorage.setItem('token-stat-query-projection', String(queryDraft.projection_id))
  } catch (cause) { queryError.value = (cause as Error).message } finally { queryLoading.value = false }
}
async function exportQuery() {
  try {
    const blob = await dynamicTokenStatisticsAPI.exportCSV(buildQuery(1))
    const url = URL.createObjectURL(blob); const link = document.createElement('a')
    link.href = url; link.download = 'token-statistics.csv'; link.click(); URL.revokeObjectURL(url)
  } catch (cause) { queryError.value = (cause as Error).message }
}
function dimensionLabel(values: Partial<Record<DimensionCode, DimensionValue>>) {
  return Object.entries(values).map(([code, value]) => `${dimensionName(code as DimensionCode)}=${value?.type === 'int64' ? value.int64 : value?.string}`).join(', ')
}
function dimensionName(code: DimensionCode) { return dimensions.value.find(item => item.code === code)?.display_name ?? code }
function readableDimensionValue(code: DimensionCode, value: number | string) {
  const id = Number(value)
  if (code === 'user_id') {
    const user = queryUsers.value.find(item => item.id === id)
    return user ? `${user.username || user.email}（${value}）` : String(value)
  }
  if (code === 'group_id') {
    const group = queryGroups.value.find(item => item.id === id)
    return group ? `${group.name}（${value}）` : String(value)
  }
  if (code === 'account_id') {
    const account = queryAccounts.value.find(item => item.id === id)
    return account ? `${account.name}（${value}）` : String(value)
  }
  return String(value)
}
function quotaDimensionEntries(item: Quota) {
  const projection = projections.value.find(candidate => candidate.id === item.projection_id)
  const orderedCodes = projection?.dimension_codes ?? Object.keys(item.dimension_values) as DimensionCode[]
  return orderedCodes
    .filter(code => item.dimension_values[code] !== undefined)
    .map(code => ({ code, name: dimensionName(code), value: readableDimensionValue(code, item.dimension_values[code]) }))
}
function projectionStatusLabel(status: Projection['status']) {
  return ({ DRAFT: '草稿', PUBLISHED: '待启用', ACTIVE: '运行中', DISABLED: '已停用' } as const)[status] ?? status
}
function projectionStatusClass(status: Projection['status']) {
  return ({ DRAFT: 'status-neutral', PUBLISHED: 'status-info', ACTIVE: 'status-success', DISABLED: 'status-disabled' } as const)[status] ?? 'status-neutral'
}
function quotaStatusLabel(status: Quota['status']) {
  return ({ PENDING: '等待统计项', ENABLED: '已启用', DISABLED: '已停用' } as const)[status] ?? status
}
function quotaStatusClass(status: Quota['status']) {
  return ({ PENDING: 'status-warning', ENABLED: 'status-success', DISABLED: 'status-disabled' } as const)[status] ?? 'status-neutral'
}
function periodLabel(period: PeriodType) { return ({ D: '自然日', W: '自然周', M: '自然月' } as const)[period] }
function formatTime(value?: string) { return value ? new Date(value).toLocaleString() : '-' }
onMounted(load)
</script>

<style scoped>
.metric-tile { @apply rounded-xl border border-white/70 bg-white/75 px-3 py-3 text-center shadow-sm backdrop-blur dark:border-dark-600 dark:bg-dark-800/70; }
.metric-tile span { @apply block text-xs text-gray-500 dark:text-gray-400; }
.metric-tile strong { @apply mt-1 block text-xl font-semibold text-gray-900 dark:text-white; }
.tab-button { @apply flex shrink-0 items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium text-gray-500 transition hover:bg-gray-50 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-dark-700 dark:hover:text-white; }
.tab-button-active { @apply bg-primary-50 text-primary-700 shadow-sm dark:bg-primary-950/50 dark:text-primary-300; }
.section-title { @apply text-base font-semibold text-gray-900 dark:text-white; }
.section-help { @apply mt-1 text-sm leading-5 text-gray-500 dark:text-gray-400; }
.field-label { @apply mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300; }
.dimension-option { @apply flex cursor-pointer items-start gap-3 rounded-xl border border-gray-200 bg-white p-3 transition hover:border-primary-300 hover:bg-primary-50/30 dark:border-dark-600 dark:bg-dark-800 dark:hover:border-primary-700; }
.dimension-option-active { @apply border-primary-400 bg-primary-50/60 ring-1 ring-primary-100 dark:border-primary-700 dark:bg-primary-950/30 dark:ring-primary-900; }
.dimension-option b { @apply block text-sm font-medium text-gray-800 dark:text-gray-100; }
.dimension-option small { @apply mt-0.5 block font-mono text-xs text-gray-400; }
.table-heading { @apply flex items-center justify-between gap-4 border-b border-gray-100 p-5 dark:border-dark-700; }
.stat-table { @apply w-full min-w-[720px] text-left text-sm; }
.stat-table th { @apply whitespace-nowrap bg-gray-50/80 px-5 py-3 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:bg-dark-900/40 dark:text-gray-400; }
.stat-table td { @apply border-t border-gray-100 px-5 py-4 text-gray-600 dark:border-dark-700 dark:text-gray-300; }
.stat-table tbody tr { @apply transition hover:bg-gray-50/60 dark:hover:bg-dark-700/30; }
.dimension-chip { @apply rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300; }
.status-pill { @apply inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-xs font-medium; }
.status-dot { @apply h-1.5 w-1.5 rounded-full bg-current; }
.status-neutral { @apply bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300; }
.status-info { @apply bg-blue-50 text-blue-700 dark:bg-blue-950/40 dark:text-blue-300; }
.status-success { @apply bg-emerald-50 text-emerald-700 dark:bg-emerald-950/40 dark:text-emerald-300; }
.status-warning { @apply bg-amber-50 text-amber-700 dark:bg-amber-950/40 dark:text-amber-300; }
.status-disabled { @apply bg-red-50 text-red-600 dark:bg-red-950/40 dark:text-red-300; }
.empty-state { @apply flex flex-col items-center gap-2 py-10 text-center text-gray-400; }
.empty-state b { @apply text-sm font-medium text-gray-600 dark:text-gray-300; }
.empty-state span { @apply max-w-md text-xs; }
.result-card { @apply rounded-xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-800; }
.result-card div { @apply text-xs text-gray-500 dark:text-gray-400; }
.result-card strong { @apply mt-2 block text-2xl font-semibold text-gray-900 dark:text-white; }
</style>
