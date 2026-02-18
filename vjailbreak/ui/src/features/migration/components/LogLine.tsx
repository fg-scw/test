import { useMemo } from 'react'
import { Box, useTheme } from '@mui/material'

interface LogLineProps {
  log: string
  index: number
  showBorder: boolean
  isDarkMode: boolean
}

const LEVEL_REGEX =
  /\b(ERROR|FATAL|WARN|WARNING|INFO|DEBUG|TRACE|SUCCESS|SUCCEEDED|FAILED|FAILURE)\b/i

export const extractLogLevel = (line: string): string | null => {
  const match = line.match(LEVEL_REGEX)
  return match ? match[1].toUpperCase() : null
}

// Log level colors for dark mode
const DARK_COLORS = {
  error: '#ff5252', // Bright red
  fatal: '#d32f2f', // Dark red
  warn: '#ffb74d', // Orange
  warning: '#ffb74d',
  info: '#4fc3f7', // Light blue
  debug: '#81c784', // Green
  trace: '#9575cd', // Purple
  success: '#66bb6a', // Green
  timestamp: '#78909c', // Blue grey
  bracket: '#546e7a', // Dark blue grey
  string: '#aed581', // Light green
  number: '#ffd54f', // Yellow
  keyword: '#ba68c8', // Purple
  podName: '#64b5f6', // Sky blue
  default: '#e0e0e0' // Light grey
}

// Log level colors for light mode
const LIGHT_COLORS = {
  error: '#d32f2f', // Red
  fatal: '#b71c1c', // Dark red
  warn: '#f57c00', // Orange
  warning: '#f57c00',
  info: '#0288d1', // Blue
  debug: '#388e3c', // Green
  trace: '#7b1fa2', // Purple
  success: '#2e7d32', // Dark green
  timestamp: '#546e7a', // Blue grey
  bracket: '#37474f', // Dark grey
  string: '#558b2f', // Dark green
  number: '#f9a825', // Dark yellow
  keyword: '#6a1b9a', // Purple
  podName: '#1976d2', // Dark blue
  default: '#212121' // Dark grey
}

export default function LogLine({ log, showBorder, isDarkMode }: LogLineProps) {
  const theme = useTheme()
  const colors = isDarkMode ? DARK_COLORS : LIGHT_COLORS

  const parseLogLine = useMemo(() => {
    const segments: Array<{ text: string; color: string; bold?: boolean }> = []

    const timestampMatch = log.match(
      /^\d{4}[\/\-]\d{2}[\/\-]\d{2}[T\s]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})?/
    )
    let remaining = log

    if (timestampMatch) {
      segments.push({ text: timestampMatch[0], color: colors.timestamp })
      remaining = log.substring(timestampMatch[0].length)
    }

    const podNameMatch = remaining.match(/^\s*\[([^\]]+)\]/)
    if (podNameMatch) {
      segments.push({ text: ' [', color: colors.bracket })
      segments.push({ text: podNameMatch[1], color: colors.podName })
      segments.push({ text: ']', color: colors.bracket })
      remaining = remaining.substring(podNameMatch[0].length)
    }

    const logLevelMatch = remaining.match(
      /^\s*(ERROR|FATAL|WARN|WARNING|INFO|DEBUG|TRACE|FAILED|FAILURE|SUCCESS|SUCCEEDED)\b/i
    )

    if (logLevelMatch) {
      const level = logLevelMatch[1].toUpperCase()
      const beforeLevel = remaining.substring(0, logLevelMatch.index || 0)
      const afterLevel = remaining.substring((logLevelMatch.index || 0) + logLevelMatch[0].length)

      if (beforeLevel) {
        segments.push({ text: beforeLevel, color: colors.default })
      }

      // Determine log level color
      let levelColor = colors.default
      if (level.includes('ERROR') || level.includes('FAIL')) {
        levelColor = colors.error
      } else if (level === 'FATAL') {
        levelColor = colors.fatal
      } else if (level.includes('WARN')) {
        levelColor = colors.warn
      } else if (level === 'INFO') {
        levelColor = colors.info
      } else if (level === 'DEBUG') {
        levelColor = colors.debug
      } else if (level === 'TRACE') {
        levelColor = colors.trace
      } else if (level.includes('SUCCESS') || level.includes('SUCCEED')) {
        levelColor = colors.success
      }

      const leadingSpaces = logLevelMatch[0].substring(
        0,
        logLevelMatch[0].length - logLevelMatch[1].length
      )
      if (leadingSpaces) {
        segments.push({ text: leadingSpaces, color: colors.default })
      }

      segments.push({ text: logLevelMatch[1], color: levelColor, bold: true })

      if (afterLevel) {
        segments.push({ text: afterLevel, color: colors.default })
      }
    } else {
      if (remaining) {
        segments.push({ text: remaining, color: colors.default })
      }
    }

    return segments
  }, [log, colors])

  return (
    <Box
      sx={{
        borderBottom: showBorder ? `1px solid ${theme.palette.divider}` : 'none',
        py: 0.5,
        px: 1,
        fontFamily: 'monospace',
        fontSize: '0.875rem',
        lineHeight: 1.6,
        wordBreak: 'break-word',
        whiteSpace: 'pre-wrap',
        '&:hover': {
          backgroundColor: isDarkMode ? 'rgba(255, 255, 255, 0.05)' : 'rgba(0, 0, 0, 0.02)'
        }
      }}
    >
      {parseLogLine.map((segment, idx) => (
        <span
          key={idx}
          style={{
            color: segment.color,
            fontWeight: segment.bold ? 'bold' : 'normal'
          }}
        >
          {segment.text}
        </span>
      ))}
    </Box>
  )
}
