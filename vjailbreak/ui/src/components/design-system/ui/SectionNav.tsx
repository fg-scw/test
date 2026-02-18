import { Box, List, ListItemButton, Tooltip, Typography } from '@mui/material'
import CheckIcon from '@mui/icons-material/Check'
import { alpha, type Theme } from '@mui/material/styles'
import { ReactNode } from 'react'

export type SectionNavStatus = 'complete' | 'attention' | 'incomplete' | 'optional'

export interface SectionNavItem {
  id: string
  title: ReactNode
  description?: ReactNode
  status?: SectionNavStatus
}

export interface SectionNavProps {
  items: SectionNavItem[]
  activeId?: string
  onSelect: (id: string) => void
  dense?: boolean
  showDescriptions?: boolean
  'data-testid'?: string
}

type Status = SectionNavStatus | undefined

const getSelectedAccent = (theme: Theme, status: Status) => {
  switch (status) {
    case 'complete':
      return theme.palette.success.main
    case 'attention':
      return theme.palette.error.main
    case 'optional':
      return theme.palette.grey[500]
    default:
      return theme.palette.grey[500]
  }
}

const getStatusChipSx = (status: Status) => {
  switch (status) {
    case 'complete':
      return {
        backgroundColor: (theme: Theme) => theme.palette.success.light,
        color: (theme: Theme) => theme.palette.success.contrastText
      }
    case 'attention':
      return {
        backgroundColor: (theme: Theme) => theme.palette.error.light,
        color: (theme: Theme) => theme.palette.error.contrastText
      }
    case 'optional':
      return {
        backgroundColor: (theme: Theme) =>
          alpha(theme.palette.text.primary, theme.palette.mode === 'dark' ? 0.12 : 0.06),
        color: (theme: Theme) => theme.palette.text.primary
      }
    default:
      return {
        backgroundColor: (theme: Theme) => theme.palette.info.light,
        color: (theme: Theme) => theme.palette.info.contrastText
      }
  }
}

const getSelectedChipSx = (theme: Theme, status: Status) => {
  const c = getSelectedAccent(theme, status)
  return {
    transform: 'scale(1.07)',
    boxShadow: `0 0 0 1px ${alpha(c, 0.22)}, 0 8px 18px ${alpha(theme.palette.common.white, 0.18)}`
  }
}

export default function SectionNav({
  items,
  activeId,
  onSelect,
  dense = false,
  showDescriptions = true,
  'data-testid': dataTestId = 'section-nav'
}: SectionNavProps) {
  const width = dense ? 56 : 68
  const listPaddingY = dense ? 0.5 : 1
  const headerPaddingY = dense ? 0.75 : 1
  const buttonPaddingY = dense ? 0.75 : 1
  const chipSize = dense ? 30 : 34
  const iconSize = dense ? 16 : 18
  const connectorHeight = dense ? 14 : 18
  const connectorMarginY = dense ? 0.25 : 0.5

  return (
    <Box
      data-testid={dataTestId}
      sx={{
        position: 'sticky',
        top: 0,
        alignSelf: 'start',
        width
      }}
    >
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'center',
          py: headerPaddingY
        }}
      >
        <Typography
          variant="caption"
          sx={{
            fontWeight: 700,
            letterSpacing: '0.08em',
            textTransform: 'uppercase',
            color: 'text.secondary',
            lineHeight: 1
          }}
        >
          Steps
        </Typography>
      </Box>
      <List dense disablePadding sx={{ py: listPaddingY }}>
        {items.map((item, index) => {
          const selected = item.id === activeId
          const stepNumber = index + 1
          const isLast = index === items.length - 1

          const tooltipTitle = (
            <Box sx={{ py: 0.25 }}>
              <Typography variant="subtitle2" sx={{ lineHeight: 1.2 }}>
                {item.title}
              </Typography>
              {showDescriptions && item.description ? (
                <Typography variant="caption" color="text.secondary" sx={{ lineHeight: 1.2 }}>
                  {item.description}
                </Typography>
              ) : null}
            </Box>
          )

          return (
            <Box
              key={item.id}
              sx={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center'
              }}
            >
              <Tooltip title={tooltipTitle} placement="right" arrow>
                <ListItemButton
                  selected={selected}
                  onClick={() => onSelect(item.id)}
                  aria-label={`Step ${stepNumber}`}
                  sx={{
                    display: 'flex',
                    justifyContent: 'center',
                    py: buttonPaddingY,
                    px: 0,
                    height: dense ? 30 : 34,
                    borderRadius: 2,
                    '&.Mui-selected': {
                      backgroundColor: (theme) => theme.palette.action.selected
                    },
                    '&:hover': {
                      backgroundColor: (theme) => theme.palette.action.hover
                    }
                  }}
                >
                  <Box
                    sx={(theme) => ({
                      width: chipSize,
                      height: chipSize,
                      borderRadius: 999,
                      display: 'grid',
                      placeItems: 'center',
                      fontVariantNumeric: 'tabular-nums',
                      transition: 'transform 140ms ease, box-shadow 140ms ease, filter 140ms ease',
                      ...getStatusChipSx(item.status),
                      ...(selected ? getSelectedChipSx(theme, item.status) : {}),
                      '&:hover': {
                        filter: 'brightness(0.98)'
                      }
                    })}
                  >
                    {item.status === 'complete' ? (
                      <CheckIcon sx={{ fontSize: iconSize }} />
                    ) : (
                      <Typography
                        variant={dense ? 'caption' : 'body2'}
                        sx={{ fontWeight: 700, lineHeight: 1 }}
                      >
                        {stepNumber}
                      </Typography>
                    )}
                  </Box>
                </ListItemButton>
              </Tooltip>
              {!isLast ? (
                <Box
                  aria-hidden
                  sx={{
                    width: 2,
                    height: connectorHeight,
                    borderRadius: 999,
                    backgroundColor: (theme) =>
                      alpha(
                        theme.palette.text.primary,
                        theme.palette.mode === 'dark' ? 0.22 : 0.16
                      ),
                    my: connectorMarginY
                  }}
                />
              ) : null}
            </Box>
          )
        })}
      </List>
    </Box>
  )
}
