export type ObservabilityStatusLevel = 'healthy' | 'attention' | 'warning' | 'overloaded'

export const OBSERVABILITY_STATUS_THRESHOLDS = {
  attention: 0.6,
  warning: 0.8,
  overloaded: 0.95,
} as const

export interface ObservabilityStatus {
  level: ObservabilityStatusLevel
  labelKey: string
  textClass: string
  barClass: string
  backgroundColor: string
}

const STATUS: Record<ObservabilityStatusLevel, ObservabilityStatus> = {
  healthy: {
    level: 'healthy',
    labelKey: 'admin.observability.statusHealthy',
    textClass: 'text-emerald-600 dark:text-emerald-400',
    barClass: 'bg-emerald-500',
    backgroundColor: '#bbf7d0',
  },
  attention: {
    level: 'attention',
    labelKey: 'admin.observability.statusAttention',
    textClass: 'text-amber-600 dark:text-amber-400',
    barClass: 'bg-amber-500',
    backgroundColor: '#fde68a',
  },
  warning: {
    level: 'warning',
    labelKey: 'admin.observability.statusWarning',
    textClass: 'text-orange-600 dark:text-orange-400',
    barClass: 'bg-orange-500',
    backgroundColor: '#fdba74',
  },
  overloaded: {
    level: 'overloaded',
    labelKey: 'admin.observability.statusOverloaded',
    textClass: 'text-red-600 dark:text-red-400',
    barClass: 'bg-red-500',
    backgroundColor: '#fca5a5',
  },
}

export function statusFor(value: number): ObservabilityStatus {
  if (!Number.isFinite(value)) return STATUS.healthy
  if (value >= OBSERVABILITY_STATUS_THRESHOLDS.overloaded) return STATUS.overloaded
  if (value >= OBSERVABILITY_STATUS_THRESHOLDS.warning) return STATUS.warning
  if (value >= OBSERVABILITY_STATUS_THRESHOLDS.attention) return STATUS.attention
  return STATUS.healthy
}

export function isHighPressure(value: number): boolean {
  return statusFor(value).level === 'warning' || statusFor(value).level === 'overloaded'
}
