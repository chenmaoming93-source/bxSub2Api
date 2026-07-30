<template>
  <div v-if="show && user" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4" @click.self="$emit('close')">
    <div class="w-full max-w-lg space-y-4 rounded-2xl border border-gray-200 bg-white p-6 shadow-2xl dark:border-dark-600 dark:bg-dark-800">
      <h2 class="text-lg font-semibold">分配角色：{{ user.email }}</h2>
      <p v-if="error" class="rounded bg-red-50 p-2 text-red-700">{{ error }}</p>
      <label v-for="role in roles" :key="role.id" class="flex items-center gap-3 rounded border p-3">
        <input v-model="selected" type="checkbox" :value="role.code" />
        <span><b>{{ role.name }}</b> <code>{{ role.code }}</code><small v-if="role.code === 'admin'" class="ml-2 text-red-600">超级管理员</small></span>
      </label>
      <p v-if="selected.includes('admin')" class="rounded bg-amber-50 p-2 text-amber-800">授予 admin 将获得所有权限；不能给自己提权，且系统不允许移除最后一名管理员。</p>
      <div class="flex justify-end gap-2">
        <button class="btn btn-secondary" @click="$emit('close')">取消</button>
        <button class="btn btn-primary" :disabled="saving || selected.length === 0" @click="save">保存</button>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, watch } from 'vue'
import { adminAPI } from '@/api/admin'
import type { RBACRole } from '@/api/admin/rbac'
import type { AdminUser } from '@/types'

const props = defineProps<{ show: boolean; user: AdminUser | null }>()
const emit = defineEmits<{ close: []; success: [] }>()
const roles = ref<RBACRole[]>([])
const selected = ref<string[]>([])
const saving = ref(false)
const error = ref('')
watch(() => [props.show, props.user?.id] as const, async ([show]) => {
  if (!show || !props.user) return
  error.value = ''
  try {
    const [page, values] = await Promise.all([
      adminAPI.rbac.listRoles({ page_size: 200, status: 'active' }),
      adminAPI.rbac.getUserRoles(props.user.id),
    ])
    roles.value = page.items
    selected.value = values
  } catch (e) { error.value = (e as Error).message }
}, { immediate: true })
async function save() {
  if (!props.user) return
  if (selected.value.includes('admin') && !confirm('确认授予超级管理员权限？')) return
  saving.value = true
  try {
    selected.value = await adminAPI.rbac.replaceUserRoles(props.user.id, selected.value)
    emit('success'); emit('close')
  } catch (e) { error.value = (e as Error).message } finally { saving.value = false }
}
</script>
