import { IconButton, Tooltip, useTheme, Menu, MenuItem } from '@mui/material'
import Brightness4Icon from '@mui/icons-material/Brightness4'
import Brightness7Icon from '@mui/icons-material/Brightness7'
import ComputerIcon from '@mui/icons-material/Computer'
import { useThemeContext } from 'src/theme/ThemeContext'
import { useState, MouseEvent } from 'react'

export default function ThemeToggle() {
  const { mode, toggleTheme, resetToSystemTheme, isUsingSystemTheme } = useThemeContext()
  const theme = useTheme()
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)

  const handleContextMenu = (event: MouseEvent<HTMLButtonElement>) => {
    event.preventDefault()
    setAnchorEl(event.currentTarget)
  }

  const handleClose = () => {
    setAnchorEl(null)
  }

  const handleResetToSystem = () => {
    resetToSystemTheme()
    handleClose()
  }

  return (
    <>
      <Tooltip
        title={
          isUsingSystemTheme
            ? 'Using system theme preference'
            : `Toggle ${mode === 'light' ? 'dark' : 'light'} theme (right-click for more options)`
        }
      >
        <IconButton
          onClick={toggleTheme}
          onContextMenu={handleContextMenu}
          color="inherit"
          sx={{ ml: 1 }}
          aria-label="toggle theme"
        >
          {isUsingSystemTheme ? (
            <ComputerIcon sx={{ color: theme.palette.text.primary }} />
          ) : mode === 'light' ? (
            <Brightness4Icon sx={{ color: theme.palette.text.primary }} />
          ) : (
            <Brightness7Icon sx={{ color: theme.palette.text.primary }} />
          )}
        </IconButton>
      </Tooltip>

      <Menu anchorEl={anchorEl} open={Boolean(anchorEl)} onClose={handleClose}>
        <MenuItem onClick={toggleTheme}>
          Switch to {mode === 'light' ? 'Dark' : 'Light'} Mode
        </MenuItem>
        <MenuItem onClick={handleResetToSystem} disabled={isUsingSystemTheme}>
          Use System Theme
        </MenuItem>
      </Menu>
    </>
  )
}
