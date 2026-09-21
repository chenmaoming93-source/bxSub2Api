<script setup lang="ts">
export interface UiNavItem { label: string; path?: string; icon?: string; active?: boolean; children?: UiNavItem[] }
withDefaults(defineProps<{ items?: UiNavItem[]; brand?: string }>(), { items: () => [], brand: 'Sub2API' })
</script>

<template>
  <aside class="ui-v2-sidebar">
    <div class="ui-v2-sidebar__brand"><span class="ui-v2-sidebar__mark">S</span><span>{{ brand }}</span></div>
    <nav class="ui-v2-sidebar__nav" aria-label="主导航">
      <template v-for="item in items" :key="item.path ?? item.label">
        <a :href="item.path ?? '#'" class="ui-v2-sidebar__item" :class="{ 'ui-v2-sidebar__item--active': item.active }" @click="$event.preventDefault()">
          <span class="ui-v2-sidebar__icon" aria-hidden="true">{{ item.icon ?? '•' }}</span><span>{{ item.label }}</span>
        </a>
        <a v-for="child in item.children" :key="child.path ?? child.label" :href="child.path ?? '#'" class="ui-v2-sidebar__child" @click="$event.preventDefault()">{{ child.label }}</a>
      </template>
    </nav>
    <div class="ui-v2-sidebar__footer"><slot name="footer" /></div>
  </aside>
</template>
