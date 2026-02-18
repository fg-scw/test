import { Box, Tooltip, Typography } from '@mui/material'
import { ReactNode } from 'react'

export interface ClickableTableCellProps {
  children: ReactNode
  onClick?: () => void
  tooltipTitle?: ReactNode
  indicatorColor?: string
  hoverTextColor?: string
}

export default function ClickableTableCell({
  children,
  onClick,
  tooltipTitle,
  indicatorColor = 'primary.main',
  hoverTextColor = 'primary.main',
}: ClickableTableCellProps) {
  return (
    <Tooltip title={tooltipTitle} arrow>
      <Box
        sx={{
          width: '100%',
          height: '100%',
          display: 'flex',
          alignItems: 'center',
          pl: 1,
          cursor: onClick ? 'pointer' : 'default',
          position: 'relative',
          '&:hover': onClick
            ? {
                '& .click-indicator': { opacity: 1 },
                '& .vm-name-text': { color: hoverTextColor },
              }
            : {},
        }}
        onClick={(e) => {
          if (onClick) {
            e.stopPropagation()
            e.preventDefault()
            onClick()
          }
        }}
      >
        {onClick && (
          <Box
            className="click-indicator"
            sx={{
              position: 'absolute',
              left: 0,
              width: '3px',
              height: '60%',
              bgcolor: indicatorColor,
              borderRadius: '0 4px 4px 0',
              opacity: 0,
              transition: 'opacity 0.2s',
            }}
          />
        )}
        <Typography
          variant="body2"
          className="vm-name-text"
          sx={{ transition: 'color 0.2s' }}
        >
          {children}
        </Typography>
      </Box>
    </Tooltip>
  )
}
