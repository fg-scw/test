import Box from '@mui/material/Box'
import Paper from '@mui/material/Paper'
import Button from '@mui/material/Button'
import { cleanupAllResources } from 'src/api/helpers'
import MenuItem from '@mui/material/MenuItem'
import Menu from '@mui/material/Menu'
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown'
import MigrationIcon from '@mui/icons-material/SwapHoriz'
import ClusterIcon from '@mui/icons-material/Hub'
import { useState } from 'react'
import ThemeToggle from './ThemeToggle'

export default function ButtonAppBar({ setOpenMigrationForm, hide = false }) {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)

  const handleMenuClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget)
  }

  const handleMenuClose = () => {
    setAnchorEl(null)
  }

  const handleMigrationSelect = (type: string) => {
    setOpenMigrationForm(true, type)
    handleMenuClose()
  }

  return (
    <Paper
      elevation={0}
      square
      sx={{
        visibility: hide ? 'hidden' : 'visible',
        position: 'sticky',
        top: 0,
        zIndex: (theme) => theme.zIndex.appBar,
        borderBottom: (theme) => `1px solid ${theme.palette.divider}`,
        backgroundColor: 'background.paper'
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'flex-end',
          gap: 1.5,
          height: 64,
          px: { xs: 2, sm: 3 },
          maxWidth: { lg: '1600px' },
          mx: 'auto',
          width: '100%'
        }}
      >
        <ThemeToggle />
        {import.meta.env.MODE === 'development' && (
          <Button
            size="medium"
            onClick={() => cleanupAllResources()}
            color="error"
            variant="outlined"
          >
            DEV ONLY: Clean Up Resources
          </Button>
        )}
        <Button
          size="medium"
          onClick={handleMenuClick}
          color="primary"
          variant="contained"
          endIcon={<KeyboardArrowDownIcon />}
        >
          Start Migration
        </Button>
        <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleMenuClose}>
          <MenuItem onClick={() => handleMigrationSelect('standard')}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <MigrationIcon fontSize="small" />
              <span>Start Migration</span>
            </Box>
          </MenuItem>
          <MenuItem onClick={() => handleMigrationSelect('rolling')}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <ClusterIcon fontSize="small" />
              <span>Start Cluster Conversion</span>
            </Box>
          </MenuItem>
        </Menu>
      </Box>
    </Paper>
  )
}
