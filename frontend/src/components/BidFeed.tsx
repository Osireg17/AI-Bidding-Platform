import { useRef, useEffect, useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
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

const BOT_COLORS: Record<string, string> = {
  'Aggressive Alice': 'rgb(220, 100, 100)',
  'Sniper Steve':     'rgb(100, 160, 220)',
  'Value Victor':     'rgb(100, 200, 140)',
  'Chaos Charlie':    'rgb(180, 120, 220)',
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
      className="grid items-center gap-3 px-4 py-2.5 border-b border-border last:border-0 transition-colors"
      style={{ gridTemplateColumns: '32px 1fr auto auto' }}
    >
      <div
        className="w-7 h-7 rounded shrink-0 flex items-center justify-center"
        style={{ background: `${color}18`, border: `1px solid ${color}44` }}
      >
        <span className="font-mono text-[8px] font-medium" style={{ color, letterSpacing: '0.05em' }}>
          {botInitial(bid.bot_name)}
        </span>
      </div>

      <span className="text-sm font-medium text-foreground/80 truncate">{bid.bot_name}</span>

      <span className="font-mono text-sm font-medium text-primary tracking-tight">
        {gbp.format(bid.amount)}
      </span>

      <span className="font-mono text-[10px] text-muted-foreground tracking-wide text-right min-w-[60px]">
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
    <Card className="overflow-hidden border-border bg-card">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3.5 border-b border-border bg-secondary/50">
        <div className="flex items-center gap-2">
          <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
            <path d="M2 10L6 2L10 10H2Z" stroke="rgb(212, 170, 80)" strokeWidth="1.2" fill="rgba(212, 170, 80, 0.1)" />
          </svg>
          <span className="font-mono text-[10px] tracking-widest uppercase text-muted-foreground">
            Bid Activity
          </span>
        </div>
        <span className="font-mono text-[10px] text-muted-foreground tracking-wide">
          {bids.length} bid{bids.length !== 1 ? 's' : ''}
        </span>
      </div>

      {/* Bid list */}
      {bids.length === 0 ? (
        <CardContent className="flex items-center justify-center py-10">
          <p className="font-mono text-xs tracking-widest uppercase text-muted-foreground">
            No bids placed yet
          </p>
        </CardContent>
      ) : (
        <ScrollArea className="h-80">
          <ul>
            {bids.map((bid, idx) => (
              <BidRow key={idx} bid={bid} isNew={idx < newCount} />
            ))}
          </ul>
        </ScrollArea>
      )}
    </Card>
  )
}