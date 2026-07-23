<template>
  <AppLayout>
    <main class="mx-auto flex min-h-[60vh] max-w-3xl items-center justify-center p-6">
      <section class="w-full rounded-2xl border border-gray-200 bg-white p-8 text-center shadow-sm dark:border-dark-600 dark:bg-dark-800">
        <div class="mx-auto mb-5 flex h-14 w-14 items-center justify-center rounded-2xl bg-primary-50 text-2xl text-primary-600 dark:bg-primary-900/30 dark:text-primary-300">
          ✓
        </div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">欢迎登录</h1>
        <p class="mt-3 text-gray-500 dark:text-gray-400">
          你好，{{ displayName }}。请选择左侧菜单进入你有权限使用的功能。
        </p>
      </section>
    </main>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useAuthStore } from '@/stores/auth'

const authStore = useAuthStore()
const displayName = computed(() =>
  authStore.user?.username || authStore.user?.email || '用户'
)
</script>
