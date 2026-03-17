import { renderHook, act } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { useAuction } from '@/hooks/useAuction'

// Minimal EventSource mock
class MockEventSource {
  static CONNECTING = 0
  static OPEN = 1
  static CLOSED = 2

  readyState = MockEventSource.OPEN
  url: string
  listeners: Record<string, Array<(e: MessageEvent) => void>> = {}
  onerror: ((e: Event) => void) | null = null

  constructor(url: string) {
    this.url = url
    MockEventSource.instance = this
  }

  addEventListener(type: string, handler: (e: MessageEvent) => void) {
    if (!this.listeners[type]) this.listeners[type] = []
    this.listeners[type].push(handler)
  }

  removeEventListener(type: string, handler: (e: MessageEvent) => void) {
    if (this.listeners[type]) {
      this.listeners[type] = this.listeners[type].filter((h) => h !== handler)
    }
  }

  close() {
    this.readyState = MockEventSource.CLOSED
  }

  emit(type: string, data: unknown) {
    const event = new MessageEvent(type, { data: JSON.stringify(data) })
    const handlers = this.listeners[type] ?? []
    handlers.forEach((h) => h(event))
  }

  triggerError() {
    this.onerror?.(new Event('error'))
  }

  static instance: MockEventSource
}

vi.stubGlobal('EventSource', MockEventSource)

vi.mock('@/lib/bffUrl', () => ({
  bffUrl: (path: string) => path,
}))

// auction.snapshot uses the normalised AuctionState shape (same as /api/state)
const snapshot = {
  auction: {
    id: 1,
    title: 'Test Item',
    description: 'A test item',
    start_price: 100,
    current_price: 100,
    status: 'active' as const,
    end_time: new Date(Date.now() + 60_000).toISOString(),
  },
  bids: [],
  winner: null,
}

describe('useAuction', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('initialises state from auction.snapshot', () => {
    const { result } = renderHook(() => useAuction())

    act(() => {
      MockEventSource.instance.emit('auction.snapshot', snapshot)
    })

    expect(result.current.auction?.title).toBe('Test Item')
    expect(result.current.bids).toHaveLength(0)
    expect(result.current.winner).toBeNull()
    expect(result.current.connectionStatus).toBe('connected')
  })

  it('prepends bid and updates current_price on bid.placed (raw wire shape)', () => {
    const { result } = renderHook(() => useAuction())

    act(() => {
      MockEventSource.instance.emit('auction.snapshot', snapshot)
    })

    // Real wire shape: bot_id + bid_amount, no bot_name
    act(() => {
      MockEventSource.instance.emit('bid.placed', {
        auction_id: 1,
        bot_id: 1,
        bid_amount: 150,
        bid_id: 42,
        timestamp: new Date().toISOString(),
      })
    })

    expect(result.current.bids).toHaveLength(1)
    expect(result.current.bids[0].bot_name).toBe('Aggressive Alice')
    expect(result.current.bids[0].amount).toBe(150)
    expect(result.current.auction?.current_price).toBe(150)
  })

  it('updates status on auction.ending_soon', () => {
    const { result } = renderHook(() => useAuction())

    act(() => {
      MockEventSource.instance.emit('auction.snapshot', snapshot)
    })

    act(() => {
      MockEventSource.instance.emit('auction.ending_soon', {
        auction_id: 1,
        end_time: new Date(Date.now() + 30_000).toISOString(),
      })
    })

    expect(result.current.auction?.status).toBe('ending_soon')
  })

  it('sets winner and closes auction on auction.ended (raw wire shape)', () => {
    const { result } = renderHook(() => useAuction())

    act(() => {
      MockEventSource.instance.emit('auction.snapshot', snapshot)
    })

    // Real wire shape: winner_bot_id + winning_bid, no bot_name
    act(() => {
      MockEventSource.instance.emit('auction.ended', {
        auction_id: 1,
        winner_bot_id: 2,
        winning_bid: 200,
        total_bids: 5,
        final_status: 'sold',
      })
    })

    expect(result.current.winner?.bot_name).toBe('Sniper Steve')
    expect(result.current.winner?.amount).toBe(200)
    expect(result.current.auction?.status).toBe('closed')
  })

  it('resets bids and winner on auction.created (raw wire shape)', () => {
    const { result } = renderHook(() => useAuction())

    act(() => {
      MockEventSource.instance.emit('auction.snapshot', {
        ...snapshot,
        bids: [{ bot_name: 'Alice', bot_id: 1, amount: 50, timestamp: new Date().toISOString() }],
        winner: { bot_name: 'Alice', bot_id: 1, amount: 50, final_status: 'sold' },
      })
    })

    // Real wire shape: auction_id, no id/status/current_price
    act(() => {
      MockEventSource.instance.emit('auction.created', {
        auction_id: 2,
        title: 'New Item',
        description: 'Another item',
        start_price: 50,
        start_time: new Date().toISOString(),
        end_time: new Date(Date.now() + 120_000).toISOString(),
      })
    })

    expect(result.current.bids).toHaveLength(0)
    expect(result.current.winner).toBeNull()
    expect(result.current.auction?.title).toBe('New Item')
    expect(result.current.auction?.current_price).toBe(50)
    expect(result.current.auction?.status).toBe('active')
  })

  it('switches to polling on SSE error', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => snapshot,
    })
    vi.stubGlobal('fetch', fetchMock)

    const { result } = renderHook(() => useAuction())

    act(() => {
      MockEventSource.instance.triggerError()
    })

    expect(result.current.connectionStatus).toBe('polling')

    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    expect(fetchMock).toHaveBeenCalledWith('/api/state')
  })

  it('closes EventSource on unmount', () => {
    const { unmount } = renderHook(() => useAuction())
    const es = MockEventSource.instance

    unmount()

    expect(es.readyState).toBe(MockEventSource.CLOSED)
  })
})
