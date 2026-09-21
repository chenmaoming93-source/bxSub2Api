<script setup lang="ts">
withDefaults(defineProps<{ selectedCount?: number }>(), { selectedCount: 0 })
</script>

<template>
  <div v-if="selectedCount > 0" class="ui-v2-bulk-actions"><span>已选择 {{ selectedCount }} 项</span><slot /></div>
</template>
