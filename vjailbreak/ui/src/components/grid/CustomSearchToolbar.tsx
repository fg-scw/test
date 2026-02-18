import { Box, IconButton, Tooltip, Typography, Menu, MenuItem } from '@mui/material'
import { GridToolbarQuickFilter } from '@mui/x-data-grid'
import {
  RefreshRounded,
  FilterList as FilterListIcon,
  CalendarToday as CalendarIcon,
  Close as CloseIcon
} from '@mui/icons-material'
import { useState } from 'react'

interface CustomSearchToolbarProps {
  title?: string
  onRefresh?: () => void
  disableRefresh?: boolean
  placeholder?: string
  onStatusFilterChange?: (filter: string) => void
  currentStatusFilter?: string
  onDateFilterChange?: (filter: string) => void
  currentDateFilter?: string
}

const CustomSearchToolbar = ({
  title,
  onRefresh,
  disableRefresh = false,
  placeholder = 'Search',
  onStatusFilterChange,
  currentStatusFilter = 'All',
  onDateFilterChange,
  currentDateFilter = 'All Time'
}: CustomSearchToolbarProps) => {
  const [statusAnchorEl, setStatusAnchorEl] = useState<null | HTMLElement>(null)
  const [dateAnchorEl, setDateAnchorEl] = useState<null | HTMLElement>(null)

  const statusMenuOpen = Boolean(statusAnchorEl)
  const dateMenuOpen = Boolean(dateAnchorEl)

  const statusFilterOptions = ['All', 'In Progress', 'Succeeded', 'Failed']
  const dateFilterOptions = ['All Time', 'Last 24 hours', 'Last 7 days', 'Last 30 days']

  const isDateFilterActive = currentDateFilter !== 'All Time'
  const isStatusFilterActive = currentStatusFilter !== 'All'

  const handleMenuClick = (event: React.MouseEvent<HTMLElement>, menu: 'status' | 'date') => {
    if (menu === 'status') setStatusAnchorEl(event.currentTarget)
    if (menu === 'date') setDateAnchorEl(event.currentTarget)
  }

  const handleMenuClose = () => {
    setStatusAnchorEl(null)
    setDateAnchorEl(null)
  }

  const handleFilterSelect = (filter: string, menu: 'status' | 'date') => {
    if (menu === 'status') onStatusFilterChange?.(filter)
    if (menu === 'date') onDateFilterChange?.(filter)
    handleMenuClose()
  }

  return (
    <Box
      sx={{
        p: 1,
        display: 'flex',
        alignItems: 'center',
        marginLeft: 2,
        marginRight: 2
      }}
    >
      {title && <Typography variant="h6">{title}</Typography>}
      <Box sx={{ marginLeft: 'auto', display: 'flex', gap: 1, alignItems: 'center' }}>
        <Box sx={{ display: 'flex', alignItems: 'center' }}>
          {onRefresh && (
            <Tooltip title="Refresh">
              <span>
                <IconButton onClick={onRefresh} disabled={disableRefresh} size="small">
                  <RefreshRounded />
                </IconButton>
              </span>
            </Tooltip>
          )}
          {onDateFilterChange && (
            <>
              <Tooltip title="Filter by creation date">
                <IconButton onClick={(e) => handleMenuClick(e, 'date')} size="small">
                  <CalendarIcon
                    fontSize="small"
                    color={isDateFilterActive ? 'primary' : 'inherit'}
                  />
                </IconButton>
              </Tooltip>
              <Menu anchorEl={dateAnchorEl} open={dateMenuOpen} onClose={handleMenuClose}>
                {isDateFilterActive && (
                  <MenuItem onClick={() => handleFilterSelect('All Time', 'date')}>
                    <CloseIcon fontSize="small" sx={{ mr: 1 }} />
                    Clear filter
                  </MenuItem>
                )}
                {dateFilterOptions.map((option) => (
                  <MenuItem
                    key={option}
                    selected={option === currentDateFilter}
                    onClick={() => handleFilterSelect(option, 'date')}
                  >
                    {option}
                  </MenuItem>
                ))}
              </Menu>
            </>
          )}
          {onStatusFilterChange && (
            <>
              <Tooltip title="Filter by status">
                <IconButton onClick={(e) => handleMenuClick(e, 'status')} size="small">
                  <FilterListIcon color={isStatusFilterActive ? 'primary' : 'inherit'} />
                </IconButton>
              </Tooltip>
              <Menu anchorEl={statusAnchorEl} open={statusMenuOpen} onClose={handleMenuClose}>
                {isStatusFilterActive && (
                  <MenuItem onClick={() => handleFilterSelect('All', 'status')}>
                    <CloseIcon fontSize="small" sx={{ mr: 1 }} />
                    Clear filter
                  </MenuItem>
                )}
                {statusFilterOptions.map((option) => (
                  <MenuItem
                    key={option}
                    selected={option === currentStatusFilter}
                    onClick={() => handleFilterSelect(option, 'status')}
                  >
                    {option}
                  </MenuItem>
                ))}
              </Menu>
            </>
          )}
        </Box>
        <Box sx={{ maxWidth: '300px' }}>
          <div>
            <GridToolbarQuickFilter
              placeholder={placeholder}
              sx={{
                '& .MuiInputBase-input': {
                  textOverflow: 'ellipsis'
                }
              }}
            />
          </div>
        </Box>
      </Box>
    </Box>
  )
}

export default CustomSearchToolbar
