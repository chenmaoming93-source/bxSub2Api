<script setup lang="ts">
import UiEmptyState from './UiEmptyState.vue'
import UiTableSkeleton from './UiTableSkeleton.vue'

export interface UiTableColumn {
  key: string
  label: string
  sortable?: boolean
  class?: string
  formatter?: (value: unknown, row: Record<string, unknown>) => string
}

const props = withDefaults(defineProps<{
  columns: UiTableColumn[]
  rows?: Record<string, unknown>[]
  loading?: boolean
  rowKey?: string
  emptyTitle?: string
}>(), { rows: () => [], loading: false, rowKey: 'id', emptyTitle: '暂无数据' })
const emit = defineEmits<{ sort: [key: string]; rowClick: [row: Record<string, unknown>] }>()
function rowId(row: Record<string, unknown>, index: number) { return String(row[props.rowKey] ?? index) }
</script>

<template>
  <div class="ui-v2-table-wrapper">
    <UiTableSkeleton v-if="loading" :columns="columns.length" />
    <UiEmptyState v-else-if="!rows.length" :title="emptyTitle" />
    <table v-else class="ui-v2-table">
      <thead><tr><th v-for="column in columns" :key="column.key" :class="column.class" @click="column.sortable && emit('sort', column.key)">{{ column.label }}<span v-if="column.sortable" class="ui-v2-table__sort">↕</span></th></tr></thead>
      <tbody><tr v-for="(row, index) in rows" :key="rowId(row, index)" @click="emit('rowClick', row)">
        <td v-for="column in columns" :key="column.key" :class="column.class">
          <slot :name="`cell-${column.key}`" :row="row" :value="row[column.key]">{{ column.formatter ? column.formatter(row[column.key], row) : row[column.key] }}</slot>
        </td>
      </tr></tbody>
    </table>
  </div>
</template>
