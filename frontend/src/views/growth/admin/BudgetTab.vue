<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { growthAdminAPI, type BudgetAccount, type BudgetEntry, type TaskAllocation } from '@/api/growth'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import BudgetUsageBar from '@/components/growth/BudgetUsageBar.vue'
import GrowthNotice from '@/components/growth/GrowthNotice.vue'
import GrowthStatusBadge from '@/components/growth/GrowthStatusBadge.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'
import { canRevokeTaskQuota, positiveInteger, quotaTone, reasonIsValid } from '@/components/growth/growthRules'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'

type DialogMode = 'INCREASE' | 'DECREASE' | 'ALLOCATE' | 'REVOKE'

const { t, te } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const budget = ref<BudgetAccount>()
const entries = ref<BudgetEntry[]>([])
const allocations = ref<TaskAllocation[]>([])
const loading = ref(true)
const error = ref('')
const dialogOpen = ref(false)
const historyOpen = ref(false)
const submitting = ref(false)
const dialogError = ref('')
const selectedTask = ref<TaskAllocation>()
const form = reactive({
  mode: 'INCREASE' as DialogMode,
  amount: '',
  reason: '',
  confirmed: false,
})

const canManage = computed(() => authStore.can('growth.manage'))
const isDanger = computed(() => form.mode === 'DECREASE' || form.mode === 'REVOKE')
const amount = computed(() => positiveInteger(form.amount))
const maxReduce = computed(() => {
  const total = budget.value?.total_amount ?? 0
  const locked = budget.value?.locked_amount
    ?? Math.max(budget.value?.allocated_amount ?? 0, budget.value?.distributed_amount ?? 0)
  return Math.max(0, total - locked)
})
const maxRevoke = computed(() => Math.max(0, selectedTask.value?.remaining_points ?? 0))
const revokeAllowed = computed(() => selectedTask.value ? canRevokeTaskQuota(selectedTask.value) : true)
const limitMessage = computed(() => {
  if (form.mode === 'DECREASE') return t('growth.admin.budget.maxReduce', { points: maxReduce.value.toLocaleString() })
  if (form.mode === 'REVOKE') return t('growth.admin.budget.maxRevoke', { points: maxRevoke.value.toLocaleString() })
  return ''
})
const dialogTitle = computed(() => {
  switch (form.mode) {
    case 'INCREASE': return t('growth.admin.budget.dialogIncreaseTitle')
    case 'DECREASE': return t('growth.admin.budget.dialogReduceTitle')
    case 'ALLOCATE': return t('growth.admin.budget.dialogAllocateTitle')
    default: return t('growth.admin.budget.dialogRevokeTitle')
  }
})
const canSubmit = computed(() => {
  if (!amount.value || !reasonIsValid(form.reason) || submitting.value) return false
  if (form.mode === 'DECREASE' && amount.value > maxReduce.value) return false
  if (form.mode === 'REVOKE' && (!revokeAllowed.value || amount.value > maxRevoke.value)) return false
  return true
})

const number = (value: number) => value.toLocaleString()
const dateTime = (value: string) => new Date(value).toLocaleString()
const entryTypeLabel = (value: string) => {
  const keys: Record<string, string> = {
    TOPUP: 'INCREASE',
    REDUCE: 'DECREASE',
    ALLOCATE: 'ALLOCATE',
    REVOKE: 'REVOKE',
  }
  if (keys[value] === 'INCREASE') return t('growth.admin.budget.increase')
  if (keys[value] === 'DECREASE') return t('growth.admin.budget.reduce')
  if (keys[value] === 'ALLOCATE') return t('growth.admin.budget.allocate')
  if (keys[value] === 'REVOKE') return t('growth.admin.budget.revoke')
  return value
}
const toneLabel = (item: TaskAllocation) => t(`growth.admin.budget.${quotaTone(item.remaining_points, item.usage_rate)}`)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [account, entryPage, allocationPage] = await Promise.all([
      growthAdminAPI.overview(),
      growthAdminAPI.budgetEntries({ page_size: 20 }),
      growthAdminAPI.allocations({ page_size: 100 }),
    ])
    budget.value = account.data
    entries.value = entryPage.data.items
    allocations.value = allocationPage.data.items
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    loading.value = false
  }
}

async function refreshEntries() {
  entries.value = (await growthAdminAPI.budgetEntries({ page_size: 20 })).data.items
}

async function refreshAllocations() {
  allocations.value = (await growthAdminAPI.allocations({ page_size: 100 })).data.items
}

function openDialog(mode: DialogMode, task?: TaskAllocation) {
  selectedTask.value = task
  form.mode = mode
  form.amount = mode === 'REVOKE' && task ? String(task.remaining_points) : ''
  form.reason = ''
  form.confirmed = false
  dialogError.value = ''
  dialogOpen.value = true
}

async function submit() {
  if (!canSubmit.value || !amount.value) return
  if (isDanger.value && !form.confirmed) {
    form.confirmed = true
    return
  }
  submitting.value = true
  dialogError.value = ''
  try {
    if (form.mode === 'INCREASE' || form.mode === 'DECREASE') {
      const result = await growthAdminAPI.createBudgetEntry({
        direction: form.mode,
        points: amount.value,
        reason: form.reason.trim(),
      })
      budget.value = result.data
      await refreshEntries()
    } else if (form.mode === 'ALLOCATE' && selectedTask.value) {
      const result = await growthAdminAPI.allocate({
        task_id: selectedTask.value.task_id,
        points: amount.value,
        reason: form.reason.trim(),
      })
      budget.value = result.data
      await refreshAllocations()
      await refreshEntries()
    } else if (form.mode === 'REVOKE' && selectedTask.value) {
      const result = await growthAdminAPI.revoke(selectedTask.value.task_id, {
        points: amount.value,
        reason: form.reason.trim(),
      })
      budget.value = result.data
      await refreshAllocations()
      await refreshEntries()
    }
    appStore.showToast('success', t('growth.common.operationSucceeded'))
    dialogOpen.value = false
  } catch (cause) {
    dialogError.value = errorMessage(cause)
  } finally {
    submitting.value = false
  }
}

function errorMessage(cause: unknown): string {
  if (cause && typeof cause === 'object') {
    const value = cause as { reason?: string; message?: string; code?: string }
    const code = value.code === 'CONFLICT' ? value.reason : value.code
    if (code && te(`growth.admin.budget.errors.${code}`)) {
      return t(`growth.admin.budget.errors.${code}`)
    }
    return value.reason || value.message || t('growth.common.requestFailed')
  }
  return t('growth.common.requestFailed')
}

onMounted(load)
</script>

<template>
  <div class="space-y-5">
    <GrowthNotice v-if="error" tone="error">
      <div class="flex items-center justify-between gap-3">
        <span>{{ error }}</span>
        <button class="btn btn-sm btn-secondary" @click="load">{{ t('growth.common.retry') }}</button>
      </div>
    </GrowthNotice>

    <div v-if="loading && !budget" class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <div v-for="index in 4" :key="index" class="card h-32 animate-pulse bg-gray-100 dark:bg-dark-800" />
    </div>

    <template v-else-if="budget">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <div class="card p-5">
          <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.total') }}</div>
          <div class="mt-2 text-2xl font-bold tabular-nums text-gray-900 dark:text-white">{{ number(budget.total_amount) }}</div>
          <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.totalHint') }}</div>
        </div>
        <div class="card p-5">
          <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.allocated') }}</div>
          <div class="mt-2 text-2xl font-bold tabular-nums text-sky-600 dark:text-sky-400">{{ number(budget.allocated_amount) }}</div>
          <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.allocatedHint') }}</div>
        </div>
        <div class="card p-5">
          <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.distributed') }}</div>
          <div class="mt-2 text-2xl font-bold tabular-nums text-teal-600 dark:text-teal-400">{{ number(budget.distributed_amount) }}</div>
          <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.distributedRate', { rate: `${(budget.distributed_rate * 100).toFixed(1)}%` }) }}</div>
        </div>
        <div class="card p-5">
          <div class="text-sm text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.unallocated') }}</div>
          <div class="mt-2 text-2xl font-bold tabular-nums text-amber-600 dark:text-amber-400">{{ number(budget.unallocated_amount) }}</div>
          <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.unallocatedHint') }}</div>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <template v-if="canManage">
          <button class="btn btn-primary" @click="openDialog('INCREASE')">{{ t('growth.admin.budget.increase') }}</button>
          <button class="btn btn-secondary" :disabled="maxReduce <= 0" @click="openDialog('DECREASE')">{{ t('growth.admin.budget.reduce') }}</button>
        </template>
        <div class="flex-1" />
        <button class="btn btn-secondary" @click="historyOpen = true">{{ t('growth.admin.budget.entries') }}</button>
      </div>

      <div v-if="budget.total_amount === 0" class="card">
        <EmptyState
          :title="t('growth.admin.budget.noBudget')"
          :description="t('growth.admin.budget.noBudgetHint')"
          :action-text="canManage ? t('growth.admin.budget.initialBudget') : undefined"
          @action="openDialog('INCREASE')"
        />
      </div>

      <div class="card overflow-hidden">
        <div class="card-header flex flex-wrap items-center justify-between gap-2">
          <h3 class="card-title">{{ t('growth.admin.budget.allocation') }}</h3>
          <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.budget.allocationCount', { count: allocations.length }) }}</span>
        </div>
        <EmptyState v-if="!allocations.length" :title="t('growth.common.noData')" />
        <div v-else class="overflow-x-auto">
          <table class="min-w-[860px] w-full table-auto divide-y divide-gray-200 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400">
              <tr>
                <th class="px-4 py-3">{{ t('growth.admin.budget.task') }}</th>
                <th class="px-4 py-3 text-right">{{ t('growth.admin.budget.allocatedColumn') }}</th>
                <th class="px-4 py-3 text-right">{{ t('growth.admin.budget.distributedColumn') }}</th>
                <th class="px-4 py-3 text-right">{{ t('growth.admin.budget.remaining') }}</th>
                <th class="px-4 py-3">{{ t('growth.admin.budget.usage') }}</th>
                <th class="px-4 py-3">{{ t('growth.admin.budget.status') }}</th>
                <th v-if="canManage" class="px-4 py-3">{{ t('growth.admin.budget.operation') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in allocations" :key="item.task_id" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
                <td class="px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ item.title }}</div>
                  <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ item.task_code }}</div>
                </td>
                <td class="px-4 py-3 text-right tabular-nums">{{ number(item.allocated_points) }}</td>
                <td class="px-4 py-3 text-right tabular-nums">{{ number(item.distributed_points) }}</td>
                <td class="px-4 py-3 text-right tabular-nums">{{ number(item.remaining_points) }}</td>
                <td class="px-4 py-3"><BudgetUsageBar :usage-rate="item.usage_rate" :remaining="item.remaining_points" /></td>
                <td class="px-4 py-3">
                  <span
                    class="inline-flex rounded-full px-2 py-0.5 text-xs font-medium"
                    :class="quotaTone(item.remaining_points, item.usage_rate) === 'healthy'
                      ? 'bg-teal-50 text-teal-700 dark:bg-teal-500/10 dark:text-teal-300'
                      : quotaTone(item.remaining_points, item.usage_rate) === 'warning'
                        ? 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'
                        : 'bg-rose-50 text-rose-700 dark:bg-rose-500/10 dark:text-rose-300'"
                  >{{ toneLabel(item) }}</span>
                </td>
                <td v-if="canManage" class="px-4 py-3">
                  <div class="flex gap-1">
                    <button class="btn btn-sm btn-secondary" :disabled="item.status === 'OFFLINE'" @click="openDialog('ALLOCATE', item)">{{ t('growth.admin.budget.allocate') }}</button>
                    <button class="btn btn-sm btn-secondary" :disabled="item.status !== 'OFFLINE' || item.remaining_points <= 0" @click="openDialog('REVOKE', item)">{{ t('growth.admin.budget.revoke') }}</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </template>

    <BaseDialog :show="dialogOpen" :title="dialogTitle" width="normal" @close="dialogOpen = false">
      <form class="space-y-4" @submit.prevent="submit">
        <GrowthNotice v-if="dialogError" tone="error">{{ dialogError }}</GrowthNotice>
        <div v-if="selectedTask" class="rounded-xl bg-gray-50 px-4 py-3 text-sm dark:bg-dark-800">
          <div class="font-medium text-gray-900 dark:text-white">{{ selectedTask.title }}</div>
          <div class="mt-1 flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
            <span>{{ selectedTask.task_code }}</span>
            <GrowthStatusBadge :status="selectedTask.status" />
          </div>
        </div>
        <label class="block">
          <span class="input-label">{{ t('growth.common.amount') }}</span>
          <input v-model="form.amount" class="input w-full" inputmode="numeric" autocomplete="off" :placeholder="form.mode === 'INCREASE' ? '10000' : '1000'">
        </label>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ limitMessage }}</p>
        <p v-if="form.mode === 'DECREASE'" class="text-xs text-rose-600 dark:text-rose-300">{{ t('growth.admin.budget.reduceHint') }}</p>
        <p v-if="form.mode === 'REVOKE'" class="text-xs text-rose-600 dark:text-rose-300">{{ t('growth.admin.budget.revokeHint') }}</p>
        <p v-if="form.mode === 'REVOKE' && !revokeAllowed" class="text-xs text-rose-600 dark:text-rose-300">{{ t('growth.admin.budget.revokeUnavailable') }}</p>
        <label class="block">
          <span class="input-label">{{ t('growth.common.reason') }}</span>
          <textarea v-model="form.reason" class="input min-h-24 w-full" :placeholder="t('growth.common.reasonPlaceholder')" />
          <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('growth.common.reasonHint') }}</span>
        </label>
        <GrowthNotice v-if="isDanger && form.confirmed" tone="error">{{ t('growth.admin.budget.auditWarning') }}</GrowthNotice>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="dialogOpen = false">{{ t('growth.common.cancel') }}</button>
          <button type="submit" class="btn" :class="isDanger ? 'btn-danger' : 'btn-primary'" :disabled="!canSubmit">
            <LoadingSpinner v-if="submitting" size="sm" color="white" />
            <span v-else>{{ isDanger && !form.confirmed ? t('growth.common.confirm') : t('growth.common.confirm') }}</span>
          </button>
        </div>
      </form>
    </BaseDialog>

    <BaseDialog :show="historyOpen" :title="t('growth.admin.budget.entries')" width="wide" @close="historyOpen = false">
      <EmptyState v-if="!entries.length" :title="t('growth.common.noData')" />
      <div v-else class="space-y-2">
        <div v-for="entry in entries" :key="entry.id" class="rounded-xl border border-gray-200 p-3 dark:border-dark-700">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="font-medium text-gray-900 dark:text-white">{{ entryTypeLabel(entry.entry_type) }} · {{ entry.task_title || '-' }}</div>
            <PointsAmount :value="entry.points" :direction="['REDUCE', 'REVOKE'].includes(entry.entry_type) ? 'OUT' : 'IN'" signed />
          </div>
          <div class="mt-2 grid gap-2 text-xs text-gray-500 dark:text-dark-400 sm:grid-cols-3">
            <span>{{ dateTime(entry.created_at) }}</span>
            <span>{{ t('growth.admin.budget.budgetAfter') }} {{ number(entry.budget_after) }}</span>
            <span>{{ t('growth.admin.budget.allocatedAfter') }} {{ number(entry.allocated_after) }}</span>
          </div>
          <p class="mt-2 text-sm text-gray-700 dark:text-dark-200">{{ entry.reason }}</p>
        </div>
      </div>
    </BaseDialog>
  </div>
</template>
