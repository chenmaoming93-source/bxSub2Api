<script setup lang="ts">
import UiButton from '../primitives/UiButton.vue'

const props = withDefaults(defineProps<{ page?: number; total?: number; pageSize?: number; pageSizeOptions?: number[]; showPageSizeSelector?: boolean }>(), {
  page: 1, total: 0, pageSize: 20, pageSizeOptions: () => [20, 50, 100], showPageSizeSelector: true
})
const emit = defineEmits<{ 'update:page': [value: number]; 'update:pageSize': [value: number] }>()
const totalPages = () => Math.max(1, Math.ceil((props.total ?? 0) / (props.pageSize ?? 20)))
function go(value: number) { if (value >= 1 && value <= totalPages() && value !== props.page) emit('update:page', value) }
</script>

<template>
  <nav class="ui-v2-pagination" aria-label="分页">
    <span class="ui-v2-pagination__summary">共 {{ total }} 条 · 第 {{ page }} / {{ totalPages() }} 页</span>
    <div class="ui-v2-pagination__actions">
      <select v-if="showPageSizeSelector" class="ui-v2-pagination__size" :value="pageSize" @change="emit('update:pageSize', Number(($event.target as HTMLSelectElement).value))">
        <option v-for="size in pageSizeOptions" :key="size" :value="size">{{ size }} / 页</option>
      </select>
      <UiButton variant="secondary" size="sm" :disabled="page <= 1" @click="go(page - 1)">上一页</UiButton>
      <UiButton variant="secondary" size="sm" :disabled="page >= totalPages()" @click="go(page + 1)">下一页</UiButton>
    </div>
  </nav>
</template>
