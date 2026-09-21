<script setup lang="ts">
withDefaults(defineProps<{ columns?: 1 | 2 | 3 | 4 | 5 }>(), { columns: 4 })
</script>

<template><div class="ui-v2-responsive-grid" :class="`ui-v2-responsive-grid--${columns}`"><slot /></div></template>
