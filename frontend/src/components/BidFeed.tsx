import { useRef, useEffect, useState } from 'react'
import type { Bid } from '@/types/auction'

interface BidFeedProps {
  bids: Bid[]
}

const gbp = new Intl.NumberFormat('en-GB', { style: 'currency', currency: 'GBP' })

function formatTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-GB', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

// Per-bot colour coding
const BOT_COLORS: Record<string, string> = {
  'Aggressive Alice': 'rgb(220, 100, 100)',
  'Sniper Steve': 'rgb(100, 160, 220)',
  'Value Victor': 'rgb(100, 200, 140)',
  'Chaos Charlie': 'rgb(180, 120, 220)',
}

function botColor(name: string): string {
  return BOT_COLORS[name] ?? 'rgb(140, 136, 128)'
}

function botInitial(name: string): string {
  return name.split(' ').map((w) => w[0]).join('').slice(0, 2).toUpperCase()
}

function BidRow({ bid, isNew }: { bid: Bid; isNew: boolean }) {
  const ref = useRef<HTMLLIElement>(null)

  useEffect(() => {
    if (isNew && ref.current) {
      ref.current.classList.add('bid-row-new')
      const t = setTimeout(() => ref.current?.classList.remove('bid-row-new'), 600)
      return () => clearTimeout(t)
    }
  }, [isNew])

  const color = botColor(bid.bot_name)

  return (
    <li
      ref={ref}
      style={{
        display: 'grid',
        gridTemplateColumns: '32px 1fr auto auto',
        alignItems: 'center',
        gap: 12,
        padding: '10px 16px',
        borderBottom: '1px solid rgb(38, 38, 48)',
        transition: 'background 0.3s',
      }}
    >
      {/* Avatar */}
      <div
        style={{
          width: 28,
          height: 28,
          borderRadius: 3,
          background: `${color}18`,
          border: `1px solid ${color}44`,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          flexShrink: 0,
        }}
      >
        <span
          style={{
            fontFamily: "'DM Mono', monospace",
            fontSize: 8,
            fontWeight: 500,
            color,
            letterSpacing: '0.05em',
          }}
        >
          {botInitial(bid.bot_name)}
        </span>
      </div>

      {/* Name */}
      <span
        style={{
          fontSize: 13,
          fontWeight: 500,
          color: 'rgb(200, 198, 190)',
          whiteSpace: 'nowrap',
          overflow: 'hidden',
          textOverflow: 'ellipsis',
        }}
      >
        {bid.bot_name}
      </span>

      {/* Amount */}
      <span
        style={{
          fontFamily: "'DM Mono', monospace",
          fontSize: 14,
          fontWeight: 500,
          color: 'rgb(212, 170, 80)',
          letterSpacing: '-0.01em',
        }}
      >
        {gbp.format(bid.amount)}
      </span>

      {/* Time */}
      <span
        style={{
          fontFamily: "'DM Mono', monospace",
          fontSize: 10,
          color: 'rgb(80, 78, 74)',
          letterSpacing: '0.04em',
          minWidth: 60,
          textAlign: 'right',
        }}
      >
        {formatTime(bid.timestamp)}
      </span>
    </li>
  )
}

export function BidFeed({ bids }: BidFeedProps) {
  const [prevLength, setPrevLength] = useState(bids.length)

  useEffect(() => {
    setPrevLength(bids.length)
  }, [bids.length])

  const newCount = bids.length > prevLength ? bids.length - prevLength : 0

  return (
    <div
      style={{
        background: 'rgb(20, 20, 24)',
        border: '1px solid rgb(38, 38, 48)',
        borderRadius: 6,
        overflow: 'hidden',
      }}
    >
      {/* Feed header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '14px 16px',
          borderBottom: '1px solid rgb(38, 38, 48)',
          background: 'rgb(17, 17, 20)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path d="M2 10L6 2L10 10H2Z" stroke="rgb(212, 170, 80)" strokeWidth="1.2" fill="rgba(212, 170, 80, 0.1)" />
          </svg>
          <span
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 10,
              letterSpacing: '0.12em',
              textTransform: 'uppercase',
              color: 'rgb(140, 136, 128)',
            }}
          >
            Bid Activity
          </span>
        </div>
        <span
          style={{
            fontFamily: "'DM Mono', monospace",
            fontSize: 10,
            color: 'rgb(80, 78, 74)',
            letterSpacing: '0.06em',
          }}
        >
          {bids.length} bid{bids.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Bid list */}
      <div style={{ maxHeight: 320, overflowY: 'auto' }}>
        {bids.length === 0 ? (
          <div
            style={{
              padding: '40px 16px',
              textAlign: 'center',
            }}
          >
            <p
              style={{
                fontFamily: "'DM Mono', monospace",
                fontSize: 11,
                letterSpacing: '0.1em',
                color: 'rgb(80, 78, 74)',
                textTransform: 'uppercase',
              }}
            >
              No bids placed yet
            </p>
          </div>
        ) : (
          <ul style={{ listStyle: 'none' }}>
            {bids.map((bid, idx) => (
              <BidRow key={idx} bid={bid} isNew={idx < newCount} />
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
