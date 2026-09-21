<script setup lang="ts">
withDefaults(defineProps<{ title?: string; description?: string }>(), { title: '暂无数据', description: '' })
</script>

<template>
  <div class="ui-v2-empty-state">
    <div class="ui-v2-empty-state__icon" aria-hidden="true">∅</div>
    <strong>{{ title }}</strong>
    <p v-if="description">{{ description }}</p>
    <slot />
  </div>
</template>
