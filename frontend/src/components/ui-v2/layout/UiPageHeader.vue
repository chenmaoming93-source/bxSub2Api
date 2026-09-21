<script setup lang="ts">
withDefaults(defineProps<{ title: string; description?: string }>(), { description: '' })
</script>

<template><div class="ui-v2-page-header"><div><h1>{{ title }}</h1><p v-if="description">{{ description }}</p></div><div class="ui-v2-page-header__actions"><slot /></div></div></template>
