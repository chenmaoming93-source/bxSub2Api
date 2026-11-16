import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'
import zh from '@/i18n/locales/zh'
import en from '@/i18n/locales/en'

const files = [
  'src/views/admin/ModelObservabilityView.vue',
  'src/components/observability/ConcurrencyCapacityCard.vue',
  'src/components/observability/DrilldownPanel.vue',
  'src/components/observability/ObservabilityAlertSummary.vue',
  'src/components/observability/PressureHeatmap.vue',
  'src/components/observability/PressureRanking.vue',
  'src/components/observability/QualityTopPeriods.vue',
  'src/components/observability/QualityTrendChart.vue',
  'src/components/observability/TokenTrendChart.vue',
]

describe('observability localization', () => {
  it('defines every observed i18n key in both Chinese and English', () => {
    const keys = new Set<string>()
    for (const file of files) {
      const source = readFileSync(resolve(process.cwd(), file), 'utf8')
      for (const match of source.matchAll(/t\('admin\.observability\.([A-Za-z0-9_]+)'/g)) {
        keys.add(match[1])
      }
    }
    const zhMessages = zh.admin.observability as Record<string, unknown>
    const enMessages = en.admin.observability as Record<string, unknown>
    for (const key of keys) {
      expect(zhMessages[key], `missing zh key ${key}`).toBeTruthy()
      expect(enMessages[key], `missing en key ${key}`).toBeTruthy()
    }
  })

  it('does not retain hard-coded dashboard labels in the components', () => {
    const combined = files
      .map((file) => readFileSync(resolve(process.cwd(), file), 'utf8'))
      .join('\n')
    for (const label of [
      'Model Observability · 实时',
      'Path matrix',
      'Call chain',
      'No pressure samples',
      'No matching paths',
      'Pending capacity and quality alerts',
      'Pressure / value heatmap',
    ]) {
      expect(combined).not.toContain(label)
    }
  })
})
