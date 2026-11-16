import { useI18n } from 'vue-i18n'

type TranslateParams = Record<string, unknown>
type Translate = (key: string, params?: TranslateParams) => string

function interpolateFallback(key: string, params?: TranslateParams): string {
  return Object.entries(params || {}).reduce(
    (message, [name, value]) => message.split(`{${name}}`).join(String(value)),
    key,
  )
}

/**
 * Component tests mount observability widgets without the application i18n
 * plugin. Keep production translation unchanged while allowing those isolated
 * mounts to render their i18n keys as stable fallback text.
 */
export function useObservabilityI18n(): { t: Translate } {
  try {
    return useI18n()
  } catch {
    return { t: interpolateFallback }
  }
}
