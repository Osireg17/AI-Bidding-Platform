import { Card, CardContent } from '@/components/ui/card'

const bots = [
  { name: 'Aggressive Alice', desc: 'Always bids high and fast' },
  { name: 'Sniper Steve',     desc: 'Waits for the last moment' },
  { name: 'Value Victor',     desc: 'Only bids below fair market value' },
  { name: 'Chaos Charlie',    desc: 'Completely unpredictable' },
]

export function InfoPanel() {
  return (
    <Card className="border-border bg-card">
      <CardContent className="pt-5 pb-5">
        <p className="font-mono text-[10px] tracking-widest uppercase text-primary mb-3">
          How it works
        </p>
        <p className="font-mono text-xs leading-relaxed text-muted-foreground mb-5">
          4 AI bots compete in live auctions, each starting with £1,000,000. A new auction runs
          every hour — 30 minutes of bidding, 30 minutes until the next round. Each bot is powered
          by Google Gemini and makes real decisions based on the auction context.
        </p>
        <div className="border-t border-border pt-4 flex flex-col gap-2.5">
          {bots.map((bot) => (
            <div key={bot.name} className="flex items-baseline justify-between gap-3">
              <span className="font-mono text-xs text-foreground/80 shrink-0">{bot.name}</span>
              <span className="font-mono text-[11px] text-muted-foreground text-right">{bot.desc}</span>
            </div>
          ))}
        </div>
      </CardContent>
    </Card>
  )
}