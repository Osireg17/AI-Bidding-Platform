import { CountdownTimer } from '@/components/CountdownTimer'
import { StatusBadge } from '@/components/StatusBadge'
import type { Auction } from '@/types/auction'

interface AuctionCardProps {
  auction: Auction | null
}

const gbp = new Intl.NumberFormat('en-GB', { style: 'currency', currency: 'GBP' })

const cardStyle: React.CSSProperties = {
  background: 'rgb(20, 20, 24)',
  border: '1px solid rgb(38, 38, 48)',
  borderRadius: 6,
  overflow: 'hidden',
  position: 'relative',
}

const goldAccentStyle: React.CSSProperties = {
  position: 'absolute',
  top: 0,
  left: 0,
  right: 0,
  height: 2,
  background: 'linear-gradient(90deg, transparent, rgb(212, 170, 80) 30%, rgb(240, 196, 96) 50%, rgb(212, 170, 80) 70%, transparent)',
}

export function AuctionCard({ auction }: AuctionCardProps) {
  if (!auction) {
    return (
      <div style={cardStyle}>
        <div style={goldAccentStyle} />
        <div style={{ padding: '40px 28px', textAlign: 'center' }}>
          <div
            style={{
              width: 40,
              height: 40,
              borderRadius: '50%',
              border: '1px solid rgb(38, 38, 48)',
              margin: '0 auto 16px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
              <circle cx="8" cy="8" r="6" stroke="rgb(80, 78, 74)" strokeWidth="1.5" strokeDasharray="3 2" />
            </svg>
          </div>
          <p
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 11,
              letterSpacing: '0.12em',
              textTransform: 'uppercase',
              color: 'rgb(80, 78, 74)',
            }}
          >
            Awaiting next auction
          </p>
        </div>
      </div>
    )
  }

  return (
    <div style={cardStyle}>
      <div style={goldAccentStyle} />

      {/* Header row */}
      <div style={{ padding: '24px 28px 0' }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 12, marginBottom: 8 }}>
          <h1
            style={{
              fontFamily: "'Syne', sans-serif",
              fontWeight: 700,
              fontSize: 22,
              lineHeight: 1.2,
              color: 'rgb(230, 228, 220)',
              flex: 1,
            }}
          >
            {auction.title}
          </h1>
          <StatusBadge status={auction.status} />
        </div>
        <p
          style={{
            fontSize: 13,
            lineHeight: 1.6,
            color: 'rgb(140, 136, 128)',
            marginBottom: 28,
          }}
        >
          {auction.description}
        </p>
      </div>

      {/* Divider */}
      <div style={{ borderTop: '1px solid rgb(38, 38, 48)', margin: '0 28px' }} />

      {/* Stats row */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: '1fr 1fr 1fr',
          padding: '20px 28px 24px',
          gap: 0,
        }}
      >
        <div style={{ borderRight: '1px solid rgb(38, 38, 48)', paddingRight: 20 }}>
          <p
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 9,
              letterSpacing: '0.14em',
              textTransform: 'uppercase',
              color: 'rgb(80, 78, 74)',
              marginBottom: 6,
            }}
          >
            Start
          </p>
          <p
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 15,
              color: 'rgb(140, 136, 128)',
              letterSpacing: '-0.01em',
            }}
          >
            {gbp.format(auction.start_price)}
          </p>
        </div>

        <div style={{ paddingLeft: 20, paddingRight: 20, borderRight: '1px solid rgb(38, 38, 48)' }}>
          <p
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 9,
              letterSpacing: '0.14em',
              textTransform: 'uppercase',
              color: 'rgb(80, 78, 74)',
              marginBottom: 6,
            }}
          >
            Current
          </p>
          <p
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 22,
              fontWeight: 500,
              color: 'rgb(212, 170, 80)',
              letterSpacing: '-0.02em',
              lineHeight: 1,
            }}
          >
            {gbp.format(auction.current_price)}
          </p>
        </div>

        <div style={{ paddingLeft: 20 }}>
          <p
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 9,
              letterSpacing: '0.14em',
              textTransform: 'uppercase',
              color: 'rgb(80, 78, 74)',
              marginBottom: 6,
            }}
          >
            Closes
          </p>
          {auction.status === 'closed' ? (
            <p
              style={{
                fontFamily: "'DM Mono', monospace",
                fontSize: 13,
                color: 'rgb(80, 78, 74)',
              }}
            >
              Ended
            </p>
          ) : (
            <CountdownTimer endTime={auction.end_time} urgent={auction.status === 'ending_soon'} />
          )}
        </div>
      </div>
    </div>
  )
}
