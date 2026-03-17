import type { AuctionStatus } from '@/types/auction'

interface StatusBadgeProps {
  status: AuctionStatus
}

const STATUS_CONFIG: Record<AuctionStatus, { label: string; color: string; bg: string; dot: string }> = {
  active: {
    label: 'LIVE',
    color: 'rgb(80, 180, 120)',
    bg: 'rgba(80, 180, 120, 0.1)',
    dot: 'rgb(80, 180, 120)',
  },
  ending_soon: {
    label: 'ENDING SOON',
    color: 'rgb(220, 160, 60)',
    bg: 'rgba(220, 160, 60, 0.1)',
    dot: 'rgb(220, 160, 60)',
  },
  closed: {
    label: 'CLOSED',
    color: 'rgb(80, 78, 74)',
    bg: 'rgba(80, 78, 74, 0.15)',
    dot: 'rgb(80, 78, 74)',
  },
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const cfg = STATUS_CONFIG[status]

  return (
    <div
      style={{
        display: 'inline-flex',
        alignItems: 'center',
        gap: 5,
        padding: '4px 10px',
        borderRadius: 3,
        background: cfg.bg,
        border: `1px solid ${cfg.color}22`,
        flexShrink: 0,
      }}
    >
      <span
        style={{
          width: 5,
          height: 5,
          borderRadius: '50%',
          background: cfg.dot,
          flexShrink: 0,
          animation: status === 'active' ? 'live-pulse 2s ease-in-out infinite' : undefined,
        }}
      />
      <span
        style={{
          fontFamily: "'DM Mono', monospace",
          fontSize: 9,
          letterSpacing: '0.12em',
          color: cfg.color,
          fontWeight: 500,
        }}
      >
        {cfg.label}
      </span>
    </div>
  )
}
