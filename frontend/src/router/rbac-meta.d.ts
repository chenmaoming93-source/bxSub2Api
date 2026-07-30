import 'vue-router'

declare module 'vue-router' {
  interface RouteMeta {
    requiredPermission?: string
    requiredPermissions?: string[]
    permissionMode?: 'any' | 'all'
  }
}

export {}
