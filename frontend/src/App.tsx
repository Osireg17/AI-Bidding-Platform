import { useAuction } from '@/hooks/useAuction'
import { AuctionCard } from '@/components/AuctionCard'
import { BidFeed } from '@/components/BidFeed'
import { WinnerBanner } from '@/components/WinnerBanner'
import type { ConnectionStatus } from '@/hooks/useAuction'

function ConnectionIndicator({ status }: { status: ConnectionStatus }) {
  const isLive = status === 'connected'
  const isPolling = status === 'polling'

  return (
    <div className="flex items-center gap-2">
      <span
        style={{
          width: 7,
          height: 7,
          borderRadius: '50%',
          background: isLive
            ? 'rgb(80, 220, 120)'
            : isPolling
              ? 'rgb(220, 160, 60)'
              : 'rgb(80, 78, 74)',
          animation: isLive ? 'live-pulse 2s ease-in-out infinite' : undefined,
          display: 'inline-block',
          flexShrink: 0,
        }}
      />
      <span
        style={{
          fontFamily: "'DM Mono', monospace",
          fontSize: 11,
          letterSpacing: '0.08em',
          textTransform: 'uppercase',
          color: isLive
            ? 'rgb(80, 220, 120)'
            : isPolling
              ? 'rgb(220, 160, 60)'
              : 'rgb(80, 78, 74)',
        }}
      >
        {isLive ? 'Live' : isPolling ? 'Polling' : 'Connecting'}
      </span>
    </div>
  )
}

export default function App() {
  const { auction, bids, winner, connectionStatus } = useAuction()

  return (
    <div style={{ minHeight: '100vh', background: 'rgb(14, 14, 16)' }}>
      {/* Header */}
      <header
        style={{
          borderBottom: '1px solid rgb(38, 38, 48)',
          padding: '0 24px',
          height: 56,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          position: 'sticky',
          top: 0,
          background: 'rgba(14, 14, 16, 0.92)',
          backdropFilter: 'blur(12px)',
          zIndex: 10,
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          {/* Logo mark */}
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
            <path d="M10 2L18 7V13L10 18L2 13V7L10 2Z" stroke="rgb(212, 170, 80)" strokeWidth="1.5" fill="none" />
            <path d="M10 6L14 8.5V13.5L10 16L6 13.5V8.5L10 6Z" fill="rgba(212, 170, 80, 0.15)" stroke="rgb(212, 170, 80)" strokeWidth="1" />
          </svg>
          <span
            style={{
              fontFamily: "'Syne', sans-serif",
              fontWeight: 700,
              fontSize: 15,
              letterSpacing: '0.04em',
              color: 'rgb(230, 228, 220)',
            }}
          >
            AUCT<span style={{ color: 'rgb(212, 170, 80)' }}>IO</span>N
          </span>
          <span
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 10,
              color: 'rgb(80, 78, 74)',
              letterSpacing: '0.1em',
              borderLeft: '1px solid rgb(38, 38, 48)',
              paddingLeft: 12,
              marginLeft: 4,
            }}
          >
            AI PLATFORM
          </span>
        </div>

        <ConnectionIndicator status={connectionStatus} />
      </header>

      {/* Main */}
      <main
        style={{
          maxWidth: 680,
          margin: '0 auto',
          padding: '32px 24px 64px',
          display: 'flex',
          flexDirection: 'column',
          gap: 24,
        }}
      >
        <AuctionCard auction={auction} />
        <WinnerBanner winner={winner} />
        <BidFeed bids={bids} />
      </main>
    </div>
  )
}
