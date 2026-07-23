<template>
  <AppLayout>
    <div class="space-y-6 p-6">
      <div class="flex items-center justify-between">
        <div><h1 class="text-2xl font-semibold">角色与权限</h1><p class="text-sm text-gray-500">权限编码由系统维护，角色可组合授权。</p></div>
        <div class="flex gap-2">
          <button v-if="can('permissions.create')" class="btn btn-secondary" @click="openPermissionCreate">新建业务权限</button>
          <button v-if="can('roles.create')" class="btn btn-primary" @click="showCreate = true">新建角色</button>
        </div>
      </div>
      <div v-if="error" class="rounded bg-red-50 p-3 text-red-700">{{ error }}</div>
      <div class="grid gap-4 lg:grid-cols-3">
        <div class="space-y-2">
          <button v-for="role in roles" :key="role.id" class="card w-full p-4 text-left" :class="{ 'ring-2 ring-primary-500': selected?.id === role.id }" @click="selectRole(role)">
            <div class="flex justify-between"><strong>{{ role.name }}</strong><span>{{ role.status }}</span></div>
            <code>{{ role.code }}</code><span v-if="role.is_system" class="ml-2 text-xs text-amber-600">内置</span>
          </button>
        </div>
        <div v-if="selected" class="card space-y-4 p-5 lg:col-span-2">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-semibold">{{ selected.name }} 的权限</h2>
            <div class="flex gap-2">
              <button v-if="can('roles.update') && selected.code !== 'admin'" class="btn btn-secondary" @click="toggleStatus">{{ selected.status === 'active' ? '停用' : '启用' }}</button>
              <button v-if="can('roles.delete') && !selected.is_system" class="btn btn-danger" @click="removeRole">删除</button>
              <button v-if="can('roles.permissions.assign')" class="btn btn-primary" :disabled="selected.code === 'admin' || saving" @click="savePermissions">保存权限</button>
            </div>
          </div>
          <p v-if="selected.code === 'admin'" class="rounded bg-amber-50 p-3 text-amber-800">超级管理员固定拥有 *，不可移除。</p>
          <p v-if="selected.code === 'user'" class="rounded bg-amber-50 p-3 text-amber-800">修改内置 user 会影响所有普通用户，保存前将再次确认。</p>
          <section v-for="[module, items] in groupedPermissions" :key="module">
            <h3 class="mb-2 font-medium">{{ module }}</h3>
            <label v-for="permission in items" :key="permission.code" class="mb-2 flex gap-3 rounded border p-3">
              <input
                type="checkbox"
                :checked="selectedPermissions.includes(permission.code)"
                :disabled="selected?.code === 'admin' || permission.status !== 'active'"
                @change="togglePermission(permission.code, ($event.target as HTMLInputElement).checked)"
              />
              <span class="flex-1"><b>{{ permission.name }}</b> <code class="text-xs">{{ permission.code }}</code><small class="ml-2" :class="riskClass(permission.risk_level)">{{ permission.risk_level }}</small><small v-if="!permission.is_system" class="ml-2 text-blue-600">业务</small><small v-if="permission.status !== 'active'" class="ml-2 text-gray-500">已停用</small><br><span class="text-sm text-gray-500">{{ permission.description }}</span></span>
              <span v-if="!permission.is_system" class="flex gap-2">
                <button v-if="can('permissions.update')" type="button" class="text-sm text-blue-600" @click.prevent="openPermissionEdit(permission)">编辑</button>
                <button v-if="can('permissions.delete')" type="button" class="text-sm text-red-600" @click.prevent="removePermission(permission)">删除</button>
              </span>
            </label>
          </section>
        </div>
      </div>
      <div v-if="showCreate" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
        <form class="w-full max-w-md space-y-4 rounded-2xl border border-gray-200 bg-white p-6 shadow-2xl dark:border-dark-600 dark:bg-dark-800" @submit.prevent="create">
          <h2 class="text-lg font-semibold">新建自定义角色</h2>
          <input v-model="draft.code" class="input" required pattern="[a-z][a-z0-9_]{1,63}" placeholder="role_code" />
          <input v-model="draft.name" class="input" required placeholder="角色名称" />
          <textarea v-model="draft.description" class="input" placeholder="说明"></textarea>
          <div class="flex justify-end gap-2"><button type="button" class="btn btn-secondary" @click="showCreate=false">取消</button><button class="btn btn-primary">创建</button></div>
        </form>
      </div>
      <div v-if="showPermissionEditor" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4">
        <form class="w-full max-w-lg space-y-4 rounded-2xl border border-gray-200 bg-white p-6 shadow-2xl dark:border-dark-600 dark:bg-dark-800" @submit.prevent="savePermission">
          <h2 class="text-lg font-semibold">{{ permissionDraft.id ? '编辑业务权限' : '新建业务权限' }}</h2>
          <input v-model.trim="permissionDraft.code" class="input" required :disabled="Boolean(permissionDraft.id)" pattern="[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+" placeholder="例如 report.export" />
          <input v-model.trim="permissionDraft.name" class="input" required placeholder="权限名称" />
          <input v-model.trim="permissionDraft.module" class="input" required pattern="[a-z][a-z0-9_]{0,63}" placeholder="模块，例如 report" />
          <textarea v-model="permissionDraft.description" class="input" placeholder="权限用途说明"></textarea>
          <select v-model="permissionDraft.risk_level" class="input">
            <option value="low">低风险</option><option value="medium">中风险</option>
            <option value="high">高风险</option><option value="critical">严重风险</option>
          </select>
          <select v-if="permissionDraft.id" v-model="permissionDraft.status" class="input">
            <option value="active">启用</option><option value="disabled">停用</option>
          </select>
          <div class="flex justify-end gap-2"><button type="button" class="btn btn-secondary" @click="showPermissionEditor=false">取消</button><button class="btn btn-primary">保存</button></div>
        </form>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import { adminAPI } from '@/api/admin'
import type { RBACPermission, RBACRole } from '@/api/admin/rbac'
import { usePermission } from '@/composables/usePermission'

const { can } = usePermission()
const roles = ref<RBACRole[]>([])
const permissions = ref<RBACPermission[]>([])
const selected = ref<RBACRole | null>(null)
const selectedPermissions = ref<string[]>([])
const showCreate = ref(false)
const saving = ref(false)
const error = ref('')
const draft = ref({ code: '', name: '', description: '' })
const showPermissionEditor = ref(false)
const permissionDraft = ref({ id: 0, code: '', name: '', module: '', description: '', risk_level: 'low' as RBACPermission['risk_level'], status: 'active' })
let roleSelectionVersion = 0
const groupedPermissions = computed(() => {
  const groups = new Map<string, RBACPermission[]>()
  for (const item of permissions.value) groups.set(item.module, [...(groups.get(item.module) ?? []), item])
  return [...groups.entries()]
})
async function load() {
  try {
    const [rolePage, catalog] = await Promise.all([adminAPI.rbac.listRoles({ page_size: 200 }), adminAPI.rbac.listPermissions()])
    roles.value = rolePage.items
    permissions.value = catalog
    if (roles.value[0]) await selectRole(roles.value[0])
  } catch (e) { error.value = (e as Error).message }
}
async function selectRole(role: RBACRole) {
  const version = ++roleSelectionVersion
  selected.value = role
  selectedPermissions.value = []
  const values = await adminAPI.rbac.getRolePermissions(role.id)
  if (version === roleSelectionVersion && selected.value?.id === role.id) {
    selectedPermissions.value = [...values]
  }
}
function togglePermission(code: string, checked: boolean) {
  const current = new Set(selectedPermissions.value)
  if (checked) current.add(code)
  else current.delete(code)
  selectedPermissions.value = [...current]
}
async function savePermissions() {
  if (!selected.value) return
  if (selected.value.code === 'user' && !confirm('修改内置 user 权限会影响全部普通用户，确认继续？')) return
  saving.value = true
  try { selectedPermissions.value = await adminAPI.rbac.replaceRolePermissions(selected.value.id, selectedPermissions.value) }
  catch (e) { error.value = (e as Error).message } finally { saving.value = false }
}
async function create() {
  try { await adminAPI.rbac.createRole(draft.value); showCreate.value = false; draft.value = { code:'',name:'',description:'' }; await load() }
  catch (e) { error.value = (e as Error).message }
}
async function toggleStatus() {
  if (!selected.value) return
  await adminAPI.rbac.updateRole(selected.value.id, { status: selected.value.status === 'active' ? 'disabled' : 'active' })
  await load()
}
async function removeRole() {
  if (!selected.value || !confirm(`确认删除角色 ${selected.value.name}？`)) return
  try { await adminAPI.rbac.deleteRole(selected.value.id); selected.value = null; await load() }
  catch (e) { error.value = (e as Error).message }
}
function openPermissionCreate() {
  permissionDraft.value = { id: 0, code: '', name: '', module: '', description: '', risk_level: 'low', status: 'active' }
  showPermissionEditor.value = true
}
function openPermissionEdit(permission: RBACPermission) {
  permissionDraft.value = { ...permission }
  showPermissionEditor.value = true
}
async function savePermission() {
  try {
    const input = { ...permissionDraft.value }
    if (input.id) await adminAPI.rbac.updatePermission(input.id, input)
    else await adminAPI.rbac.createPermission(input)
    showPermissionEditor.value = false
    permissions.value = await adminAPI.rbac.listPermissions()
  } catch (e) { error.value = (e as Error).message }
}
async function removePermission(permission: RBACPermission) {
  if (!confirm(`确认删除业务权限 ${permission.code}？它会同时从所有角色移除。`)) return
  try {
    await adminAPI.rbac.deletePermission(permission.id)
    selectedPermissions.value = selectedPermissions.value.filter(code => code !== permission.code)
    permissions.value = await adminAPI.rbac.listPermissions()
  } catch (e) { error.value = (e as Error).message }
}
function riskClass(risk: string) { return risk === 'critical' || risk === 'high' ? 'text-red-600' : 'text-gray-500' }
onMounted(load)
</script>
