<script setup lang="ts">
import { computed, ref, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import GrowthTabs from '@/components/growth/GrowthTabs.vue'
import BudgetTab from './admin/BudgetTab.vue'
import TasksTab from './admin/TasksTab.vue'
import ProductsTab from './admin/ProductsTab.vue'
import OrdersTab from './admin/OrdersTab.vue'
import UserPointsTab from './admin/UserPointsTab.vue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const tabKeys = ['budget', 'tasks', 'products', 'orders', 'users'] as const
type TabKey = typeof tabKeys[number]
const fallback: TabKey = 'budget'
const components: Record<TabKey, Component> = {
  budget: BudgetTab,
  tasks: TasksTab,
  products: ProductsTab,
  orders: OrdersTab,
  users: UserPointsTab,
}
const tabs = computed(() => tabKeys.map(key => ({ key, label: t(`growth.admin.tabs.${key}`) })))
const active = computed<TabKey>(() => tabKeys.includes(String(route.query.tab) as TabKey) ? String(route.query.tab) as TabKey : fallback)
const refreshKey = ref(0)
const component = computed(() => components[active.value])
const canRead = computed(() => authStore.can('growth.read'))

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
</script>

<template>
  <section class="content">
    <header class="page-header flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1>{{ t('growth.admin.title') }}</h1>
        <p>{{ t('growth.admin.description') }}</p>
      </div>
      <button v-if="canRead" type="button" class="btn btn-secondary self-start sm:self-auto" @click="refreshKey += 1">
        {{ t('growth.common.refresh') }}
      </button>
    </header>

    <div v-if="!canRead" class="card mx-auto mt-8 max-w-xl p-8 text-center">
      <div class="mx-auto mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-rose-50 text-2xl text-rose-600 dark:bg-rose-500/10 dark:text-rose-300">403</div>
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('growth.common.permissionDenied') }}</h2>
      <p class="mt-2 text-sm text-gray-500 dark:text-dark-400">{{ t('growth.common.permissionDenied') }}</p>
    </div>

    <template v-else>
      <GrowthTabs :tabs="tabs" :active="active" @select="selectTab" />
      <component :is="component" :key="`${active}-${refreshKey}`" />
    </template>
  </section>
</template>
