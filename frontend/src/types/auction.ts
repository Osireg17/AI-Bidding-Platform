export type AuctionStatus = 'active' | 'ending_soon' | 'closed'

export interface Auction {
  id: number
  title: string
  description: string
  start_price: number
  current_price: number
  status: AuctionStatus
  end_time: string // ISO 8601
}

export interface Bid {
  bot_name: string
  bot_id: number
  amount: number
  timestamp: string
}

export interface Winner {
  bot_name: string
  bot_id: number
  amount: number
  final_status: 'sold' | 'unsold'
}

export interface AuctionState {
  auction: Auction | null
  bids: Bid[]
  winner: Winner | null
}
