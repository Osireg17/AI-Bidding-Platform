import { Card, CardContent } from '@/components/ui/card'
import type { Winner } from '@/types/auction'

interface WinnerBannerProps {
  winner: Winner | null
}

const gbp = new Intl.NumberFormat('en-GB', { style: 'currency', currency: 'GBP' })

export function WinnerBanner({ winner }: WinnerBannerProps) {
  if (!winner) return null

  const sold = winner.final_status === 'sold'

  return (
    <Card
      className="winner-reveal relative overflow-hidden"
      style={{
        border: sold ? '1px solid rgba(212, 170, 80, 0.35)' : undefined,
        background: sold
          ? 'linear-gradient(135deg, rgba(212, 170, 80, 0.08) 0%, var(--card) 60%)'
          : undefined,
      }}
    >
      {sold && (
        <div className="absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-primary to-transparent" />
      )}

      <CardContent className="flex items-center gap-4 py-5">
        <div
          className="w-11 h-11 rounded shrink-0 flex items-center justify-center"
          style={{
            background: sold ? 'rgba(212, 170, 80, 0.12)' : 'rgba(80, 78, 74, 0.15)',
            border: sold ? '1px solid rgba(212, 170, 80, 0.3)' : '1px solid var(--border)',
          }}
        >
          {sold ? (
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
              <path
                d="M10 2L12.5 7.5L18.5 8.5L14.25 12.5L15.5 18.5L10 15.5L4.5 18.5L5.75 12.5L1.5 8.5L7.5 7.5L10 2Z"
                fill="rgba(212, 170, 80, 0.25)" stroke="rgb(212, 170, 80)" strokeWidth="1.2" strokeLinejoin="round"
              />
            </svg>
          ) : (
            <svg width="18" height="18" viewBox="0 0 18 18" fill="none">
              <circle cx="9" cy="9" r="7" stroke="currentColor" strokeWidth="1.2" className="text-muted-foreground" />
              <path d="M6 9L8 11L12 7" stroke="currentColor" strokeWidth="1.2" strokeLinecap="round" strokeLinejoin="round" className="text-muted-foreground" />
            </svg>
          )}
        </div>

        <div className="flex-1">
          <p className="font-mono text-[9px] tracking-widest uppercase mb-1" style={{ color: sold ? 'rgb(212, 170, 80)' : undefined }} >
            {sold ? 'Auction Result — Sold' : 'Auction Result — Unsold'}
          </p>
          <p className="text-base font-semibold text-foreground" style={{ fontFamily: "'Syne', sans-serif" }}>
            {sold ? (
              <>
                <span className="text-primary">{winner.bot_name}</span>
                {' '}won at{' '}
                <span className="text-primary">{gbp.format(winner.amount)}</span>
              </>
            ) : (
              <>
                No winner — highest bid was{' '}
                <span className="text-muted-foreground">{gbp.format(winner.amount)}</span>
              </>
            )}
          </p>
        </div>
      </CardContent>
    </Card>
  )
}