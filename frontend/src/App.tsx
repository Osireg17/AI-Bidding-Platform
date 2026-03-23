import { useAuction } from '@/hooks/useAuction'
import { AuctionCard } from '@/components/AuctionCard'
import { BidFeed } from '@/components/BidFeed'
import { WinnerBanner } from '@/components/WinnerBanner'
import { InfoPanel } from '@/components/InfoPanel'
import { cn } from '@/lib/utils'
import type { ConnectionStatus } from '@/hooks/useAuction'

function ConnectionIndicator({ status }: { status: ConnectionStatus }) {
  const isLive = status === 'connected'
  const isPolling = status === 'polling'

  return (
    <div className="flex items-center gap-2">
      <span
        className={cn(
          'w-1.5 h-1.5 rounded-full shrink-0',
          isLive ? 'bg-green-400' : isPolling ? 'bg-amber-400' : 'bg-muted-foreground',
        )}
        style={{ animation: isLive ? 'live-pulse 2s ease-in-out infinite' : undefined }}
      />
      <span className={cn(
        'font-mono text-[11px] tracking-widest uppercase',
        isLive ? 'text-green-400' : isPolling ? 'text-amber-400' : 'text-muted-foreground',
      )}>
        {isLive ? 'Live' : isPolling ? 'Polling' : 'Connecting'}
      </span>
    </div>
  )
}

export default function App() {
  const { auction, bids, winner, connectionStatus } = useAuction()

  return (
    <div className="min-h-screen bg-background">
      <header className="sticky top-0 z-10 border-b border-border px-6 h-14 flex items-center justify-between bg-background/90 backdrop-blur-md">
        <div className="flex items-center gap-3">
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M10 2L18 7V13L10 18L2 13V7L10 2Z" stroke="rgb(212, 170, 80)" strokeWidth="1.5" fill="none" />
            <path d="M10 6L14 8.5V13.5L10 16L6 13.5V8.5L10 6Z" fill="rgba(212, 170, 80, 0.15)" stroke="rgb(212, 170, 80)" strokeWidth="1" />
          </svg>
          <span className="font-bold text-[15px] tracking-wide text-foreground" style={{ fontFamily: "'Syne', sans-serif" }}>
            AUCT<span className="text-primary">IO</span>N
          </span>
          <span className="font-mono text-[10px] text-muted-foreground tracking-widest border-l border-border pl-3 ml-1">
            AI PLATFORM
          </span>
        </div>
        <ConnectionIndicator status={connectionStatus} />
      </header>

      <main className="max-w-[680px] mx-auto px-6 py-8 pb-16 flex flex-col gap-6">
        <InfoPanel />
        <AuctionCard auction={auction} />
        <WinnerBanner winner={winner} />
        <BidFeed bids={bids} />
      </main>
    </div>
  )
}