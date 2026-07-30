import { storeToRefs } from 'pinia'
import { useAuthStore } from '@/stores/auth'

export function usePermission() {
  const auth = useAuthStore()
  const { roles, permissions, permissionVersion, policyVersion, isSuperAdmin } = storeToRefs(auth)
  return {
    roles,
    permissions,
    permissionVersion,
    policyVersion,
    isSuperAdmin,
    can: auth.can,
    canAny: auth.canAny,
    canAll: auth.canAll,
  }
}
