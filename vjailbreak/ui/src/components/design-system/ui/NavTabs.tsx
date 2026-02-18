import { Tabs, TabsProps, Tab, TabProps, Box, Typography, Chip } from '@mui/material'
import { ReactNode } from 'react'

export interface NavTabsProps extends TabsProps {
  'data-testid'?: string
}

export interface NavTabProps extends Omit<TabProps, 'label'> {
  label: ReactNode
  description?: ReactNode
  count?: number
  'data-testid'?: string
}

export function NavTabs({
  children,
  'data-testid': dataTestId = 'nav-tabs',
  ...props
}: NavTabsProps) {
  return (
    <Tabs
      variant="scrollable"
      allowScrollButtonsMobile
      TabIndicatorProps={{ sx: { height: 3, borderRadius: 99 } }}
      data-testid={dataTestId}
      {...props}
    >
      {children}
    </Tabs>
  )
}

export function NavTab({
  label,
  description,
  count,
  'data-testid': dataTestId = 'nav-tab',
  ...props
}: NavTabProps) {
  const labelContent = (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
      <Box sx={{ textAlign: 'left' }}>
        <Typography variant="subtitle2" component="div">
          {label}
        </Typography>
        {description ? (
          <Typography variant="caption" color="text.secondary" component="div">
            {description}
          </Typography>
        ) : null}
      </Box>
      {typeof count === 'number' ? (
        <Chip size="small" label={count} color="default" variant="outlined" />
      ) : null}
    </Box>
  )

  return <Tab {...props} label={labelContent} data-testid={dataTestId} />
}
