<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { growthAdminAPI, type GrowthProduct, type ProductVersion } from '@/api/growth'
import BaseDialog from '@/components/common/BaseDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import LoadingSpinner from '@/components/common/LoadingSpinner.vue'
import GrowthNotice from '@/components/growth/GrowthNotice.vue'
import GrowthStatusBadge from '@/components/growth/GrowthStatusBadge.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'
import { positiveInteger, reasonIsValid } from '@/components/growth/growthRules'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

type ViewMode = 'table' | 'cards'
type ActionMode = 'publish' | 'offline' | 'stock' | ''

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const items = ref<GrowthProduct[]>([])
const error = ref('')
const loading = ref(false)
const search = ref('')
const typeFilter = ref('')
const statusFilter = ref('')
const view = ref<ViewMode>('table')
const selectedProduct = ref<GrowthProduct>()
const editorOpen = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const editorError = ref('')
const saving = ref(false)
const actionOpen = ref(false)
const actionMode = ref<ActionMode>('')
const actionReason = ref('')
const actionDelta = ref('')
const actionConfirmed = ref(false)
const actionError = ref('')
const versionsOpen = ref(false)
const versionsLoading = ref(false)
const versions = ref<ProductVersion[]>([])
const form = reactive({
  code: '',
  title: '',
  productType: 'TOKEN_QUOTA',
  icon: 'gift',
  fulfillmentMode: 'AUTOMATIC',
  stockMode: 'LIMITED',
  initialStock: '100',
  pointsCost: '50',
  benefitSummary: '',
  tokenAmount: '1000000',
  perUserLimit: '3',
  sortOrder: 0,
})

const canManage = computed(() => authStore.can('growth.manage'))
const pointsCost = computed(() => positiveInteger(form.pointsCost))
const tokenAmount = computed(() => Number(form.tokenAmount) || 0)
const editorValid = computed(() => {
  if (!form.title.trim() || !pointsCost.value || !form.benefitSummary.trim()) return false
  if (editorMode.value === 'create' && !form.code.trim()) return false
  if (form.productType === 'TOKEN_QUOTA' && (tokenAmount.value <= 0 || form.fulfillmentMode !== 'AUTOMATIC')) return false
  return true
})
const actionTitle = computed(() => {
  if (actionMode.value === 'publish') return t('growth.admin.product.publishTitle')
  if (actionMode.value === 'offline') return t('growth.admin.product.offlineTitle')
  return t('growth.admin.product.stockTitle')
})
const actionDisabled = computed(() => {
  if (actionMode.value === 'offline') return !reasonIsValid(actionReason.value)
  if (actionMode.value === 'stock') return !Number.isSafeInteger(Number(actionDelta.value)) || Number(actionDelta.value) === 0 || !reasonIsValid(actionReason.value)
  return false
})

const number = (value: number) => value.toLocaleString()
const dateTime = (value?: string) => value ? new Date(value).toLocaleString() : '-'

async function load() {
  loading.value = true
  error.value = ''
  try {
    items.value = (await growthAdminAPI.products({
      search: search.value,
      status: statusFilter.value,
      product_type: typeFilter.value,
      page_size: 100,
    })).data.items
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
    productType: 'TOKEN_QUOTA',
    icon: 'gift',
    fulfillmentMode: 'AUTOMATIC',
    stockMode: 'LIMITED',
    initialStock: '100',
    pointsCost: '50',
    benefitSummary: '',
    tokenAmount: '1000000',
    perUserLimit: '3',
    sortOrder: 0,
  })
  editorError.value = ''
}

function createProduct() {
  selectedProduct.value = undefined
  resetForm()
  editorMode.value = 'create'
  editorOpen.value = true
}

function editProduct(product: GrowthProduct) {
  selectedProduct.value = product
  Object.assign(form, {
    code: product.product_code,
    title: product.title,
    productType: product.product_type,
    icon: product.icon,
    fulfillmentMode: product.fulfillment_mode,
    stockMode: product.stock_mode,
    initialStock: String(product.stock),
    pointsCost: String(product.points_cost),
    benefitSummary: product.benefit_summary,
    tokenAmount: String(product.token_amount),
    perUserLimit: String(product.per_user_limit),
    sortOrder: product.sort_order,
  })
  editorError.value = ''
  editorMode.value = 'edit'
  editorOpen.value = true
}

async function saveProduct() {
  if (!editorValid.value || saving.value) return
  saving.value = true
  editorError.value = ''
  const version = {
    points_cost: pointsCost.value,
    benefit_summary: form.benefitSummary.trim(),
    token_amount: form.productType === 'TOKEN_QUOTA' ? tokenAmount.value : 0,
    per_user_limit: Math.max(0, Number(form.perUserLimit) || 0),
  }
  try {
    if (editorMode.value === 'create') {
      const result = await growthAdminAPI.createProduct({
        product_code: form.code.trim(),
        title: form.title.trim(),
        product_type: form.productType,
        icon: form.icon.trim(),
        fulfillment_mode: form.fulfillmentMode,
        sort_order: Number(form.sortOrder) || 0,
        stock_mode: form.stockMode,
        initial_stock: form.stockMode === 'UNLIMITED' ? 0 : Math.max(0, Number(form.initialStock) || 0),
        version,
        request_id: crypto.randomUUID(),
      })
      items.value = [result.data, ...items.value]
    } else if (selectedProduct.value) {
      const result = await growthAdminAPI.updateProduct(selectedProduct.value.product_id, {
        title: form.title.trim(),
        icon: form.icon.trim(),
        fulfillment_mode: form.fulfillmentMode,
        sort_order: Number(form.sortOrder) || 0,
        version,
        request_id: crypto.randomUUID(),
      })
      replaceProduct(result.data)
    }
    appStore.showToast('success', t('growth.common.operationSucceeded'))
    editorOpen.value = false
  } catch (cause) {
    editorError.value = errorMessage(cause)
  } finally {
    saving.value = false
  }
}

function replaceProduct(product: GrowthProduct) {
  const index = items.value.findIndex(item => item.product_id === product.product_id)
  if (index >= 0) items.value.splice(index, 1, product)
  else items.value.unshift(product)
  if (selectedProduct.value?.product_id === product.product_id) selectedProduct.value = product
}

function openAction(product: GrowthProduct, mode: Exclude<ActionMode, ''>) {
  selectedProduct.value = product
  actionMode.value = mode
  actionReason.value = ''
  actionDelta.value = ''
  actionConfirmed.value = false
  actionError.value = ''
  actionOpen.value = true
}

async function publishProduct() {
  if (!selectedProduct.value) return
  if (!actionConfirmed.value) {
    actionConfirmed.value = true
    return
  }
  try {
    const result = await growthAdminAPI.publishProduct(selectedProduct.value.product_id)
    replaceProduct(result.data)
    actionOpen.value = false
    appStore.showToast('success', t('growth.common.operationSucceeded'))
  } catch (cause) {
    const value = cause as { code?: string }
    actionError.value = value?.code === 'PRODUCT_PUBLISH_INVALID'
      ? t('growth.admin.product.invalidToken')
      : errorMessage(cause)
  }
}

async function submitAction() {
  if (!selectedProduct.value || actionDisabled.value) return
  if (actionMode.value === 'publish') {
    await publishProduct()
    return
  }
  try {
    let result
    if (actionMode.value === 'offline') {
      if (!actionConfirmed.value) {
        actionConfirmed.value = true
        return
      }
      result = await growthAdminAPI.offlineProduct(selectedProduct.value.product_id, {
        reason: actionReason.value.trim(),
        request_id: crypto.randomUUID(),
      })
    } else {
      result = await growthAdminAPI.adjustStock(selectedProduct.value.product_id, {
        delta: Number(actionDelta.value),
        reason: actionReason.value.trim(),
        request_id: crypto.randomUUID(),
      })
    }
    replaceProduct(result.data)
    actionOpen.value = false
    appStore.showToast('success', t('growth.common.operationSucceeded'))
  } catch (cause) {
    actionError.value = errorMessage(cause)
  }
}

async function openVersions(product: GrowthProduct) {
  selectedProduct.value = product
  versions.value = []
  versionsOpen.value = true
  versionsLoading.value = true
  try {
    versions.value = (await growthAdminAPI.productVersions(product.product_id)).data.items
  } catch (cause) {
    error.value = errorMessage(cause)
  } finally {
    versionsLoading.value = false
  }
}

function stockLabel(product: GrowthProduct): string {
  if (product.stock < 0) return t('growth.admin.product.unlimited')
  if (product.stock === 0) return t('growth.admin.product.soldOut')
  if (product.stock <= 10) return t('growth.admin.product.lowStock', { count: product.stock })
  return number(product.stock)
}

function stockClass(product: GrowthProduct): string {
  if (product.stock === 0) return 'text-rose-600 dark:text-rose-300'
  if (product.stock > 0 && product.stock <= 10) return 'text-amber-600 dark:text-amber-300'
  return 'text-gray-700 dark:text-dark-200'
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
        <input v-model="search" class="input min-w-0 flex-1" :placeholder="t('growth.admin.product.searchPlaceholder')" @keyup.enter="load">
        <select v-model="typeFilter" class="input w-full xl:w-44" @change="load">
          <option value="">{{ t('growth.admin.product.filterType') }}</option>
          <option value="TOKEN_QUOTA">TOKEN_QUOTA</option>
          <option value="SUBSCRIPTION">SUBSCRIPTION</option>
          <option value="PHYSICAL">PHYSICAL</option>
        </select>
        <select v-model="statusFilter" class="input w-full xl:w-36" @change="load">
          <option value="">{{ t('growth.admin.product.filterStatus') }}</option>
          <option value="DRAFT">DRAFT</option>
          <option value="ACTIVE">ACTIVE</option>
          <option value="OFFLINE">OFFLINE</option>
        </select>
        <button class="btn btn-secondary" @click="load">{{ t('growth.common.search') }}</button>
        <div class="inline-flex self-start rounded-lg border border-gray-200 p-0.5 dark:border-dark-700">
          <button class="rounded-md px-3 py-1.5 text-sm" :class="view === 'table' ? 'bg-gray-100 font-medium dark:bg-dark-700' : ''" @click="view = 'table'">{{ t('growth.admin.product.tableView') }}</button>
          <button class="rounded-md px-3 py-1.5 text-sm" :class="view === 'cards' ? 'bg-gray-100 font-medium dark:bg-dark-700' : ''" @click="view = 'cards'">{{ t('growth.admin.product.cardView') }}</button>
        </div>
        <button v-if="canManage" class="btn btn-primary" @click="createProduct">{{ t('growth.admin.product.create') }}</button>
      </div>
    </div>

    <div v-if="loading && !items.length" class="card p-12 text-center"><LoadingSpinner /></div>
    <div v-else-if="!items.length" class="card">
      <EmptyState
        :title="t('growth.admin.product.noProducts')"
        :description="t('growth.admin.product.noProductsHint')"
        :action-text="canManage ? t('growth.admin.product.create') : undefined"
        @action="createProduct"
      />
    </div>

    <div v-else-if="view === 'table'" class="card overflow-hidden">
      <div class="overflow-x-auto">
        <table class="min-w-[1120px] w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-3">{{ t('growth.admin.product.product') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.product.type') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.product.pointsAndVersion') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.product.inventory') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.product.limit') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.product.fulfillment') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.product.status') }}</th>
              <th class="px-4 py-3">{{ t('growth.admin.product.actions') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="product in items" :key="product.product_id" class="hover:bg-gray-50 dark:hover:bg-dark-800/60">
              <td class="px-4 py-3">
                <button class="text-left font-medium text-gray-900 hover:text-primary-600 dark:text-white" @click="openVersions(product)">{{ product.title }}</button>
                <div class="mt-0.5 font-mono text-xs text-gray-500 dark:text-dark-400">{{ product.product_code }}</div>
              </td>
              <td class="px-4 py-3"><span class="rounded-md bg-gray-100 px-2 py-1 text-xs dark:bg-dark-700">{{ product.product_type }}</span></td>
              <td class="px-4 py-3">
                <PointsAmount :value="product.points_cost" unit="PT" />
                <div class="mt-1 flex flex-wrap gap-1 text-xs text-gray-500 dark:text-dark-400">
                  <span>v{{ product.current_version }}</span>
                  <span v-if="product.pending_version" class="rounded-full bg-amber-50 px-2 py-0.5 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300">{{ t('growth.admin.product.pendingVersion', { version: product.pending_version }) }}</span>
                </div>
              </td>
              <td class="px-4 py-3 tabular-nums" :class="stockClass(product)">{{ stockLabel(product) }}</td>
              <td class="px-4 py-3">{{ product.per_user_limit || t('growth.admin.product.unlimited') }}</td>
              <td class="px-4 py-3"><GrowthStatusBadge :status="product.fulfillment_mode" /></td>
              <td class="px-4 py-3"><GrowthStatusBadge :status="product.status" /></td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1">
                  <button class="btn btn-sm btn-secondary" @click="openVersions(product)">{{ t('growth.admin.product.versions') }}</button>
                  <template v-if="canManage">
                    <button class="btn btn-sm btn-secondary" @click="editProduct(product)">{{ t('growth.admin.product.edit') }}</button>
                    <button class="btn btn-sm btn-secondary" @click="openAction(product, 'stock')">{{ t('growth.admin.product.stock') }}</button>
                    <button v-if="product.status === 'ACTIVE'" class="btn btn-sm btn-danger" @click="openAction(product, 'offline')">{{ t('growth.admin.product.offline') }}</button>
                    <button v-else class="btn btn-sm btn-primary" @click="openAction(product, 'publish')">{{ product.pending_version ? t('growth.admin.product.publishVersion') : t('growth.admin.product.publish') }}</button>
                  </template>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <article v-for="product in items" :key="product.product_id" class="card flex flex-col p-5">
        <div class="flex items-start justify-between gap-3">
          <div class="flex min-w-0 items-center gap-3">
            <div class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary-50 text-xl text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">◇</div>
            <div class="min-w-0">
              <h3 class="truncate font-semibold text-gray-900 dark:text-white">{{ product.title }}</h3>
              <p class="truncate font-mono text-xs text-gray-500 dark:text-dark-400">{{ product.product_code }}</p>
            </div>
          </div>
          <GrowthStatusBadge :status="product.status" />
        </div>
        <p class="mt-3 flex-1 text-sm text-gray-600 dark:text-dark-300">{{ product.benefit_summary }}</p>
        <div class="mt-4 flex items-center justify-between">
          <PointsAmount :value="product.points_cost" unit="PT" />
          <span class="text-sm tabular-nums" :class="stockClass(product)">{{ stockLabel(product) }}</span>
        </div>
        <div class="mt-4 flex flex-wrap gap-1 border-t border-gray-100 pt-3 dark:border-dark-700">
          <button class="btn btn-sm btn-secondary" @click="openVersions(product)">{{ t('growth.admin.product.versions') }}</button>
          <template v-if="canManage">
            <button class="btn btn-sm btn-secondary" @click="editProduct(product)">{{ t('growth.admin.product.edit') }}</button>
            <button class="btn btn-sm btn-secondary" @click="openAction(product, 'stock')">{{ t('growth.admin.product.stock') }}</button>
            <button v-if="product.status === 'ACTIVE'" class="btn btn-sm btn-danger" @click="openAction(product, 'offline')">{{ t('growth.admin.product.offline') }}</button>
            <button v-else class="btn btn-sm btn-primary" @click="openAction(product, 'publish')">{{ t('growth.admin.product.publish') }}</button>
          </template>
        </div>
      </article>
    </div>

    <BaseDialog :show="editorOpen" :title="editorMode === 'create' ? t('growth.admin.product.createTitle') : t('growth.admin.product.editTitle')" width="wide" @close="editorOpen = false">
      <form class="space-y-4" @submit.prevent="saveProduct">
        <GrowthNotice v-if="editorError" tone="error">{{ editorError }}</GrowthNotice>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.code') }}</span>
            <input v-model="form.code" class="input w-full" :disabled="editorMode === 'edit'">
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.title') }}</span>
            <input v-model="form.title" class="input w-full">
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.productType') }}</span>
            <select v-model="form.productType" class="input w-full" :disabled="editorMode === 'edit'">
              <option value="TOKEN_QUOTA">TOKEN_QUOTA</option>
              <option value="SUBSCRIPTION">SUBSCRIPTION</option>
              <option value="PHYSICAL">PHYSICAL</option>
            </select>
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.fulfillmentMode') }}</span>
            <select v-model="form.fulfillmentMode" class="input w-full">
              <option value="AUTOMATIC">AUTOMATIC</option>
              <option value="MANUAL">MANUAL</option>
            </select>
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.pointsCost') }}</span>
            <input v-model="form.pointsCost" class="input w-full" inputmode="numeric">
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.perUserLimit') }}</span>
            <input v-model="form.perUserLimit" class="input w-full" inputmode="numeric">
          </label>
          <label class="block sm:col-span-2">
            <span class="input-label">{{ t('growth.admin.product.benefitSummary') }}</span>
            <textarea v-model="form.benefitSummary" class="input min-h-20 w-full" />
          </label>
          <label v-if="form.productType === 'TOKEN_QUOTA'" class="block">
            <span class="input-label">{{ t('growth.admin.product.tokenAmount') }}</span>
            <input v-model="form.tokenAmount" class="input w-full" inputmode="numeric">
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.icon') }}</span>
            <input v-model="form.icon" class="input w-full">
          </label>
          <template v-if="editorMode === 'create'">
            <label class="block">
              <span class="input-label">{{ t('growth.admin.product.stockMode') }}</span>
              <select v-model="form.stockMode" class="input w-full">
                <option value="LIMITED">LIMITED</option>
                <option value="UNLIMITED">UNLIMITED</option>
              </select>
            </label>
            <label v-if="form.stockMode === 'LIMITED'" class="block">
              <span class="input-label">{{ t('growth.admin.product.initialStock') }}</span>
              <input v-model="form.initialStock" class="input w-full" inputmode="numeric">
            </label>
          </template>
        </div>
        <GrowthNotice v-if="form.productType === 'TOKEN_QUOTA' && (tokenAmount <= 0 || form.fulfillmentMode !== 'AUTOMATIC')" tone="warning">{{ t('growth.admin.product.invalidToken') }}</GrowthNotice>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="editorOpen = false">{{ t('growth.common.cancel') }}</button>
          <button type="submit" class="btn btn-primary" :disabled="!editorValid || saving">
            <LoadingSpinner v-if="saving" size="sm" color="white" />
            <span v-else>{{ t('growth.admin.product.saveDraft') }}</span>
          </button>
        </div>
      </form>
    </BaseDialog>

    <BaseDialog :show="actionOpen" :title="actionTitle" width="normal" @close="actionOpen = false">
      <form class="space-y-4" @submit.prevent="submitAction">
        <GrowthNotice v-if="actionError" tone="error">{{ actionError }}</GrowthNotice>
        <div v-if="selectedProduct" class="rounded-xl bg-gray-50 px-4 py-3 dark:bg-dark-800">
          <div class="font-medium text-gray-900 dark:text-white">{{ selectedProduct.title }}</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ selectedProduct.product_code }} · v{{ selectedProduct.current_version }}</div>
        </div>
        <template v-if="actionMode === 'publish'">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('growth.admin.product.publishHint') }}</p>
          <GrowthNotice v-if="actionConfirmed" tone="warning">{{ t('growth.admin.budget.auditWarning') }}</GrowthNotice>
        </template>
        <template v-else-if="actionMode === 'offline'">
          <p class="text-sm text-gray-600 dark:text-dark-300">{{ t('growth.admin.product.offlineHint') }}</p>
          <label class="block">
            <span class="input-label">{{ t('growth.common.reason') }}</span>
            <textarea v-model="actionReason" class="input min-h-24 w-full" :placeholder="t('growth.common.reasonPlaceholder')" />
          </label>
          <GrowthNotice v-if="actionConfirmed" tone="error">{{ t('growth.admin.budget.auditWarning') }}</GrowthNotice>
        </template>
        <template v-else>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.stockDelta') }}</span>
            <input v-model="actionDelta" class="input w-full" inputmode="numeric">
            <span class="mt-1 block text-xs text-gray-500 dark:text-dark-400">{{ t('growth.admin.product.stockDeltaHint') }}</span>
          </label>
          <label class="block">
            <span class="input-label">{{ t('growth.admin.product.stockReason') }}</span>
            <textarea v-model="actionReason" class="input min-h-24 w-full" />
          </label>
        </template>
        <div class="flex justify-end gap-2">
          <button type="button" class="btn btn-secondary" @click="actionOpen = false">{{ t('growth.common.cancel') }}</button>
          <button type="submit" class="btn" :class="actionMode === 'offline' ? 'btn-danger' : 'btn-primary'" :disabled="actionDisabled">{{ t('growth.common.confirm') }}</button>
        </div>
      </form>
    </BaseDialog>

    <BaseDialog :show="versionsOpen" :title="t('growth.admin.product.versionHistoryTitle', { title: selectedProduct?.title ?? '' })" width="wide" @close="versionsOpen = false">
      <div v-if="versionsLoading" class="py-12 text-center"><LoadingSpinner /></div>
      <EmptyState v-else-if="!versions.length" :title="t('growth.common.noData')" />
      <div v-else class="space-y-3">
        <div v-for="version in versions" :key="version.id" class="rounded-xl border border-gray-200 p-4 dark:border-dark-700">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <div class="flex items-center gap-2">
              <span class="font-semibold text-gray-900 dark:text-white">v{{ version.version }}</span>
              <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="version.published_at
                ? 'bg-teal-50 text-teal-700 dark:bg-teal-500/10 dark:text-teal-300'
                : 'bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-300'">
                {{ version.published_at ? t('growth.admin.task.published') : t('growth.admin.product.draftVersion') }}
              </span>
            </div>
            <PointsAmount :value="version.points_cost" unit="PT" />
          </div>
          <p class="mt-2 text-sm text-gray-700 dark:text-dark-200">{{ version.benefit_summary }}</p>
          <div class="mt-2 grid gap-1 text-xs text-gray-500 dark:text-dark-400 sm:grid-cols-2">
            <span v-if="selectedProduct?.product_type === 'TOKEN_QUOTA'">{{ t('growth.admin.product.tokenAmount') }}: {{ number(version.token_amount) }}</span>
            <span>{{ t('growth.admin.product.limit') }}: {{ version.per_user_limit || t('growth.admin.product.unlimited') }}</span>
            <span class="sm:col-span-2">{{ dateTime(version.effective_from) }} — {{ dateTime(version.effective_to) }}</span>
          </div>
        </div>
      </div>
    </BaseDialog>
  </div>
</template>
