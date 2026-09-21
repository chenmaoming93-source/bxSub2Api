<script setup lang="ts">
withDefaults(defineProps<{ rows?: number; columns?: number }>(), { rows: 5, columns: 4 })
</script>

<template>
  <div class="ui-v2-table-skeleton" aria-label="加载中">
    <div v-for="row in rows" :key="row" class="ui-v2-table-skeleton__row">
      <span v-for="column in columns" :key="column" class="ui-v2-skeleton-line" />
    </div>
  </div>
</template>
