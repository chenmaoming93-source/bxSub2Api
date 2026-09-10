<template>
  <AppLayout>
    <div class="space-y-6 p-4 sm:p-6">
      <header class="flex flex-col gap-4 rounded-2xl border border-indigo-100 bg-gradient-to-br from-indigo-50 via-white to-primary-50 p-6 shadow-sm dark:border-indigo-900/40 dark:from-indigo-950/40 dark:via-dark-800 dark:to-primary-950/30 lg:flex-row lg:items-center lg:justify-between">
        <div>
          <div class="mb-2 flex items-center gap-2 text-sm font-medium text-indigo-600 dark:text-indigo-400"><Icon name="shield" size="sm" />SingGuard Security</div>
          <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">安全检查日志</h1>
          <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">查看请求安全检查状态、决策结果和 SingGuard 返回详情。</p>
        </div>
        <div class="flex items-center gap-3 text-sm"><span class="status-dot" :class="status?.circuit_open ? 'bg-red-500' : 'bg-emerald-500'"></span><span>{{ status?.circuit_open ? '采集已熔断' : '采集正常' }}</span><button v-if="status?.circuit_open && canManage" type="button" class="btn btn-secondary" @click="reopen">恢复采集</button></div>
      </header>

      <section class="card space-y-4 p-5">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between"><div><h2 class="section-title">日志保留与清理</h2><p class="section-help">按服务器本地时间每天自动清理过期安全检查记录。</p></div><span v-if="retentionConfig.next_cleanup_at" class="text-xs text-gray-500">下次清理：{{ formatTime(retentionConfig.next_cleanup_at) }}</span></div>
        <div class="grid gap-4 md:grid-cols-[12rem_12rem_1fr_auto] md:items-end">
          <label><span class="field-label">保留天数</span><input v-model.number="retentionConfig.retention_days" class="input" type="number" min="1" max="3650" required :disabled="retentionLoading || retentionSaving || !canManage" /><span class="mt-1 block text-xs text-gray-500">范围：1～3650 天</span></label>
          <label><span class="field-label">每日清理时间</span><input v-model="retentionConfig.cleanup_time" class="input" type="time" required :disabled="retentionLoading || retentionSaving || !canManage" /></label>
          <div class="text-sm text-gray-500 dark:text-gray-400">时区：{{ retentionConfig.timezone || '服务器本地时区' }}</div>
          <button v-if="canManage" class="btn btn-primary" :disabled="retentionLoading || retentionSaving" @click="saveRetention">{{ retentionSaving ? '保存中…' : '保存清理配置' }}</button>
        </div>
        <div v-if="retentionError" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300">{{ retentionError }}</div>
        <div v-if="retentionSaved" class="rounded-lg bg-emerald-50 p-3 text-sm text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300">清理配置已保存。</div>
      </section>

      <section class="card space-y-4 p-4">
        <div class="grid gap-3 md:grid-cols-4">
          <label><span class="field-label">分组 ID</span><input v-model="filters.group_id" class="input" type="number" min="1" placeholder="全部分组" /></label>
          <label><span class="field-label">决策</span><select v-model="filters.decision" class="input"><option value="">全部决策</option><option value="allow">放行</option><option value="warn">告警</option><option value="block">阻断</option></select></label>
          <label><span class="field-label">检查状态</span><select v-model="filters.status" class="input"><option value="">全部状态</option><option value="success">检查成功</option><option value="timeout">检查超时</option><option value="error">检查异常</option><option value="skipped">未执行</option></select></label>
          <div class="flex items-end gap-2"><button class="btn btn-primary" :disabled="loading" @click="applyFilters">查询</button><button class="btn btn-secondary" :disabled="loading" @click="load">刷新</button></div>
        </div>
        <div v-if="error" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300">{{ error }}</div>
      </section>

      <section class="card overflow-hidden">
        <div class="table-heading"><div><h2 class="section-title">检查记录</h2><p class="section-help">共 {{ total }} 条记录</p></div><span v-if="loading" class="text-sm text-gray-500">加载中…</span></div>
        <div class="overflow-x-auto">
          <table class="stat-table">
            <thead><tr><th>时间</th><th>分组 / 模型</th><th>协议</th><th>检查状态</th><th>决策</th><th>请求体</th><th>操作</th></tr></thead>
            <tbody>
              <tr v-for="item in items" :key="item.id">
                <td class="whitespace-nowrap">{{ formatTime(item.created_at) }}</td>
                <td><div class="font-medium">{{ item.group_name || (item.group_id ? `#${item.group_id}` : '-') }}</div><div class="text-xs text-gray-500">{{ item.model || '-' }}</div></td>
                <td>{{ item.protocol || '-' }}</td>
                <td><span class="status-pill" :class="statusClass(item.check_status)"><span class="status-dot"></span>{{ statusLabel(item.check_status) }}</span></td>
                <td><span class="status-pill" :class="decisionClass(item.decision)">{{ decisionLabel(item.decision) }}</span></td>
                <td>{{ formatBytes(item.request_body_stored_bytes) }} / {{ formatBytes(item.request_body_original_bytes) }}<span v-if="item.request_body_truncated" class="ml-1 text-xs text-red-600">已截断</span></td>
                <td><button type="button" class="link" @click="showDetail(item.id)">查看详情</button></td>
              </tr>
              <tr v-if="!loading && items.length === 0"><td colspan="7"><div class="empty-state"><Icon name="shield" size="lg" /><b>暂无安全检查记录</b><span>调整筛选条件或等待新的安全检查记录。</span></div></td></tr>
            </tbody>
          </table>
        </div>
        <div class="flex items-center justify-between border-t border-gray-100 px-4 py-3 text-sm text-gray-500 dark:border-dark-700"><span>第 {{ page }} 页</span><div class="flex gap-2"><button class="btn btn-secondary" :disabled="page <= 1 || loading" @click="page--; load()">上一页</button><button class="btn btn-secondary" :disabled="page * pageSize >= total || loading" @click="page++; load()">下一页</button></div></div>
      </section>

      <BaseDialog :show="detailOpen" title="安全检查记录详情" width="wide" @close="closeDetail">
        <div v-if="detailLoading" class="flex items-center justify-center py-12 text-sm text-gray-500">加载详情中…</div>
        <div v-else-if="detailError" class="rounded-lg bg-red-50 p-4 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300">{{ detailError }}</div>
        <div v-else-if="detail" class="space-y-4">
          <div><h2 class="section-title">记录 #{{ detail.id }}</h2><p class="section-help">{{ formatTime(detail.created_at) }}</p></div>
          <div class="grid gap-3 text-sm md:grid-cols-4"><div>检查状态：<b>{{ statusLabel(detail.check_status) }}</b></div><div>最终决策：<b>{{ decisionLabel(detail.decision) }}</b></div><div>配置版本：<b>{{ detail.config_version }}</b></div><div>检查耗时：<b>{{ detail.latency_ms ?? '-' }} ms</b></div></div>
          <div v-if="detail.exception_type || detail.exception_message" class="rounded-lg border border-red-200 bg-red-50 p-4 dark:border-red-900/50 dark:bg-red-950/30">
            <h3 class="font-medium text-red-800 dark:text-red-200">异常信息</h3>
            <p v-if="detail.exception_type" class="mt-2 text-sm text-red-700 dark:text-red-300">异常类型：{{ detail.exception_type }}</p>
            <pre v-if="detail.exception_message" class="mt-2 max-h-40 overflow-auto whitespace-pre-wrap text-sm text-red-700 dark:text-red-300">{{ detail.exception_message }}</pre>
          </div>
          <div v-if="detail.request_body_truncated" class="rounded-lg bg-amber-50 p-3 text-sm text-amber-800">请求体已截断：保存 {{ formatBytes(detail.request_body_stored_bytes) }}，原始 {{ formatBytes(detail.request_body_original_bytes) }}。</div>
          <div v-if="detail.triggered_rules?.length" class="space-y-2"><h3 class="font-medium">命中规则</h3><div v-for="(rule, index) in detail.triggered_rules" :key="index" class="rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-700"><b>{{ dimensionName(rule.dimension) }}</b><span class="ml-2 text-gray-500">{{ dimensionCode(rule.dimension) }}</span><span class="ml-2">风险概率 {{ rule.risk_prob }}，阈值 {{ rule.threshold }}，动作 {{ rule.action === 'block' ? '阻断' : '告警' }}</span></div></div>
          <div class="grid gap-4 lg:grid-cols-2"><div><h3 class="mb-2 font-medium">请求体</h3><pre class="max-h-96 overflow-auto rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-800">{{ detail.request_body || '(empty)' }}</pre></div><div><h3 class="mb-2 font-medium">SingGuard 完整返回</h3><pre class="max-h-96 overflow-auto rounded-lg bg-gray-50 p-3 text-xs dark:bg-dark-800">{{ formatResponse(detail.singguard_response) }}</pre></div></div>
        </div>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref, reactive } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { SecurityCheckLogRetentionConfig, SecurityCheckLogSummary } from '@/api/admin/groups'
import { extractApiErrorMessage } from '@/utils/apiError'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const canManage = authStore.can('groups.update')
const items = ref<SecurityCheckLogSummary[]>([])
const detail = ref<any>(null)
const detailOpen = ref(false)
const detailLoading = ref(false)
const detailError = ref('')
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const error = ref('')
const status = ref<{ circuit_open: boolean; failure_count: number } | null>(null)
const retentionConfig = ref<SecurityCheckLogRetentionConfig>({ retention_days: 3, cleanup_time: '03:00', timezone: '服务器本地时区', next_cleanup_at: '' })
const retentionLoading = ref(false)
const retentionSaving = ref(false)
const retentionError = ref('')
const retentionSaved = ref(false)
const filters = reactive({ group_id: '', decision: '', status: '' })

const dimensions: Record<string, { name: string; code: string }> = {
  Dangerous_Operations_Tool_Abuse: { name: '危险操作与工具滥用', code: 'Dangerous_Operations_Tool_Abuse' },
  Malicious_Code_and_Cyberattack: { name: '恶意代码与网络攻击', code: 'Malicious_Code_and_Cyberattack' },
  Prompt_Injection_and_Jailbreak: { name: '提示词注入与越狱', code: 'Prompt_Injection_and_Jailbreak' },
  Resource_Abuse: { name: '资源滥用', code: 'Resource_Abuse' },
  Sensitive_Information_Stealing: { name: '敏感信息窃取', code: 'Sensitive_Information_Stealing' },
}
const statusLabel = (value: string) => ({ success: '检查成功', timeout: '检查超时', error: '检查异常', skipped: '未执行' }[value] || value || '-')
const statusClass = (value: string) => ({ success: 'status-pill-success', timeout: 'status-pill-warning', error: 'status-pill-danger', skipped: 'status-pill-muted' }[value] || 'status-pill-muted')
const decisionLabel = (value: string) => ({ allow: '放行', warn: '告警', block: '阻断' }[value] || value || '-')
const decisionClass = (value: string) => ({ allow: 'status-pill-success', warn: 'status-pill-warning', block: 'status-pill-danger' }[value] || 'status-pill-muted')
const dimensionName = (value: string) => dimensions[value]?.name || value
const dimensionCode = (value: string) => dimensions[value]?.code || value
const formatTime = (value: string) => new Date(value).toLocaleString()
const formatBytes = (value: number) => value < 1024 ? `${value} B` : value < 1024 * 1024 ? `${(value / 1024).toFixed(1)} KB` : `${(value / 1024 / 1024).toFixed(1)} MB`
const formatResponse = (value?: string) => { if (!value) return '(no response)'; try { return JSON.stringify(JSON.parse(value), null, 2) } catch { return value } }
const loadRetention = async () => {
  retentionLoading.value = true
  retentionError.value = ''
  try { retentionConfig.value = await adminAPI.groups.getSecurityCheckLogRetention() } catch (err) { retentionError.value = extractApiErrorMessage(err) } finally { retentionLoading.value = false }
}
const saveRetention = async () => {
  retentionSaving.value = true
  retentionError.value = ''
  retentionSaved.value = false
  try {
    retentionConfig.value = await adminAPI.groups.updateSecurityCheckLogRetention({ retention_days: retentionConfig.value.retention_days, cleanup_time: retentionConfig.value.cleanup_time })
    retentionSaved.value = true
  } catch (err) { retentionError.value = extractApiErrorMessage(err) } finally { retentionSaving.value = false }
}
const load = async () => {
  loading.value = true; error.value = ''
  try {
    const result = await adminAPI.groups.listSecurityCheckLogs({ page: page.value, page_size: pageSize, ...filters })
    items.value = result.items || []; total.value = result.total || 0
    status.value = await adminAPI.groups.getSecurityCheckCollectionStatus()
  } catch (err) { error.value = extractApiErrorMessage(err) } finally { loading.value = false }
}
const applyFilters = () => { page.value = 1; closeDetail(); load() }
const showDetail = async (id: number) => {
  detailOpen.value = true
  detail.value = null
  detailError.value = ''
  detailLoading.value = true
  try { detail.value = await adminAPI.groups.getSecurityCheckLog(id) } catch (err) { detailError.value = extractApiErrorMessage(err) } finally { detailLoading.value = false }
}
const closeDetail = () => {
  detailOpen.value = false
  detail.value = null
  detailError.value = ''
  detailLoading.value = false
}
const reopen = async () => { try { await adminAPI.groups.reopenSecurityCheckCollection(); await load() } catch (err) { error.value = extractApiErrorMessage(err) } }
onMounted(() => {
  void Promise.all([load(), loadRetention()])
})
</script>

<style scoped>
.section-title { @apply text-base font-semibold text-gray-900 dark:text-white; }
.section-help { @apply mt-1 text-sm leading-5 text-gray-500 dark:text-gray-400; }
.field-label { @apply mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300; }
.table-heading { @apply flex items-center justify-between gap-4 border-b border-gray-100 p-5 dark:border-dark-700; }
.stat-table { @apply w-full min-w-[900px] text-left text-sm; }
.stat-table th { @apply whitespace-nowrap bg-gray-50/80 px-5 py-3 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:bg-dark-900/40 dark:text-gray-400; }
.stat-table td { @apply border-t border-gray-100 px-5 py-4 text-gray-600 dark:border-dark-700 dark:text-gray-300; }
.stat-table tbody tr { @apply transition hover:bg-gray-50/60 dark:hover:bg-dark-700/30; }
.status-dot { @apply inline-block h-1.5 w-1.5 rounded-full bg-current; }
.empty-state { @apply flex flex-col items-center gap-2 py-10 text-center text-gray-400; }
.empty-state b { @apply text-sm font-medium text-gray-600 dark:text-gray-300; }
.empty-state span { @apply max-w-md text-xs; }
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  border-radius: 9999px;
  padding: 0.2rem 0.55rem;
  font-size: 0.75rem;
  font-weight: 600;
}
.status-pill-success { color: rgb(4 120 87); background: rgb(209 250 229); }
.status-pill-warning { color: rgb(180 83 9); background: rgb(254 243 199); }
.status-pill-danger { color: rgb(185 28 28); background: rgb(254 226 226); }
.status-pill-muted { color: rgb(75 85 99); background: rgb(243 244 246); }
.dark .status-pill-success { color: rgb(110 231 183); background: rgb(6 78 59 / 0.45); }
.dark .status-pill-warning { color: rgb(252 211 77); background: rgb(120 53 15 / 0.45); }
.dark .status-pill-danger { color: rgb(252 165 165); background: rgb(127 29 29 / 0.45); }
.dark .status-pill-muted { color: rgb(209 213 219); background: rgb(55 65 81 / 0.65); }
</style>
