import { createContext, useState, useContext, useMemo, useEffect, ReactNode } from 'react'
import { ThemeProvider as MuiThemeProvider, createTheme, alpha } from '@mui/material/styles'
import { CssBaseline } from '@mui/material'
import { grey, blue } from '@mui/material/colors'
import customTypography from './typography'
import {
  // Basic colors
  WHITE,
  BLACK,
  // Background colors
  DARK_BG_PRIMARY,
  DARK_BG_PAPER,
  LIGHT_BG_DEFAULT,
  // Text colors
  LIGHT_TEXT_PRIMARY,
  LIGHT_TEXT_SECONDARY,
  DARK_TEXT_PRIMARY,
  DARK_TEXT_SECONDARY,
  // UI Element colors - Dark mode
  DARK_DIVIDER,
  DARK_BORDER_LIGHT,
  DARK_BORDER_MEDIUM,
  DARK_HOVER_BG,
  DARK_HOVER_BG_STRONGER,
  DARK_BUTTON_BG,
  DARK_BUTTON_BG_HOVER,
  // UI Element colors - Light mode
  LIGHT_DIVIDER,
  // Scrollbar colors
  DARK_SCROLLBAR_BG,
  DARK_SCROLLBAR_THUMB,
  LIGHT_SCROLLBAR_BG,
  LIGHT_SCROLLBAR_THUMB,
  // Status colors
  DARK_ERROR,
  DARK_WARNING,
  DARK_INFO,
  DARK_SUCCESS,
  LIGHT_ERROR,
  LIGHT_WARNING,
  LIGHT_INFO,
  LIGHT_SUCCESS,
  // Secondary palette colors
  DARK_SECONDARY,
  DARK_SECONDARY_DARK,
  DARK_SECONDARY_LIGHT,
  LIGHT_SECONDARY,
  LIGHT_SECONDARY_DARK,
  LIGHT_SECONDARY_LIGHT
} from './colors'
import { useMediaQuery } from '@mui/material'

type ThemeMode = 'light' | 'dark'

interface ThemeContextType {
  mode: ThemeMode
  toggleTheme: () => void
  resetToSystemTheme: () => void
  isUsingSystemTheme: boolean
}

const ThemeContext = createContext<ThemeContextType>({
  mode: 'light',
  toggleTheme: () => {},
  resetToSystemTheme: () => {},
  isUsingSystemTheme: false
})

export const useThemeContext = () => useContext(ThemeContext)

interface ThemeProviderProps {
  children: ReactNode
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  // Get the system color scheme preference
  const prefersDarkMode = useMediaQuery('(prefers-color-scheme: dark)')

  // Track if we're using system theme
  const [isUsingSystemTheme, setIsUsingSystemTheme] = useState<boolean>(() => {
    return localStorage.getItem('themeMode') === null
  })

  // Try to get the theme from localStorage, fall back to system preference
  const [mode, setMode] = useState<ThemeMode>(() => {
    const savedMode = localStorage.getItem('themeMode')
    if (savedMode === 'light' || savedMode === 'dark') {
      return savedMode as ThemeMode
    }
    // Use system preference if no saved preference
    return prefersDarkMode ? 'dark' : 'light'
  })

  // Update theme when system preference changes (if using system theme)
  useEffect(() => {
    if (isUsingSystemTheme) {
      setMode(prefersDarkMode ? 'dark' : 'light')
    }
  }, [prefersDarkMode, isUsingSystemTheme])

  // Update localStorage when theme changes (if not using system theme)
  useEffect(() => {
    if (!isUsingSystemTheme) {
      localStorage.setItem('themeMode', mode)
    }
  }, [mode, isUsingSystemTheme])

  // Toggle between light and dark theme
  const toggleTheme = () => {
    setIsUsingSystemTheme(false)
    setMode((prevMode) => (prevMode === 'light' ? 'dark' : 'light'))
  }

  // Reset to system theme
  const resetToSystemTheme = () => {
    localStorage.removeItem('themeMode')
    setIsUsingSystemTheme(true)
    setMode(prefersDarkMode ? 'dark' : 'light')
  }

  // Create the theme based on current mode
  const theme = useMemo(
    () =>
      createTheme({
        spacing: 8,
        palette: {
          mode,
          primary: {
            main: mode === 'dark' ? blue[300] : blue[700],
            dark: mode === 'dark' ? blue[200] : blue[800],
            light: mode === 'dark' ? blue[400] : blue[500],
            contrastText: mode === 'dark' ? BLACK : WHITE
          },
          secondary: {
            main: mode === 'dark' ? DARK_SECONDARY : LIGHT_SECONDARY,
            dark: mode === 'dark' ? DARK_SECONDARY_DARK : LIGHT_SECONDARY_DARK,
            light: mode === 'dark' ? DARK_SECONDARY_LIGHT : LIGHT_SECONDARY_LIGHT,
            contrastText: WHITE
          },
          background: {
            default: mode === 'light' ? LIGHT_BG_DEFAULT : DARK_BG_PRIMARY,
            paper: mode === 'light' ? WHITE : DARK_BG_PAPER
          },
          divider: mode === 'dark' ? DARK_DIVIDER : LIGHT_DIVIDER,
          text: {
            primary: mode === 'dark' ? DARK_TEXT_PRIMARY : LIGHT_TEXT_PRIMARY,
            secondary: mode === 'dark' ? DARK_TEXT_SECONDARY : LIGHT_TEXT_SECONDARY
          },
          error: {
            main: mode === 'dark' ? DARK_ERROR : LIGHT_ERROR
          },
          warning: {
            main: mode === 'dark' ? DARK_WARNING : LIGHT_WARNING
          },
          info: {
            main: mode === 'dark' ? DARK_INFO : LIGHT_INFO
          },
          success: {
            main: mode === 'dark' ? DARK_SUCCESS : LIGHT_SUCCESS
          }
        },
        typography: {
          ...customTypography
        },
        components: {
          MuiPaper: {
            styleOverrides: {
              root: {
                backgroundImage: 'none',
                color: mode === 'dark' ? DARK_TEXT_PRIMARY : LIGHT_TEXT_PRIMARY
              }
            }
          },
          MuiOutlinedInput: {
            styleOverrides: {
              root: {
                '&:hover .MuiOutlinedInput-notchedOutline': {
                  borderColor: mode === 'dark' ? blue[300] : blue[700]
                }
              }
            }
          },
          MuiTableCell: {
            styleOverrides: {
              head: {
                backgroundColor: mode === 'light' ? grey[200] : grey[900]
              },
              body: {
                borderBottom: `1px solid ${mode === 'dark' ? DARK_DIVIDER : LIGHT_DIVIDER}`
              }
            }
          },
          MuiCssBaseline: {
            styleOverrides: {
              body: {
                scrollbarColor:
                  mode === 'dark'
                    ? `${DARK_SCROLLBAR_THUMB} ${DARK_SCROLLBAR_BG}`
                    : `${LIGHT_SCROLLBAR_THUMB} ${LIGHT_SCROLLBAR_BG}`,
                '&::-webkit-scrollbar, & *::-webkit-scrollbar': {
                  backgroundColor: mode === 'dark' ? DARK_SCROLLBAR_BG : LIGHT_SCROLLBAR_BG,
                  width: 8,
                  height: 8
                },
                '&::-webkit-scrollbar-thumb, & *::-webkit-scrollbar-thumb': {
                  borderRadius: 8,
                  backgroundColor: mode === 'dark' ? DARK_SCROLLBAR_THUMB : LIGHT_SCROLLBAR_THUMB,
                  minHeight: 24
                },
                '&::-webkit-scrollbar-corner, & *::-webkit-scrollbar-corner': {
                  backgroundColor: mode === 'dark' ? DARK_SCROLLBAR_BG : LIGHT_SCROLLBAR_BG
                }
              }
            }
          },
          MuiButton: {
            styleOverrides: {
              root: {
                ...(mode === 'dark' && {
                  '&.MuiButton-outlined': {
                    borderColor: DARK_BORDER_LIGHT
                  }
                })
              },
              outlined: {
                ...(mode === 'dark' && {
                  borderColor: DARK_BORDER_LIGHT,
                  '&:hover': {
                    backgroundColor: DARK_HOVER_BG
                  }
                })
              },
              containedSecondary: {
                ...(mode === 'dark' && {
                  backgroundColor: DARK_BUTTON_BG,
                  color: WHITE,
                  '&:hover': {
                    backgroundColor: DARK_BUTTON_BG_HOVER
                  }
                })
              },
              outlinedSecondary: {
                ...(mode === 'dark' && {
                  borderColor: DARK_BORDER_MEDIUM,
                  color: WHITE,
                  '&:hover': {
                    backgroundColor: DARK_HOVER_BG,
                    borderColor: WHITE
                  }
                })
              },
              text: {
                ...(mode === 'dark' && {
                  color: alpha(WHITE, 0.85),
                  '&:hover': {
                    backgroundColor: DARK_HOVER_BG_STRONGER
                  }
                })
              }
            }
          },
          MuiAppBar: {
            styleOverrides: {
              colorDefault: {
                backgroundColor: mode === 'dark' ? DARK_BG_PAPER : WHITE
              }
            }
          },
          MuiDialog: {
            styleOverrides: {
              paper: {
                backgroundImage: 'none'
              }
            }
          },
          MuiChip: {
            styleOverrides: {
              outlined: {
                ...(mode === 'dark' && {
                  borderColor: DARK_BORDER_LIGHT
                })
              }
            }
          },
          MuiSwitch: {
            styleOverrides: {
              switchBase: {
                ...(mode === 'dark' && {
                  opacity: 0.8,
                  '&.Mui-checked': {
                    opacity: 1
                  }
                })
              }
            }
          }
        }
      }),
    [mode]
  )

  const contextValue = useMemo(() => {
    return {
      mode,
      toggleTheme,
      resetToSystemTheme,
      isUsingSystemTheme
    }
  }, [mode, isUsingSystemTheme, prefersDarkMode])

  return (
    <ThemeContext.Provider value={contextValue}>
      <MuiThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </MuiThemeProvider>
    </ThemeContext.Provider>
  )
}
