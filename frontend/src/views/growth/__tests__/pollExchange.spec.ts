import { describe, expect, it, vi } from 'vitest'
import { pollExchangeOrder } from '../pollExchange'

describe('pollExchangeOrder', () => {
  it('stops on terminal state', async () => {
    const load = vi.fn()
      .mockResolvedValueOnce({ status: 'CREATED' })
      .mockResolvedValueOnce({ status: 'SUCCEEDED' })
    const sleep = vi.fn().mockResolvedValue(undefined)

    await expect(pollExchangeOrder(7, load, { sleep, isVisible: () => true })).resolves.toEqual({ status: 'SUCCEEDED' })
    expect(load).toHaveBeenCalledTimes(2)
    expect(sleep).toHaveBeenCalledTimes(1)
  })

  it('stops when the page becomes hidden and never exceeds five attempts', async () => {
    const hiddenLoad = vi.fn().mockResolvedValue({ status: 'FULFILLING' })
    await expect(pollExchangeOrder(7, hiddenLoad, { isVisible: () => false })).resolves.toBeNull()
    expect(hiddenLoad).not.toHaveBeenCalled()

    const load = vi.fn().mockResolvedValue({ status: 'FULFILLING' })
    const sleep = vi.fn().mockResolvedValue(undefined)
    await expect(pollExchangeOrder(7, load, { sleep, isVisible: () => true })).resolves.toBeNull()
    expect(load).toHaveBeenCalledTimes(5)
    expect(sleep).toHaveBeenCalledTimes(4)
  })
})
