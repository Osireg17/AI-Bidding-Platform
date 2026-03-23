import { Card, CardContent } from '@/components/ui/card'
import { CountdownTimer } from '@/components/CountdownTimer'
import { StatusBadge } from '@/components/StatusBadge'
import type { Auction } from '@/types/auction'

interface AuctionCardProps {
  auction: Auction | null
}

const gbp = new Intl.NumberFormat('en-GB', { style: 'currency', currency: 'GBP' })

export function AuctionCard({ auction }: AuctionCardProps) {
  if (!auction) {
    return (
      <Card className="relative overflow-hidden border-border bg-card">
        <div className="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-transparent via-primary to-transparent" />
        <CardContent className="flex flex-col items-center justify-center py-12 gap-4">
          <div className="w-10 h-10 rounded-full border border-border flex items-center justify-center">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <circle cx="8" cy="8" r="6" stroke="currentColor" strokeWidth="1.5" strokeDasharray="3 2" className="text-muted-foreground" />
            </svg>
          </div>
          <p className="font-mono text-xs tracking-widest uppercase text-muted-foreground">
            Awaiting next auction
          </p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="relative overflow-hidden border-border bg-card">
      <div className="absolute inset-x-0 top-0 h-0.5 bg-gradient-to-r from-transparent via-primary to-transparent" />

      {/* Header */}
      <CardContent className="pt-6 pb-0">
        <div className="flex items-start justify-between gap-3 mb-2">
          <h1 className="font-display text-2xl font-bold leading-tight text-foreground flex-1" style={{ fontFamily: "'Syne', sans-serif" }}>
            {auction.title}
          </h1>
          <StatusBadge status={auction.status} />
        </div>
        <p className="text-sm leading-relaxed text-muted-foreground mb-6">
          {auction.description}
        </p>
      </CardContent>

      {/* Divider */}
      <div className="border-t border-border mx-7" />

      {/* Stats row */}
      <CardContent className="pt-5 pb-6">
        <div className="grid grid-cols-3 gap-0">
          <div className="border-r border-border pr-5">
            <p className="font-mono text-[9px] tracking-widest uppercase text-muted-foreground mb-1.5">Start</p>
            <p className="font-mono text-base text-muted-foreground tracking-tight">{gbp.format(auction.start_price)}</p>
          </div>

          <div className="px-5 border-r border-border">
            <p className="font-mono text-[9px] tracking-widest uppercase text-muted-foreground mb-1.5">Current</p>
            <p className="font-mono text-2xl font-medium text-primary tracking-tight leading-none">
              {gbp.format(auction.current_price)}
            </p>
          </div>

          <div className="pl-5">
            <p className="font-mono text-[9px] tracking-widest uppercase text-muted-foreground mb-1.5">Closes</p>
            {auction.status === 'closed' ? (
              <p className="font-mono text-sm text-muted-foreground">Ended</p>
            ) : (
              <CountdownTimer endTime={auction.end_time} urgent={auction.status === 'ending_soon'} />
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}