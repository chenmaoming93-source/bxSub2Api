<template>
  <BaseDialog :show="show" title="安全检查记录" width="wide" @close="emit('close')">
    <div class="space-y-4">
      <div class="flex flex-wrap items-end gap-3">
        <label class="block"><span class="input-label">决策</span><select v-model="filters.decision" class="input"><option value="">全部</option><option value="allow">放行</option><option value="warn">告警</option><option value="block">阻断</option></select></label>
        <label class="block"><span class="input-label">状态</span><select v-model="filters.status" class="input"><option value="">全部</option><option value="success">成功</option><option value="timeout">超时</option><option value="error">异常</option><option value="skipped">跳过</option></select></label>
        <button type="button" class="btn btn-secondary" :disabled="loading" @click="load">{{ loading ? '加载中…' : '刷新' }}</button>
        <div class="ml-auto flex items-center gap-2 text-sm" :class="status?.circuit_open ? 'text-red-600' : 'text-green-600'">采集：{{ status?.circuit_open ? '熔断' : '正常' }}<button v-if="status?.circuit_open" type="button" class="btn btn-secondary" @click="reopen">恢复</button></div>
      </div>
      <div v-if="error" class="rounded-lg bg-red-50 p-3 text-sm text-red-700">{{ error }}</div>
      <div v-if="items.length === 0 && !loading" class="rounded-lg bg-gray-50 p-6 text-center text-sm text-gray-500 dark:bg-dark-700 dark:text-gray-400">暂无安全检查记录</div>
      <div v-else class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="min-w-full text-left text-sm"><thead class="bg-gray-50 text-xs uppercase text-gray-500 dark:bg-dark-700"><tr><th class="px-3 py-2">时间</th><th class="px-3 py-2">分组/模型</th><th class="px-3 py-2">协议</th><th class="px-3 py-2">检查状态</th><th class="px-3 py-2">决策</th><th class="px-3 py-2">请求体</th><th class="px-3 py-2"></th></tr></thead><tbody><tr v-for="item in items" :key="item.id" class="border-t border-gray-100 dark:border-dark-600"><td class="whitespace-nowrap px-3 py-2">{{ formatTime(item.created_at) }}</td><td class="px-3 py-2">{{ item.group_name || item.group_id || '-' }}<div class="text-xs text-gray-500">{{ item.model || '-' }}</div></td><td class="px-3 py-2">{{ item.protocol || '-' }}</td><td class="px-3 py-2"><span :class="statusClass(item.check_status)">{{ statusLabel(item.check_status) }}</span></td><td class="px-3 py-2"><span :class="item.decision === 'block' ? 'text-red-600' : item.decision === 'warn' ? 'text-amber-600' : 'text-green-600'">{{ item.decision }}</span></td><td class="px-3 py-2">{{ item.request_body_stored_bytes }}/{{ item.request_body_original_bytes }} bytes<span v-if="item.request_body_truncated" class="ml-1 text-red-600">已截断</span></td><td class="px-3 py-2"><button type="button" class="btn btn-secondary" @click="showDetail(item.id)">详情</button></td></tr></tbody></table>
      </div>
      <div class="flex items-center justify-between text-sm text-gray-500"><span>共 {{ total }} 条</span><div class="flex gap-2"><button class="btn btn-secondary" :disabled="page <= 1" @click="page--; load()">上一页</button><button class="btn btn-secondary" :disabled="page * pageSize >= total" @click="page++; load()">下一页</button></div></div>
      <div v-if="detail" class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"><div class="mb-2 flex items-center justify-between"><h3 class="font-medium">记录详情 #{{ detail.id }}</h3><button class="btn btn-secondary" @click="detail = null">关闭详情</button></div><div class="grid gap-2 text-sm md:grid-cols-3"><span>决策：{{ detail.decision }}</span><span>状态：{{ detail.check_status }}</span><span>配置版本：{{ detail.config_version }}</span><span>延迟：{{ detail.latency_ms ?? '-' }} ms</span><span>请求体：{{ detail.request_body_truncated ? '已截断' : '完整' }}</span></div><pre class="mt-3 max-h-64 overflow-auto rounded bg-gray-50 p-3 text-xs dark:bg-dark-800">{{ detail.request_body || '(empty)' }}</pre><pre class="mt-3 max-h-64 overflow-auto rounded bg-gray-50 p-3 text-xs dark:bg-dark-800">{{ detail.singguard_response || '(no response)' }}</pre></div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import type { SecurityCheckLogSummary } from '@/api/admin/groups'
import { extractApiErrorMessage } from '@/utils/apiError'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits<{ close: [] }>()
const items = ref<SecurityCheckLogSummary[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)
const error = ref('')
const detail = ref<any>(null)
const status = ref<{ circuit_open: boolean; failure_count: number } | null>(null)
const filters = ref({ decision: '', status: '' })
const formatTime = (value: string) => new Date(value).toLocaleString()
const statusLabel = (value: string) => ({ success: '检查成功', timeout: '检查超时', error: '检查异常', skipped: '未执行' }[value] || value || '-')
const statusClass = (value: string) => value === 'success' ? 'text-green-600' : value === 'timeout' ? 'text-amber-600' : value === 'error' ? 'text-red-600' : 'text-gray-500'
const load = async () => {
  loading.value = true; error.value = ''
  try {
    const result = await adminAPI.groups.listSecurityCheckLogs({ page: page.value, page_size: pageSize, ...filters.value })
    items.value = result.items || []; total.value = result.total || 0
    status.value = await adminAPI.groups.getSecurityCheckCollectionStatus()
  } catch (err) { error.value = extractApiErrorMessage(err) } finally { loading.value = false }
}
const showDetail = async (id: number) => { try { detail.value = await adminAPI.groups.getSecurityCheckLog(id) } catch (err) { error.value = extractApiErrorMessage(err) } }
const reopen = async () => { await adminAPI.groups.reopenSecurityCheckCollection(); await load() }
watch(() => props.show, (show) => { if (show) { page.value = 1; detail.value = null; load() } })
</script>
