export type PendingOrderStatus = 'CREATED' | 'FULFILLING' | 'SUCCEEDED' | 'FAILED' | 'REFUNDED'

export interface PendingOrderSnapshot {
  status: PendingOrderStatus | string
}

export interface PollExchangeOptions {
  attempts?: number
  intervalMs?: number
  isVisible?: () => boolean
  sleep?: (ms: number) => Promise<void>
}

export async function pollExchangeOrder(
  orderId: number,
  loadOrder: (id: number) => Promise<PendingOrderSnapshot>,
  options: PollExchangeOptions = {}
): Promise<PendingOrderSnapshot | null> {
  const attempts = Math.min(5, Math.max(0, options.attempts ?? 5))
  const intervalMs = options.intervalMs ?? 2000
  const isVisible = options.isVisible ?? (() => document.visibilityState === 'visible')
  const sleep = options.sleep ?? ((ms: number) => new Promise<void>(resolve => setTimeout(resolve, ms)))
  const terminal = new Set(['SUCCEEDED', 'FAILED', 'REFUNDED'])

  for (let attempt = 0; attempt < attempts; attempt += 1) {
    if (!isVisible()) return null
    const order = await loadOrder(orderId)
    if (terminal.has(String(order.status))) return order
    if (attempt < attempts - 1) await sleep(intervalMs)
  }
  return null
}
