<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { growthAdminAPI, type LedgerItem, type UserExchange, type UserPoints } from '@/api/growth'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import GrowthNotice from '@/components/growth/GrowthNotice.vue'
import GrowthStatusBadge from '@/components/growth/GrowthStatusBadge.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'
import { positiveInteger, reasonIsValid } from '@/components/growth/growthRules'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const appStore = useAppStore()
const items = ref<UserPoints[]>([])
const loading = ref(false)
const error = ref('')
const searchInput = ref(String(route.query.q ?? ''))
const minInput = ref(String(route.query.min ?? ''))
const maxInput = ref(String(route.query.max ?? ''))
const sort = ref(String(route.query.sort ?? 'balance_desc'))
const drawerOpen = ref(false)
const selected = ref<UserPoints>()
const ledger = ref<LedgerItem[]>([])
const exchanges = ref<UserExchange[]>([])
const detailLoading = ref(false)
const ledgerCursor = ref('')
const ledgerHasMore = ref(false)
const adjustOpen = ref(false)
const adjustDirection = ref<'INCREASE' | 'DECREASE'>('INCREASE')
const adjustPoints = ref('')
const adjustReason = ref('')
const adjustError = ref('')
const adjustConfirmed = ref(false)
const adjusting = ref(false)

const canAdjust = computed(() => authStore.can('growth.adjust'))
const sortedItems = computed(() => [...items.value].sort((left, right) => {
  if (sort.value === 'balance_asc') return left.available_points - right.available_points
  if (sort.value === 'recent') return new Date(right.latest_change_at ?? 0).getTime() - new Date(left.latest_change_at ?? 0).getTime()
  return right.available_points - left.available_points
}))
const adjustAmount = computed(() => positiveInteger(adjustPoints.value))
const projectedBalance = computed(() => {
  const current = selected.value?.available_points ?? 0
  if (!adjustAmount.value) return current
  return adjustDirection.value === 'INCREASE' ? current + adjustAmount.value : current - adjustAmount.value
})
const adjustDisabled = computed(() => {
  if (!adjustAmount.value || !reasonIsValid(adjustReason.value)) return true
  return adjustDirection.value === 'DECREASE' && (selected.value?.available_points ?? 0) < adjustAmount.value
})

const number = (value: number) => value.toLocaleString()
const userName = (user: UserPoints) => user.username || user.email
const dateTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'

async function load() {
  loading.value = true
  error.value = ''
  try {
    const min = Number(minInput.value)
    const max = Number(maxInput.value)
    items.value = (await growthAdminAPI.users({
      search: searchInput.value.trim(),
      min_points: Number.isFinite(min) && min > 0 ? min : undefined,
      max_points: Number.isFinite(max) && max > 0 ? max : undefined,
      page_size: 100,
    })).data.items
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    loading.value = false
  }
}

async function applyFilters() {
  const query: Record<string, string> = { tab: 'users' }
  if (searchInput.value.trim()) query.q = searchInput.value.trim()
  if (positiveInteger(minInput.value)) query.min = String(positiveInteger(minInput.value))
  if (positiveInteger(maxInput.value)) query.max = String(positiveInteger(maxInput.value))
  if (sort.value !== 'balance_desc') query.sort = sort.value
  await router.replace({ query })
  await load()
}

async function resetFilters() {
  searchInput.value = ''
  minInput.value = ''
  maxInput.value = ''
  sort.value = 'balance_desc'
  await router.replace({ query: { tab: 'users' } })
  await load()
}

async function openUser(user: UserPoints) {
  selected.value = user
  drawerOpen.value = true
  await loadUserDetails()
}

async function loadUserDetails(reset = true) {
  if (!selected.value) return
  detailLoading.value = true
  try {
    const [ledgerResult, exchangeResult] = await Promise.all([
      growthAdminAPI.userLedger(selected.value.user_id, {
        limit: 20,
        before_entry_id: reset ? undefined : Number(ledgerCursor.value) || undefined,
      }),
      growthAdminAPI.userExchanges(selected.value.user_id, { page_size: 20 }),
    ])
    ledger.value = reset ? ledgerResult.data.items : [...ledger.value, ...ledgerResult.data.items.filter(item => !ledger.value.some(existing => existing.entry_id === item.entry_id))]
    ledgerCursor.value = ledgerResult.data.next_cursor
    ledgerHasMore.value = ledgerResult.data.has_more
    exchanges.value = exchangeResult.data.items
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    detailLoading.value = false
  }
}

function openAdjust() {
  adjustDirection.value = 'INCREASE'
  adjustPoints.value = ''
  adjustReason.value = ''
  adjustError.value = ''
  adjustConfirmed.value = false
  adjustOpen.value = true
}

async function submitAdjust() {
  if (!selected.value || adjustDisabled.value || adjusting.value) return
  if (!adjustConfirmed.value) {
    adjustConfirmed.value = true
    return
  }
  adjusting.value = true
  adjustError.value = ''
  try {
    const result = await growthAdminAPI.adjust(selected.value.user_id, {
      direction: adjustDirection.value,
      points: adjustAmount.value,
      reason: adjustReason.value.trim(),
      request_id: crypto.randomUUID(),
    })
    const updated: UserPoints = {
      ...selected.value,
      available_points: result.data.after_balance,
      total_earned: adjustDirection.value === 'INCREASE'
        ? selected.value.total_earned + result.data.points
        : selected.value.total_earned,
      total_spent: adjustDirection.value === 'DECREASE'
        ? selected.value.total_spent + result.data.points
        : selected.value.total_spent,
      latest_change_at: new Date().toISOString(),
    }
    selected.value = updated
    const index = items.value.findIndex(item => item.user_id === updated.user_id)
    if (index >= 0) items.value.splice(index, 1, updated)
    adjustOpen.value = false
    appStore.showToast('success', t('growth.admin.userPoints.submitted', {
      before: number(result.data.before_balance),
      after: number(result.data.after_balance),
    }))
    await loadUserDetails(true)
  } catch (cause) {
    adjustError.value = errorMessage(cause)
  } finally {
    adjusting.value = false
  }
}

function exportCsv() {
  const rows = [
    ['user_id', 'username', 'email', 'available_points', 'total_earned', 'total_spent', 'latest_change_at'],
    ...sortedItems.value.map(user => [
      String(user.user_id),
      user.username,
      user.email,
      String(user.available_points),
      String(user.total_earned),
      String(user.total_spent),
      user.latest_change_at ?? '',
    ]),
  ]
  const csv = rows.map(row => row.map(value => `"${value.replace(/"/g, '""')}"`).join(',')).join('\n')
  const blob = new Blob([`\uFEFF${csv}`], { type: 'text/csv;charset=utf-8' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.href = url
  link.download = `growth-users-${new Date().toISOString().slice(0, 10)}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

function errorMessage(cause: unknown): string {
  if (cause && typeof cause === 'object') {
    const value = cause as { reason?: string; message?: string }
    return value.reason || value.message || t('growth.common.requestFailed')
  }
  return t('growth.common.requestFailed')
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
      <div class="flex flex-col gap-3 xl:flex-row xl:items-center">
        <input v-model="searchInput" class="input min-w-0 flex-1" :placeholder="t('growth.admin.userPoints.searchPlaceholder')" @keyup.enter="applyFilters">
        <input v-model="minInput" class="input w-full xl:w-32" inputmode="numeric" :placeholder="'≥ ' + t('growth.common.points')">
        <input v-model="maxInput" class="input w-full xl:w-32" inputmode="numeric" :placeholder="'≤ ' + t('growth.common.points')">
        <select v-model="sort" class="input w-full xl:w-48">
          <option value="balance_desc">{{ t('growth.admin.userPoints.balanceDesc') }}</option>
          <option value="balance_asc">{{ t('growth.admin.userPoints.balanceAsc') }}</option>
          <option value="recent">{{ t('growth.admin.userPoints.recentDesc') }}</option>
        </select>
        <button class="btn btn-secondary" @click="applyFilters">{{ t('growth.common.search') }}</button>
        <button class="btn btn-secondary" @click="resetFilters">{{ t('growth.common.reset') }}</button>
        <button class="btn btn-secondary" @click="exportCsv">{{ t('growth.admin.userPoints.export') }}</button>
      </div>
    </div>

    <div v-if="loading && !items.length" class="card p-12 text-center"><LoadingSpinner /></div>
    <div v-else-if="!sortedItems.length" class="card">
      <EmptyState :title="t('growth.admin.userPoints.empty')" :description="t('growth.admin.userPoints.emptyHint')" />
    </div>
    <div v-else class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-[920px] w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-3">{{ t('growth.admin.userPoints.user') }}</th>
              <th class="px-4 py-3 text-right">{{ t('growth.admin.userPoints.current') }}</th>
              <th class="px-4 py-3 text-right">{{ t('growth.admin.userPoints.earned') }}</th>
              <th class="px-4 py-3 text-right">{{ t('growth.admin.userPoints.spent') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.userPoints.latest') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.userPoints.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="user in sortedItems" :key="user.user_id" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
              <td class="px-4 py-3">
                <div class="flex items-center gap-3">
                  <div class="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-primary-100 text-sm font-semibold text-primary-700 dark:bg-primary-500/20 dark:text-primary-300">{{ userName(user).slice(0, 1).toUpperCase() }}</div>
                  <div class="min-w-0">
                    <div class="truncate font-medium text-gray-900 dark:text-white">{{ userName(user) }}</div>
                    <div class="truncate text-xs text-gray-500 dark:text-dark-400">{{ user.email }}</div>
                  </div>
                </div>
              </td>
              <td class="px-4 py-3 text-right"><PointsAmount :value="user.available_points" unit="PT" /></td>
              <td class="px-4 py-3 text-right tabular-nums">{{ number(user.total_earned) }}</td>
              <td class="px-4 py-3 text-right tabular-nums">{{ number(user.total_spent) }}</td>
              <td class="px-4 py-3 text-xs text-gray-500 dark:text-dark-400">{{ dateTime(user.latest_change_at) }}</td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1">
                  <button class="btn btn-sm btn-secondary" @click="openUser(user)">{{ t('growth.admin.userPoints.view') }}</button>
                  <button v-if="canAdjust" class="btn btn-sm btn-secondary" @click="selected = user; openAdjust()">{{ t('growth.admin.userPoints.adjust') }}</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <BaseDialog :show="drawerOpen" :title="t('growth.admin.userPoints.drawerTitle', { user: selected ? userName(selected) : '' })" width="wide" @close="drawerOpen = false">
      <div v-if="selected" class="space-y-5">
        <div class="grid grid-cols-2 gap-3 rounded-xl bg-gray-50 p-4 dark:bg-dark-800 sm:grid-cols-4">
          <div><div class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.userPoints.current') }}</div><div class="mt-1 text-lg font-bold">{{ number(selected.available_points) }}</div></div>
          <div><div class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.userPoints.earned') }}</div><div class="mt-1 text-lg font-bold">{{ number(selected.total_earned) }}</div></div>
          <div><div class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.userPoints.spent') }}</div><div class="mt-1 text-lg font-bold">{{ number(selected.total_spent) }}</div></div>
          <div class="flex items-end"><button v-if="canAdjust" class="btn btn-sm btn-primary" @click="openAdjust">{{ t('growth.admin.userPoints.adjust') }}</button></div>
        </div>

        <section>
          <div class="mb-2 flex items-center justify-between">
            <h4 class="font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.userPoints.ledger') }}</h4>
            <span class="text-xs text-gray-500 dark:text-dark-400">{{ detailLoading ? t('growth.common.loading') : ledger.length }}</span>
          </div>
          <EmptyState v-if="!ledger.length && !detailLoading" :title="t('growth.admin.userPoints.noLedger')" />
          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="entry in ledger" :key="entry.entry_id" class="grid gap-2 py-3 text-sm sm:grid-cols-[7rem_1fr_auto]">
              <span class="text-xs text-gray-500 dark:text-dark-400">{{ dateTime(entry.created_at) }}</span>
              <div><div class="font-medium text-gray-800 dark:text-dark-100">{{ entry.title }}</div><div class="text-xs text-gray-500 dark:text-dark-400">{{ entry.biz_type }}#{{ entry.transfer_id }}</div></div>
              <PointsAmount :value="entry.points" :direction="entry.direction" signed />
            </div>
          </div>
          <button v-if="ledgerHasMore" class="btn btn-sm btn-secondary mt-2" @click="loadUserDetails(false)">{{ t('growth.user.ledger.loadMore') }}</button>
        </section>

        <section>
          <h4 class="mb-2 font-semibold text-gray-900 dark:text-white">{{ t('growth.admin.userPoints.exchanges') }}</h4>
          <EmptyState v-if="!exchanges.length" :title="t('growth.admin.userPoints.noExchanges')" />
          <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="exchange in exchanges" :key="exchange.order_id" class="flex flex-wrap items-center justify-between gap-2 py-3 text-sm">
              <div><div class="font-medium text-gray-800 dark:text-dark-100">{{ exchange.product_title }}</div><div class="text-xs text-gray-500 dark:text-dark-400">{{ exchange.order_no }} · v{{ exchange.product_version }}</div></div>
              <div class="flex items-center gap-3"><PointsAmount :value="exchange.points_cost" direction="OUT" /><GrowthStatusBadge :status="exchange.status" /></div>
            </div>
          </div>
        </section>
      </div>
    </BaseDialog>

    <BaseDialog :show="adjustOpen" :title="t('growth.admin.userPoints.adjustTitle')" width="normal" @close="adjustOpen = false">
      <form class="space-y-4" @submit.prevent="submitAdjust">
        <GrowthNotice v-if="adjustError" tone="error">{{ adjustError }}</GrowthNotice>
        <div class="grid grid-cols-2 gap-2 rounded-xl bg-gray-100 p-1 dark:bg-dark-800">
          <button type="button" class="rounded-lg px-3 py-2 text-sm font-medium" :class="adjustDirection === 'INCREASE' ? 'bg-white text-teal-700 shadow-sm dark:bg-dark-700 dark:text-teal-300' : 'text-gray-500'" @click="adjustDirection = 'INCREASE'; adjustConfirmed = false">{{ t('growth.admin.userPoints.increase') }}</button>
          <button type="button" class="rounded-lg px-3 py-2 text-sm font-medium" :class="adjustDirection === 'DECREASE' ? 'bg-white text-rose-700 shadow-sm dark:bg-dark-700 dark:text-rose-300' : 'text-gray-500'" @click="adjustDirection = 'DECREASE'; adjustConfirmed = false">{{ t('growth.admin.userPoints.decrease') }}</button>
        </div>
        <label class="block">
          <span class="input-label">{{ t('growth.common.amount') }}</span>
          <input v-model="adjustPoints" class="input w-full" inputmode="numeric" @input="adjustConfirmed = false">
        </label>
        <label class="block">
          <span class="input-label">{{ t('growth.common.reason') }}</span>
          <textarea v-model="adjustReason" class="input min-h-24 w-full" :placeholder="t('growth.common.reasonPlaceholder')" @input="adjustConfirmed = false" />
        </label>
        <div class="grid grid-cols-2 gap-3 rounded-xl border border-gray-200 p-3 text-sm dark:border-dark-700">
          <div><div class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.userPoints.before') }}</div><div class="mt-1 text-lg font-semibold">{{ number(selected?.available_points ?? 0) }}</div></div>
          <div><div class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.userPoints.after') }}</div><div class="mt-1 text-lg font-semibold">{{ number(projectedBalance) }}</div></div>
        </div>
        <GrowthNotice v-if="adjustConfirmed" tone="error">{{ t('growth.admin.userPoints.warning') }}</GrowthNotice>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="adjustOpen = false">{{ t('growth.common.cancel') }}</button>
          <button type="submit" class="btn" :class="adjustDirection === 'DECREASE' ? 'btn-danger' : 'btn-primary'" :disabled="adjustDisabled || adjusting">
            <LoadingSpinner v-if="adjusting" size="sm" color="white" />
            <span v-else>{{ t('growth.common.confirm') }}</span>
          </button>
        </div>
      </form>
    </BaseDialog>
  </div>
</template>
