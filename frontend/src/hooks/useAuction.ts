import { useEffect, useRef, useState } from 'react'
import type { Auction, AuctionState, Bid, Winner } from '@/types/auction'
import { bffUrl } from '@/lib/bffUrl'

export type ConnectionStatus = 'connected' | 'polling' | 'connecting'

interface UseAuctionReturn extends AuctionState {
  connectionStatus: ConnectionStatus
}

// ── Wire shapes broadcast by the BFF over SSE ─────────────────────────────
// These match the Go shared/events structs exactly.

interface SnapshotPayload {
  auction: Auction | null
  bids: Bid[]
  winner: Winner | null
}

// auction.created — raw event, no current_price/status, bot names not resolved
interface AuctionCreatedPayload {
  auction_id: number
  title: string
  description: string
  start_price: number
  start_time: string
  end_time: string
}

// auction.ending_soon — just IDs/time, no extra fields needed
interface AuctionEndingSoonPayload {
  auction_id: number
  end_time: string
}

// auction.ended — uses winner_bot_id + winning_bid, no bot_name
interface AuctionEndedPayload {
  auction_id: number
  winner_bot_id: number
  winning_bid: number
  total_bids: number
  final_status: 'sold' | 'unsold'
}

// bid.placed — uses bid_amount + bot_id, no bot_name
interface BidPlacedPayload {
  auction_id: number
  bot_id: number
  bid_amount: number
  bid_id: number
  timestamp: string
}

// ── Bot name lookup (mirrors BFF domain.BotNames) ─────────────────────────

const BOT_NAMES: Record<number, string> = {
  1: 'Aggressive Alice',
  2: 'Sniper Steve',
  3: 'Value Victor',
  4: 'Chaos Charlie',
}

function botName(id: number): string {
  return BOT_NAMES[id] ?? `Bot #${id}`
}

// ─────────────────────────────────────────────────────────────────────────

const POLL_INTERVAL_MS = 5000

export function useAuction(): UseAuctionReturn {
  const [auction, setAuction] = useState<Auction | null>(null)
  const [bids, setBids] = useState<Bid[]>([])
  const [winner, setWinner] = useState<Winner | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('connecting')

  const esRef = useRef<EventSource | null>(null)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  function stopPolling() {
    if (pollRef.current !== null) {
      clearInterval(pollRef.current)
      pollRef.current = null
    }
  }

  function closeSSE() {
    if (esRef.current) {
      esRef.current.close()
      esRef.current = null
    }
  }

  function startPolling() {
    stopPolling()
    setConnectionStatus('polling')

    async function poll() {
      try {
        const res = await fetch(bffUrl('/api/state'))
        if (!res.ok) return
        const data: SnapshotPayload = await res.json()
        setAuction(data.auction)
        setBids(data.bids ?? [])
        setWinner(data.winner)
      } catch {
        // silently retry on next interval
      }
    }

    void poll()
    pollRef.current = setInterval(() => void poll(), POLL_INTERVAL_MS)
  }

  function connectSSE() {
    closeSSE()
    setConnectionStatus('connecting')

    const es = new EventSource(bffUrl('/api/stream'))
    esRef.current = es

    // First event — full snapshot already shaped as AuctionState
    es.addEventListener('auction.snapshot', (e: MessageEvent) => {
      const data: SnapshotPayload = JSON.parse(e.data)
      setAuction(data.auction)
      setBids(data.bids ?? [])
      setWinner(data.winner)
      setConnectionStatus('connected')
    })

    // Raw event: auction_id, title, description, start_price, start_time, end_time
    es.addEventListener('auction.created', (e: MessageEvent) => {
      const data: AuctionCreatedPayload = JSON.parse(e.data)
      setAuction({
        id: data.auction_id,
        title: data.title,
        description: data.description,
        start_price: data.start_price,
        current_price: data.start_price,
        status: 'active',
        end_time: data.end_time,
      })
      setBids([])
      setWinner(null)
    })

    // Raw event: auction_id, end_time
    es.addEventListener('auction.ending_soon', (e: MessageEvent) => {
      const data: AuctionEndingSoonPayload = JSON.parse(e.data)
      setAuction((prev) => {
        if (!prev || prev.id !== data.auction_id) return prev
        return { ...prev, status: 'ending_soon', end_time: data.end_time }
      })
    })

    // Raw event: winner_bot_id, winning_bid, final_status
    es.addEventListener('auction.ended', (e: MessageEvent) => {
      const data: AuctionEndedPayload = JSON.parse(e.data)
      setWinner({
        bot_name: botName(data.winner_bot_id),
        bot_id: data.winner_bot_id,
        amount: data.winning_bid,
        final_status: data.final_status,
      })
      setAuction((prev) => {
        if (!prev || prev.id !== data.auction_id) return prev
        return { ...prev, status: 'closed' }
      })
    })

    // Raw event: bot_id, bid_amount (no bot_name)
    es.addEventListener('bid.placed', (e: MessageEvent) => {
      const data: BidPlacedPayload = JSON.parse(e.data)
      const newBid: Bid = {
        bot_name: botName(data.bot_id),
        bot_id: data.bot_id,
        amount: data.bid_amount,
        timestamp: data.timestamp,
      }
      setBids((prev) => [newBid, ...prev])
      setAuction((prev) => {
        if (!prev || prev.id !== data.auction_id) return prev
        return { ...prev, current_price: data.bid_amount }
      })
    })

    es.onerror = () => {
      closeSSE()
      startPolling()
    }
  }

  useEffect(() => {
    connectSSE()

    function handleVisibilityChange() {
      if (document.visibilityState === 'visible' && connectionStatus === 'polling') {
        stopPolling()
        connectSSE()
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)

    return () => {
      closeSSE()
      stopPolling()
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  return { auction, bids, winner, connectionStatus }
}
