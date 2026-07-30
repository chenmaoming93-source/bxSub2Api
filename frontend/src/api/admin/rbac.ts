import { apiClient } from '../client'
import type { PaginatedResponse } from '@/types'

export interface RBACRole {
  id: number
  code: string
  name: string
  description: string
  is_system: boolean
  status: 'active' | 'disabled'
}

export interface RBACPermission {
  id: number
  code: string
  name: string
  module: string
  description: string
  risk_level: 'low' | 'medium' | 'high' | 'critical'
  is_system: boolean
  status: 'active' | 'disabled'
}

async function listRoles(params: { page?: number; page_size?: number; status?: string; search?: string } = {}) {
  const { data } = await apiClient.get<PaginatedResponse<RBACRole>>('/admin/rbac/roles', { params })
  return data
}
async function createRole(input: { code: string; name: string; description?: string }) {
  const { data } = await apiClient.post<RBACRole>('/admin/rbac/roles', input)
  return data
}
async function updateRole(id: number, input: { name?: string; description?: string; status?: string }) {
  const { data } = await apiClient.put<RBACRole>(`/admin/rbac/roles/${id}`, input)
  return data
}
async function deleteRole(id: number) { await apiClient.delete(`/admin/rbac/roles/${id}`) }
async function listPermissions() {
  const { data } = await apiClient.get<RBACPermission[]>('/admin/rbac/permissions')
  return data
}
type PermissionInput = Pick<RBACPermission, 'name' | 'module' | 'description' | 'risk_level'> & { code?: string; status?: string }
async function createPermission(input: PermissionInput & { code: string }) {
  const { data } = await apiClient.post<RBACPermission>('/admin/rbac/permissions', input)
  return data
}
async function updatePermission(id: number, input: PermissionInput & { status: string }) {
  const { data } = await apiClient.put<RBACPermission>(`/admin/rbac/permissions/${id}`, input)
  return data
}
async function deletePermission(id: number) { await apiClient.delete(`/admin/rbac/permissions/${id}`) }
async function getRolePermissions(id: number) {
  const { data } = await apiClient.get<{ permissions: string[] }>(`/admin/rbac/roles/${id}/permissions`)
  return data.permissions
}
async function replaceRolePermissions(id: number, permissions: string[]) {
  const { data } = await apiClient.put<{ permissions: string[] }>(`/admin/rbac/roles/${id}/permissions`, { permissions })
  return data.permissions
}
async function getUserRoles(userId: number) {
  const { data } = await apiClient.get<{ roles: string[] }>(`/admin/users/${userId}/roles`)
  return data.roles
}
async function replaceUserRoles(userId: number, roles: string[]) {
  const { data } = await apiClient.put<{ roles: string[] }>(`/admin/users/${userId}/roles`, { roles })
  return data.roles
}

export default {
  listRoles, createRole, updateRole, deleteRole, listPermissions, createPermission, updatePermission, deletePermission,
  getRolePermissions, replaceRolePermissions, getUserRoles, replaceUserRoles,
}
