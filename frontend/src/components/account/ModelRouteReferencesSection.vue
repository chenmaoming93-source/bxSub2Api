<template>
  <div class="rounded-lg border border-gray-200 p-4 dark:border-dark-600">
    <h3 class="mb-2 text-sm font-semibold text-gray-700 dark:text-gray-200">分组路由引用</h3>
    <div v-if="loading" class="text-sm text-gray-500">加载中...</div>
    <div v-else-if="references.length === 0" class="text-sm text-gray-500">未被模型路由引用</div>
    <div v-else class="flex min-w-0 max-w-full flex-wrap items-center gap-1">
      <span
        v-for="item in references"
        :key="`${item.group_id}-${item.route_alias}-${item.account_id}`"
        class="inline-flex min-w-0 max-w-full items-center rounded-md bg-indigo-100 px-1.5 py-1 text-[10px] font-medium leading-tight text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300"
        :title="`${item.group_name} / ${item.route_alias} / ${item.max_concurrency == null ? '不限制' : `最大并发 ${item.max_concurrency}`}`"
      >
        <span class="truncate whitespace-nowrap">
          <span class="font-semibold">{{ item.group_name }}</span>
          <span class="font-normal"> / {{ item.route_alias }}</span>
          <span class="ml-1 text-indigo-600/80 dark:text-indigo-300/80">
            · {{ item.max_concurrency == null ? '无限制（不参与分配）' : `${item.max_concurrency}/${item.account_concurrency}（${item.account_concurrency > 0 ? Math.round(item.max_concurrency * 100 / item.account_concurrency) : '不适用'}%）` }}
          </span>
        </span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { GroupModelRouteReference } from '@/api/admin/accounts'

defineProps<{
  references: GroupModelRouteReference[]
  loading: boolean
}>()
</script>
