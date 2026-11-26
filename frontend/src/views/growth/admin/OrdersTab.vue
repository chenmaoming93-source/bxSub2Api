<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { growthAdminAPI, type ExchangeOrder } from '@/api/growth'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import GrowthNotice from '@/components/growth/GrowthNotice.vue'
import GrowthStatusBadge from '@/components/growth/GrowthStatusBadge.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'
import { isOverdueFailed, orderMatches, productTitle, reasonIsValid } from '@/components/growth/growthRules'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { createRequestId } from '@/utils/requestId'

type ActionMode = 'retry' | 'refund' | 'complete' | ''

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const items = ref<ExchangeOrder[]>([])
const loading = ref(false)
const error = ref('')
const selected = ref<ExchangeOrder>()
const detailOpen = ref(false)
const actionOpen = ref(false)
const actionMode = ref<ActionMode>('')
const actionReason = ref('')
const actionConfirmed = ref(false)
const actionError = ref('')
const actionLoading = ref(false)
const searchInput = ref(String(route.query.search ?? ''))

const canManage = computed(() => authStore.can('growth.manage'))
const status = computed(() => String(route.query.status ?? ''))
const product = computed(() => String(route.query.product ?? ''))
const dateFrom = computed(() => String(route.query.from ?? ''))
const dateTo = computed(() => String(route.query.to ?? ''))
const products = computed(() => [...new Set(items.value.map(productTitle))].sort())
const filteredItems = computed(() => items.value.filter(order => orderMatches(order, {
  search: String(route.query.search ?? ''),
  status: status.value,
  product: product.value,
  from: dateFrom.value,
  to: dateTo.value,
})))
const overdueCount = computed(() => filteredItems.value.filter(order => isOverdueFailed(order)).length)
const actionTitle = computed(() => {
  if (actionMode.value === 'refund') return t('growth.admin.order.refundTitle')
  if (actionMode.value === 'complete') return t('growth.admin.order.completeTitle')
  return t('growth.admin.order.retry')
})
const actionDisabled = computed(() => {
  if (actionMode.value === 'refund' || actionMode.value === 'complete') return !reasonIsValid(actionReason.value)
  return false
})

function updateQuery(key: string, value: string) {
  const query = { ...route.query }
  if (value) query[key] = value
  else delete query[key]
  void router.replace({ query })
}

async function applySearch() {
  await router.replace({ query: { ...route.query, search: searchInput.value.trim() } })
  await load()
}

function showFailedOnly() {
  updateQuery('status', 'FAILED')
}

async function resetFilters() {
  searchInput.value = ''
  await router.replace({ query: {} })
  await load()
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = (await growthAdminAPI.exchanges({
      status: status.value,
      search: String(route.query.search ?? ''),
      page_size: 100,
    })).data.items
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    loading.value = false
  }
}

async function openDetail(order: ExchangeOrder) {
  selected.value = order
  detailOpen.value = true
  try {
    selected.value = (await growthAdminAPI.order(order.order_id)).data
  } catch (cause) {
    error.value = errorMessage(cause)
  }
}

function openAction(order: ExchangeOrder, mode: Exclude<ActionMode, ''>) {
  selected.value = order
  actionMode.value = mode
  actionReason.value = ''
  actionConfirmed.value = false
  actionError.value = ''
  actionOpen.value = true
}

async function submitAction() {
  if (!selected.value || actionDisabled.value || actionLoading.value) return
  if (actionMode.value !== 'retry' && !actionConfirmed.value) {
    actionConfirmed.value = true
    return
  }
  actionLoading.value = true
  actionError.value = ''
  try {
    let result
    if (actionMode.value === 'retry') {
      result = await growthAdminAPI.retryOrder(selected.value.order_id)
    } else if (actionMode.value === 'refund') {
      result = await growthAdminAPI.refundOrder(selected.value.order_id, {
        reason: actionReason.value.trim(),
        request_id: createRequestId(),
      })
    } else {
      result = await growthAdminAPI.completeOrder(selected.value.order_id, {
        reason: actionReason.value.trim(),
        request_id: createRequestId(),
      })
    }
    replaceOrder(result.data)
    actionOpen.value = false
    appStore.showToast('success', t('growth.common.operationSucceeded'))
  } catch (cause) {
    actionError.value = errorMessage(cause)
  } finally {
    actionLoading.value = false
  }
}

function replaceOrder(order: ExchangeOrder) {
  const index = items.value.findIndex(item => item.order_id === order.order_id)
  if (index >= 0) items.value.splice(index, 1, order)
  if (selected.value?.order_id === order.order_id) selected.value = order
}

async function copyOrderNo(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    appStore.showToast('success', t('growth.common.operationSucceeded'))
  } catch {
    appStore.showToast('error', t('growth.common.requestFailed'))
  }
}

function exportCsv() {
  const rows = [
    ['order_no', 'product', 'points', 'status', 'fulfillment', 'failure_reason', 'refund_transfer_id', 'created_at'],
    ...filteredItems.value.map(order => [
      order.order_no,
      productTitle(order),
      String(order.points_cost),
      order.status,
      order.fulfillment_status,
      order.failure_reason,
      String(order.refund_transfer_id ?? ''),
      order.created_at,
    ]),
  ]
  const csv = rows.map(row => row.map(value => `"${value.replace(/"/g, '""')}"`).join(',')).join('\n')
  const blob = new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.href = url
  link.download = `growth-orders-${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

function timeline(order: ExchangeOrder) {
  return [
    { key: 'created', label: t('growth.admin.order.createdStep'), done: true, time: order.created_at },
    { key: 'deducted', label: t('growth.admin.order.deductedStep'), done: Boolean(order.transfer_id), time: order.created_at },
    { key: 'fulfilling', label: t('growth.admin.order.fulfillingStep'), done: ['FULFILLING', 'SUCCEEDED', 'FAILED', 'REFUNDED'].includes(order.status), time: '' },
    { key: 'terminal', label: order.status === 'FAILED' ? t('growth.admin.order.failedStep') : order.status === 'REFUNDED' ? t('growth.admin.order.refundedStep') : t('growth.admin.order.fulfilledStep'), done: ['SUCCEEDED', 'FAILED', 'REFUNDED'].includes(order.status), time: order.fulfilled_at ?? '' },
  ]
}

function errorMessage(cause: unknown): string {
  if (cause && typeof cause === 'object') {
    const value = cause as { reason?: string; message?: string }
    return value.reason || value.message || t('growth.common.requestFailed')
  }
  return t('growth.common.requestFailed')
}

watch(
  () => route.query.status,
  () => void load()
)
onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <GrowthNotice v-if="error" tone="error">
      <div class="flex items-center justify-between gap-3">
        <span>{{ error }}</span>
        <button class="btn btn-sm btn-secondary" @click="load">{{ t('growth.common.retry') }}</button>
      </div>
    </GrowthNotice>
    <GrowthNotice v-if="overdueCount" tone="warning">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <span>{{ t('growth.admin.order.overdueCount', { count: overdueCount }) }}</span>
        <button class="font-semibold underline underline-offset-2" @click="showFailedOnly">{{ t('growth.admin.order.onlyFailed') }}</button>
      </div>
    </GrowthNotice>

    <div class="card p-4">
      <div class="grid gap-3 lg:grid-cols-12">
        <input v-model="searchInput" class="input lg:col-span-3" :placeholder="t('growth.admin.order.searchPlaceholder')" @keyup.enter="applySearch">
        <select :value="status" class="input lg:col-span-2" @change="updateQuery('status', ($event.target as HTMLSelectElement).value)">
          <option value="">{{ t('growth.admin.order.status') }}</option>
          <option value="CREATED">CREATED</option>
          <option value="FULFILLING">FULFILLING</option>
          <option value="SUCCEEDED">SUCCEEDED</option>
          <option value="FAILED">FAILED</option>
          <option value="REFUNDED">REFUNDED</option>
        </select>
        <select :value="product" class="input lg:col-span-2" @change="updateQuery('product', ($event.target as HTMLSelectElement).value)">
          <option value="">{{ t('growth.admin.order.product') }}</option>
          <option v-for="name in products" :key="name" :value="name">{{ name }}</option>
        </select>
        <input :value="dateFrom" type="date" class="input lg:col-span-2" :aria-label="t('growth.admin.order.dateFrom')" @change="updateQuery('from', ($event.target as HTMLInputElement).value)">
        <input :value="dateTo" type="date" class="input lg:col-span-2" :aria-label="t('growth.admin.order.dateTo')" @change="updateQuery('to', ($event.target as HTMLInputElement).value)">
        <div class="flex flex-wrap items-center gap-2 lg:col-span-12">
          <button class="btn btn-secondary" @click="applySearch">{{ t('growth.common.search') }}</button>
          <button class="btn btn-secondary" @click="resetFilters">{{ t('growth.common.reset') }}</button>
          <div class="flex-1" />
          <span class="text-xs text-gray-500 dark:text-dark-400">{{ filteredItems.length }}</span>
          <button class="btn btn-secondary" @click="exportCsv">{{ t('growth.admin.order.export') }}</button>
        </div>
      </div>
    </div>

    <div v-if="loading && !items.length" class="card p-12 text-center"><LoadingSpinner /></div>
    <div v-else-if="!filteredItems.length" class="card">
      <EmptyState :title="t('growth.admin.order.empty')" :description="t('growth.admin.order.emptyHint')" />
    </div>
    <div v-else class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-[1040px] w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-3">{{ t('growth.admin.order.orderNo') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.order.productColumn') }}</th>
              <th class="px-4 py-3 text-right">{{ t('growth.admin.order.pointsCost') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.order.status') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.order.fulfillment') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.order.createdAt') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.order.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="order in filteredItems" :key="order.order_id" class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-800/60" @click="openDetail(order)">
              <td class="px-4 py-3">
                <button class="font-mono text-xs text-primary-600 hover:underline dark:text-primary-400" @click.stop="copyOrderNo(order.order_no)">{{ order.order_no }}</button>
              </td>
              <td class="px-4 py-3">
                <div class="font-medium text-gray-900 dark:text-white">{{ productTitle(order) }}</div>
                <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">v{{ String(order.product_snapshot.version ?? '-') }}</div>
              </td>
              <td class="px-4 py-3 text-right"><PointsAmount :value="order.points_cost" direction="OUT" /></td>
              <td class="px-4 py-3"><GrowthStatusBadge :status="order.status" /></td>
              <td class="px-4 py-3"><GrowthStatusBadge :status="order.fulfillment_status" /></td>
              <td class="px-4 py-3 text-xs text-gray-500 dark:text-dark-400">{{ new Date(order.created_at).toLocaleString() }}</td>
              <td class="px-4 py-3" @click.stop>
                <div class="flex flex-wrap gap-1">
                  <button class="btn btn-sm btn-secondary" @click="openDetail(order)">{{ t('growth.admin.order.detail') }}</button>
                  <template v-if="canManage">
                    <button v-if="order.status === 'FAILED' && !order.refund_transfer_id" class="btn btn-sm btn-secondary" @click="openAction(order, 'retry')">{{ t('growth.admin.order.retry') }}</button>
                    <button v-if="order.fulfillment_status === 'PENDING'" class="btn btn-sm btn-secondary" @click="openAction(order, 'complete')">{{ t('growth.admin.order.complete') }}</button>
                    <button v-if="order.status === 'FAILED' && !order.refund_transfer_id" class="btn btn-sm btn-danger" @click="openAction(order, 'refund')">{{ t('growth.admin.order.refund') }}</button>
                  </template>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <BaseDialog :show="detailOpen" :title="t('growth.admin.order.detailTitle', { orderNo: selected?.order_no ?? '' })" width="wide" @close="detailOpen = false">
      <div v-if="selected" class="space-y-5">
        <div class="grid gap-3 rounded-xl bg-gray-50 p-4 text-sm dark:bg-dark-800 sm:grid-cols-3">
          <div><span class="text-gray-500 dark:text-dark-400">{{ t('growth.admin.order.status') }}</span><div class="mt-1"><GrowthStatusBadge :status="selected.status" /></div></div>
          <div><span class="text-gray-500 dark:text-dark-400">{{ t('growth.admin.order.fulfillment') }}</span><div class="mt-1"><GrowthStatusBadge :status="selected.fulfillment_status" /></div></div>
          <div><span class="text-gray-500 dark:text-dark-400">{{ t('growth.admin.order.pointsCost') }}</span><div class="mt-1"><PointsAmount :value="selected.points_cost" direction="OUT" /></div></div>
        </div>
        <div>
          <h4 class="mb-3 font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.order.timeline') }}</h4>
          <div class="space-y-3">
            <div v-for="step in timeline(selected)" :key="step.key" class="flex gap-3">
              <div class="mt-1 h-2.5 w-2.5 shrink-0 rounded-full" :class="step.done ? 'bg-teal-500' : 'bg-gray-300 dark:bg-dark-600'" />
              <div class="min-w-0 flex-1">
                <div class="text-sm font-medium text-gray-800 dark:text-dark-100">{{ step.label }}</div>
                <div v-if="step.time" class="text-xs text-gray-500 dark:text-dark-400">{{ new Date(step.time).toLocaleString() }}</div>
              </div>
            </div>
          </div>
        </div>
        <div class="grid gap-3 text-sm sm:grid-cols-2">
          <div><div class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.order.transfer') }}</div><div class="mt-1 font-mono">{{ selected.transfer_id ?? '-' }}</div></div>
          <div><div class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.order.refundTransfer') }}</div><div class="mt-1 font-mono">{{ selected.refund_transfer_id ?? '-' }}</div></div>
        </div>
        <div v-if="selected.failure_reason">
          <div class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.order.failureReason') }}</div>
          <p class="mt-1 rounded-xl bg-rose-50 p-3 text-sm text-rose-700 dark:bg-rose-950/30 dark:text-rose-200">{{ selected.failure_reason }}</p>
        </div>
        <div v-if="canManage && selected.status === 'FAILED' && !selected.refund_transfer_id" class="flex flex-wrap justify-end gap-2 border-t border-gray-200 pt-4 dark:border-dark-700">
          <button class="btn btn-secondary" @click="openAction(selected, 'retry')">{{ t('growth.admin.order.retry') }}</button>
          <button class="btn btn-danger" @click="openAction(selected, 'refund')">{{ t('growth.admin.order.refund') }}</button>
        </div>
      </div>
    </BaseDialog>

    <BaseDialog :show="actionOpen" :title="actionTitle" width="normal" @close="actionOpen = false">
      <form class="space-y-4" @submit.prevent="submitAction">
        <GrowthNotice v-if="actionError" tone="error">{{ actionError }}</GrowthNotice>
        <div v-if="selected" class="rounded-xl bg-gray-50 px-4 py-3 dark:bg-dark-800">
          <div class="font-mono text-xs text-gray-500 dark:text-dark-400">{{ selected.order_no }}</div>
          <div class="mt-1 font-medium text-gray-900 dark:text-white">{{ productTitle(selected) }}</div>
        </div>
        <template v-if="actionMode === 'retry'">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('growth.admin.order.retryHint') }}</p>
        </template>
        <template v-else>
          <GrowthNotice tone="warning">{{ actionMode === 'refund' ? t('growth.admin.order.refundWarning') : t('growth.admin.order.completeWarning') }}</GrowthNotice>
          <label class="block">
            <span class="input-label">{{ actionMode === 'refund' ? t('growth.common.reason') : t('growth.admin.order.actionReason') }}</span>
            <textarea v-model="actionReason" class="input min-h-24 w-full" :placeholder="t('growth.common.reasonPlaceholder')" />
          </label>
          <GrowthNotice v-if="actionConfirmed" tone="error">{{ t('growth.admin.budget.auditWarning') }}</GrowthNotice>
        </template>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="actionOpen = false">{{ t('growth.common.cancel') }}</button>
          <button type="submit" class="btn" :class="actionMode === 'refund' ? 'btn-danger' : 'btn-primary'" :disabled="actionDisabled || actionLoading">
            <LoadingSpinner v-if="actionLoading" size="sm" color="white" />
            <span v-else>{{ t('growth.common.confirm') }}</span>
          </button>
        </div>
      </form>
    </BaseDialog>
  </div>
</template>
