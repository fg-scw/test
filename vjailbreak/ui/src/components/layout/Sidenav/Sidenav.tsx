import {
  Drawer,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  IconButton,
  Box,
  Chip,
  Tooltip,
  Divider,
  styled,
  alpha
} from '@mui/material'
import { ChevronLeft, ChevronRight, OpenInNew } from '@mui/icons-material'
import { useState, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { NavigationItem, SidenavProps } from 'src/types/navigation'
import { useVersionQuery } from 'src/hooks/api/useVersionQuery'
import Platform9Logo from '../Platform9Logo'
import { UpgradeModal } from 'src/features/migration/components'
import UpgradeIcon from '@mui/icons-material/Upgrade'
import Button from '@mui/material/Button'
import Typography from '@mui/material/Typography'
import { useTheme } from '@mui/material/styles'
import { useMigrationsQuery } from 'src/hooks/api/useMigrationsQuery'
import { Phase } from 'src/api/migrations/model'
import { FIVE_SECONDS, THIRTY_SECONDS } from 'src/constants'

const DRAWER_WIDTH = 280
const DRAWER_WIDTH_COLLAPSED = 72

const StyledDrawer = styled(Drawer, {
  shouldForwardProp: (prop) => prop !== 'collapsed'
})<{ collapsed: boolean }>(({ theme, collapsed }) => ({
  width: collapsed ? DRAWER_WIDTH_COLLAPSED : DRAWER_WIDTH,
  flexShrink: 0,
  whiteSpace: 'nowrap',
  boxSizing: 'border-box',
  '& .MuiDrawer-paper': {
    width: collapsed ? DRAWER_WIDTH_COLLAPSED : DRAWER_WIDTH,
    transition: theme.transitions.create('width', {
      easing: theme.transitions.easing.sharp,
      duration: theme.transitions.duration.enteringScreen
    }),
    overflow: 'visible',
    borderRight: `1px solid ${theme.palette.divider}`,
    backgroundColor: theme.palette.background.paper,
    '&::-webkit-scrollbar': {
      display: 'none'
    },
    scrollbarWidth: 'none',
    msOverflowStyle: 'none'
  }
}))

const DrawerHeader = styled('div')(({ theme }) => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  padding: theme.spacing(1.5, 2),
  ...theme.mixins.toolbar,
  minHeight: '64px !important',
  borderBottom: `1px solid ${alpha(theme.palette.divider, 0.12)}`,
  marginBottom: theme.spacing(1),
  position: 'relative'
}))

const BrandContainer = styled(Box, {
  shouldForwardProp: (prop) => prop !== 'collapsed'
})<{ collapsed?: boolean }>(({ theme, collapsed }) => ({
  display: 'flex',
  alignItems: 'center',
  justifyContent: collapsed ? 'center' : 'flex-start',
  width: '100%',
  paddingLeft: collapsed ? 0 : theme.spacing(2),
  transition: theme.transitions.create(['justify-content', 'padding-left'], {
    easing: theme.transitions.easing.sharp,
    duration: theme.transitions.duration.enteringScreen
  })
}))

const VersionBadge = styled(Box, {
  shouldForwardProp: (prop) => prop !== 'collapsed'
})<{ collapsed?: boolean }>(({ theme, collapsed }) => ({
  fontSize: '0.8rem',
  color: alpha(theme.palette.text.secondary, 0.6),
  fontWeight: 400,
  width: collapsed ? '60px' : 'auto',
  textAlign: 'center',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
  padding: collapsed ? theme.spacing(0.5, 1) : 0
}))

const VersionDisplay = ({ collapsed }: { collapsed?: boolean }) => {
  const { data: versionInfo, isLoading, error } = useVersionQuery()

  if (isLoading) {
    const content = (
      <VersionBadge collapsed={collapsed}>{collapsed ? '...' : 'Loading version...'}</VersionBadge>
    )

    if (collapsed) {
      return (
        <Tooltip title="Loading version..." placement="right" arrow>
          {content}
        </Tooltip>
      )
    }
    return content
  }

  if (error) {
    const content = (
      <VersionBadge collapsed={collapsed}>
        {collapsed ? 'v?' : 'Version: Unable to load'}
      </VersionBadge>
    )

    if (collapsed) {
      return (
        <Tooltip title="Version: Unable to load" placement="right" arrow>
          {content}
        </Tooltip>
      )
    }
    return content
  }

  const content = (
    <VersionBadge collapsed={collapsed}>
      {collapsed ? `${versionInfo?.version || '?'}` : `Version: ${versionInfo?.version}`}
    </VersionBadge>
  )

  if (collapsed) {
    return (
      <Tooltip
        title={
          <Box>
            Version: {versionInfo?.version}
            {versionInfo?.upgradeAvailable && versionInfo?.upgradeVersion && (
              <Box component="span" sx={{ display: 'block', fontSize: '0.85rem', mt: 0.5 }}>
                Update available: {versionInfo.upgradeVersion}
              </Box>
            )}
          </Box>
        }
        placement="right"
        arrow
      >
        {content}
      </Tooltip>
    )
  }

  return content
}

const StyledListItemButton = styled(ListItemButton, {
  shouldForwardProp: (prop) => prop !== 'active' && prop !== 'collapsed'
})<{ active?: boolean; collapsed?: boolean }>(({ theme, active, collapsed }) => ({
  minHeight: 48,
  margin: collapsed ? theme.spacing(0.5, 1) : theme.spacing(0.5, 1),
  borderRadius: theme.spacing(1),
  paddingLeft: collapsed ? 'auto' : theme.spacing(2),
  paddingRight: collapsed ? 'auto' : theme.spacing(2),
  justifyContent: collapsed ? 'center' : 'initial',
  position: 'relative',
  overflow: 'hidden',
  cursor: 'pointer',
  transition: theme.transitions.create(['background-color', 'box-shadow'], {
    duration: theme.transitions.duration.shorter
  }),
  ...(active && {
    backgroundColor: alpha(theme.palette.primary.main, 0.08),
    color: theme.palette.primary.main,
    '& .MuiTypography-root': {
      fontWeight: 600
    },
    '&::before': {
      content: '""',
      position: 'absolute',
      left: 0,
      top: 0,
      bottom: 0,
      width: 3,
      backgroundColor: theme.palette.primary.main
    },
    '& .MuiListItemIcon-root': {
      color: theme.palette.primary.main
    }
  }),
  '&:hover': {
    backgroundColor: alpha(theme.palette.primary.main, 0.05)
  },
  '&.Mui-focusVisible': {
    boxShadow: `0 0 0 2px ${alpha(theme.palette.primary.main, 0.25)}`
  }
}))

const StyledListItemIcon = styled(ListItemIcon, {
  shouldForwardProp: (prop) => prop !== 'collapsed'
})<{ collapsed?: boolean }>(({ collapsed }) => ({
  minWidth: collapsed ? 0 : 56,
  justifyContent: 'center'
}))

const NavigationBadge = styled(Chip)(({ theme }) => ({
  fontSize: '0.6rem',
  height: '16px',
  fontWeight: 600,
  marginLeft: theme.spacing(1),
  transform: 'translateY(-6px)',
  px: 0.75,
  lineHeight: '16px',
  display: 'flex',
  alignItems: 'center'
}))

const CornerToggleButton = styled(IconButton)(({ theme }) => ({
  position: 'absolute',
  right: -18,
  bottom: -18,
  width: 32,
  height: 32,
  borderRadius: '50%',
  backgroundColor: theme.palette.background.paper,
  border: `1px solid ${theme.palette.divider}`,
  color: theme.palette.text.secondary,
  boxShadow: theme.shadows[1],
  transition: theme.transitions.create(
    ['background-color', 'border-color', 'box-shadow', 'color'],
    {
      duration: theme.transitions.duration.shorter
    }
  ),
  '&:hover': {
    backgroundColor: alpha(theme.palette.primary.main, theme.palette.mode === 'dark' ? 0.18 : 0.12),
    borderColor: alpha(theme.palette.primary.main, theme.palette.mode === 'dark' ? 0.55 : 0.4),
    color: theme.palette.primary.main,
    boxShadow: theme.shadows[3]
  },
  '&:active': {
    backgroundColor: alpha(theme.palette.primary.main, theme.palette.mode === 'dark' ? 0.24 : 0.16),
    borderColor: alpha(theme.palette.primary.main, theme.palette.mode === 'dark' ? 0.65 : 0.5),
    boxShadow: theme.shadows[2]
  },
  '&.Mui-focusVisible': {
    boxShadow: `0 0 0 2px ${alpha(theme.palette.primary.main, 0.25)}`
  }
}))

interface NavigationItemProps {
  item: NavigationItem
  isActive: boolean
  isCollapsed: boolean
  onClick: (item: NavigationItem) => void
}

function NavigationItemComponent({ item, isActive, isCollapsed, onClick }: NavigationItemProps) {
  const handleClick = () => {
    if (!item.disabled) {
      onClick(item)
    }
  }

  const listItemContent = (
    <StyledListItemButton
      active={isActive}
      collapsed={isCollapsed}
      onClick={handleClick}
      disabled={item.disabled}
      data-tour={`nav-${item.id}`}
    >
      {item.icon && <StyledListItemIcon collapsed={isCollapsed}>{item.icon}</StyledListItemIcon>}
      {!isCollapsed && (
        <ListItemText
          primary={
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                {item.label}
                {item.badge && (
                  <NavigationBadge
                    label={item.badge.label}
                    size="small"
                    color={item.badge.color}
                    variant={item.badge.variant}
                  />
                )}
              </Box>
              {item.external && <OpenInNew sx={{ fontSize: '0.875rem', opacity: 0.7 }} />}
            </Box>
          }
          sx={{
            opacity: isCollapsed ? 0 : 1,
            '& .MuiTypography-root': {
              fontSize: '0.875rem'
            }
          }}
        />
      )}
    </StyledListItemButton>
  )

  if (isCollapsed && (item.label || item.badge)) {
    return (
      <Tooltip
        title={
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            {item.label}
            {item.badge && (
              <NavigationBadge
                label={item.badge.label}
                size="small"
                color={item.badge.color}
                variant={item.badge.variant}
              />
            )}
            {item.external && <OpenInNew sx={{ fontSize: '0.75rem' }} />}
          </Box>
        }
        placement="right"
        arrow
      >
        <ListItem disablePadding sx={{ display: 'block' }}>
          {listItemContent}
        </ListItem>
      </Tooltip>
    )
  }

  return (
    <ListItem disablePadding sx={{ display: 'block' }}>
      {listItemContent}
    </ListItem>
  )
}

export default function Sidenav({
  items,
  isCollapsed: controlledCollapsed,
  onToggleCollapse,
  onItemClick,
  activeItem: controlledActiveItem
}: SidenavProps) {
  const navigate = useNavigate()
  const location = useLocation()
  const theme = useTheme()

  const [internalCollapsed, setInternalCollapsed] = useState(() => {
    const saved = localStorage.getItem('sidenav-collapsed')
    return saved ? JSON.parse(saved) : false
  })

  const isCollapsed = controlledCollapsed ?? internalCollapsed
  const activeItem = controlledActiveItem ?? location.pathname

  const { data: versionInfo } = useVersionQuery()
  const [isUpgradeModalOpen, setIsUpgradeModalOpen] = useState(false)
  const { data: migrations } = useMigrationsQuery(undefined, {
    refetchInterval: (query) => {
      // if we're not on the migrations page, don't refetch
      if (activeItem !== '/dashboard/migrations') return Infinity
      const migrations = query?.state?.data || []
      const hasPendingMigration = !!migrations.find((m) => m.status === undefined)
      return hasPendingMigration ? FIVE_SECONDS : THIRTY_SECONDS
    },
    refetchOnMount: true
  })

  // checking if there's any active migrations
  const activePhases = new Set<Phase>([
    Phase.Pending,
    Phase.Validating,
    Phase.AwaitingDataCopyStart,
    Phase.CopyingBlocks,
    Phase.CopyingChangedBlocks,
    Phase.ConvertingDisk,
    Phase.AwaitingCutOverStartTime,
    Phase.AwaitingAdminCutOver,
    Phase.Unknown
  ])

  const hasActiveMigrations = Array.isArray(migrations)
    ? migrations.some((m) => activePhases.has(m.status?.phase as Phase))
    : false

  useEffect(() => {
    if (controlledCollapsed === undefined) {
      localStorage.setItem('sidenav-collapsed', JSON.stringify(internalCollapsed))
    }
  }, [internalCollapsed, controlledCollapsed])

  const handleToggleCollapse = () => {
    if (onToggleCollapse) {
      onToggleCollapse()
    } else {
      setInternalCollapsed((prev: boolean) => !prev)
    }
  }

  const handleItemClick = (item: NavigationItem) => {
    if ((window as any).__VDDK_UPLOAD_IN_PROGRESS__) {
      const confirmed = window.confirm(
        'File upload is in progress. If you leave this page, the upload may be interrupted. Do you want to continue?'
      )
      if (!confirmed) return
    }

    if (onItemClick) {
      onItemClick(item)
    } else if (item.external) {
      const url =
        item.externalUrl && item.externalUrl.trim()
          ? item.externalUrl
          : `https://${window.location.host}${item.path}`
      window.open(url, '_blank', 'noopener,noreferrer')
    } else {
      navigate(item.path)
    }
  }

  const getActiveItem = (currentPath: string): string => {
    const exactMatch = items.find((item) => item.path === currentPath)
    if (exactMatch) return exactMatch.path

    const partialMatch = items.find(
      (item) => currentPath.startsWith(item.path) && item.path !== '/dashboard'
    )
    return partialMatch?.path || currentPath
  }

  const currentActiveItem = getActiveItem(activeItem)

  const drawerContent = (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <DrawerHeader>
        <BrandContainer collapsed={isCollapsed}>
          <Platform9Logo collapsed={isCollapsed} />
        </BrandContainer>
        <Tooltip
          title={isCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          placement="right"
          arrow
        >
          <CornerToggleButton onClick={handleToggleCollapse} aria-label="toggle sidebar">
            {isCollapsed ? <ChevronRight fontSize="small" /> : <ChevronLeft fontSize="small" />}
          </CornerToggleButton>
        </Tooltip>
      </DrawerHeader>

      <List
        sx={{
          pt: 1,
          flex: 1,
          overflowY: 'auto',
          overscrollBehavior: 'contain',
          '&::-webkit-scrollbar': {
            display: 'none'
          },
          scrollbarWidth: 'none',
          msOverflowStyle: 'none'
        }}
      >
        {items
          .filter((item) => !item.hidden)
          .map((item) => (
            <Box key={item.id}>
              {item.id === 'monitoring' && <Divider sx={{ my: 1, mx: 2 }} />}
              <NavigationItemComponent
                item={item}
                isActive={currentActiveItem === item.path}
                isCollapsed={isCollapsed}
                onClick={handleItemClick}
              />
            </Box>
          ))}
      </List>

      <Box sx={{ mt: 'auto', px: 2, pb: 1.5, pt: 1.5, position: 'relative' }}>
        <Divider sx={{ mb: 1.5, opacity: 0.6 }} />
        <VersionDisplay collapsed={isCollapsed} />
        {versionInfo?.upgradeAvailable &&
          versionInfo?.upgradeVersion &&
          (isCollapsed ? (
            <Tooltip
              title={
                <Typography variant="body2">
                  {hasActiveMigrations
                    ? "Migrations are in progress, can't upgrade"
                    : 'Upgrade Available'}
                </Typography>
              }
              placement="right"
              arrow
              slotProps={{ tooltip: { sx: { ...theme.typography.body2 } } }}
            >
              <span>
                <Button
                  color="primary"
                  variant="contained"
                  sx={{ mt: 2, minWidth: 0, width: 40, height: 40, borderRadius: '50%' }}
                  onClick={() => setIsUpgradeModalOpen(true)}
                  disabled={hasActiveMigrations}
                >
                  <UpgradeIcon />
                </Button>
              </span>
            </Tooltip>
          ) : (
            <Tooltip
              title={
                hasActiveMigrations ? (
                  <Typography variant="body2">Migrations are in progress, can't upgrade</Typography>
                ) : (
                  ''
                )
              }
              placement="top"
              arrow
              disableHoverListener={!hasActiveMigrations}
              slotProps={{ tooltip: { sx: { ...theme.typography.body2 } } }}
            >
              <span>
                <Button
                  color="primary"
                  variant="contained"
                  fullWidth
                  sx={{ mt: 1, fontWeight: 600 }}
                  startIcon={<UpgradeIcon />}
                  onClick={() => setIsUpgradeModalOpen(true)}
                  disabled={hasActiveMigrations}
                >
                  Upgrade Available
                </Button>
              </span>
            </Tooltip>
          ))}
        <UpgradeModal show={isUpgradeModalOpen} onClose={() => setIsUpgradeModalOpen(false)} />
      </Box>
    </Box>
  )

  return (
    <StyledDrawer variant="permanent" collapsed={isCollapsed}>
      {drawerContent}
    </StyledDrawer>
  )
}
