<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import {
  growthUserAPI,
  type LedgerItem,
  type UserExchange,
  type UserOverview,
  type UserProduct,
  type UserTask,
} from '@/api/growth'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import GrowthNotice from '@/components/growth/GrowthNotice.vue'
import GrowthStatusBadge from '@/components/growth/GrowthStatusBadge.vue'
import GrowthTabs from '@/components/growth/GrowthTabs.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'
import PointsHero from '@/components/growth/PointsHero.vue'
import { appendUniqueByEntryId } from '@/components/growth/growthRules'
import OverviewTab from './user/OverviewTab.vue'
import StoreTab from './user/StoreTab.vue'
import LedgerTab from './user/LedgerTab.vue'
import { pollExchangeOrder } from './pollExchange'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const tabKeys = ['overview', 'store', 'ledger'] as const
type TabKey = typeof tabKeys[number]
const fallback: TabKey = 'overview'
const tabs = computed(() => tabKeys.map(key => ({ key, label: t(`growth.user.tabs.${key}`) })))
const active = computed<TabKey>(() => tabKeys.includes(String(route.query.tab) as TabKey) ? String(route.query.tab) as TabKey : fallback)
const canRead = computed(() => authStore.can('growth.self.read'))

const overview = ref<UserOverview>()
const tasks = ref<UserTask[]>([])
const products = ref<UserProduct[]>([])
const ledger = ref<LedgerItem[]>([])
const exchanges = ref<UserExchange[]>([])
const ledgerType = ref('')
const ledgerCursor = ref('')
const hasMore = ref(false)
const loading = ref(true)
const loadingMore = ref(false)
const error = ref('')
const selected = ref<UserProduct>()
const detailOpen = ref(false)
const confirmOpen = ref(false)
const exchanging = ref(false)
const exchangeError = ref('')
const exchangeNotice = ref('')
const pendingStatus = ref('')
const idempotencyKey = ref('')

function selectTab(key: string) {
  if (!tabKeys.includes(key as TabKey)) key = fallback
  void router.replace({ query: { ...route.query, tab: key } })
}

watch(
  () => route.query.tab,
  value => {
    if (String(value ?? '') !== active.value) selectTab(String(value ?? fallback))
  },
  { immediate: true }
)

async function loadInitial() {
  if (!canRead.value) {
    loading.value = false
    return
  }
  loading.value = true
  error.value = ''
  try {
    const [overviewResult, taskResult, productResult, ledgerResult] = await Promise.all([
      growthUserAPI.overview(),
      growthUserAPI.tasks(),
      growthUserAPI.products(),
      growthUserAPI.ledger({ limit: 10 }),
    ])
    overview.value = overviewResult.data
    tasks.value = taskResult.data.items
    products.value = productResult.data.items
    ledger.value = ledgerResult.data.items
    ledgerCursor.value = ledgerResult.data.next_cursor
    hasMore.value = ledgerResult.data.has_more
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    loading.value = false
  }
}

async function loadExchanges() {
  try {
    exchanges.value = (await growthUserAPI.exchanges({ page_size: 20 })).data.items
  } catch (cause) {
    error.value = errorMessage(cause)
  }
}

async function openProduct(product: UserProduct) {
  exchangeError.value = ''
  exchangeNotice.value = ''
  pendingStatus.value = ''
  idempotencyKey.value = crypto.randomUUID()
  selected.value = product
  detailOpen.value = true
  try {
    selected.value = (await growthUserAPI.product(product.product_id)).data
  } catch {
    selected.value = product
  }
}

function requestExchange() {
  if (!selected.value?.can_exchange) return
  exchangeError.value = ''
  confirmOpen.value = true
}

async function confirmExchange() {
  if (!selected.value || exchanging.value) return
  exchanging.value = true
  exchangeError.value = ''
  try {
    const result = await growthUserAPI.exchange(selected.value.product_id, idempotencyKey.value)
    const created = result.data
    if (overview.value) {
      overview.value = {
        ...overview.value,
        available_points: created.current_balance,
        total_spent: overview.value.total_spent + created.points_cost,
      }
    }
    products.value = products.value.map(product => product.points_cost > created.current_balance
      ? { ...product, can_exchange: false, reason: 'INSUFFICIENT_POINTS' }
      : product)
    const ledgerResult = await growthUserAPI.ledger({ limit: 10, type: ledgerType.value })
    ledger.value = ledgerResult.data.items
    ledgerCursor.value = ledgerResult.data.next_cursor
    hasMore.value = ledgerResult.data.has_more
    confirmOpen.value = false
    detailOpen.value = false
    pendingStatus.value = created.status
    exchangeNotice.value = t('growth.user.store.exchangePending', { orderNo: created.order_no })
    await loadExchanges()
    const finalOrder = await pollExchangeOrder(created.order_id, async orderId => {
      const response = await growthUserAPI.order(orderId)
      pendingStatus.value = response.data.status
      return response.data
    })
    await loadExchanges()
    if (finalOrder?.status === 'SUCCEEDED') {
      appStore.showToast('success', t('growth.user.store.exchangeSuccess'))
    } else if (finalOrder?.status === 'FAILED') {
      appStore.showToast('error', t('growth.user.store.exchangeFailed'))
    } else if (finalOrder?.status === 'REFUNDED') {
      appStore.showToast('warning', t('growth.user.store.exchangeRefunded'))
    }
    idempotencyKey.value = ''
  } catch (cause) {
    exchangeError.value = errorMessage(cause)
    confirmOpen.value = false
    detailOpen.value = true
  } finally {
    exchanging.value = false
  }
}

async function changeLedgerType(value: string) {
  ledgerType.value = value
  ledgerCursor.value = ''
  loadingMore.value = true
  try {
    const result = await growthUserAPI.ledger({ limit: 10, type: value })
    ledger.value = result.data.items
    ledgerCursor.value = result.data.next_cursor
    hasMore.value = result.data.has_more
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    loadingMore.value = false
  }
}

async function loadMoreLedger() {
  if (!hasMore.value || loadingMore.value) return
  loadingMore.value = true
  try {
    const result = await growthUserAPI.ledger({
      before_entry_id: Number(ledgerCursor.value),
      limit: 10,
      type: ledgerType.value,
    })
    ledger.value = appendUniqueByEntryId(ledger.value, result.data.items)
    ledgerCursor.value = result.data.next_cursor
    hasMore.value = result.data.has_more
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    loadingMore.value = false
  }
}

function showAiworkGuide() {
  appStore.showToast('info', t('growth.user.checkin.guidance'), 5000)
}

function errorMessage(cause: unknown): string {
  if (cause && typeof cause === 'object') {
    const value = cause as { reason?: string; message?: string }
    return value.reason || value.message || t('growth.common.requestFailed')
  }
  return t('growth.common.requestFailed')
}

onMounted(async () => {
  await loadInitial()
  if (active.value === 'store' && canRead.value) await loadExchanges()
})
</script>

<template>
  <section class="content">
    <header class="page-header">
      <div>
        <h1>{{ t('growth.user.title') }}</h1>
        <p>{{ t('growth.user.description') }}</p>
      </div>
    </header>

    <div v-if="!canRead" class="card mx-auto mt-8 max-w-xl p-8 text-center">
      <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-rose-50 text-2xl text-rose-600 dark:bg-rose-500/10 dark:text-rose-300">403</div>
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('growth.common.permissionDenied') }}</h2>
    </div>

    <template v-else>
      <GrowthNotice v-if="error" tone="error">
        <div class="flex items-center justify-between gap-3">
          <span>{{ error }}</span>
          <button class="btn btn-sm btn-secondary" @click="loadInitial">{{ t('growth.common.retry') }}</button>
        </div>
      </GrowthNotice>
      <GrowthNotice v-if="exchangeNotice" tone="info">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <span>{{ exchangeNotice }}</span>
          <GrowthStatusBadge v-if="pendingStatus" :status="pendingStatus" />
        </div>
      </GrowthNotice>

      <div v-if="loading && !overview" class="space-y-4">
        <div class="h-56 animate-pulse rounded-2xl bg-gray-100 dark:bg-dark-800" />
        <div class="h-10 w-96 max-w-full animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800" />
      </div>

      <template v-else-if="overview">
        <PointsHero :overview="overview" @aiwork="showAiworkGuide" />
        <GrowthTabs class="mt-6" :tabs="tabs" :active="active" @select="selectTab" />
        <OverviewTab
          v-if="active === 'overview'"
          :overview="overview"
          :tasks="tasks"
          :products="products"
          :ledger="ledger"
          @tab="selectTab"
          @product="openProduct"
          @aiwork="showAiworkGuide"
        />
        <StoreTab
          v-else-if="active === 'store'"
          :products="products"
          :exchanges="exchanges"
          :balance="overview.available_points"
          @product="openProduct"
        />
        <LedgerTab
          v-else
          :items="ledger"
          :has-more="hasMore"
          :loading-more="loadingMore"
          @type-change="changeLedgerType"
          @load-more="loadMoreLedger"
          @overview="selectTab('overview')"
        />
      </template>
    </template>

    <BaseDialog :show="detailOpen" :title="selected?.name || t('growth.user.store.detailTitle')" width="wide" @close="detailOpen = false">
      <div v-if="selected" class="space-y-5">
        <GrowthNotice v-if="exchangeError" tone="error">{{ exchangeError }}</GrowthNotice>
        <div class="grid gap-5 sm:grid-cols-[10rem_1fr]">
          <div class="flex h-36 items-center justify-center rounded-2xl bg-gradient-to-br from-primary-50 to-teal-50 text-5xl text-primary-500 dark:from-primary-950/40 dark:to-teal-950/40 dark:text-primary-300">◇</div>
          <div>
            <div class="flex flex-wrap items-center justify-between gap-2">
              <h3 class="text-xl font-semibold text-gray-900 dark:text-white">{{ selected.name }}</h3>
              <PointsAmount :value="selected.points_cost" unit="PT" />
            </div>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">{{ selected.description }}</p>
            <div class="mt-3 flex flex-wrap gap-2 text-xs">
              <span class="rounded-full bg-gray-100 px-2 py-1 text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ selected.stock < 0 ? t('growth.user.store.unlimited') : t('growth.user.store.stock', { count: selected.stock }) }}</span>
              <span v-if="selected.per_user_limit > 0" class="rounded-full bg-gray-100 px-2 py-1 text-gray-600 dark:bg-dark-700 dark:text-dark-300">{{ t('growth.user.store.limit', { count: selected.per_user_limit }) }}</span>
            </div>
          </div>
        </div>
        <div class="rounded-xl border border-gray-200 p-4 text-sm dark:border-dark-700">
          <div class="text-xs font-medium text-gray-500 dark:text-dark-400">{{ t('growth.user.store.fulfillmentNote') }}</div>
          <p class="mt-1 text-gray-700 dark:text-dark-200">{{ selected.fulfillment_note || '-' }}</p>
          <pre v-if="selected.benefit_detail" class="mt-3 overflow-x-auto whitespace-pre-wrap rounded-lg bg-gray-50 p-3 text-xs text-gray-600 dark:bg-dark-800 dark:text-dark-300">{{ JSON.stringify(selected.benefit_detail, null, 2) }}</pre>
        </div>
        <div class="flex justify-end">
          <button class="btn btn-primary" :disabled="!selected.can_exchange || exchanging" @click="requestExchange">
            <LoadingSpinner v-if="exchanging" size="sm" color="white" />
            <span v-else>{{ t('growth.user.store.confirmExchange') }}</span>
          </button>
        </div>
      </div>
    </BaseDialog>

    <ConfirmDialog
      :show="confirmOpen"
      :title="t('growth.user.store.confirmTitle')"
      :message="selected ? t('growth.user.store.confirmMessage', { points: selected.points_cost, name: selected.name }) : ''"
      :confirm-text="t('growth.user.store.confirmExchange')"
      @confirm="confirmExchange"
      @cancel="confirmOpen = false"
    />
  </section>
</template>
