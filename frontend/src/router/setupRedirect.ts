export function resolveCompletedSetupRedirectPath(isAuthenticated: boolean, _hasAdminAccess: boolean): string {
  if (!isAuthenticated) {
    return '/login'
  }

  return '/welcome'
}
