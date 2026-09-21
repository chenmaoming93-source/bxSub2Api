<script setup lang="ts">
withDefaults(defineProps<{ title?: string; description?: string }>(), { title: '', description: '' })
</script>

<template>
  <div class="ui-v2-table-toolbar">
    <div><h3 v-if="title">{{ title }}</h3><p v-if="description">{{ description }}</p></div>
    <div class="ui-v2-table-toolbar__actions"><slot /></div>
  </div>
</template>
