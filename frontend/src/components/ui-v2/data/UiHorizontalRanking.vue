<script setup lang="ts">
import { computed, ref } from 'vue'
import UiSearchInput from '@/components/ui-v2/forms/UiSearchInput.vue'

export interface UiRankingItem { id: string | number; label: string; value: number; displayValue?: string; percentage?: number }
const props = withDefaults(defineProps<{ items?: UiRankingItem[]; maxItems?: number; valueLabel?: string; searchPlaceholder?: string; showAction?: boolean }>(), { items: () => [], maxItems: 12, valueLabel: 'Token', searchPlaceholder: '搜索名称', showAction: true })
const emit = defineEmits<{ select: [item: UiRankingItem] }>()
const query = ref('')
const scope = ref<'top12' | 'all'>('top12')
const filteredItems = computed(() => {
  const keyword = query.value.trim().toLowerCase()
  const result = props.items.filter((item) => !keyword || item.label.toLowerCase().includes(keyword))
  return scope.value === 'top12' ? result.slice(0, props.maxItems) : result
})
function width(item: UiRankingItem, items: UiRankingItem[]) { const max = Math.max(...items.map(entry => entry.value), 1); return `${Math.max(3, (item.value / max) * 100)}%` }
</script>

<template>
  <div class="ui-v2-ranking">
    <div class="ui-v2-ranking__toolbar">
      <UiSearchInput v-model="query" :placeholder="searchPlaceholder" />
      <div class="ui-v2-ranking__scope" role="group" aria-label="排行范围">
        <button type="button" :class="{ 'ui-v2-ranking__scope-button--active': scope === 'top12' }" @click="scope = 'top12'">Top 12</button>
        <button type="button" :class="{ 'ui-v2-ranking__scope-button--active': scope === 'all' }" @click="scope = 'all'">全部</button>
      </div>
    </div>
    <div v-if="!filteredItems.length" class="ui-v2-ranking__empty">暂无匹配数据</div>
    <button v-for="item in filteredItems" :key="String(item.id)" type="button" class="ui-v2-ranking__row" :class="{ 'ui-v2-ranking__row--static': !showAction }" @click="showAction && emit('select', item)">
      <span class="ui-v2-ranking__label">{{ item.label }}</span>
      <span class="ui-v2-ranking__track"><span class="ui-v2-ranking__bar" :style="{ width: width(item, filteredItems) }" /></span>
      <span class="ui-v2-ranking__value">{{ item.displayValue ?? item.value.toLocaleString() }}<small> {{ valueLabel }}</small></span>
      <span v-if="item.percentage !== undefined" class="ui-v2-ranking__percentage">{{ item.percentage.toFixed(1) }}%</span>
      <span v-if="showAction" class="ui-v2-ranking__action">查看详情</span>
    </button>
  </div>
</template>
