<script setup lang="ts">
withDefaults(defineProps<{ title?: string; description?: string }>(), { title: '', description: '' })
</script>

<template><section class="ui-v2-section"><header v-if="title || description" class="ui-v2-section__header"><div><h2 v-if="title">{{ title }}</h2><p v-if="description">{{ description }}</p></div><slot name="actions" /></header><slot /></section></template>
