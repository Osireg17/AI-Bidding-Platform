import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { AuctionCard } from '@/components/AuctionCard'
import type { Auction } from '@/types/auction'

const auction: Auction = {
  id: 1,
  title: 'Vintage Watch',
  description: 'A rare vintage timepiece.',
  start_price: 500,
  current_price: 750,
  status: 'active',
  end_time: new Date(Date.now() + 60_000).toISOString(),
}

describe('AuctionCard', () => {
  it('renders waiting placeholder when auction is null', () => {
    render(<AuctionCard auction={null} />)
    expect(screen.getByText(/awaiting next auction/i)).toBeInTheDocument()
  })

  it('renders auction title when auction is provided', () => {
    render(<AuctionCard auction={auction} />)
    expect(screen.getByText('Vintage Watch')).toBeInTheDocument()
  })

  it('formats start price as GBP', () => {
    render(<AuctionCard auction={auction} />)
    expect(screen.getByText('£500.00')).toBeInTheDocument()
  })

  it('formats current price as GBP', () => {
    render(<AuctionCard auction={auction} />)
    expect(screen.getByText('£750.00')).toBeInTheDocument()
  })
})
