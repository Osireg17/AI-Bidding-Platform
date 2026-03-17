import type { Winner } from '@/types/auction'

interface WinnerBannerProps {
  winner: Winner | null
}

const gbp = new Intl.NumberFormat('en-GB', { style: 'currency', currency: 'GBP' })

export function WinnerBanner({ winner }: WinnerBannerProps) {
  if (!winner) return null

  const sold = winner.final_status === 'sold'

  return (
    <div
      className="winner-reveal"
      style={{
        position: 'relative',
        borderRadius: 6,
        overflow: 'hidden',
        border: sold ? '1px solid rgba(212, 170, 80, 0.35)' : '1px solid rgb(38, 38, 48)',
        background: sold
          ? 'linear-gradient(135deg, rgba(212, 170, 80, 0.08) 0%, rgba(20, 20, 24, 0.95) 60%)'
          : 'rgb(20, 20, 24)',
      }}
    >
      {/* Top accent */}
      {sold && (
        <div
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            height: 1,
            background: 'linear-gradient(90deg, transparent, rgb(212, 170, 80) 40%, rgb(240, 196, 96) 50%, rgb(212, 170, 80) 60%, transparent)',
          }}
        />
      )}

      <div style={{ padding: '20px 24px', display: 'flex', alignItems: 'center', gap: 16 }}>
        {/* Icon */}
        <div
          style={{
            width: 44,
            height: 44,
            borderRadius: 4,
            background: sold ? 'rgba(212, 170, 80, 0.12)' : 'rgba(80, 78, 74, 0.15)',
            border: sold ? '1px solid rgba(212, 170, 80, 0.3)' : '1px solid rgb(38, 38, 48)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}
        >
          {sold ? (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path d="M10 2L12.5 7.5L18.5 8.5L14.25 12.5L15.5 18.5L10 15.5L4.5 18.5L5.75 12.5L1.5 8.5L7.5 7.5L10 2Z"
                fill="rgba(212, 170, 80, 0.25)" stroke="rgb(212, 170, 80)" strokeWidth="1.2" strokeLinejoin="round" />
            </svg>
          ) : (
            <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
              <circle cx="9" cy="9" r="7" stroke="rgb(80, 78, 74)" strokeWidth="1.2" />
              <path d="M6 9L8 11L12 7" stroke="rgb(80, 78, 74)" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          )}
        </div>

        {/* Text */}
        <div style={{ flex: 1 }}>
          <p
            style={{
              fontFamily: "'DM Mono', monospace",
              fontSize: 9,
              letterSpacing: '0.14em',
              textTransform: 'uppercase',
              color: sold ? 'rgb(212, 170, 80)' : 'rgb(80, 78, 74)',
              marginBottom: 4,
            }}
          >
            {sold ? 'Auction Result — Sold' : 'Auction Result — Unsold'}
          </p>
          <p
            style={{
              fontFamily: "'Syne', sans-serif",
              fontWeight: 600,
              fontSize: 16,
              color: 'rgb(230, 228, 220)',
            }}
          >
            {sold
              ? (
                <>
                  <span style={{ color: 'rgb(212, 170, 80)' }}>{winner.bot_name}</span>
                  {' '}won at{' '}
                  <span style={{ color: 'rgb(212, 170, 80)' }}>{gbp.format(winner.amount)}</span>
                </>
              )
              : (
                <>
                  No winner — highest bid was{' '}
                  <span style={{ color: 'rgb(140, 136, 128)' }}>{gbp.format(winner.amount)}</span>
                </>
              )}
          </p>
        </div>
      </div>
    </div>
  )
}
