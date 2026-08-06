/**
 * 模型基本属性（ModelAttributes）纯函数工具。
 * 组件与单测共用；value 由前端解析后原样提交（后端信任前端）。
 */

import type { ModelAttributes } from '@/types'

export interface ModelAttributeRow {
  uid: string
  key: string
  description: string
  valueText: string
}

/**
 * value 智能解析：
 * - 空文本返回空字符串 ""（显式空字符串是合法值）；
 * - JSON.parse 成功则使用解析结果（"200000"→200000、"true"→true、'["a"]'→["a"]）；
 * - 解析失败按普通字符串处理（如 "2025-06-01"、"任意文本"）。
 */
export function parseAttributeValue(text: string): unknown {
  const trimmed = text.trim()
  if (trimmed === '') {
    return ''
  }
  try {
    return JSON.parse(trimmed)
  } catch {
    return trimmed
  }
}

/** 回显用：把存储值转为可编辑文本。字符串原样；其余类型 JSON.stringify。 */
export function displayAttributeValue(value: unknown): string {
  if (typeof value === 'string') {
    return value
  }
  if (value === undefined || value === null) {
    return ''
  }
  return JSON.stringify(value)
}

/**
 * 由行数组构建提交用 map：
 * - key 去首尾空白后为空的行丢弃（A-01）；
 * - value 输入留空的行丢弃（A-01）；
 * - 重复 key 保留第一份（A-02），后续重复项丢弃（由 UI 提示）。
 */
export function buildModelAttributes(rows: ModelAttributeRow[]): ModelAttributes {
  const out: ModelAttributes = {}
  const seen = new Set<string>()
  for (const row of rows) {
    const key = row.key.trim()
    if (key === '' || row.valueText.trim() === '') {
      continue
    }
    if (seen.has(key)) {
      continue
    }
    seen.add(key)
    const item: { description?: string; value?: unknown } = {
      value: parseAttributeValue(row.valueText)
    }
    const description = row.description.trim()
    if (description !== '') {
      item.description = description
    }
    out[key] = item
  }
  return out
}

/** 回显：map → 行数组（key 保持原有顺序）。 */
export function rowsFromModelAttributes(
  attrs: ModelAttributes | null | undefined,
  makeUid: () => string
): ModelAttributeRow[] {
  if (!attrs) {
    return []
  }
  return Object.entries(attrs).map(([key, item]) => ({
    uid: makeUid(),
    key,
    description: item?.description ?? '',
    valueText: displayAttributeValue(item?.value)
  }))
}
