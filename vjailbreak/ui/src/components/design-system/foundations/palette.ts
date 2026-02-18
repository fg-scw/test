export const palette = {
  base: {
    white: '#ffffff',
    black: '#000000'
  },
  light: {
    background: {
      default: '#f5f5f5',
      paper: '#ffffff',
      elevated: '#ffffff'
    },
    text: {
      primary: 'rgba(0, 0, 0, 0.87)',
      secondary: 'rgba(0, 0, 0, 0.6)'
    },
    divider: 'rgba(0, 0, 0, 0.12)',
    border: {
      main: 'rgba(0, 0, 0, 0.23)'
    },
    hover: {
      low: 'rgba(0, 0, 0, 0.04)'
    }
  },
  dark: {
    background: {
      default: '#121212',
      paper: '#1e1e1e',
      elevated: '#2c2c2c'
    },
    text: {
      primary: '#ffffff',
      secondary: 'rgba(255, 255, 255, 0.7)'
    },
    divider: 'rgba(255, 255, 255, 0.12)',
    border: {
      light: 'rgba(255, 255, 255, 0.3)',
      medium: 'rgba(255, 255, 255, 0.5)'
    },
    hover: {
      low: 'rgba(255, 255, 255, 0.08)',
      strong: 'rgba(255, 255, 255, 0.1)'
    },
    button: {
      contained: 'rgba(255, 255, 255, 0.15)',
      containedHover: 'rgba(255, 255, 255, 0.25)'
    }
  },
  secondary: {
    lightMode: {
      main: '#444f5f',
      light: '#5a6679',
      dark: '#353f4c',
      contrastText: '#ffffff'
    },
    darkMode: {
      main: '#8c9db5',
      light: '#adbdd1',
      dark: '#6c7a93',
      contrastText: '#ffffff'
    }
  },
  status: {
    light: {
      error: '#d32f2f',
      warning: '#ed6c02',
      info: '#0288d1',
      success: '#2e7d32'
    },
    dark: {
      error: '#f44336',
      warning: '#ff9800',
      info: '#29b6f6',
      success: '#66bb6a'
    }
  },
  scrollbar: {
    light: {
      track: '#f5f5f5',
      thumb: '#959595'
    },
    dark: {
      track: '#2b2b2b',
      thumb: '#6b6b6b'
    }
  }
} as const

export type Palette = typeof palette

export const getSurfacePalette = (mode: 'light' | 'dark') => palette[mode]
