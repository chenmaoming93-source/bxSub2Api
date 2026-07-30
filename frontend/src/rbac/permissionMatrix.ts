export type PagePermissionDefinition = {
  path: string
  permission: string | null
  classification: 'public' | 'authenticated' | 'rbac'
}

// Migration baseline only. Router guards consume this catalog in a later MVP.
export const pagePermissionMatrix: PagePermissionDefinition[] = [
  { path: '/', permission: null, classification: 'authenticated' },
  { path: '/setup', permission: null, classification: 'public' },
  { path: '/home', permission: null, classification: 'public' },
  { path: '/welcome', permission: null, classification: 'authenticated' },
  { path: '/login', permission: null, classification: 'public' },
  { path: '/register', permission: null, classification: 'public' },
  { path: '/email-verify', permission: null, classification: 'public' },
  { path: '/auth/callback', permission: null, classification: 'public' },
  { path: '/auth/linuxdo/callback', permission: null, classification: 'public' },
  { path: '/auth/wechat/callback', permission: null, classification: 'public' },
  { path: '/auth/wechat/payment/callback', permission: null, classification: 'public' },
  { path: '/auth/dingtalk/callback', permission: null, classification: 'public' },
  { path: '/auth/dingtalk/email-completion', permission: null, classification: 'public' },
  { path: '/auth/oidc/callback', permission: null, classification: 'public' },
  { path: '/forgot-password', permission: null, classification: 'public' },
  { path: '/reset-password', permission: null, classification: 'public' },
  { path: '/legal/:documentId', permission: null, classification: 'public' },
  { path: '/dashboard', permission: 'usage.self.read', classification: 'rbac' },
  { path: '/keys', permission: 'api_keys.self.read', classification: 'rbac' },
  { path: '/key-usage', permission: 'usage.self.read', classification: 'rbac' },
  { path: '/usage', permission: 'usage.self.read', classification: 'rbac' },
  { path: '/redeem', permission: 'redeem.self.read', classification: 'rbac' },
  { path: '/affiliate', permission: 'affiliate.self.read', classification: 'rbac' },
  { path: '/available-channels', permission: 'channels.self.read', classification: 'rbac' },
  { path: '/profile', permission: 'profile.self.read', classification: 'rbac' },
  { path: '/subscriptions', permission: 'subscriptions.self.read', classification: 'rbac' },
  { path: '/purchase', permission: 'payments.self.read', classification: 'rbac' },
  { path: '/orders', permission: 'payments.self.read', classification: 'rbac' },
  { path: '/payment/qrcode', permission: 'payments.self.read', classification: 'rbac' },
  { path: '/payment/result', permission: 'payments.self.read', classification: 'rbac' },
  { path: '/payment/stripe', permission: 'payments.self.read', classification: 'rbac' },
  { path: '/payment/airwallex', permission: 'payments.self.read', classification: 'rbac' },
  { path: '/payment/stripe-popup', permission: 'payments.self.read', classification: 'rbac' },
  { path: '/custom/:id', permission: 'pages.self.read', classification: 'rbac' },
  { path: '/monitor', permission: 'monitors.self.read', classification: 'rbac' },
  { path: '/admin', permission: 'dashboard.read', classification: 'rbac' },
  { path: '/admin/dashboard', permission: 'dashboard.read', classification: 'rbac' },
  { path: '/admin/ops', permission: 'ops.read', classification: 'rbac' },
  { path: '/admin/users', permission: 'users.read', classification: 'rbac' },
  { path: '/admin/roles', permission: 'roles.read', classification: 'rbac' },
  { path: '/admin/groups', permission: 'groups.read', classification: 'rbac' },
  { path: '/admin/default-group-routing', permission: 'groups.read', classification: 'rbac' },
  { path: '/admin/accounts', permission: 'accounts.read', classification: 'rbac' },
  { path: '/admin/announcements', permission: 'announcements.read', classification: 'rbac' },
  { path: '/admin/proxies', permission: 'proxies.read', classification: 'rbac' },
  { path: '/admin/redeem', permission: 'redeem_codes.read', classification: 'rbac' },
  { path: '/admin/promo-codes', permission: 'promo_codes.read', classification: 'rbac' },
  { path: '/admin/settings', permission: 'settings.read', classification: 'rbac' },
  { path: '/admin/risk-control', permission: 'risk.read', classification: 'rbac' },
  { path: '/admin/usage', permission: 'usage.admin.read', classification: 'rbac' },
  { path: '/admin/token-usage/models', permission: 'token_usage.read', classification: 'rbac' },
  { path: '/admin/token-usage/routes', permission: 'token_usage.read', classification: 'rbac' },
  { path: '/admin/token-usage/users', permission: 'token_usage.read', classification: 'rbac' },
  { path: '/admin/token-usage/user-group-model-daily', permission: 'token_usage.read', classification: 'rbac' },
  { path: '/admin/affiliates/invites', permission: 'affiliates.read', classification: 'rbac' },
  { path: '/admin/affiliates/rebates', permission: 'affiliates.read', classification: 'rbac' },
  { path: '/admin/affiliates/transfers', permission: 'affiliates.read', classification: 'rbac' },
  { path: '/admin/orders/dashboard', permission: 'billing.read', classification: 'rbac' },
  { path: '/admin/orders', permission: 'billing.read', classification: 'rbac' },
  { path: '/admin/orders/plans', permission: 'billing.read', classification: 'rbac' },
  { path: '/admin/subscriptions', permission: 'subscriptions.read', classification: 'rbac' },
  { path: '/admin/channels', permission: 'channels.read', classification: 'rbac' },
  { path: '/admin/channels/pricing', permission: 'channels.read', classification: 'rbac' },
  { path: '/admin/channels/monitor', permission: 'monitors.read', classification: 'rbac' },
  { path: '/admin/affiliates', permission: 'affiliates.read', classification: 'rbac' },
  { path: '/:pathMatch(.*)*', permission: null, classification: 'public' },
]

export function permissionForPage(path: string): string | null | undefined {
  const exact = pagePermissionMatrix.find((entry) => entry.path === path)
  if (exact) return exact.permission
  if (path.startsWith('/custom/')) {
    return pagePermissionMatrix.find((entry) => entry.path === '/custom/:id')?.permission
  }
  return undefined
}
