<template><section class="ui-v2-card"><header v-if="$slots.header || title" class="ui-v2-card__header"><div><h3 v-if="title">{{ title }}</h3><slot name="header" /></div><slot name="actions" /></header><div class="ui-v2-card__body"><slot /></div><footer v-if="$slots.footer" class="ui-v2-card__footer"><slot name="footer" /></footer></section></template>

<script setup lang="ts">
defineProps<{ title?: string }>()
</script>
