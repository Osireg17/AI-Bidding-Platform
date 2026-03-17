import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { BidFeed } from '@/components/BidFeed'
import type { Bid } from '@/types/auction'

const bids: Bid[] = [
  { bot_name: 'Aggressive Alice', bot_id: 1, amount: 150, timestamp: '2026-03-17T10:00:00Z' },
  { bot_name: 'Sniper Steve', bot_id: 2, amount: 200, timestamp: '2026-03-17T10:01:00Z' },
]

describe('BidFeed', () => {
  it('renders "No bids yet" when bids array is empty', () => {
    render(<BidFeed bids={[]} />)
    expect(screen.getByText(/no bids placed yet/i)).toBeInTheDocument()
  })

  it('renders all bot names', () => {
    render(<BidFeed bids={bids} />)
    expect(screen.getByText('Aggressive Alice')).toBeInTheDocument()
    expect(screen.getByText('Sniper Steve')).toBeInTheDocument()
  })

  it('formats bid amounts as GBP', () => {
    render(<BidFeed bids={bids} />)
    expect(screen.getByText('£150.00')).toBeInTheDocument()
    expect(screen.getByText('£200.00')).toBeInTheDocument()
  })
})
