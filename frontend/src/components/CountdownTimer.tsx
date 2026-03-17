import { useEffect, useState } from 'react'

interface CountdownTimerProps {
  endTime: string
  urgent?: boolean
}

function formatDuration(ms: number): string {
  if (ms <= 0) return '00:00'
  const totalSeconds = Math.floor(ms / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60

  if (hours > 0) {
    return `${String(hours).padStart(2, '0')}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
  }
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
}

export function CountdownTimer({ endTime, urgent }: CountdownTimerProps) {
  const [remaining, setRemaining] = useState(() => new Date(endTime).getTime() - Date.now())

  useEffect(() => {
    setRemaining(new Date(endTime).getTime() - Date.now())
    const id = setInterval(() => {
      setRemaining(new Date(endTime).getTime() - Date.now())
    }, 1000)
    return () => clearInterval(id)
  }, [endTime])

  if (remaining <= 0) {
    return (
      <span
        style={{
          fontFamily: "'DM Mono', monospace",
          fontSize: 13,
          color: 'rgb(80, 78, 74)',
        }}
      >
        Ended
      </span>
    )
  }

  return (
    <span
      style={{
        fontFamily: "'DM Mono', monospace",
        fontSize: 22,
        fontWeight: 500,
        letterSpacing: '-0.02em',
        lineHeight: 1,
        color: urgent ? 'rgb(220, 80, 60)' : 'rgb(230, 228, 220)',
        animation: urgent ? 'tick-urgent 1s ease-in-out infinite' : undefined,
        tabularNums: 'tabular-nums',
      } as React.CSSProperties}
    >
      {formatDuration(remaining)}
    </span>
  )
}
