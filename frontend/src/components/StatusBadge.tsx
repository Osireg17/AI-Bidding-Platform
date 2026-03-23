import { cn } from '@/lib/utils'
import type { AuctionStatus } from '@/types/auction'

interface StatusBadgeProps {
  status: AuctionStatus
}

const STATUS_CONFIG: Record<AuctionStatus, { label: string; className: string; dotClass: string }> = {
  active: {
    label: 'LIVE',
    className: 'bg-green-500/10 border border-green-500/20 text-green-400',
    dotClass: 'bg-green-400',
  },
  ending_soon: {
    label: 'ENDING SOON',
    className: 'bg-amber-500/10 border border-amber-500/20 text-amber-400',
    dotClass: 'bg-amber-400',
  },
  closed: {
    label: 'CLOSED',
    className: 'bg-muted border border-border text-muted-foreground',
    dotClass: 'bg-muted-foreground',
  },
}

export function StatusBadge({ status }: StatusBadgeProps) {
  const cfg = STATUS_CONFIG[status]

  return (
    <div className={cn('inline-flex items-center gap-1.5 px-2.5 py-1 rounded-sm shrink-0 font-mono text-[9px] tracking-widest font-medium', cfg.className)}>
      <span
        className={cn('w-1.5 h-1.5 rounded-full shrink-0', cfg.dotClass)}
        style={{ animation: status === 'active' ? 'live-pulse 2s ease-in-out infinite' : undefined }}
      />
      {cfg.label}
    </div>
  )
}
