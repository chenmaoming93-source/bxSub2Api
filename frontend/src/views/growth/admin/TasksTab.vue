<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  growthAdminAPI,
  type BudgetAccount,
  type GrowthTask,
  type TaskRuleVersion,
} from '@/api/growth'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import BudgetUsageBar from '@/components/growth/BudgetUsageBar.vue'
import GrowthNotice from '@/components/growth/GrowthNotice.vue'
import GrowthStatusBadge from '@/components/growth/GrowthStatusBadge.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'
import {
  positiveInteger,
  reasonIsValid,
  taskActions,
  taskRuleDifferences,
  type TaskAction,
} from '@/components/growth/growthRules'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

type EditorMode = 'create' | 'edit'
type ActionMode = 'offline' | 'online' | 'publish' | 'quota' | ''

const { t, te } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const items = ref<GrowthTask[]>([])
const budget = ref<BudgetAccount>()
const error = ref('')
const loading = ref(false)
const search = ref('')
const categoryFilter = ref('')
const statusFilter = ref('')
const editorOpen = ref(false)
const editorMode = ref<EditorMode>('create')
const editorError = ref('')
const saving = ref(false)
const selectedTask = ref<GrowthTask>()
const actionOpen = ref(false)
const actionMode = ref<ActionMode>('')
const actionAmount = ref('')
const actionReason = ref('')
const actionDateTime = ref('')
const revokeRemaining = ref(true)
const actionConfirmed = ref(false)
const actionError = ref('')
const historyOpen = ref(false)
const historyLoading = ref(false)
const history = ref<TaskRuleVersion[]>([])
const codeTouched = ref(false)
const form = reactive({
  code: '',
  title: '',
  description: '',
  icon: 'sparkles',
  category: 'DAILY',
  taskType: 'DAILY',
  sortOrder: 0,
  initialAllocation: '',
  rewardPoints: '10',
  target: '1',
  periodType: 'DAILY',
  dailyLimit: '1',
  perUserLimit: '1',
})

const canManage = computed(() => authStore.can('growth.manage'))
const availableBudget = computed(() => budget.value?.unallocated_amount ?? 0)
const existingCode = computed(() => items.value.some(item =>
  item.task_code.toLowerCase() === form.code.trim().toLowerCase() &&
  item.task_id !== selectedTask.value?.task_id
))
const initialQuota = computed(() => positiveInteger(form.initialAllocation))
const rewardPoints = computed(() => positiveInteger(form.rewardPoints))
const editorValid = computed(() => {
  if (!form.code.trim() && editorMode.value === 'create') return false
  if (!form.title.trim() || !rewardPoints.value) return false
  if (Number(form.target) <= 0 || Number(form.dailyLimit) <= 0 || Number(form.perUserLimit) <= 0) return false
  if (editorMode.value === 'create' && (!initialQuota.value || initialQuota.value > availableBudget.value || existingCode.value)) return false
  return true
})
const actionTitle = computed(() => {
  if (actionMode.value === 'offline') return t('growth.admin.task.offlineTitle')
  if (actionMode.value === 'online') return t('growth.admin.task.onlineTitle')
  if (actionMode.value === 'publish') return t('growth.admin.task.publishTitle')
  return t('growth.admin.budget.dialogAllocateTitle')
})
const actionSubmitDisabled = computed(() => {
  if (actionMode.value === 'quota') {
    const points = positiveInteger(actionAmount.value)
    return !points || points > availableBudget.value || !reasonIsValid(actionReason.value)
  }
  if (actionMode.value === 'online') {
    const points = positiveInteger(actionAmount.value)
    return !points || points > availableBudget.value
  }
  if (actionMode.value === 'offline') return !reasonIsValid(actionReason.value)
  return false
})

const number = (value: number) => value.toLocaleString()
const dateTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'
const enumLabel = (group: 'taskTypes' | 'categories' | 'periodTypes', value: string) => {
  const key = `growth.admin.task.${group}.${value}`
  return te(key) ? t(key) : value
}
const taskTypeLabel = (value: string) => enumLabel('taskTypes', value)
const categoryLabel = (value: string) => enumLabel('categories', value)
const periodTypeLabel = (value: string) => enumLabel('periodTypes', value)
const estimate = computed(() => rewardPoints.value ? Math.floor(availableBudget.value / rewardPoints.value) : 0)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const [taskPage, account] = await Promise.all([
      growthAdminAPI.tasks({
        search: search.value,
        category: categoryFilter.value,
        status: statusFilter.value,
        page_size: 100,
      }),
      growthAdminAPI.overview(),
    ])
    items.value = taskPage.data.items
    budget.value = account.data
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    loading.value = false
  }
}

function resetForm() {
  Object.assign(form, {
    code: '',
    title: '',
    description: '',
    icon: 'sparkles',
    category: 'DAILY',
    taskType: 'DAILY',
    sortOrder: 0,
    initialAllocation: '',
    rewardPoints: '10',
    target: '1',
    periodType: 'DAILY',
    dailyLimit: '1',
    perUserLimit: '1',
  })
  codeTouched.value = false
  editorError.value = ''
}

function createTask() {
  selectedTask.value = undefined
  resetForm()
  editorMode.value = 'create'
  editorOpen.value = true
}

function editTask(task: GrowthTask) {
  selectedTask.value = task
  Object.assign(form, {
    code: task.task_code,
    title: task.title,
    description: task.description,
    icon: task.icon,
    category: task.category,
    taskType: task.task_type,
    sortOrder: task.sort_order,
    initialAllocation: '',
    rewardPoints: String(task.reward_points),
    target: '1',
    periodType: task.period_type,
    dailyLimit: String(task.daily_limit),
    perUserLimit: String(task.per_user_limit),
  })
  codeTouched.value = true
  editorError.value = ''
  editorMode.value = 'edit'
  editorOpen.value = true
}

async function saveTask() {
  if (!editorValid.value || saving.value) return
  saving.value = true
  editorError.value = ''
  const rule = {
    target: Number(form.target),
    reward_points: rewardPoints.value,
    period_type: form.periodType,
    daily_limit: Number(form.dailyLimit),
    per_user_limit: Number(form.perUserLimit),
  }
  try {
    if (editorMode.value === 'create') {
      const result = await growthAdminAPI.createTask({
        task_code: form.code.trim(),
        title: form.title.trim(),
        description: form.description.trim(),
        category: form.category,
        task_type: form.taskType,
        icon: form.icon.trim(),
        sort_order: Number(form.sortOrder) || 0,
        initial_allocation: initialQuota.value,
        rule,
        request_id: crypto.randomUUID(),
      })
      items.value = [result.data, ...items.value]
      if (budget.value) {
        budget.value = {
          ...budget.value,
          allocated_amount: budget.value.allocated_amount + (initialQuota.value ?? 0),
          unallocated_amount: budget.value.unallocated_amount - (initialQuota.value ?? 0),
        }
      }
    } else if (selectedTask.value) {
      const result = await growthAdminAPI.updateTask(selectedTask.value.task_id, {
        title: form.title.trim(),
        description: form.description.trim(),
        category: form.category,
        task_type: form.taskType,
        icon: form.icon.trim(),
        sort_order: Number(form.sortOrder) || 0,
        rule,
        request_id: crypto.randomUUID(),
      })
      replaceTask(result.data)
    }
    appStore.showToast('success', t('growth.common.operationSucceeded'))
    editorOpen.value = false
  } catch (cause) {
    const value = cause as { code?: string }
    editorError.value = value?.code === 'TASK_CODE_DUPLICATED'
      ? t('growth.admin.task.duplicateCode')
      : errorMessage(cause)
  } finally {
    saving.value = false
  }
}

function replaceTask(task: GrowthTask) {
  const index = items.value.findIndex(item => item.task_id === task.task_id)
  if (index >= 0) items.value.splice(index, 1, task)
  else items.value.unshift(task)
  if (selectedTask.value?.task_id === task.task_id) selectedTask.value = task
}

function availableActions(task: GrowthTask): TaskAction[] {
  return taskActions(task, canManage.value)
}

function openAction(task: GrowthTask, mode: Exclude<ActionMode, ''>) {
  selectedTask.value = task
  actionMode.value = mode
  actionAmount.value = mode === 'online' ? '' : ''
  actionReason.value = ''
  actionDateTime.value = ''
  revokeRemaining.value = true
  actionConfirmed.value = false
  actionError.value = ''
  actionOpen.value = true
}

async function submitAction() {
  if (!selectedTask.value || actionSubmitDisabled.value) return
  if (actionMode.value === 'offline' && !actionConfirmed.value) {
    actionConfirmed.value = true
    return
  }
  try {
    let result
    if (actionMode.value === 'publish') {
      result = await growthAdminAPI.publishTask(selectedTask.value.task_id, {
        effective_from: actionDateTime.value ? new Date(actionDateTime.value).toISOString() : null,
        request_id: crypto.randomUUID(),
      })
    } else if (actionMode.value === 'offline') {
      result = await growthAdminAPI.offlineTask(selectedTask.value.task_id, {
        reason: actionReason.value.trim(),
        revoke_remaining: revokeRemaining.value,
        request_id: crypto.randomUUID(),
      })
    } else if (actionMode.value === 'online') {
      result = await growthAdminAPI.onlineTask(selectedTask.value.task_id, {
        initial_allocation: positiveInteger(actionAmount.value),
        request_id: crypto.randomUUID(),
      })
    } else {
      const budgetResult = await growthAdminAPI.allocate({
        task_id: selectedTask.value.task_id,
        points: positiveInteger(actionAmount.value),
        reason: actionReason.value.trim(),
        request_id: crypto.randomUUID(),
      })
      budget.value = budgetResult.data
      await load()
      actionOpen.value = false
      appStore.showToast('success', t('growth.common.operationSucceeded'))
      return
    }
    replaceTask(result.data)
    actionOpen.value = false
    appStore.showToast('success', t('growth.common.operationSucceeded'))
  } catch (cause) {
    actionError.value = errorMessage(cause)
  }
}

async function openHistory(task: GrowthTask) {
  selectedTask.value = task
  history.value = []
  historyOpen.value = true
  historyLoading.value = true
  try {
    history.value = (await growthAdminAPI.ruleVersions(task.task_id)).data.items
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    historyLoading.value = false
  }
}

function differences(rule: TaskRuleVersion, index: number): string[] {
  const previous = history.value[index + 1]
  return taskRuleDifferences(rule as unknown as Record<string, unknown>, previous as unknown as Record<string, unknown> | undefined)
}

function errorMessage(cause: unknown): string {
  if (cause && typeof cause === 'object') {
    const value = cause as { reason?: string; message?: string }
    return value.reason || value.message || t('growth.common.requestFailed')
  }
  return t('growth.common.requestFailed')
}

function actionLabel(action: TaskAction): string {
  const labels: Record<TaskAction, string> = {
    edit: t('growth.admin.task.edit'),
    publish: selectedTask.value?.pending_rule_version ? t('growth.admin.task.publishVersion') : t('growth.admin.task.publish'),
    offline: t('growth.admin.task.offline'),
    online: t('growth.admin.task.online'),
    quota: t('growth.admin.task.quota'),
    history: t('growth.admin.task.history'),
  }
  return labels[action]
}

function runAction(task: GrowthTask, action: TaskAction) {
  selectedTask.value = task
  if (action === 'edit') editTask(task)
  else if (action === 'history') void openHistory(task)
  else openAction(task, action)
}

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

    <div class="card p-4">
      <div class="flex flex-col gap-3 lg:flex-row lg:items-center">
        <input v-model="search" class="input min-w-0 flex-1" :placeholder="t('growth.admin.task.searchPlaceholder')" @keyup.enter="load">
        <select v-model="categoryFilter" class="input w-full lg:w-40" @change="load">
          <option value="">{{ t('growth.admin.task.filterType') }}</option>
          <option value="CHECKIN">{{ categoryLabel('CHECKIN') }}</option>
          <option value="DAILY">{{ categoryLabel('DAILY') }}</option>
          <option value="ACHIEVEMENT">{{ categoryLabel('ACHIEVEMENT') }}</option>
        </select>
        <select v-model="statusFilter" class="input w-full lg:w-40" @change="load">
          <option value="">{{ t('growth.admin.task.filterStatus') }}</option>
          <option value="DRAFT">{{ t('growth.status.DRAFT') }}</option>
          <option value="ACTIVE">{{ t('growth.status.ACTIVE') }}</option>
          <option value="OFFLINE">{{ t('growth.status.OFFLINE') }}</option>
        </select>
        <button class="btn btn-secondary" @click="load">{{ t('growth.common.search') }}</button>
        <button v-if="canManage" class="btn btn-primary" :disabled="availableBudget <= 0" @click="createTask">{{ t('growth.admin.task.create') }}</button>
      </div>
      <p v-if="canManage && availableBudget <= 0" class="mt-2 text-xs text-amber-600 dark:text-amber-300">
        {{ t('growth.admin.task.insufficientBudget') }}
      </p>
    </div>

    <div class="card overflow-hidden">
      <div v-if="loading && !items.length" class="p-12 text-center"><LoadingSpinner /></div>
      <EmptyState
        v-else-if="!items.length"
        :title="t('growth.admin.task.noTasks')"
        :description="t('growth.admin.task.noTasksHint')"
        :action-text="canManage ? t('growth.admin.task.create') : undefined"
        @action="createTask"
      />
      <div v-else class="overflow-x-auto">
        <table class="min-w-[1080px] w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-3">{{ t('growth.admin.task.task') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.task.type') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.task.reward') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.task.period') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.task.quotaColumn') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.task.status') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.task.updated') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.task.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="task in items" :key="task.task_id" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
              <td class="px-4 py-3">
                <button class="text-left font-medium text-gray-900 hover:text-primary-600 dark:text-white dark:hover:text-primary-400" @click="runAction(task, 'history')">{{ task.title }}</button>
                <div class="mt-0.5 font-mono text-xs text-gray-500 dark:text-dark-400">{{ task.task_code }}</div>
              </td>
              <td class="px-4 py-3"><span class="rounded-md bg-gray-100 px-2 py-1 text-xs font-medium dark:bg-dark-700">{{ taskTypeLabel(task.task_type) }}</span></td>
              <td class="px-4 py-3">
                <PointsAmount :value="task.reward_points" unit="PT" />
                <div class="mt-1 flex flex-wrap gap-1">
                  <span v-if="task.rule_version" class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.task.publishedVersion', { version: task.rule_version }) }}</span>
                  <span v-if="task.pending_rule_version" class="rounded-full bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">{{ t('growth.admin.task.pendingVersion', { version: task.pending_rule_version }) }}</span>
                </div>
              </td>
              <td class="px-4 py-3 text-xs text-gray-600 dark:text-dark-300">
                <div>{{ periodTypeLabel(task.period_type) }}</div>
                <div>{{ t('growth.admin.task.timesPerDay', { count: task.daily_limit }) }} · {{ t('growth.admin.task.timesPerUser', { count: task.per_user_limit }) }}</div>
              </td>
              <td class="min-w-44 px-4 py-3">
                <div class="mb-1 text-xs tabular-nums text-gray-600 dark:text-dark-300">{{ number(task.allocated_points) }} · {{ t('growth.admin.task.distributed', { points: number(task.distributed_points) }) }}</div>
                <BudgetUsageBar :usage-rate="task.usage_rate" :remaining="task.remaining_points" />
              </td>
              <td class="px-4 py-3">
                <GrowthStatusBadge :status="task.status" />
                <div v-if="task.remaining_points <= 0 && task.status === 'ACTIVE'" class="mt-1 text-xs text-rose-600 dark:text-rose-300">{{ t('growth.admin.budget.exhausted') }}</div>
              </td>
              <td class="px-4 py-3 text-xs text-gray-500 dark:text-dark-400">{{ dateTime(task.updated_at) }}</td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1">
                  <button v-for="action in availableActions(task)" :key="action" class="btn btn-sm btn-secondary" @click="runAction(task, action)">{{ actionLabel(action) }}</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <BaseDialog :show="editorOpen" :title="editorMode === 'create' ? t('growth.admin.task.createTitle') : t('growth.admin.task.editTitle')" width="wide" @close="editorOpen = false">
      <form class="space-y-5" @submit.prevent="saveTask">
        <GrowthNotice v-if="editorError" tone="error">{{ editorError }}</GrowthNotice>
        <div v-if="editorMode === 'create'" class="rounded-xl bg-sky-50 px-4 py-3 text-sm text-sky-800 dark:bg-sky-950/30 dark:text-sky-200">
          {{ t('growth.admin.task.availableBudget', { points: number(availableBudget) }) }}
        </div>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.code') }}</span>
            <input v-model="form.code" class="input w-full" :disabled="editorMode === 'edit'" @blur="codeTouched = true">
            <span class="mt-1 block text-xs" :class="codeTouched && existingCode ? 'text-rose-600' : 'text-gray-500 dark:text-dark-400'">
              {{ codeTouched && existingCode ? t('growth.admin.task.duplicateCode') : t('growth.admin.task.codeHint') }}
            </span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.title') }}</span>
            <input v-model="form.title" class="input w-full">
          </label>
          <label class="block sm:col-span-2">
            <span class="input-label">{{ t('growth.admin.task.description') }}</span>
            <textarea v-model="form.description" class="input min-h-20 w-full" :placeholder="t('growth.admin.task.descriptionPlaceholder')" />
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.icon') }}</span>
            <input v-model="form.icon" class="input w-full">
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.taskType') }}</span>
            <select v-model="form.taskType" class="input w-full">
              <option value="CHECKIN">{{ taskTypeLabel('CHECKIN') }}</option>
              <option value="DAILY">{{ taskTypeLabel('DAILY') }}</option>
              <option value="ACHIEVEMENT">{{ taskTypeLabel('ACHIEVEMENT') }}</option>
            </select>
            <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.task.taskTypeHelp') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.category') }}</span>
            <select v-model="form.category" class="input w-full">
              <option value="CHECKIN">{{ categoryLabel('CHECKIN') }}</option>
              <option value="DAILY">{{ categoryLabel('DAILY') }}</option>
              <option value="ACHIEVEMENT">{{ categoryLabel('ACHIEVEMENT') }}</option>
            </select>
            <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.task.categoryHelp') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.sortOrder') }}</span>
            <input v-model="form.sortOrder" class="input w-full" type="number">
          </label>
          <label v-if="editorMode === 'create'" class="block sm:col-span-2">
            <span class="input-label">{{ t('growth.admin.task.initialAllocation') }}</span>
            <input v-model="form.initialAllocation" class="input w-full" inputmode="numeric">
          </label>
        </div>
        <div class="border-t border-gray-200 pt-4 dark:border-dark-700">
          <h4 class="mb-3 font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.task.reward') }}</h4>
          <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <label class="block">
              <span class="input-label">{{ t('growth.admin.task.rewardPoints') }}</span>
              <input v-model="form.rewardPoints" class="input w-full" inputmode="numeric">
            </label>
            <label class="block">
              <span class="input-label">{{ t('growth.admin.task.periodType') }}</span>
              <select v-model="form.periodType" class="input w-full">
                <option value="DAILY">{{ periodTypeLabel('DAILY') }}</option>
                <option value="WEEKLY">{{ periodTypeLabel('WEEKLY') }}</option>
                <option value="ONCE">{{ periodTypeLabel('ONCE') }}</option>
              </select>
            </label>
            <label class="block">
              <span class="input-label">{{ t('growth.admin.task.target') }}</span>
              <input v-model="form.target" class="input w-full" inputmode="numeric">
              <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.task.targetHelp') }}</span>
            </label>
            <label class="block">
              <span class="input-label">{{ t('growth.admin.task.dailyLimit') }}</span>
              <input v-model="form.dailyLimit" class="input w-full" inputmode="numeric">
            </label>
            <label class="block">
              <span class="input-label">{{ t('growth.admin.task.perUserLimit') }}</span>
              <input v-model="form.perUserLimit" class="input w-full" inputmode="numeric">
            </label>
          </div>
          <p class="mt-2 text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.task.rewardEstimate', { count: number(estimate) }) }}</p>
          <p v-if="editorMode === 'edit'" class="mt-2 text-xs text-amber-600 dark:text-amber-300">{{ t('growth.admin.task.pendingVersion', { version: (selectedTask?.rule_version ?? 0) + 1 }) }}</p>
        </div>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="editorOpen = false">{{ t('growth.common.cancel') }}</button>
          <button type="submit" class="btn btn-primary" :disabled="!editorValid || saving">
            <LoadingSpinner v-if="saving" size="sm" color="white" />
            <span v-else>{{ t('growth.admin.task.save') }}</span>
          </button>
        </div>
      </form>
    </BaseDialog>

    <BaseDialog :show="actionOpen" :title="actionTitle" width="normal" @close="actionOpen = false">
      <form class="space-y-4" @submit.prevent="submitAction">
        <GrowthNotice v-if="actionError" tone="error">{{ actionError }}</GrowthNotice>
        <div v-if="selectedTask" class="rounded-xl bg-gray-50 px-4 py-3 dark:bg-dark-800">
          <div class="font-medium text-gray-900 dark:text-white">{{ selectedTask.title }}</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ selectedTask.task_code }}</div>
        </div>

        <template v-if="actionMode === 'publish'">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('growth.admin.task.effectiveHint') }}</p>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.effectiveFrom') }}</span>
            <input v-model="actionDateTime" type="datetime-local" class="input w-full">
          </label>
        </template>

        <template v-else-if="actionMode === 'offline'">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('growth.admin.task.offlineHint') }}</p>
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-dark-200">
            <input v-model="revokeRemaining" type="checkbox" class="rounded border-gray-300">
            {{ t('growth.admin.task.revokeRemaining', { points: number(selectedTask?.remaining_points ?? 0) }) }}
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.common.reason') }}</span>
            <textarea v-model="actionReason" class="input min-h-24 w-full" :placeholder="t('growth.common.reasonPlaceholder')" />
          </label>
          <GrowthNotice v-if="actionConfirmed" tone="error">{{ t('growth.admin.budget.auditWarning') }}</GrowthNotice>
        </template>

        <template v-else-if="actionMode === 'online'">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('growth.admin.task.onlineHint') }}</p>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.task.availableBudget', { points: number(availableBudget) }) }}</p>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.initialQuota') }}</span>
            <input v-model="actionAmount" class="input w-full" inputmode="numeric">
          </label>
        </template>

        <template v-else>
          <p class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.task.availableBudget', { points: number(availableBudget) }) }}</p>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.task.initialQuota') }}</span>
            <input v-model="actionAmount" class="input w-full" inputmode="numeric">
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.common.reason') }}</span>
            <textarea v-model="actionReason" class="input min-h-24 w-full" />
          </label>
        </template>

        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="actionOpen = false">{{ t('growth.common.cancel') }}</button>
          <button type="submit" class="btn" :class="actionMode === 'offline' ? 'btn-danger' : 'btn-primary'" :disabled="actionSubmitDisabled">{{ t('growth.common.confirm') }}</button>
        </div>
      </form>
    </BaseDialog>

    <BaseDialog :show="historyOpen" :title="t('growth.admin.task.historyTitle', { title: selectedTask?.title ?? '' })" width="wide" @close="historyOpen = false">
      <div v-if="historyLoading" class="py-12 text-center"><LoadingSpinner /></div>
      <EmptyState v-else-if="!history.length" :title="t('growth.common.noData')" />
      <div v-else class="space-y-3">
        <div v-for="(rule, index) in history" :key="rule.id" class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <span class="font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.task.version') }} v{{ rule.version }}</span>
              <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="rule.published_at
                ? 'bg-teal-50 text-teal-700 dark:bg-teal-500/10 dark:text-teal-300'
                : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'">
                {{ rule.published_at ? t('growth.admin.task.published') : t('growth.admin.task.pending') }}
              </span>
            </div>
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.task.operator') }} #{{ rule.created_by }}</span>
          </div>
          <div class="mt-3 grid gap-2 text-sm sm:grid-cols-3">
            <div>{{ t('growth.admin.task.rewardPoints') }}: <b>{{ number(rule.reward_points) }}</b></div>
            <div>{{ t('growth.admin.task.dailyLimit') }}: <b>{{ rule.daily_limit }}</b></div>
            <div>{{ t('growth.admin.task.perUserLimit') }}: <b>{{ rule.per_user_limit }}</b></div>
          </div>
          <div class="mt-2 text-xs text-gray-500 dark:text-dark-400">
            {{ t('growth.admin.task.effectiveRange') }}: {{ dateTime(rule.effective_from) }} — {{ dateTime(rule.effective_to) }}
          </div>
          <p v-if="index < history.length - 1" class="mt-2 text-xs text-amber-700 dark:text-amber-300">
            {{ differences(rule, index).length
              ? t('growth.admin.task.differences', { fields: differences(rule, index).join(', ') })
              : t('growth.admin.task.noDifference') }}
          </p>
        </div>
      </div>
    </BaseDialog>
  </div>
</template>
