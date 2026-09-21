<script setup lang="ts">
withDefaults(defineProps<{ title: string; description?: string; loading?: boolean; error?: string }>(), { description: '', loading: false, error: '' })
</script>

<template><section class="ui-v2-chart-card"><header class="ui-v2-chart-card__header"><div><h3>{{ title }}</h3><p v-if="description">{{ description }}</p></div><div><slot name="actions" /></div></header><div class="ui-v2-chart-card__body"><div v-if="loading" class="ui-v2-chart-card__state">加载中…</div><div v-else-if="error" class="ui-v2-chart-card__state ui-v2-chart-card__state--error">{{ error }}</div><slot v-else /></div></section></template>
