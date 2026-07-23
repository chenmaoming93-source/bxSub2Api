import type { Directive, DirectiveBinding } from 'vue'
import { useAuthStore } from '@/stores/auth'

function applyPermission(element: HTMLElement, binding: DirectiveBinding<string>) {
  element.style.display = useAuthStore().can(binding.value) ? '' : 'none'
}

export const permissionDirective: Directive<HTMLElement, string> = {
  mounted: applyPermission,
  updated: applyPermission,
}
