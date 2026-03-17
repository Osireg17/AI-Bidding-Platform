import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { WinnerBanner } from '@/components/WinnerBanner'
import type { Winner } from '@/types/auction'

const soldWinner: Winner = {
  bot_name: 'Aggressive Alice',
  bot_id: 1,
  amount: 350,
  final_status: 'sold',
}

const unsoldWinner: Winner = {
  bot_name: 'Aggressive Alice',
  bot_id: 1,
  amount: 50,
  final_status: 'unsold',
}

describe('WinnerBanner', () => {
  it('renders nothing when winner is null', () => {
    const { container } = render(<WinnerBanner winner={null} />)
    expect(container).toBeEmptyDOMElement()
  })

  it('shows sold message with winner name and amount', () => {
    render(<WinnerBanner winner={soldWinner} />)
    expect(screen.getByText(/auction result.*sold/i)).toBeInTheDocument()
    expect(screen.getByText(/aggressive alice/i)).toBeInTheDocument()
    expect(screen.getByText(/£350\.00/)).toBeInTheDocument()
  })

  it('shows unsold message when final_status is unsold', () => {
    render(<WinnerBanner winner={unsoldWinner} />)
    expect(screen.getByText(/auction result.*unsold/i)).toBeInTheDocument()
    expect(screen.getByText(/£50\.00/)).toBeInTheDocument()
  })
})
