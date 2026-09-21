<template>
  <div class="ui-v2-shell"><UiSidebar :items="items" /><div class="ui-v2-shell__main"><UiTopbar><slot name="topbar" /></UiTopbar><main class="ui-v2-shell__content"><slot /></main></div></div>
</template>

<script setup lang="ts">
import UiSidebar, { type UiNavItem } from './UiSidebar.vue'
import UiTopbar from './UiTopbar.vue'
defineProps<{ items?: UiNavItem[] }>()
</script>
